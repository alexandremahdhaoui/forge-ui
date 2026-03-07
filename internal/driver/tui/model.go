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

package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// view represents which screen the TUI is showing.
type view int

const (
	viewPortfolioList view = iota
	viewWorkspaceList
	viewRepoDetail
	viewForgeResults
)

// Model is the bubbletea model for forge-ui-tui.
type Model struct {
	// Services
	baseDir      string
	portfolioSvc controller.PortfolioService
	workspaceSvc controller.WorkspaceService
	forgeSvc     controller.ForgeService
	wsMgmtSvc    controller.WorkspaceMgmtService
	cuSvc        controller.CUService

	// UI state
	currentView view
	cursor      int
	err         error
	width       int
	height      int
	statusMsg   string

	// Data
	portfolios        []types.PortfolioSummary
	selectedPortfolio string
	workspaceData     types.PortfolioPageData
	managedWs     []types.ManagedWorkspaceSummary
	repoData      types.WorkspacePageData
	forgeData     types.ForgePageData
	compoState        types.CompoState
}

// NewModel creates a new TUI Model.
func NewModel(
	baseDir string,
	ps controller.PortfolioService,
	ws controller.WorkspaceService,
	fs controller.ForgeService,
	wms controller.WorkspaceMgmtService,
	cus controller.CUService,
) Model {
	return Model{
		baseDir:      baseDir,
		portfolioSvc: ps,
		workspaceSvc: ws,
		forgeSvc:     fs,
		wsMgmtSvc:    wms,
		cuSvc:        cus,
	}
}

// Messages
type portfoliosLoaded struct {
	data types.PortfoliosPageData
	err  error
}

type portfolioLoaded struct {
	data types.PortfolioPageData
	err  error
}

type managedWsLoaded struct {
	data []types.ManagedWorkspaceSummary
	err  error
}

type workspaceLoaded struct {
	data types.WorkspacePageData
	err  error
}

type forgeLoaded struct {
	data types.ForgePageData
	err  error
}

type compoLoaded struct {
	data types.CompoState
	err  error
}

type crudResult struct {
	msg string
	err error
}

// Init loads the initial portfolio list.
func (m Model) Init() tea.Cmd {
	return m.loadPortfolios()
}

func (m Model) loadPortfolios() tea.Cmd {
	return func() tea.Msg {
		data, err := m.portfolioSvc.ListPortfolios(m.baseDir, "name")
		return portfoliosLoaded{data: data, err: err}
	}
}

func (m Model) loadPortfolio(name string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.portfolioSvc.GetPortfolio(m.baseDir, name, "name")
		return portfolioLoaded{data: data, err: err}
	}
}

func (m Model) loadManagedWorkspaces() tea.Cmd {
	return func() tea.Msg {
		data, err := m.wsMgmtSvc.ListManagedWorkspaces("default")
		return managedWsLoaded{data: data, err: err}
	}
}

func (m Model) loadWorkspace(portfolio, workspace string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.workspaceSvc.GetWorkspace(m.baseDir, portfolio, workspace, "name")
		return workspaceLoaded{data: data, err: err}
	}
}

func (m Model) loadForge(portfolio, workspace, repo string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.forgeSvc.GetForge(m.baseDir, portfolio, workspace, repo)
		return forgeLoaded{data: data, err: err}
	}
}

func (m Model) loadCompo(portfolio, workspace string) tea.Cmd {
	return func() tea.Msg {
		wsPath := m.baseDir + "/" + portfolio + "/" + workspace
		if portfolio == "default" {
			wsPath = m.baseDir + "/" + workspace
		}
		data, err := m.cuSvc.GetCompoState(wsPath)
		return compoLoaded{data: data, err: err}
	}
}

// Update handles messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case portfoliosLoaded:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.portfolios = msg.data.Portfolios
		m.currentView = viewPortfolioList
		return m, nil

	case portfolioLoaded:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.workspaceData = msg.data
		m.currentView = viewWorkspaceList
		m.cursor = 0
		return m, m.loadManagedWorkspaces()

	case managedWsLoaded:
		if msg.err != nil {
			// Non-fatal: managed ws API might be unavailable
			m.managedWs = nil
			m.statusMsg = fmt.Sprintf("managed workspaces unavailable: %v", msg.err)
		} else {
			m.managedWs = msg.data
		}
		return m, nil

	case workspaceLoaded:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.repoData = msg.data
		m.currentView = viewRepoDetail
		m.cursor = 0
		return m, nil

	case forgeLoaded:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.forgeData = msg.data
		m.currentView = viewForgeResults
		return m, nil

	case compoLoaded:
		if msg.err == nil {
			m.compoState = msg.data
		}
		return m, nil

	case crudResult:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMsg = msg.msg
		}
		// Refresh managed workspaces after CRUD
		return m, m.loadManagedWorkspaces()

	case tea.KeyMsg:
		// Clear status on any key press
		m.statusMsg = ""

		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentView == viewPortfolioList {
				return m, tea.Quit
			}
			// "q" goes back from non-root views
			return m.goBack()

		case "esc", "backspace":
			return m.goBack()

		case "j", "down":
			m.cursor++
			m.clampCursor()
			return m, nil

		case "k", "up":
			m.cursor--
			m.clampCursor()
			return m, nil

		case "enter":
			return m.handleSelect()

		case "s":
			if m.currentView == viewWorkspaceList {
				return m.handleSuspend()
			}

		case "r":
			if m.currentView == viewWorkspaceList {
				return m.handleResume()
			}

		case "d":
			if m.currentView == viewWorkspaceList {
				return m.handleDelete()
			}
		}
	}

	return m, nil
}

