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

package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/mocks/mockcontroller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestHandler creates an APIHandler wired to mock services and returns
// an http.Handler using the generated strict handler + mux.
func newTestHandler(
	ps *mockcontroller.MockPortfolioService,
	ws *mockcontroller.MockWorkspaceService,
	fs *mockcontroller.MockForgeService,
) (*APIHandler, http.Handler) {
	h := NewAPIHandler("/base", ps, ws, fs, nil, nil)
	strict := NewStrictHandler(h, nil)
	mux := http.NewServeMux()
	return h, HandlerFromMux(strict, mux)
}

// --- ListPortfolios ---

func TestListPortfolios_Success(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "time").Return(types.PortfoliosPageData{
		Portfolios: []types.PortfolioSummary{{Name: "p1"}},
		SortMode:   "time",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var body types.PortfoliosPageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	require.Len(t, body.Portfolios, 1)
	assert.Equal(t, "p1", body.Portfolios[0].Name)

	ps.AssertExpectations(t)
}

func TestListPortfolios_SortParam(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "name").Return(types.PortfoliosPageData{
		SortMode: "name",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios?sort=name", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body types.PortfoliosPageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "name", body.SortMode)

	ps.AssertExpectations(t)
}

func TestListPortfolios_DefaultSort(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	// No sort param should default to "time".
	ps.On("ListPortfolios", "/base", "time").Return(types.PortfoliosPageData{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	ps.AssertExpectations(t)
}

func TestListPortfolios_Error(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "time").Return(types.PortfoliosPageData{}, errors.New("disk error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "disk error", body.Error)

	ps.AssertExpectations(t)
}

func TestListPortfolios_ExcludesDarkModeFields(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "time").Return(types.PortfoliosPageData{
		Portfolios: []types.PortfolioSummary{{Name: "p1"}},
		DarkMode:   true,
		HomeURL:    "/portfolios",
		LightPalette: "2",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	rawBody := w.Body.String()
	assert.NotContains(t, rawBody, "darkMode")
	assert.NotContains(t, rawBody, "homeURL")
	assert.NotContains(t, rawBody, "lightPalette")

	ps.AssertExpectations(t)
}

// --- GetPortfolio ---

func TestGetPortfolio_Success(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	ps.On("GetPortfolio", "/base", "myp", "time").Return(types.PortfolioPageData{
		Name: "myp",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/myp", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var body types.PortfolioPageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "myp", body.Name)

	ps.AssertExpectations(t)
}

func TestGetPortfolio_SortParam(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	ps.On("GetPortfolio", "/base", "myp", "name").Return(types.PortfolioPageData{
		Name:     "myp",
		SortMode: "name",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/myp?sort=name", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body types.PortfolioPageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "name", body.SortMode)

	ps.AssertExpectations(t)
}

func TestGetPortfolio_NotFound(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	// Service returns data with empty Name -- signals not found.
	ps.On("GetPortfolio", "/base", "missing", "time").Return(types.PortfolioPageData{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/missing", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "portfolio not found", body.Error)

	ps.AssertExpectations(t)
}

func TestGetPortfolio_Error(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	ps.On("GetPortfolio", "/base", "bad", "time").Return(types.PortfolioPageData{}, errors.New("read error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/bad", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "read error", body.Error)

	ps.AssertExpectations(t)
}

func TestGetPortfolio_ExcludesDarkModeFields(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.MockPortfolioService)
	_, handler := newTestHandler(ps, nil, nil)

	ps.On("GetPortfolio", "/base", "myp", "time").Return(types.PortfolioPageData{
		Name:         "myp",
		DarkMode:     true,
		HomeURL:      "/portfolios",
		LightPalette: "3",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/myp", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	rawBody := w.Body.String()
	assert.NotContains(t, rawBody, "darkMode")
	assert.NotContains(t, rawBody, "homeURL")
	assert.NotContains(t, rawBody, "lightPalette")

	ps.AssertExpectations(t)
}

// --- GetWorkspace ---

func TestGetWorkspace_Success(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	_, handler := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "time").Return(types.WorkspacePageData{
		Name: "ws1",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var body types.WorkspacePageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ws1", body.Name)

	wsSvc.AssertExpectations(t)
}

func TestGetWorkspace_SortParam(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	_, handler := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "name").Return(types.WorkspacePageData{
		Name:     "ws1",
		SortMode: "name",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1?sort=name", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body types.WorkspacePageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "name", body.SortMode)

	wsSvc.AssertExpectations(t)
}

func TestGetWorkspace_NotFound(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	_, handler := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "missing", "time").Return(types.WorkspacePageData{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/missing", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "workspace not found", body.Error)

	wsSvc.AssertExpectations(t)
}

func TestGetWorkspace_Error(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	_, handler := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "time").Return(types.WorkspacePageData{}, errors.New("io error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "io error", body.Error)

	wsSvc.AssertExpectations(t)
}

func TestGetWorkspace_ExcludesDarkModeFields(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	_, handler := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "time").Return(types.WorkspacePageData{
		Name:         "ws1",
		DarkMode:     true,
		HomeURL:      "/portfolios",
		LightPalette: "4",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	rawBody := w.Body.String()
	assert.NotContains(t, rawBody, "darkMode")
	assert.NotContains(t, rawBody, "homeURL")
	assert.NotContains(t, rawBody, "lightPalette")

	wsSvc.AssertExpectations(t)
}

// --- GetRepo ---

func TestGetRepo_Success(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	fsSvc := new(mockcontroller.MockForgeService)
	_, handler := newTestHandler(nil, wsSvc, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "repo-a").Return(types.ForgePageData{
		RepoName: "repo-a",
	}, nil)
	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "name").Return(types.WorkspacePageData{
		Repos: []types.RepoSummary{
			{Name: "repo-a", RepoLink: "/portfolios/p1/workspaces/ws1/repos/repo-a"},
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1/repos/repo-a", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var body types.ForgePageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "repo-a", body.RepoName)

	fsSvc.AssertExpectations(t)
	wsSvc.AssertExpectations(t)
}

func TestGetRepo_NotFound(t *testing.T) {
	t.Parallel()

	fsSvc := new(mockcontroller.MockForgeService)
	_, handler := newTestHandler(nil, nil, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "missing").Return(types.ForgePageData{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1/repos/missing", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "repo not found", body.Error)

	fsSvc.AssertExpectations(t)
}

func TestGetRepo_Error(t *testing.T) {
	t.Parallel()

	fsSvc := new(mockcontroller.MockForgeService)
	_, handler := newTestHandler(nil, nil, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "bad-repo").Return(types.ForgePageData{}, errors.New("no forge.yaml"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1/repos/bad-repo", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "no forge.yaml", body.Error)

	fsSvc.AssertExpectations(t)
}

func TestGetRepo_ExcludesDarkModeFields(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	fsSvc := new(mockcontroller.MockForgeService)
	_, handler := newTestHandler(nil, wsSvc, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "repo-a").Return(types.ForgePageData{
		RepoName:     "repo-a",
		DarkMode:     true,
		HomeURL:      "/portfolios",
		LightPalette: "1",
	}, nil)
	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "name").Return(types.WorkspacePageData{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1/repos/repo-a", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	rawBody := w.Body.String()
	assert.NotContains(t, rawBody, "darkMode")
	assert.NotContains(t, rawBody, "homeURL")
	assert.NotContains(t, rawBody, "lightPalette")

	fsSvc.AssertExpectations(t)
	wsSvc.AssertExpectations(t)
}

func TestGetRepo_PopulatesSiblingRepos(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	fsSvc := new(mockcontroller.MockForgeService)
	_, handler := newTestHandler(nil, wsSvc, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "repo-b").Return(types.ForgePageData{
		RepoName: "repo-b",
	}, nil)
	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "name").Return(types.WorkspacePageData{
		Repos: []types.RepoSummary{
			{Name: "repo-a", RepoLink: "/portfolios/p1/workspaces/ws1/repos/repo-a"},
			{Name: "repo-b", RepoLink: "/portfolios/p1/workspaces/ws1/repos/repo-b"},
			{Name: "repo-c", RepoLink: "/portfolios/p1/workspaces/ws1/repos/repo-c"},
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1/repos/repo-b", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body types.ForgePageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	require.Len(t, body.SiblingRepos, 3)

	// Verify names.
	assert.Equal(t, "repo-a", body.SiblingRepos[0].Name)
	assert.Equal(t, "repo-b", body.SiblingRepos[1].Name)
	assert.Equal(t, "repo-c", body.SiblingRepos[2].Name)

	// Verify the requested repo is marked active.
	assert.False(t, body.SiblingRepos[0].IsActive)
	assert.True(t, body.SiblingRepos[1].IsActive)
	assert.False(t, body.SiblingRepos[2].IsActive)

	// Verify links do NOT have "#" prefix (server-side links are plain paths).
	assert.Equal(t, "/portfolios/p1/workspaces/ws1/repos/repo-a", body.SiblingRepos[0].Link)
	assert.Equal(t, "/portfolios/p1/workspaces/ws1/repos/repo-b", body.SiblingRepos[1].Link)
	assert.Equal(t, "/portfolios/p1/workspaces/ws1/repos/repo-c", body.SiblingRepos[2].Link)

	fsSvc.AssertExpectations(t)
	wsSvc.AssertExpectations(t)
}

func TestGetRepo_WorkspaceServiceError_SkipsSiblingRepos(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.MockWorkspaceService)
	fsSvc := new(mockcontroller.MockForgeService)
	_, handler := newTestHandler(nil, wsSvc, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "repo-a").Return(types.ForgePageData{
		RepoName: "repo-a",
	}, nil)
	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "name").Return(types.WorkspacePageData{}, errors.New("workspace lookup failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/p1/workspaces/ws1/repos/repo-a", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body types.ForgePageData
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "repo-a", body.RepoName)
	assert.Nil(t, body.SiblingRepos)

	fsSvc.AssertExpectations(t)
	wsSvc.AssertExpectations(t)
}

// --- sortOrDefault ---

func TestSortOrDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *ListPortfoliosParamsSort
		expected string
	}{
		{"nil defaults to time", nil, "time"},
		{"name passes through", ptrSort(ListPortfoliosParamsSortName), "name"},
		{"time passes through", ptrSort(ListPortfoliosParamsSortTime), "time"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sortOrDefault(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func ptrSort(s ListPortfoliosParamsSort) *ListPortfoliosParamsSort {
	return &s
}
