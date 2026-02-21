package controller

import (
	"path/filepath"
	"sort"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// WorkspaceService provides business logic for workspace pages.
type WorkspaceService interface {
	GetWorkspace(baseDir, portfolio, workspace, sortMode string) (types.WorkspacePageData, error)
}

type workspaceService struct {
	workspaceDisc adapter.WorkspaceDiscovery
	cache         adapter.Cache
	forgeLoader   adapter.ForgeLoader
}

// NewWorkspaceService creates a WorkspaceService.
func NewWorkspaceService(ws adapter.WorkspaceDiscovery, c adapter.Cache, fl adapter.ForgeLoader) WorkspaceService {
	return &workspaceService{
		workspaceDisc: ws,
		cache:         c,
		forgeLoader:   fl,
	}
}

func (s *workspaceService) GetWorkspace(baseDir, portfolio, workspace, sortMode string) (types.WorkspacePageData, error) {
	wsBaseDir := baseDir
	if portfolio != "default" {
		wsBaseDir = filepath.Join(baseDir, portfolio)
	}

	data, err := s.workspaceDisc.Get(wsBaseDir, workspace)
	if err != nil {
		return types.WorkspacePageData{}, err
	}

	cacheKey := portfolio + "/" + workspace

	// Enrich each repo with cached git information.
	for i, repo := range data.Repos {
		if cached, ok := s.cache.GetRepoSummary(cacheKey, repo.Name); ok {
			data.Repos[i].Branch = cached.Branch
			data.Repos[i].IsDirty = cached.IsDirty
			data.Repos[i].StatusFiles = cached.StatusFiles
			data.Repos[i].DiffStat = cached.DiffStat
			data.Repos[i].RecentLogs = cached.RecentLogs
			data.Repos[i].Ahead = cached.Ahead
			data.Repos[i].Behind = cached.Behind
			data.Repos[i].HasUpstream = cached.HasUpstream
			data.Repos[i].LastCommitTime = cached.LastCommitTime
		}
	}

	// Sort by last commit time if requested.
	if sortMode == "time" {
		sort.Slice(data.Repos, func(a, b int) bool {
			return data.Repos[a].LastCommitTime.After(data.Repos[b].LastCommitTime)
		})
	}

	// Rewrite repo links for portfolio-scoped URLs.
	for i := range data.Repos {
		data.Repos[i].RepoLink = "/portfolios/" + portfolio + "/workspaces/" + workspace + "/repos/" + data.Repos[i].Name
	}

	// Load forge data for repos that have forge.yaml and build heatmap data.
	stageSeen := make(map[string]struct{})
	var allStages []string
	var stats types.WorkspaceStats
	stats.TotalRepos = len(data.Repos)

	for _, repo := range data.Repos {
		if !repo.HasForge {
			continue
		}
		stats.ForgeRepos++

		forgeData, err := s.forgeLoader.Load(repo.Path)
		if err != nil {
			continue
		}

		stageResults := make(map[string]string)
		for _, rpt := range forgeData.TestReports {
			if _, seen := stageResults[rpt.Stage]; !seen {
				stageResults[rpt.Stage] = rpt.Status
			}
			stats.TotalTests += rpt.Stats.Total
			stats.Passed += rpt.Stats.Passed
			stats.Failed += rpt.Stats.Failed
			stats.Skipped += rpt.Stats.Skipped
		}

		for _, ts := range forgeData.Spec.Test {
			if _, seen := stageSeen[ts.Name]; !seen {
				stageSeen[ts.Name] = struct{}{}
				allStages = append(allStages, ts.Name)
			}
		}
		for _, rpt := range forgeData.TestReports {
			if _, seen := stageSeen[rpt.Stage]; !seen {
				stageSeen[rpt.Stage] = struct{}{}
				allStages = append(allStages, rpt.Stage)
			}
		}

		data.RepoForge = append(data.RepoForge, types.RepoForgeStats{
			RepoName:     repo.Name,
			RepoLink:     "/portfolios/" + portfolio + "/workspaces/" + workspace + "/repos/" + repo.Name,
			StageResults: stageResults,
		})
	}

	data.AllStages = allStages
	data.Stats = stats
	data.PortfolioName = portfolio
	data.SortMode = sortMode

	return data, nil
}
