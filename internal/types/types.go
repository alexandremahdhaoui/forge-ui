package types

import "time"

type WorkspaceSummary struct {
	Name        string             `json:"name"`        // directory name under $WORKSPACES
	Path        string             `json:"path"`        // absolute path
	RepoCount   int                `json:"repoCount"`   // count of subdirs with .git/
	Repos       []RepoOverview     `json:"repos"`       // lightweight repo summaries
	AllStages   []string           `json:"allStages"`   // test stage names for heatmap columns
	RepoForge   []RepoForgeStats   `json:"repoForge"`   // per-repo forge test results
	Description    string             `json:"description"`
	MetaPlans      []MetaPlan         `json:"metaPlans"`
	Progress       WorkspaceProgress  `json:"progress"`
	LastActivity   time.Time          `json:"-"`
	DirtyRepoCount int               `json:"-"`
}

// --- Page 2: /workspaces/{name} ---

type WorkspacePageData struct {
	Name              string            `json:"name"`
	PortfolioName     string            `json:"portfolioName"`
	Path              string            `json:"path"`
	Repos             []RepoSummary     `json:"repos"`
	Stats             WorkspaceStats    `json:"stats"`
	AllStages         []string          `json:"allStages"`
	RepoForge         []RepoForgeStats  `json:"repoForge"`
	SortMode          string            `json:"sortMode"`
	DarkMode          bool              `json:"-"`
	HomeURL           string            `json:"-"`
	LightPalette      string            `json:"-"`
	TopRecentRepos    []RepoSummary     `json:"-"`
	Unattended        []RepoSummary     `json:"-"`
	Description       string            `json:"description"`
	RepoRoles         map[string]string `json:"repoRoles"`
	MetaPlans         []MetaPlan        `json:"metaPlans"`
	RepoPlanSummaries []RepoPlanSummary `json:"repoPlanSummaries"`
}

type RepoSummary struct {
	Name        string        `json:"name"`        // directory name
	Path        string        `json:"path"`        // absolute path
	Branch      string        `json:"branch"`      // current branch
	IsDirty     bool          `json:"isDirty"`     // len(StatusFiles) > 0
	StatusFiles []StatusEntry `json:"statusFiles"` // git status --porcelain
	DiffStat    string        `json:"diffStat"`    // git diff --stat raw output
	RecentLogs  []LogEntry    `json:"recentLogs"`  // git log --oneline -10
	HasForge    bool          `json:"hasForge"`    // forge.yaml exists in repo
	RepoLink   string         `json:"repoLink"`    // URL path: /workspaces/{ws}/repos/{repo}
	Ahead       int           `json:"ahead"`       // commits ahead of upstream
	Behind      int           `json:"behind"`      // commits behind upstream
	HasUpstream    bool       `json:"hasUpstream"`    // tracking branch exists
	LastCommitTime time.Time  `json:"lastCommitTime"`
}

// RepoOverview is a lightweight repo summary for the workspaces listing page.
type RepoOverview struct {
	Name          string    `json:"name"`
	WorkspaceName string    `json:"workspaceName"`
	Path          string    `json:"path"`
	Branch        string    `json:"branch"`
	IsDirty       bool      `json:"isDirty"`
	Ahead         int       `json:"ahead"`
	Behind        int       `json:"behind"`
	HasUpstream   bool      `json:"hasUpstream"`
	HasForge       bool     `json:"hasForge"`
	RepoLink       string   `json:"repoLink"`
	LastCommitTime time.Time `json:"lastCommitTime"`
}

type StatusEntry struct {
	Code     string `json:"code"`     // "M", "A", "D", "??"
	FilePath string `json:"filePath"` // file path from status
}

type LogEntry struct {
	Hash    string `json:"hash"`    // short commit hash
	Message string `json:"message"` // commit subject line
}

// --- Page 3: /workspaces/{ws}/repos/{repo} ---

type ForgePageData struct {
	WorkspaceName  string            `json:"workspaceName"`
	RepoName       string            `json:"repoName"`
	PortfolioName  string            `json:"portfolioName"`
	Spec           ForgeSpec         `json:"spec"`
	Artifacts      []Artifact        `json:"artifacts"`
	TestReports    []TestReport      `json:"testReports"`
	TestEnvs       []TestEnv         `json:"testEnvs"`
	Stats          ForgeStats        `json:"stats"`
	StageStatusMap map[string]string `json:"stageStatusMap"`
	DarkMode       bool              `json:"-"`
	HomeURL        string            `json:"-"`
	LightPalette   string            `json:"-"`
	RepoPlans          []RepoPlan        `json:"repoPlans"`
	SiblingRepos       []SideNavItem     `json:"siblingRepos"`
	LatestStageReports []TestReport      `json:"-"`
}

