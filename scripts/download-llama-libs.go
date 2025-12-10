//go:build ignore
// +build ignore

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/r74tech/virga/internal/shared/logger"
)

const (
	repoOwner = "r74tech"
	repoName  = "llama.go"
	baseDir   = "internal/implant/llama/libs"
)

var (
	targetPlatform = flag.String("platform", "", "Target platform (darwin, linux, windows). Defaults to current platform")
	targetArch     = flag.String("arch", "", "Target architecture (amd64, arm64). Defaults to current architecture")
	listSupported  = flag.Bool("list", false, "List supported platform/architecture combinations")
	allPlatforms   = flag.Bool("all", false, "Download libraries for all supported platforms")
)

// LibraryInfo contains information about a specific library variant
type LibraryInfo struct {
	Platform string
	Arch     string
	Variant  string // e.g., "metal" for macOS with Metal support
	FileName string
}

// Supported platform/architecture combinations
var supportedCombinations = map[string][]string{
	"darwin":  {"amd64", "arm64"},
	"linux":   {"amd64"},
	"windows": {"amd64"},
}

// Available library variants
// Note: Metal variants will be added in future releases
var availableLibraries = []LibraryInfo{
	{Platform: "darwin", Arch: "amd64", Variant: "", FileName: "llama.go-darwin-amd64.tar.gz"},
	{Platform: "darwin", Arch: "arm64", Variant: "", FileName: "llama.go-darwin-arm64.tar.gz"},
	{Platform: "linux", Arch: "amd64", Variant: "", FileName: "llama.go-linux-amd64.tar.gz"},
	{Platform: "windows", Arch: "amd64", Variant: "", FileName: "llama.go-windows-amd64.zip"},
}

