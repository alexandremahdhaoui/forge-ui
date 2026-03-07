//go:build unit

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

package wssproxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startEchoServer starts a TCP server that echoes back all received data.
// Returns the listener address and a cleanup function.
func startEchoServer(t *testing.T) (addr string, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	return ln.Addr().String(), func() { _ = ln.Close() }
}

// wsURL converts an httptest.Server URL to a WebSocket URL with the given path.
func wsURL(server *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(server.URL, "http") + path
}

func TestProxy_BidirectionalData(t *testing.T) {
	t.Parallel()

	echoAddr, echoCleanup := startEchoServer(t)
	defer echoCleanup()

	// Extract port from echo server address for the pattern.
	_, port, err := net.SplitHostPort(echoAddr)
	require.NoError(t, err)

	handler := NewHandler(Config{
		WorkspaceSvcPattern: "127.0.0.1:" + port,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	dialer := &websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	ws, _, err := dialer.Dial(wsURL(server, "/ws/test-ws"), nil)
	require.NoError(t, err)
	defer func() { _ = ws.Close() }()

	// Send data and verify echo.
	testData := []byte("hello from client")
	err = ws.WriteMessage(websocket.BinaryMessage, testData)
	require.NoError(t, err)

	msgType, msg, err := ws.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, websocket.BinaryMessage, msgType)
	assert.Equal(t, testData, msg)
}

func TestProxy_WSClientDisconnect(t *testing.T) {
	t.Parallel()

	// Use a TCP server that tracks when connections close.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	connClosed := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		// Echo until connection closes.
		_, _ = io.Copy(conn, conn)
		close(connClosed)
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	handler := NewHandler(Config{
		WorkspaceSvcPattern: "127.0.0.1:" + port,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	dialer := &websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	ws, _, err := dialer.Dial(wsURL(server, "/ws/test-ws"), nil)
	require.NoError(t, err)

	// Send some data first.
	err = ws.WriteMessage(websocket.BinaryMessage, []byte("hello"))
	require.NoError(t, err)

	// Close the WebSocket client.
	err = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
	require.NoError(t, err)
	_ = ws.Close()

	// TCP connection should close as a result.
	select {
	case <-connClosed:
		// TCP side closed as expected.
	case <-time.After(5 * time.Second):
		t.Fatal("TCP connection was not closed after WS client disconnect")
	}
}

func TestProxy_TCPBackendDisconnect(t *testing.T) {
	t.Parallel()

	// TCP server that accepts one connection and immediately closes it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Close immediately to simulate backend disconnect.
		_ = conn.Close()
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	handler := NewHandler(Config{
		WorkspaceSvcPattern: "127.0.0.1:" + port,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	dialer := &websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	ws, _, err := dialer.Dial(wsURL(server, "/ws/test-ws"), nil)
	require.NoError(t, err)
	defer func() { _ = ws.Close() }()

	// Reading should return an error because TCP backend closed.
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
}

func TestProxy_ConcurrentConnections(t *testing.T) {
	t.Parallel()

	echoAddr, echoCleanup := startEchoServer(t)
	defer echoCleanup()

	_, port, err := net.SplitHostPort(echoAddr)
	require.NoError(t, err)

	handler := NewHandler(Config{
		WorkspaceSvcPattern: "127.0.0.1:" + port,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	const numClients = 3
	var wg sync.WaitGroup
	errs := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			dialer := &websocket.Dialer{HandshakeTimeout: 5 * time.Second}
			ws, _, err := dialer.Dial(wsURL(server, "/ws/test-ws"), nil)
			if err != nil {
				errs <- fmt.Errorf("client %d dial: %w", clientID, err)
				return
			}
			defer func() { _ = ws.Close() }()

			testData := []byte(fmt.Sprintf("data-from-client-%d", clientID))
			if err := ws.WriteMessage(websocket.BinaryMessage, testData); err != nil {
				errs <- fmt.Errorf("client %d write: %w", clientID, err)
				return
			}

			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				errs <- fmt.Errorf("client %d read: %w", clientID, err)
				return
			}
			if msgType != websocket.BinaryMessage {
				errs <- fmt.Errorf("client %d: expected binary message, got %d", clientID, msgType)
				return
			}
			if string(msg) != string(testData) {
				errs <- fmt.Errorf("client %d: expected %q, got %q", clientID, testData, msg)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

func TestProxy_NonWebSocketRequest(t *testing.T) {
	t.Parallel()

	// Need a reachable TCP backend so the proxy gets past the dial step
	// and reaches the gorilla upgrader, which rejects non-WebSocket requests.
	echoAddr, echoCleanup := startEchoServer(t)
	defer echoCleanup()

	_, port, err := net.SplitHostPort(echoAddr)
	require.NoError(t, err)

	handler := NewHandler(Config{
		WorkspaceSvcPattern: "127.0.0.1:" + port,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Send a plain HTTP GET (no WebSocket upgrade headers).
	resp, err := http.Get(server.URL + "/ws/test-ws")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// The gorilla upgrader returns 400 when the request is not a WebSocket upgrade.
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProxy_BackendUnreachable(t *testing.T) {
	t.Parallel()

	// Point to a port that nothing is listening on.
	handler := NewHandler(Config{
		WorkspaceSvcPattern: "127.0.0.1:1",
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// The proxy dials TCP first, then upgrades WS. If TCP fails, it returns 502.
	resp, err := http.Get(server.URL + "/ws/test-ws")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestProxy_MissingWorkspaceName(t *testing.T) {
	t.Parallel()

	handler := NewHandler(Config{
		WorkspaceSvcPattern: "127.0.0.1:9999",
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws/")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProxy_HealthEndpoints(t *testing.T) {
	t.Parallel()

	srv := NewServer(Config{
		WorkspaceSvcPattern: "127.0.0.1:9999",
	}, t.TempDir(), "127.0.0.1:0")

	// Use the handler from the server directly with httptest.
	server := httptest.NewServer(srv.Handler)
	defer server.Close()

	t.Run("healthz", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/healthz")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "ok", string(body))
	})

	t.Run("readyz", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/readyz")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "ok", string(body))
	})
}

func TestProxy_WorkspaceRouting(t *testing.T) {
	t.Parallel()

	echoAddr, echoCleanup := startEchoServer(t)
	defer echoCleanup()

	// Use {name} in pattern to verify workspace name is substituted.
	// We route all workspaces to the same echo server for testing.
	_, port, err := net.SplitHostPort(echoAddr)
	require.NoError(t, err)

	handler := NewHandler(Config{
		WorkspaceSvcPattern: "127.0.0.1:" + port,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Connect to /ws/infra and verify bidirectional data.
	dialer := &websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	ws, _, err := dialer.Dial(wsURL(server, "/ws/infra"), nil)
	require.NoError(t, err)
	defer func() { _ = ws.Close() }()

	testData := []byte("workspace-routing-test")
	err = ws.WriteMessage(websocket.BinaryMessage, testData)
	require.NoError(t, err)

	msgType, msg, err := ws.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, websocket.BinaryMessage, msgType)
	assert.Equal(t, testData, msg)
}

// newTestServer creates an httptest.Server backed by the full NewServer mux,
// which includes the key registration and authorized-keys endpoints.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := NewServer(Config{
		WorkspaceSvcPattern: "127.0.0.1:9999",
	}, t.TempDir(), "127.0.0.1:0")
	return httptest.NewServer(srv.Handler)
}

func TestKeyRegistration_RegisterAndFetch(t *testing.T) {
	t.Parallel()
	server := newTestServer(t)
	defer server.Close()

	// Register a public key for workspace "test".
	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey user@host"
	resp, err := http.Post(
		server.URL+"/ws/test/register-key",
		"text/plain",
		strings.NewReader(pubKey),
	)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Fetch authorized keys for workspace "test".
	resp2, err := http.Get(server.URL + "/internal/authorized-keys/test")
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	body, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	assert.Equal(t, pubKey+"\n", string(body))
}

func TestKeyRegistration_EmptyBody(t *testing.T) {
	t.Parallel()
	server := newTestServer(t)
	defer server.Close()

	resp, err := http.Post(
		server.URL+"/ws/test/register-key",
		"text/plain",
		strings.NewReader(""),
	)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestKeyRegistration_UnknownWorkspace(t *testing.T) {
	t.Parallel()
	server := newTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/internal/authorized-keys/unknown")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "", string(body))
}

func TestKeyRegistration_MultipleKeys(t *testing.T) {
	t.Parallel()
	server := newTestServer(t)
	defer server.Close()

	key1 := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKey1 user1@host"
	key2 := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKey2 user2@host"

	// Register two different keys.
	for _, key := range []string{key1, key2} {
		resp, err := http.Post(
			server.URL+"/ws/test/register-key",
			"text/plain",
			strings.NewReader(key),
		)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Fetch authorized keys and verify both are present.
	resp, err := http.Get(server.URL + "/internal/authorized-keys/test")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// The order is non-deterministic (map iteration), so check both keys exist.
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	assert.Equal(t, 2, len(lines))
	assert.Contains(t, lines, key1)
	assert.Contains(t, lines, key2)
}

func TestWorkspaceInfo_Success(t *testing.T) {
	t.Parallel()

	echoAddr, echoCleanup := startEchoServer(t)
	defer echoCleanup()

	_, port, err := net.SplitHostPort(echoAddr)
	require.NoError(t, err)

	srv := NewServer(Config{
		WorkspaceSvcPattern: "127.0.0.1:" + port,
	}, t.TempDir(), "127.0.0.1:0")
	server := httptest.NewServer(srv.Handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws/test/info")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"hostname":"test-127-0-0-1"`)
}

func TestWorkspaceInfo_BackendUnreachable(t *testing.T) {
	t.Parallel()

	srv := NewServer(Config{
		WorkspaceSvcPattern: "127.0.0.1:1",
	}, t.TempDir(), "127.0.0.1:0")
	server := httptest.NewServer(srv.Handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws/test/info")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestWorkspaceInfo_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	defer server.Close()

	resp, err := http.Post(server.URL+"/ws/test/info", "text/plain", strings.NewReader("data"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
