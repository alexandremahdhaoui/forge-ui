package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testRepo holds paths and metadata for a temporary git repository used in tests.
type testRepo struct {
	dir       string // path to the working tree
	remoteDir string // path to the bare remote (empty until addRemote is called)
	branch    string // default branch name detected after init
}

// newTestRepo creates a temporary git repo with one initial commit.
// It detects the default branch name from the system configuration.
func newTestRepo(t *testing.T) *testRepo {
	t.Helper()

	dir := t.TempDir()

	runGitCmd(t, dir, "init")

	branch := strings.TrimSpace(runGitCmd(t, dir, "branch", "--show-current"))

	runGitCmd(t, dir, "config", "user.email", "test@example.com")
	runGitCmd(t, dir, "config", "user.name", "Test User")

	// Create initial commit.
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("init"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "init")

	return &testRepo{dir: dir, branch: branch}
}

// addRemote creates a bare remote repository and pushes the current branch to it.
func (tr *testRepo) addRemote(t *testing.T) {
	t.Helper()

	remoteDir := t.TempDir()
	runGitCmd(t, remoteDir, "init", "--bare")

	runGitCmd(t, tr.dir, "remote", "add", "origin", remoteDir)
	runGitCmd(t, tr.dir, "push", "-u", "origin", "HEAD")

	tr.remoteDir = remoteDir
}

// makeCommit writes a file and commits it in the working tree.
func (tr *testRepo) makeCommit(t *testing.T, filename, content, message string) {
	t.Helper()

	fp := filepath.Join(tr.dir, filename)
	if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, tr.dir, "add", filename)
	runGitCmd(t, tr.dir, "commit", "-m", message)
}

// pushCommit makes a commit and pushes it to origin.
func (tr *testRepo) pushCommit(t *testing.T, filename, content, message string) {
	t.Helper()

	tr.makeCommit(t, filename, content, message)
	runGitCmd(t, tr.dir, "push")
}

// makeRemoteCommit simulates another developer pushing a commit to the remote.
// It clones the bare remote into a temporary directory, commits there, and pushes.
func (tr *testRepo) makeRemoteCommit(t *testing.T, filename, content, message string) {
	t.Helper()

	cloneDir := t.TempDir()
	runGitCmd(t, cloneDir, "clone", tr.remoteDir, ".")
	runGitCmd(t, cloneDir, "config", "user.email", "other@example.com")
	runGitCmd(t, cloneDir, "config", "user.name", "Other User")

	fp := filepath.Join(cloneDir, filename)
	if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, cloneDir, "add", filename)
	runGitCmd(t, cloneDir, "commit", "-m", message)
	runGitCmd(t, cloneDir, "push")
}

// runGitCmd executes a git command in the given directory and returns stdout.
// It isolates from host git configuration by setting GIT_CONFIG_GLOBAL and
// GIT_CONFIG_SYSTEM to /dev/null.
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"-c", "safe.directory=*", "-C", dir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)

	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		t.Fatalf("git %v failed: %v\nstderr: %s", args, err, stderr)
	}
	return string(out)
}
