package model

import "time"

// --- Page 1: /workspaces ---

type WorkspacesPageData struct {
	Workspaces []WorkspaceSummary
}

type WorkspaceSummary struct {
	Name      string // directory name under $WORKSPACES
	Path      string // absolute path
	RepoCount int    // count of subdirs with .git/
}

// --- Page 2: /workspaces/{name} ---

type WorkspacePageData struct {
	Name  string
	Path  string
	Repos []RepoSummary
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
	ForgeLink   string        // URL path: /workspaces/{ws}/repos/{repo}/forge
}

type StatusEntry struct {
	Code     string // "M", "A", "D", "??"
	FilePath string // file path from status
}

type LogEntry struct {
	Hash    string // short commit hash
	Message string // commit subject line
}

// --- Page 3: /workspaces/{ws}/repos/{repo}/forge ---

type ForgePageData struct {
	WorkspaceName string
	RepoName      string
	Spec          ForgeSpec
	Artifacts     []Artifact
	TestReports   []TestReport
	TestEnvs      []TestEnv
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
