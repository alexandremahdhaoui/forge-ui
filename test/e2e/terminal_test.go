//go:build e2e

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

package e2e

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/keygen"
)

const (
	sshConnTimeout     = 10 * time.Second
	tmuxSessionTimeout = 5 * time.Second
	testWorkspace      = "test-ws"
	sshUser            = "forge"
)

// Run with: go test -tags e2e -timeout 300s ./test/e2e/

// wsNetConn wraps a gorilla/websocket.Conn as a net.Conn for use with
// golang.org/x/crypto/ssh.NewClientConn.
type wsNetConn struct {
	ws  *websocket.Conn
	buf []byte
}

var _ net.Conn = (*wsNetConn)(nil)

func (c *wsNetConn) Read(b []byte) (int, error) {
	if len(c.buf) == 0 {
		_, p, err := c.ws.ReadMessage()
		if err != nil {
			return 0, err
		}
		c.buf = p
	}
	n := copy(b, c.buf)
	c.buf = c.buf[n:]
	return n, nil
}

func (c *wsNetConn) Write(b []byte) (int, error) {
	return len(b), c.ws.WriteMessage(websocket.BinaryMessage, b)
}

func (c *wsNetConn) Close() error                       { return c.ws.Close() }
func (c *wsNetConn) LocalAddr() net.Addr                { return c.ws.NetConn().LocalAddr() }
func (c *wsNetConn) RemoteAddr() net.Addr               { return c.ws.NetConn().RemoteAddr() }
func (c *wsNetConn) SetDeadline(t time.Time) error {
	if err := c.ws.SetReadDeadline(t); err != nil {
		return err
	}
	return c.ws.SetWriteDeadline(t)
}
func (c *wsNetConn) SetReadDeadline(t time.Time) error  { return c.ws.SetReadDeadline(t) }
func (c *wsNetConn) SetWriteDeadline(t time.Time) error { return c.ws.SetWriteDeadline(t) }

// setupPortForward starts a kubectl port-forward to the forge-wss-proxy
// service and returns the WebSocket URL. The port-forward process is killed
// when the test finishes via t.Cleanup.
func setupPortForward(t *testing.T) string {
	t.Helper()

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		t.Fatal("KUBECONFIG environment variable is not set")
	}

	cmd := exec.Command("kubectl", "port-forward",
		"svc/forge-wss-proxy", "0:8080",
		"--kubeconfig", kubeconfig,
	)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "failed to create stdout pipe")

	require.NoError(t, cmd.Start(), "failed to start kubectl port-forward")

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	// Parse the local port from kubectl stdout output.
	// kubectl prints: "Forwarding from 127.0.0.1:{port} -> 8080"
	portRe := regexp.MustCompile(`Forwarding from 127\.0\.0\.1:(\d+)`)

	scanner := bufio.NewScanner(stdout)
	portCh := make(chan string, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			t.Logf("kubectl: %s", line)
			if matches := portRe.FindStringSubmatch(line); len(matches) == 2 {
				portCh <- matches[1]
				return
			}
		}
	}()

	select {
	case port := <-portCh:
		return fmt.Sprintf("ws://localhost:%s/ws/%s", port, testWorkspace)
	case <-time.After(sshConnTimeout):
		t.Fatal("timed out waiting for kubectl port-forward to be ready")
		return ""
	}
}

// loadTestKey reads the test user private key from fixtures and parses it
// into an ssh.Signer for authentication.
func loadTestKey(t *testing.T) ssh.Signer {
	t.Helper()

	keyPath := "fixtures/test_user_ed25519_key"
	keyData, err := os.ReadFile(keyPath)
	require.NoError(t, err, "failed to read test private key: %s", keyPath)

	signer, err := ssh.ParsePrivateKey(keyData)
	require.NoError(t, err, "failed to parse test private key")
	return signer
}

// connectSSH establishes a WebSocket connection to the WSS proxy and
// performs the SSH handshake over it. Returns the SSH session and the
// underlying WebSocket connection.
func connectSSH(t *testing.T, wsURL string, signer ssh.Signer) (*ssh.Session, *websocket.Conn) {
	t.Helper()

	dialer := websocket.Dialer{
		HandshakeTimeout: sshConnTimeout,
	}
	wsConn, resp, err := dialer.Dial(wsURL, http.Header{})
	require.NoError(t, err, "failed to dial WebSocket")
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	netConn := &wsNetConn{ws: wsConn}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, "forge-workspace-"+testWorkspace, &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // test only
		Timeout:         sshConnTimeout,
	})
	require.NoError(t, err, "SSH handshake failed")

	client := ssh.NewClient(sshConn, chans, reqs)
	session, err := client.NewSession()
	require.NoError(t, err, "failed to create SSH session")

	return session, wsConn
}

