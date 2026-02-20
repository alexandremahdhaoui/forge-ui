package wsconfig

import (
	"errors"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

// Load reads forge-workspace.yaml from wsPath and returns a WsConfig.
// Missing file returns zero WsConfig and nil error.
// Parse failure returns an error.
func Load(wsPath string) (model.WsConfig, error) {
	var result model.WsConfig

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

	result.Repos = make([]model.WsRepoEntry, len(f.Repos))
	for i, r := range f.Repos {
		result.Repos[i] = model.WsRepoEntry{
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
