//go:build !js || !wasm

package controller

import (
	"sort"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// PortfolioService provides business logic for portfolio pages.
type PortfolioService interface {
	ListPortfolios(baseDir, sortMode string) (types.PortfoliosPageData, error)
	GetPortfolio(baseDir, name, sortMode string) (types.PortfolioPageData, error)
}

type portfolioService struct {
	portfolioDisc   adapter.PortfolioDiscovery
	cache           adapter.Cache
	forgeLoader     adapter.ForgeLoader
	wsConfigLoader  adapter.WsConfigLoader
	metaPlanLoader  adapter.MetaPlanLoader
	portfolioConfig adapter.PortfolioConfigLoader
}

// NewPortfolioService creates a PortfolioService.
func NewPortfolioService(
	pd adapter.PortfolioDiscovery,
	c adapter.Cache,
	fl adapter.ForgeLoader,
	wc adapter.WsConfigLoader,
	mp adapter.MetaPlanLoader,
	pc adapter.PortfolioConfigLoader,
) PortfolioService {
	return &portfolioService{
		portfolioDisc:   pd,
		cache:           c,
		forgeLoader:     fl,
		wsConfigLoader:  wc,
		metaPlanLoader:  mp,
		portfolioConfig: pc,
	}
}

func (s *portfolioService) ListPortfolios(baseDir, sortMode string) (types.PortfoliosPageData, error) {
	portfolios, err := s.portfolioDisc.List(baseDir)
	if err != nil {
		return types.PortfoliosPageData{}, err
	}

	var globalStats types.PortfoliosStats
	globalStats.TotalPortfolios = len(portfolios)

	for i := range portfolios {
		p := &portfolios[i]

		// Load portfolio config for description.
		if pc, err := s.portfolioConfig.Load(p.Path); err == nil {
			p.Description = pc.Description
		}

		rewriteRepoLinks(p.Workspaces, p.Name)

		cacheKeyFn := func(wsName string) string {
			return p.Name + "/" + wsName
		}

		totalRepos, dirtyRepos, totalTests, passed, failed := enrichWorkspaces(p.Workspaces, s.cache, s.forgeLoader, cacheKeyFn)
		s.enrichWorkspaceOrchestration(p.Workspaces)

		p.Stats = types.WorkspacesStats{
			TotalWorkspaces: len(p.Workspaces),
			TotalRepos:      totalRepos,
			DirtyRepos:      dirtyRepos,
			TotalTests:      totalTests,
			Passed:          passed,
			Failed:          failed,
			Portfolio:        aggregatePortfolioProgress(p.Workspaces),
		}

		if sortMode == "time" {
			for j := range p.Workspaces {
				sort.Slice(p.Workspaces[j].Repos, func(a, b int) bool {
					return p.Workspaces[j].Repos[a].LastCommitTime.After(p.Workspaces[j].Repos[b].LastCommitTime)
				})
			}
			sort.Slice(p.Workspaces, func(a, b int) bool {
				return maxCommitTime(p.Workspaces[a].Repos).After(maxCommitTime(p.Workspaces[b].Repos))
			})
		}

		globalStats.TotalWorkspaces += p.Stats.TotalWorkspaces
		globalStats.TotalRepos += p.Stats.TotalRepos
		globalStats.DirtyRepos += p.Stats.DirtyRepos
		globalStats.TotalTests += p.Stats.TotalTests
		globalStats.Passed += p.Stats.Passed
		globalStats.Failed += p.Stats.Failed
		globalStats.Portfolio.TotalMetaPlans += p.Stats.Portfolio.TotalMetaPlans
		globalStats.Portfolio.ActiveMetaPlans += p.Stats.Portfolio.ActiveMetaPlans
		globalStats.Portfolio.CompletedMetaPlans += p.Stats.Portfolio.CompletedMetaPlans
		globalStats.Portfolio.TasksTotal += p.Stats.Portfolio.TasksTotal
		globalStats.Portfolio.TasksDone += p.Stats.Portfolio.TasksDone
	}

	if globalStats.Portfolio.TasksTotal > 0 {
		globalStats.Portfolio.PercentDone = (globalStats.Portfolio.TasksDone * 100) / globalStats.Portfolio.TasksTotal
	}

	return types.PortfoliosPageData{
		Portfolios: portfolios,
		Stats:      globalStats,
		SortMode:   sortMode,
	}, nil
}

func (s *portfolioService) GetPortfolio(baseDir, name, sortMode string) (types.PortfolioPageData, error) {
	data, err := s.portfolioDisc.Get(baseDir, name)
	if err != nil {
		return types.PortfolioPageData{}, err
	}

	// Load portfolio config for description.
	if pc, err := s.portfolioConfig.Load(data.Path); err == nil {
		data.Description = pc.Description
	}

	rewriteRepoLinks(data.Workspaces, name)

	cacheKeyFn := func(wsName string) string {
		return name + "/" + wsName
	}

	totalRepos, dirtyRepos, totalTests, passed, failed := enrichWorkspaces(data.Workspaces, s.cache, s.forgeLoader, cacheKeyFn)
	s.enrichWorkspaceOrchestration(data.Workspaces)

	data.Stats = types.WorkspacesStats{
		TotalWorkspaces: len(data.Workspaces),
		TotalRepos:      totalRepos,
		DirtyRepos:      dirtyRepos,
		TotalTests:      totalTests,
		Passed:          passed,
		Failed:          failed,
		Portfolio:        aggregatePortfolioProgress(data.Workspaces),
	}

	if sortMode == "time" {
		for i := range data.Workspaces {
			sort.Slice(data.Workspaces[i].Repos, func(a, b int) bool {
				return data.Workspaces[i].Repos[a].LastCommitTime.After(data.Workspaces[i].Repos[b].LastCommitTime)
			})
		}
		sort.Slice(data.Workspaces, func(i, j int) bool {
			return maxCommitTime(data.Workspaces[i].Repos).After(maxCommitTime(data.Workspaces[j].Repos))
		})
	}

	data.SortMode = sortMode

	return data, nil
}

// enrichWorkspaces enriches a slice of WorkspaceSummary in-place with cached
// git info and forge heatmap data.
func enrichWorkspaces(
	workspaces []types.WorkspaceSummary,
	c adapter.Cache,
	forgeLoader adapter.ForgeLoader,
	cacheKey func(wsName string) string,
) (totalRepos, dirtyRepos, totalTests, passed, failed int) {
	for i := range workspaces {
		ws := &workspaces[i]
		totalRepos += ws.RepoCount

		for j := range ws.Repos {
			repo := &ws.Repos[j]
			if cached, ok := c.GetRepoOverview(cacheKey(ws.Name), repo.Name); ok {
				repo.Branch = cached.Branch
				repo.IsDirty = cached.IsDirty
				repo.Ahead = cached.Ahead
				repo.Behind = cached.Behind
				repo.HasUpstream = cached.HasUpstream
				repo.LastCommitTime = cached.LastCommitTime
			}
			if repo.IsDirty {
				dirtyRepos++
			}
		}

		stageSeen := make(map[string]struct{})

		for _, repo := range ws.Repos {
			if !repo.HasForge {
				continue
			}
			forgeData, err := forgeLoader.Load(repo.Path)
			if err != nil {
				continue
			}

			stageResults := make(map[string]string)
			for _, rpt := range forgeData.TestReports {
				if _, seen := stageResults[rpt.Stage]; !seen {
					stageResults[rpt.Stage] = rpt.Status
				}
				totalTests += rpt.Stats.Total
				passed += rpt.Stats.Passed
				failed += rpt.Stats.Failed
			}

			for _, ts := range forgeData.Spec.Test {
				if _, seen := stageSeen[ts.Name]; !seen {
					stageSeen[ts.Name] = struct{}{}
					ws.AllStages = append(ws.AllStages, ts.Name)
				}
			}
			for _, rpt := range forgeData.TestReports {
				if _, seen := stageSeen[rpt.Stage]; !seen {
					stageSeen[rpt.Stage] = struct{}{}
					ws.AllStages = append(ws.AllStages, rpt.Stage)
				}
			}

			ws.RepoForge = append(ws.RepoForge, types.RepoForgeStats{
				RepoName:     repo.Name,
				RepoLink:     repo.RepoLink,
				StageResults: stageResults,
			})
		}
	}

	return totalRepos, dirtyRepos, totalTests, passed, failed
}

// rewriteRepoLinks rewrites all RepoLink fields to use portfolio-scoped URL paths.
func rewriteRepoLinks(workspaces []types.WorkspaceSummary, portfolioName string) {
	for i := range workspaces {
		for j := range workspaces[i].Repos {
			repo := &workspaces[i].Repos[j]
			repo.RepoLink = "/portfolios/" + portfolioName + "/workspaces/" + workspaces[i].Name + "/repos/" + repo.Name
		}
	}
}

// maxCommitTime returns the most recent LastCommitTime across all repos.
func maxCommitTime(repos []types.RepoOverview) time.Time {
	var max time.Time
	for _, r := range repos {
		if r.LastCommitTime.After(max) {
			max = r.LastCommitTime
		}
	}
	return max
}

// enrichWorkspaceOrchestration populates Description, MetaPlans, and Progress
// for each workspace using WsConfigLoader and MetaPlanLoader.
func (s *portfolioService) enrichWorkspaceOrchestration(workspaces []types.WorkspaceSummary) {
	for i := range workspaces {
		ws := &workspaces[i]
		cfg, err := s.wsConfigLoader.Load(ws.Path)
		if err == nil && cfg.Description != "" {
			ws.Description = cfg.Description
		}

		plans, err := s.metaPlanLoader.LoadAll(ws.Path)
		if err == nil && len(plans) > 0 {
			ws.MetaPlans = plans
			var tasksTotal, tasksDone int
			for _, mp := range plans {
				for _, st := range mp.Stages {
					for _, r := range st.Repos {
						tasksTotal += r.TasksTotal
						tasksDone += r.TasksDone
					}
				}
			}
			pct := 0
			if tasksTotal > 0 {
				pct = (tasksDone * 100) / tasksTotal
			}
			ws.Progress = types.WorkspaceProgress{
				MetaPlanCount: len(plans),
				TasksTotal:    tasksTotal,
				TasksDone:     tasksDone,
				PercentDone:   pct,
			}
		}
	}
}

// aggregatePortfolioProgress computes aggregate PortfolioProgress from workspaces.
func aggregatePortfolioProgress(workspaces []types.WorkspaceSummary) types.PortfolioProgress {
	var pp types.PortfolioProgress
	for _, ws := range workspaces {
		pp.TotalMetaPlans += len(ws.MetaPlans)
		pp.TasksTotal += ws.Progress.TasksTotal
		pp.TasksDone += ws.Progress.TasksDone
		for _, mp := range ws.MetaPlans {
			switch mp.Status {
			case "in_progress":
				pp.ActiveMetaPlans++
			case "completed":
				pp.CompletedMetaPlans++
			}
		}
	}
	if pp.TasksTotal > 0 {
		pp.PercentDone = (pp.TasksDone * 100) / pp.TasksTotal
	}
	return pp
}
