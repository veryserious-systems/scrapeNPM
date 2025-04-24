package processor

import (
	"fmt"
	"math"
	"time"

	"scrapeNPM/internal/models"

	"github.com/google/uuid"
)

type Extractor struct{}

func NewExtractor() *Extractor {
	return &Extractor{}
}

func (e *Extractor) ExtractPackageData(
	pkgName string,
	rawData map[string]interface{},
) (models.Package, error) {
	pkg := models.Package{
		Name: pkgName,
	}

	if description, ok := rawData["description"].(string); ok {
		pkg.Description = description
	}

	if distTags, ok := rawData["dist-tags"].(map[string]interface{}); ok {
		if latest, ok := distTags["latest"].(string); ok {
			pkg.Version = latest
		}
	}

	if author, ok := rawData["author"].(map[string]interface{}); ok {
		if name, ok := author["name"].(string); ok {
			pkg.Author = name
		}
	} else if authorStr, ok := rawData["author"].(string); ok {
		pkg.Author = authorStr
	}

	if homepage, ok := rawData["homepage"].(string); ok {
		pkg.Homepage = homepage
	}

	if repo, ok := rawData["repository"].(map[string]interface{}); ok {
		if url, ok := repo["url"].(string); ok {
			pkg.Repository = url
		}
	} else if repoStr, ok := rawData["repository"].(string); ok {
		pkg.Repository = repoStr
	}

	if license, ok := rawData["license"].(string); ok {
		pkg.License = license
	} else if licenses, ok := rawData["licenses"].([]interface{}); ok && len(licenses) > 0 {
		if licenseObj, ok := licenses[0].(map[string]interface{}); ok {
			if licType, ok := licenseObj["type"].(string); ok {
				pkg.License = licType
			}
		}
	}

	if times, ok := rawData["time"].(map[string]interface{}); ok {
		if created, ok := times["created"].(string); ok {
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				pkg.CreatedAt = t
			}
		}
		if modified, ok := times["modified"].(string); ok {
			if t, err := time.Parse(time.RFC3339, modified); err == nil {
				pkg.UpdatedAt = t
			}
		}
	}
	if pkg.CreatedAt.IsZero() {
		pkg.CreatedAt = time.Now()
	}
	if pkg.UpdatedAt.IsZero() {
		pkg.UpdatedAt = time.Now()
	}

	return pkg, nil
}

func (e *Extractor) ExtractScripts(rawData map[string]interface{}, packageID uuid.UUID, version string) ([]models.PackageScript, error) {
	var scripts []models.PackageScript

	versionsData, ok := rawData["versions"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("versions data not found or invalid")
	}

	versionData, ok := versionsData[version].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("version %s not found or invalid", version)
	}

	scriptsData, ok := versionData["scripts"].(map[string]interface{})
	if !ok {
		return scripts, nil
	}

	scriptTypes := []string{"install", "preinstall", "postinstall"}
	for _, scriptType := range scriptTypes {
		if content, ok := scriptsData[scriptType].(string); ok && content != "" {
			script := models.PackageScript{
				PackageID:  packageID,
				ScriptType: scriptType,
				Content:    content,
			}
			scripts = append(scripts, script)
		}
	}

	return scripts, nil
}

// CalculatePopularityScore calculates a popularity score based on downloads
func (e *Extractor) CalculatePopularityScore(downloads int64) float64 {
	return math.Min(1.0, float64(downloads)/1000000.0)
}
