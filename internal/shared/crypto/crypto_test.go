package crypto

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
)

// TestEncryptDecrypt tests the basic encryption and decryption functionality
func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
		key       []byte
		wantErr   bool
	}{
		{
			name:      "valid encryption with 32-byte key",
			plaintext: []byte("Hello, World!"),
			key:       []byte("this-is-a-32-byte-key-for-aes256"),
			wantErr:   false,
		},
		{
			name:      "valid encryption with short key",
			plaintext: []byte("Test data"),
			key:       []byte("short-key"),
			wantErr:   false,
		},
		{
			name:      "empty plaintext",
			plaintext: []byte{},
			key:       []byte("test-key"),
			wantErr:   false,
		},
		{
			name:      "large plaintext",
			plaintext: make([]byte, 1024*1024), // 1MB
			key:       []byte("test-key"),
			wantErr:   false,
		},
		{
			name:      "unicode plaintext",
			plaintext: []byte("hello world 🌍"),
			key:       []byte("test-key"),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill large plaintext with random data
			if len(tt.plaintext) == 1024*1024 {
				rand.Read(tt.plaintext)
			}

			// Encrypt
			encrypted, err := Encrypt(tt.plaintext, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify encrypted data is different from plaintext
			if len(tt.plaintext) > 0 && bytes.Equal(encrypted, tt.plaintext) {
				t.Error("Encrypted data should be different from plaintext")
			}

			// Decrypt
			decrypted, err := Decrypt(encrypted, tt.key)
			if err != nil {
				t.Errorf("Decrypt() error = %v", err)
				return
			}

			// Verify decrypted data matches original
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

// TestDecryptInvalidData tests decryption with invalid data
func TestDecryptInvalidData(t *testing.T) {
	key := []byte("test-key")

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "too short data",
			data:    []byte("short"),
			wantErr: true,
		},
		{
			name:    "corrupted nonce",
			data:    make([]byte, 20),
			wantErr: true,
		},
		{
			name: "corrupted ciphertext",
			data: func() []byte {
				encrypted, _ := Encrypt([]byte("test"), key)
				// Corrupt the last byte (part of auth tag)
				encrypted[len(encrypted)-1] ^= 0xFF
				return encrypted
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.data, key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNormalizeKey tests key normalization function
func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name       string
		key        []byte
		wantLength int
	}{
		{
			name:       "32-byte key unchanged",
			key:        []byte("this-is-a-32-byte-key-for-aes256"),
			wantLength: 32,
		},
		{
			name:       "short key normalized",
			key:        []byte("short"),
			wantLength: 32,
		},
		{
			name:       "long key normalized",
			key:        []byte("this-is-a-very-long-key-that-is-more-than-32-bytes-long"),
			wantLength: 32,
		},
		{
			name:       "empty key normalized",
			key:        []byte{},
			wantLength: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeKey(tt.key)
			if len(normalized) != tt.wantLength {
				t.Errorf("normalizeKey() length = %v, want %v", len(normalized), tt.wantLength)
			}

			// Verify same input produces same output (deterministic)
			normalized2 := normalizeKey(tt.key)
			if !bytes.Equal(normalized, normalized2) {
				t.Error("normalizeKey() should be deterministic")
			}
		})
	}
}

// TestKeyCompatibility tests that different keys produce different results
func TestKeyCompatibility(t *testing.T) {
	plaintext := []byte("sensitive data")
	key1 := []byte("key1")
	key2 := []byte("key2")

	encrypted1, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Failed to encrypt with key1: %v", err)
	}

	encrypted2, err := Encrypt(plaintext, key2)
	if err != nil {
		t.Fatalf("Failed to encrypt with key2: %v", err)
	}

	// Different keys should produce different ciphertext
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("Different keys should produce different ciphertext")
	}

	// Cannot decrypt with wrong key
	_, err = Decrypt(encrypted1, key2)
	if err == nil {
		t.Error("Should not be able to decrypt with wrong key")
	}
}

// TestNonceUniqueness tests that each encryption produces unique nonce
func TestNonceUniqueness(t *testing.T) {
	plaintext := []byte("test data")
	key := []byte("test-key")

	// Encrypt same data multiple times
	var nonces [][]byte
	for i := 0; i < 100; i++ {
		encrypted, err := Encrypt(plaintext, key)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		// Extract nonce (first 12 bytes)
		if len(encrypted) >= 12 {
			nonce := encrypted[:12]
			// Check if this nonce already exists
			for _, prevNonce := range nonces {
				if bytes.Equal(nonce, prevNonce) {
					t.Error("Duplicate nonce detected")
					return
				}
			}
			nonces = append(nonces, nonce)
		}
	}
}

// BenchmarkEncrypt benchmarks encryption performance
func BenchmarkEncrypt(b *testing.B) {
	sizes := []int{64, 1024, 16384, 1048576} // 64B, 1KB, 16KB, 1MB
	key := []byte("benchmark-key-32-bytes-long!!!!!!")

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := Encrypt(data, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDecrypt benchmarks decryption performance
func BenchmarkDecrypt(b *testing.B) {
	sizes := []int{64, 1024, 16384, 1048576} // 64B, 1KB, 16KB, 1MB
	key := []byte("benchmark-key-32-bytes-long!!!!!!")

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)

			encrypted, err := Encrypt(data, key)
			if err != nil {
				b.Fatal(err)
			}

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := Decrypt(encrypted, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
