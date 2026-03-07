// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
