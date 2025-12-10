package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to a file atomically
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	// Validate the path
	if err := ValidateFilePathEx(path, DefaultValidationOptions()); err != nil {
		return err
	}

	dir := filepath.Dir(path)

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return newFileError("write", path, fmt.Errorf("create directory: %w", err))
	}

	// Create temporary file in the same directory (ensures same filesystem)
	tmp, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return newFileError("write", path, fmt.Errorf("create temp file: %w", err))
	}
	tmpName := tmp.Name()

	// Ensure cleanup on error or panic
	defer func() {
		if tmp != nil {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	// Write data
	if _, err := tmp.Write(data); err != nil {
		return newFileError("write", path, fmt.Errorf("write data: %w", err))
	}

	// Set permissions
	if err := tmp.Chmod(perm); err != nil {
		return newFileError("write", path, fmt.Errorf("set permissions: %w", err))
	}

	// Sync to disk
	if err := tmp.Sync(); err != nil {
		return newFileError("write", path, fmt.Errorf("sync to disk: %w", err))
	}

	// Close the file
	if err := tmp.Close(); err != nil {
		return newFileError("write", path, fmt.Errorf("close temp file: %w", err))
	}
	tmp = nil // Prevent defer from removing it

	// Atomic rename
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return newFileError("write", path, fmt.Errorf("atomic rename: %w", err))
	}

	return nil
}

// CopyFileAtomic copies a file atomically with metadata preservation
func CopyFileAtomic(src, dst string) error {
	// Validate paths
	if err := ValidateFilePathEx(src, DefaultValidationOptions()); err != nil {
		return newFileError("copy", src, err)
	}
	if err := ValidateFilePathEx(dst, DefaultValidationOptions()); err != nil {
		return newFileError("copy", dst, err)
	}

	// Get source file info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return newFileError("copy", src, fmt.Errorf("stat source: %w", err))
	}

	// Don't copy directories
	if srcInfo.IsDir() {
		return newFileError("copy", src, ErrNotAFile)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return newFileError("copy", src, fmt.Errorf("open source: %w", err))
	}
	defer srcFile.Close()

	// Create destination directory
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return newFileError("copy", dst, fmt.Errorf("create directory: %w", err))
	}

	// Create temporary file
	tmp, err := os.CreateTemp(dstDir, ".tmp-")
	if err != nil {
		return newFileError("copy", dst, fmt.Errorf("create temp file: %w", err))
	}
	tmpName := tmp.Name()

	// Ensure cleanup
	defer func() {
		if tmp != nil {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	// Copy data
	if _, err := io.Copy(tmp, srcFile); err != nil {
		return newFileError("copy", dst, fmt.Errorf("copy data: %w", err))
	}

	// Set permissions
	if err := tmp.Chmod(srcInfo.Mode()); err != nil {
		return newFileError("copy", dst, fmt.Errorf("set permissions: %w", err))
	}

	// Sync to disk
	if err := tmp.Sync(); err != nil {
		return newFileError("copy", dst, fmt.Errorf("sync to disk: %w", err))
	}

	// Close temp file
	if err := tmp.Close(); err != nil {
		return newFileError("copy", dst, fmt.Errorf("close temp file: %w", err))
	}
	tmp = nil

	// Atomic rename
	if err := os.Rename(tmpName, dst); err != nil {
		os.Remove(tmpName)
		return newFileError("copy", dst, fmt.Errorf("atomic rename: %w", err))
	}

	// Preserve timestamps
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		// Non-critical error, log but don't fail
		// Could add logging here
	}

	return nil
}

// CreateTempFile creates a temporary file with optional data
func CreateTempFile(pattern string, data []byte) (string, func(), error) {
	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}

	tmpName := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	if data != nil {
		if _, err := tmp.Write(data); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("write temp file: %w", err)
		}
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close temp file: %w", err)
	}

	return tmpName, func() { os.Remove(tmpName) }, nil
}

// GenerateTempName generates a unique temporary filename
func GenerateTempName(prefix string) string {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(randomBytes))
}

// MoveFileAtomic moves a file atomically
func MoveFileAtomic(src, dst string) error {
	// Validate paths
	if err := ValidateFilePathEx(src, DefaultValidationOptions()); err != nil {
		return newFileError("move", src, err)
	}
	if err := ValidateFilePathEx(dst, DefaultValidationOptions()); err != nil {
		return newFileError("move", dst, err)
	}

	// Same file check
	if src == dst {
		return nil
	}

	// Get source info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return newFileError("move", src, fmt.Errorf("stat source: %w", err))
	}

	// Don't move directories with this function
	if srcInfo.IsDir() {
		return newFileError("move", src, ErrNotAFile)
	}

	// Try direct rename first (same filesystem)
	err = os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If rename failed due to cross-device, copy then delete
	if os.IsNotExist(err) {
		return newFileError("move", src, err)
	}

	// Copy the file
	if err := CopyFileAtomic(src, dst); err != nil {
		return fmt.Errorf("copy during move: %w", err)
	}

	// Remove the source
	if err := os.Remove(src); err != nil {
		// Try to clean up the copy
		os.Remove(dst)
		return newFileError("move", src, fmt.Errorf("remove source: %w", err))
	}

	return nil
}

// SafeRemove safely removes a file (not a directory)
func SafeRemove(path string) error {
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone
		}
		return newFileError("remove", path, err)
	}

	// Don't remove directories
	if info.IsDir() {
		return newFileError("remove", path, ErrIsDirectory)
	}

	// Remove the file
	if err := os.Remove(path); err != nil {
		return newFileError("remove", path, err)
	}

	return nil
}

// EnsureFileSync ensures a file is synced to disk
func EnsureFileSync(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return newFileError("sync", path, err)
	}
	defer file.Close()

	if err := file.Sync(); err != nil {
		return newFileError("sync", path, fmt.Errorf("sync to disk: %w", err))
	}

	return nil
}
