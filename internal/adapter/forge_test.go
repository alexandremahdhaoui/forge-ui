//go:build !js || !wasm

package adapter

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestForgeLoad_SpecOnly(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "forge.yaml", `
name: my-repo
build:
  - name: app
    src: ./cmd/app
    dest: ./build
    engine: go://go-build
test:
  - name: unit
    runner: go://go-test
`)

	fl := NewForgeLoader()
	data, err := fl.Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if data.Spec.Name != "my-repo" {
		t.Errorf("Spec.Name = %q, want %q", data.Spec.Name, "my-repo")
	}
	if len(data.Spec.Build) != 1 {
		t.Fatalf("len(Spec.Build) = %d, want 1", len(data.Spec.Build))
	}
	if data.Spec.Build[0].Name != "app" {
		t.Errorf("Build[0].Name = %q, want %q", data.Spec.Build[0].Name, "app")
	}
	if len(data.Spec.Test) != 1 {
		t.Fatalf("len(Spec.Test) = %d, want 1", len(data.Spec.Test))
	}
	if data.Spec.Test[0].Name != "unit" {
		t.Errorf("Test[0].Name = %q, want %q", data.Spec.Test[0].Name, "unit")
	}
	// No artifact-store.yaml, so no reports/artifacts
	if len(data.Artifacts) != 0 {
		t.Errorf("len(Artifacts) = %d, want 0", len(data.Artifacts))
	}
	if len(data.TestReports) != 0 {
		t.Errorf("len(TestReports) = %d, want 0", len(data.TestReports))
	}
}

func TestForgeLoad_FullArtifactStore(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "forge.yaml", `
name: full-repo
test:
  - name: unit
    runner: go://go-test
  - name: lint
    runner: go://go-lint
`)
	writeFile(t, filepath.Join(tmp, ".forge"), "artifact-store.yaml", `
artifacts:
  - name: app
    type: binary
    location: ./build/bin/app
    timestamp: "2025-01-15T10:00:00Z"
    version: abc123
    dependencies:
      - type: file
        filePath: go.mod
        timestamp: "2025-01-15T09:00:00Z"
testReports:
  unit-run-1:
    id: unit-run-1
    stage: unit
    status: passed
    startTime: "2025-01-15T10:01:00Z"
    duration: 1.5
    testStats:
      total: 20
      passed: 19
      failed: 1
      skipped: 0
    coverage:
      enabled: true
      percentage: 85.5
  lint-run-1:
    id: lint-run-1
    stage: lint
    status: passed
    startTime: "2025-01-15T10:02:00Z"
    duration: 0.8
    testStats:
      total: 5
      passed: 5
      failed: 0
      skipped: 0
    coverage:
      enabled: false
testEnvironments:
  env-1:
    id: env-1
    name: dev
    status: running
    createdAt: "2025-01-15T09:00:00Z"
    updatedAt: "2025-01-15T10:00:00Z"
    managedResources:
      - pod/dev-1
      - svc/dev-svc
`)

	fl := NewForgeLoader()
	data, err := fl.Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(data.Artifacts) != 1 {
		t.Fatalf("len(Artifacts) = %d, want 1", len(data.Artifacts))
	}
	if data.Artifacts[0].Name != "app" {
		t.Errorf("Artifacts[0].Name = %q, want %q", data.Artifacts[0].Name, "app")
	}
	if len(data.Artifacts[0].Dependencies) != 1 {
		t.Errorf("len(Dependencies) = %d, want 1", len(data.Artifacts[0].Dependencies))
	}

	if len(data.TestReports) != 2 {
		t.Fatalf("len(TestReports) = %d, want 2", len(data.TestReports))
	}

	if len(data.TestEnvs) != 1 {
		t.Fatalf("len(TestEnvs) = %d, want 1", len(data.TestEnvs))
	}
	if data.TestEnvs[0].Name != "dev" {
		t.Errorf("TestEnvs[0].Name = %q, want %q", data.TestEnvs[0].Name, "dev")
	}
	if len(data.TestEnvs[0].ManagedResources) != 2 {
		t.Errorf("len(ManagedResources) = %d, want 2", len(data.TestEnvs[0].ManagedResources))
	}
}

func TestForgeLoad_NoForgeYaml(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	fl := NewForgeLoader()
	data, err := fl.Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	// Should return empty data, no error
	if data.Spec.Name != "" {
		t.Errorf("Spec.Name = %q, want empty", data.Spec.Name)
	}
}

func TestForgeLoad_InvalidYaml(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "forge.yaml", `{{{invalid yaml`)

	fl := NewForgeLoader()
	_, err := fl.Load(tmp)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestForgeLoad_TestReportsSorted(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "forge.yaml", `name: sorted-test`)
	writeFile(t, filepath.Join(tmp, ".forge"), "artifact-store.yaml", `
testReports:
  older:
    id: older
    stage: unit
    status: passed
    startTime: "2025-01-15T08:00:00Z"
    duration: 1.0
    testStats: {total: 1, passed: 1, failed: 0, skipped: 0}
    coverage: {enabled: false}
  newer:
    id: newer
    stage: lint
    status: passed
    startTime: "2025-01-15T10:00:00Z"
    duration: 0.5
    testStats: {total: 2, passed: 2, failed: 0, skipped: 0}
    coverage: {enabled: false}
`)

	fl := NewForgeLoader()
	data, err := fl.Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(data.TestReports) != 2 {
		t.Fatalf("len(TestReports) = %d, want 2", len(data.TestReports))
	}
	// Reports should be sorted by StartTime descending (newer first)
	if data.TestReports[0].ID != "newer" {
		t.Errorf("TestReports[0].ID = %q, want %q (sorted by time desc)", data.TestReports[0].ID, "newer")
	}
	if data.TestReports[1].ID != "older" {
		t.Errorf("TestReports[1].ID = %q, want %q", data.TestReports[1].ID, "older")
	}
}

func TestForgeLoad_TestEnvsSorted(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "forge.yaml", `name: env-sort-test`)
	writeFile(t, filepath.Join(tmp, ".forge"), "artifact-store.yaml", `
testEnvironments:
  old-env:
    id: old-env
    name: staging
    status: passed
    createdAt: "2025-01-10T08:00:00Z"
    updatedAt: "2025-01-10T09:00:00Z"
    managedResources: []
  new-env:
    id: new-env
    name: dev
    status: running
    createdAt: "2025-01-15T08:00:00Z"
    updatedAt: "2025-01-15T09:00:00Z"
    managedResources: []
`)

	fl := NewForgeLoader()
	data, err := fl.Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(data.TestEnvs) != 2 {
		t.Fatalf("len(TestEnvs) = %d, want 2", len(data.TestEnvs))
	}
	// Envs should be sorted by CreatedAt descending (newer first)
	if data.TestEnvs[0].ID != "new-env" {
		t.Errorf("TestEnvs[0].ID = %q, want %q (sorted by time desc)", data.TestEnvs[0].ID, "new-env")
	}
	if data.TestEnvs[1].ID != "old-env" {
		t.Errorf("TestEnvs[1].ID = %q, want %q", data.TestEnvs[1].ID, "old-env")
	}
}
