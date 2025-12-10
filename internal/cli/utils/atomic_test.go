package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWriteFileAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("test content")

	// Test basic write
	err := WriteFileAtomic(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("WriteFileAtomic() error = %v", err)
	}

	// Verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(content, testData) {
		t.Errorf("File content = %v, want %v", content, testData)
	}

	// Verify permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if info.Mode().Perm() != 0644 {
		t.Errorf("File permissions = %v, want %v", info.Mode().Perm(), os.FileMode(0644))
	}
}

func TestWriteFileAtomic_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Write initial content
	initialData := []byte("initial content")
	err := WriteFileAtomic(testFile, initialData, 0644)
	if err != nil {
		t.Fatalf("Initial write error = %v", err)
	}

	// Overwrite with new content
	newData := []byte("new content")
	err = WriteFileAtomic(testFile, newData, 0600)
	if err != nil {
		t.Fatalf("Overwrite error = %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(content, newData) {
		t.Errorf("File content = %v, want %v", content, newData)
	}

	// Verify new permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("File permissions = %v, want %v", info.Mode().Perm(), os.FileMode(0600))
	}
}

func TestWriteFileAtomic_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "concurrent.txt")

	// Run concurrent writes
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("content-%d", n))
			if err := WriteFileAtomic(testFile, data, 0644); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
	}

	// Verify file exists and is readable
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Content should be from one of the writes
	if len(content) == 0 {
		t.Error("File is empty after concurrent writes")
	}
}

func TestWriteFileAtomic_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "nested", "test.txt")
	testData := []byte("test content")

	// Write to nested directory that doesn't exist
	err := WriteFileAtomic(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("WriteFileAtomic() error = %v", err)
	}

	// Verify file exists
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(content, testData) {
		t.Errorf("File content = %v, want %v", content, testData)
	}
}

func TestWriteFileAtomic_InvalidPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "empty path",
			path: "",
		},
		{
			name: "invalid characters",
			path: "/tmp/test\x00file.txt",
		},
		{
			name: "directory path",
			path: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteFileAtomic(tt.path, []byte("test"), 0644)
			if err == nil {
				t.Error("WriteFileAtomic() expected error for invalid path")
			}
		})
	}
}

func TestCopyFileAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	testData := []byte("test content for copy")

	// Create source file
	err := os.WriteFile(srcFile, testData, 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Set specific modification time
	modTime := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	err = os.Chtimes(srcFile, modTime, modTime)
	if err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	// Copy file
	err = CopyFileAtomic(srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyFileAtomic() error = %v", err)
	}

	// Verify content
	content, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(content, testData) {
		t.Errorf("Copied content = %v, want %v", content, testData)
	}

	// Verify permissions
	srcInfo, _ := os.Stat(srcFile)
	dstInfo, _ := os.Stat(dstFile)

	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("Copied file mode = %v, want %v", dstInfo.Mode(), srcInfo.Mode())
	}

	// Verify modification time (allow 1 second difference due to filesystem precision)
	timeDiff := dstInfo.ModTime().Sub(srcInfo.ModTime())
	if timeDiff < -time.Second || timeDiff > time.Second {
		t.Errorf("Copied file modtime = %v, want %v", dstInfo.ModTime(), srcInfo.ModTime())
	}
}

func TestCopyFileAtomic_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	// Create source and destination files
	srcData := []byte("source content")
	dstData := []byte("destination content")

	err := os.WriteFile(srcFile, srcData, 0644)
	if err != nil {
		t.Fatalf("WriteFile(src) error = %v", err)
	}

	err = os.WriteFile(dstFile, dstData, 0600)
	if err != nil {
		t.Fatalf("WriteFile(dst) error = %v", err)
	}

	// Copy should overwrite
	err = CopyFileAtomic(srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyFileAtomic() error = %v", err)
	}

	// Verify content was overwritten
	content, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(content, srcData) {
		t.Errorf("Copied content = %v, want %v", content, srcData)
	}
}

