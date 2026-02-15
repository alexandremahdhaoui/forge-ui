package handler

import "net/http"

// HandleRedirect handles GET / by redirecting to /workspaces.
func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/workspaces", http.StatusFound)
}
