package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/wssproxy"
)

func main() {
	listen := flag.String("listen", ":8080", "listen address")
	serveDir := flag.String("serve-dir", "/web", "directory with terminal web assets")
	wsSvcPattern := flag.String("workspace-service-pattern", "forge-workspace-{name}:22", "backend service DNS pattern")
	configFile := flag.String("config", "", "path to JSON config file (overrides flags)")
	flag.Parse()

	// If config file provided, read and apply overrides.
	if *configFile != "" {
		data, err := os.ReadFile(*configFile)
		if err != nil {
			log.Fatalf("failed to read config: %v", err)
		}
		var fileCfg struct {
			ListenAddr              string `json:"listenAddr"`
			ServeDir                string `json:"serveDir"`
			WorkspaceServicePattern string `json:"workspaceServicePattern"`
		}
		if err := json.Unmarshal(data, &fileCfg); err != nil {
			log.Fatalf("failed to parse config: %v", err)
		}
		if fileCfg.ListenAddr != "" {
			*listen = fileCfg.ListenAddr
		}
		if fileCfg.ServeDir != "" {
			*serveDir = fileCfg.ServeDir
		}
		if fileCfg.WorkspaceServicePattern != "" {
			*wsSvcPattern = fileCfg.WorkspaceServicePattern
		}
	}

	cfg := wssproxy.Config{
		WorkspaceSvcPattern: *wsSvcPattern,
	}

	srv := wssproxy.NewServer(cfg, *serveDir, *listen)

	go func() {
		log.Printf("forge-wss-proxy listening on http://localhost%s", *listen)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
