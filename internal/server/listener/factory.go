package listener

import (
	"fmt"

	"github.com/r74tech/virga/internal/server/config"
)

// NewListener creates a listener of the specified type
func NewListener(cfg config.ListenerConfig) (Listener, error) {
	switch cfg.Type {
	case "http", "https":
		return NewHTTPListener(cfg), nil
	case "mtls":
		// TODO: Implement the mTLS listener type
		return nil, fmt.Errorf("mtls listener type not implemented yet")
	case "dns":
		// TODO: Implement the DNS listener type
		return nil, fmt.Errorf("dns listener type not implemented yet")
	case "smb":
		// TODO: Implement the SMB listener type
		return nil, fmt.Errorf("smb listener type not implemented yet")
	default:
		return nil, fmt.Errorf("unknown listener type: %s", cfg.Type)
	}
}
