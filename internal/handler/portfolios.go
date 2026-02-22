package handler

import (
	"log"
	"net/http"
	"sort"

	"github.com/alexandremahdhaoui/forge-ui/internal/metaplan"
	"github.com/alexandremahdhaoui/forge-ui/internal/model"
	"github.com/alexandremahdhaoui/forge-ui/internal/portfolio"
	"github.com/alexandremahdhaoui/forge-ui/internal/portfolioconfig"
	"github.com/alexandremahdhaoui/forge-ui/internal/wsconfig"
)

// HandlePortfolios handles GET /portfolios.
func (h *Handler) HandlePortfolios(w http.ResponseWriter, r *http.Request) {
	portfolios, err := portfolio.List(h.BaseDir)
	if err != nil {
		http.Error(w, "failed to list portfolios: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sortMode := r.URL.Query().Get("sort")
	if sortMode != "name" {
		sortMode = "time"
	}

	var globalStats model.PortfoliosStats
	globalStats.TotalPortfolios = len(portfolios)

	for i := range portfolios {
		p := &portfolios[i]

		rewriteRepoLinks(p.Workspaces, p.Name)

		cacheKeyFn := func(wsName string) string {
			return p.Name + "/" + wsName
		}

		totalRepos, dirtyRepos, totalTests, passed, failed := enrichWorkspaces(p.Workspaces, h.Cache, cacheKeyFn)

		for j := range p.Workspaces {
			ws := &p.Workspaces[j]
			cfg, err := wsconfig.Load(ws.Path)
			if err != nil {
				log.Printf("wsconfig.Load(%s): %v", ws.Path, err)
			}
			ws.Description = cfg.Description

			mps, err := metaplan.LoadAll(ws.Path)
			if err != nil {
				log.Printf("metaplan.LoadAll(%s): %v", ws.Path, err)
			}
			ws.MetaPlans = mps

			// Compute workspace progress.
			var wsTotalTasks, wsDoneTasks int
			for _, mp := range mps {
				for _, stage := range mp.Stages {
					for _, repo := range stage.Repos {
						wsTotalTasks += repo.TasksTotal
						wsDoneTasks += repo.TasksDone
					}
				}
			}
			pct := 0
			if wsTotalTasks > 0 {
				pct = (wsDoneTasks * 100) / wsTotalTasks
			}
			ws.Progress = model.WorkspaceProgress{
				MetaPlanCount: len(mps),
				TasksTotal:    wsTotalTasks,
				TasksDone:     wsDoneTasks,
				PercentDone:   pct,
			}
		}

		p.Stats = model.WorkspacesStats{
			TotalWorkspaces: len(p.Workspaces),
			TotalRepos:      totalRepos,
			DirtyRepos:      dirtyRepos,
			TotalTests:      totalTests,
			Passed:          passed,
			Failed:          failed,
		}

		// Load portfolio config.
		pcfg, err := portfolioconfig.Load(p.Path)
		if err != nil {
			log.Printf("portfolioconfig.Load(%s): %v", p.Path, err)
		}
		p.Description = pcfg.Description

		// Aggregate portfolio-level progress from workspaces.
		var pProgress model.PortfolioProgress
		for _, ws := range p.Workspaces {
			pProgress.TasksTotal += ws.Progress.TasksTotal
			pProgress.TasksDone += ws.Progress.TasksDone
			for _, mp := range ws.MetaPlans {
				pProgress.TotalMetaPlans++
				switch mp.Status {
				case "in_progress":
					pProgress.ActiveMetaPlans++
				case "completed":
					pProgress.CompletedMetaPlans++
				}
			}
		}
		if pProgress.TasksTotal > 0 {
			pProgress.PercentDone = (pProgress.TasksDone * 100) / pProgress.TasksTotal
		}
		p.Stats.Portfolio = pProgress

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
	}

	var portfolioProgress model.PortfolioProgress
	for _, p := range portfolios {
		portfolioProgress.TotalMetaPlans += p.Stats.Portfolio.TotalMetaPlans
		portfolioProgress.ActiveMetaPlans += p.Stats.Portfolio.ActiveMetaPlans
		portfolioProgress.CompletedMetaPlans += p.Stats.Portfolio.CompletedMetaPlans
		portfolioProgress.TasksTotal += p.Stats.Portfolio.TasksTotal
		portfolioProgress.TasksDone += p.Stats.Portfolio.TasksDone
	}
	if portfolioProgress.TasksTotal > 0 {
		portfolioProgress.PercentDone = (portfolioProgress.TasksDone * 100) / portfolioProgress.TasksTotal
	}
	globalStats.Portfolio = portfolioProgress

	data := model.PortfoliosPageData{
		Portfolios:   portfolios,
		Stats:        globalStats,
		SortMode:     sortMode,
		DarkMode:     isDarkMode(r),
		HomeURL:      h.HomeURL,
		LightPalette: lightPalette(r),
	}

	h.render(w, "portfolios", data)
}

// HandlePortfolio handles GET /portfolios/{name}.
func (h *Handler) HandlePortfolio(w http.ResponseWriter, r *http.Request) {
	pName := r.PathValue("name")
	if pName == "" {
		http.Error(w, "missing portfolio name", http.StatusBadRequest)
		return
	}

	data, err := portfolio.Get(h.BaseDir, pName)
	if err != nil {
		http.Error(w, "failed to get portfolio: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Load portfolio config.
	pcfg, err := portfolioconfig.Load(data.Path)
	if err != nil {
		log.Printf("portfolioconfig.Load(%s): %v", data.Path, err)
	}
	data.Description = pcfg.Description

	sortMode := r.URL.Query().Get("sort")
	if sortMode != "name" {
		sortMode = "time"
	}

	rewriteRepoLinks(data.Workspaces, pName)

	cacheKeyFn := func(wsName string) string {
		return pName + "/" + wsName
	}

	totalRepos, dirtyRepos, totalTests, passed, failed := enrichWorkspaces(data.Workspaces, h.Cache, cacheKeyFn)

	for j := range data.Workspaces {
		ws := &data.Workspaces[j]
		cfg, err := wsconfig.Load(ws.Path)
		if err != nil {
			log.Printf("wsconfig.Load(%s): %v", ws.Path, err)
		}
		ws.Description = cfg.Description

		mps, err := metaplan.LoadAll(ws.Path)
		if err != nil {
			log.Printf("metaplan.LoadAll(%s): %v", ws.Path, err)
		}
		ws.MetaPlans = mps

		// Compute workspace progress.
		var wsTotalTasks, wsDoneTasks int
		for _, mp := range mps {
			for _, stage := range mp.Stages {
				for _, repo := range stage.Repos {
					wsTotalTasks += repo.TasksTotal
					wsDoneTasks += repo.TasksDone
				}
			}
		}
		pct := 0
		if wsTotalTasks > 0 {
			pct = (wsDoneTasks * 100) / wsTotalTasks
		}
		ws.Progress = model.WorkspaceProgress{
			MetaPlanCount: len(mps),
			TasksTotal:    wsTotalTasks,
			TasksDone:     wsDoneTasks,
			PercentDone:   pct,
		}
	}

	data.Stats = model.WorkspacesStats{
		TotalWorkspaces: len(data.Workspaces),
		TotalRepos:      totalRepos,
		DirtyRepos:      dirtyRepos,
		TotalTests:      totalTests,
		Passed:          passed,
		Failed:          failed,
	}

	// Aggregate portfolio-level progress from workspaces.
	var pProgress model.PortfolioProgress
	for _, ws := range data.Workspaces {
		pProgress.TasksTotal += ws.Progress.TasksTotal
		pProgress.TasksDone += ws.Progress.TasksDone
		for _, mp := range ws.MetaPlans {
			pProgress.TotalMetaPlans++
			switch mp.Status {
			case "in_progress":
				pProgress.ActiveMetaPlans++
			case "completed":
				pProgress.CompletedMetaPlans++
			}
		}
	}
	if pProgress.TasksTotal > 0 {
		pProgress.PercentDone = (pProgress.TasksDone * 100) / pProgress.TasksTotal
	}
	data.Stats.Portfolio = pProgress

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
	data.DarkMode = isDarkMode(r)
	data.HomeURL = h.HomeURL
	data.LightPalette = lightPalette(r)

	h.render(w, "portfolio", data)
}
