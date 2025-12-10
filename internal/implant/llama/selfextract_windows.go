//go:build llama_selfextract && windows
// +build llama_selfextract,windows

package llama

// LoadModelFromSelf loads embedded model using streaming approach on Windows
// Windows doesn't support Unix mmap, so we use memory loading
func LoadModelFromSelf() ([]byte, error) {
	// Windows: Use streaming/memory loading approach
	// This is less memory-efficient than mmap but works on Windows
	return LoadModelFromSelfStreaming()
}
