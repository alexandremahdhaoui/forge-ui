package httpdriver

import (
	"net/http"
)

// HandleForge handles GET /portfolios/{p}/workspaces/{w}/repos/{r}.
func (h *Handler) HandleForge(w http.ResponseWriter, r *http.Request) {
	pName := r.PathValue("p")
	wsName := r.PathValue("w")
	repoName := r.PathValue("r")
	if pName == "" || wsName == "" || repoName == "" {
		http.Error(w, "portfolio, workspace and repo name required", http.StatusBadRequest)
		return
	}

	data, err := h.ForgeService.GetForge(h.BaseDir, pName, wsName, repoName)
	if err != nil {
		http.Error(w, "failed to load forge data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data.DarkMode = isDarkMode(r)
	data.HomeURL = h.HomeURL
	data.LightPalette = lightPalette(r)

	h.render(w, "forge", data)
}
