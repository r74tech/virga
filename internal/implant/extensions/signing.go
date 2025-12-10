package extensions

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SignatureAlgorithm represents the signing algorithm
type SignatureAlgorithm string

const (
	SignatureAlgorithmEd25519 SignatureAlgorithm = "ed25519"
)

// ExtensionSignature contains signature information
type ExtensionSignature struct {
	Algorithm   SignatureAlgorithm `json:"algorithm"`
	PublicKey   string             `json:"public_key"` // Base64 encoded
	Signature   string             `json:"signature"`  // Base64 encoded
	Timestamp   int64              `json:"timestamp"`
	SignedFiles []string           `json:"signed_files"` // List of files that were signed
	Hash        string             `json:"hash"`         // SHA256 of all signed content
}

// Signer handles extension signing
type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewSigner creates a new signer with a generated key pair
func NewSigner() (*Signer, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &Signer{
		privateKey: priv,
		publicKey:  pub,
	}, nil
}

// NewSignerFromKey creates a signer from an existing private key
func NewSignerFromKey(privateKeyBase64 string) (*Signer, error) {
	privKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}

	privateKey := ed25519.PrivateKey(privKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// GetPublicKey returns the base64 encoded public key
func (s *Signer) GetPublicKey() string {
	return base64.StdEncoding.EncodeToString(s.publicKey)
}

// GetPrivateKey returns the base64 encoded private key
func (s *Signer) GetPrivateKey() string {
	return base64.StdEncoding.EncodeToString(s.privateKey)
}

// SignExtension signs an extension directory
func (s *Signer) SignExtension(extensionPath string) (*ExtensionSignature, error) {
	// Get list of files to sign
	files, err := s.getFilesToSign(extensionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}

	// Calculate combined hash of all files
	hash, err := s.calculateExtensionHash(extensionPath, files)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Sign the hash
	signature := ed25519.Sign(s.privateKey, hash)

	// Create signature object
	sig := &ExtensionSignature{
		Algorithm:   SignatureAlgorithmEd25519,
		PublicKey:   s.GetPublicKey(),
		Signature:   base64.StdEncoding.EncodeToString(signature),
		Timestamp:   time.Now().Unix(),
		SignedFiles: files,
		Hash:        hex.EncodeToString(hash),
	}

	// Write signature file
	sigPath := filepath.Join(extensionPath, "signature.json")
	sigData, err := json.MarshalIndent(sig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signature: %w", err)
	}

	if err := os.WriteFile(sigPath, sigData, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write signature file: %w", err)
	}

	return sig, nil
}

// getFilesToSign returns list of files that should be signed
func (s *Signer) getFilesToSign(extensionPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(extensionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(extensionPath, path)
		if err != nil {
			return err
		}

		// Skip signature file itself
		if relPath == "signature.json" {
			return nil
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// calculateExtensionHash calculates SHA256 hash of all files
func (s *Signer) calculateExtensionHash(extensionPath string, files []string) ([]byte, error) {
	h := sha256.New()

	// Sort files for consistent hashing
	sort.Strings(files)

	for _, file := range files {
		fullPath := filepath.Join(extensionPath, file)

		// Write filename to hash
		h.Write([]byte(file))

		// Write file content to hash
		f, err := os.Open(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", file, err)
		}

		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to hash %s: %w", file, err)
		}
		f.Close()
	}

	return h.Sum(nil), nil
}

// Verifier handles extension verification
type Verifier struct {
	trustedKeys map[string]ed25519.PublicKey // Map of key ID to public key
}

// NewVerifier creates a new verifier
func NewVerifier() *Verifier {
	return &Verifier{
		trustedKeys: make(map[string]ed25519.PublicKey),
	}
}

// AddTrustedKey adds a trusted public key
func (v *Verifier) AddTrustedKey(keyID, publicKeyBase64 string) error {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size")
	}

	v.trustedKeys[keyID] = ed25519.PublicKey(pubKeyBytes)
	return nil
}

// VerifyExtension verifies an extension's signature
func (v *Verifier) VerifyExtension(extensionPath string) error {
	// Read signature file
	sigPath := filepath.Join(extensionPath, "signature.json")
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Errorf("signature file not found: %w", err)
	}

	var sig ExtensionSignature
	if err := json.Unmarshal(sigData, &sig); err != nil {
		return fmt.Errorf("failed to parse signature: %w", err)
	}

	// Check if we trust this public key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(sig.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	var trusted bool
	for _, trustedKey := range v.trustedKeys {
		if bytes.Equal(trustedKey, pubKeyBytes) {
			trusted = true
			break
		}
	}

	if !trusted {
		return fmt.Errorf("untrusted public key")
	}

	// Calculate current hash
	hash, err := v.calculateExtensionHash(extensionPath, sig.SignedFiles)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Verify hash matches
	expectedHash, err := hex.DecodeString(sig.Hash)
	if err != nil {
		return fmt.Errorf("failed to decode hash: %w", err)
	}

	if !bytes.Equal(hash, expectedHash) {
		return fmt.Errorf("extension files have been modified")
	}

	// Verify signature
	sigBytes, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	publicKey := ed25519.PublicKey(pubKeyBytes)
	if !ed25519.Verify(publicKey, hash, sigBytes) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// calculateExtensionHash calculates hash for verification (same as signer)
func (v *Verifier) calculateExtensionHash(extensionPath string, files []string) ([]byte, error) {
	h := sha256.New()

	// Sort files for consistent hashing
	sort.Strings(files)

	for _, file := range files {
		fullPath := filepath.Join(extensionPath, file)

		// Write filename to hash
		h.Write([]byte(file))

		// Write file content to hash
		f, err := os.Open(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", file, err)
		}

		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to hash %s: %w", file, err)
		}
		f.Close()
	}

	return h.Sum(nil), nil
}
