package httpdriver

import (
	"net/http"
)

// HandlePortfolios handles GET /portfolios.
func (h *Handler) HandlePortfolios(w http.ResponseWriter, r *http.Request) {
	sortMode := r.URL.Query().Get("sort")
	if sortMode != "name" {
		sortMode = "time"
	}

	data, err := h.PortfolioService.ListPortfolios(h.BaseDir, sortMode)
	if err != nil {
		http.Error(w, "failed to list portfolios: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data.DarkMode = isDarkMode(r)
	data.HomeURL = h.HomeURL
	data.LightPalette = lightPalette(r)

	h.render(w, "portfolios", data)
}

// HandlePortfolio handles GET /portfolios/{name}.
func (h *Handler) HandlePortfolio(w http.ResponseWriter, r *http.Request) {
	pName := r.PathValue("name")
	if pName == "" {
		http.Error(w, "missing portfolio name", http.StatusBadRequest)
		return
	}

	sortMode := r.URL.Query().Get("sort")
	if sortMode != "name" {
		sortMode = "time"
	}

	data, err := h.PortfolioService.GetPortfolio(h.BaseDir, pName, sortMode)
	if err != nil {
		http.Error(w, "failed to get portfolio: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data.DarkMode = isDarkMode(r)
	data.HomeURL = h.HomeURL
	data.LightPalette = lightPalette(r)

	h.render(w, "portfolio", data)
}
