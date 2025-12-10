package extensions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
)

// Manager manages extensions for sessions
type Manager struct {
	db             *database.Database
	storage        Storage
	sessionManager *session.Manager
	mu             sync.RWMutex
}

// NewManager creates a new extension manager
func NewManager(db *database.Database, storage Storage, sessionManager *session.Manager) *Manager {
	return &Manager{
		db:             db,
		storage:        storage,
		sessionManager: sessionManager,
	}
}

// UploadExtension uploads an extension for a session
func (m *Manager) UploadExtension(sessionID string, manifest protocol.ExtensionManifest, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify session exists
	sess := m.sessionManager.GetSession(sessionID)
	if sess == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Calculate checksum
	hash := sha256.Sum256(data)
	checksum := hex.EncodeToString(hash[:])

	// Save to storage
	if err := m.storage.Save(sessionID, manifest, data); err != nil {
		return fmt.Errorf("failed to save extension: %w", err)
	}

	// Create upload task for implant
	uploadTask := protocol.ExtensionUploadTask{
		Manifest: manifest,
		Data:     data,
		Checksum: checksum,
	}

	task := protocol.Task{
		ID:        generateTaskID(),
		Type:      protocol.TaskTypeExtensionUpload,
		SessionID: sessionID,
		Status:    protocol.TaskStatusPending,
		CreatedAt: time.Now(),
		Payload:   uploadTask,
	}

	// Add task to session
	sess.AddTask(task)

	// Log to database
	if m.db != nil {
		// TODO: Log extension upload to database
	}

	return nil
}

// ListExtensions lists extensions for a session
func (m *Manager) ListExtensions(sessionID string) ([]protocol.ExtensionManifest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.storage.List(sessionID)
}

// GetExtension gets a specific extension for a session
func (m *Manager) GetExtension(sessionID, name string) (protocol.ExtensionManifest, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.storage.Get(sessionID, name)
}

// DeleteExtension deletes an extension for a session
func (m *Manager) DeleteExtension(sessionID, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify session exists
	sess := m.sessionManager.GetSession(sessionID)
	if sess == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Delete from storage
	if err := m.storage.Delete(sessionID, name); err != nil {
		return fmt.Errorf("failed to delete extension: %w", err)
	}

	// Create unload task for implant
	task := protocol.Task{
		ID:        generateTaskID(),
		Type:      protocol.TaskTypeExtensionUnload,
		SessionID: sessionID,
		Status:    protocol.TaskStatusPending,
		CreatedAt: time.Now(),
		Payload: map[string]string{
			"extension_name": name,
		},
	}

	// Add task to session
	sess.AddTask(task)

	return nil
}

// ExecuteExtension creates an extension execution task
func (m *Manager) ExecuteExtension(sessionID, extensionName, command string, args map[string]interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Verify session exists
	sess := m.sessionManager.GetSession(sessionID)
	if sess == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Verify extension exists
	_, _, err := m.storage.Get(sessionID, extensionName)
	if err != nil {
		return fmt.Errorf("extension not found: %s", extensionName)
	}

	// Create execute task
	executeTask := protocol.ExtensionExecuteTask{
		ExtensionName: extensionName,
		Command:       command,
		Arguments:     args,
	}

	task := protocol.Task{
		ID:        generateTaskID(),
		Type:      protocol.TaskTypeExtensionExecute,
		SessionID: sessionID,
		Status:    protocol.TaskStatusPending,
		CreatedAt: time.Now(),
		Payload:   executeTask,
	}

	// Add task to session
	sess.AddTask(task)

	return nil
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("ext-%d", time.Now().UnixNano())
}
