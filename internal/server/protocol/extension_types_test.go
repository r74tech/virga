package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionManifestSerialization(t *testing.T) {
	manifest := ExtensionManifest{
		Name:        "test-extension",
		Version:     "1.0.0",
		Author:      "Test Author",
		Description: "Test extension for unit tests",
		Type:        "bof",
		Platforms: []PlatformInfo{
			{
				OS:   "windows",
				Arch: "amd64",
				Path: "test.x64.o",
			},
			{
				OS:   "linux",
				Arch: "amd64",
				Path: "test.x64.o",
			},
		},
		Commands: []ExtensionCommand{
			{
				Name:        "test",
				Description: "Run test command",
				Arguments: []CommandArgument{
					{
						Name:        "target",
						Type:        "string",
						Description: "Target to test",
						Required:    true,
					},
				},
			},
		},
		Dependencies: []string{"base-extension"},
	}

	// Test serialization
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-extension")

	// Test deserialization
	var decoded ExtensionManifest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, manifest.Name, decoded.Name)
	assert.Equal(t, manifest.Version, decoded.Version)
	assert.Len(t, decoded.Platforms, 2)
	assert.Len(t, decoded.Commands, 1)
	assert.Len(t, decoded.Dependencies, 1)
}

func TestExtensionUploadTask(t *testing.T) {
	task := ExtensionUploadTask{
		Manifest: ExtensionManifest{
			Name:    "upload-test",
			Version: "1.0.0",
			Type:    "dll",
		},
		Data:     []byte("fake binary data"),
		Checksum: "sha256:abcdef123456",
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded ExtensionUploadTask
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, task.Manifest.Name, decoded.Manifest.Name)
	assert.Equal(t, task.Checksum, decoded.Checksum)
	assert.Equal(t, task.Data, decoded.Data)
}

func TestExtensionExecuteTask(t *testing.T) {
	task := ExtensionExecuteTask{
		ExtensionName: "test-extension",
		Command:       "scan",
		Arguments: map[string]interface{}{
			"target":  "192.168.1.0/24",
			"ports":   []int{80, 443, 8080},
			"timeout": 30,
		},
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded ExtensionExecuteTask
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, task.ExtensionName, decoded.ExtensionName)
	assert.Equal(t, task.Command, decoded.Command)
	assert.Equal(t, "192.168.1.0/24", decoded.Arguments["target"])
}

func TestExtensionListResult(t *testing.T) {
	result := ExtensionListResult{
		Extensions: []ExtensionInfo{
			{
				Name:        "ext1",
				Version:     "1.0.0",
				Type:        "bof",
				LoadedAt:    1234567890,
				MemoryUsage: 1024 * 1024, // 1MB
			},
			{
				Name:     "ext2",
				Version:  "2.0.0",
				Type:     "dll",
				LoadedAt: 1234567900,
			},
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded ExtensionListResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Len(t, decoded.Extensions, 2)
	assert.Equal(t, "ext1", decoded.Extensions[0].Name)
	assert.Equal(t, int64(1024*1024), decoded.Extensions[0].MemoryUsage)
}

func TestExtensionExecuteResult(t *testing.T) {
	result := ExtensionExecuteResult{
		Success:  true,
		Output:   "Command executed successfully",
		ExitCode: 0,
		Data: map[string]interface{}{
			"processes_found": 42,
			"scan_duration":   "1.5s",
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded ExtensionExecuteResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.True(t, decoded.Success)
	assert.Equal(t, result.Output, decoded.Output)
	assert.Equal(t, float64(42), decoded.Data["processes_found"]) // JSON numbers decode as float64
}

func TestPlatformInfo(t *testing.T) {
	platforms := []PlatformInfo{
		{OS: "windows", Arch: "amd64", Path: "ext.x64.dll"},
		{OS: "windows", Arch: "386", Path: "ext.x86.dll"},
		{OS: "linux", Arch: "amd64", Path: "ext.x64.so"},
		{OS: "darwin", Arch: "amd64", Path: "ext.x64.dylib"},
		{OS: "darwin", Arch: "arm64", Path: "ext.arm64.dylib"},
	}

	for _, platform := range platforms {
		data, err := json.Marshal(platform)
		require.NoError(t, err)

		var decoded PlatformInfo
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, platform, decoded)
	}
}

func TestCommandArgument(t *testing.T) {
	args := []CommandArgument{
		{
			Name:        "file",
			Type:        "file",
			Description: "File to process",
			Required:    true,
		},
		{
			Name:        "verbose",
			Type:        "bool",
			Description: "Enable verbose output",
			Required:    false,
			Default:     "false",
		},
		{
			Name:        "timeout",
			Type:        "int",
			Description: "Timeout in seconds",
			Required:    false,
			Default:     "30",
		},
	}

	for _, arg := range args {
		data, err := json.Marshal(arg)
		require.NoError(t, err)

		var decoded CommandArgument
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, arg, decoded)
	}
}

func TestExtensionTaskTypes(t *testing.T) {
	// Ensure new task types are valid
	taskTypes := []TaskType{
		TaskTypeExtensionUpload,
		TaskTypeExtensionList,
		TaskTypeExtensionExecute,
		TaskTypeExtensionUnload,
	}

	for _, tt := range taskTypes {
		assert.NotEmpty(t, tt.String())
		assert.Contains(t, tt.String(), "extension")
	}
}
