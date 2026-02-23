package controller

import (
	"path/filepath"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// ForgeService provides business logic for forge (repo detail) pages.
type ForgeService interface {
	GetForge(baseDir, portfolio, workspace, repo string) (types.ForgePageData, error)
}

type forgeService struct {
	forgeLoader adapter.ForgeLoader
}

// NewForgeService creates a ForgeService.
func NewForgeService(fl adapter.ForgeLoader) ForgeService {
	return &forgeService{forgeLoader: fl}
}

func (s *forgeService) GetForge(baseDir, portfolio, workspace, repo string) (types.ForgePageData, error) {
	var repoPath string
	if portfolio == "default" {
		repoPath = filepath.Join(baseDir, workspace, repo)
	} else {
		repoPath = filepath.Join(baseDir, portfolio, workspace, repo)
	}

	data, err := s.forgeLoader.Load(repoPath)
	if err != nil {
		return types.ForgePageData{}, err
	}

	data.WorkspaceName = workspace
	data.RepoName = repo
	data.PortfolioName = portfolio

	// Compute test statistics from reports.
	var stats types.ForgeStats
	var coverageSum float64
	var coverageCount int
	stageSet := make(map[string]struct{})

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

	return data, nil
}
