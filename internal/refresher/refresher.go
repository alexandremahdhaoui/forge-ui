package refresher

import (
	"log"
	"sync"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/cache"
	gitpkg "github.com/alexandremahdhaoui/forge-ui/internal/git"
	"github.com/alexandremahdhaoui/forge-ui/internal/model"
	"github.com/alexandremahdhaoui/forge-ui/internal/workspace"
)

// Config controls the refresher behavior.
type Config struct {
	BaseDir    string
	Interval   time.Duration
	NumWorkers int
}

// Refresher discovers workspaces and repos on a timer, runs RepoInfo for each
// repo, and writes results to the cache. HTTP handlers read from the cache.
type Refresher struct {
	cache *cache.Cache
	cfg   Config
	queue chan string
	done  chan struct{}
	wg    sync.WaitGroup
}

// New creates a Refresher with sensible defaults for zero-valued config fields.
func New(c *cache.Cache, cfg Config) *Refresher {
	if cfg.Interval <= 0 {
		cfg.Interval = 1 * time.Minute
	}
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 3
	}
	return &Refresher{
		cache: c,
		cfg:   cfg,
		queue: make(chan string, 100),
		done:  make(chan struct{}),
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

// refreshAll refreshes every workspace sequentially.
func (r *Refresher) refreshAll() {
	wsList, err := workspace.List(r.cfg.BaseDir)
	if err != nil {
		log.Printf("refresher: list workspaces: %v", err)
		return
	}
	for _, ws := range wsList {
		r.refreshWorkspace(ws.Name)
	}
}

// refreshWorkspace discovers repos in a workspace, calls RepoInfo for each,
// merges repo metadata from workspace.Get, and writes results to the cache.
func (r *Refresher) refreshWorkspace(wsName string) {
	data, err := workspace.Get(r.cfg.BaseDir, wsName)
	if err != nil {
		log.Printf("refresher: get workspace %q: %v", wsName, err)
		return
	}

	summaries := make(map[string]model.RepoSummary)
	overviews := make(map[string]model.RepoOverview)

	for _, repo := range data.Repos {
		gitInfo, err := gitpkg.RepoInfo(repo.Path)
		if err != nil {
			log.Printf("refresher: repo info %q: %v", repo.Path, err)
			continue
		}

		// RepoInfo does NOT set Name, Path, HasForge, or RepoLink.
		// Copy these from the repo discovered by workspace.Get.
		gitInfo.Name = repo.Name
		gitInfo.Path = repo.Path
		gitInfo.HasForge = repo.HasForge
		gitInfo.RepoLink = repo.RepoLink
		summaries[repo.Name] = gitInfo

		overviews[repo.Name] = model.RepoOverview{
			Name:           repo.Name,
			WorkspaceName:  wsName,
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

	r.cache.SetWorkspace(wsName, cache.WorkspaceData{
		Summaries: summaries,
		Overviews: overviews,
		UpdatedAt: time.Now(),
	})
}

// scheduler sends workspace names to the work queue on each tick.
func (r *Refresher) scheduler() {
	defer r.wg.Done()
	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			wsList, err := workspace.List(r.cfg.BaseDir)
			if err != nil {
				log.Printf("refresher: list workspaces: %v", err)
				continue
			}
			for _, ws := range wsList {
				select {
				case r.queue <- ws.Name:
				case <-r.done:
					return
				}
			}
		}
	}
}

// worker reads workspace names from the queue and refreshes each one.
func (r *Refresher) worker() {
	defer r.wg.Done()
	for {
		select {
		case <-r.done:
			return
		case wsName := <-r.queue:
			r.refreshWorkspace(wsName)
		}
	}
}
