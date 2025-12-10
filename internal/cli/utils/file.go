package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// FileExists checks if a file exists and is a regular file
func FileExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Return true only if it's a regular file
	return info.Mode().IsRegular()
}

// DirectoryExists checks if a directory exists
func DirectoryExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// IsDirectory checks if a path is a directory
func IsDirectory(path string) bool {
	return DirectoryExists(path)
}

// IsFile checks if a path is a regular file
func IsFile(path string) bool {
	return FileExists(path)
}

// CreateDirectoryIfNotExists creates a directory if it doesn't exist
func CreateDirectoryIfNotExists(path string) error {
	if path == "" {
		return ErrEmptyPath
	}

	// Check if it already exists
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return newFileError("mkdir", path, ErrNotADirectory)
		}
		return nil
	}

	// Create the directory with secure permissions
	if err := os.MkdirAll(path, 0755); err != nil {
		return newFileError("mkdir", path, err)
	}

	return nil
}

// EnsureDirectory is an alias for CreateDirectoryIfNotExists
func EnsureDirectory(path string) error {
	return CreateDirectoryIfNotExists(path)
}

// ExpandPath expands ~ to home directory and returns absolute path
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", newFileError("expand", path, err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", newFileError("expand", path, err)
	}

	return absPath, nil
}

// SanitizeFilename removes dangerous characters from a filename
func SanitizeFilename(filename string) string {
	if filename == "" {
		return ""
	}

	// Replace forbidden characters
	forbidden := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	result := filename

	for _, char := range forbidden {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Replace spaces with underscores
	result = strings.ReplaceAll(result, " ", "_")

	// Handle Windows reserved names
	if err := validateWindowsPath(result); err != nil {
		// If it's a reserved name, append underscore
		result = result + "_"
	}

	// Remove trailing dots and spaces
	result = strings.TrimRight(result, ". ")

	return result
}

// GetFileSize gets the size of a file
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, newFileError("stat", path, err)
	}

	if info.IsDir() {
		return 0, newFileError("stat", path, ErrNotAFile)
	}

	return info.Size(), nil
}

// GetFileInfo returns detailed file information
func GetFileInfo(path string) (*FileInfo, error) {
	absPath, err := ExpandPath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, newFileError("stat", path, err)
	}

	return &FileInfo{
		Path:    absPath,
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}

// FileInfo contains detailed file information
type FileInfo struct {
	Path    string
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

// CopyFile copies a file preserving metadata
func CopyFile(src, dst string) error {
	// Use atomic copy for safety
	return CopyFileAtomic(src, dst)
}

// MoveFile moves a file efficiently
func MoveFile(src, dst string) error {
	// Validate input
	if src == "" || dst == "" {
		return ErrEmptyPath
	}

	// Check if source and destination are the same
	if src == dst {
		return nil
	}

	// Try rename first (most efficient for same filesystem)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Check if it's a cross-device error
	if !errors.Is(err, syscall.EXDEV) {
		return newFileError("move", src, err)
	}

	// Fall back to copy and delete for cross-device move
	return moveFileCrossDevice(src, dst)
}

// moveFileCrossDevice handles moving files across different filesystems
func moveFileCrossDevice(src, dst string) error {
	// Get source info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return newFileError("move", src, fmt.Errorf("stat source: %w", err))
	}

	// Don't move directories
	if srcInfo.IsDir() {
		return newFileError("move", src, errors.New("cannot move directories across devices"))
	}

	// Copy the file
	if err := CopyFileAtomic(src, dst); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	// Remove the source
	if err := os.Remove(src); err != nil {
		// Try to clean up the copy
		os.Remove(dst)
		return newFileError("move", src, fmt.Errorf("remove source: %w", err))
	}

	return nil
}

// DeleteFile safely deletes a file (not a directory)
func DeleteFile(path string) error {
	if path == "" {
		return ErrEmptyPath
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return newFileError("delete", path, err)
	}

	// Don't delete directories with this function
	if info.IsDir() {
		return newFileError("delete", path, ErrNotADirectory)
	}

	// Delete the file
	if err := os.Remove(path); err != nil {
		return newFileError("delete", path, err)
	}

	return nil
}

// DeleteDirectory safely deletes a directory
func DeleteDirectory(path string, requireEmpty bool) error {
	if path == "" {
		return ErrEmptyPath
	}

	// Get directory info
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return newFileError("delete", path, err)
	}

	if !info.IsDir() {
		return newFileError("delete", path, ErrNotADirectory)
	}

	if requireEmpty {
		// Check if directory is empty
		entries, err := os.ReadDir(path)
		if err != nil {
			return newFileError("delete", path, err)
		}
		if len(entries) > 0 {
			return newFileError("delete", path, errors.New("directory not empty"))
		}
		return os.Remove(path)
	}

	// Remove all contents
	return os.RemoveAll(path)
}

