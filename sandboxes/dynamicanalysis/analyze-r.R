#!/usr/bin/env Rscript
#
# Dynamic analysis driver for CRAN packages.
#
# Usage: analyze-r.R [--local <file> | --version <version>] <phase> <package>
#
# Phases:
#   install   install the package
#   import    load the package namespace and attach it to the search path
#   all       install, then import
#
# Follows the CLI contract shared by the other analyze-* scripts in this
# directory; see sandboxes/README.md, "Adding a new Runtime Analysis script".
#
# NOTE: the shebang deliberately omits --vanilla, because arguments after the
# interpreter in a shebang line are not portable. If a stray ~/.Rprofile ever
# ends up in the sandbox image it would be sourced here and pollute the strace
# output; the Dockerfile should ensure none exists.

# A single explicit mirror. A non-interactive session has no repos option set,
# and install.packages() fails before reaching the network without one.
CRAN_MIRROR <- "https://cloud.r-project.org"

# Superseded versions are not served from /src/contrib -- that holds only the
# current release of each package. Everything older moves under Archive/.
CRAN_CONTRIB <- "https://cloud.r-project.org/src/contrib"
CRAN_ARCHIVE <- "https://cloud.r-project.org/src/contrib/Archive"

# Everything is installed into a library of our own, set by R_LIBS_USER in the
# sandbox Dockerfile. R ships a set of recommended packages (MASS, Matrix,
# survival, ...) in the base library; without a separate library, find.package()
# would report those as installed before we had installed anything.
ANALYSIS_LIB <- .libPaths()[1]

# Dependency fields of a DESCRIPTION file that have to be installed for the
# package to load. Suggests and Enhances are deliberately excluded.
DEPENDENCY_FIELDS <- c("Depends", "Imports", "LinkingTo")

# Print warnings as they occur rather than batching them at the end, so the
# output interleaves correctly with the syscall timeline.
options(warn = 1)


new_package <- function(name, version = NULL, local_path = NULL) {
  list(name = name, version = version, local_path = local_path)
}


# install.packages() signals a *warning* on failure, not an error, and returns
# NULL either way -- so its return value tells us nothing. The only reliable
# check is whether the package actually landed in the library.
is_installed <- function(name) {
  length(find.package(name, lib.loc = ANALYSIS_LIB, quiet = TRUE)) > 0
}


# Source tarball URLs to try, in order. A version pinned by the worker is
# usually the current release, which lives in /src/contrib; only once it has
# been superseded does it move to Archive/. Try both rather than guessing.
version_urls <- function(name, version) {
  tarball <- sprintf("%s_%s.tar.gz", name, version)
  c(
    sprintf("%s/%s", CRAN_CONTRIB, tarball),
    sprintf("%s/%s/%s", CRAN_ARCHIVE, name, tarball)
  )
}


# repos = NULL installs exactly the given tarball and nothing else, so a
# package's declared dependencies have to be present beforehand. Only the direct
# dependencies are resolved here; install.packages() follows the transitive ones
# itself when repos is set.
#
# The dependencies come from the mirror's index, which describes the current
# release. For an archived version these are an approximation, and a mismatch
# surfaces as an install failure, which is reported like any other.
install_deps <- function(name) {
  db <- available.packages(repos = CRAN_MIRROR, type = "source")
  deps <- tools::package_dependencies(name, db = db, which = DEPENDENCY_FIELDS)[[name]]
  deps <- setdiff(deps, c("R", rownames(installed.packages())))

  if (length(deps) == 0) {
    return(invisible(NULL))
  }

  cat(sprintf("Installing dependencies: %s\n", paste(deps, collapse = ", ")))
  install.packages(deps, repos = CRAN_MIRROR, type = "source")
}


install_pkg <- function(pkg) {
  if (!is.null(pkg$local_path)) {
    install_deps(pkg$name)
    cat(sprintf("Installing from local path %s\n", pkg$local_path))
    try(install.packages(pkg$local_path, repos = NULL, type = "source"),
        silent = FALSE)
  } else if (!is.null(pkg$version)) {
    install_deps(pkg$name)
    for (url in version_urls(pkg$name, pkg$version)) {
      cat(sprintf("Installing from %s\n", url))
      try(install.packages(url, repos = NULL, type = "source"), silent = FALSE)
      if (is_installed(pkg$name)) {
        break
      }
      cat(sprintf("Not installed from %s, trying next candidate\n", url))
    }
  } else {
    # Dependencies need no special handling here: with repos set,
    # install.packages() resolves and installs them itself.
    cat(sprintf("Installing %s from %s\n", pkg$name, CRAN_MIRROR))
    try(install.packages(pkg$name, repos = CRAN_MIRROR, type = "source"),
        silent = FALSE)
  }

  if (!is_installed(pkg$name)) {
    # Always stop.
    # Install failing is either an interesting issue, or an opportunity to
    # improve the analysis.
    stop(sprintf("Failed to install: %s", pkg$name))
  }

  cat("Install succeeded:\n")
  cat(sprintf("%s\n", find.package(pkg$name, lib.loc = ANALYSIS_LIB)))
}


# library() both loads the namespace and attaches it to the search path, which
# runs the package's .onLoad and .onAttach hooks -- the main place load-time
# side effects hide in an R package.
import_pkg <- function(pkg) {
  cat(sprintf("Importing %s\n", pkg$name))

  ok <- tryCatch({
    library(pkg$name, character.only = TRUE)
    TRUE
  }, error = function(e) {
    cat(sprintf("Failed to import %s: %s\n", pkg$name, conditionMessage(e)))
    FALSE
  })

  if (ok) {
    cat("Import succeeded. Search path:\n")
    cat(sprintf("%s\n", paste(search(), collapse = " ")))
  }
}


PHASE_FUNCTIONS <- list(
  install = install_pkg,
  import = import_pkg
)

PHASES <- list(
  all = c("install", "import"),
  install = c("install"),
  import = c("import")
)


main <- function() {
  args <- commandArgs(trailingOnly = TRUE)

  if (length(args) < 2 || length(args) > 4) {
    cat("Usage: analyze-r.R [--local file | --version version] phase package_name\n")
    quit(status = 1)
  }

  # Parse the arguments manually to avoid introducing unnecessary dependencies
  # and side effects that add noise to the strace output.
  local_path <- NULL
  version <- NULL

  if (args[1] == "--local") {
    local_path <- args[2]
    args <- args[-c(1, 2)]
  } else if (args[1] == "--version") {
    version <- args[2]
    args <- args[-c(1, 2)]
  }

  if (length(args) != 2) {
    cat("Usage: analyze-r.R [--local file | --version version] phase package_name\n")
    quit(status = 1)
  }

  phase <- args[1]
  package_name <- args[2]

  if (!phase %in% names(PHASES)) {
    cat(sprintf("Unknown phase %s specified.\n", phase))
    quit(status = 1)
  }

  pkg <- new_package(package_name, version = version, local_path = local_path)

  # Execute for the specified phase.
  for (step in PHASES[[phase]]) {
    PHASE_FUNCTIONS[[step]](pkg)
  }
}


main()
