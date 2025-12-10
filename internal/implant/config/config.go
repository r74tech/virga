package config

// Config holds the implant configuration
type Config struct {
	// Values injected by go build -ldflags
	C2Host     string
	C2Port     string
	C2Protocol string
	C2Path     string
	UserAgent  string
	SleepTime  string
	Jitter     string

	// Values generated at runtime
	AgentID string
	AESKey  []byte
}

// Variables injected by go build -ldflags (default values)
var (
	// These values are overridden by go build -ldflags
	BuildC2Host     = "localhost"
	BuildC2Port     = "443"
	BuildC2Protocol = "https"
	BuildC2Path     = "api/updates"
	BuildUserAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	BuildSleepTime  = "60"
	BuildJitter     = "20"

	// Implant log settings
	BuildImplantLogEnabled  = "false"
	BuildImplantLogFilePath = ""
	BuildImplantLogLevel    = "info"
)

// NewConfig creates a new configuration
func NewConfig() *Config {
	return &Config{
		C2Host:     BuildC2Host,
		C2Port:     BuildC2Port,
		C2Protocol: BuildC2Protocol,
		C2Path:     BuildC2Path,
		UserAgent:  BuildUserAgent,
		SleepTime:  BuildSleepTime,
		Jitter:     BuildJitter,
	}
}

// GetImplantLogEnabled returns whether the implant logging is enabled
func GetImplantLogEnabled() bool {
	return BuildImplantLogEnabled == "true"
}

// GetImplantLogFilePath returns the path of the implant log file
func GetImplantLogFilePath() string {
	if BuildImplantLogFilePath == "" {
		return "implant.log"
	}
	return BuildImplantLogFilePath
}

// GetImplantLogLevel returns the log level of the implant
func GetImplantLogLevel() string {
	if BuildImplantLogLevel == "" {
		return "info"
	}
	return BuildImplantLogLevel
}
