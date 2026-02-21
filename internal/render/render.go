package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").ParseFS(templateFS, "templates/*.html")
	if err != nil {
		panic(fmt.Sprintf("render: failed to parse templates: %v", err))
	}
}

// Execute takes a raw input string (route + query params) and returns rendered HTML.
func Execute(raw string) (string, error) {
	input := ParseInput(raw)
	return executeRoute(input)
}

func executeRoute(input Input) (string, error) {
	// Parse route segments: /portfolios/{name}/workspaces/{ws}/repos/{repo}
	parts := splitRoute(input.Route)

	switch {
	case len(parts) == 1 && parts[0] == "portfolios":
		return renderPortfolios(input)
	case len(parts) == 2 && parts[0] == "portfolios":
		return renderPortfolio(parts[1], input)
	case len(parts) == 4 && parts[0] == "portfolios" && parts[2] == "workspaces":
		return renderWorkspace(parts[1], parts[3], input)
	case len(parts) == 6 && parts[0] == "portfolios" && parts[2] == "workspaces" && parts[4] == "repos":
		return renderForge(parts[1], parts[3], parts[5], input)
	default:
		// Default to portfolios.
		return renderPortfolios(input)
	}
}

func splitRoute(route string) []string {
	route = strings.Trim(route, "/")
	if route == "" {
		return []string{"portfolios"}
	}
	return strings.Split(route, "/")
}

func renderPortfolios(input Input) (string, error) {
	portfolios := make([]model.PortfolioSummary, 0, len(DemoData.Portfolios))
	var stats model.PortfoliosStats

	for _, p := range DemoData.Portfolios {
		portfolios = append(portfolios, p)
		stats.TotalWorkspaces += p.Stats.TotalWorkspaces
		stats.TotalRepos += p.Stats.TotalRepos
		stats.DirtyRepos += p.Stats.DirtyRepos
		stats.Passed += p.Stats.Passed
		stats.Failed += p.Stats.Failed
	}
	stats.TotalPortfolios = len(portfolios)

	data := model.PortfoliosPageData{
		Portfolios: portfolios,
		Stats:      stats,
		SortMode:   input.Sort,
		DarkMode:   input.Theme == "dark",
		HomeURL:    "#/portfolios",
	}

	return renderTemplate("portfolios", data)
}

func renderPortfolio(name string, input Input) (string, error) {
	p, ok := DemoData.Portfolios[name]
	if !ok {
		return renderPortfolios(input)
	}

	data := model.PortfolioPageData{
		Name:       p.Name,
		Path:       p.Path,
		IsDefault:  p.IsDefault,
		Workspaces: p.Workspaces,
		Stats:      p.Stats,
		SortMode:   input.Sort,
		DarkMode:   input.Theme == "dark",
		HomeURL:    "#/portfolios",
	}

	return renderTemplate("portfolio", data)
}

func renderWorkspace(portfolio, workspace string, input Input) (string, error) {
	key := portfolio + "/" + workspace
	ws, ok := DemoData.Workspaces[key]
	if !ok {
		return renderPortfolio(portfolio, input)
	}

	ws.SortMode = input.Sort
	ws.DarkMode = input.Theme == "dark"
	ws.HomeURL = "#/portfolios"

	return renderTemplate("workspace", ws)
}

func renderForge(portfolio, workspace, repo string, input Input) (string, error) {
	key := portfolio + "/" + workspace + "/" + repo
	f, ok := DemoData.Forges[key]
	if !ok {
		return renderWorkspace(portfolio, workspace, input)
	}

	f.DarkMode = input.Theme == "dark"
	f.HomeURL = "#/portfolios"

	return renderTemplate("forge", f)
}

func renderTemplate(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.String(), nil
}