// readUntil reads from r until the output contains the target string or
// the timeout expires. Returns the accumulated output.
func readUntil(t *testing.T, r io.Reader, target string, timeout time.Duration) string {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var output strings.Builder
	buf := make([]byte, 4096)

	for time.Now().Before(deadline) {
		if nc, ok := r.(interface{ SetReadDeadline(time.Time) error }); ok {
			_ = nc.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		}
		n, err := r.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
			if strings.Contains(output.String(), target) {
				return output.String()
			}
		}
		if err != nil && !isTimeoutError(err) {
			t.Logf("read error: %v (accumulated output: %q)", err, output.String())
			break
		}
	}
	return output.String()
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok {
		return ne.Timeout()
	}
	return false
}

// httpBaseURL converts a WebSocket URL (ws://localhost:PORT/ws/...) to an HTTP
// base URL (http://localhost:PORT) suitable for REST API calls.
func httpBaseURL(wsURL string) string {
	u := strings.Replace(wsURL, "ws://", "http://", 1)
	u = strings.Replace(u, "wss://", "https://", 1)
	// Strip the path to get just the base.
	if idx := strings.Index(u, "/ws/"); idx != -1 {
		u = u[:idx]
	}
	return u
}

func TestE2E_KeyRegistrationAndSSHAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	wsURL := setupPortForward(t)
	baseURL := httpBaseURL(wsURL)

	// 1. Generate a fresh ed25519 key pair (same as browser WASM flow).
	key, err := keygen.Generate()
	require.NoError(t, err, "failed to generate ed25519 key pair")
	require.True(t, strings.HasPrefix(key.PublicKey, "ssh-ed25519 "), "public key must be ed25519")

	// 2. Register the public key with the wss-proxy via HTTP POST.
	registerURL := baseURL + "/ws/" + testWorkspace + "/register-key"
	resp, err := http.Post(registerURL, "text/plain", strings.NewReader(key.PublicKey)) //nolint:noctx // test only
	require.NoError(t, err, "failed to POST register-key")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode, "register-key should return 200")

	// 3. Parse the generated private key into an ssh.Signer.
	signer, err := ssh.ParsePrivateKey(key.PrivateKey)
	require.NoError(t, err, "failed to parse generated private key")

	// 4. Connect via SSH using the dynamically registered key.
	session, wsConn := connectSSH(t, wsURL, signer)
	t.Cleanup(func() {
		_ = session.Close()
		_ = wsConn.Close()
	})

	// 5. Verify the session works by requesting a PTY and running a command.
	err = session.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	require.NoError(t, err, "failed to request PTY")

	stdout, err := session.StdoutPipe()
	require.NoError(t, err, "failed to get stdout pipe")

	stdin, err := session.StdinPipe()
	require.NoError(t, err, "failed to get stdin pipe")

	err = session.Start("sh")
	require.NoError(t, err, "failed to start shell")

	_ = readUntil(t, stdout, "$", tmuxSessionTimeout)

	_, err = fmt.Fprint(stdin, "echo key-reg-ok\n")
	require.NoError(t, err, "failed to send echo command")

	output := readUntil(t, stdout, "key-reg-ok", tmuxSessionTimeout)
	assert.Contains(t, output, "key-reg-ok", "expected echo output from dynamically registered key session")
}

func TestE2E_WSConnectAndSSHAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	wsURL := setupPortForward(t)
	signer := loadTestKey(t)

	session, wsConn := connectSSH(t, wsURL, signer)
	t.Cleanup(func() {
		_ = session.Close()
		_ = wsConn.Close()
	})

	// If we got here, WebSocket connection and SSH auth succeeded.
	assert.NotNil(t, session, "SSH session should be established")
}

func TestE2E_TmuxSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	wsURL := setupPortForward(t)
	signer := loadTestKey(t)
	session, wsConn := connectSSH(t, wsURL, signer)
	t.Cleanup(func() {
		_ = session.Close()
		_ = wsConn.Close()
	})

	// Request a PTY for the tmux session.
	err := session.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	require.NoError(t, err, "failed to request PTY")

	stdout, err := session.StdoutPipe()
	require.NoError(t, err, "failed to get stdout pipe")

	err = session.Start(fmt.Sprintf("tmux new-session -A -s forge-%s", testWorkspace))
	require.NoError(t, err, "failed to start tmux session")

	// Verify tmux produces output within the timeout.
	output := readUntil(t, stdout, "$", tmuxSessionTimeout)
	assert.NotEmpty(t, output, "expected tmux session output")
}

