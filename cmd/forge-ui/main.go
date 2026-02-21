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
	httpdriver "github.com/alexandremahdhaoui/forge-ui/internal/driver/http"
	"github.com/alexandremahdhaoui/forge-ui/internal/refresher"
)

func main() {
	// Flags
	port := flag.Int("port", 8080, "HTTP server port")
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

	// Resolve templates directory.
	templateDir := resolveTemplateDir()

	// Create adapters.
	c := adapter.NewCache()
	gi := adapter.NewGitInfo()
	ws := adapter.NewWorkspaceDiscovery()
	pd := adapter.NewPortfolioDiscovery(ws)
	fl := adapter.NewForgeLoader()

	// Start background refresher.
	r := refresher.New(c, gi, pd, ws, refresher.Config{
		BaseDir:    baseDir,
		Interval:   *refreshInterval,
		NumWorkers: *refreshWorkers,
	})
	r.Start() // blocks until initial refresh completes

	// Create controller services.
	ps := controller.NewPortfolioService(pd, c, fl)
	wsSvc := controller.NewWorkspaceService(ws, c, fl)
	fsSvc := controller.NewForgeService(fl)

	// Create handler
	h, err := httpdriver.New(baseDir, templateDir, ps, wsSvc, fsSvc)
	if err != nil {
		log.Fatalf("failed to initialize handlers: %v", err)
	}

	// Register routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.HandleRedirect)
	mux.HandleFunc("GET /portfolios", h.HandlePortfolios)
	mux.HandleFunc("GET /portfolios/{name}", h.HandlePortfolio)
	mux.HandleFunc("GET /portfolios/{p}/workspaces/{w}", h.HandleWorkspace)
	mux.HandleFunc("GET /portfolios/{p}/workspaces/{w}/repos/{r}", h.HandleForge)
	mux.HandleFunc("GET /theme/toggle", h.HandleToggleTheme)

	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{Addr: addr, Handler: mux}

	// Start server in a goroutine.
	go func() {
		log.Printf("forge-ui listening on http://localhost%s", addr)
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

// resolveTemplateDir finds the templates directory.
// Priority: ./templates, then <binary-dir>/templates.
func resolveTemplateDir() string {
	// Try current working directory
	if info, err := os.Stat("templates"); err == nil && info.IsDir() {
		abs, _ := filepath.Abs("templates")
		return abs
	}

	// Try relative to binary
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "templates")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	// Try /templates (container path)
	if info, err := os.Stat("/templates"); err == nil && info.IsDir() {
		return "/templates"
	}

	// Fallback
	log.Fatal("cannot find templates/ directory. Run from the project root or place templates/ next to the binary.")
	return ""
}
