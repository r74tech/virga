//go:build windows
// +build windows

package extensions

import (
	"encoding/json"
	"fmt"
	"syscall"
	"unsafe"
)

// WindowsNativeExtension represents a Windows DLL extension
type WindowsNativeExtension struct {
	manifest    *Manifest
	callbacks   *ExtensionCallbacks
	dllHandle   syscall.Handle
	executeFunc uintptr
	cleanupFunc uintptr
}

// loadWindows loads a Windows DLL extension
func (l *NativeLoader) loadWindows(manifest *Manifest, path string) (Extension, error) {
	// Load the DLL
	dll, err := syscall.LoadDLL(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load DLL: %w", err)
	}

	// Create extension instance
	ext := &WindowsNativeExtension{
		manifest:  manifest,
		dllHandle: syscall.Handle(dll.Handle),
	}

	// Look for required functions
	// ext_execute is the main entry point
	execProc, err := dll.FindProc("ext_execute")
	if err != nil {
		syscall.FreeLibrary(ext.dllHandle)
		return nil, fmt.Errorf("DLL does not export 'ext_execute' function: %w", err)
	}
	ext.executeFunc = execProc.Addr()

	// ext_cleanup is optional
	cleanupProc, err := dll.FindProc("ext_cleanup")
	if err == nil {
		ext.cleanupFunc = cleanupProc.Addr()
	}

	return ext, nil
}

// GetManifest returns the extension manifest
func (e *WindowsNativeExtension) GetManifest() *Manifest {
	return e.manifest
}

// Initialize initializes the Windows DLL extension
func (e *WindowsNativeExtension) Initialize(callbacks *ExtensionCallbacks) error {
	e.callbacks = callbacks

	// Call ext_init if it exists
	initProc, err := syscall.GetProcAddress(e.dllHandle, "ext_init")
	if err == nil && initProc != 0 {
		// Call init function
		ret, _, _ := syscall.Syscall(initProc, 0, 0, 0, 0)
		if ret != 0 {
			return fmt.Errorf("ext_init failed with code: %d", ret)
		}
	}

	return nil
}

// Execute executes the Windows DLL extension
func (e *WindowsNativeExtension) Execute(args map[string]interface{}) (*ExtensionResult, error) {
	if e.executeFunc == 0 {
		return nil, fmt.Errorf("execute function not found")
	}

	// Convert args to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	// Prepare arguments
	argsPtr, err := syscall.UTF16PtrFromString(string(argsJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to convert arguments: %w", err)
	}

	// Call ext_execute(wchar_t* args) -> wchar_t*
	// The DLL should return a JSON string with the result
	ret, _, _ := syscall.Syscall(e.executeFunc, 1, uintptr(unsafe.Pointer(argsPtr)), 0, 0)

	if ret == 0 {
		return &ExtensionResult{
			Success:  false,
			Error:    "ext_execute returned null",
			ExitCode: -1,
		}, nil
	}

	// Convert the returned wide string to Go string
	resultPtr := (*uint16)(unsafe.Pointer(ret))
	resultStr := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(resultPtr))[:])

	// Parse the result JSON
	var result ExtensionResult
	if err := json.Unmarshal([]byte(resultStr), &result); err != nil {
		// If not JSON, treat as plain text output
		result = ExtensionResult{
			Success:  true,
			Output:   resultStr,
			ExitCode: 0,
		}
	}

	// Free the returned string if the DLL exports ext_free
	freeProc, err := syscall.GetProcAddress(e.dllHandle, "ext_free")
	if err == nil && freeProc != 0 {
		syscall.Syscall(freeProc, 1, ret, 0, 0)
	}

	return &result, nil
}

// Cleanup cleans up the Windows DLL extension
func (e *WindowsNativeExtension) Cleanup() error {
	// Call cleanup function if available
	if e.cleanupFunc != 0 {
		syscall.Syscall(e.cleanupFunc, 0, 0, 0, 0)
	}

	// Unload the DLL
	if e.dllHandle != 0 {
		if err := syscall.FreeLibrary(e.dllHandle); err != nil {
			return fmt.Errorf("failed to unload DLL: %w", err)
		}
		e.dllHandle = 0
	}

	return nil
}