type ForgeSpec struct {
	Name  string      `json:"name"`
	Build []BuildSpec `json:"build"`
	Test  []TestSpec  `json:"test"`
}

type BuildSpec struct {
	Name   string `json:"name"`
	Src    string `json:"src"`
	Dest   string `json:"dest"`
	Engine string `json:"engine"`
}

type TestSpec struct {
	Name    string `json:"name"`
	Testenv string `json:"testenv"`
	Runner  string `json:"runner"`
}

type Artifact struct {
	Name         string               `json:"name"`
	Type         string               `json:"type"`         // "binary" or "container"
	Location     string               `json:"location"`
	Timestamp    string               `json:"timestamp"`    // RFC3339
	Version      string               `json:"version"`      // git SHA
	Dependencies []ArtifactDependency `json:"dependencies"`
}

type ArtifactDependency struct {
	Type            string `json:"type"`            // "file" or "externalPackage"
	FilePath        string `json:"filePath"`
	Timestamp       string `json:"timestamp"`
	ExternalPackage string `json:"externalPackage"`
	Semver          string `json:"semver"`
}

type TestReport struct {
	ID           string    `json:"id"`
	Stage        string    `json:"stage"`
	Status       string    `json:"status"` // "passed" or "failed"
	StartTime    time.Time `json:"startTime"`
	Duration     float64   `json:"duration"` // seconds
	Stats        TestStats `json:"stats"`
	Coverage     Coverage  `json:"coverage"`
	ErrorMessage string    `json:"errorMessage"`
}

type TestStats struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type Coverage struct {
	Enabled    bool    `json:"enabled"`
	Percentage float64 `json:"percentage"`
	FilePath   string  `json:"filePath"`
}

type TestEnv struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Status           string    `json:"status"` // "created","running","passed","failed"
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	ManagedResources []string  `json:"managedResources"`
}

// --- Statistics types ---

type WorkspacesStats struct {
	TotalWorkspaces int               `json:"totalWorkspaces"`
	TotalRepos      int               `json:"totalRepos"`
	DirtyRepos      int               `json:"dirtyRepos"`
	TotalTests      int               `json:"totalTests"`
	Passed          int               `json:"passed"`
	Failed          int               `json:"failed"`
	Portfolio       PortfolioProgress `json:"portfolio"`
}

// --- Portfolio types ---

type PortfolioSummary struct {
	Name        string             `json:"name"`        // directory name under baseDir (or "default")
	Path        string             `json:"path"`        // absolute path (baseDir for "default" portfolio)
	IsDefault   bool               `json:"isDefault"`   // true for the catch-all portfolio
	Workspaces  []WorkspaceSummary `json:"workspaces"`  // workspaces within this portfolio
	Stats        WorkspacesStats    `json:"stats"`       // reuse existing type for aggregate stats
	Description  string             `json:"description"`
	LastActivity time.Time          `json:"-"`
}

type PortfoliosPageData struct {
	Portfolios   []PortfolioSummary `json:"portfolios"`
	Stats        PortfoliosStats    `json:"stats"`
	SortMode     string             `json:"sortMode"`
	DarkMode              bool               `json:"-"`
	HomeURL               string             `json:"-"` // always "/portfolios"
	LightPalette          string             `json:"-"`
	TopRecentPortfolios   []PortfolioSummary `json:"-"`
	Unattended            []PortfolioSummary `json:"-"`
}

type PortfoliosStats struct {
	TotalPortfolios int               `json:"totalPortfolios"`
	TotalWorkspaces int               `json:"totalWorkspaces"`
	TotalRepos      int               `json:"totalRepos"`
	DirtyRepos      int               `json:"dirtyRepos"`
	TotalTests      int               `json:"totalTests"`
	Passed          int               `json:"passed"`
	Failed          int               `json:"failed"`
	Portfolio       PortfolioProgress `json:"portfolio"`
}

type PortfolioPageData struct {
	Name         string             `json:"name"`
	Path         string             `json:"path"`
	IsDefault    bool               `json:"isDefault"`
	Workspaces   []WorkspaceSummary `json:"workspaces"`
	Stats        WorkspacesStats    `json:"stats"` // reuse existing type
	Description  string             `json:"description"`
	SortMode     string             `json:"sortMode"`
	DarkMode              bool               `json:"-"`
	HomeURL               string             `json:"-"` // always "/portfolios"
	LightPalette          string             `json:"-"`
	TopRecentWorkspaces   []WorkspaceSummary `json:"-"`
	Unattended            []WorkspaceSummary `json:"-"`
}

