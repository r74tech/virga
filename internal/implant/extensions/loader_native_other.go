//go:build !windows
// +build !windows

package extensions

import (
	"fmt"
)

// loadWindows is a stub for non-Windows platforms
func (l *NativeLoader) loadWindows(_ *Manifest, _ string) (Extension, error) {
	return nil, fmt.Errorf("Windows DLL loading is only available on Windows")
}
