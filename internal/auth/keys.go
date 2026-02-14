package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	llmKeyPrefix        = "pxb_"
	managementKeyPrefix = "pxm_"
	keyRandomBytes      = 20 // 20 bytes = 40 hex chars
)

func GenerateLLMKey() (plaintext, hash, prefix string) {
	plaintext = llmKeyPrefix + randomHex(keyRandomBytes)
	hash = HashKey(plaintext)
	prefix = plaintext[:8]
	return
}

func GenerateManagementKey() (plaintext, hash, prefix string) {
	plaintext = managementKeyPrefix + randomHex(keyRandomBytes)
	hash = HashKey(plaintext)
	prefix = plaintext[:8]
	return
}

func HashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func ValidateKeyFormat(key string) (string, error) {
	switch {
	case strings.HasPrefix(key, llmKeyPrefix):
		return "llm", nil
	case strings.HasPrefix(key, managementKeyPrefix):
		return "management", nil
	default:
		return "", fmt.Errorf("invalid key format: unrecognized prefix")
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