type WorkspaceStats struct {
	TotalRepos int `json:"totalRepos"`
	ForgeRepos int `json:"forgeRepos"`
	TotalTests int `json:"totalTests"`
	Passed     int `json:"passed"`
	Failed     int `json:"failed"`
	Skipped    int `json:"skipped"`
}

type RepoForgeStats struct {
	RepoName     string            `json:"repoName"`
	RepoLink    string             `json:"repoLink"`
	StageResults map[string]string `json:"stageResults"`
}

type ForgeStats struct {
	TotalTests  int     `json:"totalTests"`
	Passed      int     `json:"passed"`
	Failed      int     `json:"failed"`
	Skipped     int     `json:"skipped"`
	AvgCoverage float64 `json:"avgCoverage"`
	HasCoverage bool    `json:"hasCoverage"`
	StageCount  int     `json:"stageCount"`
}

// --- Workspace orchestration types ---

type WsConfig struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Repos       []WsRepoEntry `json:"repos"`
	MetaPlans   []string      `json:"metaPlans"`
}

type WsRepoEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PortfolioConfig struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	TrackerPaths []string `json:"trackerPaths"`
}

type MetaPlan struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Status      string           `json:"status"` // "pending", "in_progress", "completed"
	Stages      []MetaPlanStage  `json:"stages"`
	Checkpoints []MetaCheckpoint `json:"checkpoints"`
}

type MetaPlanStage struct {
	Name   string      `json:"name"`
	Status string      `json:"status"` // "pending", "in_progress", "completed"
	Repos  []StageRepo `json:"repos"`
}

type StageRepo struct {
	Name       string `json:"name"`
	Plan       string `json:"plan"`
	TasksTotal int    `json:"tasksTotal"`
	TasksDone  int    `json:"tasksDone"`
}

type MetaCheckpoint struct {
	Name      string `json:"name"`
	Stage     string `json:"stage"`
	Condition string `json:"condition"`
	Met       bool   `json:"met"`
}

type RepoPlan struct {
	Name       string `json:"name"`
	TasksTotal int    `json:"tasksTotal"`
	TasksDone  int    `json:"tasksDone"`
}

type RepoPlanSummary struct {
	RepoName   string     `json:"repoName"`
	Plans      []RepoPlan `json:"plans"`
	TasksTotal int        `json:"tasksTotal"`
	TasksDone  int        `json:"tasksDone"`
}

type PortfolioProgress struct {
	TotalMetaPlans     int `json:"totalMetaPlans"`
	ActiveMetaPlans    int `json:"activeMetaPlans"`
	CompletedMetaPlans int `json:"completedMetaPlans"`
	TasksTotal         int `json:"tasksTotal"`
	TasksDone          int `json:"tasksDone"`
	PercentDone        int `json:"percentDone"`
}

type WorkspaceProgress struct {
	MetaPlanCount int `json:"metaPlanCount"`
	TasksTotal    int `json:"tasksTotal"`
	TasksDone     int `json:"tasksDone"`
	PercentDone   int `json:"percentDone"`
}

// CacheWorkspaceData holds cached git data for all repos in a single workspace.
type CacheWorkspaceData struct {
	Summaries map[string]RepoSummary  `json:"summaries"` // keyed by repo name
	Overviews map[string]RepoOverview `json:"overviews"`  // keyed by repo name
	UpdatedAt time.Time               `json:"updatedAt"`
}

// SideNavData holds data for rendering the side navigation bar.
type SideNavData struct {
	Header SideNavHeader `json:"header"`
	Items  []SideNavItem `json:"items"`
}

// SideNavHeader holds breadcrumb segments for the side nav header.
type SideNavHeader struct {
	Segments []SideNavBreadcrumb `json:"segments"`
}

// SideNavBreadcrumb represents a single breadcrumb segment.
type SideNavBreadcrumb struct {
	Text string `json:"text"`
	Link string `json:"link"`
}

// SideNavItem represents a single navigation item in the side nav.
type SideNavItem struct {
	Name     string `json:"name"`
	Link     string `json:"link"`
	IsActive bool   `json:"isActive"`
	Badge    string `json:"badge,omitempty"`
}
