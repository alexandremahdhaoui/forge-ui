package refresher

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/cache"
)

func TestRefresher_DefaultConfig(t *testing.T) {
	t.Parallel()

	r := New(cache.New(), Config{BaseDir: "/nonexistent"})
	if got, want := r.cfg.Interval, 1*time.Minute; got != want {
		t.Errorf("Interval = %v, want %v", got, want)
	}
	if got, want := r.cfg.NumWorkers, 3; got != want {
		t.Errorf("NumWorkers = %d, want %d", got, want)
	}
}

func TestRefresher_CustomConfig(t *testing.T) {
	t.Parallel()

	r := New(cache.New(), Config{
		BaseDir:    "/tmp",
		Interval:   30 * time.Second,
		NumWorkers: 5,
	})
	if got, want := r.cfg.Interval, 30*time.Second; got != want {
		t.Errorf("Interval = %v, want %v", got, want)
	}
	if got, want := r.cfg.NumWorkers, 5; got != want {
		t.Errorf("NumWorkers = %d, want %d", got, want)
	}
}

func TestRefresher_StartAndStop(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	c := cache.New()
	r := New(c, Config{
		BaseDir:    baseDir,
		Interval:   1 * time.Hour,
		NumWorkers: 1,
	})

	// Start should complete without hanging (no workspaces to refresh).
	done := make(chan struct{})
	go func() {
		r.Start()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start() did not return within 5 seconds")
	}

	// Stop should complete without hanging.
	stopDone := make(chan struct{})
	go func() {
		r.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds")
	}
}

func TestRefresher_PopulatesCache(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create a workspace directory with go.work.
	wsDir := filepath.Join(baseDir, "ws1")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "go.work"), []byte("go 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a git repo inside the workspace.
	repoDir := filepath.Join(wsDir, "repo-a")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "config", "user.email", "test@test.com")
	runGitCmd(t, repoDir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repoDir, "add", ".")
	runGitCmd(t, repoDir, "commit", "-m", "init")

	c := cache.New()
	r := New(c, Config{
		BaseDir:    baseDir,
		Interval:   1 * time.Hour,
		NumWorkers: 1,
	})

	r.Start()
	defer r.Stop()

	summary, found := c.GetRepoSummary("ws1", "repo-a")
	if !found {
		t.Fatal("expected repo-a to be found in cache after initial refresh")
	}
	if summary.Branch == "" {
		t.Error("expected Branch to be non-empty")
	}
	if summary.Name != "repo-a" {
		t.Errorf("Name = %q, want %q", summary.Name, "repo-a")
	}
	if summary.Path != repoDir {
		t.Errorf("Path = %q, want %q", summary.Path, repoDir)
	}
}

// runGitCmd runs a git command in the given directory. It isolates the test
// from the host git config by setting GIT_CONFIG_GLOBAL and GIT_CONFIG_SYSTEM.
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}
