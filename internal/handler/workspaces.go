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

	// Compute summary statistics.
	totalRepos := 0
	for _, ws := range workspaces {
		totalRepos += ws.RepoCount
	}

	data := model.WorkspacesPageData{
		Stats: model.WorkspacesStats{
			TotalWorkspaces: len(workspaces),
			TotalRepos:      totalRepos,
		},
		Workspaces: workspaces,
	}

	h.render(w, "workspaces", data)
}
