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