func TestCopyFileAtomic_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "large.bin")
	dstFile := filepath.Join(tmpDir, "large_copy.bin")

	// Create a 10MB file
	size := 10 * 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err := os.WriteFile(srcFile, data, 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Copy large file
	start := time.Now()
	err = CopyFileAtomic(srcFile, dstFile)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("CopyFileAtomic() error = %v", err)
	}

	t.Logf("Copied %d MB in %v", size/1024/1024, duration)

	// Verify size
	srcInfo, _ := os.Stat(srcFile)
	dstInfo, _ := os.Stat(dstFile)

	if dstInfo.Size() != srcInfo.Size() {
		t.Errorf("Copied file size = %v, want %v", dstInfo.Size(), srcInfo.Size())
	}

	// Spot check content
	dstFileHandle, err := os.Open(dstFile)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer dstFileHandle.Close()

	// Check first and last 1KB
	checkBuf := make([]byte, 1024)
	_, err = dstFileHandle.Read(checkBuf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !bytes.Equal(checkBuf, data[:1024]) {
		t.Error("First 1KB of copied file doesn't match")
	}

	_, err = dstFileHandle.Seek(-1024, io.SeekEnd)
	if err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	_, err = dstFileHandle.Read(checkBuf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !bytes.Equal(checkBuf, data[len(data)-1024:]) {
		t.Error("Last 1KB of copied file doesn't match")
	}
}

func TestMoveFileAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "moved.txt")
	testData := []byte("content to move")

	// Create source file
	err := os.WriteFile(srcFile, testData, 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Move file
	err = MoveFileAtomic(srcFile, dstFile)
	if err != nil {
		t.Fatalf("MoveFileAtomic() error = %v", err)
	}

	// Verify source doesn't exist
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("Source file still exists after move")
	}

	// Verify destination exists with correct content
	content, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(content, testData) {
		t.Errorf("Moved content = %v, want %v", content, testData)
	}
}

func TestMoveFileAtomic_AcrossDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	err := os.MkdirAll(srcDir, 0755)
	if err != nil {
		t.Fatalf("MkdirAll(src) error = %v", err)
	}

	err = os.MkdirAll(dstDir, 0755)
	if err != nil {
		t.Fatalf("MkdirAll(dst) error = %v", err)
	}

	srcFile := filepath.Join(srcDir, "file.txt")
	dstFile := filepath.Join(dstDir, "file.txt")
	testData := []byte("cross-directory move")

	// Create source file
	err = os.WriteFile(srcFile, testData, 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Move across directories
	err = MoveFileAtomic(srcFile, dstFile)
	if err != nil {
		t.Fatalf("MoveFileAtomic() error = %v", err)
	}

	// Verify move
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("Source file still exists after cross-directory move")
	}

	content, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(content, testData) {
		t.Errorf("Moved content = %v, want %v", content, testData)
	}
}

func TestSafeRemove(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "remove_me.txt")

	// Create file
	err := os.WriteFile(testFile, []byte("delete this"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Remove file
	err = SafeRemove(testFile)
	if err != nil {
		t.Fatalf("SafeRemove() error = %v", err)
	}

	// Verify removed
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File still exists after SafeRemove")
	}
}

func TestSafeRemove_NonExistent(t *testing.T) {
	// Removing non-existent file should not error
	err := SafeRemove("/tmp/does_not_exist_12345.txt")
	if err != nil {
		t.Errorf("SafeRemove() error for non-existent file = %v", err)
	}
}

func TestSafeRemove_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "remove_dir")

	// Create directory
	err := os.Mkdir(testDir, 0755)
	if err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	// Try to remove directory (should fail)
	err = SafeRemove(testDir)
	if err == nil {
		t.Error("SafeRemove() should error when trying to remove directory")
	}

	// Directory should still exist
	if _, err := os.Stat(testDir); err != nil {
		t.Error("Directory was removed by SafeRemove")
	}
}

func TestEnsureFileSync(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sync_test.txt")
	testData := []byte("data to sync")

	// Create file
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Write data
	_, err = file.Write(testData)
	if err != nil {
		file.Close()
		t.Fatalf("Write() error = %v", err)
	}

	// Ensure sync
	err = EnsureFileSync(testFile)
	if err != nil {
		file.Close()
		t.Fatalf("EnsureFileSync() error = %v", err)
	}

	file.Close()

	// Verify content persisted
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(content, testData) {
		t.Errorf("Synced content = %v, want %v", content, testData)
	}
}
