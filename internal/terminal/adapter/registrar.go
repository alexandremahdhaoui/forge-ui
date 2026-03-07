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

package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
)

type httpKeyRegistrar struct {
	baseURL string
}

// NewKeyRegistrar creates a KeyRegistrar that POSTs public keys to the
// wss-proxy registration endpoint. The baseURL is derived from the
// WebSocket endpoint URL (e.g., "ws://host/ws/default" -> "http://host").
func NewKeyRegistrar(wsEndpoint string) KeyRegistrar {
	u, err := url.Parse(wsEndpoint)
	if err != nil {
		return &httpKeyRegistrar{}
	}
	scheme := "http"
	if u.Scheme == "wss" || u.Scheme == "https" {
		scheme = "https"
	}
	return &httpKeyRegistrar{baseURL: scheme + "://" + u.Host}
}

type httpWorkspaceInfoClient struct {
	baseURL string
}

// NewWorkspaceInfoClient creates a WorkspaceInfoClient that fetches workspace
// info from the proxy. The wsEndpoint URL is converted to an HTTP base URL
// (same logic as NewKeyRegistrar).
func NewWorkspaceInfoClient(wsEndpoint string) WorkspaceInfoClient {
	u, err := url.Parse(wsEndpoint)
	if err != nil {
		return &httpWorkspaceInfoClient{}
	}
	scheme := "http"
	if u.Scheme == "wss" || u.Scheme == "https" {
		scheme = "https"
	}
	return &httpWorkspaceInfoClient{baseURL: scheme + "://" + u.Host}
}

func (c *httpWorkspaceInfoClient) GetInfo(workspace string) (types.WorkspaceInfo, error) {
	resp, err := http.Get(c.baseURL + "/ws/" + workspace + "/info")
	if err != nil {
		return types.WorkspaceInfo{}, fmt.Errorf("get workspace info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return types.WorkspaceInfo{}, fmt.Errorf("get workspace info: %s: %s", resp.Status, body)
	}
	var info types.WorkspaceInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return types.WorkspaceInfo{}, fmt.Errorf("decode workspace info: %w", err)
	}
	return info, nil
}

func (r *httpKeyRegistrar) RegisterKey(workspace, publicKey string) error {
	resp, err := http.Post(
		r.baseURL+"/ws/"+workspace+"/register-key",
		"text/plain",
		strings.NewReader(publicKey),
	)
	if err != nil {
		return fmt.Errorf("register key: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register key: %s: %s", resp.Status, body)
	}
	return nil
}
