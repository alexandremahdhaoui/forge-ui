package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge-ui/internal/handler"
)

func main() {
	// Flags
	port := flag.Int("port", 8080, "HTTP server port")
	workspaces := flag.String("workspaces", "", "base directory containing workspaces (default: $WORKSPACES or $HOME/workspaces)")
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
	// Look for templates/ relative to the current working directory first,
	// then fall back to the binary's directory.
	templateDir := resolveTemplateDir()

	// Create handler
	h, err := handler.New(baseDir, templateDir)
	if err != nil {
		log.Fatalf("failed to initialize handlers: %v", err)
	}

	// Register routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.HandleRedirect)
	mux.HandleFunc("GET /workspaces", h.HandleWorkspaces)
	mux.HandleFunc("GET /workspaces/{name}", h.HandleWorkspace)
	mux.HandleFunc("GET /workspaces/{ws}/repos/{repo}", h.HandleForge)
	mux.HandleFunc("GET /theme/toggle", h.HandleToggleTheme)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("forge-ui listening on http://localhost%s", addr)
	log.Printf("workspaces directory: %s", baseDir)
	log.Fatal(http.ListenAndServe(addr, mux))
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
