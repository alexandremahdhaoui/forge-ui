package handler

import (
	"net/http"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
	"github.com/alexandremahdhaoui/forge-ui/internal/workspace"
)

// HandleWorkspaces handles GET /workspaces.
func (h *Handler) HandleWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := workspace.List(h.BaseDir)
	if err != nil {
		http.Error(w, "failed to list workspaces: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := model.WorkspacesPageData{
		Workspaces: workspaces,
	}

	h.render(w, "workspaces", data)
}
