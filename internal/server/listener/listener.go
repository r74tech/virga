package listener

import (
	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/protocol"
)

// BeaconManager interface for beacon configuration management
type BeaconManager interface {
	GetBeaconConfig(sessionID string) (*beacons.BeaconConfig, error)
}

// Global beacon manager reference
var globalBeaconManager BeaconManager

// SetBeaconManager sets the global beacon manager
func SetBeaconManager(manager BeaconManager) {
	globalBeaconManager = manager
}

// GetBeaconManager returns the global beacon manager
func GetBeaconManager() BeaconManager {
	return globalBeaconManager
}

// Listener is the C2 listener interface
type Listener interface {
	Name() string
	Start(agentHandler func(string, protocol.AgentConnection)) error
	Stop() error
	SetDebugMode(debug bool)
}
