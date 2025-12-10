package protocol

// ExtensionManifest represents extension metadata
type ExtensionManifest struct {
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Author       string             `json:"author"`
	Description  string             `json:"description"`
	Type         string             `json:"type"`         // "bof", "dll", "script", etc.
	Platforms    []PlatformInfo     `json:"platforms"`    // Supported platforms
	Commands     []ExtensionCommand `json:"commands"`     // Commands provided by extension
	Dependencies []string           `json:"dependencies"` // Dependencies on other extensions
}

// PlatformInfo describes a supported platform
type PlatformInfo struct {
	OS   string `json:"os"`   // "windows", "linux", "darwin"
	Arch string `json:"arch"` // "amd64", "386", "arm64"
	Path string `json:"path"` // Path to platform-specific binary
}

// ExtensionCommand describes a command provided by the extension
type ExtensionCommand struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Arguments   []CommandArgument `json:"arguments,omitempty"`
}

// CommandArgument describes an argument for an extension command
type CommandArgument struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "string", "int", "bool", "file"
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// ExtensionUploadTask represents a task to upload an extension to implant
type ExtensionUploadTask struct {
	Manifest ExtensionManifest `json:"manifest"`
	Data     []byte            `json:"data"`     // Extension binary (base64 encoded in JSON)
	Checksum string            `json:"checksum"` // SHA256 checksum
}

// ExtensionExecuteTask represents a task to execute an extension
type ExtensionExecuteTask struct {
	ExtensionName string                 `json:"extension_name"`
	Command       string                 `json:"command"`
	Arguments     map[string]interface{} `json:"arguments"`
}

// ExtensionListResult represents the result of listing extensions
type ExtensionListResult struct {
	Extensions []ExtensionInfo `json:"extensions"`
}

// ExtensionInfo represents basic information about a loaded extension
type ExtensionInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	LoadedAt    int64  `json:"loaded_at"` // Unix timestamp
	MemoryUsage int64  `json:"memory_usage,omitempty"`
}

// ExtensionExecuteResult represents the result of executing an extension
type ExtensionExecuteResult struct {
	Success  bool                   `json:"success"`
	Output   string                 `json:"output,omitempty"`
	Error    string                 `json:"error,omitempty"`
	ExitCode int                    `json:"exit_code"`
	Data     map[string]interface{} `json:"data,omitempty"` // Extension-specific data
}
