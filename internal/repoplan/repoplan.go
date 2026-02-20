package repoplan

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

var (
	reDone    = regexp.MustCompile(`^\s*- \[x\]`)
	rePending = regexp.MustCompile(`^\s*- \[ \]`)
)

// LoadAll reads .forge-ai/plan/*/tasks.md from repoPath and returns a slice
// of RepoPlan sorted by name.
// Missing directory returns empty slice and nil error.
func LoadAll(repoPath string) ([]model.RepoPlan, error) {
	planDir := filepath.Join(repoPath, ".forge-ai", "plan")

	entries, err := os.ReadDir(planDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var plans []model.RepoPlan
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		tasksFile := filepath.Join(planDir, e.Name(), "tasks.md")
		total, done, err := countTasks(tasksFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			continue
		}
		if total == 0 {
			continue
		}

		plans = append(plans, model.RepoPlan{
			Name:       e.Name(),
			TasksTotal: total,
			TasksDone:  done,
		})
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Name < plans[j].Name
	})

	return plans, nil
}

// LoadSummary returns an aggregated RepoPlanSummary for a repo.
func LoadSummary(repoPath, repoName string) (model.RepoPlanSummary, error) {
	plans, err := LoadAll(repoPath)
	if err != nil {
		return model.RepoPlanSummary{}, err
	}

	var totalTasks, totalDone int
	for _, p := range plans {
		totalTasks += p.TasksTotal
		totalDone += p.TasksDone
	}

	return model.RepoPlanSummary{
		RepoName:   repoName,
		Plans:      plans,
		TasksTotal: totalTasks,
		TasksDone:  totalDone,
	}, nil
}

// countTasks scans a tasks.md file and returns (total, done).
// Total = done + pending. Lines matching `- [x]` count as done,
// lines matching `- [ ]` count as pending.
func countTasks(path string) (total, done int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if reDone.MatchString(line) {
			total++
			done++
		} else if rePending.MatchString(line) {
			total++
		}
	}

	return total, done, scanner.Err()
}
