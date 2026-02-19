package handler

import (
	"net/http"
	"path/filepath"

	forgepkg "github.com/alexandremahdhaoui/forge-ui/internal/forge"
	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

// HandleForge handles GET /workspaces/{ws}/repos/{repo}.
func (h *Handler) HandleForge(w http.ResponseWriter, r *http.Request) {
	ws := r.PathValue("ws")
	repo := r.PathValue("repo")
	if ws == "" || repo == "" {
		http.Error(w, "workspace and repo name required", http.StatusBadRequest)
		return
	}

	repoPath := filepath.Join(h.BaseDir, ws, repo)

	data, err := forgepkg.Load(repoPath)
	if err != nil {
		http.Error(w, "failed to load forge data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data.WorkspaceName = ws
	data.RepoName = repo

	// Compute test statistics from reports.
	var stats model.ForgeStats
	var coverageSum float64
	var coverageCount int
	stageSet := make(map[string]struct{})

	// Build stage status map for heatmap (newest status per stage).
	stageStatusMap := make(map[string]string)
	for _, rpt := range data.TestReports {
		stats.TotalTests += rpt.Stats.Total
		stats.Passed += rpt.Stats.Passed
		stats.Failed += rpt.Stats.Failed
		stats.Skipped += rpt.Stats.Skipped
		stageSet[rpt.Stage] = struct{}{}
		if rpt.Coverage.Enabled {
			coverageSum += rpt.Coverage.Percentage
			coverageCount++
		}
		// Record newest status per stage (reports are sorted newest-first).
		if _, seen := stageStatusMap[rpt.Stage]; !seen {
			stageStatusMap[rpt.Stage] = rpt.Status
		}
	}
	if coverageCount > 0 {
		stats.AvgCoverage = coverageSum / float64(coverageCount)
		stats.HasCoverage = true
	}
	stats.StageCount = len(stageSet)
	data.Stats = stats
	data.StageStatusMap = stageStatusMap
	data.DarkMode = isDarkMode(r)

	h.render(w, "forge", data)
}
