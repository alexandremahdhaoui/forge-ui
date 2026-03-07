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

import "github.com/alexandremahdhaoui/forge-ui/internal/types"

// DataSource provides page data for rendering.
type DataSource interface {
	ListPortfolios(sort string) (types.PortfoliosPageData, error)
	GetPortfolio(name, sort string) (types.PortfolioPageData, error)
	GetWorkspace(portfolio, workspace, sort string) (types.WorkspacePageData, error)
	GetForge(portfolio, workspace, repo string) (types.ForgePageData, error)
}

// MetaPlanLoader loads meta-plan YAML files from a workspace directory.
type MetaPlanLoader interface {
	LoadAll(wsPath string) ([]types.MetaPlan, error)
	Load(path string) (types.MetaPlan, error)
}

// RepoPlanLoader loads repo plan task files from a repository directory.
type RepoPlanLoader interface {
	LoadAll(repoPath string) ([]types.RepoPlan, error)
	LoadSummary(repoPath, repoName string) (types.RepoPlanSummary, error)
}

// WsConfigLoader loads workspace configuration from a workspace directory.
type WsConfigLoader interface {
	Load(wsPath string) (types.WsConfig, error)
}

// PortfolioConfigLoader loads portfolio configuration from a portfolio directory.
type PortfolioConfigLoader interface {
	Load(portfolioPath string) (types.PortfolioConfig, error)
}

// WorkspaceAPIClient calls the forge-workspace REST API for workspace CRUD.
type WorkspaceAPIClient interface {
	ListWorkspaces(namespace string) ([]types.ManagedWorkspaceSummary, error)
	CreateWorkspace(namespace string, req types.CreateManagedWorkspaceRequest) (*types.ManagedWorkspaceDetail, error)
	GetWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error)
	DeleteWorkspace(namespace, name string) error
	SuspendWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error)
	ResumeWorkspace(namespace, name string) (*types.ManagedWorkspaceDetail, error)
}

// CUAdapter reads CU composition state from the local filesystem.
type CUAdapter interface {
	LoadCompo(cuRepoPath string) (types.CompoState, error)
}
