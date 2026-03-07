//go:build unit

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
	"errors"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/mocks/mockadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func stubRepoPlanLoaderForForge() *mockadapter.MockRepoPlanLoader {
	rp := new(mockadapter.MockRepoPlanLoader)
	rp.On("LoadAll", mock.Anything).Return(nil, errors.New("none")).Maybe()
	return rp
}

func TestGetForge_Success(t *testing.T) {
	t.Parallel()

	fl := new(mockadapter.MockForgeLoader)

	fl.On("Load", "/base/myportfolio/ws1/repo-a").Return(types.ForgePageData{
		Spec: types.ForgeSpec{
			Name: "repo-a",
			Test: []types.TestSpec{{Name: "unit", Runner: "go://go-test"}},
		},
	}, nil)

	rp := stubRepoPlanLoaderForForge()
	svc := NewForgeService(fl, rp)
	result, err := svc.GetForge("/base", "myportfolio", "ws1", "repo-a")

	require.NoError(t, err)
	assert.Equal(t, "ws1", result.WorkspaceName)
	assert.Equal(t, "repo-a", result.RepoName)
	assert.Equal(t, "myportfolio", result.PortfolioName)
	assert.Equal(t, "repo-a", result.Spec.Name)

	fl.AssertExpectations(t)
}

func TestGetForge_WithTestReports(t *testing.T) {
	t.Parallel()

	fl := new(mockadapter.MockForgeLoader)

	fl.On("Load", "/base/p1/ws1/repo-a").Return(types.ForgePageData{
		TestReports: []types.TestReport{
			{
				Stage:  "unit",
				Status: "passed",
				Stats:  types.TestStats{Total: 50, Passed: 48, Failed: 2, Skipped: 0},
			},
			{
				Stage:  "integration",
				Status: "failed",
				Stats:  types.TestStats{Total: 10, Passed: 7, Failed: 3, Skipped: 0},
			},
		},
	}, nil)

	rp := stubRepoPlanLoaderForForge()
	svc := NewForgeService(fl, rp)
	result, err := svc.GetForge("/base", "p1", "ws1", "repo-a")

	require.NoError(t, err)
	assert.Equal(t, 60, result.Stats.TotalTests)
	assert.Equal(t, 55, result.Stats.Passed)
	assert.Equal(t, 5, result.Stats.Failed)
	assert.Equal(t, 2, result.Stats.StageCount)
	assert.Equal(t, "passed", result.StageStatusMap["unit"])
	assert.Equal(t, "failed", result.StageStatusMap["integration"])

	fl.AssertExpectations(t)
}

func TestGetForge_WithCoverage(t *testing.T) {
	t.Parallel()

	fl := new(mockadapter.MockForgeLoader)

	fl.On("Load", "/base/p1/ws1/repo-a").Return(types.ForgePageData{
		TestReports: []types.TestReport{
			{
				Stage:    "unit",
				Status:   "passed",
				Stats:    types.TestStats{Total: 20, Passed: 20},
				Coverage: types.Coverage{Enabled: true, Percentage: 85.5},
			},
			{
				Stage:    "lint",
				Status:   "passed",
				Stats:    types.TestStats{Total: 5, Passed: 5},
				Coverage: types.Coverage{Enabled: true, Percentage: 90.0},
			},
		},
	}, nil)

	rp := stubRepoPlanLoaderForForge()
	svc := NewForgeService(fl, rp)
	result, err := svc.GetForge("/base", "p1", "ws1", "repo-a")

	require.NoError(t, err)
	assert.True(t, result.Stats.HasCoverage)
	assert.InDelta(t, 87.75, result.Stats.AvgCoverage, 0.01)

	fl.AssertExpectations(t)
}

func TestGetForge_LoadError(t *testing.T) {
	t.Parallel()

	fl := new(mockadapter.MockForgeLoader)

	fl.On("Load", "/base/p1/ws1/repo-a").Return(types.ForgePageData{}, errors.New("no forge.yaml"))

	rp := stubRepoPlanLoaderForForge()
	svc := NewForgeService(fl, rp)
	_, err := svc.GetForge("/base", "p1", "ws1", "repo-a")

	assert.EqualError(t, err, "no forge.yaml")
	fl.AssertExpectations(t)
}

func TestGetForge_DefaultPortfolio(t *testing.T) {
	t.Parallel()

	fl := new(mockadapter.MockForgeLoader)

	// When portfolio is "default", path should be baseDir/workspace/repo (no portfolio dir)
	fl.On("Load", "/base/ws1/repo-a").Return(types.ForgePageData{
		Spec: types.ForgeSpec{Name: "repo-a"},
	}, nil)

	rp := stubRepoPlanLoaderForForge()
	svc := NewForgeService(fl, rp)
	result, err := svc.GetForge("/base", "default", "ws1", "repo-a")

	require.NoError(t, err)
	assert.Equal(t, "default", result.PortfolioName)
	assert.Equal(t, "repo-a", result.Spec.Name)

	fl.AssertExpectations(t)
}
