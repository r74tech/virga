package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSigning_BasicFlow(t *testing.T) {
	// Create temporary extension directory
	tmpDir := t.TempDir()

	// Create some test files
	files := map[string]string{
		"extension.json": `{"name":"test","version":"1.0.0"}`,
		"script.py":      `print("Hello, World!")`,
		"README.md":      `# Test Extension`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Create signer
	signer, err := NewSigner()
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Sign the extension
	signature, err := signer.SignExtension(tmpDir)
	if err != nil {
		t.Fatalf("Failed to sign extension: %v", err)
	}

	// Verify signature was created
	if signature.Algorithm != SignatureAlgorithmEd25519 {
		t.Errorf("Expected algorithm %s, got %s", SignatureAlgorithmEd25519, signature.Algorithm)
	}

	if len(signature.SignedFiles) != 3 {
		t.Errorf("Expected 3 signed files, got %d", len(signature.SignedFiles))
	}

	// Verify signature file exists
	sigPath := filepath.Join(tmpDir, "signature.json")
	if _, err := os.Stat(sigPath); os.IsNotExist(err) {
		t.Error("Signature file was not created")
	}

	// Create verifier
	verifier := NewVerifier()
	verifier.AddTrustedKey("test-key", signer.GetPublicKey())

	// Verify the extension
	if err := verifier.VerifyExtension(tmpDir); err != nil {
		t.Fatalf("Failed to verify extension: %v", err)
	}
}

func TestSigning_ModifiedFile(t *testing.T) {
	// Create temporary extension directory
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Sign
	signer, _ := NewSigner()
	if _, err := signer.SignExtension(tmpDir); err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// Modify file after signing
	if err := os.WriteFile(testFile, []byte("modified content"), 0o644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Verify should fail
	verifier := NewVerifier()
	verifier.AddTrustedKey("test", signer.GetPublicKey())

	err := verifier.VerifyExtension(tmpDir)
	if err == nil {
		t.Error("Expected verification to fail for modified file")
	}
	if err.Error() != "extension files have been modified" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSigning_UntrustedKey(t *testing.T) {
	// Create temporary extension directory
	tmpDir := t.TempDir()

	// Create test file
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Sign with one key
	signer1, _ := NewSigner()
	if _, err := signer1.SignExtension(tmpDir); err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// Create verifier with different key
	signer2, _ := NewSigner()
	verifier := NewVerifier()
	verifier.AddTrustedKey("different", signer2.GetPublicKey())

	// Verify should fail
	err := verifier.VerifyExtension(tmpDir)
	if err == nil {
		t.Error("Expected verification to fail for untrusted key")
	}
	if err.Error() != "untrusted public key" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSigning_NoSignature(t *testing.T) {
	// Create temporary extension directory without signature
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	verifier := NewVerifier()
	err := verifier.VerifyExtension(tmpDir)
	if err == nil {
		t.Error("Expected verification to fail for missing signature")
	}
}

func TestSigning_KeySerialization(t *testing.T) {
	// Create signer
	signer1, err := NewSigner()
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Get keys
	privKey := signer1.GetPrivateKey()
	pubKey := signer1.GetPublicKey()

	// Create new signer from private key
	signer2, err := NewSignerFromKey(privKey)
	if err != nil {
		t.Fatalf("Failed to create signer from key: %v", err)
	}

	// Public keys should match
	if signer2.GetPublicKey() != pubKey {
		t.Error("Public keys don't match after deserialization")
	}
}
