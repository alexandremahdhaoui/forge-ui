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
