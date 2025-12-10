package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/r74tech/virga/internal/server/protocol"
)

// FileSystemStorage implements Storage using the file system
type FileSystemStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileSystemStorage creates a new file system storage
func NewFileSystemStorage(basePath string) (*FileSystemStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &FileSystemStorage{
		basePath: basePath,
	}, nil
}

// Save saves an extension for a session
func (fs *FileSystemStorage) Save(sessionID string, manifest protocol.ExtensionManifest, data []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Create session directory
	sessionDir := filepath.Join(fs.basePath, sessionID)
	if err := os.MkdirAll(sessionDir, 0o750); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Create extension directory
	extDir := filepath.Join(sessionDir, manifest.Name)
	if err := os.MkdirAll(extDir, 0o750); err != nil {
		return fmt.Errorf("failed to create extension directory: %w", err)
	}

	// Save manifest
	manifestPath := filepath.Join(extDir, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0o640); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Save binary data
	binaryPath := filepath.Join(extDir, "data.bin")
	if err := os.WriteFile(binaryPath, data, 0o640); err != nil {
		return fmt.Errorf("failed to write binary data: %w", err)
	}

	return nil
}

// Get retrieves an extension for a session
func (fs *FileSystemStorage) Get(sessionID, name string) (protocol.ExtensionManifest, []byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var manifest protocol.ExtensionManifest

	extDir := filepath.Join(fs.basePath, sessionID, name)

	// Check if extension exists
	if _, err := os.Stat(extDir); os.IsNotExist(err) {
		return manifest, nil, fmt.Errorf("extension not found: %s", name)
	}

	// Read manifest
	manifestPath := filepath.Join(extDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return manifest, nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return manifest, nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// Read binary data
	binaryPath := filepath.Join(extDir, "data.bin")
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return manifest, nil, fmt.Errorf("failed to read binary data: %w", err)
	}

	return manifest, data, nil
}

// List lists all extensions for a session
func (fs *FileSystemStorage) List(sessionID string) ([]protocol.ExtensionManifest, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var manifests []protocol.ExtensionManifest

	sessionDir := filepath.Join(fs.basePath, sessionID)

	// Check if session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return manifests, nil // Return empty list if no extensions
	}

	// Read all extension directories
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Read manifest
		manifestPath := filepath.Join(sessionDir, entry.Name(), "manifest.json")
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // Skip if can't read manifest
		}

		var manifest protocol.ExtensionManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			continue // Skip if can't parse manifest
		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// Delete deletes an extension for a session
func (fs *FileSystemStorage) Delete(sessionID, name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	extDir := filepath.Join(fs.basePath, sessionID, name)

	// Check if extension exists
	if _, err := os.Stat(extDir); os.IsNotExist(err) {
		return fmt.Errorf("extension not found: %s", name)
	}

	// Remove extension directory
	if err := os.RemoveAll(extDir); err != nil {
		return fmt.Errorf("failed to delete extension: %w", err)
	}

	return nil
}

// DeleteAll deletes all extensions for a session
func (fs *FileSystemStorage) DeleteAll(sessionID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	sessionDir := filepath.Join(fs.basePath, sessionID)

	// Check if session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil // Nothing to delete
	}

	// Remove session directory
	if err := os.RemoveAll(sessionDir); err != nil {
		return fmt.Errorf("failed to delete session extensions: %w", err)
	}

	return nil
}
