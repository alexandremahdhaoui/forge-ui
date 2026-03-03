package adapter

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type httpKeyRegistrar struct {
	baseURL string
}

// NewKeyRegistrar creates a KeyRegistrar that POSTs public keys to the
// wss-proxy registration endpoint. The baseURL is derived from the
// WebSocket endpoint URL (e.g., "ws://host/ws/default" -> "http://host").
func NewKeyRegistrar(wsEndpoint string) KeyRegistrar {
	u, err := url.Parse(wsEndpoint)
	if err != nil {
		return &httpKeyRegistrar{}
	}
	scheme := "http"
	if u.Scheme == "wss" || u.Scheme == "https" {
		scheme = "https"
	}
	return &httpKeyRegistrar{baseURL: scheme + "://" + u.Host}
}

func (r *httpKeyRegistrar) RegisterKey(workspace, publicKey string) error {
	resp, err := http.Post(
		r.baseURL+"/ws/"+workspace+"/register-key",
		"text/plain",
		strings.NewReader(publicKey),
	)
	if err != nil {
		return fmt.Errorf("register key: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register key: %s: %s", resp.Status, body)
	}
	return nil
}
