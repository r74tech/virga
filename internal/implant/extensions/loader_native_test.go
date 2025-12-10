package extensions

import (
	"runtime"
	"testing"
)

func TestNativeLoader_Basic(t *testing.T) {
	loader := NewNativeLoader()

	if loader.GetType() != ExtensionTypeNative {
		t.Errorf("Expected type %v, got %v", ExtensionTypeNative, loader.GetType())
	}
}

func TestNativeLoader_LoadWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	loader := NewNativeLoader()
	manifest := &Manifest{
		Name:    "test-dll",
		Version: "1.0.0",
		Type:    "native",
	}

	// This will fail on non-Windows or without a real DLL
	_, err := loader.Load(manifest, "test.dll")
	if err == nil {
		t.Error("Expected error loading non-existent DLL")
	}
}

func TestNativeLoader_LoadNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Non-Windows test")
	}

	loader := NewNativeLoader()
	manifest := &Manifest{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    "native",
	}

	// This will fail without a real plugin
	_, err := loader.Load(manifest, "test.so")
	if err == nil {
		t.Error("Expected error loading non-existent plugin")
	}
}
