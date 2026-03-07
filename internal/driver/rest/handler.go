// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"context"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// APIHandler implements the generated StrictServerInterface, delegating to
// controller services and returning typed JSON responses.
type APIHandler struct {
	BaseDir              string
	PortfolioService     controller.PortfolioService
	WorkspaceService     controller.WorkspaceService
	ForgeService         controller.ForgeService
	WorkspaceMgmtService controller.WorkspaceMgmtService
	CUService            controller.CUService
}

// NewAPIHandler creates an APIHandler wired to the given controller services.
func NewAPIHandler(
	baseDir string,
	ps controller.PortfolioService,
	ws controller.WorkspaceService,
	fs controller.ForgeService,
	wms controller.WorkspaceMgmtService,
	cus controller.CUService,
) *APIHandler {
	return &APIHandler{
		BaseDir:              baseDir,
		PortfolioService:     ps,
		WorkspaceService:     ws,
		ForgeService:         fs,
		WorkspaceMgmtService: wms,
		CUService:            cus,
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

// nsOrDefault returns the namespace string, defaulting to "default" when nil.
func nsOrDefault(ns *string) string {
	if ns == nil || *ns == "" {
		return "default"
	}
	return *ns
}

// ListManagedWorkspaces handles GET /api/v1/managed-workspaces.
func (h *APIHandler) ListManagedWorkspaces(ctx context.Context, request ListManagedWorkspacesRequestObject) (ListManagedWorkspacesResponseObject, error) {
	ns := nsOrDefault(request.Params.Namespace)

	data, err := h.WorkspaceMgmtService.ListManagedWorkspaces(ns)
	if err != nil {
		return ListManagedWorkspaces500JSONResponse{Error: err.Error()}, nil
	}

	return ListManagedWorkspaces200JSONResponse(data), nil
}

// CreateManagedWorkspace handles POST /api/v1/managed-workspaces.
func (h *APIHandler) CreateManagedWorkspace(ctx context.Context, request CreateManagedWorkspaceRequestObject) (CreateManagedWorkspaceResponseObject, error) {
	ns := nsOrDefault(request.Params.Namespace)

	if request.Body == nil {
		return CreateManagedWorkspace400JSONResponse{Error: "request body required"}, nil
	}

	data, err := h.WorkspaceMgmtService.CreateManagedWorkspace(ns, *request.Body)
	if err != nil {
		return CreateManagedWorkspace500JSONResponse{Error: err.Error()}, nil
	}

	return CreateManagedWorkspace201JSONResponse(*data), nil
}

// GetManagedWorkspace handles GET /api/v1/managed-workspaces/{name}.
func (h *APIHandler) GetManagedWorkspace(ctx context.Context, request GetManagedWorkspaceRequestObject) (GetManagedWorkspaceResponseObject, error) {
	ns := nsOrDefault(request.Params.Namespace)

	data, err := h.WorkspaceMgmtService.GetManagedWorkspace(ns, request.Name)
	if err != nil {
		return GetManagedWorkspace500JSONResponse{Error: err.Error()}, nil
	}

	if data == nil {
		return GetManagedWorkspace404JSONResponse{Error: "managed workspace not found"}, nil
	}

	return GetManagedWorkspace200JSONResponse(*data), nil
}

// DeleteManagedWorkspace handles DELETE /api/v1/managed-workspaces/{name}.
func (h *APIHandler) DeleteManagedWorkspace(ctx context.Context, request DeleteManagedWorkspaceRequestObject) (DeleteManagedWorkspaceResponseObject, error) {
	ns := nsOrDefault(request.Params.Namespace)

	err := h.WorkspaceMgmtService.DeleteManagedWorkspace(ns, request.Name)
	if err != nil {
		return DeleteManagedWorkspace500JSONResponse{Error: err.Error()}, nil
	}

	return DeleteManagedWorkspace204Response{}, nil
}

// SuspendManagedWorkspace handles PUT /api/v1/managed-workspaces/{name}/suspend.
func (h *APIHandler) SuspendManagedWorkspace(ctx context.Context, request SuspendManagedWorkspaceRequestObject) (SuspendManagedWorkspaceResponseObject, error) {
	ns := nsOrDefault(request.Params.Namespace)

	data, err := h.WorkspaceMgmtService.SuspendManagedWorkspace(ns, request.Name)
	if err != nil {
		return SuspendManagedWorkspace500JSONResponse{Error: err.Error()}, nil
	}

	if data == nil {
		return SuspendManagedWorkspace404JSONResponse{Error: "managed workspace not found"}, nil
	}

	return SuspendManagedWorkspace200JSONResponse(*data), nil
}

// ResumeManagedWorkspace handles PUT /api/v1/managed-workspaces/{name}/resume.
func (h *APIHandler) ResumeManagedWorkspace(ctx context.Context, request ResumeManagedWorkspaceRequestObject) (ResumeManagedWorkspaceResponseObject, error) {
	ns := nsOrDefault(request.Params.Namespace)

	data, err := h.WorkspaceMgmtService.ResumeManagedWorkspace(ns, request.Name)
	if err != nil {
		return ResumeManagedWorkspace500JSONResponse{Error: err.Error()}, nil
	}

	if data == nil {
		return ResumeManagedWorkspace404JSONResponse{Error: "managed workspace not found"}, nil
	}

	return ResumeManagedWorkspace200JSONResponse(*data), nil
}

// GetWorkspaceCU handles GET /api/v1/portfolios/{portfolio}/workspaces/{workspace}/cu.
func (h *APIHandler) GetWorkspaceCU(ctx context.Context, request GetWorkspaceCURequestObject) (GetWorkspaceCUResponseObject, error) {
	wsBaseDir := h.BaseDir
	if request.Portfolio != "default" {
		wsBaseDir = filepath.Join(h.BaseDir, request.Portfolio)
	}
	wsPath := filepath.Join(wsBaseDir, request.Workspace)

	data, err := h.CUService.GetCompoState(wsPath)
	if err != nil {
		return GetWorkspaceCU500JSONResponse{Error: err.Error()}, nil
	}

	return GetWorkspaceCU200JSONResponse(data), nil
}