func (m Model) goBack() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case viewWorkspaceList:
		m.currentView = viewPortfolioList
		m.cursor = 0
		return m, nil
	case viewRepoDetail:
		m.currentView = viewWorkspaceList
		m.cursor = 0
		return m, nil
	case viewForgeResults:
		m.currentView = viewRepoDetail
		return m, nil
	}
	return m, nil
}

func (m Model) handleSelect() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case viewPortfolioList:
		if m.cursor < len(m.portfolios) {
			p := m.portfolios[m.cursor]
			m.selectedPortfolio = p.Name
			return m, m.loadPortfolio(p.Name)
		}
	case viewWorkspaceList:
		if m.cursor < len(m.workspaceData.Workspaces) {
			ws := m.workspaceData.Workspaces[m.cursor]
			return m, tea.Batch(
				m.loadWorkspace(m.selectedPortfolio, ws.Name),
				m.loadCompo(m.selectedPortfolio, ws.Name),
			)
		}
	case viewRepoDetail:
		if m.cursor < len(m.repoData.Repos) {
			repo := m.repoData.Repos[m.cursor]
			return m, m.loadForge(m.selectedPortfolio, m.repoData.Name, repo.Name)
		}
	}
	return m, nil
}

func (m Model) handleSuspend() (tea.Model, tea.Cmd) {
	if len(m.managedWs) == 0 {
		return m, nil
	}
	// Find managed workspace matching current cursor in local workspace list
	if m.cursor >= len(m.workspaceData.Workspaces) {
		return m, nil
	}
	wsName := m.workspaceData.Workspaces[m.cursor].Name
	return m, func() tea.Msg {
		_, err := m.wsMgmtSvc.SuspendManagedWorkspace("default", wsName)
		if err != nil {
			return crudResult{err: err}
		}
		return crudResult{msg: fmt.Sprintf("Suspended %s", wsName)}
	}
}

func (m Model) handleResume() (tea.Model, tea.Cmd) {
	if len(m.managedWs) == 0 {
		return m, nil
	}
	if m.cursor >= len(m.workspaceData.Workspaces) {
		return m, nil
	}
	wsName := m.workspaceData.Workspaces[m.cursor].Name
	return m, func() tea.Msg {
		_, err := m.wsMgmtSvc.ResumeManagedWorkspace("default", wsName)
		if err != nil {
			return crudResult{err: err}
		}
		return crudResult{msg: fmt.Sprintf("Resumed %s", wsName)}
	}
}

func (m Model) handleDelete() (tea.Model, tea.Cmd) {
	if len(m.managedWs) == 0 {
		return m, nil
	}
	if m.cursor >= len(m.workspaceData.Workspaces) {
		return m, nil
	}
	wsName := m.workspaceData.Workspaces[m.cursor].Name
	return m, func() tea.Msg {
		err := m.wsMgmtSvc.DeleteManagedWorkspace("default", wsName)
		if err != nil {
			return crudResult{err: err}
		}
		return crudResult{msg: fmt.Sprintf("Deleted %s", wsName)}
	}
}

func (m *Model) clampCursor() {
	max := 0
	switch m.currentView {
	case viewPortfolioList:
		max = len(m.portfolios)
	case viewWorkspaceList:
		max = len(m.workspaceData.Workspaces)
	case viewRepoDetail:
		max = len(m.repoData.Repos)
	case viewForgeResults:
		max = 0
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if max > 0 && m.cursor >= max {
		m.cursor = max - 1
	}
}

// View renders the current view.
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err)
	}

	header := headerStyle.Render("forge-ui                              [q]uit [?]help")

	var content string
	switch m.currentView {
	case viewPortfolioList:
		content = renderPortfolioList(m)
	case viewWorkspaceList:
		content = renderWorkspaceList(m)
	case viewRepoDetail:
		content = renderRepoDetail(m)
	case viewForgeResults:
		content = renderForgeResults(m)
	default:
		content = "Loading..."
	}

	status := ""
	if m.statusMsg != "" {
		status = "\n " + helpStyle.Render(m.statusMsg)
	}

	return header + "\n" + content + status + "\n"
}
