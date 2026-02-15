package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

// Handler holds shared state for all HTTP handlers.
type Handler struct {
	BaseDir   string                        // $WORKSPACES path
	Templates map[string]*template.Template // keyed by page name
}

// New creates a Handler by parsing all templates from the given directory.
// templateDir is the absolute path to the templates/ directory.
func New(baseDir, templateDir string) (*Handler, error) {
	h := &Handler{
		BaseDir:   baseDir,
		Templates: make(map[string]*template.Template),
	}

	layoutPath := filepath.Join(templateDir, "layout.html")

	// Parse each page template together with the layout.
	// The layout defines "layout" and each page defines "content".
	pages := []string{"workspaces", "workspace", "forge"}
	for _, page := range pages {
		pagePath := filepath.Join(templateDir, page+".html")
		tmpl, err := template.ParseFiles(layoutPath, pagePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", page, err)
		}
		h.Templates[page] = tmpl
	}

	return h, nil
}

// render executes the named template with data, writing to the response.
// It executes the "layout" template which in turn calls "content".
func (h *Handler) render(w http.ResponseWriter, templateName string, data any) {
	tmpl, ok := h.Templates[templateName]
	if !ok {
		http.Error(w, "template not found: "+templateName, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