func TestE2E_SendKeystrokesAndEcho(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	wsURL := setupPortForward(t)
	signer := loadTestKey(t)
	session, wsConn := connectSSH(t, wsURL, signer)
	t.Cleanup(func() {
		_ = session.Close()
		_ = wsConn.Close()
	})

	err := session.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	require.NoError(t, err, "failed to request PTY")

	stdout, err := session.StdoutPipe()
	require.NoError(t, err, "failed to get stdout pipe")

	stdin, err := session.StdinPipe()
	require.NoError(t, err, "failed to get stdin pipe")

	err = session.Start(fmt.Sprintf("tmux new-session -A -s forge-%s", testWorkspace))
	require.NoError(t, err, "failed to start tmux session")

	// Wait for the shell prompt.
	_ = readUntil(t, stdout, "$", tmuxSessionTimeout)

	// Send a command.
	_, err = fmt.Fprint(stdin, "echo hello\n")
	require.NoError(t, err, "failed to send keystrokes")

	// Verify output.
	output := readUntil(t, stdout, "hello", tmuxSessionTimeout)
	assert.Contains(t, output, "hello", "expected echo output to contain 'hello'")
}

func TestE2E_TerminalResize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	wsURL := setupPortForward(t)
	signer := loadTestKey(t)
	session, wsConn := connectSSH(t, wsURL, signer)
	t.Cleanup(func() {
		_ = session.Close()
		_ = wsConn.Close()
	})

	err := session.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	require.NoError(t, err, "failed to request PTY")

	stdout, err := session.StdoutPipe()
	require.NoError(t, err, "failed to get stdout pipe")

	stdin, err := session.StdinPipe()
	require.NoError(t, err, "failed to get stdin pipe")

	err = session.Start(fmt.Sprintf("tmux new-session -A -s forge-%s", testWorkspace))
	require.NoError(t, err, "failed to start tmux session")

	// Wait for the shell prompt.
	_ = readUntil(t, stdout, "$", tmuxSessionTimeout)

	// Resize the terminal.
	err = session.WindowChange(40, 120)
	require.NoError(t, err, "failed to send window change")

	// Give the terminal time to process the resize.
	time.Sleep(500 * time.Millisecond)

	// Query the terminal width.
	_, err = fmt.Fprint(stdin, "tput cols\n")
	require.NoError(t, err, "failed to send tput command")

	output := readUntil(t, stdout, "120", tmuxSessionTimeout)
	assert.Contains(t, output, "120", "expected tput cols output to contain '120'")
}

func TestE2E_DisconnectAndReconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	wsURL := setupPortForward(t)
	signer := loadTestKey(t)

	// First connection: create a tmux session and send a marker command.
	session1, wsConn1 := connectSSH(t, wsURL, signer)

	err := session1.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	require.NoError(t, err, "failed to request PTY")

	stdout1, err := session1.StdoutPipe()
	require.NoError(t, err, "failed to get stdout pipe")

	stdin1, err := session1.StdinPipe()
	require.NoError(t, err, "failed to get stdin pipe")

	err = session1.Start(fmt.Sprintf("tmux new-session -A -s forge-%s", testWorkspace))
	require.NoError(t, err, "failed to start tmux session")

	_ = readUntil(t, stdout1, "$", tmuxSessionTimeout)

	// Send a marker that we can verify after reconnection.
	_, err = fmt.Fprint(stdin1, "export FORGE_RECONNECT_MARKER=alive\n")
	require.NoError(t, err, "failed to send marker command")
	_ = readUntil(t, stdout1, "$", tmuxSessionTimeout)

	// Disconnect by closing the WebSocket.
	_ = session1.Close()
	_ = wsConn1.Close()

	// Give the server time to detect the disconnection.
	time.Sleep(1 * time.Second)

	// Reconnect.
	session2, wsConn2 := connectSSH(t, wsURL, signer)
	t.Cleanup(func() {
		_ = session2.Close()
		_ = wsConn2.Close()
	})

	err = session2.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	require.NoError(t, err, "failed to request PTY on reconnect")

	stdout2, err := session2.StdoutPipe()
	require.NoError(t, err, "failed to get stdout pipe on reconnect")

	stdin2, err := session2.StdinPipe()
	require.NoError(t, err, "failed to get stdin pipe on reconnect")

	// Reattach to the same tmux session.
	err = session2.Start(fmt.Sprintf("tmux new-session -A -s forge-%s", testWorkspace))
	require.NoError(t, err, "failed to reattach tmux session")

	_ = readUntil(t, stdout2, "$", tmuxSessionTimeout)

	// Verify the session persisted by checking our marker.
	_, err = fmt.Fprint(stdin2, "echo $FORGE_RECONNECT_MARKER\n")
	require.NoError(t, err, "failed to send echo marker command")

	output := readUntil(t, stdout2, "alive", tmuxSessionTimeout)
	assert.Contains(t, output, "alive", "tmux session should persist after reconnection")
}
