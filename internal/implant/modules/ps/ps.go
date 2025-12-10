package ps

import (
	"fmt"

	"github.com/r74tech/virga/internal/implant/logger"
)

// PsModule lists running processes
type PsModule struct{}

// Name returns the module name
func (m *PsModule) Name() string {
	return "ps"
}

// Execute lists running processes
func (m *PsModule) Execute(args []string) (string, int, error) {
	log := logger.Get()
	log.Debug("Ps module executing", map[string]interface{}{})

	// Call platform-specific implementation
	processes, err := getProcessList()
	if err != nil {
		log.LogCommand("ps", "", 1, err)
		return fmt.Sprintf("Failed to list processes: %s", err), 1, err
	}

	log.LogCommand("ps", processes, 0, nil)
	return processes, 0, nil
}
