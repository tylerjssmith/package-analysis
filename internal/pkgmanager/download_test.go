package pkgmanager

import (
	"testing"

	"github.com/ossf/package-analysis/internal/utils"
	"github.com/ossf/package-analysis/pkg/api/pkgecosystem"
)

type downloadTestCase struct {
	name        string
	ecosystem   pkgecosystem.Ecosystem
	pkgName     string
	pkgVersion  string
	archiveHash string
	wantErr     bool
}

var downloadTestCases = []downloadTestCase{
	{
		name:       "NPM async 'latest' version",
		ecosystem:  pkgecosystem.NPM,
		pkgName:    "async",
		pkgVersion: "latest",
		wantErr:    false,
	},
	{
		name:       "NPM async valid version",
		ecosystem:  pkgecosystem.NPM,
		pkgName:    "async",
		pkgVersion: "3.2.4",
		wantErr:    false,
	},
	{
		name:       "NPM async invalid version",
		ecosystem:  pkgecosystem.NPM,
		pkgName:    "async",
		pkgVersion: "3.2.4444444",
		wantErr:    true,
	},
	{
		name:       "NPM invalid package name",
		ecosystem:  pkgecosystem.NPM,
		pkgName:    "fr(2t5j923)",
		pkgVersion: "latest",
		wantErr:    true,
	},
	{
		name:       "PyPI urllib3 valid version",
		ecosystem:  pkgecosystem.PyPI,
		pkgName:    "urllib3",
		pkgVersion: "1.26.11",
		wantErr:    false,
	},
	{
		name:       "PyPI urllib3 invalid version",
		ecosystem:  pkgecosystem.PyPI,
		pkgName:    "urllib3",
		pkgVersion: "1.26.111",
		wantErr:    true,
	},
	{
		name:       "PyPI invalid package name",
		ecosystem:  pkgecosystem.PyPI,
		pkgName:    "fr(2t5j923)",
		pkgVersion: "123",
		wantErr:    true,
	},
	{
		name:       "crates.io rand valid version",
		ecosystem:  pkgecosystem.CratesIO,
		pkgName:    "rand",
		pkgVersion: "0.8.5",
		wantErr:    false,
	},
	{
		name:       "crates.io rand invalid version",
		ecosystem:  pkgecosystem.CratesIO,
		pkgName:    "rand",
		pkgVersion: "0.8.55",
		wantErr:    true,
	},
	{
		name:       "crates.io invalid package name",
		ecosystem:  pkgecosystem.CratesIO,
		pkgName:    "fr(2t5j923)",
		pkgVersion: "123",
		wantErr:    true,
	},
	{
		name:        "pypi black 23.3.0",
		ecosystem:   pkgecosystem.PyPI,
		pkgName:     "black",
		pkgVersion:  "23.3.0",
		archiveHash: "1c7b8d606e728a41ea1ccbd7264677e494e87cf630e399262ced92d4a8dac940",
		wantErr:     false,
	},
	{
		name:        "npm invalid package name",
		ecosystem:   pkgecosystem.PyPI,
		pkgName:     "3i3ii3ii3i3i",
		pkgVersion:  "",
		archiveHash: "",
		wantErr:     true,
	},
	{
		name:        "pypi black invalid version",
		ecosystem:   pkgecosystem.PyPI,
		pkgName:     "black",
		pkgVersion:  "23333.3.0",
		archiveHash: "",
		wantErr:     true,
	},
	// CRAN package names are case sensitive, so the mixed-case name here is
	// part of what is under test. Note DownloadArchive does not normalize
	// names, so the name reaches archiveURL exactly as written.
	//
	// A given version lives under /src/contrib while it is the current
	// release and moves to /src/contrib/Archive once superseded. getCRANArchiveURL
	// tries both, so these cases stay valid as Rcpp is released.
	{
		name:       "CRAN Rcpp valid version",
		ecosystem:  pkgecosystem.CRAN,
		pkgName:    "Rcpp",
		pkgVersion: "1.1.2",
		wantErr:    false,
	},
	{
		// Pinned to a superseded release, so this always exercises the
		// /src/contrib/Archive fallback.
		name:       "CRAN jsonlite archived version",
		ecosystem:  pkgecosystem.CRAN,
		pkgName:    "jsonlite",
		pkgVersion: "1.8.9",
		wantErr:    false,
	},
	{
		name:       "CRAN Rcpp invalid version",
		ecosystem:  pkgecosystem.CRAN,
		pkgName:    "Rcpp",
		pkgVersion: "1.1.22222",
		wantErr:    true,
	},
	{
		// Lowercased spelling of a real package; CRAN serves nothing here.
		name:       "CRAN wrong case package name",
		ecosystem:  pkgecosystem.CRAN,
		pkgName:    "rcpp",
		pkgVersion: "1.1.2",
		wantErr:    true,
	},
	{
		name:       "CRAN invalid package name",
		ecosystem:  pkgecosystem.CRAN,
		pkgName:    "fr(2t5j923)",
		pkgVersion: "123",
		wantErr:    true,
	},
}

func TestDownload(t *testing.T) {
	for _, tt := range downloadTestCases {
		t.Run(tt.name, func(t *testing.T) {
			downloadDir := t.TempDir()
			downloadPath, err := Manager(tt.ecosystem).DownloadArchive(tt.pkgName, tt.pkgVersion, downloadDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Want error: %v; got error: %v", tt.wantErr, err)
				return
			}
			if err != nil {
				// File wasn't meant to download properly
				return
			}

			if tt.archiveHash != "" {
				gotHash, err := utils.SHA256Hash(downloadPath)
				if err != nil {
					// hashing isn't meant to throw an error
					t.Errorf("hashing failed: %v", err)
					return
				}

				if tt.archiveHash != gotHash {
					t.Errorf("Expected hash %s, got %s", tt.archiveHash, gotHash)
				}
			}

		})
	}
}
