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

package controller

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/mocks/mockterminaladapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// extractHostname parses the given URL and returns its hostname (without port).
func extractHostname(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	h := u.Hostname()
	if h == "" {
		return "", fmt.Errorf("no hostname in URL %q", rawURL)
	}
	return h, nil
}

func TestNew(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	require.NotNil(t, ctrl)
}

func TestStart_NoEndpoints(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: nil,
	}, termIO)

	assert.EqualError(t, err, "no endpoints configured")
}

func TestStart_AutoGeneratesKeyWhenMissing(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	// GetKey fails -> triggers auto-generation.
	keyStore.On("GetKey", "default").Return(types.SSHKey{}, errors.New("key not found"))
	// SaveKey is called with the generated key.
	keyStore.On("SaveKey", mock.MatchedBy(func(k types.SSHKey) bool {
		return k.Name == "default" && k.Type == "ed25519" &&
			strings.HasPrefix(k.PublicKey, "ssh-ed25519 ") &&
			len(k.PrivateKey) > 0
	})).Return(nil)
	// RegisterKey is called with the generated public key.
	registrar.On("RegisterKey", "test-ws", mock.MatchedBy(func(pk string) bool {
		return strings.HasPrefix(pk, "ssh-ed25519 ")
	})).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Return()

	stdinBuf := &bytes.Buffer{}
	stdoutBuf := &bytes.Buffer{}

	session.On("Stdin").Return(io.Writer(stdinBuf))
	session.On("Stdout").Return(io.Reader(stdoutBuf))
	session.On("Close").Return(nil).Maybe()

	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(0, io.EOF).Maybe()
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return len(b), nil
	}).Maybe()

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Return(session, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.NoError(t, err)
	keyStore.AssertCalled(t, "SaveKey", mock.Anything)
	registrar.AssertCalled(t, "RegisterKey", "test-ws", mock.Anything)
}

func TestStart_RegistersExistingKey(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PublicKey:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBn6GTRlLW3EG34EvSDbaJWXIfNCYrlySXROA7Mkz2Zp",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	// RegisterKey is called with the existing public key.
	registrar.On("RegisterKey", "test-ws", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBn6GTRlLW3EG34EvSDbaJWXIfNCYrlySXROA7Mkz2Zp").
		Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Return()

	stdinBuf := &bytes.Buffer{}
	stdoutBuf := &bytes.Buffer{}

	session.On("Stdin").Return(io.Writer(stdinBuf))
	session.On("Stdout").Return(io.Reader(stdoutBuf))
	session.On("Close").Return(nil).Maybe()

	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(0, io.EOF).Maybe()
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return len(b), nil
	}).Maybe()

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Return(session, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.NoError(t, err)
	// SaveKey should NOT be called since key already existed.
	keyStore.AssertNotCalled(t, "SaveKey", mock.Anything)
	registrar.AssertCalled(t, "RegisterKey", "test-ws", mock.Anything)
}

func TestStart_RegisterKeyError(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PublicKey:  "ssh-ed25519 AAAA...",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", "ssh-ed25519 AAAA...").
		Return(errors.New("connection refused"))

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.ErrorContains(t, err, "register SSH public key")
	assert.ErrorContains(t, err, "connection refused")
}

func TestStart_SaveKeyErrorAfterGeneration(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)

	// GetKey fails -> triggers auto-generation.
	keyStore.On("GetKey", "default").Return(types.SSHKey{}, errors.New("key not found"))
	// SaveKey fails.
	keyStore.On("SaveKey", mock.Anything).Return(errors.New("storage full"))

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.ErrorContains(t, err, "save generated SSH key")
	assert.ErrorContains(t, err, "storage full")
}

func TestStart_ParsePrivateKeyError(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: []byte("not-a-valid-key"),
	}, nil)
	registrar.On("RegisterKey", "test-ws", "").Return(nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.ErrorContains(t, err, "failed to parse SSH private key")
}

func TestStart_GetInfoError(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{}, errors.New("info endpoint unreachable"))

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.ErrorContains(t, err, "failed to get workspace info")
	assert.ErrorContains(t, err, "info endpoint unreachable")
}

