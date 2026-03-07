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
	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// WorkspaceMgmtService provides workspace lifecycle operations via the
// forge-workspace REST API.
type WorkspaceMgmtService interface {
	ListManagedWorkspaces(namespace string) ([]types.ManagedWorkspaceSummary, error)
	CreateManagedWorkspace(namespace string, req types.CreateManagedWorkspaceRequest) (*types.ManagedWorkspaceDetail, error)
	GetManagedWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error)
	DeleteManagedWorkspace(namespace, name string) error
	SuspendManagedWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error)
	ResumeManagedWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error)
}

// Compile-time check.
var _ WorkspaceMgmtService = (*workspaceMgmtService)(nil)

type workspaceMgmtService struct {
	client adapter.WorkspaceAPIClient
}

// NewWorkspaceMgmtService creates a WorkspaceMgmtService.
func NewWorkspaceMgmtService(client adapter.WorkspaceAPIClient) WorkspaceMgmtService {
	return &workspaceMgmtService{client: client}
}

func (s *workspaceMgmtService) ListManagedWorkspaces(namespace string) ([]types.ManagedWorkspaceSummary, error) {
	return s.client.ListWorkspaces(namespace)
}

func (s *workspaceMgmtService) CreateManagedWorkspace(namespace string, req types.CreateManagedWorkspaceRequest) (*types.ManagedWorkspaceDetail, error) {
	return s.client.CreateWorkspace(namespace, req)
}

func (s *workspaceMgmtService) GetManagedWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	return s.client.GetWorkspace(namespace, name)
}

func (s *workspaceMgmtService) DeleteManagedWorkspace(namespace, name string) error {
	return s.client.DeleteWorkspace(namespace, name)
}

func (s *workspaceMgmtService) SuspendManagedWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	return s.client.SuspendWorkspace(namespace, name)
}

func (s *workspaceMgmtService) ResumeManagedWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error) {
	return s.client.ResumeWorkspace(namespace, name)
}
