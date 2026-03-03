package keygen

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
)

// Generate creates a new ed25519 SSH key pair and returns it as an SSHKey
// with Name "default". The private key is in OpenSSH PEM format. The public
// key is in OpenSSH authorized_keys format (without trailing newline).
func Generate() (types.SSHKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return types.SSHKey{}, fmt.Errorf("generate ed25519 key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return types.SSHKey{}, fmt.Errorf("create SSH public key: %w", err)
	}

	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return types.SSHKey{}, fmt.Errorf("marshal private key: %w", err)
	}

	return types.SSHKey{
		Name:       "default",
		Type:       "ed25519",
		PublicKey:  strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))),
		PrivateKey: pem.EncodeToMemory(block),
	}, nil
}
