package git

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

// RepoInfo populates the git-related fields of model.RepoSummary by running
// git commands in the given repo directory. It does NOT set Name, Path,
// HasForge, or RepoLink. Individual git command failures use fallback values;
// the function always returns (result, nil).
func RepoInfo(repoPath string) (model.RepoSummary, error) {
	var result model.RepoSummary

	// 0. Fetch from origin to ensure remote refs are fresh.
	// Errors are ignored: if the network is down or there is no remote,
	// we fall back to stale local data.
	runGit(repoPath, "fetch", "origin")

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
			result.StatusFiles = append(result.StatusFiles, model.StatusEntry{
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
			entry := model.LogEntry{Hash: parts[0]}
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

	return result, nil
}

// runGit executes a git command with -C <repoPath> and returns stdout as a string.
// It sets safe.directory=* so mounted volumes owned by a different UID work.
func runGit(repoPath string, args ...string) (string, error) {
	cmdArgs := append([]string{"-c", "safe.directory=*", "-C", repoPath}, args...)
	cmd := exec.Command("git", cmdArgs...)
	output, err := cmd.Output()
	return string(output), err
}