func TestStart_ConnectError(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Return(nil, errors.New("connection refused"))

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.ErrorContains(t, err, "SSH connection failed")
}

func TestStart_SuccessfulSession(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Return()

	// Simulate a short session: termIO reads "hello" then EOF.
	stdinBuf := &bytes.Buffer{}
	stdoutBuf := bytes.NewBufferString("world")

	session.On("Stdin").Return(io.Writer(stdinBuf))
	session.On("Stdout").Return(io.Reader(stdoutBuf))
	session.On("Close").Return(nil).Maybe()

	// termIO.Read returns "hello" then EOF.
	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return copy(b, "hello"), io.EOF
	}).Once()
	// termIO.Write accepts whatever comes from session stdout.
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return len(b), nil
	}).Maybe()

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			cfg := args.Get(0).(types.SSHSessionConfig)
			assert.Equal(t, "wss://proxy.example.com/ws/test-ws", cfg.Endpoint)
			assert.Equal(t, "forge", cfg.Username)
			assert.Equal(t, "test-ws-10-0-1-5", cfg.Hostname)
			assert.Equal(t, "tmux new-session -A -s forge-test-ws", cfg.Command)
			assert.Equal(t, 80, cfg.Cols)
			assert.Equal(t, 24, cfg.Rows)
		}).
		Return(session, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	// Session ends when io.Copy finishes (EOF from termIO.Read).
	assert.NoError(t, err)
}

func TestStart_SelectsDefaultEndpoint(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(120)
	termIO.On("Rows").Return(40)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Return()

	stdinBuf := &bytes.Buffer{}
	stdoutBuf := &bytes.Buffer{} // empty -> EOF immediately

	session.On("Stdin").Return(io.Writer(stdinBuf))
	session.On("Stdout").Return(io.Reader(stdoutBuf))
	session.On("Close").Return(nil).Maybe()

	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(0, io.EOF).Maybe()
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return len(b), nil
	}).Maybe()

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			cfg := args.Get(0).(types.SSHSessionConfig)
			// Should use the second endpoint (the one with Default=true).
			assert.Equal(t, "wss://secondary.example.com/ws/test-ws", cfg.Endpoint)
			assert.Equal(t, "test-ws-10-0-1-5", cfg.Hostname)
		}).
		Return(session, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "primary", URL: "wss://primary.example.com/ws/test-ws", Default: false},
			{Name: "secondary", URL: "wss://secondary.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.NoError(t, err)
}

func TestStart_FallsBackToFirstEndpoint(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Return()

	stdinBuf := &bytes.Buffer{}
	stdoutBuf := &bytes.Buffer{}

	session.On("Stdin").Return(io.Writer(stdinBuf))
	session.On("Stdout").Return(io.Reader(stdoutBuf))
	session.On("Close").Return(nil).Maybe()

	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(0, io.EOF).Maybe()
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return len(b), nil
	}).Maybe()

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			cfg := args.Get(0).(types.SSHSessionConfig)
			// No default, should use first endpoint.
			assert.Equal(t, "wss://first.example.com/ws/test-ws", cfg.Endpoint)
		}).
		Return(session, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "first", URL: "wss://first.example.com/ws/test-ws"},
			{Name: "second", URL: "wss://second.example.com/ws/test-ws"},
		},
	}, termIO)

	assert.NoError(t, err)
}

func TestStop_NoActiveSession(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Stop()

	assert.NoError(t, err)
}

func TestStop_ClosesActiveSession(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)

	ctrl := New(sshClient, keyStore, registrar, infoClient).(*sessionController)

	session := mockterminaladapter.NewMockSSHSession(t)
	session.On("Close").Return(nil)

	ctrl.mu.Lock()
	ctrl.session = session
	ctrl.mu.Unlock()

	err := ctrl.Stop()
	assert.NoError(t, err)

	// Session should be nil after Stop.
	ctrl.mu.Lock()
	assert.Nil(t, ctrl.session)
	ctrl.mu.Unlock()
}

func TestSelectEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		_, err := selectEndpoint(nil)
		assert.Error(t, err)
	})

	t.Run("default selected", func(t *testing.T) {
		ep, err := selectEndpoint([]types.TerminalEndpoint{
			{Name: "a", URL: "wss://a"},
			{Name: "b", URL: "wss://b", Default: true},
		})
		require.NoError(t, err)
		assert.Equal(t, "b", ep.Name)
	})

	t.Run("first used when no default", func(t *testing.T) {
		ep, err := selectEndpoint([]types.TerminalEndpoint{
			{Name: "a", URL: "wss://a"},
			{Name: "b", URL: "wss://b"},
		})
		require.NoError(t, err)
		assert.Equal(t, "a", ep.Name)
	})
}

func TestExtractHostname(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{"wss URL", "wss://proxy.example.com/ws/test", "proxy.example.com", false},
		{"wss URL with port", "wss://proxy.example.com:8443/ws/test", "proxy.example.com", false},
		{"https URL", "https://host.example.com/path", "host.example.com", false},
		{"empty hostname", "wss:///path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractHostname(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestStart_ListHostsError(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return(nil, errors.New("storage failure"))

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.ErrorContains(t, err, "failed to build host key callback")
}

func TestStart_ConcurrentBidirectionalIO(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Return()

	// Use io.Pipe for both directions to simulate concurrent bidirectional flow.
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	session.On("Stdin").Return(io.Writer(stdinWriter))
	session.On("Stdout").Return(io.Reader(stdoutReader))
	session.On("Close").Run(func(_ mock.Arguments) {
		_ = stdinWriter.Close()
		_ = stdoutWriter.Close()
	}).Return(nil).Maybe()

	// termIO.Read provides data from a buffer, then EOF.
	// Use a channel to signal when all input data has been consumed so we
	// can coordinate the stdout pipe close to happen after stdin data flows.
	inputData := []byte("input-from-terminal")
	termIOReader := bytes.NewReader(inputData)
	inputConsumed := make(chan struct{})
	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		n, err := termIOReader.Read(b)
		if err == io.EOF {
			select {
			case <-inputConsumed:
			default:
				close(inputConsumed)
			}
		}
		return n, err
	})

	// termIO.Write collects data written to the terminal.
	var termIOWritten bytes.Buffer
	var writeMu sync.Mutex
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		writeMu.Lock()
		defer writeMu.Unlock()
		return termIOWritten.Write(b)
	})

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Return(session, nil)

	// Write output data from the "SSH server" side. Wait for the input
	// direction to finish before closing the stdout pipe so both directions
	// have time to transfer data concurrently.
	outputData := []byte("output-from-server")
	go func() {
		_, _ = stdoutWriter.Write(outputData)
		// Wait until the termIO input data has been fully read before closing,
		// ensuring the stdin direction completes before the session ends.
		<-inputConsumed
		_ = stdoutWriter.Close()
	}()

	// Read what was written to session stdin from a goroutine.
	var stdinReceived bytes.Buffer
	var stdinWg sync.WaitGroup
	stdinWg.Add(1)
	go func() {
		defer stdinWg.Done()
		_, _ = io.Copy(&stdinReceived, stdinReader)
	}()

	ctrl := New(sshClient, keyStore, registrar, infoClient)

	done := make(chan error, 1)
	go func() {
		done <- ctrl.Start(types.TerminalConfig{
			Workspace: "test-ws",
			Endpoints: []types.TerminalEndpoint{
				{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
			},
		}, termIO)
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return within timeout, possible deadlock")
	}

	// Wait for the stdin reader goroutine to finish before asserting,
	// to avoid a data race on stdinReceived.
	stdinWg.Wait()

	// Verify data flowed in both directions.
	assert.Equal(t, string(inputData), stdinReceived.String())
	writeMu.Lock()
	assert.Equal(t, string(outputData), termIOWritten.String())
	writeMu.Unlock()
}

