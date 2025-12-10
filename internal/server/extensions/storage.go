package extensions

import (
	"github.com/r74tech/virga/internal/server/protocol"
)

// Storage interface for extension storage
type Storage interface {
	// Save saves an extension for a session
	Save(sessionID string, manifest protocol.ExtensionManifest, data []byte) error

	// Get retrieves an extension for a session
	Get(sessionID, name string) (protocol.ExtensionManifest, []byte, error)

	// List lists all extensions for a session
	List(sessionID string) ([]protocol.ExtensionManifest, error)

	// Delete deletes an extension for a session
	Delete(sessionID, name string) error

	// DeleteAll deletes all extensions for a session
	DeleteAll(sessionID string) error
}
