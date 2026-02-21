package httpdriver

import (
	"net/http"
)

// HandleWorkspace handles GET /portfolios/{p}/workspaces/{w}.
func (h *Handler) HandleWorkspace(w http.ResponseWriter, r *http.Request) {
	pName := r.PathValue("p")
	wsName := r.PathValue("w")
	if pName == "" || wsName == "" {
		http.Error(w, "portfolio and workspace name required", http.StatusBadRequest)
		return
	}

	sortMode := r.URL.Query().Get("sort")
	if sortMode != "name" {
		sortMode = "time"
	}

	data, err := h.WorkspaceService.GetWorkspace(h.BaseDir, pName, wsName, sortMode)
	if err != nil {
		http.Error(w, "workspace not found: "+err.Error(), http.StatusNotFound)
		return
	}

	data.DarkMode = isDarkMode(r)
	data.HomeURL = h.HomeURL

	h.render(w, "workspace", data)
}
