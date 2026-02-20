package repoplan

import (
	"os"
	"path/filepath"
	"testing"
)

func makePlanDir(t *testing.T, repoPath, planName, content string) {
	t.Helper()
	dir := filepath.Join(repoPath, ".forge-ai", "plan", planName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if content != "" {
		if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadAll_ValidPlans(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	makePlanDir(t, dir, "alpha", `# Alpha Plan

- [x] Task 1
- [x] Task 2
- [ ] Task 3
`)
	makePlanDir(t, dir, "beta", `# Beta Plan

- [ ] Task A
- [ ] Task B
`)

	plans, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("len(plans) = %d, want 2", len(plans))
	}

	// Sorted: alpha < beta
	if plans[0].Name != "alpha" {
		t.Errorf("plans[0].Name = %q, want %q", plans[0].Name, "alpha")
	}
	if plans[0].TasksTotal != 3 {
		t.Errorf("plans[0].TasksTotal = %d, want 3", plans[0].TasksTotal)
	}
	if plans[0].TasksDone != 2 {
		t.Errorf("plans[0].TasksDone = %d, want 2", plans[0].TasksDone)
	}

	if plans[1].Name != "beta" {
		t.Errorf("plans[1].Name = %q, want %q", plans[1].Name, "beta")
	}
	if plans[1].TasksTotal != 2 {
		t.Errorf("plans[1].TasksTotal = %d, want 2", plans[1].TasksTotal)
	}
	if plans[1].TasksDone != 0 {
		t.Errorf("plans[1].TasksDone = %d, want 0", plans[1].TasksDone)
	}
}

func TestLoadAll_MissingDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	plans, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("len(plans) = %d, want 0", len(plans))
	}
}

func TestLoadAll_EmptyTasksMd(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	makePlanDir(t, dir, "empty-plan", "# No tasks here\n\nJust notes.\n")

	plans, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	// No task markers, so no plan entries emitted.
	if len(plans) != 0 {
		t.Errorf("len(plans) = %d, want 0", len(plans))
	}
}

func TestLoadAll_MixedTasks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	makePlanDir(t, dir, "mixed", `# Mixed

- [x] Done task
- [ ] Pending task
- [x] Another done
- Some non-task line
- [ ] One more pending
`)

	plans, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("len(plans) = %d, want 1", len(plans))
	}
	if plans[0].TasksTotal != 4 {
		t.Errorf("TasksTotal = %d, want 4", plans[0].TasksTotal)
	}
	if plans[0].TasksDone != 2 {
		t.Errorf("TasksDone = %d, want 2", plans[0].TasksDone)
	}
}

func TestLoadAll_IndentedTasks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	makePlanDir(t, dir, "indented", `# Indented

- [x] Top-level done
  - [x] Indented done
  - [ ] Indented pending
    - [ ] Deeply indented pending
`)

	plans, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("len(plans) = %d, want 1", len(plans))
	}
	if plans[0].TasksTotal != 4 {
		t.Errorf("TasksTotal = %d, want 4", plans[0].TasksTotal)
	}
	if plans[0].TasksDone != 2 {
		t.Errorf("TasksDone = %d, want 2", plans[0].TasksDone)
	}
}

func TestLoadSummary_Aggregation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	makePlanDir(t, dir, "plan-a", `- [x] Done
- [ ] Pending
`)
	makePlanDir(t, dir, "plan-b", `- [x] Done 1
- [x] Done 2
- [ ] Pending
`)

	summary, err := LoadSummary(dir, "my-repo")
	if err != nil {
		t.Fatalf("LoadSummary() error: %v", err)
	}
	if summary.RepoName != "my-repo" {
		t.Errorf("RepoName = %q, want %q", summary.RepoName, "my-repo")
	}
	if len(summary.Plans) != 2 {
		t.Fatalf("len(Plans) = %d, want 2", len(summary.Plans))
	}
	if summary.TasksTotal != 5 {
		t.Errorf("TasksTotal = %d, want 5", summary.TasksTotal)
	}
	if summary.TasksDone != 3 {
		t.Errorf("TasksDone = %d, want 3", summary.TasksDone)
	}
}