func main() {
	updateGoMod := flag.Bool("update-go-mod", false, "Update go.mod to use the appropriate platform-specific library")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Downloads pre-built go-llama.cpp libraries for the specified platform and architecture.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                           # Download for current platform\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -platform=linux -arch=amd64   # Download Linux AMD64 version\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -platform=darwin -arch=arm64  # Download macOS ARM64 version\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -all                          # Download all supported libraries\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -list                         # List supported combinations\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -update-go-mod                # Update go.mod for current platform\n", os.Args[0])
	}

	flag.Parse()

	// Handle update-go-mod flag
	if *updateGoMod {
		if err := updateGoModForPlatform(); err != nil {
			logger.Error("Failed to update go.mod: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle list flag
	if *listSupported {
		logger.Info("Available library variants:")
		for _, lib := range availableLibraries {
			variant := ""
			if lib.Variant != "" {
				variant = fmt.Sprintf(" (%s)", lib.Variant)
			}
			logger.Info("  - %s/%s%s: %s", lib.Platform, lib.Arch, variant, lib.FileName)
		}
		os.Exit(0)
	}

	// Create libs directory
	libsPath := filepath.Join(baseDir)
	if err := os.MkdirAll(libsPath, 0755); err != nil {
		logger.Error("Failed to create libs directory: %v", err)
		os.Exit(1)
	}

	// Determine which libraries to download
	var librariesToDownload []LibraryInfo

	if *allPlatforms {
		// Download all available libraries
		librariesToDownload = availableLibraries
		logger.Info("Downloading all supported libraries...")
	} else {
		// Determine platform
		platform := *targetPlatform
		if platform == "" {
			platform = runtime.GOOS
		}

		arch := *targetArch
		if arch == "" {
			arch = runtime.GOARCH
		}

		// Validate platform/architecture combination
		if archs, ok := supportedCombinations[platform]; ok {
			valid := false
			for _, supportedArch := range archs {
				if arch == supportedArch {
					valid = true
					break
				}
			}
			if !valid {
				logger.Error("Architecture %s is not supported for platform %s", arch, platform)
				logger.Error("Supported architectures for %s: %v", platform, archs)
				os.Exit(1)
			}
		} else {
			logger.Error("Platform %s is not supported", platform)
			platforms := make([]string, 0, len(supportedCombinations))
			for p := range supportedCombinations {
				platforms = append(platforms, p)
			}
			logger.Error("Supported platforms: %v", platforms)
			os.Exit(1)
		}

		// Find matching libraries
		for _, lib := range availableLibraries {
			if lib.Platform == platform && lib.Arch == arch {
				// For macOS ARM64, prefer Metal variant
				if platform == "darwin" && arch == "arm64" && lib.Variant == "metal" {
					librariesToDownload = []LibraryInfo{lib}
					break
				} else if lib.Variant == "" {
					librariesToDownload = append(librariesToDownload, lib)
				}
			}
		}
	}

	// Get the latest release tag
	releaseTag := os.Getenv("LLAMA_CPP_RELEASE_TAG")
	if releaseTag == "" {
		releaseTag = getLatestRelease()
		logger.Info("Using latest release: %s", releaseTag)
	}

	// Download each library
	for _, lib := range librariesToDownload {
		logger.Info("\nProcessing %s...", lib.FileName)

		// Construct target directory name based on library info
		var targetDirName string
		if lib.Variant != "" {
			targetDirName = fmt.Sprintf("%s-%s-%s", lib.Platform, lib.Arch, lib.Variant)
		} else {
			targetDirName = fmt.Sprintf("%s-%s", lib.Platform, lib.Arch)
		}
		targetDir := filepath.Join(libsPath, targetDirName)

		// Check if library already exists
		libPath := filepath.Join(targetDir, "libbinding.a")
		if _, err := os.Stat(libPath); err == nil {
			logger.Info("Library already exists at %s, skipping...", libPath)
			continue
		}

		// Construct download URL
		downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
			repoOwner, repoName, releaseTag, lib.FileName)

		logger.Info("Downloading from %s", downloadURL)

		// Download the file
		tmpFile := filepath.Join(libsPath, lib.FileName)
		if err := downloadFile(tmpFile, downloadURL); err != nil {
			logger.Error("Failed to download %s: %v", lib.FileName, err)
			continue
		}

		// Extract the archive to a temporary directory
		tmpExtractDir := filepath.Join(libsPath, "tmp_extract")
		if err := os.MkdirAll(tmpExtractDir, 0755); err != nil {
			logger.Error("Failed to create temp directory: %v", err)
			os.Remove(tmpFile)
			continue
		}

		// Extract based on file extension
		var extractErr error
		if strings.HasSuffix(lib.FileName, ".zip") {
			extractErr = extractZip(tmpFile, tmpExtractDir)
		} else if strings.HasSuffix(lib.FileName, ".tar.gz") {
			extractErr = extractTarGz(tmpFile, tmpExtractDir)
		} else {
			extractErr = fmt.Errorf("unsupported archive format")
		}

		if extractErr != nil {
			logger.Error("Failed to extract %s: %v", lib.FileName, extractErr)
			os.Remove(tmpFile)
			os.RemoveAll(tmpExtractDir)
			continue
		}

		// Clean up downloaded file
		os.Remove(tmpFile)

		// Find the extracted directory (it should be the only one in tmpExtractDir)
		entries, err := os.ReadDir(tmpExtractDir)
		if err != nil || len(entries) == 0 {
			logger.Error("Failed to find extracted directory")
			os.RemoveAll(tmpExtractDir)
			continue
		}

		extractedDir := filepath.Join(tmpExtractDir, entries[0].Name())

		// Remove existing target directory if it exists
		if _, err := os.Stat(targetDir); err == nil {
			os.RemoveAll(targetDir)
		}

		// Move to final location
		if err := os.Rename(extractedDir, targetDir); err != nil {
			logger.Error("Failed to move directory: %v", err)
			os.RemoveAll(tmpExtractDir)
			continue
		}

		// Clean up temp directory
		os.RemoveAll(tmpExtractDir)

		// Verify the library
		libPath = filepath.Join(targetDir, "libbinding.a")
		if stat, err := os.Stat(libPath); err == nil {
			logger.Info("Successfully downloaded %s (%.2f MB)",
				lib.FileName, float64(stat.Size())/(1024*1024))

			// Apply patches
			applyPatches(targetDir)

			// Create build directory
			buildDir := filepath.Join(targetDir, "build")
			if err := os.MkdirAll(buildDir, 0755); err != nil {
				logger.Warn("failed to create build directory: %v", err)
			}
		} else {
		}
	}

	logger.Info("\nDownload complete!")

	// Always update go.mod after downloading libraries
	logger.Info("\nUpdating go.mod for current platform...")
	if err := updateGoModForPlatform(); err != nil {
		logger.Warn("Failed to update go.mod: %v", err)
	}
}

func getLatestRelease() string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	resp, err := http.Get(url)
	if err != nil {
		logger.Error("Failed to get latest release: %v", err)
		// Fallback to a known version
		return "v0.1.3"
	}
	defer resp.Body.Close()

	// Simple JSON parsing to get tag_name
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Look for "tag_name":"v..."
	tagStart := strings.Index(bodyStr, `"tag_name":"`)
	if tagStart == -1 {
		return "v0.1.3"
	}

	tagStart += len(`"tag_name":"`)
	tagEnd := strings.Index(bodyStr[tagStart:], `"`)
	if tagEnd == -1 {
		return "v0.1.3"
	}

	return bodyStr[tagStart : tagStart+tagEnd]
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func applyPatches(targetDir string) {
	patches := []struct {
		name      string
		checkStr  string
		checkFile string
	}{
		{
			name:      "1902-cuda.patch",
			checkStr:  "llama_binding_state",
			checkFile: filepath.Join("llama.cpp", "common", "common.h"),
		},
		{
			name:      "memory-loading.patch",
			checkStr:  "llama_load_model_from_buffer",
			checkFile: filepath.Join("llama.cpp", "llama.h"),
		},
	}

	llamaCppDir := filepath.Join(targetDir, "llama.cpp")

	// Check if this is a Windows library (contains CRLF line endings)
	isWindowsLib := false
	if testFile := filepath.Join(llamaCppDir, "common", "common.h"); testFile != "" {
		if content, err := os.ReadFile(testFile); err == nil {
			if bytes.Contains(content, []byte("\r\n")) {
				isWindowsLib = true
				logger.Info("Detected Windows library with CRLF line endings")
			}
		}
	}

	for _, patch := range patches {
		patchPath := filepath.Join(targetDir, "patches", patch.name)

		// Check if patch file exists
		if _, err := os.Stat(patchPath); err != nil {
			logger.Warn("patch file not found: %s", patch.name)
			continue
		}

		// Check if it's a .note file
		if strings.HasSuffix(patch.name, ".note") {
			continue
		}

		// Check if patch is already applied
		checkFilePath := filepath.Join(targetDir, patch.checkFile)
		patchNeeded := true

		if content, err := os.ReadFile(checkFilePath); err == nil {
			if strings.Contains(string(content), patch.checkStr) {
				patchNeeded = false
				logger.Info("Patch %s already applied", patch.name)
			}
		}

		if patchNeeded {
			logger.Info("Applying patch %s...", patch.name)

			// Build patch command with appropriate options
			args := []string{"-p1", "-i", filepath.Join("..", "patches", patch.name)}

			// For Windows libraries, add --binary flag to handle CRLF
			if isWindowsLib {
				args = append([]string{"--binary"}, args...)
			}

			// Try to apply the patch
			cmd := exec.Command("patch", args...)
			cmd.Dir = llamaCppDir

			output, err := cmd.CombinedOutput()
			if err != nil {
				// Check if it's a line ending issue
				if strings.Contains(string(output), "different line endings") || strings.Contains(string(output), "Stripping trailing CRs") {
					logger.Warn("patch %s has line ending conflicts. This is expected for Windows libraries and usually doesn't affect functionality.", patch.name)
				} else if strings.Contains(string(output), "malformed") {
					logger.Warn("patch %s appears to be malformed (known issue per .note file)", patch.name)
				} else {
					logger.Warn("failed to apply patch %s: %v", patch.name, err)
					if len(output) > 0 {
						logger.Warn("Output: %s", output)
					}
				}
			} else {
				logger.Info("Patch %s applied successfully", patch.name)
			}
		}
	}
}

func extractTarGz(gzipPath, dest string) error {
	file, err := os.Open(gzipPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure the directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}

			file.Close()
		}
	}

	return nil
}

