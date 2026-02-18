// Package vault provides AES-256-GCM encryption/decryption for sensitive config values.
//
// Encrypted strings are prefixed with "$FORVAULT;" so they can be identified.
// Use the vault sub-command or the Encrypt helper to produce encrypted values,
// then store them in config.yaml or inventory files.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

// Prefix identifies vault-encrypted strings.
const Prefix = "$FORVAULT;"

func deriveKey(password string) []byte {
	h := sha256.Sum256([]byte(password))
	return h[:]
}

// Encrypt encrypts plaintext with AES-256-GCM using the given password.
// The result is prefixed with Prefix so it can later be identified and decrypted.
func Encrypt(plaintext, password string) (string, error) {
	key := deriveKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return Prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts a vault-encrypted string. If the string does not start with
// Prefix it is returned unchanged (pass-through for plain-text values).
func Decrypt(ciphertext, password string) (string, error) {
	if !strings.HasPrefix(ciphertext, Prefix) {
		return ciphertext, nil
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, Prefix))
	if err != nil {
		return "", fmt.Errorf("vault decode: %w", err)
	}
	key := deriveKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", fmt.Errorf("vault: ciphertext too short")
	}
	plain, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("vault decrypt: %w", err)
	}
	return string(plain), nil
}

// IsEncrypted reports whether s is vault-encrypted.
func IsEncrypted(s string) bool {
	return strings.HasPrefix(s, Prefix)
}

// LoadPassword reads the vault password from a file, trimming whitespace.
func LoadPassword(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("reading vault password file %q: %w", file, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// DecryptMap decrypts every vault-encrypted value in m in-place.
func DecryptMap(m map[string]string, password string) error {
	for k, v := range m {
		dec, err := Decrypt(v, password)
		if err != nil {
			return fmt.Errorf("decrypting key %q: %w", k, err)
		}
		m[k] = dec
	}
	return nil
}