func TestStart_ReadErrorPropagates(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Return()

	stdinBuf := &bytes.Buffer{}
	// Stdout pipe that stays open until session closes.
	stdoutReader, stdoutWriter := io.Pipe()

	session.On("Stdin").Return(io.Writer(stdinBuf))
	session.On("Stdout").Return(io.Reader(stdoutReader))
	session.On("Close").Run(func(_ mock.Arguments) {
		_ = stdoutWriter.Close()
	}).Return(nil).Maybe()

	readErr := errors.New("terminal read failure")
	callCount := 0
	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		callCount++
		if callCount == 1 {
			return copy(b, "partial"), nil
		}
		return 0, readErr
	})
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return len(b), nil
	}).Maybe()

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Return(session, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)

	done := make(chan error, 1)
	go func() {
		done <- ctrl.Start(types.TerminalConfig{
			Workspace: "test-ws",
			Endpoints: []types.TerminalEndpoint{
				{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
			},
		}, termIO)
	}()

	select {
	case err := <-done:
		assert.ErrorContains(t, err, "terminal read failure")
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return within timeout")
	}
}

func TestStart_RegistersResizeCallback(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)

	stdinBuf := &bytes.Buffer{}
	stdoutBuf := &bytes.Buffer{} // empty -> EOF immediately

	session.On("Stdin").Return(io.Writer(stdinBuf))
	session.On("Stdout").Return(io.Reader(stdoutBuf))
	session.On("Close").Return(nil).Maybe()
	session.On("Resize", 120, 40).Return(nil).Once()

	// Capture the resize callback and invoke it.
	var resizeCb func(int, int)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Run(func(args mock.Arguments) {
		resizeCb = args.Get(0).(func(int, int))
	}).Return()

	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(0, io.EOF).Maybe()
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return len(b), nil
	}).Maybe()

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Return(session, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)
	err := ctrl.Start(types.TerminalConfig{
		Workspace: "test-ws",
		Endpoints: []types.TerminalEndpoint{
			{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
		},
	}, termIO)

	assert.NoError(t, err)

	// Invoke the captured resize callback -- this exercises the OnResize body.
	require.NotNil(t, resizeCb, "OnResize callback was not registered")
	resizeCb(120, 40)

	session.AssertCalled(t, "Resize", 120, 40)
}

func TestBuildHostKeyCallback_TOFUAcceptsNewHost(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)

	// No existing known hosts -- TOFU should accept and save.
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Once()

	ctrl := New(sshClient, keyStore, registrar, infoClient).(*sessionController)
	cb, err := ctrl.buildHostKeyCallback()
	require.NoError(t, err)

	// Generate a test public key to pass to the callback.
	signer, err := ssh.ParsePrivateKey(testEd25519PrivateKeyPEM)
	require.NoError(t, err)
	pubKey := signer.PublicKey()

	err = cb("new-host.example.com", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, pubKey)
	assert.NoError(t, err)

	keyStore.AssertCalled(t, "SaveHost", mock.MatchedBy(func(h types.KnownHost) bool {
		return h.Name == "new-host.example.com" && h.Key != ""
	}))
}

func TestBuildHostKeyCallback_KnownHostMatches(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)

	// Parse the test key to get its fingerprint.
	signer, err := ssh.ParsePrivateKey(testEd25519PrivateKeyPEM)
	require.NoError(t, err)
	pubKey := signer.PublicKey()
	fp := ssh.FingerprintSHA256(pubKey)

	keyStore.On("ListHosts").Return([]types.KnownHost{
		{Name: "known-host.example.com", Key: fp},
	}, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient).(*sessionController)
	cb, err := ctrl.buildHostKeyCallback()
	require.NoError(t, err)

	err = cb("known-host.example.com", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, pubKey)
	assert.NoError(t, err)
}

func TestBuildHostKeyCallback_KnownHostChanged(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)

	// Known host has a different fingerprint than what we will present.
	keyStore.On("ListHosts").Return([]types.KnownHost{
		{Name: "changed-host.example.com", Key: "SHA256:old-fingerprint-that-wont-match"},
	}, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient).(*sessionController)
	cb, err := ctrl.buildHostKeyCallback()
	require.NoError(t, err)

	signer, err := ssh.ParsePrivateKey(testEd25519PrivateKeyPEM)
	require.NoError(t, err)
	pubKey := signer.PublicKey()

	err = cb("changed-host.example.com", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, pubKey)
	assert.ErrorContains(t, err, "host key for")
	assert.ErrorContains(t, err, "changed")
}

