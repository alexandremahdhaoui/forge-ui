package types

import "time"

type WorkspaceSummary struct {
	Name      string         // directory name under $WORKSPACES
	Path      string         // absolute path
	RepoCount int            // count of subdirs with .git/
	Repos     []RepoOverview // lightweight repo summaries
	AllStages []string       // test stage names for heatmap columns
	RepoForge []RepoForgeStats // per-repo forge test results
}

// --- Page 2: /workspaces/{name} ---

type WorkspacePageData struct {
	Name          string
	PortfolioName string
	Path          string
	Repos         []RepoSummary
	Stats         WorkspaceStats
	AllStages     []string
	RepoForge     []RepoForgeStats
	SortMode      string
	DarkMode      bool
	HomeURL       string
}

type RepoSummary struct {
	Name        string        // directory name
	Path        string        // absolute path
	Branch      string        // current branch
	IsDirty     bool          // len(StatusFiles) > 0
	StatusFiles []StatusEntry // git status --porcelain
	DiffStat    string        // git diff --stat raw output
	RecentLogs  []LogEntry    // git log --oneline -10
	HasForge    bool          // forge.yaml exists in repo
	RepoLink   string        // URL path: /workspaces/{ws}/repos/{repo}
	Ahead       int           // commits ahead of upstream
	Behind      int           // commits behind upstream
	HasUpstream    bool          // tracking branch exists
	LastCommitTime time.Time
}

// RepoOverview is a lightweight repo summary for the workspaces listing page.
type RepoOverview struct {
	Name          string
	WorkspaceName string
	Path          string
	Branch        string
	IsDirty       bool
	Ahead         int
	Behind        int
	HasUpstream   bool
	HasForge       bool
	RepoLink       string
	LastCommitTime time.Time
}

type StatusEntry struct {
	Code     string // "M", "A", "D", "??"
	FilePath string // file path from status
}

type LogEntry struct {
	Hash    string // short commit hash
	Message string // commit subject line
}

// --- Page 3: /workspaces/{ws}/repos/{repo} ---

type ForgePageData struct {
	WorkspaceName  string
	RepoName       string
	PortfolioName  string
	Spec           ForgeSpec
	Artifacts      []Artifact
	TestReports    []TestReport
	TestEnvs       []TestEnv
	Stats          ForgeStats
	StageStatusMap map[string]string
	DarkMode       bool
	HomeURL        string
}

type ForgeSpec struct {
	Name  string
	Build []BuildSpec
	Test  []TestSpec
}

type BuildSpec struct {
	Name   string
	Src    string
	Dest   string
	Engine string
}

type TestSpec struct {
	Name    string
	Testenv string
	Runner  string
}

type Artifact struct {
	Name         string
	Type         string // "binary" or "container"
	Location     string
	Timestamp    string // RFC3339
	Version      string // git SHA
	Dependencies []ArtifactDependency
}

type ArtifactDependency struct {
	Type            string // "file" or "externalPackage"
	FilePath        string
	Timestamp       string
	ExternalPackage string
	Semver          string
}

type TestReport struct {
	ID           string
	Stage        string
	Status       string // "passed" or "failed"
	StartTime    time.Time
	Duration     float64 // seconds
	Stats        TestStats
	Coverage     Coverage
	ErrorMessage string
}

type TestStats struct {
	Total   int
	Passed  int
	Failed  int
	Skipped int
}

type Coverage struct {
	Enabled    bool
	Percentage float64
	FilePath   string
}

type TestEnv struct {
	ID               string
	Name             string
	Status           string // "created","running","passed","failed"
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ManagedResources []string
}

// --- Statistics types ---

type WorkspacesStats struct {
	TotalWorkspaces int
	TotalRepos      int
	DirtyRepos      int
	TotalTests      int
	Passed          int
	Failed          int
}

// --- Portfolio types ---

type PortfolioSummary struct {
	Name       string             // directory name under baseDir (or "default")
	Path       string             // absolute path (baseDir for "default" portfolio)
	IsDefault  bool               // true for the catch-all portfolio
	Workspaces []WorkspaceSummary // workspaces within this portfolio
	Stats      WorkspacesStats    // reuse existing type for aggregate stats
}

type PortfoliosPageData struct {
	Portfolios []PortfolioSummary
	Stats      PortfoliosStats
	SortMode   string
	DarkMode   bool
	HomeURL    string // always "/portfolios"
}

type PortfoliosStats struct {
	TotalPortfolios int
	TotalWorkspaces int
	TotalRepos      int
	DirtyRepos      int
	TotalTests      int
	Passed          int
	Failed          int
}

type PortfolioPageData struct {
	Name       string
	Path       string
	IsDefault  bool
	Workspaces []WorkspaceSummary
	Stats      WorkspacesStats // reuse existing type
	SortMode   string
	DarkMode   bool
	HomeURL    string // always "/portfolios"
}

type WorkspaceStats struct {
	TotalRepos int
	ForgeRepos int
	TotalTests int
	Passed     int
	Failed     int
	Skipped    int
}

type RepoForgeStats struct {
	RepoName     string
	RepoLink    string
	StageResults map[string]string
}

type ForgeStats struct {
	TotalTests  int
	Passed      int
	Failed      int
	Skipped     int
	AvgCoverage float64
	HasCoverage bool
	StageCount  int
}

// CacheWorkspaceData holds cached git data for all repos in a single workspace.
type CacheWorkspaceData struct {
	Summaries map[string]RepoSummary  // keyed by repo name
	Overviews map[string]RepoOverview // keyed by repo name
	UpdatedAt time.Time
}
