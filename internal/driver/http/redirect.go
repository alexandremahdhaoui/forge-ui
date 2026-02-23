package httpdriver

import "net/http"

// HandleRedirect handles GET / by redirecting to /portfolios.
func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.HomeURL, http.StatusFound)
}
