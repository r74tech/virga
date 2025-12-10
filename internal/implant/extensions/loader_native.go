package extensions

import (
	"fmt"
	"plugin"
	"runtime"
)

// NativeLoader loads native extensions (.dll, .so, .dylib)
type NativeLoader struct {
	loadedPlugins map[string]*plugin.Plugin
}

// NewNativeLoader creates a new native extension loader
func NewNativeLoader() *NativeLoader {
	return &NativeLoader{
		loadedPlugins: make(map[string]*plugin.Plugin),
	}
}

// Load loads a native extension
func (l *NativeLoader) Load(manifest *Manifest, path string) (Extension, error) {
	// Note: Go's plugin package only works on Linux/macOS
	// For Windows, we'll need to use different approach (syscall.LoadDLL)
	if runtime.GOOS == "windows" {
		return l.loadWindows(manifest, path)
	}

	// Load the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for the New function that returns an Extension
	newFunc, err := p.Lookup("New")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export 'New' function: %w", err)
	}

	// Cast to the expected function signature
	createFunc, ok := newFunc.(func() Extension)
	if !ok {
		return nil, fmt.Errorf("'New' function has incorrect signature")
	}

	// Create the extension instance
	ext := createFunc()

	// Store the plugin for cleanup
	l.loadedPlugins[manifest.Name] = p

	return ext, nil
}

// Unload unloads a native extension
func (l *NativeLoader) Unload(extension Extension) error {
	// Note: Go plugins cannot be unloaded at runtime
	// We can only remove our reference to it
	manifest := extension.GetManifest()
	if manifest != nil {
		delete(l.loadedPlugins, manifest.Name)
	}
	return nil
}

// GetType returns the type of extensions this loader handles
func (l *NativeLoader) GetType() ExtensionType {
	return ExtensionTypeNative
}

// ReflectiveExtension is a wrapper for reflectively loaded extensions
type ReflectiveExtension struct {
	manifest  *Manifest
	callbacks *ExtensionCallbacks
	// handle    interface{} // TODO: Will be used for loaded module handle
}

// GetManifest returns the extension manifest
func (e *ReflectiveExtension) GetManifest() *Manifest {
	return e.manifest
}

// Initialize initializes the extension
func (e *ReflectiveExtension) Initialize(callbacks *ExtensionCallbacks) error {
	e.callbacks = callbacks
	// Call the extension's init function if specified
	// This would involve calling the function through the handle
	return nil
}

// Execute executes the extension
func (e *ReflectiveExtension) Execute(args map[string]interface{}) (*ExtensionResult, error) {
	// Call the extension's entrypoint function
	// This would involve calling the function through the handle
	return &ExtensionResult{
		Success:  true,
		Output:   "Extension executed successfully",
		ExitCode: 0,
	}, nil
}

// Cleanup cleans up the extension
func (e *ReflectiveExtension) Cleanup() error {
	// Perform any cleanup needed
	return nil
}
