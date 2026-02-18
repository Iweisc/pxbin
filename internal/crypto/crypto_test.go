package crypto

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := DeriveKey("test-passphrase")
	plaintext := []byte("sk-my-secret-api-key-12345")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := DeriveKey("test-passphrase")
	plaintext := []byte("secret-data")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with the ciphertext
	tampered := []byte(ciphertext)
	tampered[len(tampered)/2] ^= 0xFF
	_, err = Decrypt(string(tampered), key)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := DeriveKey("passphrase-one")
	key2 := DeriveKey("passphrase-two")

	ciphertext, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("expected error for wrong key")
	}
}

func TestEncryptEmptyInput(t *testing.T) {
	key := DeriveKey("passphrase")

	ciphertext, err := Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty, got %q", decrypted)
	}
}

func TestIsEncrypted(t *testing.T) {
	key := DeriveKey("passphrase")
	ciphertext, _ := Encrypt([]byte("test"), key)

	if !IsEncrypted(ciphertext) {
		t.Error("expected encrypted ciphertext to be detected")
	}

	if IsEncrypted("sk-plaintext-api-key") {
		t.Error("expected plaintext key to not be detected as encrypted")
	}

	if IsEncrypted("") {
		t.Error("expected empty string to not be detected as encrypted")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	k1 := DeriveKey("same-passphrase")
	k2 := DeriveKey("same-passphrase")
	if string(k1) != string(k2) {
		t.Error("same passphrase should produce same key")
	}
}
