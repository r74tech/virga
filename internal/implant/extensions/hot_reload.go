package extensions

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// HotReloadConfig defines configuration for hot reload
type HotReloadConfig struct {
	Enabled       bool          `json:"enabled"`
	CheckInterval time.Duration `json:"check_interval"`
	AutoReload    bool          `json:"auto_reload"`
}

// DefaultHotReloadConfig returns default hot reload configuration
func DefaultHotReloadConfig() *HotReloadConfig {
	return &HotReloadConfig{
		Enabled:       false,
		CheckInterval: 5 * time.Second,
		AutoReload:    true,
	}
}

// ExtensionState tracks the state of a loaded extension
type ExtensionState struct {
	Path        string
	LoadTime    time.Time
	Version     string
	Checksum    string
	ModTime     time.Time
	ReloadCount int
	LastError   error
	LastErrorAt time.Time
}

// HotReloadManager manages hot reloading of extensions
type HotReloadManager struct {
	manager   *Manager
	config    *HotReloadConfig
	states    map[string]*ExtensionState // extension name -> state
	watcher   *fileWatcher
	mu        sync.RWMutex
	stopChan  chan struct{}
	wg        sync.WaitGroup
	callbacks HotReloadCallbacks
}

// HotReloadCallbacks defines callbacks for hot reload events
type HotReloadCallbacks struct {
	OnReloadStart    func(name string)
	OnReloadComplete func(name string, err error)
	OnReloadError    func(name string, err error)
}

// NewHotReloadManager creates a new hot reload manager
func NewHotReloadManager(manager *Manager, config *HotReloadConfig) *HotReloadManager {
	if config == nil {
		config = DefaultHotReloadConfig()
	}

	return &HotReloadManager{
		manager:  manager,
		config:   config,
		states:   make(map[string]*ExtensionState),
		stopChan: make(chan struct{}),
	}
}

// SetCallbacks sets the hot reload callbacks
func (h *HotReloadManager) SetCallbacks(callbacks HotReloadCallbacks) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callbacks = callbacks
}

// Start starts the hot reload manager
func (h *HotReloadManager) Start() error {
	if !h.config.Enabled {
		return fmt.Errorf("hot reload is disabled")
	}

	// Create file watcher
	h.watcher = newFileWatcher()

	// Start monitoring loop
	h.wg.Add(1)
	go h.monitorLoop()

	return nil
}

// Stop stops the hot reload manager
func (h *HotReloadManager) Stop() {
	close(h.stopChan)
	h.wg.Wait()

	if h.watcher != nil {
		h.watcher.Close()
	}
}

// RegisterExtension registers an extension for hot reload monitoring
func (h *HotReloadManager) RegisterExtension(name, path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat extension path: %w", err)
	}

	// Get extension manifest
	ext, exists := h.manager.GetExtension(name)
	if !exists {
		return fmt.Errorf("extension not found: %s", name)
	}

	manifest := ext.GetManifest()
	version := ""
	if manifest != nil {
		version = manifest.Version
	}

	// Create state
	state := &ExtensionState{
		Path:        path,
		LoadTime:    time.Now(),
		Version:     version,
		ModTime:     info.ModTime(),
		ReloadCount: 0,
	}

	// Calculate checksum
	state.Checksum, _ = h.calculateChecksum(path)

	h.states[name] = state

	// Add to file watcher
	if h.watcher != nil && h.config.AutoReload {
		h.watcher.Add(path)
	}

	return nil
}

// UnregisterExtension removes an extension from hot reload monitoring
func (h *HotReloadManager) UnregisterExtension(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if state, exists := h.states[name]; exists {
		if h.watcher != nil {
			h.watcher.Remove(state.Path)
		}
		delete(h.states, name)
	}
}

// ReloadExtension manually reloads an extension
func (h *HotReloadManager) ReloadExtension(name string) error {
	h.mu.RLock()
	state, exists := h.states[name]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("extension not registered for hot reload: %s", name)
	}

	// Call reload start callback
	if h.callbacks.OnReloadStart != nil {
		h.callbacks.OnReloadStart(name)
	}

	// Perform reload
	err := h.doReload(name, state.Path)

	// Update state
	h.mu.Lock()
	state.ReloadCount++
	if err != nil {
		state.LastError = err
		state.LastErrorAt = time.Now()
	} else {
		state.LoadTime = time.Now()
		// Update checksum
		state.Checksum, _ = h.calculateChecksum(state.Path)
	}
	h.mu.Unlock()

	// Call reload complete callback
	if h.callbacks.OnReloadComplete != nil {
		h.callbacks.OnReloadComplete(name, err)
	}

	if err != nil && h.callbacks.OnReloadError != nil {
		h.callbacks.OnReloadError(name, err)
	}

	return err
}

