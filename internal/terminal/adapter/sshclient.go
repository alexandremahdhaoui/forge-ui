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

// Adapted from sshterm (MIT, c2FmZQ/sshterm)
//
// MIT License
//
// Copyright (c) 2024 TTBT Enterprises LLC
// Copyright (c) 2024 Robin Thellend <rthellend@rthellend.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

//go:build js && wasm

package adapter

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
)

// Compile-time checks.
var _ SSHClient = (*sshClient)(nil)
var _ SSHSession = (*sshSession)(nil)

// sshClient implements SSHClient. It is stateless; each Connect call creates
// a new WebSocket, SSH client, and session.
type sshClient struct{}

// NewSSHClient returns a new SSHClient adapter.
func NewSSHClient() *sshClient {
	return &sshClient{}
}

// terminalModes matches the terminal modes from sshterm's ssh.go.
var terminalModes = ssh.TerminalModes{
	ssh.ECHO:          1,
	ssh.ICRNL:         1,
	ssh.IXON:          1,
	ssh.IXANY:         1,
	ssh.IMAXBEL:       1,
	ssh.OPOST:         1,
	ssh.ONLCR:         1,
	ssh.ISIG:          1,
	ssh.ICANON:        1,
	ssh.IEXTEN:        1,
	ssh.ECHOE:         1,
	ssh.ECHOK:         1,
	ssh.ECHOCTL:       1,
	ssh.ECHOKE:        1,
	ssh.TTY_OP_ISPEED: 14400,
	ssh.TTY_OP_OSPEED: 14400,
}

// Connect creates a WebSocket connection to the endpoint, establishes an SSH
// session with PTY, and starts the given command.
func (c *sshClient) Connect(cfg types.SSHSessionConfig, signers []ssh.Signer, hostCb ssh.HostKeyCallback) (SSHSession, error) {
	// 1. Create WebSocket connection.
	wsConn, err := NewWebSocket(context.Background(), cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	// 2. Create SSH client connection over WebSocket.
	conn, chans, reqs, err := ssh.NewClientConn(wsConn, cfg.Hostname, &ssh.ClientConfig{
		User: cfg.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
		},
		HostKeyCallback: hostCb,
	})
	if err != nil {
		wsConn.Close()
		return nil, fmt.Errorf("ssh client conn: %w", err)
	}

	// 3. Create SSH client.
	client := ssh.NewClient(conn, chans, reqs)

	// 4. Create SSH session.
	sess, err := client.NewSession()
	if err != nil {
		client.Close()
		wsConn.Close()
		return nil, fmt.Errorf("ssh new session: %w", err)
	}

	// 5. Request PTY.
	if err := sess.RequestPty("xterm", cfg.Rows, cfg.Cols, terminalModes); err != nil {
		sess.Close()
		client.Close()
		wsConn.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}

	// 6. Get stdin pipe.
	stdin, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		client.Close()
		wsConn.Close()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	// 7. Get stdout pipe.
	stdout, err := sess.StdoutPipe()
	if err != nil {
		sess.Close()
		client.Close()
		wsConn.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// 8. Start the command.
	if err := sess.Start(cfg.Command); err != nil {
		sess.Close()
		client.Close()
		wsConn.Close()
		return nil, fmt.Errorf("session start: %w", err)
	}

	return &sshSession{
		sess:   sess,
		client: client,
		wsConn: wsConn,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

// sshSession wraps an active SSH session, implementing the SSHSession interface.
type sshSession struct {
	sess   *ssh.Session
	client *ssh.Client
	wsConn *WebSocket
	stdin  io.WriteCloser
	stdout io.Reader
}

// Stdin returns the writer connected to the remote process stdin.
func (s *sshSession) Stdin() io.Writer {
	return s.stdin
}

// Stdout returns the reader connected to the remote process stdout.
func (s *sshSession) Stdout() io.Reader {
	return s.stdout
}

// Resize sends a window-change request to the SSH session.
func (s *sshSession) Resize(cols, rows int) error {
	return s.sess.WindowChange(rows, cols)
}

// Close closes the SSH session, client, and WebSocket connection.
func (s *sshSession) Close() error {
	s.sess.Close()
	s.client.Close()
	s.wsConn.Close()
	return nil
}
