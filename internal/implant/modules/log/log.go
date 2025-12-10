package log

import (
	"fmt"
	"strings"

	"github.com/r74tech/virga/internal/implant/logger"
)

// LogModule is a module for controlling implant logging
type LogModule struct{}

// Name returns the module name
func (m *LogModule) Name() string {
	return "log"
}

// Execute handles log control commands
func (m *LogModule) Execute(args []string) (string, int, error) {
	if len(args) == 0 || (len(args) > 0 && strings.ToLower(args[0]) == "help") {
		return m.showHelp(), 0, nil
	}

	log := logger.Get()
	if !log.IsEnabled() {
		return "Logging is disabled", 1, nil
	}

	command := strings.ToLower(args[0])
	switch command {
	case "level":
		if len(args) < 2 {
			currentLevel := log.GetLevel()
			return fmt.Sprintf("Current log level: %s", currentLevel.String()), 0, nil
		}
		// Set new log level
		newLevel := args[1]
		log.SetLevelFromString(newLevel)
		return fmt.Sprintf("Log level set to: %s", newLevel), 0, nil

	case "disable":
		// Note: We can't actually disable a running logger instance
		// But we can set it to the highest level (OFF)
		log.SetLevelFromString("off")
		return "Logging disabled", 0, nil

	case "enable":
		// Reset to INFO level when enabling
		log.SetLevelFromString("info")
		return "Logging enabled", 0, nil

	case "status":
		level := log.GetLevel()
		path := log.GetLogFilePath()
		enabled := log.IsEnabled()

		status := fmt.Sprintf("Logging Status:\n")
		status += fmt.Sprintf("  Enabled: %v\n", enabled)
		status += fmt.Sprintf("  Level: %s\n", level.String())
		status += fmt.Sprintf("  Path: %s", path)
		return status, 0, nil

	default:
		return fmt.Sprintf("Unknown log command: %s\n%s", command, m.showHelp()), 1, nil
	}
}

func (m *LogModule) showHelp() string {
	help := "Log Module Commands:\n"
	help += "  log level [debug|info|warn|error|off]  - Get or set log level\n"
	help += "  log disable                             - Disable logging (set to OFF)\n"
	help += "  log enable                              - Enable logging (set to INFO)\n"
	help += "  log status                              - Show logging status\n"
	help += "  log help                                - Show this help"
	return help
}
