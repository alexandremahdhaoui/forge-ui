package metaplan

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

// LoadAll reads all .yaml files from wsPath/.forge-ai/meta-plan/ and returns
// a slice of MetaPlan sorted by name.
// Missing directory returns empty slice and nil error.
// Malformed files are logged and skipped.
func LoadAll(wsPath string) ([]model.MetaPlan, error) {
	dir := filepath.Join(wsPath, ".forge-ai", "meta-plan")

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var plans []model.MetaPlan
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}

		p, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			log.Printf("metaplan.Load(%s): %v", e.Name(), err)
			continue
		}
		plans = append(plans, p)
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Name < plans[j].Name
	})

	return plans, nil
}

// Load reads a single meta-plan YAML file and returns a MetaPlan.
func Load(path string) (model.MetaPlan, error) {
	var result model.MetaPlan

	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}

	var f metaPlanFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return result, err
	}

	result.Name = f.Name
	result.Description = f.Description
	result.Status = f.Status

	result.Stages = make([]model.MetaPlanStage, len(f.Stages))
	for i, s := range f.Stages {
		repos := make([]model.StageRepo, len(s.Repos))
		for j, r := range s.Repos {
			repos[j] = model.StageRepo{
				Name:       r.Name,
				Plan:       r.Plan,
				TasksTotal: r.TasksTotal,
				TasksDone:  r.TasksDone,
			}
		}
		result.Stages[i] = model.MetaPlanStage{
			Name:   s.Name,
			Status: s.Status,
			Repos:  repos,
		}
	}

	result.Checkpoints = make([]model.MetaCheckpoint, len(f.Checkpoints))
	for i, c := range f.Checkpoints {
		result.Checkpoints[i] = model.MetaCheckpoint{
			Name:      c.Name,
			Stage:     c.Stage,
			Condition: c.Condition,
			Met:       c.Met,
		}
	}

	return result, nil
}

// --- Local structs for YAML deserialization (json tags for sigs.k8s.io/yaml) ---

type metaPlanFile struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      string               `json:"status"`
	Stages      []metaPlanStageFile  `json:"stages"`
	Checkpoints []metaCheckpointFile `json:"checkpoints"`
}

type metaPlanStageFile struct {
	Name   string          `json:"name"`
	Status string          `json:"status"`
	Repos  []stageRepoFile `json:"repos"`
}

type stageRepoFile struct {
	Name       string `json:"name"`
	Plan       string `json:"plan"`
	TasksTotal int    `json:"tasksTotal"`
	TasksDone  int    `json:"tasksDone"`
}

type metaCheckpointFile struct {
	Name      string `json:"name"`
	Stage     string `json:"stage"`
	Condition string `json:"condition"`
	Met       bool   `json:"met"`
}
