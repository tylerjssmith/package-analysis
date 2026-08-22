package pkgmanager

import (
	"testing"

	"github.com/ossf/package-analysis/pkg/api/pkgecosystem"
)

// Package names are normalized when a Pkg is constructed. Most ecosystems
// lowercase names, but CRAN names are case sensitive and must be preserved.
var normalizeTestCases = []struct {
	name      string
	ecosystem pkgecosystem.Ecosystem
	pkgName   string
	want      string
}{
	{
		name:      "CRAN preserves mixed case",
		ecosystem: pkgecosystem.CRAN,
		pkgName:   "Rcpp",
		want:      "Rcpp",
	},
	{
		name:      "CRAN preserves upper case",
		ecosystem: pkgecosystem.CRAN,
		pkgName:   "MASS",
		want:      "MASS",
	},
	{
		name:      "CRAN leaves lower case alone",
		ecosystem: pkgecosystem.CRAN,
		pkgName:   "jsonlite",
		want:      "jsonlite",
	},
	{
		name:      "PyPI lowercases",
		ecosystem: pkgecosystem.PyPI,
		pkgName:   "Django",
		want:      "django",
	},
	{
		name:      "NPM lowercases",
		ecosystem: pkgecosystem.NPM,
		pkgName:   "MyPkg",
		want:      "mypkg",
	},
	{
		name:      "RubyGems lowercases",
		ecosystem: pkgecosystem.RubyGems,
		pkgName:   "MyGem",
		want:      "mygem",
	},
	{
		name:      "Packagist lowercases",
		ecosystem: pkgecosystem.Packagist,
		pkgName:   "Symfony/Deprecation-Contracts",
		want:      "symfony/deprecation-contracts",
	},
	{
		name:      "crates.io lowercases",
		ecosystem: pkgecosystem.CratesIO,
		pkgName:   "Inflector",
		want:      "inflector",
	},
}

func TestPackageNormalizesName(t *testing.T) {
	for _, tt := range normalizeTestCases {
		t.Run(tt.name, func(t *testing.T) {
			got := Manager(tt.ecosystem).Package(tt.pkgName, "1.0.0").Name()
			if got != tt.want {
				t.Errorf("Package(%q).Name() = %q; want %q", tt.pkgName, got, tt.want)
			}
		})
	}
}

func TestLocalNormalizesName(t *testing.T) {
	for _, tt := range normalizeTestCases {
		t.Run(tt.name, func(t *testing.T) {
			got := Manager(tt.ecosystem).Local(tt.pkgName, "1.0.0", "/tmp/pkg.tar.gz").Name()
			if got != tt.want {
				t.Errorf("Local(%q).Name() = %q; want %q", tt.pkgName, got, tt.want)
			}
		})
	}
}

// Every supported ecosystem must be reachable via Manager(), otherwise
// normalization (and everything else) silently nil-pointers.
func TestAllSupportedEcosystemsHaveManager(t *testing.T) {
	for _, ecosystem := range pkgecosystem.SupportedEcosystems {
		if Manager(ecosystem) == nil {
			t.Errorf("Manager(%q) is nil; want a registered PkgManager", ecosystem)
		}
	}
}
