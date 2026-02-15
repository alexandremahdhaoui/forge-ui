package handler

import (
	"net/http"

	gitpkg "github.com/alexandremahdhaoui/forge-ui/internal/git"
	"github.com/alexandremahdhaoui/forge-ui/internal/workspace"
)

// HandleWorkspace handles GET /workspaces/{name}.
func (h *Handler) HandleWorkspace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "workspace name required", http.StatusBadRequest)
		return
	}

	data, err := workspace.Get(h.BaseDir, name)
	if err != nil {
		http.Error(w, "workspace not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Enrich each repo with git information.
	for i, repo := range data.Repos {
		gitInfo, err := gitpkg.RepoInfo(repo.Path)
		if err != nil {
			// On git error, leave fields at zero values
			continue
		}
		data.Repos[i].Branch = gitInfo.Branch
		data.Repos[i].IsDirty = gitInfo.IsDirty
		data.Repos[i].StatusFiles = gitInfo.StatusFiles
		data.Repos[i].DiffStat = gitInfo.DiffStat
		data.Repos[i].RecentLogs = gitInfo.RecentLogs
	}

	h.render(w, "workspace", data)
}
