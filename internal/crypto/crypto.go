package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// deriveKeySalt is a fixed salt for key derivation. Using a fixed salt is
// acceptable here because we are deriving a single encryption key from a
// server-level passphrase (not hashing user passwords). The passphrase
// has high entropy (config requires >= 16 chars) and there is only one
// derived key per deployment, so rainbow tables are not a concern.
var deriveKeySalt = []byte("pxbin-encryption-key-v1")

// DeriveKey returns a 32-byte AES-256 key from a passphrase using Argon2id.
// Argon2id is memory-hard, making GPU/ASIC brute-force attacks expensive.
func DeriveKey(passphrase string) []byte {
	return argon2.IDKey([]byte(passphrase), deriveKeySalt, 1, 64*1024, 4, 32)
}

// Encrypt encrypts plaintext with AES-256-GCM. A random 12-byte nonce is
// prepended to the ciphertext. The result is returned as a base64 string.
func Encrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes a base64 ciphertext string, splits the prepended nonce,
// and decrypts with AES-256-GCM.
func Decrypt(ciphertext string, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// IsEncrypted returns true if s looks like a base64-encoded AES-GCM
// ciphertext (minimum length: 12-byte nonce + 16-byte tag = 28 bytes
// → 40 base64 chars). Plaintext API keys never look like valid base64
// of that length.
func IsEncrypted(s string) bool {
	if len(s) < 40 {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}
