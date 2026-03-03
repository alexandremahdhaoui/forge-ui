package wssproxy

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Config holds the WSS proxy configuration.
type Config struct {
	// WorkspaceSvcPattern is the backend TCP address pattern.
	// The placeholder {name} is replaced with the workspace name
	// extracted from the URL path.
	// Example: "forge-workspace-{name}:22"
	WorkspaceSvcPattern string
}

// NewHandler returns an HTTP handler that upgrades WebSocket connections at
// /ws/{workspace} and bridges them to a TCP backend resolved from the
// workspace service pattern.
func NewHandler(cfg Config) http.Handler {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspace := strings.TrimPrefix(r.URL.Path, "/ws/")
		if workspace == "" || workspace == r.URL.Path {
			http.Error(w, "missing workspace name", http.StatusBadRequest)
			return
		}

		backendAddr := strings.Replace(cfg.WorkspaceSvcPattern, "{name}", workspace, 1)
		slog.Info("ws connect", "workspace", workspace, "backend", backendAddr, "remote", r.RemoteAddr)

		tcpConn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
		if err != nil {
			slog.Error("tcp dial failed", "backend", backendAddr, "error", err)
			http.Error(w, "workspace unavailable", http.StatusBadGateway)
			return
		}

		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			_ = tcpConn.Close()
			return
		}

		slog.Info("bridge established", "workspace", workspace, "backend", backendAddr)
		bridge(wsConn, tcpConn)
	})
}

// bridge copies data bidirectionally between a WebSocket connection and a TCP
// connection. When either side closes or errors, both sides are closed.
func bridge(wsConn *websocket.Conn, tcpConn net.Conn) {
	done := make(chan struct{}, 2)

	// TCP -> WebSocket
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			n, err := tcpConn.Read(buf)
			if n > 0 {
				if werr := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					slog.Debug("ws write error", "error", werr)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					slog.Debug("tcp read error", "error", err)
				}
				return
			}
		}
	}()

	// WebSocket -> TCP
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			_, msg, err := wsConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					slog.Debug("ws read error", "error", err)
				}
				return
			}
			if _, werr := tcpConn.Write(msg); werr != nil {
				slog.Debug("tcp write error", "error", werr)
				return
			}
		}
	}()

	// Wait for either goroutine to finish, then close both sides.
	<-done
	_ = wsConn.Close()
	_ = tcpConn.Close()
	// Wait for the other goroutine to finish.
	<-done
	slog.Info("bridge closed", "remote", wsConn.RemoteAddr())
}

// NewServer creates an HTTP server that serves static files and routes
// WebSocket connections to workspace backends.
func NewServer(cfg Config, serveDir string, listenAddr string) *http.Server {
	mux := http.NewServeMux()
	ks := newKeyStore()

	// Health endpoints.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Authorized keys endpoint for workspace sshd AuthorizedKeysCommand.
	// GET /internal/authorized-keys/{workspace}
	mux.HandleFunc("/internal/authorized-keys/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		workspace := strings.TrimPrefix(r.URL.Path, "/internal/authorized-keys/")
		if workspace == "" {
			http.Error(w, "missing workspace name", http.StatusBadRequest)
			return
		}
		keys := ks.List(workspace)
		for _, k := range keys {
			_, _ = io.WriteString(w, k+"\n")
		}
	})

	// WebSocket proxy and key registration under /ws/ prefix.
	// POST /ws/{workspace}/register-key is handled inline; all other
	// requests are forwarded to the WebSocket handler.
	wsHandler := NewHandler(cfg)
	mux.HandleFunc("/ws/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/ws/")
		if strings.HasSuffix(path, "/register-key") {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			workspace := strings.TrimSuffix(path, "/register-key")
			if workspace == "" {
				http.Error(w, "missing workspace name", http.StatusBadRequest)
				return
			}
			body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
			if err != nil {
				http.Error(w, "failed to read body", http.StatusInternalServerError)
				return
			}
			publicKey := strings.TrimSpace(string(body))
			if publicKey == "" {
				http.Error(w, "empty public key", http.StatusBadRequest)
				return
			}
			ks.Add(workspace, publicKey)
			w.WriteHeader(http.StatusOK)
			return
		}
		wsHandler.ServeHTTP(w, r)
	})

	// Static file server.
	mux.Handle("/", http.FileServer(http.Dir(serveDir)))

	return &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}
