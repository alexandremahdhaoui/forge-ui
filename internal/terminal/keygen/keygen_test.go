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

package keygen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestGenerate_ReturnsValidSSHKey(t *testing.T) {
	key, err := Generate()
	require.NoError(t, err)

	assert.Equal(t, "default", key.Name)
	assert.Equal(t, "ed25519", key.Type)
	assert.False(t, key.Encrypted)
}

func TestGenerate_PrivateKeyParseable(t *testing.T) {
	key, err := Generate()
	require.NoError(t, err)

	signer, err := ssh.ParsePrivateKey(key.PrivateKey)
	require.NoError(t, err, "PrivateKey must be parseable by ssh.ParsePrivateKey")
	assert.NotNil(t, signer)
}

func TestGenerate_PublicKeyFormat(t *testing.T) {
	key, err := Generate()
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(key.PublicKey, "ssh-ed25519 "),
		"PublicKey should start with 'ssh-ed25519 ', got: %s", key.PublicKey)
	assert.NotEmpty(t, key.PublicKey)
	assert.False(t, strings.HasSuffix(key.PublicKey, "\n"),
		"PublicKey should not end with newline")
}

func TestGenerate_PrivateKeyPEMFormat(t *testing.T) {
	key, err := Generate()
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(string(key.PrivateKey), "-----BEGIN OPENSSH PRIVATE KEY-----"),
		"PrivateKey should start with OpenSSH PEM header")
}

func TestGenerate_MultipleCallsProduceDifferentKeys(t *testing.T) {
	key1, err := Generate()
	require.NoError(t, err)

	key2, err := Generate()
	require.NoError(t, err)

	assert.NotEqual(t, key1.PublicKey, key2.PublicKey,
		"two calls to Generate should produce different key pairs")
}

func TestGenerate_PrivateKeyMatchesPublicKey(t *testing.T) {
	key, err := Generate()
	require.NoError(t, err)

	// Parse the private key to extract its public key.
	signer, err := ssh.ParsePrivateKey(key.PrivateKey)
	require.NoError(t, err)

	// Marshal the signer's public key in authorized_keys format and trim.
	derivedPub := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))

	assert.Equal(t, key.PublicKey, derivedPub,
		"public key derived from private key must match stored PublicKey")
}
