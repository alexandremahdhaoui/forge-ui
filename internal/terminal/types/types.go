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

package types

// TerminalConfig holds the configuration for a terminal session.
type TerminalConfig struct {
	Workspace   string             `json:"workspace"`
	Endpoints   []TerminalEndpoint `json:"endpoints"`
	AutoConnect bool               `json:"autoConnect"`
	Theme       string             `json:"theme"`
	Persist     bool               `json:"persist"`
}

// TerminalEndpoint represents a WebSocket endpoint for terminal connections.
type TerminalEndpoint struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Default bool   `json:"default"`
}

// SSHKey holds an SSH key pair.
type SSHKey struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	PublicKey  string `json:"publicKey"`
	PrivateKey []byte `json:"privateKey"`
	Encrypted  bool   `json:"encrypted"`
}

// KnownHost holds a known host entry for SSH host verification.
type KnownHost struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// TerminalSession represents an active terminal session (in-memory only).
type TerminalSession struct {
	Workspace   string `json:"workspace"`
	SessionName string `json:"sessionName"`
	Connected   bool   `json:"connected"`
	Endpoint    string `json:"endpoint"`
	Username    string `json:"username"`
}

// WorkspaceInfo holds per-workspace metadata returned by the proxy info endpoint.
type WorkspaceInfo struct {
	Hostname string `json:"hostname"`
}

// SSHSessionConfig is passed from the controller to the SSHClient adapter
// to establish a connection.
type SSHSessionConfig struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Hostname string `json:"hostname"`
	Command  string `json:"command"`
	Cols     int    `json:"cols"`
	Rows     int    `json:"rows"`
}
