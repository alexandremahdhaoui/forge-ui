//go:build !js || !wasm

package adapter

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// GitInfo provides git repository information.
type GitInfo interface {
	RepoInfo(repoPath string) (types.RepoSummary, error)
}

type gitInfoImpl struct{}

// NewGitInfo returns a GitInfo backed by real git commands.
func NewGitInfo() GitInfo {
	return &gitInfoImpl{}
}

func (g *gitInfoImpl) RepoInfo(repoPath string) (types.RepoSummary, error) {
	var result types.RepoSummary

	// 0. Fetch from origin to ensure remote refs are fresh.
	_, _ = runGit(repoPath, "fetch", "origin")

	// 1. Branch
	branchOut, err := runGit(repoPath, "branch", "--show-current")
	if err != nil {
		result.Branch = "(unknown)"
	} else {
		result.Branch = strings.TrimSpace(branchOut)
	}

	// 2. Status
	statusOut, err := runGit(repoPath, "status", "--porcelain")
	if err == nil {
		lines := strings.Split(statusOut, "\n")
		for _, line := range lines {
			if len(line) < 3 {
				continue
			}
			code := strings.TrimRight(line[:2], " ")
			filePath := line[3:]
			result.StatusFiles = append(result.StatusFiles, types.StatusEntry{
				Code:     code,
				FilePath: filePath,
			})
		}
		result.IsDirty = len(result.StatusFiles) > 0
	}

	// 3. Log
	logOut, err := runGit(repoPath, "log", "--oneline", "-10")
	if err == nil {
		lines := strings.Split(logOut, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			entry := types.LogEntry{Hash: parts[0]}
			if len(parts) > 1 {
				entry.Message = parts[1]
			}
			result.RecentLogs = append(result.RecentLogs, entry)
		}
	}

	// 4. Diff
	diffOut, err := runGit(repoPath, "diff", "--stat")
	if err == nil {
		result.DiffStat = strings.TrimRight(diffOut, "\n")
	}

	// 5. Ahead/behind upstream
	revOut, err := runGit(repoPath, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(revOut))
		if len(parts) == 2 {
			result.HasUpstream = true
			result.Behind, _ = strconv.Atoi(parts[0])
			result.Ahead, _ = strconv.Atoi(parts[1])
		}
	}

	// 6. Last commit time
	commitTimeOut, err := runGit(repoPath, "log", "-1", "--format=%cI")
	if err == nil {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(commitTimeOut)); err == nil {
			result.LastCommitTime = t
		}
	}

	// 7. If dirty, check file modification times so LastCommitTime reflects
	// actual working directory activity, not just the last commit.
	// git status --porcelain already respects .gitignore, so no risk of
	// walking into node_modules or .git.
	if result.IsDirty {
		for _, sf := range result.StatusFiles {
			p := sf.FilePath
			// Renames show as "old -> new"; use the new path.
			if idx := strings.Index(p, " -> "); idx >= 0 {
				p = p[idx+4:]
			}
			info, err := os.Stat(filepath.Join(repoPath, p))
			if err != nil {
				continue // deleted file or unreadable
			}
			if info.ModTime().After(result.LastCommitTime) {
				result.LastCommitTime = info.ModTime()
			}
		}
	}

	return result, nil
}

// runGit executes a git command with -C <repoPath> and returns stdout as a string.
func runGit(repoPath string, args ...string) (string, error) {
	cmdArgs := append([]string{"-c", "safe.directory=*", "-C", repoPath}, args...)
	cmd := exec.Command("git", cmdArgs...)
	output, err := cmd.Output()
	return string(output), err
}
