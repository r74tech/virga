package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// normalizeKey standardizes the key for AES encryption to 32 bytes (256 bits)
// SECURITY: This function uses SHA-256 as a simple key derivation function.
// For production use, consider using a proper KDF like PBKDF2, scrypt, or Argon2.
func normalizeKey(key []byte) []byte {
	// If it's exactly 32 bytes, no change is needed
	if len(key) == 32 {
		return key
	}

	// Convert to 32-byte key using SHA-256 hash
	// WARNING: This is not a proper KDF and should be replaced in production
	hasher := sha256.New()
	hasher.Write(key)
	return hasher.Sum(nil)
}

// Encrypt encrypts the given data using AES-256-GCM (Galois/Counter Mode)
// The function returns the ciphertext with the nonce prepended.
// Format: [nonce(12 bytes)][ciphertext+tag]
func Encrypt(data []byte, key []byte) ([]byte, error) {
	// Standardize the key
	normalizedKey := normalizeKey(key)

	// Create AES block
	block, err := aes.NewCipher(normalizedKey)
	if err != nil {
		return nil, fmt.Errorf("could not create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("could not generate nonce: %w", err)
	}

	// Encrypt
	encrypted := gcm.Seal(nonce, nonce, data, nil)
	return encrypted, nil
}

// Decrypt decrypts the given encrypted data using AES-256-GCM
// The function expects the ciphertext format: [nonce(12 bytes)][ciphertext+tag]
// It verifies the authentication tag to ensure data integrity.
func Decrypt(data []byte, key []byte) ([]byte, error) {
	// Standardize the key
	normalizedKey := normalizeKey(key)

	// Create AES block
	block, err := aes.NewCipher(normalizedKey)
	if err != nil {
		return nil, fmt.Errorf("could not create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create GCM: %w", err)
	}

	// Verify nonce size
	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("malformed ciphertext")
	}

	// Get nonce and decrypt
	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt data: %w", err)
	}

	return plaintext, nil
}