// updateGoModForPlatform updates the go.mod file to use the appropriate platform-specific library
func updateGoModForPlatform() error {
	// Detect current platform
	platform := runtime.GOOS + "-" + runtime.GOARCH

	// Check if platform-specific library directory exists
	platformDir := filepath.Join(baseDir, platform)

	// For macOS ARM64, prefer metal variant if available
	if platform == "darwin-arm64" {
		metalDir := filepath.Join(baseDir, "darwin-arm64-metal")
		if _, err := os.Stat(metalDir); err == nil {
			platform = "darwin-arm64-metal"
			platformDir = metalDir
		}
	}

	// Check if the platform directory exists
	if _, err := os.Stat(platformDir); err != nil {
		// If not, check for any available directory
		entries, err := os.ReadDir(baseDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") && entry.Name() != "placeholder" {
					// Check if it has libbinding.a
					libPath := filepath.Join(baseDir, entry.Name(), "libbinding.a")
					if _, err := os.Stat(libPath); err == nil {
						platform = entry.Name()
						platformDir = filepath.Join(baseDir, entry.Name())
						logger.Warn("Platform-specific library not found for %s, using available: %s",
							runtime.GOOS+"-"+runtime.GOARCH, platform)
						break
					}
				}
			}
		}

		// If still not found, use placeholder
		if _, err := os.Stat(platformDir); err != nil {
			platform = "placeholder"
			platformDir = filepath.Join(baseDir, "placeholder")
			logger.Warn("No LLAMA libraries found, using placeholder")

			// Create placeholder directory and go.mod if needed
			if err := os.MkdirAll(platformDir, 0755); err != nil {
				return fmt.Errorf("failed to create placeholder directory: %w", err)
			}

			placeholderGoMod := filepath.Join(platformDir, "go.mod")
			if _, err := os.Stat(placeholderGoMod); err != nil {
				content := `module github.com/Qitmeer/llama.go

go 1.19

// Placeholder go.mod file for initial dependency resolution
// This will be replaced by the actual library during build`

				if err := os.WriteFile(placeholderGoMod, []byte(content), 0644); err != nil {
					return fmt.Errorf("failed to create placeholder go.mod: %w", err)
				}
			}
		}
	}

	logger.Info("Updating go.mod to use platform: %s", platform)

	// Read go.mod
	goModPath := "go.mod"
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	updated := false

	// Update or add replace directive
	for i, line := range lines {
		if strings.HasPrefix(line, "replace github.com/Qitmeer/llama.go =>") ||
			strings.Contains(line, "// replace github.com/Qitmeer/llama.go =>") {
			lines[i] = fmt.Sprintf("replace github.com/Qitmeer/llama.go => ./internal/implant/llama/libs/%s", platform)
			updated = true
			logger.Info("Updated replace directive: %s", lines[i])
			break
		}
	}

	// If no replace directive was found, add one
	if !updated {
		// Find the end of the file or the closing parenthesis
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) == ")" || (i == len(lines)-1 && lines[i] == "") {
				insertAt := i
				if strings.TrimSpace(lines[i]) == ")" {
					insertAt = i + 1
				}

				newLines := make([]string, 0, len(lines)+3)
				newLines = append(newLines, lines[:insertAt]...)
				newLines = append(newLines, "")
				newLines = append(newLines, "// This replace directive will be updated dynamically during beacon generation")
				newLines = append(newLines, "// to point to the appropriate platform-specific library directory")
				newLines = append(newLines, fmt.Sprintf("replace github.com/Qitmeer/llama.go => ./internal/implant/llama/libs/%s", platform))
				if insertAt < len(lines) {
					newLines = append(newLines, lines[insertAt:]...)
				}
				lines = newLines
				updated = true
				logger.Info("Added replace directive for platform: %s", platform)
				break
			}
		}
	}

	if !updated {
		return fmt.Errorf("failed to update go.mod: could not find or add replace directive")
	}

	// Write back to go.mod
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(goModPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	logger.Info("go.mod updated successfully!")
	return nil
}

// extractZip extracts a ZIP archive to the specified destination
func extractZip(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Construct target path
		target := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			// Create directory
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		// Extract file
		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
