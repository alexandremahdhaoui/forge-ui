package controller

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/keygen"
	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
)

// sessionController implements SessionController using injected adapters.
type sessionController struct {
	sshClient adapter.SSHClient
	keyStore  adapter.KeyStore
	registrar adapter.KeyRegistrar

	mu      sync.Mutex
	session adapter.SSHSession
}

// New creates a SessionController with the given SSH client, key store, and
// key registrar adapters.
func New(sshClient adapter.SSHClient, keyStore adapter.KeyStore, registrar adapter.KeyRegistrar) SessionController {
	return &sessionController{
		sshClient: sshClient,
		keyStore:  keyStore,
		registrar: registrar,
	}
}

// Start connects to the remote host and bridges TerminalIO with the SSH session.
// It blocks until the session ends or an error occurs.
func (s *sessionController) Start(cfg types.TerminalConfig, termIO adapter.TerminalIO) error {
	// 1. Select endpoint: first with Default=true, or first endpoint.
	ep, err := selectEndpoint(cfg.Endpoints)
	if err != nil {
		return err
	}

	// 2. Load or auto-generate SSH key.
	key, err := s.keyStore.GetKey("default")
	if err != nil {
		key, err = keygen.Generate()
		if err != nil {
			return fmt.Errorf("auto-generate SSH key: %w", err)
		}
		if err := s.keyStore.SaveKey(key); err != nil {
			return fmt.Errorf("save generated SSH key: %w", err)
		}
	}

	// 3. Register public key with proxy (idempotent, handles proxy restarts).
	if err := s.registrar.RegisterKey(cfg.Workspace, key.PublicKey); err != nil {
		return fmt.Errorf("register SSH public key: %w", err)
	}

	// 4. Parse private key into ssh.Signer.
	signer, err := ssh.ParsePrivateKey(key.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse SSH private key: %w", err)
	}

	// 5. Build SSHSessionConfig.
	hostname, err := extractHostname(ep.URL)
	if err != nil {
		return fmt.Errorf("failed to extract hostname from endpoint URL: %w", err)
	}

	sshCfg := types.SSHSessionConfig{
		Endpoint: ep.URL,
		Username: "forge",
		Hostname: hostname,
		Command:  "tmux new-session -A -s forge-" + cfg.Workspace,
		Cols:     termIO.Cols(),
		Rows:     termIO.Rows(),
	}

	// 6. Build host key callback (TOFU: trust on first use).
	hostKeyCb, err := s.buildHostKeyCallback()
	if err != nil {
		return fmt.Errorf("failed to build host key callback: %w", err)
	}

	// 7. Connect via SSH client adapter.
	session, err := s.sshClient.Connect(sshCfg, []ssh.Signer{signer}, hostKeyCb)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	// 8. Store session.
	s.mu.Lock()
	s.session = session
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.session = nil
		s.mu.Unlock()
	}()

	// 9. Register resize callback.
	termIO.OnResize(func(cols, rows int) {
		_ = session.Resize(cols, rows)
	})

	// 10. Bridge I/O bidirectionally.
	errc := make(chan error, 2)

	// goroutine 1: user input -> SSH stdin
	go func() {
		_, err := io.Copy(session.Stdin(), termIO)
		errc <- err
	}()

	// goroutine 2: SSH stdout -> terminal output
	go func() {
		_, err := io.Copy(termIO, session.Stdout())
		errc <- err
	}()

	// 11. Wait for the first goroutine to finish (session ended).
	// When one direction closes, the session is effectively over.
	err = <-errc

	// Close the session to unblock the other goroutine.
	_ = session.Close()

	// Wait for the second goroutine.
	<-errc

	// 12. Return error from the copy if meaningful.
	if err != nil {
		return fmt.Errorf("session ended: %w", err)
	}
	return nil
}

// Stop closes the active SSH session.
func (s *sessionController) Stop() error {
	s.mu.Lock()
	session := s.session
	s.session = nil
	s.mu.Unlock()

	if session != nil {
		return session.Close()
	}
	return nil
}

// selectEndpoint returns the first endpoint with Default=true, or the first
// endpoint if none is marked as default.
func selectEndpoint(endpoints []types.TerminalEndpoint) (types.TerminalEndpoint, error) {
	if len(endpoints) == 0 {
		return types.TerminalEndpoint{}, fmt.Errorf("no endpoints configured")
	}
	for _, ep := range endpoints {
		if ep.Default {
			return ep, nil
		}
	}
	return endpoints[0], nil
}

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

// buildHostKeyCallback implements TOFU (trust on first use): accept and store
// unknown hosts, reject changed host keys.
func (s *sessionController) buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	knownHosts, err := s.keyStore.ListHosts()
	if err != nil {
		return nil, fmt.Errorf("failed to list known hosts: %w", err)
	}

	hostMap := make(map[string]string, len(knownHosts))
	for _, h := range knownHosts {
		hostMap[h.Name] = h.Key
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fp := ssh.FingerprintSHA256(key)
		if knownFP, exists := hostMap[hostname]; exists {
			if knownFP != fp {
				return fmt.Errorf("host key for %q changed: expected %s, got %s", hostname, knownFP, fp)
			}
			return nil
		}
		// TOFU: accept and persist the new host key.
		if err := s.keyStore.SaveHost(types.KnownHost{
			Name: hostname,
			Key:  fp,
		}); err != nil {
			return fmt.Errorf("failed to save host key for %q: %w", hostname, err)
		}
		hostMap[hostname] = fp
		return nil
	}, nil
}
