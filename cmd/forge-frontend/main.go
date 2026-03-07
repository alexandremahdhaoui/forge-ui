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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
	restdriver "github.com/alexandremahdhaoui/forge-ui/internal/driver/rest"
)

func main() {
	// Flags
	port := flag.Int("port", 8081, "HTTP server port")
	workspaces := flag.String("workspaces", "", "base directory containing workspaces (default: $WORKSPACES or $HOME/workspaces)")
	refreshInterval := flag.Duration("refresh-interval", 1*time.Minute, "background git refresh interval")
	refreshWorkers := flag.Int("refresh-workers", 3, "number of background git refresh workers")
	workspaceAPIURL := flag.String("workspace-api-url", "", "forge-workspace REST API URL (default: http://localhost:8080, env: FORGE_WORKSPACE_API_URL)")
	flag.Parse()

	// Resolve workspace API URL.
	apiURL := *workspaceAPIURL
	if apiURL == "" {
		apiURL = os.Getenv("FORGE_WORKSPACE_API_URL")
	}
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	// Resolve workspaces directory
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

	// Verify base directory exists
	if _, err := os.Stat(baseDir); err != nil {
		log.Fatalf("workspaces directory does not exist: %s", baseDir)
	}

	// Create adapters.
	c := adapter.NewCache()
	gi := adapter.NewGitInfo()
	ws := adapter.NewWorkspaceDiscovery()
	pd := adapter.NewPortfolioDiscovery(ws)
	fl := adapter.NewForgeLoader()
	wc := adapter.NewWsConfigLoader()
	mp := adapter.NewMetaPlanLoader()
	rp := adapter.NewRepoPlanLoader()
	pc := adapter.NewPortfolioConfigLoader()

	// Start background refresher.
	r := controller.NewRefresher(c, gi, pd, ws, controller.RefresherConfig{
		BaseDir:    baseDir,
		Interval:   *refreshInterval,
		NumWorkers: *refreshWorkers,
	})
	r.Start() // blocks until initial refresh completes

	// Create controller services.
	ps := controller.NewPortfolioService(pd, c, fl, wc, mp, pc)
	wsSvc := controller.NewWorkspaceService(ws, c, fl, wc, mp, rp)
	fsSvc := controller.NewForgeService(fl, rp)

	// Create workspace management and CU services.
	wsAPI := adapter.NewWorkspaceAPIClient(apiURL)
	cuAdapt := adapter.NewCUAdapter()
	wsMgmtSvc := controller.NewWorkspaceMgmtService(wsAPI)
	cuSvc := controller.NewCUService(cuAdapt)

	// Create REST API handler and register routes.
	h := restdriver.NewAPIHandler(baseDir, ps, wsSvc, fsSvc, wsMgmtSvc, cuSvc)
	mux := http.NewServeMux()
	si := restdriver.NewStrictHandler(h, nil)
	restdriver.HandlerFromMux(si, mux)

	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{Addr: addr, Handler: corsMiddleware(mux)}

	// Start server in a goroutine.
	go func() {
		log.Printf("forge-frontend listening on http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	log.Printf("workspaces directory: %s", baseDir)

	// Wait for interrupt signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	r.Stop()
	_ = srv.Close()
}

// corsMiddleware wraps an http.Handler with permissive CORS headers.
// The WASM frontend (served on :8080) calls this backend (:8081) cross-origin.
// Allow-Origin "*" is acceptable because the API is read-only.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
