// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

const unattendedThreshold = 48 * time.Hour

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
		"timeAgo": timeAgo,
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
	// Compute top 3 recently active portfolios by most recent commit.
	data.TopRecentPortfolios = topRecentPortfolios(data.Portfolios, 3)
	for i := range data.Portfolios {
		data.Portfolios[i].LastActivity = maxPortfolioTime(data.Portfolios[i].Workspaces)
	}
	data.Unattended = unattendedPortfolios(data.Portfolios)

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
	// Compute top 3 recently active workspaces by most recent commit.
	data.TopRecentWorkspaces = topRecentWorkspaces(data.Workspaces, 3)
	for i := range data.Workspaces {
		data.Workspaces[i].LastActivity = maxWorkspaceTime(data.Workspaces[i].Repos)
		for _, r := range data.Workspaces[i].Repos {
			if r.IsDirty {
				data.Workspaces[i].DirtyRepoCount++
			}
		}
	}
	data.Unattended = unattendedWorkspaces(data.Workspaces)

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
	// Compute top 3 recently active repos by most recent commit.
	data.TopRecentRepos = topRecentRepos(data.Repos, 3)
	data.Unattended = unattendedRepos(data.Repos)

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
	seen := make(map[string]bool)
	for _, rpt := range data.TestReports {
		if !seen[rpt.Stage] {
			seen[rpt.Stage] = true
			data.LatestStageReports = append(data.LatestStageReports, rpt)
		}
	}

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

// topRecentPortfolios returns up to n portfolios sorted by most recent commit time.
func topRecentPortfolios(portfolios []types.PortfolioSummary, n int) []types.PortfolioSummary {
	if len(portfolios) == 0 {
		return nil
	}
	sorted := make([]types.PortfolioSummary, len(portfolios))
	copy(sorted, portfolios)
	sort.Slice(sorted, func(i, j int) bool {
		return maxPortfolioTime(sorted[i].Workspaces).After(maxPortfolioTime(sorted[j].Workspaces))
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// topRecentWorkspaces returns up to n workspaces sorted by most recent commit time.
func topRecentWorkspaces(workspaces []types.WorkspaceSummary, n int) []types.WorkspaceSummary {
	if len(workspaces) == 0 {
		return nil
	}
	sorted := make([]types.WorkspaceSummary, len(workspaces))
	copy(sorted, workspaces)
	sort.Slice(sorted, func(i, j int) bool {
		return maxWorkspaceTime(sorted[i].Repos).After(maxWorkspaceTime(sorted[j].Repos))
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// topRecentRepos returns up to n repos sorted by most recent commit time.
func topRecentRepos(repos []types.RepoSummary, n int) []types.RepoSummary {
	if len(repos) == 0 {
		return nil
	}
	sorted := make([]types.RepoSummary, len(repos))
	copy(sorted, repos)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LastCommitTime.After(sorted[j].LastCommitTime)
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// maxPortfolioTime returns the most recent commit time across all repos in all workspaces.
func maxPortfolioTime(workspaces []types.WorkspaceSummary) time.Time {
	var max time.Time
	for _, ws := range workspaces {
		if t := maxWorkspaceTime(ws.Repos); t.After(max) {
			max = t
		}
	}
	return max
}

// maxWorkspaceTime returns the most recent commit time across RepoOverview items.
func maxWorkspaceTime(repos []types.RepoOverview) time.Time {
	var max time.Time
	for _, r := range repos {
		if r.LastCommitTime.After(max) {
			max = r.LastCommitTime
		}
	}
	return max
}

// timeAgo returns a human-readable relative time string.
func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%d mo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}

// isRepoUnattended returns true if a repo has uncommitted changes or unpushed
// commits and the last commit is older than the unattended threshold.
func isRepoUnattended(dirty bool, ahead int, lastCommit time.Time) bool {
	if !dirty && ahead <= 0 {
		return false
	}
	return time.Since(lastCommit) > unattendedThreshold
}

// unattendedPortfolios returns portfolios containing at least one unattended repo.
func unattendedPortfolios(portfolios []types.PortfolioSummary) []types.PortfolioSummary {
	var result []types.PortfolioSummary
	for _, p := range portfolios {
		for _, ws := range p.Workspaces {
			if hasUnattendedRepo(ws.Repos) {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

// unattendedWorkspaces returns workspaces containing at least one unattended repo.
func unattendedWorkspaces(workspaces []types.WorkspaceSummary) []types.WorkspaceSummary {
	var result []types.WorkspaceSummary
	for _, ws := range workspaces {
		if hasUnattendedRepo(ws.Repos) {
			result = append(result, ws)
		}
	}
	return result
}

// hasUnattendedRepo checks if any RepoOverview in the slice is unattended.
func hasUnattendedRepo(repos []types.RepoOverview) bool {
	for _, r := range repos {
		if isRepoUnattended(r.IsDirty, r.Ahead, r.LastCommitTime) {
			return true
		}
	}
	return false
}

// unattendedRepos returns repos that have dirty/ahead state and old last commit.
func unattendedRepos(repos []types.RepoSummary) []types.RepoSummary {
	var result []types.RepoSummary
	for _, r := range repos {
		if isRepoUnattended(r.IsDirty, r.Ahead, r.LastCommitTime) {
			result = append(result, r)
		}
	}
	return result
}
