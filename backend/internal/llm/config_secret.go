package llm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// ConfigSecretCipher encrypts administrator-managed provider credentials.
// The key is derived from the server secret and never persisted alongside the
// ciphertext. Runtime snapshots must not contain ciphertext or plaintext keys.
type ConfigSecretCipher struct {
	key [32]byte
}

func NewConfigSecretCipher(secret string) *ConfigSecretCipher {
	return &ConfigSecretCipher{key: sha256.Sum256([]byte(secret))}
}

func (c *ConfigSecretCipher) Encrypt(plain string) (string, error) {
	if c == nil || plain == "" {
		return "", errors.New("llm config secret is empty")
	}
	block, err := aes.NewCipher(c.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (c *ConfigSecretCipher) Decrypt(encoded string) (string, error) {
	if c == nil || encoded == "" {
		return "", errors.New("llm config ciphertext is empty")
	}
	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize() {
		return "", errors.New("llm config ciphertext is truncated")
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
