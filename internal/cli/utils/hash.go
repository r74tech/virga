package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

// HashType represents the type of hash algorithm
type HashType string

const (
	HashMD5    HashType = "md5"
	HashSHA1   HashType = "sha1"
	HashSHA256 HashType = "sha256"
	HashSHA512 HashType = "sha512"
)

// CalculateFileHash calculates the hash of a file using the specified algorithm
func CalculateFileHash(path string, hashType HashType) (string, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return "", newFileError("hash", path, fmt.Errorf("open file: %w", err))
	}
	defer file.Close()

	// Create the appropriate hasher
	var hasher hash.Hash
	switch hashType {
	case HashMD5:
		hasher = md5.New()
	case HashSHA1:
		hasher = sha1.New()
	case HashSHA256:
		hasher = sha256.New()
	case HashSHA512:
		hasher = sha512.New()
	default:
		return "", newFileError("hash", path, fmt.Errorf("unsupported hash type: %s", hashType))
	}

	// Copy file content to hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", newFileError("hash", path, fmt.Errorf("read file: %w", err))
	}

	// Return hex-encoded hash
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// CalculateFileHashSHA256 is a convenience function for SHA256
func CalculateFileHashSHA256(path string) (string, error) {
	return CalculateFileHash(path, HashSHA256)
}

// VerifyFileHash verifies a file matches the expected hash
func VerifyFileHash(path string, expectedHash string, hashType HashType) (bool, error) {
	actualHash, err := CalculateFileHash(path, hashType)
	if err != nil {
		return false, err
	}

	return actualHash == expectedHash, nil
}

// HashReader calculates the hash of data from a reader
func HashReader(r io.Reader, hashType HashType) (string, error) {
	var hasher hash.Hash
	switch hashType {
	case HashMD5:
		hasher = md5.New()
	case HashSHA1:
		hasher = sha1.New()
	case HashSHA256:
		hasher = sha256.New()
	case HashSHA512:
		hasher = sha512.New()
	default:
		return "", fmt.Errorf("unsupported hash type: %s", hashType)
	}

	if _, err := io.Copy(hasher, r); err != nil {
		return "", fmt.Errorf("read data: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashBytes calculates the hash of byte data
func HashBytes(data []byte, hashType HashType) string {
	var hasher hash.Hash
	switch hashType {
	case HashMD5:
		hasher = md5.New()
	case HashSHA1:
		hasher = sha1.New()
	case HashSHA256:
		hasher = sha256.New()
	case HashSHA512:
		hasher = sha512.New()
	default:
		// Default to SHA256 if invalid type
		hasher = sha256.New()
	}

	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}
