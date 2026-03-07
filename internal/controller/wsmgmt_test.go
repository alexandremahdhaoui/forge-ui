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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWorkspaceAPIClient is a manual mock for adapter.WorkspaceAPIClient.
type mockWorkspaceAPIClient struct {
	listFn    func(namespace string) ([]types.ManagedWorkspaceSummary, error)
	createFn  func(namespace string, req types.CreateManagedWorkspaceRequest) (*types.ManagedWorkspaceDetail, error)
	getFn     func(namespace, name string) (*types.ManagedWorkspaceDetail, error)
	deleteFn  func(namespace, name string) error
	suspendFn func(namespace, name string) (*types.ManagedWorkspaceDetail, error)
	resumeFn  func(namespace, name string) (*types.ManagedWorkspaceDetail, error)
}

func (m *mockWorkspaceAPIClient) ListWorkspaces(namespace string) ([]types.ManagedWorkspaceSummary, error) {
	return m.listFn(namespace)
}

func (m *mockWorkspaceAPIClient) CreateWorkspace(namespace string, req types.CreateManagedWorkspaceRequest) (*types.ManagedWorkspaceDetail, error) {
	return m.createFn(namespace, req)
}

func (m *mockWorkspaceAPIClient) GetWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	return m.getFn(namespace, name)
}

func (m *mockWorkspaceAPIClient) DeleteWorkspace(namespace, name string) error {
	return m.deleteFn(namespace, name)
}

func (m *mockWorkspaceAPIClient) SuspendWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	return m.suspendFn(namespace, name)
}

func (m *mockWorkspaceAPIClient) ResumeWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	return m.resumeFn(namespace, name)
}

func TestListManagedWorkspaces(t *testing.T) {
	t.Parallel()

	want := []types.ManagedWorkspaceSummary{
		{Name: "ws-1", Namespace: "default", Phase: "Running"},
		{Name: "ws-2", Namespace: "default", Phase: "Suspended"},
	}

	client := &mockWorkspaceAPIClient{
		listFn: func(namespace string) ([]types.ManagedWorkspaceSummary, error) {
			assert.Equal(t, "default", namespace)
			return want, nil
		},
	}

	svc := NewWorkspaceMgmtService(client)
	got, err := svc.ListManagedWorkspaces("default")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListManagedWorkspaces_Error(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("connection refused")

	client := &mockWorkspaceAPIClient{
		listFn: func(namespace string) ([]types.ManagedWorkspaceSummary, error) {
			return nil, wantErr
		},
	}

	svc := NewWorkspaceMgmtService(client)
	got, err := svc.ListManagedWorkspaces("default")

	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, got)
}

func TestCreateManagedWorkspace(t *testing.T) {
	t.Parallel()

	req := types.CreateManagedWorkspaceRequest{
		Name:  "new-ws",
		Image: "ubuntu:latest",
	}
	want := &types.ManagedWorkspaceDetail{
		Name:      "new-ws",
		Namespace: "dev",
		Image:     "ubuntu:latest",
	}

	client := &mockWorkspaceAPIClient{
		createFn: func(namespace string, r types.CreateManagedWorkspaceRequest) (*types.ManagedWorkspaceDetail, error) {
			assert.Equal(t, "dev", namespace)
			assert.Equal(t, req, r)
			return want, nil
		},
	}

	svc := NewWorkspaceMgmtService(client)
	got, err := svc.CreateManagedWorkspace("dev", req)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetManagedWorkspace(t *testing.T) {
	t.Parallel()

	want := &types.ManagedWorkspaceDetail{
		Name:      "ws-1",
		Namespace: "default",
		Image:     "golang:1.22",
	}

	client := &mockWorkspaceAPIClient{
		getFn: func(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
			assert.Equal(t, "default", namespace)
			assert.Equal(t, "ws-1", name)
			return want, nil
		},
	}

	svc := NewWorkspaceMgmtService(client)
	got, err := svc.GetManagedWorkspace("default", "ws-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDeleteManagedWorkspace(t *testing.T) {
	t.Parallel()

	client := &mockWorkspaceAPIClient{
		deleteFn: func(namespace, name string) error {
			assert.Equal(t, "prod", namespace)
			assert.Equal(t, "ws-old", name)
			return nil
		},
	}

	svc := NewWorkspaceMgmtService(client)
	err := svc.DeleteManagedWorkspace("prod", "ws-old")

	require.NoError(t, err)
}

func TestDeleteManagedWorkspace_Error(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("not found")

	client := &mockWorkspaceAPIClient{
		deleteFn: func(namespace, name string) error {
			return wantErr
		},
	}

	svc := NewWorkspaceMgmtService(client)
	err := svc.DeleteManagedWorkspace("default", "nonexistent")

	require.ErrorIs(t, err, wantErr)
}

func TestSuspendManagedWorkspace(t *testing.T) {
	t.Parallel()

	want := &types.ManagedWorkspaceDetail{
		Name:      "ws-1",
		Namespace: "default",
	}

	client := &mockWorkspaceAPIClient{
		suspendFn: func(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
			assert.Equal(t, "default", namespace)
			assert.Equal(t, "ws-1", name)
			return want, nil
		},
	}

	svc := NewWorkspaceMgmtService(client)
	got, err := svc.SuspendManagedWorkspace("default", "ws-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestResumeManagedWorkspace(t *testing.T) {
	t.Parallel()

	want := &types.ManagedWorkspaceDetail{
		Name:      "ws-1",
		Namespace: "default",
	}

	client := &mockWorkspaceAPIClient{
		resumeFn: func(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
			assert.Equal(t, "default", namespace)
			assert.Equal(t, "ws-1", name)
			return want, nil
		},
	}

	svc := NewWorkspaceMgmtService(client)
	got, err := svc.ResumeManagedWorkspace("default", "ws-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}
