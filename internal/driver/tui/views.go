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

//go:build !js || !wasm

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderPortfolioList(m Model) string {
	var sidebar strings.Builder
	sidebar.WriteString(titleStyle.Render("Portfolios") + "\n\n")

	for i, p := range m.portfolios {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		sidebar.WriteString(cursor + style.Render(p.Name) + "\n")
	}

	sideContent := sidebarStyle.Render(sidebar.String())

	// Right panel: stats
	var right strings.Builder
	if m.cursor < len(m.portfolios) {
		p := m.portfolios[m.cursor]
		right.WriteString(titleStyle.Render("Portfolio: "+p.Name) + "\n\n")
		right.WriteString(fmt.Sprintf("  Workspaces: %d\n", p.Stats.TotalWorkspaces))
		right.WriteString(fmt.Sprintf("  Repos:      %d\n", p.Stats.TotalRepos))
		right.WriteString(fmt.Sprintf("  Dirty:      %d\n", p.Stats.DirtyRepos))
		if p.Description != "" {
			right.WriteString(fmt.Sprintf("\n  %s\n", p.Description))
		}
	}
	right.WriteString("\n" + helpStyle.Render("  j/k: navigate  enter: select  q: quit"))

	rightContent := contentStyle.Render(right.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, sideContent, rightContent)
}

func renderWorkspaceList(m Model) string {
	var sidebar strings.Builder
	sidebar.WriteString(titleStyle.Render("Workspaces") + "\n\n")

	for i, ws := range m.workspaceData.Workspaces {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		dirtyMark := ""
		if ws.DirtyRepoCount > 0 {
			dirtyMark = " *"
		}
		sidebar.WriteString(cursor + style.Render(ws.Name+dirtyMark) + "\n")
	}

	sideContent := sidebarStyle.Render(sidebar.String())

	// Right panel
	var right strings.Builder
	right.WriteString(titleStyle.Render("Portfolio: "+m.selectedPortfolio) + "\n\n")

	if m.cursor < len(m.workspaceData.Workspaces) {
		ws := m.workspaceData.Workspaces[m.cursor]
		right.WriteString(fmt.Sprintf("  %s  %d repos\n", ws.Name, ws.RepoCount))
		if ws.Description != "" {
			right.WriteString(fmt.Sprintf("  %s\n", ws.Description))
		}

		// Show managed workspace status if available
		for _, mws := range m.managedWs {
			if mws.Name == ws.Name {
				phase := statusStyle(mws.Phase).Render(mws.Phase)
				right.WriteString(fmt.Sprintf("\n  K8s Status: %s", phase))
				if mws.Suspended {
					right.WriteString("  (suspended)")
				}
				right.WriteString("\n")
				break
			}
		}

		// Show CU state if available
		if m.compoState.Name != "" {
			right.WriteString(fmt.Sprintf("\n  CU: %s  branch: %s\n", m.compoState.Name, m.compoState.CurrentBranch))
			if len(m.compoState.Repos) > 0 {
				right.WriteString("  Repos:\n")
				for _, r := range m.compoState.Repos {
					right.WriteString(fmt.Sprintf("    %s\n", r.Name))
				}
			}
		}
	}

	// Stats
	right.WriteString(fmt.Sprintf("\n  Stats: %d ws, %d repos, %d dirty\n",
		m.workspaceData.Stats.TotalWorkspaces,
		m.workspaceData.Stats.TotalRepos,
		m.workspaceData.Stats.DirtyRepos))

	right.WriteString("\n" + helpStyle.Render("  j/k: navigate  enter: repos  esc: back"))
	right.WriteString("\n" + helpStyle.Render("  s: suspend  r: resume  d: delete"))

	rightContent := contentStyle.Render(right.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, sideContent, rightContent)
}

