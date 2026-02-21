package controller

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// refreshItem identifies a single workspace to refresh, including the portfolio
// it belongs to and the base directory where the workspace lives.
type refreshItem struct {
	PortfolioName string // always set, "default" for loose workspaces
	WorkspaceName string
	WorkspaceBase string // absolute path to parent of workspace dir
}

// RefresherConfig controls the refresher behavior.
type RefresherConfig struct {
	BaseDir    string
	Interval   time.Duration
	NumWorkers int
}

// Refresher discovers workspaces and repos on a timer, runs RepoInfo for each
// repo, and writes results to the cache. HTTP handlers read from the cache.
type Refresher struct {
	cache         adapter.Cache
	gitInfo       adapter.GitInfo
	portfolioDisc adapter.PortfolioDiscovery
	workspaceDisc adapter.WorkspaceDiscovery
	cfg           RefresherConfig
	queue         chan refreshItem
	done          chan struct{}
	wg            sync.WaitGroup
}

// NewRefresher creates a Refresher with sensible defaults for zero-valued config fields.
func NewRefresher(c adapter.Cache, gi adapter.GitInfo, pd adapter.PortfolioDiscovery, ws adapter.WorkspaceDiscovery, cfg RefresherConfig) *Refresher {
	if cfg.Interval <= 0 {
		cfg.Interval = 1 * time.Minute
	}
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 3
	}
	return &Refresher{
		cache:         c,
		gitInfo:       gi,
		portfolioDisc: pd,
		workspaceDisc: ws,
		cfg:           cfg,
		queue:         make(chan refreshItem, 100),
		done:          make(chan struct{}),
	}
}

// Start runs one synchronous refresh (blocking until complete), then spawns
// background workers and a scheduler goroutine.
func (r *Refresher) Start() {
	r.refreshAll()
	log.Printf("refresher: initial refresh complete")

	for i := 0; i < r.cfg.NumWorkers; i++ {
		r.wg.Add(1)
		go r.worker()
	}

	r.wg.Add(1)
	go r.scheduler()
}

// Stop signals all goroutines to exit and waits for them to finish.
func (r *Refresher) Stop() {
	close(r.done)
	r.wg.Wait()
}

// refreshAll refreshes every workspace sequentially across all portfolios.
func (r *Refresher) refreshAll() {
	portfolios, err := r.portfolioDisc.List(r.cfg.BaseDir)
	if err != nil {
		log.Printf("refresher: list portfolios: %v", err)
		return
	}
	for _, p := range portfolios {
		wsBase := filepath.Join(r.cfg.BaseDir, p.Name)
		if p.IsDefault {
			wsBase = r.cfg.BaseDir
		}
		for _, ws := range p.Workspaces {
			r.refreshWorkspace(refreshItem{
				PortfolioName: p.Name,
				WorkspaceName: ws.Name,
				WorkspaceBase: wsBase,
			})
		}
	}
}

// refreshWorkspace discovers repos in a workspace, calls RepoInfo for each,
// merges repo metadata from workspace.Get, and writes results to the cache.
func (r *Refresher) refreshWorkspace(item refreshItem) {
	data, err := r.workspaceDisc.Get(item.WorkspaceBase, item.WorkspaceName)
	if err != nil {
		log.Printf("refresher: get workspace %q: %v", item.WorkspaceName, err)
		return
	}

	cacheKey := item.PortfolioName + "/" + item.WorkspaceName

	summaries := make(map[string]types.RepoSummary)
	overviews := make(map[string]types.RepoOverview)

	for _, repo := range data.Repos {
		gitInfo, err := r.gitInfo.RepoInfo(repo.Path)
		if err != nil {
			log.Printf("refresher: repo info %q: %v", repo.Path, err)
			continue
		}

		gitInfo.Name = repo.Name
		gitInfo.Path = repo.Path
		gitInfo.HasForge = repo.HasForge
		gitInfo.RepoLink = repo.RepoLink
		summaries[repo.Name] = gitInfo

		overviews[repo.Name] = types.RepoOverview{
			Name:           repo.Name,
			WorkspaceName:  item.WorkspaceName,
			Path:           repo.Path,
			Branch:         gitInfo.Branch,
			IsDirty:        gitInfo.IsDirty,
			Ahead:          gitInfo.Ahead,
			Behind:         gitInfo.Behind,
			HasUpstream:    gitInfo.HasUpstream,
			HasForge:       repo.HasForge,
			RepoLink:       repo.RepoLink,
			LastCommitTime: gitInfo.LastCommitTime,
		}
	}

	r.cache.SetWorkspace(cacheKey, types.CacheWorkspaceData{
		Summaries: summaries,
		Overviews: overviews,
		UpdatedAt: time.Now(),
	})
}

// scheduler sends refresh items to the work queue on each tick.
func (r *Refresher) scheduler() {
	defer r.wg.Done()
	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			portfolios, err := r.portfolioDisc.List(r.cfg.BaseDir)
			if err != nil {
				log.Printf("refresher: list portfolios: %v", err)
				continue
			}
			for _, p := range portfolios {
				wsBase := filepath.Join(r.cfg.BaseDir, p.Name)
				if p.IsDefault {
					wsBase = r.cfg.BaseDir
				}
				for _, ws := range p.Workspaces {
					item := refreshItem{
						PortfolioName: p.Name,
						WorkspaceName: ws.Name,
						WorkspaceBase: wsBase,
					}
					select {
					case r.queue <- item:
					case <-r.done:
						return
					}
				}
			}
		}
	}
}

// worker reads refresh items from the queue and refreshes each one.
func (r *Refresher) worker() {
	defer r.wg.Done()
	for {
		select {
		case <-r.done:
			return
		case item := <-r.queue:
			r.refreshWorkspace(item)
		}
	}
}
