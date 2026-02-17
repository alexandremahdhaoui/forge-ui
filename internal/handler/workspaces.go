package handler

import (
	"net/http"

	forgepkg "github.com/alexandremahdhaoui/forge-ui/internal/forge"
	gitpkg "github.com/alexandremahdhaoui/forge-ui/internal/git"
	"github.com/alexandremahdhaoui/forge-ui/internal/model"
	"github.com/alexandremahdhaoui/forge-ui/internal/workspace"
)

// HandleWorkspaces handles GET /workspaces.
func (h *Handler) HandleWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := workspace.List(h.BaseDir)
	if err != nil {
		http.Error(w, "failed to list workspaces: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich repos with git info.
	totalRepos := 0
	dirtyRepos := 0
	for i := range workspaces {
		totalRepos += workspaces[i].RepoCount
		for j := range workspaces[i].Repos {
			repo := &workspaces[i].Repos[j]
			gitInfo, err := gitpkg.RepoInfo(repo.Path)
			if err != nil {
				continue
			}
			repo.Branch = gitInfo.Branch
			repo.IsDirty = gitInfo.IsDirty
			repo.Ahead = gitInfo.Ahead
			repo.Behind = gitInfo.Behind
			repo.HasUpstream = gitInfo.HasUpstream
			if repo.IsDirty {
				dirtyRepos++
			}
		}
	}

	// Build per-workspace forge heatmap data.
	var totalTests, passed, failed int

	for i := range workspaces {
		ws := &workspaces[i]
		stageSeen := make(map[string]struct{})

		for _, repo := range ws.Repos {
			if !repo.HasForge {
				continue
			}
			forgeData, err := forgepkg.Load(repo.Path)
			if err != nil {
				continue
			}

			stageResults := make(map[string]string)
			for _, rpt := range forgeData.TestReports {
				if _, seen := stageResults[rpt.Stage]; !seen {
					stageResults[rpt.Stage] = rpt.Status
				}
				totalTests += rpt.Stats.Total
				passed += rpt.Stats.Passed
				failed += rpt.Stats.Failed
			}

			for _, ts := range forgeData.Spec.Test {
				if _, seen := stageSeen[ts.Name]; !seen {
					stageSeen[ts.Name] = struct{}{}
					ws.AllStages = append(ws.AllStages, ts.Name)
				}
			}
			for _, rpt := range forgeData.TestReports {
				if _, seen := stageSeen[rpt.Stage]; !seen {
					stageSeen[rpt.Stage] = struct{}{}
					ws.AllStages = append(ws.AllStages, rpt.Stage)
				}
			}

			ws.RepoForge = append(ws.RepoForge, model.RepoForgeStats{
				RepoName:     repo.Name,
				RepoLink:    repo.RepoLink,
				StageResults: stageResults,
			})
		}
	}

	data := model.WorkspacesPageData{
		Stats: model.WorkspacesStats{
			TotalWorkspaces: len(workspaces),
			TotalRepos:      totalRepos,
			DirtyRepos:      dirtyRepos,
			TotalTests:      totalTests,
			Passed:          passed,
			Failed:          failed,
		},
		Workspaces: workspaces,
	}

	h.render(w, "workspaces", data)
}
