package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge-ui/internal/cache"
)

// Handler holds shared state for all HTTP handlers.
type Handler struct {
	BaseDir   string                        // $WORKSPACES path
	Templates map[string]*template.Template // keyed by page name
	Cache     *cache.Cache                  // background-refreshed git data
	HomeURL   string                        // always "/portfolios"
}

// New creates a Handler by parsing all templates from the given directory.
// templateDir is the absolute path to the templates/ directory.
func New(baseDir, templateDir string, c *cache.Cache) (*Handler, error) {
	h := &Handler{
		BaseDir:   baseDir,
		Templates: make(map[string]*template.Template),
		Cache:     c,
		HomeURL:   "/portfolios",
	}

	layoutPath := filepath.Join(templateDir, "layout.html")

	funcMap := template.FuncMap{
		"percent": func(done, total int) int {
			if total == 0 {
				return 0
			}
			return (done * 100) / total
		},
	}

	// Parse each page template together with the layout.
	// The layout defines "layout" and each page defines "content".
	pages := []string{"portfolios", "portfolio", "workspace", "forge"}
	for _, page := range pages {
		pagePath := filepath.Join(templateDir, page+".html")
		tmpl, err := template.New(page).Funcs(funcMap).ParseFiles(layoutPath, pagePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", page, err)
		}
		h.Templates[page] = tmpl
	}

	return h, nil
}

const themeCookieName = "theme"
const lightPaletteCookieName = "light-palette"

// isDarkMode reads the "theme" cookie and returns true when its value is "dark".
func isDarkMode(r *http.Request) bool {
	c, err := r.Cookie(themeCookieName)
	if err != nil {
		return false
	}
	return c.Value == "dark"
}

// lightPalette returns the active light palette identifier ("1"-"4").
// Returns "" when the user is in dark mode (no palette attribute needed).
func lightPalette(r *http.Request) string {
	if isDarkMode(r) {
		return ""
	}
	c, err := r.Cookie(lightPaletteCookieName)
	if err != nil {
		return "1"
	}
	switch c.Value {
	case "2", "3", "4":
		return c.Value
	default:
		return "1"
	}
}

// HandleSetLightPalette sets the light palette cookie to the value from the
// URL path and redirects back to the referring page.
func (h *Handler) HandleSetLightPalette(w http.ResponseWriter, r *http.Request) {
	n := r.PathValue("n")
	switch n {
	case "1", "2", "3", "4":
		// valid
	default:
		n = "1"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     lightPaletteCookieName,
		Value:    n,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	ref := r.Referer()
	if ref == "" {
		ref = h.HomeURL
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

// HandleToggleTheme toggles the theme cookie between "light" and "dark"
// and redirects back to the referring page.
func (h *Handler) HandleToggleTheme(w http.ResponseWriter, r *http.Request) {
	current := "light"
	if c, err := r.Cookie(themeCookieName); err == nil {
		current = c.Value
	}
	next := "dark"
	if current == "dark" {
		next = "light"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     themeCookieName,
		Value:    next,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	ref := r.Referer()
	if ref == "" {
		ref = h.HomeURL
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
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
