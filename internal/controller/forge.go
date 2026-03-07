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

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// ForgeService provides business logic for forge (repo detail) pages.
type ForgeService interface {
	GetForge(baseDir, portfolio, workspace, repo string) (types.ForgePageData, error)
}

type forgeService struct {
	forgeLoader    adapter.ForgeLoader
	repoPlanLoader adapter.RepoPlanLoader
}

// NewForgeService creates a ForgeService.
func NewForgeService(fl adapter.ForgeLoader, rp adapter.RepoPlanLoader) ForgeService {
	return &forgeService{forgeLoader: fl, repoPlanLoader: rp}
}

func (s *forgeService) GetForge(baseDir, portfolio, workspace, repo string) (types.ForgePageData, error) {
	var repoPath string
	if portfolio == "default" {
		repoPath = filepath.Join(baseDir, workspace, repo)
	} else {
		repoPath = filepath.Join(baseDir, portfolio, workspace, repo)
	}

	data, err := s.forgeLoader.Load(repoPath)
	if err != nil {
		return types.ForgePageData{}, err
	}

	data.WorkspaceName = workspace
	data.RepoName = repo
	data.PortfolioName = portfolio

	// Compute test statistics from reports.
	var stats types.ForgeStats
	var coverageSum float64
	var coverageCount int
	stageSet := make(map[string]struct{})

	stageStatusMap := make(map[string]string)
	for _, rpt := range data.TestReports {
		stats.TotalTests += rpt.Stats.Total
		stats.Passed += rpt.Stats.Passed
		stats.Failed += rpt.Stats.Failed
		stats.Skipped += rpt.Stats.Skipped
		stageSet[rpt.Stage] = struct{}{}
		if rpt.Coverage.Enabled {
			coverageSum += rpt.Coverage.Percentage
			coverageCount++
		}
		if _, seen := stageStatusMap[rpt.Stage]; !seen {
			stageStatusMap[rpt.Stage] = rpt.Status
		}
	}
	if coverageCount > 0 {
		stats.AvgCoverage = coverageSum / float64(coverageCount)
		stats.HasCoverage = true
	}
	stats.StageCount = len(stageSet)
	data.Stats = stats
	data.StageStatusMap = stageStatusMap

	// Load repo plan progress.
	if plans, err := s.repoPlanLoader.LoadAll(repoPath); err == nil && len(plans) > 0 {
		data.RepoPlans = plans
	}

	return data, nil
}
