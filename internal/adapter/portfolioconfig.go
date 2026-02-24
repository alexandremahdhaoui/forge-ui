//go:build !js || !wasm

package adapter

import (
	"errors"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

type portfolioConfigLoaderImpl struct{}

// NewPortfolioConfigLoader returns a PortfolioConfigLoader backed by real filesystem reads.
func NewPortfolioConfigLoader() PortfolioConfigLoader {
	return &portfolioConfigLoaderImpl{}
}

// Load reads forge-portfolio.yaml from portfolioPath and returns a PortfolioConfig.
// Missing file returns zero PortfolioConfig and nil error.
// Parse failure returns an error.
func (p *portfolioConfigLoaderImpl) Load(portfolioPath string) (types.PortfolioConfig, error) {
	var result types.PortfolioConfig

	data, err := os.ReadFile(filepath.Join(portfolioPath, "forge-portfolio.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return result, err
	}

	var f portfolioConfigFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return result, err
	}

	result.Name = f.Name
	result.Description = f.Description
	result.TrackerPaths = f.TrackerPaths

	return result, nil
}

type portfolioConfigFile struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	TrackerPaths []string `json:"trackerPaths"`
}
