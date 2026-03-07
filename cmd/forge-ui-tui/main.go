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

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
	"github.com/alexandremahdhaoui/forge-ui/internal/driver/tui"
)

func main() {
	// Flags
	workspaces := flag.String("workspaces", "", "base directory containing workspaces (default: $WORKSPACES or $HOME/workspaces)")
	workspaceAPIURL := flag.String("workspace-api-url", "", "forge-workspace REST API URL (default: http://localhost:8080, env: FORGE_WORKSPACE_API_URL)")
	_ = flag.Duration("refresh-interval", 30*time.Second, "polling interval (reserved for future use)")
	flag.Parse()

	// Resolve workspace API URL.
	apiURL := *workspaceAPIURL
	if apiURL == "" {
		apiURL = os.Getenv("FORGE_WORKSPACE_API_URL")
	}
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	// Resolve workspaces directory.
	baseDir := *workspaces
	if baseDir == "" {
		baseDir = os.Getenv("WORKSPACES")
	}
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home directory: %v", err)
		}
		baseDir = filepath.Join(home, "workspaces")
	}

	// Verify base directory exists.
	if _, err := os.Stat(baseDir); err != nil {
		log.Fatalf("workspaces directory does not exist: %s", baseDir)
	}

	// Create adapters.
	c := adapter.NewCache()
	ws := adapter.NewWorkspaceDiscovery()
	pd := adapter.NewPortfolioDiscovery(ws)
	fl := adapter.NewForgeLoader()
	wc := adapter.NewWsConfigLoader()
	mp := adapter.NewMetaPlanLoader()
	rp := adapter.NewRepoPlanLoader()
	pc := adapter.NewPortfolioConfigLoader()
	wsAPI := adapter.NewWorkspaceAPIClient(apiURL)
	cuAdapt := adapter.NewCUAdapter()

	// Create controller services.
	ps := controller.NewPortfolioService(pd, c, fl, wc, mp, pc)
	wsSvc := controller.NewWorkspaceService(ws, c, fl, wc, mp, rp)
	fsSvc := controller.NewForgeService(fl, rp)
	wsMgmtSvc := controller.NewWorkspaceMgmtService(wsAPI)
	cuSvc := controller.NewCUService(cuAdapt)

	// Create TUI model and run.
	model := tui.NewModel(baseDir, ps, wsSvc, fsSvc, wsMgmtSvc, cuSvc)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}
