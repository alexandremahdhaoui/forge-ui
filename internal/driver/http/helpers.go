package httpdriver

import (
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/cache"
	forgepkg "github.com/alexandremahdhaoui/forge-ui/internal/forge"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// enrichWorkspaces enriches a slice of WorkspaceSummary in-place with cached
// git info and forge heatmap data. It returns aggregate stats across all
// workspaces. The cacheKey function maps a workspace name to a cache lookup
// key (e.g. "portfolioName/wsName").
func enrichWorkspaces(
	workspaces []types.WorkspaceSummary,
	c *cache.Cache,
	cacheKey func(wsName string) string,
) (totalRepos, dirtyRepos, totalTests, passed, failed int) {
	for i := range workspaces {
		ws := &workspaces[i]
		totalRepos += ws.RepoCount

		// Enrich repos with cached git info.
		for j := range ws.Repos {
			repo := &ws.Repos[j]
			if cached, ok := c.GetRepoOverview(cacheKey(ws.Name), repo.Name); ok {
				repo.Branch = cached.Branch
				repo.IsDirty = cached.IsDirty
				repo.Ahead = cached.Ahead
				repo.Behind = cached.Behind
				repo.HasUpstream = cached.HasUpstream
				repo.LastCommitTime = cached.LastCommitTime
			}
			if repo.IsDirty {
				dirtyRepos++
			}
		}

		// Build forge heatmap data.
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

			ws.RepoForge = append(ws.RepoForge, types.RepoForgeStats{
				RepoName:     repo.Name,
				RepoLink:     repo.RepoLink,
				StageResults: stageResults,
			})
		}
	}

	return totalRepos, dirtyRepos, totalTests, passed, failed
}

// rewriteRepoLinks rewrites all RepoLink fields in the workspaces slice to
// use portfolio-scoped URL paths: /portfolios/{p}/workspaces/{ws}/repos/{repo}.
func rewriteRepoLinks(workspaces []types.WorkspaceSummary, portfolioName string) {
	for i := range workspaces {
		for j := range workspaces[i].Repos {
			repo := &workspaces[i].Repos[j]
			repo.RepoLink = "/portfolios/" + portfolioName + "/workspaces/" + workspaces[i].Name + "/repos/" + repo.Name
		}
	}
}

// maxCommitTime returns the most recent LastCommitTime across all repos.
// Returns the zero time if the slice is empty.
func maxCommitTime(repos []types.RepoOverview) time.Time {
	var max time.Time
	for _, r := range repos {
		if r.LastCommitTime.After(max) {
			max = r.LastCommitTime
		}
	}
	return max
}
