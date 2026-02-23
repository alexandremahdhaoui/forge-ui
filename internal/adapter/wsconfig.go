package adapter

import (
	"errors"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

type wsConfigLoaderImpl struct{}

// NewWsConfigLoader returns a WsConfigLoader backed by real filesystem reads.
func NewWsConfigLoader() WsConfigLoader {
	return &wsConfigLoaderImpl{}
}

// Load reads forge-workspace.yaml from wsPath and returns a WsConfig.
// Missing file returns zero WsConfig and nil error.
// Parse failure returns an error.
func (w *wsConfigLoaderImpl) Load(wsPath string) (types.WsConfig, error) {
	var result types.WsConfig

	data, err := os.ReadFile(filepath.Join(wsPath, "forge-workspace.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return result, err
	}

	var f wsConfigFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return result, err
	}

	result.Name = f.Name
	result.Description = f.Description
	result.MetaPlans = f.MetaPlans

	result.Repos = make([]types.WsRepoEntry, len(f.Repos))
	for i, r := range f.Repos {
		result.Repos[i] = types.WsRepoEntry{
			Name:        r.Name,
			Description: r.Description,
		}
	}

	return result, nil
}

// --- Local structs for YAML deserialization (json tags for sigs.k8s.io/yaml) ---

type wsConfigFile struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Repos       []wsRepoEntryFile `json:"repos"`
	MetaPlans   []string          `json:"metaPlans"`
}

type wsRepoEntryFile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