func TestBuildHostKeyCallback_SaveHostError(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)

	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(errors.New("storage write failed"))

	ctrl := New(sshClient, keyStore, registrar, infoClient).(*sessionController)
	cb, err := ctrl.buildHostKeyCallback()
	require.NoError(t, err)

	signer, err := ssh.ParsePrivateKey(testEd25519PrivateKeyPEM)
	require.NoError(t, err)
	pubKey := signer.PublicKey()

	err = cb("new-host.example.com", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, pubKey)
	assert.ErrorContains(t, err, "failed to save host key")
}

func TestStart_SessionEndedError(t *testing.T) {
	t.Parallel()
	sshClient := mockterminaladapter.NewMockSSHClient(t)
	keyStore := mockterminaladapter.NewMockKeyStore(t)
	registrar := mockterminaladapter.NewMockKeyRegistrar(t)
	infoClient := mockterminaladapter.NewMockWorkspaceInfoClient(t)
	termIO := mockterminaladapter.NewMockTerminalIO(t)
	session := mockterminaladapter.NewMockSSHSession(t)

	keyStore.On("GetKey", "default").Return(types.SSHKey{
		Name:       "default",
		PrivateKey: testEd25519PrivateKeyPEM,
	}, nil)
	registrar.On("RegisterKey", "test-ws", mock.Anything).Return(nil)
	infoClient.On("GetInfo", "test-ws").Return(types.WorkspaceInfo{Hostname: "test-ws-10-0-1-5"}, nil)
	keyStore.On("ListHosts").Return([]types.KnownHost{}, nil)
	keyStore.On("SaveHost", mock.AnythingOfType("types.KnownHost")).Return(nil).Maybe()

	termIO.On("Cols").Return(80)
	termIO.On("Rows").Return(24)
	termIO.On("OnResize", mock.AnythingOfType("func(int, int)")).Return()

	// Both sides return non-EOF errors to exercise the "session ended" path.
	stdoutReader, stdoutWriter := io.Pipe()
	stdinBuf := &bytes.Buffer{}

	session.On("Stdin").Return(io.Writer(stdinBuf))
	session.On("Stdout").Return(io.Reader(stdoutReader))
	session.On("Close").Run(func(_ mock.Arguments) {
		_ = stdoutWriter.Close()
	}).Return(nil).Maybe()

	// termIO.Read returns a network error.
	termIO.On("Read", mock.AnythingOfType("[]uint8")).Return(0, errors.New("network broken"))
	termIO.On("Write", mock.AnythingOfType("[]uint8")).Return(func(b []byte) (int, error) {
		return len(b), nil
	}).Maybe()

	sshClient.On("Connect", mock.AnythingOfType("types.SSHSessionConfig"), mock.Anything, mock.Anything).
		Return(session, nil)

	ctrl := New(sshClient, keyStore, registrar, infoClient)

	done := make(chan error, 1)
	go func() {
		done <- ctrl.Start(types.TerminalConfig{
			Workspace: "test-ws",
			Endpoints: []types.TerminalEndpoint{
				{Name: "default", URL: "wss://proxy.example.com/ws/test-ws", Default: true},
			},
		}, termIO)
	}()

	select {
	case err := <-done:
		assert.ErrorContains(t, err, "session ended")
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return within timeout")
	}
}

// testEd25519PrivateKeyPEM is a test-only Ed25519 private key in OpenSSH format.
// Generated with: ssh-keygen -t ed25519 -f /dev/stdout -N "" -C "test"
// This key is NOT used for any real authentication.
var testEd25519PrivateKeyPEM = []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAZ+hk0ZS1txBt+BL0g22iVlyHzQmK5ckl0TgOzJM9maQAAAIiarC3imqwt
4gAAAAtzc2gtZWQyNTUxOQAAACAZ+hk0ZS1txBt+BL0g22iVlyHzQmK5ckl0TgOzJM9maQ
AAAEAf0t1cfMMm6gHZ+qdHDEQTh1PSi/m51ql0BXV0MzA91xn6GTRlLW3EG34EvSDbaJWX
IfNCYrlySXROA7Mkz2ZpAAAABHRlc3QB
-----END OPENSSH PRIVATE KEY-----
`)
