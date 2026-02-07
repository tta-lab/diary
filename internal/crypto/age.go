package crypto

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
// TODO: Implement using filippo.io/age
func Encrypt(plaintext []byte, recipientPublicKey string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Decrypt decrypts the ciphertext using the identity key
// TODO: Implement using filippo.io/age
func Decrypt(ciphertext []byte, identityKeyPath string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// LoadIdentity loads an age identity from the given key file
// TODO: Implement using filippo.io/age
func LoadIdentity(keyPath string) (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateKey generates a new age key pair
// TODO: Implement using filippo.io/age
func GenerateKey(w io.Writer) error {
	return fmt.Errorf("not implemented")
}
