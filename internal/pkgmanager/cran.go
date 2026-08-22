package pkgmanager

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ossf/package-analysis/pkg/api/pkgecosystem"
)

// cranMirror is the CRAN mirror used to resolve package metadata and source
// archives. cloud.r-project.org is a CDN-backed mirror of CRAN which serves
// both the package web pages and the source archives.
const cranMirror = "https://cloud.r-project.org"

// cranVersionField is the DESCRIPTION field holding the package version.
// DESCRIPTION files use Debian Control Format, i.e. 'Field: value' lines.
const cranVersionField = "Version:"

var errCRANNoVersionField = errors.New("no Version field in CRAN DESCRIPTION file")

// getCRANLatest returns the current released version of a CRAN package, read
// from the Version field of the package's DESCRIPTION file.
//
// Note that CRAN package names are case sensitive, so pkg must be spelled
// exactly as the package is published (e.g. 'Rcpp', not 'rcpp').
func getCRANLatest(pkg string) (string, error) {
	descriptionURL := fmt.Sprintf("%s/web/packages/%s/DESCRIPTION", cranMirror, pkg)

	resp, err := http.Get(descriptionURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not fetch %s: http status %s", descriptionURL, resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if version, found := strings.CutPrefix(line, cranVersionField); found {
			return strings.TrimSpace(version), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("%w: %s", errCRANNoVersionField, pkg)
}

// getCRANArchiveURL returns the URL of the source archive for the given CRAN
// package version.
//
// CRAN serves only the current release of each package from /src/contrib;
// once a release is superseded it moves to /src/contrib/Archive/<name>/. Which
// of the two applies is not known ahead of time, so both are tried in turn and
// the status code is used to decide.
//
// If neither location holds the requested version, an empty URL is returned
// with a nil error, following the convention of the other package managers
// here; DownloadArchive converts this into ErrNoArchiveURL.
func getCRANArchiveURL(pkgName, version string) (string, error) {
	archiveName := fmt.Sprintf("%s_%s.tar.gz", pkgName, version)

	candidates := []string{
		fmt.Sprintf("%s/src/contrib/%s", cranMirror, archiveName),
		fmt.Sprintf("%s/src/contrib/Archive/%s/%s", cranMirror, pkgName, archiveName),
	}

	for _, pkgURL := range candidates {
		resp, err := http.Head(pkgURL)
		if err != nil {
			return "", err
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return pkgURL, nil
		}
	}

	return "", nil
}

// Note: normalizeName is deliberately unset. CRAN package names are case
// sensitive and mixed case is common (e.g. MASS, Rcpp, Matrix), so names must
// be passed through exactly as given.
var cranPkgManager = PkgManager{
	ecosystem:       pkgecosystem.CRAN,
	latestVersion:   getCRANLatest,
	archiveURL:      getCRANArchiveURL,
	archiveFilename: defaultArchiveFilename,
}
