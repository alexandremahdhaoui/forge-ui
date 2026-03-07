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

package controller

import (
	"path/filepath"
	"sort"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// WorkspaceService provides business logic for workspace pages.
type WorkspaceService interface {
	GetWorkspace(baseDir, portfolio, workspace, sortMode string) (types.WorkspacePageData, error)
}

type workspaceService struct {
	workspaceDisc  adapter.WorkspaceDiscovery
	cache          adapter.Cache
	forgeLoader    adapter.ForgeLoader
	wsConfigLoader adapter.WsConfigLoader
	metaPlanLoader adapter.MetaPlanLoader
	repoPlanLoader adapter.RepoPlanLoader
}

// NewWorkspaceService creates a WorkspaceService.
func NewWorkspaceService(
	ws adapter.WorkspaceDiscovery,
	c adapter.Cache,
	fl adapter.ForgeLoader,
	wc adapter.WsConfigLoader,
	mp adapter.MetaPlanLoader,
	rp adapter.RepoPlanLoader,
) WorkspaceService {
	return &workspaceService{
		workspaceDisc:  ws,
		cache:          c,
		forgeLoader:    fl,
		wsConfigLoader: wc,
		metaPlanLoader: mp,
		repoPlanLoader: rp,
	}
}

func (s *workspaceService) GetWorkspace(baseDir, portfolio, workspace, sortMode string) (types.WorkspacePageData, error) {
	wsBaseDir := baseDir
	if portfolio != "default" {
		wsBaseDir = filepath.Join(baseDir, portfolio)
	}

	data, err := s.workspaceDisc.Get(wsBaseDir, workspace)
	if err != nil {
		return types.WorkspacePageData{}, err
	}

	cacheKey := portfolio + "/" + workspace

	// Enrich each repo with cached git information.
	for i, repo := range data.Repos {
		if cached, ok := s.cache.GetRepoSummary(cacheKey, repo.Name); ok {
			data.Repos[i].Branch = cached.Branch
			data.Repos[i].IsDirty = cached.IsDirty
			data.Repos[i].StatusFiles = cached.StatusFiles
			data.Repos[i].DiffStat = cached.DiffStat
			data.Repos[i].RecentLogs = cached.RecentLogs
			data.Repos[i].Ahead = cached.Ahead
			data.Repos[i].Behind = cached.Behind
			data.Repos[i].HasUpstream = cached.HasUpstream
			data.Repos[i].LastCommitTime = cached.LastCommitTime
		}
	}

	// Sort by last commit time if requested, otherwise sort by name.
	if sortMode == "time" {
		sort.Slice(data.Repos, func(a, b int) bool {
			return data.Repos[a].LastCommitTime.After(data.Repos[b].LastCommitTime)
		})
	} else {
		sort.Slice(data.Repos, func(a, b int) bool {
			return data.Repos[a].Name < data.Repos[b].Name
		})
	}

	// Rewrite repo links for portfolio-scoped URLs.
	for i := range data.Repos {
		data.Repos[i].RepoLink = "/portfolios/" + portfolio + "/workspaces/" + workspace + "/repos/" + data.Repos[i].Name
	}

	// Load forge data for repos that have forge.yaml and build heatmap data.
	stageSeen := make(map[string]struct{})
	var allStages []string
	var stats types.WorkspaceStats
	stats.TotalRepos = len(data.Repos)

	for _, repo := range data.Repos {
		if !repo.HasForge {
			continue
		}
		stats.ForgeRepos++

		forgeData, err := s.forgeLoader.Load(repo.Path)
		if err != nil {
			continue
		}

		stageResults := make(map[string]string)
		for _, rpt := range forgeData.TestReports {
			if _, seen := stageResults[rpt.Stage]; !seen {
				stageResults[rpt.Stage] = rpt.Status
			}
			stats.TotalTests += rpt.Stats.Total
			stats.Passed += rpt.Stats.Passed
			stats.Failed += rpt.Stats.Failed
			stats.Skipped += rpt.Stats.Skipped
		}

		for _, ts := range forgeData.Spec.Test {
			if _, seen := stageSeen[ts.Name]; !seen {
				stageSeen[ts.Name] = struct{}{}
				allStages = append(allStages, ts.Name)
			}
		}
		for _, rpt := range forgeData.TestReports {
			if _, seen := stageSeen[rpt.Stage]; !seen {
				stageSeen[rpt.Stage] = struct{}{}
				allStages = append(allStages, rpt.Stage)
			}
		}

		data.RepoForge = append(data.RepoForge, types.RepoForgeStats{
			RepoName:     repo.Name,
			RepoLink:     "/portfolios/" + portfolio + "/workspaces/" + workspace + "/repos/" + repo.Name,
			StageResults: stageResults,
		})
	}

	data.AllStages = allStages
	data.Stats = stats
	data.PortfolioName = portfolio
	data.SortMode = sortMode

	// Load workspace orchestration data.
	wsPath := filepath.Join(wsBaseDir, workspace)
	if cfg, err := s.wsConfigLoader.Load(wsPath); err == nil {
		data.Description = cfg.Description
		if len(cfg.Repos) > 0 {
			data.RepoRoles = make(map[string]string, len(cfg.Repos))
			for _, r := range cfg.Repos {
				if r.Description != "" {
					data.RepoRoles[r.Name] = r.Description
				}
			}
		}
	}

	if plans, err := s.metaPlanLoader.LoadAll(wsPath); err == nil && len(plans) > 0 {
		data.MetaPlans = plans
	}

	for _, repo := range data.Repos {
		summary, err := s.repoPlanLoader.LoadSummary(repo.Path, repo.Name)
		if err != nil || summary.TasksTotal == 0 {
			continue
		}
		data.RepoPlanSummaries = append(data.RepoPlanSummaries, summary)
	}

	return data, nil
}
