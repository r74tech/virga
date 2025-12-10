package netstat

import (
	"fmt"

	"github.com/r74tech/virga/internal/implant/logger"
)

// NetstatModule lists network connections
type NetstatModule struct{}

// Name returns the module name
func (m *NetstatModule) Name() string {
	return "netstat"
}

// Execute lists network connections
func (m *NetstatModule) Execute(args []string) (string, int, error) {
	log := logger.Get()
	log.Debug("Netstat module executing", map[string]interface{}{})

	// Call platform-specific implementation
	connections, err := getNetworkConnections()
	if err != nil {
		log.LogCommand("netstat", "", 1, err)
		return fmt.Sprintf("Failed to list network connections: %s", err), 1, err
	}

	log.LogCommand("netstat", connections, 0, nil)
	return connections, 0, nil
}
