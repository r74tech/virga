package extensions

import (
	"fmt"
	"sync"

	"github.com/r74tech/virga/internal/server/protocol"
)

// extensionData holds extension manifest and binary data
type extensionData struct {
	Manifest protocol.ExtensionManifest
	Data     []byte
}

// MemoryStorage implements Storage using in-memory storage
type MemoryStorage struct {
	extensions map[string]map[string]*extensionData // sessionID -> extensionName -> data
	mu         sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		extensions: make(map[string]map[string]*extensionData),
	}
}

// Save saves an extension for a session
func (ms *MemoryStorage) Save(sessionID string, manifest protocol.ExtensionManifest, data []byte) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Initialize session map if needed
	if _, exists := ms.extensions[sessionID]; !exists {
		ms.extensions[sessionID] = make(map[string]*extensionData)
	}

	// Store extension data
	ms.extensions[sessionID][manifest.Name] = &extensionData{
		Manifest: manifest,
		Data:     data,
	}

	return nil
}

// Get retrieves an extension for a session
func (ms *MemoryStorage) Get(sessionID, name string) (protocol.ExtensionManifest, []byte, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	sessionExts, exists := ms.extensions[sessionID]
	if !exists {
		return protocol.ExtensionManifest{}, nil, fmt.Errorf("no extensions for session: %s", sessionID)
	}

	extData, exists := sessionExts[name]
	if !exists {
		return protocol.ExtensionManifest{}, nil, fmt.Errorf("extension not found: %s", name)
	}

	return extData.Manifest, extData.Data, nil
}

// List lists all extensions for a session
func (ms *MemoryStorage) List(sessionID string) ([]protocol.ExtensionManifest, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var manifests []protocol.ExtensionManifest

	sessionExts, exists := ms.extensions[sessionID]
	if !exists {
		return manifests, nil // Return empty list
	}

	for _, extData := range sessionExts {
		manifests = append(manifests, extData.Manifest)
	}

	return manifests, nil
}

// Delete deletes an extension for a session
func (ms *MemoryStorage) Delete(sessionID, name string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	sessionExts, exists := ms.extensions[sessionID]
	if !exists {
		return fmt.Errorf("no extensions for session: %s", sessionID)
	}

	if _, exists := sessionExts[name]; !exists {
		return fmt.Errorf("extension not found: %s", name)
	}

	delete(sessionExts, name)

	// Clean up empty session map
	if len(sessionExts) == 0 {
		delete(ms.extensions, sessionID)
	}

	return nil
}

// DeleteAll deletes all extensions for a session
func (ms *MemoryStorage) DeleteAll(sessionID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.extensions, sessionID)
	return nil
}
