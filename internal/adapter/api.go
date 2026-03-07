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

//go:build js && wasm

package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

type apiDataSource struct {
	baseURL string
	client  *http.Client
}

// NewAPIDataSource returns a DataSource that fetches data from the REST API.
// baseURL is the full backend URL (e.g., "http://localhost:8081/api/v1").
func NewAPIDataSource(baseURL string) DataSource {
	return &apiDataSource{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (a *apiDataSource) ListPortfolios(sort string) (types.PortfoliosPageData, error) {
	u := a.baseURL + "/portfolios?sort=" + url.QueryEscape(sort)

	var data types.PortfoliosPageData
	if err := a.getJSON(u, &data); err != nil {
		return types.PortfoliosPageData{}, err
	}

	for i := range data.Portfolios {
		prefixSummaryRepoLinks(data.Portfolios[i].Workspaces)
	}
	return data, nil
}

func (a *apiDataSource) GetPortfolio(name, sort string) (types.PortfolioPageData, error) {
	u := a.baseURL + "/portfolios/" + url.PathEscape(name) + "?sort=" + url.QueryEscape(sort)

	var data types.PortfolioPageData
	if err := a.getJSON(u, &data); err != nil {
		if isNotFound(err) {
			return types.PortfolioPageData{}, nil
		}
		return types.PortfolioPageData{}, err
	}

	prefixSummaryRepoLinks(data.Workspaces)
	return data, nil
}

func (a *apiDataSource) GetWorkspace(portfolio, workspace, sort string) (types.WorkspacePageData, error) {
	u := a.baseURL + "/portfolios/" + url.PathEscape(portfolio) +
		"/workspaces/" + url.PathEscape(workspace) +
		"?sort=" + url.QueryEscape(sort)

	var data types.WorkspacePageData
	if err := a.getJSON(u, &data); err != nil {
		if isNotFound(err) {
			return types.WorkspacePageData{}, nil
		}
		return types.WorkspacePageData{}, err
	}

	prefixWorkspaceRepoLinks(&data)
	return data, nil
}

func (a *apiDataSource) GetForge(portfolio, workspace, repo string) (types.ForgePageData, error) {
	u := a.baseURL + "/portfolios/" + url.PathEscape(portfolio) +
		"/workspaces/" + url.PathEscape(workspace) +
		"/repos/" + url.PathEscape(repo)

	var data types.ForgePageData
	if err := a.getJSON(u, &data); err != nil {
		if isNotFound(err) {
			return types.ForgePageData{}, nil
		}
		return types.ForgePageData{}, err
	}

	// Add "#" prefix to SiblingRepos links for hash-based routing.
	for i := range data.SiblingRepos {
		data.SiblingRepos[i].Link = "#" + data.SiblingRepos[i].Link
	}

	return data, nil
}

// getJSON performs an HTTP GET and decodes the JSON response into dst.
// Returns an apiError for non-200 status codes.
func (a *apiDataSource) getJSON(rawURL string, dst any) error {
	resp, err := a.client.Get(rawURL)
	if err != nil {
		return fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &apiError{
			statusCode: resp.StatusCode,
			body:       string(body),
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("api response decode failed: %w", err)
	}
	return nil
}

// apiError represents a non-200 HTTP response from the API.
type apiError struct {
	statusCode int
	body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("api returned status %d: %s", e.statusCode, e.body)
}

// isNotFound returns true if the error represents a 404 response.
func isNotFound(err error) bool {
	if ae, ok := err.(*apiError); ok {
		return ae.statusCode == http.StatusNotFound
	}
	return false
}

// prefixWorkspaceRepoLinks adds "#" prefix to all RepoLink fields in WorkspacePageData.
func prefixWorkspaceRepoLinks(data *types.WorkspacePageData) {
	for i := range data.Repos {
		data.Repos[i].RepoLink = "#" + data.Repos[i].RepoLink
	}
	for i := range data.RepoForge {
		data.RepoForge[i].RepoLink = "#" + data.RepoForge[i].RepoLink
	}
}

// prefixSummaryRepoLinks adds "#" prefix to RepoLink fields in a slice of WorkspaceSummary.
func prefixSummaryRepoLinks(workspaces []types.WorkspaceSummary) {
	for i := range workspaces {
		for j := range workspaces[i].Repos {
			workspaces[i].Repos[j].RepoLink = "#" + workspaces[i].Repos[j].RepoLink
		}
		for j := range workspaces[i].RepoForge {
			workspaces[i].RepoForge[j].RepoLink = "#" + workspaces[i].RepoForge[j].RepoLink
		}
	}
}
