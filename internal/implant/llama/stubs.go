//go:build !llama_embed && !llama_external && !llama_selfextract
// +build !llama_embed,!llama_external,!llama_selfextract

package llama

import (
	"github.com/r74tech/virga/internal/implant/memdb"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// Stub implementations when Llama is not enabled

type LlamaIntegration struct{}

func RegisterLlamaIntegration(integration *LlamaIntegration) {
	// No-op when Llama is disabled
}

func GetLlamaIntegration() *LlamaIntegration {
	return nil
}

func IsLlamaAvailable() bool {
	return false
}

func SetMemDB(db *memdb.DB) {
	// No-op when Llama is disabled
}

func (li *LlamaIntegration) SetServerInfo(serverURL, sessionID string) {
	// No-op when Llama is disabled
}

func (li *LlamaIntegration) SetPartialResultCallback(callback func(taskID, output string)) {
	// No-op when Llama is disabled
}

func (li *LlamaIntegration) SetFinalResultCallback(callback func(taskID, output string)) {
	// No-op when Llama is disabled
}

func (li *LlamaIntegration) ProcessRequest(request protocol.Request) protocol.Response {
	return protocol.Response{
		RequestID: request.ID,
		Status:    protocol.StatusError,
		Error:     "Llama integration is not enabled in this build",
	}
}
