package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ServerEncryptor handles server-side encryption for data at rest
// NOTE: This is NOT zero-knowledge encryption. For true zero-knowledge,
// encryption should happen on the client side only.
// This is a temporary solution to ensure data is encrypted in the database.
type ServerEncryptor struct {
	key []byte
}

// NewServerEncryptor creates a new server encryptor
// It uses a server-side key from environment or generates one
func NewServerEncryptor() (*ServerEncryptor, error) {
	// Try to get key from environment
	keyHex := os.Getenv("GOPHKEEPER_SERVER_KEY")
	var key []byte

	if keyHex != "" {
		var err error
		key, err = hex.DecodeString(keyHex)
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("invalid server key in environment (must be 32 bytes hex)")
		}
	} else {
		// Generate a key from a default passphrase (NOT SECURE FOR PRODUCTION)
		// In production, use a proper key management system
		passphrase := os.Getenv("GOPHKEEPER_SERVER_PASSPHRASE")
		if passphrase == "" {
			passphrase = "default-server-passphrase-change-in-production"
		}
		hash := sha256.Sum256([]byte(passphrase))
		key = hash[:]
	}

	return &ServerEncryptor{
		key: key,
	}, nil
}

// Encrypt encrypts data using AES-256-GCM
func (e *ServerEncryptor) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts data encrypted with Encrypt
func (e *ServerEncryptor) Decrypt(ciphertextHex string) ([]byte, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}