// doReload performs the actual reload operation
func (h *HotReloadManager) doReload(name, path string) error {
	// Verify extension exists before attempting reload
	_, exists := h.manager.GetExtension(name)
	if !exists {
		return fmt.Errorf("extension not found: %s", name)
	}

	// Unload current extension
	if err := h.manager.UnloadExtension(name); err != nil {
		return fmt.Errorf("failed to unload extension: %w", err)
	}

	// Reload from path
	if err := h.manager.LoadExtension(path); err != nil {
		// Try to restore previous extension on failure
		// This is best-effort, ignore errors
		_ = h.manager.LoadExtension(path)
		return fmt.Errorf("failed to reload extension: %w", err)
	}

	return nil
}

// monitorLoop monitors for extension changes
func (h *HotReloadManager) monitorLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return

		case <-ticker.C:
			h.checkForChanges()

		case event := <-h.watcher.Events():
			if h.config.AutoReload {
				h.handleFileEvent(event)
			}
		}
	}
}

// checkForChanges checks all registered extensions for changes
func (h *HotReloadManager) checkForChanges() {
	h.mu.RLock()
	states := make(map[string]*ExtensionState)
	for name, state := range h.states {
		states[name] = state
	}
	h.mu.RUnlock()

	for name, state := range states {
		if h.shouldReload(name, state) {
			_ = h.ReloadExtension(name)
		}
	}
}

// shouldReload checks if an extension should be reloaded
func (h *HotReloadManager) shouldReload(name string, state *ExtensionState) bool {
	// Check if file exists
	info, err := os.Stat(state.Path)
	if err != nil {
		return false
	}

	// Check modification time
	if info.ModTime().After(state.ModTime) {
		// Double-check with checksum
		newChecksum, err := h.calculateChecksum(state.Path)
		if err == nil && newChecksum != state.Checksum {
			return true
		}
	}

	return false
}

// handleFileEvent handles file system events
func (h *HotReloadManager) handleFileEvent(event FileEvent) {
	h.mu.RLock()
	var targetName string
	for name, state := range h.states {
		if state.Path == event.Path {
			targetName = name
			break
		}
	}
	h.mu.RUnlock()

	if targetName != "" && (event.Op == FileModified || event.Op == FileCreated) {
		_ = h.ReloadExtension(targetName)
	}
}

// calculateChecksum calculates the checksum of a file or directory
func (h *HotReloadManager) calculateChecksum(path string) (string, error) {
	// For simplicity, using modification time as checksum
	// In production, use proper checksumming
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", info.ModTime().UnixNano()), nil
}

// GetState returns the state of a registered extension
func (h *HotReloadManager) GetState(name string) *ExtensionState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.states[name]
}

// GetAllStates returns all extension states
func (h *HotReloadManager) GetAllStates() map[string]*ExtensionState {
	h.mu.RLock()
	defer h.mu.RUnlock()

	states := make(map[string]*ExtensionState)
	for name, state := range h.states {
		states[name] = state
	}
	return states
}

// FileOp represents a file operation
type FileOp int

const (
	FileCreated FileOp = iota
	FileModified
	FileDeleted
)

// FileEvent represents a file system event
type FileEvent struct {
	Path string
	Op   FileOp
}

// fileWatcher watches for file changes (simplified implementation)
type fileWatcher struct {
	paths  map[string]time.Time
	events chan FileEvent
	stop   chan struct{}
	mu     sync.Mutex
}

func newFileWatcher() *fileWatcher {
	w := &fileWatcher{
		paths:  make(map[string]time.Time),
		events: make(chan FileEvent, 100),
		stop:   make(chan struct{}),
	}
	go w.watch()
	return w
}

func (w *fileWatcher) Add(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := os.Stat(path)
	if err == nil {
		w.paths[path] = info.ModTime()
	}
}

func (w *fileWatcher) Remove(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.paths, path)
}

func (w *fileWatcher) Events() <-chan FileEvent {
	return w.events
}

func (w *fileWatcher) Close() {
	close(w.stop)
}

func (w *fileWatcher) watch() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.checkChanges()
		}
	}
}

func (w *fileWatcher) checkChanges() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for path, lastMod := range w.paths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				select {
				case w.events <- FileEvent{Path: path, Op: FileDeleted}:
				default:
				}
				delete(w.paths, path)
			}
			continue
		}

		if info.ModTime().After(lastMod) {
			select {
			case w.events <- FileEvent{Path: path, Op: FileModified}:
			default:
			}
			w.paths[path] = info.ModTime()
		}
	}
}
