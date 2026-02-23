package adapter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetaPlan_LoadAll_ValidFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mpDir := filepath.Join(dir, ".forge-ai", "meta-plan")
	if err := os.MkdirAll(mpDir, 0755); err != nil {
		t.Fatal(err)
	}

	planA := []byte(`name: alpha-plan
description: First plan
status: in_progress
stages:
  - name: stage-1
    status: completed
    repos:
      - name: repo-a
        plan: plan-a
        tasksTotal: 10
        tasksDone: 10
  - name: stage-2
    status: in_progress
    repos:
      - name: repo-b
        plan: plan-b
        tasksTotal: 8
        tasksDone: 3
checkpoints:
  - name: integration-check
    stage: stage-1
    condition: all tests pass
    met: true
`)
	planB := []byte(`name: beta-plan
description: Second plan
status: pending
stages: []
checkpoints: []
`)
	if err := os.WriteFile(filepath.Join(mpDir, "alpha.yaml"), planA, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mpDir, "beta.yaml"), planB, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewMetaPlanLoader()
	plans, err := loader.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("len(plans) = %d, want 2", len(plans))
	}

	// Sorted by name: alpha < beta
	if plans[0].Name != "alpha-plan" {
		t.Errorf("plans[0].Name = %q, want %q", plans[0].Name, "alpha-plan")
	}
	if plans[0].Status != "in_progress" {
		t.Errorf("plans[0].Status = %q, want %q", plans[0].Status, "in_progress")
	}
	if len(plans[0].Stages) != 2 {
		t.Fatalf("len(plans[0].Stages) = %d, want 2", len(plans[0].Stages))
	}
	if plans[0].Stages[0].Repos[0].TasksTotal != 10 {
		t.Errorf("TasksTotal = %d, want 10", plans[0].Stages[0].Repos[0].TasksTotal)
	}
	if plans[0].Stages[0].Repos[0].TasksDone != 10 {
		t.Errorf("TasksDone = %d, want 10", plans[0].Stages[0].Repos[0].TasksDone)
	}
	if len(plans[0].Checkpoints) != 1 {
		t.Fatalf("len(Checkpoints) = %d, want 1", len(plans[0].Checkpoints))
	}
	if !plans[0].Checkpoints[0].Met {
		t.Error("Checkpoints[0].Met = false, want true")
	}

	if plans[1].Name != "beta-plan" {
		t.Errorf("plans[1].Name = %q, want %q", plans[1].Name, "beta-plan")
	}
}

func TestMetaPlan_LoadAll_MissingDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	loader := NewMetaPlanLoader()
	plans, err := loader.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("len(plans) = %d, want 0", len(plans))
	}
}

func TestMetaPlan_LoadAll_MalformedYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mpDir := filepath.Join(dir, ".forge-ai", "meta-plan")
	if err := os.MkdirAll(mpDir, 0755); err != nil {
		t.Fatal(err)
	}

	good := []byte(`name: good-plan
status: pending
`)
	bad := []byte(`name: [broken yaml
  status: bad
`)
	if err := os.WriteFile(filepath.Join(mpDir, "good.yaml"), good, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mpDir, "bad.yaml"), bad, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewMetaPlanLoader()
	plans, err := loader.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	// Bad file is skipped, good file is returned.
	if len(plans) != 1 {
		t.Fatalf("len(plans) = %d, want 1", len(plans))
	}
	if plans[0].Name != "good-plan" {
		t.Errorf("plans[0].Name = %q, want %q", plans[0].Name, "good-plan")
	}
}

func TestMetaPlan_Load_ValidFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := []byte(`name: test-plan
description: A test plan
status: completed
stages:
  - name: deploy
    status: completed
    repos:
      - name: svc
        plan: deploy-plan
        tasksTotal: 5
        tasksDone: 5
checkpoints:
  - name: smoke-test
    stage: deploy
    condition: smoke tests pass
    met: true
`)
	path := filepath.Join(dir, "plan.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewMetaPlanLoader()
	mp, err := loader.Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if mp.Name != "test-plan" {
		t.Errorf("Name = %q, want %q", mp.Name, "test-plan")
	}
	if mp.Description != "A test plan" {
		t.Errorf("Description = %q, want %q", mp.Description, "A test plan")
	}
	if mp.Status != "completed" {
		t.Errorf("Status = %q, want %q", mp.Status, "completed")
	}
	if len(mp.Stages) != 1 {
		t.Fatalf("len(Stages) = %d, want 1", len(mp.Stages))
	}
	if mp.Stages[0].Name != "deploy" {
		t.Errorf("Stages[0].Name = %q, want %q", mp.Stages[0].Name, "deploy")
	}
}

func TestMetaPlan_Load_MissingFile(t *testing.T) {
	t.Parallel()
	loader := NewMetaPlanLoader()
	_, err := loader.Load("/nonexistent/path/plan.yaml")
	if err == nil {
		t.Fatal("Load() expected error for missing file, got nil")
	}
}
