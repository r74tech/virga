//go:build llama_embed || llama_external || llama_selfextract
// +build llama_embed llama_external llama_selfextract

package llama

import (
	"sync"

	"github.com/r74tech/virga/internal/implant/memdb"
)

var (
	globalIntegration *LlamaIntegration
	integrationMutex  sync.RWMutex
)

// RegisterLlamaIntegration registers the Llama integration globally
func RegisterLlamaIntegration(integration *LlamaIntegration) {
	integrationMutex.Lock()
	defer integrationMutex.Unlock()

	if globalIntegration != nil {
		globalIntegration.Close()
	}

	globalIntegration = integration
}

// GetLlamaIntegration returns the global Llama integration instance
func GetLlamaIntegration() *LlamaIntegration {
	integrationMutex.RLock()
	defer integrationMutex.RUnlock()

	return globalIntegration
}

// IsLlamaAvailable checks if Llama integration is available
func IsLlamaAvailable() bool {
	integrationMutex.RLock()
	defer integrationMutex.RUnlock()

	return globalIntegration != nil
}

// SetMemDB sets the memdb instance for the global Llama integration
func SetMemDB(db *memdb.DB) {
	integrationMutex.Lock()
	defer integrationMutex.Unlock()

	if globalIntegration != nil {
		globalIntegration.SetMemDB(db)
	}
}
