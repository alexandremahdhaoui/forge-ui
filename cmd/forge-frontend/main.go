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
	flag.Parse()

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

	// Create REST API handler and register routes.
	h := restdriver.NewAPIHandler(baseDir, ps, wsSvc, fsSvc)
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
