// Package sbom implements standardized SBoM tooling that allows multiple SBoM
// formats to be generated from the same scanning information.
package sbom

import (
	"fmt"
	"os"

	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/pkg"
	"github.com/anchore/syft/syft/pkg/cataloger"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
	"github.com/paketo-buildpacks/packit/v2/postal"
)

// SBOM holds the internal representation of the generated software
// bill-of-materials. This type can be combined with a FormattedReader to
// output the SBoM in a number of file formats.
type SBOM struct {
	syft sbom.SBOM
}

// Generate returns a populated SBOM given a path to a directory to scan.
func Generate(path string) (SBOM, error) {
	info, err := os.Stat(path)
	if err != nil {
		return SBOM{}, err
	}

	var src source.Source
	if info.IsDir() {
		src, err = source.NewFromDirectory(path)
		if err != nil {
			return SBOM{}, err
		}
	} else {
		var cleanup func()
		src, cleanup = source.NewFromFile(path)
		defer cleanup()
	}

	config := cataloger.Config{
		Search: cataloger.SearchConfig{
			Scope: source.UnknownScope,
		},
	}

	catalog, _, release, err := syft.CatalogPackages(&src, config)
	if err != nil {
		return SBOM{}, err
	}

	return SBOM{
		syft: sbom.SBOM{
			Artifacts: sbom.Artifacts{
				PackageCatalog:    catalog,
				LinuxDistribution: release,
			},
			Source: src.Metadata,
		},
	}, nil
}

// GenerateFromDependency returns a populated SBOM given a postal.Dependency
// and the directory path where the dependency will be located within the
// application image.
func GenerateFromDependency(dependency postal.Dependency, path string) (SBOM, error) {
	cpe, err := pkg.NewCPE(dependency.CPE)
	if err != nil {
		return SBOM{}, err
	}

	catalog := pkg.NewCatalog(pkg.Package{
		Name:     dependency.Name,
		Version:  dependency.Version,
		Licenses: dependency.Licenses,
		CPEs:     []pkg.CPE{cpe},
		PURL:     dependency.PURL,
	})

	return SBOM{
		syft: sbom.SBOM{
			Artifacts: sbom.Artifacts{
				PackageCatalog: catalog,
			},
			Source: source.Metadata{
				Scheme: source.DirectoryScheme,
				Path:   path,
			},
		},
	}, nil
}

// InFormats returns a Formatter containing mappings for the given Formats.
func (s SBOM) InFormats(formats ...string) (Formatter, error) {
	var fs []Format
	for _, f := range formats {
		format := Format(f)
		if format.Extension() == "" {
			return Formatter{}, fmt.Errorf("%q is not a supported SBOM format", f)
		}

		fs = append(fs, format)
	}

	return Formatter{sbom: s, formats: fs}, nil
}
