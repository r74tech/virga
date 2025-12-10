package extensions

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ResourceLimits defines limits for extension execution
type ResourceLimits struct {
	MaxMemoryMB      int           `json:"max_memory_mb"`      // Maximum memory in MB (0 = unlimited)
	MaxCPUPercent    int           `json:"max_cpu_percent"`    // Maximum CPU percentage (0 = unlimited)
	MaxExecutionTime time.Duration `json:"max_execution_time"` // Maximum execution time
	MaxThreads       int           `json:"max_threads"`        // Maximum number of threads/goroutines
}

// DefaultResourceLimits returns sensible default limits
func DefaultResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		MaxMemoryMB:      100,             // 100MB
		MaxCPUPercent:    50,              // 50% CPU
		MaxExecutionTime: 5 * time.Minute, // 5 minutes
		MaxThreads:       10,              // 10 threads/goroutines
	}
}

// ResourceMonitor monitors resource usage
type ResourceMonitor struct {
	limits      *ResourceLimits
	startTime   time.Time
	startMemory uint64
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.Mutex
	violations  []string
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(limits *ResourceLimits) *ResourceMonitor {
	if limits == nil {
		limits = DefaultResourceLimits()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceMonitor{
		limits:      limits,
		startTime:   time.Now(),
		startMemory: getCurrentMemory(),
		ctx:         ctx,
		cancel:      cancel,
		violations:  make([]string, 0),
	}
}

// Start starts monitoring resources
func (m *ResourceMonitor) Start() {
	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop stops monitoring
func (m *ResourceMonitor) Stop() {
	m.cancel()
	m.wg.Wait()
}

// GetContext returns a context that will be cancelled if limits are exceeded
func (m *ResourceMonitor) GetContext() context.Context {
	// Create a timeout context if execution time limit is set
	if m.limits.MaxExecutionTime > 0 {
		ctx, cancel := context.WithTimeout(m.ctx, m.limits.MaxExecutionTime)
		// Store cancel func to be called in Stop()
		m.mu.Lock()
		oldCancel := m.cancel
		m.cancel = func() {
			cancel()
			oldCancel()
		}
		m.mu.Unlock()
		return ctx
	}
	return m.ctx
}

// CheckViolations returns any resource limit violations
func (m *ResourceMonitor) CheckViolations() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.violations...)
}

// monitorLoop continuously monitors resources
func (m *ResourceMonitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkLimits()
		}
	}
}

// checkLimits checks if any limits are exceeded
func (m *ResourceMonitor) checkLimits() {
	// Check memory limit
	if m.limits.MaxMemoryMB > 0 {
		currentMem := getCurrentMemory()
		usedMem := currentMem - m.startMemory
		usedMB := usedMem / (1024 * 1024)

		if usedMB > uint64(m.limits.MaxMemoryMB) {
			violation := fmt.Sprintf("Memory limit exceeded: %dMB > %dMB", usedMB, m.limits.MaxMemoryMB)
			m.addViolation(violation)
			m.cancel() // Stop execution
		}
	}

	// Check execution time
	if m.limits.MaxExecutionTime > 0 {
		elapsed := time.Since(m.startTime)
		if elapsed > m.limits.MaxExecutionTime {
			violation := fmt.Sprintf("Execution time exceeded: %v > %v", elapsed, m.limits.MaxExecutionTime)
			m.addViolation(violation)
			m.cancel() // Stop execution
		}
	}

	// Check goroutine count
	if m.limits.MaxThreads > 0 {
		numGoroutines := runtime.NumGoroutine()
		if numGoroutines > m.limits.MaxThreads {
			violation := fmt.Sprintf("Thread limit exceeded: %d > %d", numGoroutines, m.limits.MaxThreads)
			m.addViolation(violation)
			// Don't cancel for this, just record
		}
	}
}

// addViolation records a violation
func (m *ResourceMonitor) addViolation(violation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.violations = append(m.violations, violation)
}

// getCurrentMemory returns current memory usage in bytes
func getCurrentMemory() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// LimitedExtension wraps an extension with resource limits
type LimitedExtension struct {
	Extension
	limits *ResourceLimits
}

// NewLimitedExtension creates a new limited extension
func NewLimitedExtension(ext Extension, limits *ResourceLimits) *LimitedExtension {
	return &LimitedExtension{
		Extension: ext,
		limits:    limits,
	}
}

// Execute executes the extension with resource limits
func (e *LimitedExtension) Execute(args map[string]interface{}) (*ExtensionResult, error) {
	// Create resource monitor
	monitor := NewResourceMonitor(e.limits)
	monitor.Start()
	defer monitor.Stop()

	// Create result channel
	resultChan := make(chan *ExtensionResult, 1)
	errorChan := make(chan error, 1)

	// Execute in goroutine with monitoring
	go func() {
		// Set GOMAXPROCS to limit CPU usage
		if e.limits.MaxCPUPercent > 0 {
			maxProcs := runtime.NumCPU() * e.limits.MaxCPUPercent / 100
			if maxProcs < 1 {
				maxProcs = 1
			}
			oldMaxProcs := runtime.GOMAXPROCS(maxProcs)
			defer runtime.GOMAXPROCS(oldMaxProcs)
		}

		result, err := e.Extension.Execute(args)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Wait for result or context cancellation
	select {
	case result := <-resultChan:
		// Check for violations even on success
		violations := monitor.CheckViolations()
		if len(violations) > 0 {
			result.Error = fmt.Sprintf("Resource violations: %v", violations)
		}
		return result, nil

	case err := <-errorChan:
		return nil, err

	case <-monitor.GetContext().Done():
		violations := monitor.CheckViolations()
		return &ExtensionResult{
			Success:  false,
			Error:    fmt.Sprintf("Extension terminated due to resource limits: %v", violations),
			ExitCode: -1,
		}, fmt.Errorf("resource limits exceeded")
	}
}

// ResourceLimitConfig holds resource limit configuration for extensions
type ResourceLimitConfig struct {
	// Default limits for all extensions
	DefaultLimits *ResourceLimits `json:"default_limits"`

	// Per-extension type limits
	ScriptLimits *ResourceLimits `json:"script_limits"`
	NativeLimits *ResourceLimits `json:"native_limits"`
	BOFLimits    *ResourceLimits `json:"bof_limits"`
	AliasLimits  *ResourceLimits `json:"alias_limits"`

	// Per-extension name limits (overrides type limits)
	ExtensionLimits map[string]*ResourceLimits `json:"extension_limits"`
}

// GetLimitsForExtension returns the appropriate limits for an extension
func (c *ResourceLimitConfig) GetLimitsForExtension(manifest *Manifest) *ResourceLimits {
	// Check for specific extension limits
	if c.ExtensionLimits != nil {
		if limits, ok := c.ExtensionLimits[manifest.Name]; ok {
			return limits
		}
	}

	// Check for type-specific limits
	switch manifest.Type {
	case "script":
		if c.ScriptLimits != nil {
			return c.ScriptLimits
		}
	case "native":
		if c.NativeLimits != nil {
			return c.NativeLimits
		}
	case "bof":
		if c.BOFLimits != nil {
			return c.BOFLimits
		}
	case "alias":
		if c.AliasLimits != nil {
			return c.AliasLimits
		}
	}

	// Fall back to default limits
	if c.DefaultLimits != nil {
		return c.DefaultLimits
	}

	return DefaultResourceLimits()
}
