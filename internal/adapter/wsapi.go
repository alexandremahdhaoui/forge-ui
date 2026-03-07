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

package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// Compile-time check.
var _ WorkspaceAPIClient = (*httpWorkspaceAPIClient)(nil)

type httpWorkspaceAPIClient struct {
	baseURL string
	client  *http.Client
}

// NewWorkspaceAPIClient returns a WorkspaceAPIClient that calls the forge-workspace REST API.
func NewWorkspaceAPIClient(baseURL string) WorkspaceAPIClient {
	return &httpWorkspaceAPIClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// wsapiError represents a non-success HTTP response from the workspace API.
type wsapiError struct {
	statusCode int
	body       string
}

func (e *wsapiError) Error() string {
	return fmt.Sprintf("workspace api returned status %d: %s", e.statusCode, e.body)
}

func (c *httpWorkspaceAPIClient) ListWorkspaces(namespace string) ([]types.ManagedWorkspaceSummary, error) {
	u := c.baseURL + "/api/v1/workspaces?namespace=" + url.QueryEscape(namespace)

	var out []types.ManagedWorkspaceSummary
	if err := c.doJSON(http.MethodGet, u, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *httpWorkspaceAPIClient) CreateWorkspace(namespace string, req types.CreateManagedWorkspaceRequest) (*types.ManagedWorkspaceDetail, error) {
	u := c.baseURL + "/api/v1/workspaces?namespace=" + url.QueryEscape(namespace)

	var out types.ManagedWorkspaceDetail
	if err := c.doJSON(http.MethodPost, u, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpWorkspaceAPIClient) GetWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	u := c.baseURL + "/api/v1/workspaces/" + url.PathEscape(name) + "?namespace=" + url.QueryEscape(namespace)

	var out types.ManagedWorkspaceDetail
	if err := c.doJSON(http.MethodGet, u, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpWorkspaceAPIClient) DeleteWorkspace(namespace, name string) error {
	u := c.baseURL + "/api/v1/workspaces/" + url.PathEscape(name) + "?namespace=" + url.QueryEscape(namespace)
	return c.doJSON(http.MethodDelete, u, nil, nil)
}

func (c *httpWorkspaceAPIClient) SuspendWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	u := c.baseURL + "/api/v1/workspaces/" + url.PathEscape(name) + "/suspend?namespace=" + url.QueryEscape(namespace)

	var out types.ManagedWorkspaceDetail
	if err := c.doJSON(http.MethodPut, u, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpWorkspaceAPIClient) ResumeWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	u := c.baseURL + "/api/v1/workspaces/" + url.PathEscape(name) + "/resume?namespace=" + url.QueryEscape(namespace)

	var out types.ManagedWorkspaceDetail
	if err := c.doJSON(http.MethodPut, u, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// doJSON makes an HTTP request, checks the status code, and optionally decodes the JSON response.
func (c *httpWorkspaceAPIClient) doJSON(method, rawURL string, body any, dst any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, rawURL, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("workspace api request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return &wsapiError{
			statusCode: resp.StatusCode,
			body:       string(respBody),
		}
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