func renderRepoDetail(m Model) string {
	var sidebar strings.Builder
	sidebar.WriteString(titleStyle.Render("Repos") + "\n\n")

	for i, repo := range m.repoData.Repos {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		branchInfo := ""
		if repo.Branch != "" {
			branchInfo = " [" + repo.Branch + "]"
		}
		dirtyMark := ""
		if repo.IsDirty {
			dirtyMark = " *"
		}
		sidebar.WriteString(cursor + style.Render(repo.Name+branchInfo+dirtyMark) + "\n")
	}

	sideContent := sidebarStyle.Render(sidebar.String())

	// Right panel
	var right strings.Builder
	right.WriteString(titleStyle.Render(m.repoData.Name) + "\n\n")

	if m.cursor < len(m.repoData.Repos) {
		repo := m.repoData.Repos[m.cursor]
		right.WriteString(fmt.Sprintf("  Repo: %s\n", repo.Name))
		right.WriteString(fmt.Sprintf("  Branch: %s\n", repo.Branch))
		if repo.IsDirty {
			right.WriteString(fmt.Sprintf("  Status: dirty (%d files)\n", len(repo.StatusFiles)))
		}
		if repo.HasForge {
			right.WriteString("  Forge: yes\n")
		}
		if repo.Ahead > 0 || repo.Behind > 0 {
			right.WriteString(fmt.Sprintf("  Ahead: %d  Behind: %d\n", repo.Ahead, repo.Behind))
		}
		if len(repo.RecentLogs) > 0 {
			right.WriteString("\n  Recent commits:\n")
			limit := min(5, len(repo.RecentLogs))
			for _, log := range repo.RecentLogs[:limit] {
				right.WriteString(fmt.Sprintf("    %s %s\n", log.Hash, log.Message))
			}
		}
	}

	right.WriteString(fmt.Sprintf("\n  Stats: %d repos, %d forge\n",
		m.repoData.Stats.TotalRepos, m.repoData.Stats.ForgeRepos))

	right.WriteString("\n" + helpStyle.Render("  j/k: navigate  enter: forge  esc: back"))

	rightContent := contentStyle.Render(right.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, sideContent, rightContent)
}

func renderForgeResults(m Model) string {
	var content strings.Builder
	content.WriteString(titleStyle.Render(m.forgeData.RepoName+" — Forge Results") + "\n\n")

	// Build artifacts
	if len(m.forgeData.Artifacts) > 0 {
		content.WriteString("  Build Artifacts:\n")
		for _, a := range m.forgeData.Artifacts {
			content.WriteString(fmt.Sprintf("    %s (%s) %s\n", a.Name, a.Type, a.Version))
		}
		content.WriteString("\n")
	}

	// Test results
	if len(m.forgeData.TestReports) > 0 {
		content.WriteString("  Test Results:\n")
		for _, tr := range m.forgeData.TestReports {
			status := statusStyle(tr.Status).Render(tr.Status)
			content.WriteString(fmt.Sprintf("    %-12s %s  %d/%d passed  %.1fs\n",
				tr.Stage, status, tr.Stats.Passed, tr.Stats.Total, tr.Duration))
		}
		content.WriteString("\n")
	}

	// Test envs
	if len(m.forgeData.TestEnvs) > 0 {
		content.WriteString("  Test Environments:\n")
		for _, te := range m.forgeData.TestEnvs {
			content.WriteString(fmt.Sprintf("    %s (%s)\n", te.Name, te.Status))
		}
		content.WriteString("\n")
	}

	// Stats
	content.WriteString(fmt.Sprintf("  Total: %d  Passed: %d  Failed: %d  Skipped: %d\n",
		m.forgeData.Stats.TotalTests, m.forgeData.Stats.Passed,
		m.forgeData.Stats.Failed, m.forgeData.Stats.Skipped))

	if m.forgeData.Stats.HasCoverage {
		content.WriteString(fmt.Sprintf("  Coverage: %.1f%%\n", m.forgeData.Stats.AvgCoverage))
	}

	content.WriteString("\n" + helpStyle.Render("  esc: back  q: quit"))

	return contentStyle.Render(content.String())
}
