package httpdriver

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
)

// Handler holds shared state for all HTTP handlers.
type Handler struct {
	BaseDir          string
	Templates        map[string]*template.Template
	HomeURL          string
	PortfolioService controller.PortfolioService
	WorkspaceService controller.WorkspaceService
	ForgeService     controller.ForgeService
}

// New creates a Handler by parsing all templates from the given directory.
func New(baseDir, templateDir string, ps controller.PortfolioService, ws controller.WorkspaceService, fs controller.ForgeService) (*Handler, error) {
	h := &Handler{
		BaseDir:          baseDir,
		Templates:        make(map[string]*template.Template),
		HomeURL:          "/portfolios",
		PortfolioService: ps,
		WorkspaceService: ws,
		ForgeService:     fs,
	}

	layoutPath := filepath.Join(templateDir, "layout.html")

	pages := []string{"portfolios", "portfolio", "workspace", "forge"}
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

const themeCookieName = "theme"

// isDarkMode reads the "theme" cookie and returns true when its value is "dark".
func isDarkMode(r *http.Request) bool {
	c, err := r.Cookie(themeCookieName)
	if err != nil {
		return false
	}
	return c.Value == "dark"
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
