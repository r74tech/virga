package command

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
)

// UploadCommand is the implementation of the 'upload' command.
type UploadCommand struct{}

// Execute executes the 'upload' command.
func (c *UploadCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	if len(args) < 2 {
		return fmt.Errorf("source and destination paths required. Usage: upload <local_path> <remote_path>")
	}

	localPath := args[0]
	remotePath := args[1]

	// Validate local path
	if localPath == "" {
		return fmt.Errorf("local path cannot be empty")
	}

	// Security: Validate local path to prevent directory traversal
	if err := validateLocalPath(localPath); err != nil {
		return fmt.Errorf("invalid local path: %w", err)
	}

	// Check if local file exists
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("local file does not exist: %s", localPath)
		}
		return fmt.Errorf("failed to stat local file: %v", err)
	}

	// Check if it's a regular file
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("local path is not a regular file: %s", localPath)
	}

	// Validate remote path
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	// Read the local file
	fileContent, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %v", err)
	}

	fileSize := fileInfo.Size()
	fileName := filepath.Base(localPath)

	logger.Info("Uploading %s (%d bytes) to %s...", fileName, fileSize, remotePath)

	// Encode file content to base64
	encodedContent := base64.StdEncoding.EncodeToString(fileContent)

	// Send the upload request with args in the expected format
	taskID, err := currentSession.ExecuteCommand("upload", map[string]interface{}{
		"args": []string{remotePath, encodedContent},
	})
	if err != nil {
		return fmt.Errorf("error sending upload request: %v", err)
	}

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 120)
	if err != nil {
		return fmt.Errorf("error waiting for upload result: %v", err)
	}

	// Display the result
	if result.ExitCode == 0 {
		logger.Info("Upload complete: %s -> %s", localPath, remotePath)
	} else {
		logger.Error("Upload failed: %s", result.Error)
	}

	return nil
}

// Help returns the help for the 'upload' command.
func (c *UploadCommand) Help() string {
	return "Upload a file to the target system.\n" +
		"Usage: upload <local_path> <remote_path>\n" +
		"  local_path    Path to the local file to upload\n" +
		"  remote_path   Destination path on the target system\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// DownloadCommand is the implementation of the 'download' command.
type DownloadCommand struct{}

// Execute executes the 'download' command.
func (c *DownloadCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	if len(args) < 2 {
		return fmt.Errorf("remote and local paths required. Usage: download <remote_path> <local_path>")
	}

	remotePath := args[0]
	localPath := args[1]

	// Validate remote path
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	// Validate local path
	if localPath == "" {
		return fmt.Errorf("local path cannot be empty")
	}

	// Security: Validate local path to prevent directory traversal
	if err := validateLocalPath(localPath); err != nil {
		return fmt.Errorf("invalid local path: %w", err)
	}

	// Check if local directory exists for the target file
	localDir := filepath.Dir(localPath)
	if localDir != "." && localDir != "" {
		if _, err := os.Stat(localDir); os.IsNotExist(err) {
			return fmt.Errorf("local directory does not exist: %s", localDir)
		}
	}

	logger.Info("Downloading %s to %s...", remotePath, localPath)

	// Send the download request
	taskID, err := currentSession.ExecuteCommand("download", map[string]interface{}{
		"remote_path": remotePath,
	})
	if err != nil {
		return fmt.Errorf("error sending download request: %v", err)
	}

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 120)
	if err != nil {
		return fmt.Errorf("error waiting for download result: %v", err)
	}

	// Check the result
	if result.ExitCode != 0 {
		return fmt.Errorf("download failed: %s", result.Error)
	}

	// Get the file data (Base64 encoded)
	if result.Output == "" {
		return fmt.Errorf("invalid download response: empty output")
	}

	// Decode the Base64 output
	fileData, err := base64.StdEncoding.DecodeString(result.Output)
	if err != nil {
		return fmt.Errorf("failed to decode file data: %v", err)
	}

	if len(fileData) == 0 {
		return fmt.Errorf("invalid download response: empty file data")
	}

	// Save to a local file
	if err := os.WriteFile(localPath, fileData, 0o644); err != nil {
		return fmt.Errorf("failed to save file locally: %v", err)
	}

	// Check the size
	fileSize := len(fileData)
	logger.Info("Download complete: %s -> %s (%d bytes)", remotePath, localPath, fileSize)

	return nil
}

// Help returns the help for the 'download' command.
func (c *DownloadCommand) Help() string {
	return "Download a file from the target system.\n" +
		"Usage: download <remote_path> <local_path>\n" +
		"  remote_path   Path to the file on the target system\n" +
		"  local_path    Destination path on the local system\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// LsCommand is the implementation of the 'ls' command.
type LsCommand struct{}

// Execute executes the 'ls' command.
func (c *LsCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Set the default path
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Send the request
	taskID, err := currentSession.ExecuteCommand("ls", map[string]interface{}{
		"path": path,
	})
	if err != nil {
		return fmt.Errorf("error sending ls request: %v", err)
	}

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 30)
	if err != nil {
		return fmt.Errorf("error waiting for ls result: %v", err)
	}

	// Display the result
	logger.Info("Directory listing for %s:", path)
	logger.Info(result.Output)

	return nil
}

// Help returns the help for the 'ls' command.
func (c *LsCommand) Help() string {
	return "List files and directories on the target system.\n" +
		"Usage: ls [path]\n" +
		"  path    Optional path to list (defaults to current directory)\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// CdCommand is the implementation of the 'cd' command.
type CdCommand struct{}

// Execute executes the 'cd' command.
func (c *CdCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	if len(args) == 0 {
		return fmt.Errorf("directory path required. Usage: cd <path>")
	}

	path := args[0]

	// Send the request
	taskID, err := currentSession.ExecuteCommand("cd", map[string]interface{}{
		"path": path,
	})
	if err != nil {
		return fmt.Errorf("error sending cd request: %v", err)
	}

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 10)
	if err != nil {
		return fmt.Errorf("error waiting for cd result: %v", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("cd failed: %s", result.Error)
	}

	logger.Info("Changed working directory to: %s", result.Output)

	return nil
}

// Help returns the help for the 'cd' command.
func (c *CdCommand) Help() string {
	return "Change the working directory on the target system.\n" +
		"Usage: cd <path>\n" +
		"  path    Directory path to change to\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// PwdCommand is the implementation of the 'pwd' command.
type PwdCommand struct{}

// Execute executes the 'pwd' command.
func (c *PwdCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Send the request
	taskID, err := currentSession.ExecuteCommand("pwd", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("error sending pwd request: %v", err)
	}

	logger.Debug("Task ID: %s - Executing: pwd", taskID)

	// Wait for the result (increased timeout)
	result, err := waitForTaskResult(currentSession, taskID, 30)
	if err != nil {
		return fmt.Errorf("error waiting for pwd result: %v", err)
	}

	// Display the result
	logger.Info("Current directory:")
	logger.Info("------------------------------------------------------------")
	logger.Info("%s", result.Output)
	logger.Info("------------------------------------------------------------")
	logger.Debug("Exit code: %d", result.ExitCode)

	return nil
}

// Help returns the help for the 'pwd' command.
func (c *PwdCommand) Help() string {
	return "Print the current working directory on the target system.\n" +
		"Usage: pwd\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}
