package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate test identity
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	// Get recipient (public key)
	recipient := identity.Recipient().String()

	// Create temp key file
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.age.key")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	if err := GenerateKey(keyFile, "testuser"); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	keyFile.Close()

	// Load the identity we just created
	loadedIdentity, err := LoadIdentity(keyPath)
	if err != nil {
		t.Fatalf("failed to load identity: %v", err)
	}

	// Use loaded identity's recipient for encryption
	recipient = loadedIdentity.Recipient().String()

	// Test data
	plaintext := []byte("This is a test diary entry.\n\nSecond paragraph with secrets!")

	// Encrypt
	ciphertext, err := Encrypt(plaintext, recipient)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Verify ciphertext is not empty
	if len(ciphertext) == 0 {
		t.Fatal("ciphertext is empty")
	}

	// Verify ciphertext is different from plaintext
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext equals plaintext (not encrypted)")
	}

	// Decrypt
	decrypted, err := Decrypt(ciphertext, keyPath)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	// Verify decrypted matches original
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted text doesn't match original.\nOriginal: %s\nDecrypted: %s", plaintext, decrypted)
	}
}

func TestEncryptDecryptLargeText(t *testing.T) {
	// Generate test identity
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	recipient := identity.Recipient().String()

	// Create temp key file
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.age.key")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	if err := GenerateKey(keyFile, "testuser"); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	keyFile.Close()

	// Load identity
	loadedIdentity, err := LoadIdentity(keyPath)
	if err != nil {
		t.Fatalf("failed to load identity: %v", err)
	}
	recipient = loadedIdentity.Recipient().String()

	// Large test data (simulating long diary entry)
	plaintext := bytes.Repeat([]byte("Long diary entry paragraph. "), 1000)

	// Encrypt
	ciphertext, err := Encrypt(plaintext, recipient)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Decrypt
	decrypted, err := Decrypt(ciphertext, keyPath)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	// Verify
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("decrypted large text doesn't match original")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	// Create temp key files
	tmpDir := t.TempDir()

	// Key file 1
	keyPath1 := filepath.Join(tmpDir, "key1.age.key")
	keyFile1, _ := os.Create(keyPath1)
	GenerateKey(keyFile1, "user1")
	keyFile1.Close()

	// Key file 2
	keyPath2 := filepath.Join(tmpDir, "key2.age.key")
	keyFile2, _ := os.Create(keyPath2)
	GenerateKey(keyFile2, "user2")
	keyFile2.Close()

	// Load identity 1
	loadedIdentity1, _ := LoadIdentity(keyPath1)
	recipient1 := loadedIdentity1.Recipient().String()

	// Encrypt with identity 1
	plaintext := []byte("Secret message")
	ciphertext, err := Encrypt(plaintext, recipient1)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Try to decrypt with identity 2 (should fail)
	_, err = Decrypt(ciphertext, keyPath2)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key, but it succeeded")
	}

	// Verify decrypt works with correct key
	decrypted, err := Decrypt(ciphertext, keyPath1)
	if err != nil {
		t.Fatalf("failed to decrypt with correct key: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("decrypted text doesn't match with correct key")
	}
}
