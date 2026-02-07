package crypto

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// KeyPath returns the path to the age key for the given user
func KeyPath(user string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	keyPath := filepath.Join(home, ".config", "diary", fmt.Sprintf("%s.age.key", user))
	return keyPath, nil
}

// Encrypt encrypts the plaintext using the recipient's public key
func Encrypt(plaintext []byte, recipientPublicKey string) ([]byte, error) {
	// Parse recipient public key
	recipient, err := age.ParseX25519Recipient(recipientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse recipient: %w", err)
	}

	// Create buffer for ciphertext
	ciphertextBuffer := &bytes.Buffer{}

	// Wrap with armor writer for ASCII output
	armoredWriter := armor.NewWriter(ciphertextBuffer)

	// Create encrypt writer
	encryptWriter, err := age.Encrypt(armoredWriter, recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to create encrypt writer: %w", err)
	}

	// Copy plaintext to encrypt writer
	if _, err := io.Copy(encryptWriter, bytes.NewReader(plaintext)); err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	// IMPORTANT: Close encrypt writer first
	if err := encryptWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encrypt writer: %w", err)
	}

	// IMPORTANT: Close armored writer second
	if err := armoredWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close armored writer: %w", err)
	}

	return ciphertextBuffer.Bytes(), nil
}

// Decrypt decrypts the ciphertext using the identity key
func Decrypt(ciphertext []byte, identityKeyPath string) ([]byte, error) {
	// Load identity from key file
	identities, err := loadIdentities(identityKeyPath)
	if err != nil {
		return nil, err
	}

	// Create reader from ciphertext
	var ciphertextReader io.Reader = bytes.NewReader(ciphertext)

	// Check if armored
	if bytes.HasPrefix(ciphertext, []byte(armor.Header)) {
		ciphertextReader = armor.NewReader(ciphertextReader)
	}

	// Decrypt
	plaintextReader, err := age.Decrypt(ciphertextReader, identities...)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Read plaintext to buffer
	plaintextBuffer := &bytes.Buffer{}
	if _, err := io.Copy(plaintextBuffer, plaintextReader); err != nil {
		return nil, fmt.Errorf("failed to read plaintext: %w", err)
	}

	return plaintextBuffer.Bytes(), nil
}

// LoadIdentity loads an age identity from the given key file
func LoadIdentity(keyPath string) (*age.X25519Identity, error) {
	identities, err := loadIdentities(keyPath)
	if err != nil {
		return nil, err
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no identities found in %s", keyPath)
	}

	// Return first identity (we only use one per user)
	identity, ok := identities[0].(*age.X25519Identity)
	if !ok {
		return nil, fmt.Errorf("unexpected identity type: %T", identities[0])
	}

	return identity, nil
}

// GenerateKey generates a new age key pair and writes to writer
func GenerateKey(w io.Writer, user string) error {
	// Generate new X25519 identity
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Get recipient (public key)
	recipient := identity.Recipient()

	// Write formatted output
	fmt.Fprintf(w, "# created: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(w, "# user: %s\n", user)
	fmt.Fprintf(w, "# public key: %s\n", recipient)
	fmt.Fprintf(w, "%s\n", identity)

	return nil
}

// loadIdentities loads identities from a key file
func loadIdentities(keyPath string) ([]age.Identity, error) {
	file, err := os.Open(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()

	identities, err := age.ParseIdentities(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse identities: %w", err)
	}

	return identities, nil
}
