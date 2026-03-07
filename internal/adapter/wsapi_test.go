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

package adapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWorkspaces(t *testing.T) {
	t.Parallel()

	want := []types.ManagedWorkspaceSummary{
		{Name: "ws-1", Namespace: "default", Phase: "Running", Suspended: false, Image: "ubuntu:22.04"},
		{Name: "ws-2", Namespace: "default", Phase: "Pending", Suspended: true, Image: "debian:12"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/workspaces", r.URL.Path)
		assert.Equal(t, "default", r.URL.Query().Get("namespace"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	got, err := client.ListWorkspaces("default")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListWorkspaces_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	_, err := client.ListWorkspaces("default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCreateWorkspace(t *testing.T) {
	t.Parallel()

	want := &types.ManagedWorkspaceDetail{
		Name:      "new-ws",
		Namespace: "default",
		Image:     "ubuntu:22.04",
		Phase:     "Pending",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/workspaces", r.URL.Path)
		assert.Equal(t, "default", r.URL.Query().Get("namespace"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req types.CreateManagedWorkspaceRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "new-ws", req.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	got, err := client.CreateWorkspace("default", types.CreateManagedWorkspaceRequest{
		Name:         "new-ws",
		Image:        "ubuntu:22.04",
		StorageClass: "standard",
		StorageSize:  "10Gi",
	})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetWorkspace(t *testing.T) {
	t.Parallel()

	want := &types.ManagedWorkspaceDetail{
		Name:      "my-ws",
		Namespace: "default",
		Image:     "ubuntu:22.04",
		Phase:     "Running",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/workspaces/my-ws", r.URL.Path)
		assert.Equal(t, "default", r.URL.Query().Get("namespace"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	got, err := client.GetWorkspace("default", "my-ws")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetWorkspace_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	_, err := client.GetWorkspace("default", "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDeleteWorkspace(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/workspaces/my-ws", r.URL.Path)
		assert.Equal(t, "default", r.URL.Query().Get("namespace"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	err := client.DeleteWorkspace("default", "my-ws")
	require.NoError(t, err)
}

func TestDeleteWorkspace_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	err := client.DeleteWorkspace("default", "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestSuspendWorkspace(t *testing.T) {
	t.Parallel()

	want := &types.ManagedWorkspaceDetail{
		Name:      "my-ws",
		Namespace: "default",
		Image:     "ubuntu:22.04",
		Phase:     "Suspended",
		Suspended: true,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/api/v1/workspaces/my-ws/suspend", r.URL.Path)
		assert.Equal(t, "default", r.URL.Query().Get("namespace"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	got, err := client.SuspendWorkspace("default", "my-ws")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestResumeWorkspace(t *testing.T) {
	t.Parallel()

	want := &types.ManagedWorkspaceDetail{
		Name:      "my-ws",
		Namespace: "default",
		Image:     "ubuntu:22.04",
		Phase:     "Running",
		Suspended: false,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/api/v1/workspaces/my-ws/resume", r.URL.Path)
		assert.Equal(t, "default", r.URL.Query().Get("namespace"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewWorkspaceAPIClient(srv.URL)
	got, err := client.ResumeWorkspace("default", "my-ws")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
