package config

import "fmt"

// Transport represents MCP transport types
type Transport string

const (
	TransportStdio      Transport = "stdio"
	TransportSSE        Transport = "sse"
	TransportStreamable Transport = "streamable"
)

// Config represents unified MCP configuration
type Config struct {
	Name    string
	Version string

	// Stdio configuration
	StdioEnabled bool

	// SSE configuration
	SSEEnabled   bool
	SSEPort      string
	SSEBasePath  string
	SSERemoteURL string

	// Streamable configuration
	StreamableEnabled bool
	StreamablePort    string
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.SSEEnabled && c.SSEPort == "" {
		return fmt.Errorf("SSE port is required when SSE is enabled")
	}
	if c.StreamableEnabled && c.StreamablePort == "" {
		return fmt.Errorf("Streamable port is required when Streamable is enabled")
	}
	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Name:              "Virga MCP Server",
		Version:           "1.0.0",
		SSEEnabled:        true,
		SSEPort:           ":8444",
		SSEBasePath:       "/mcp",
		StdioEnabled:      false,
		StreamableEnabled: false,
		StreamablePort:    ":50012",
	}
}

// Copy returns a deep copy of the configuration
func (c *Config) Copy() *Config {
	return &Config{
		Name:              c.Name,
		Version:           c.Version,
		StdioEnabled:      c.StdioEnabled,
		SSEEnabled:        c.SSEEnabled,
		SSEPort:           c.SSEPort,
		SSEBasePath:       c.SSEBasePath,
		SSERemoteURL:      c.SSERemoteURL,
		StreamableEnabled: c.StreamableEnabled,
		StreamablePort:    c.StreamablePort,
	}
}
