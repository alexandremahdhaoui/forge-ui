package handler

import (
	"net/http"
	"path/filepath"

	forgepkg "github.com/alexandremahdhaoui/forge-ui/internal/forge"
)

// HandleForge handles GET /workspaces/{ws}/repos/{repo}/forge.
func (h *Handler) HandleForge(w http.ResponseWriter, r *http.Request) {
	ws := r.PathValue("ws")
	repo := r.PathValue("repo")
	if ws == "" || repo == "" {
		http.Error(w, "workspace and repo name required", http.StatusBadRequest)
		return
	}

	repoPath := filepath.Join(h.BaseDir, ws, repo)

	data, err := forgepkg.Load(repoPath)
	if err != nil {
		http.Error(w, "failed to load forge data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data.WorkspaceName = ws
	data.RepoName = repo

	h.render(w, "forge", data)
}