// GetFileExtension gets the extension of a file
func GetFileExtension(filename string) string {
	return filepath.Ext(filename)
}

// ReadFileString reads a file as a string
func ReadFileString(path string) (string, error) {
	data, err := ReadFileBytes(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadFileBytes reads a file as bytes
func ReadFileBytes(path string) ([]byte, error) {
	absPath, err := ExpandPath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, newFileError("read", path, err)
	}

	return data, nil
}

// WriteFile writes data to a file with specified permissions
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return WriteFileAtomic(path, data, perm)
}

// AppendToFile appends data to a file
func AppendToFile(path string, data []byte) error {
	absPath, err := ExpandPath(path)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return newFileError("append", path, err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return newFileError("append", path, err)
	}

	return nil
}

// ListFiles lists files in a directory
func ListFiles(dir string, recursive bool) ([]string, error) {
	absDir, err := ExpandPath(dir)
	if err != nil {
		return nil, err
	}

	var files []string

	if recursive {
		err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(absDir)
		if err != nil {
			return nil, newFileError("list", dir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				files = append(files, filepath.Join(absDir, entry.Name()))
			}
		}
	}

	return files, err
}

// FindFiles finds files matching a pattern
func FindFiles(root string, pattern string) ([]string, error) {
	absRoot, err := ExpandPath(root)
	if err != nil {
		return nil, err
	}

	var matches []string

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}

		if matched && !info.IsDir() {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

// GetMimeType attempts to determine the MIME type of a file
func GetMimeType(path string) string {
	// First try by extension
	ext := strings.ToLower(filepath.Ext(path))

	mimeTypes := map[string]string{
		".txt":   "text/plain",
		".html":  "text/html",
		".htm":   "text/html",
		".css":   "text/css",
		".js":    "text/javascript",
		".json":  "application/json",
		".xml":   "application/xml",
		".pdf":   "application/pdf",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".zip":   "application/zip",
		".tar":   "application/x-tar",
		".gz":    "application/gzip",
		".exe":   "application/x-msdownload",
		".dll":   "application/x-msdownload",
		".so":    "application/x-sharedlib",
		".dylib": "application/x-sharedlib",
		".mp3":   "audio/mpeg",
		".mp4":   "video/mp4",
		".avi":   "video/x-msvideo",
		".doc":   "application/msword",
		".docx":  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":   "application/vnd.ms-excel",
		".xlsx":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	// Default to binary
	return "application/octet-stream"
}

// --- Backward compatibility functions ---

// ValidateFilePath validates a file path (backward compatibility)
// Deprecated: Use ValidateFilePathEx with ValidationOptions instead
func ValidateFilePath(path string) bool {
	err := ValidateFilePathEx(path, DefaultValidationOptions())
	return err == nil
}

// ValidateFilePathEx validates a file path with options
func ValidateFilePathEx(path string, opts ValidationOptions) error {
	return ValidateFilePathWithOptions(path, opts)
}

// ValidateFilePathWithOptions is an alias for the validation function
func ValidateFilePathWithOptions(path string, opts ValidationOptions) error {
	// Delegate to the validation module
	return validateFilePathInternal(path, opts)
}

// validateFilePathInternal performs the actual validation
func validateFilePathInternal(path string, opts ValidationOptions) error {
	// Check for empty path
	if path == "" {
		return ErrEmptyPath
	}

	// Check for null characters
	if strings.Contains(path, "\x00") {
		return ErrNullCharacter
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// Check if relative paths are allowed
	if !opts.AllowRelative && !filepath.IsAbs(cleaned) {
		// Try to make it absolute
		absPath, err := filepath.Abs(cleaned)
		if err != nil {
			return newFileError("validate", path, fmt.Errorf("resolve absolute path: %w", err))
		}
		cleaned = absPath
	}

	// Resolve symlinks if requested
	if opts.ResolveSymlinks {
		resolved, err := filepath.EvalSymlinks(cleaned)
		if err != nil && !os.IsNotExist(err) {
			return newFileError("validate", path, fmt.Errorf("resolve symlinks: %w", err))
		}
		if err == nil {
			cleaned = resolved
		}
	}

	// Check path length
	if opts.MaxPathLength > 0 && len(cleaned) > opts.MaxPathLength {
		return newFileError("validate", path, fmt.Errorf("%w: %d > %d", ErrPathTooLong, len(cleaned), opts.MaxPathLength))
	}

	// Check if path escapes root
	if opts.RootDir != "" {
		absRoot, err := filepath.Abs(opts.RootDir)
		if err != nil {
			return newFileError("validate", path, fmt.Errorf("resolve root path: %w", err))
		}

		// Ensure the path is within the root directory
		if !strings.HasPrefix(cleaned, absRoot) {
			return newFileError("validate", path, ErrPathEscapesRoot)
		}
	}

	// Platform-specific validation
	if err := validatePlatformSpecific(cleaned); err != nil {
		return newFileError("validate", path, err)
	}

	return nil
}
