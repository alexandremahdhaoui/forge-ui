package rest

import (
	"context"

	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// APIHandler implements the generated StrictServerInterface, delegating to
// controller services and returning typed JSON responses.
type APIHandler struct {
	BaseDir          string
	PortfolioService controller.PortfolioService
	WorkspaceService controller.WorkspaceService
	ForgeService     controller.ForgeService
}

// NewAPIHandler creates an APIHandler wired to the given controller services.
func NewAPIHandler(baseDir string, ps controller.PortfolioService, ws controller.WorkspaceService, fs controller.ForgeService) *APIHandler {
	return &APIHandler{
		BaseDir:          baseDir,
		PortfolioService: ps,
		WorkspaceService: ws,
		ForgeService:     fs,
	}
}

// Compile-time check that APIHandler satisfies StrictServerInterface.
var _ StrictServerInterface = (*APIHandler)(nil)

// sortOrDefault returns the sort string from a typed sort param pointer,
// defaulting to "time" when nil or unrecognized.
func sortOrDefault[T ~string](s *T) string {
	if s == nil {
		return "time"
	}
	v := string(*s)
	if v == "name" || v == "time" {
		return v
	}
	return "time"
}

// ListPortfolios handles GET /api/v1/portfolios.
func (h *APIHandler) ListPortfolios(ctx context.Context, request ListPortfoliosRequestObject) (ListPortfoliosResponseObject, error) {
	sortMode := sortOrDefault(request.Params.Sort)

	data, err := h.PortfolioService.ListPortfolios(h.BaseDir, sortMode)
	if err != nil {
		return ListPortfolios500JSONResponse{Error: err.Error()}, nil
	}

	return ListPortfolios200JSONResponse(data), nil
}

// GetPortfolio handles GET /api/v1/portfolios/{name}.
func (h *APIHandler) GetPortfolio(ctx context.Context, request GetPortfolioRequestObject) (GetPortfolioResponseObject, error) {
	sortMode := sortOrDefault(request.Params.Sort)

	data, err := h.PortfolioService.GetPortfolio(h.BaseDir, request.Name, sortMode)
	if err != nil {
		return GetPortfolio500JSONResponse{Error: err.Error()}, nil
	}

	if data.Name == "" {
		return GetPortfolio404JSONResponse{Error: "portfolio not found"}, nil
	}

	return GetPortfolio200JSONResponse(data), nil
}

// GetWorkspace handles GET /api/v1/portfolios/{portfolio}/workspaces/{workspace}.
func (h *APIHandler) GetWorkspace(ctx context.Context, request GetWorkspaceRequestObject) (GetWorkspaceResponseObject, error) {
	sortMode := sortOrDefault(request.Params.Sort)

	data, err := h.WorkspaceService.GetWorkspace(h.BaseDir, request.Portfolio, request.Workspace, sortMode)
	if err != nil {
		return GetWorkspace500JSONResponse{Error: err.Error()}, nil
	}

	if data.Name == "" {
		return GetWorkspace404JSONResponse{Error: "workspace not found"}, nil
	}

	return GetWorkspace200JSONResponse(data), nil
}

// GetRepo handles GET /api/v1/portfolios/{portfolio}/workspaces/{workspace}/repos/{repo}.
func (h *APIHandler) GetRepo(ctx context.Context, request GetRepoRequestObject) (GetRepoResponseObject, error) {
	data, err := h.ForgeService.GetForge(h.BaseDir, request.Portfolio, request.Workspace, request.Repo)
	if err != nil {
		return GetRepo500JSONResponse{Error: err.Error()}, nil
	}

	if data.RepoName == "" {
		return GetRepo404JSONResponse{Error: "repo not found"}, nil
	}

	// Populate sibling repos for side navigation.
	wsData, err := h.WorkspaceService.GetWorkspace(h.BaseDir, request.Portfolio, request.Workspace, "name")
	if err == nil {
		siblings := make([]types.SideNavItem, 0, len(wsData.Repos))
		for _, r := range wsData.Repos {
			siblings = append(siblings, types.SideNavItem{
				Name:     r.Name,
				Link:     r.RepoLink,
				IsActive: r.Name == request.Repo,
			})
		}
		data.SiblingRepos = siblings
	}

	return GetRepo200JSONResponse(data), nil
}
