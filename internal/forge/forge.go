package forge

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

// Load reads forge.yaml and .forge/artifact-store.yaml from repoPath
// and returns a populated ForgePageData. WorkspaceName and RepoName
// are left empty; the caller sets them.
func Load(repoPath string) (model.ForgePageData, error) {
	var result model.ForgePageData

	// 1. Read forge.yaml
	specData, err := os.ReadFile(filepath.Join(repoPath, "forge.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return result, err
	}

	var sf specFile
	if err := yaml.Unmarshal(specData, &sf); err != nil {
		return result, err
	}
	result.Spec = convertSpec(sf)

	// 2. Read .forge/artifact-store.yaml
	storeData, err := os.ReadFile(filepath.Join(repoPath, ".forge", "artifact-store.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return result, err
	}

	var asf artifactStoreFile
	if err := yaml.Unmarshal(storeData, &asf); err != nil {
		return result, err
	}

	result.Artifacts = convertArtifacts(asf.Artifacts)
	result.TestReports = convertTestReports(asf.TestReports)
	result.TestEnvs = convertTestEnvs(asf.TestEnvironments)

	return result, nil
}

// --- Local structs for YAML deserialization (json tags for sigs.k8s.io/yaml) ---

type specFile struct {
	Name  string          `json:"name"`
	Build []buildSpecFile `json:"build"`
	Test  []testSpecFile  `json:"test"`
}

type buildSpecFile struct {
	Name   string `json:"name"`
	Src    string `json:"src"`
	Dest   string `json:"dest,omitempty"`
	Engine string `json:"engine"`
}

type testSpecFile struct {
	Name    string `json:"name"`
	Testenv string `json:"testenv,omitempty"`
	Runner  string `json:"runner"`
}

type artifactStoreFile struct {
	Artifacts        []artifactFile             `json:"artifacts"`
	TestReports      map[string]*testReportFile `json:"testReports,omitempty"`
	TestEnvironments map[string]*testEnvFile    `json:"testEnvironments,omitempty"`
}

type artifactFile struct {
	Name         string                   `json:"name"`
	Type         string                   `json:"type"`
	Location     string                   `json:"location"`
	Timestamp    string                   `json:"timestamp"`
	Version      string                   `json:"version"`
	Dependencies []artifactDependencyFile `json:"dependencies,omitempty"`
}

type artifactDependencyFile struct {
	Type            string `json:"type"`
	FilePath        string `json:"filePath,omitempty"`
	Timestamp       string `json:"timestamp,omitempty"`
	ExternalPackage string `json:"externalPackage,omitempty"`
	Semver          string `json:"semver,omitempty"`
}

type testReportFile struct {
	ID        string    `json:"id"`
	Stage     string    `json:"stage"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"startTime"`
	Duration  float64   `json:"duration"`
	TestStats struct {
		Total   int `json:"total"`
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Skipped int `json:"skipped"`
	} `json:"testStats"`
	Coverage struct {
		Enabled    bool    `json:"enabled"`
		Percentage float64 `json:"percentage"`
		FilePath   string  `json:"filePath,omitempty"`
	} `json:"coverage"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type testEnvFile struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	ManagedResources []string  `json:"managedResources"`
}

// --- Conversion functions ---

func convertSpec(s specFile) model.ForgeSpec {
	spec := model.ForgeSpec{
		Name:  s.Name,
		Build: make([]model.BuildSpec, len(s.Build)),
		Test:  make([]model.TestSpec, len(s.Test)),
	}
	for i, b := range s.Build {
		spec.Build[i] = model.BuildSpec{
			Name:   b.Name,
			Src:    b.Src,
			Dest:   b.Dest,
			Engine: b.Engine,
		}
	}
	for i, t := range s.Test {
		spec.Test[i] = model.TestSpec{
			Name:    t.Name,
			Testenv: t.Testenv,
			Runner:  t.Runner,
		}
	}
	return spec
}

func convertArtifacts(as []artifactFile) []model.Artifact {
	artifacts := make([]model.Artifact, len(as))
	for i, a := range as {
		deps := make([]model.ArtifactDependency, len(a.Dependencies))
		for j, d := range a.Dependencies {
			deps[j] = model.ArtifactDependency{
				Type:            d.Type,
				FilePath:        d.FilePath,
				Timestamp:       d.Timestamp,
				ExternalPackage: d.ExternalPackage,
				Semver:          d.Semver,
			}
		}
		artifacts[i] = model.Artifact{
			Name:         a.Name,
			Type:         a.Type,
			Location:     a.Location,
			Timestamp:    a.Timestamp,
			Version:      a.Version,
			Dependencies: deps,
		}
	}
	return artifacts
}

func convertTestReports(m map[string]*testReportFile) []model.TestReport {
	reports := make([]model.TestReport, 0, len(m))
	for _, r := range m {
		if r == nil {
			continue
		}
		reports = append(reports, model.TestReport{
			ID:        r.ID,
			Stage:     r.Stage,
			Status:    r.Status,
			StartTime: r.StartTime,
			Duration:  r.Duration,
			Stats: model.TestStats{
				Total:   r.TestStats.Total,
				Passed:  r.TestStats.Passed,
				Failed:  r.TestStats.Failed,
				Skipped: r.TestStats.Skipped,
			},
			Coverage: model.Coverage{
				Enabled:    r.Coverage.Enabled,
				Percentage: r.Coverage.Percentage,
				FilePath:   r.Coverage.FilePath,
			},
			ErrorMessage: r.ErrorMessage,
		})
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].StartTime.After(reports[j].StartTime)
	})
	return reports
}

func convertTestEnvs(m map[string]*testEnvFile) []model.TestEnv {
	envs := make([]model.TestEnv, 0, len(m))
	for _, e := range m {
		if e == nil {
			continue
		}
		envs = append(envs, model.TestEnv{
			ID:               e.ID,
			Name:             e.Name,
			Status:           e.Status,
			CreatedAt:        e.CreatedAt,
			UpdatedAt:        e.UpdatedAt,
			ManagedResources: e.ManagedResources,
		})
	}
	sort.Slice(envs, func(i, j int) bool {
		return envs[i].CreatedAt.After(envs[j].CreatedAt)
	})
	return envs
}
