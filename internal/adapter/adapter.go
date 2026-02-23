package adapter

import "github.com/alexandremahdhaoui/forge-ui/internal/types"

// DataSource provides page data for rendering.
type DataSource interface {
	ListPortfolios(sort string) (types.PortfoliosPageData, error)
	GetPortfolio(name, sort string) (types.PortfolioPageData, error)
	GetWorkspace(portfolio, workspace, sort string) (types.WorkspacePageData, error)
	GetForge(portfolio, workspace, repo string) (types.ForgePageData, error)
}

// MetaPlanLoader loads meta-plan YAML files from a workspace directory.
type MetaPlanLoader interface {
	LoadAll(wsPath string) ([]types.MetaPlan, error)
	Load(path string) (types.MetaPlan, error)
}

// RepoPlanLoader loads repo plan task files from a repository directory.
type RepoPlanLoader interface {
	LoadAll(repoPath string) ([]types.RepoPlan, error)
	LoadSummary(repoPath, repoName string) (types.RepoPlanSummary, error)
}

// WsConfigLoader loads workspace configuration from a workspace directory.
type WsConfigLoader interface {
	Load(wsPath string) (types.WsConfig, error)
}

// PortfolioConfigLoader loads portfolio configuration from a portfolio directory.
type PortfolioConfigLoader interface {
	Load(portfolioPath string) (types.PortfolioConfig, error)
}
