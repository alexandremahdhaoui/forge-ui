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

package adapter

import (
	"io"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
	"golang.org/x/crypto/ssh"
)

// TerminalIO bridges the terminal UI (xterm.js) with the Go domain.
type TerminalIO interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	OnResize(func(cols, rows int))
	Cols() int
	Rows() int
	Close() error
}

// SSHClient establishes SSH sessions over WebSocket.
type SSHClient interface {
	Connect(cfg types.SSHSessionConfig, signers []ssh.Signer, hostCb ssh.HostKeyCallback) (SSHSession, error)
}

// SSHSession represents an established SSH session returned by SSHClient.Connect.
type SSHSession interface {
	Stdin() io.Writer
	Stdout() io.Reader
	Resize(cols, rows int) error
	Close() error
}

// KeyRegistrar registers SSH public keys with the key provisioning service.
type KeyRegistrar interface {
	RegisterKey(workspace, publicKey string) error
}

// WorkspaceInfoClient retrieves per-workspace metadata from the proxy.
type WorkspaceInfoClient interface {
	GetInfo(workspace string) (types.WorkspaceInfo, error)
}

// KeyStore persists SSH keys, endpoints, known hosts, and parameters
// in browser-local storage (IndexedDB).
type KeyStore interface {
	ListKeys() ([]types.SSHKey, error)
	GetKey(name string) (types.SSHKey, error)
	SaveKey(key types.SSHKey) error
	DeleteKey(name string) error

	ListEndpoints() ([]types.TerminalEndpoint, error)
	SaveEndpoint(ep types.TerminalEndpoint) error

	ListHosts() ([]types.KnownHost, error)
	SaveHost(host types.KnownHost) error

	GetParams() (map[string]string, error)
	SaveParams(params map[string]string) error
}
