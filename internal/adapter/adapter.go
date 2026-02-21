package adapter

import "github.com/alexandremahdhaoui/forge-ui/internal/types"

// DataSource provides page data for rendering.
type DataSource interface {
	ListPortfolios(sort string) (types.PortfoliosPageData, error)
	GetPortfolio(name, sort string) (types.PortfolioPageData, error)
	GetWorkspace(portfolio, workspace, sort string) (types.WorkspacePageData, error)
	GetForge(portfolio, workspace, repo string) (types.ForgePageData, error)
}
