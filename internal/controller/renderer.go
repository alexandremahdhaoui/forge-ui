package controller

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

//go:embed templates/*.html
var templateFS embed.FS

// renderer implements PageRenderer.
type renderer struct {
	ds        adapter.DataSource
	templates *template.Template
}

// NewPageRenderer creates a PageRenderer backed by the given DataSource.
func NewPageRenderer(ds adapter.DataSource) PageRenderer {
	funcMap := template.FuncMap{
		"percent": func(done, total int) int {
			if total == 0 {
				return 0
			}
			return (done * 100) / total
		},
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		panic(fmt.Sprintf("controller: failed to parse templates: %v", err))
	}
	return &renderer{ds: ds, templates: tmpl}
}

// Render takes a raw route string (with optional query params and hash prefix)
// and returns rendered HTML.
func (r *renderer) Render(route string) (RenderResult, error) {
	input := parseInput(route)
	return r.executeRoute(input)
}

// input represents the parsed route input.
type input struct {
	Route string
	Sort  string // "name" or "time"; defaults to "time"
	Theme string // "light" or "dark"; defaults to "light"
}

// parseInput parses a raw string into an input.
func parseInput(raw string) input {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "#")

	if raw == "" || raw[0] != '/' {
		raw = "/" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return input{Route: "/portfolios", Sort: "time", Theme: "light"}
	}

	sort := u.Query().Get("sort")
	if sort == "" {
		sort = "time"
	}

	theme := u.Query().Get("theme")
	if theme == "" {
		theme = "light"
	}

	return input{
		Route: u.Path,
		Sort:  sort,
		Theme: theme,
	}
}

func (r *renderer) executeRoute(in input) (RenderResult, error) {
	parts := splitRoute(in.Route)

	switch {
	case len(parts) == 1 && parts[0] == "portfolios":
		return r.renderPortfolios(in)
	case len(parts) == 2 && parts[0] == "portfolios":
		return r.renderPortfolio(parts[1], in)
	case len(parts) == 4 && parts[0] == "portfolios" && parts[2] == "workspaces":
		return r.renderWorkspace(parts[1], parts[3], in)
	case len(parts) == 6 && parts[0] == "portfolios" && parts[2] == "workspaces" && parts[4] == "repos":
		return r.renderForge(parts[1], parts[3], parts[5], in)
	default:
		return r.renderPortfolios(in)
	}
}

func splitRoute(route string) []string {
	route = strings.Trim(route, "/")
	if route == "" {
		return []string{"portfolios"}
	}
	return strings.Split(route, "/")
}

func (r *renderer) renderPortfolios(in input) (RenderResult, error) {
	data, err := r.ds.ListPortfolios(in.Sort)
	if err != nil {
		return RenderResult{}, fmt.Errorf("list portfolios: %w", err)
	}
	data.DarkMode = in.Theme == "dark"

	var nav types.SideNavData
	for _, p := range data.Portfolios {
		nav.Items = append(nav.Items, types.SideNavItem{
			Name: p.Name,
			Link: "#/portfolios/" + p.Name,
		})
	}
	sideNav, err := r.renderSideNav(nav)
	if err != nil {
		return RenderResult{}, err
	}

	content, err := r.renderTemplate("portfolios", data)
	if err != nil {
		return RenderResult{}, err
	}
	return RenderResult{SideNav: sideNav, Content: content}, nil
}

func (r *renderer) renderPortfolio(name string, in input) (RenderResult, error) {
	data, err := r.ds.GetPortfolio(name, in.Sort)
	if err != nil {
		return RenderResult{}, fmt.Errorf("get portfolio: %w", err)
	}
	if data.Name == "" {
		return r.renderPortfolios(in)
	}
	data.DarkMode = in.Theme == "dark"

	nav := types.SideNavData{
		Header: types.SideNavHeader{
			Segments: []types.SideNavBreadcrumb{
				{Text: name, Link: "#/portfolios/" + name},
			},
		},
	}
	for _, ws := range data.Workspaces {
		nav.Items = append(nav.Items, types.SideNavItem{
			Name: ws.Name,
			Link: "#/portfolios/" + name + "/workspaces/" + ws.Name,
		})
	}
	sideNav, err := r.renderSideNav(nav)
	if err != nil {
		return RenderResult{}, err
	}

	content, err := r.renderTemplate("portfolio", data)
	if err != nil {
		return RenderResult{}, err
	}
	return RenderResult{SideNav: sideNav, Content: content}, nil
}

func (r *renderer) renderWorkspace(portfolio, workspace string, in input) (RenderResult, error) {
	data, err := r.ds.GetWorkspace(portfolio, workspace, in.Sort)
	if err != nil {
		return RenderResult{}, fmt.Errorf("get workspace: %w", err)
	}
	if data.Name == "" {
		return r.renderPortfolio(portfolio, in)
	}
	data.DarkMode = in.Theme == "dark"

	nav := types.SideNavData{
		Header: types.SideNavHeader{
			Segments: []types.SideNavBreadcrumb{
				{Text: portfolio, Link: "#/portfolios/" + portfolio},
				{Text: workspace, Link: "#/portfolios/" + portfolio + "/workspaces/" + workspace},
			},
		},
	}
	for _, repo := range data.Repos {
		nav.Items = append(nav.Items, types.SideNavItem{
			Name: repo.Name,
			Link: repo.RepoLink,
		})
	}
	sideNav, err := r.renderSideNav(nav)
	if err != nil {
		return RenderResult{}, err
	}

	content, err := r.renderTemplate("workspace", data)
	if err != nil {
		return RenderResult{}, err
	}
	return RenderResult{SideNav: sideNav, Content: content}, nil
}

func (r *renderer) renderForge(portfolio, workspace, repo string, in input) (RenderResult, error) {
	data, err := r.ds.GetForge(portfolio, workspace, repo)
	if err != nil {
		return RenderResult{}, fmt.Errorf("get forge: %w", err)
	}
	if data.RepoName == "" {
		return r.renderWorkspace(portfolio, workspace, in)
	}
	data.DarkMode = in.Theme == "dark"

	nav := types.SideNavData{
		Header: types.SideNavHeader{
			Segments: []types.SideNavBreadcrumb{
				{Text: portfolio, Link: "#/portfolios/" + portfolio},
				{Text: workspace, Link: "#/portfolios/" + portfolio + "/workspaces/" + workspace},
			},
		},
		Items: data.SiblingRepos,
	}
	sideNav, err := r.renderSideNav(nav)
	if err != nil {
		return RenderResult{}, err
	}

	content, err := r.renderTemplate("forge", data)
	if err != nil {
		return RenderResult{}, err
	}
	return RenderResult{SideNav: sideNav, Content: content}, nil
}

func (r *renderer) renderSideNav(nav types.SideNavData) (string, error) {
	if len(nav.Items) == 0 {
		return "", nil
	}
	return r.renderTemplate("sidenav", nav)
}

func (r *renderer) renderTemplate(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := r.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.String(), nil
}
