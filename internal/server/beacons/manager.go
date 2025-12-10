package beacons

import (
	"fmt"
	"sync"
	"time"
)

// BeaconConfig represents the beacon configuration
type BeaconConfig struct {
	SleepTime         int       // Sleep time in seconds
	Jitter            int       // Jitter rate (0-50%)
	MaxMissedCheckins int       // Maximum missed checkins
	KillDate          time.Time // Kill date
}

// Manager manages the beacon configurations
type Manager struct {
	beaconConfigs  map[string]*BeaconConfig // sessionID -> configuration
	lastCheckins   map[string]time.Time     // sessionID -> last checkin
	missedCheckins map[string]int           // sessionID -> missed checkin count
	beacons        map[string]*Beacon       // beaconID -> Beacon
	mu             sync.RWMutex
}

// NewManager creates a new beacon manager
func NewManager() *Manager {
	return &Manager{
		beaconConfigs:  make(map[string]*BeaconConfig),
		lastCheckins:   make(map[string]time.Time),
		missedCheckins: make(map[string]int),
		beacons:        make(map[string]*Beacon),
	}
}

// RegisterBeacon registers a new beacon configuration
func (m *Manager) RegisterBeacon(sessionID string, config *BeaconConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.beaconConfigs[sessionID] = config
	m.lastCheckins[sessionID] = time.Now()
	m.missedCheckins[sessionID] = 0
}

// GetBeaconConfig retrieves the beacon configuration for a session
func (m *Manager) GetBeaconConfig(sessionID string) (*BeaconConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.beaconConfigs[sessionID]
	if !exists {
		return nil, fmt.Errorf("no beacon configuration for session %s", sessionID)
	}

	return config, nil
}

// UpdateBeaconConfig updates the beacon configuration
func (m *Manager) UpdateBeaconConfig(sessionID string, sleepTime, jitter int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, exists := m.beaconConfigs[sessionID]
	if !exists {
		return fmt.Errorf("no beacon configuration for session %s", sessionID)
	}

	// Validate the values
	if sleepTime < 0 {
		return fmt.Errorf("sleep time must be non-negative")
	}

	if jitter < 0 || jitter > 50 {
		return fmt.Errorf("jitter must be between 0 and 50")
	}

	// Update the configuration
	config.SleepTime = sleepTime
	config.Jitter = jitter

	return nil
}

// RecordCheckin records the checkin of a beacon
func (m *Manager) RecordCheckin(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastCheckins[sessionID] = time.Now()
	m.missedCheckins[sessionID] = 0
}

// GetExpectedNextCheckin calculates the expected next checkin time
func (m *Manager) GetExpectedNextCheckin(sessionID string) (time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.beaconConfigs[sessionID]
	if !exists {
		return time.Time{}, fmt.Errorf("no beacon configuration for session %s", sessionID)
	}

	lastCheckin, exists := m.lastCheckins[sessionID]
	if !exists {
		return time.Time{}, fmt.Errorf("no checkin record for session %s", sessionID)
	}

	// Calculate the next checkin time (considering the maximum jitter)
	maxJitterSeconds := float64(config.SleepTime) * float64(config.Jitter) / 100.0
	maxNextCheckin := lastCheckin.Add(time.Duration(float64(config.SleepTime)+maxJitterSeconds) * time.Second)

	return maxNextCheckin, nil
}

// CheckMissedBeacons checks for missed beacons
func (m *Manager) CheckMissedBeacons() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var missingSessionIDs []string
	now := time.Now()

	for sessionID, config := range m.beaconConfigs {
		lastCheckin, exists := m.lastCheckins[sessionID]
		if !exists {
			continue
		}

		// Calculate the expected checkin time considering the maximum jitter
		maxJitterSeconds := float64(config.SleepTime) * float64(config.Jitter) / 100.0
		expectedCheckin := lastCheckin.Add(time.Duration(float64(config.SleepTime)+maxJitterSeconds) * time.Second)

		// If the expected time has passed
		if now.After(expectedCheckin) {
			m.missedCheckins[sessionID]++

			// If the maximum missed checkins is exceeded
			if m.missedCheckins[sessionID] > config.MaxMissedCheckins {
				missingSessionIDs = append(missingSessionIDs, sessionID)
			}
		}
	}

	return missingSessionIDs
}

// RemoveBeacon removes the beacon configuration
func (m *Manager) RemoveBeacon(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.beaconConfigs, sessionID)
	delete(m.lastCheckins, sessionID)
	delete(m.missedCheckins, sessionID)
}

// GetStatus gets the beacon status for a specified session
func (m *Manager) GetStatus(sessionID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.beaconConfigs[sessionID]
	if !exists {
		return nil, fmt.Errorf("no beacon configuration for session %s", sessionID)
	}

	lastCheckin, exists := m.lastCheckins[sessionID]
	if !exists {
		return nil, fmt.Errorf("no checkin record for session %s", sessionID)
	}

	status := map[string]interface{}{
		"sleep_time":      config.SleepTime,
		"jitter":          config.Jitter,
		"last_checkin":    lastCheckin,
		"missed_checkins": m.missedCheckins[sessionID],
		"kill_date":       config.KillDate,
	}

	return status, nil
}

// Configuration represents the beacon configuration
type Configuration struct {
	Interval   int
	Jitter     int
	MaxRetries int
	RetryDelay int
	LastUpdate time.Time
}

// GetConfiguration gets the current beacon configuration
func (m *Manager) GetConfiguration() Configuration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return the default configuration
	return Configuration{
		Interval:   30,
		Jitter:     20,
		MaxRetries: 3,
		RetryDelay: 5,
		LastUpdate: time.Now(),
	}
}

// GetActiveBeaconCount gets the number of active beacons
func (m *Manager) GetActiveBeaconCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Count the active beacons
	count := 0
	now := time.Now()
	for sessionID, config := range m.beaconConfigs {
		lastCheckin, exists := m.lastCheckins[sessionID]
		if !exists {
			continue
		}

		// Calculate the expected checkin time considering the maximum jitter
		maxJitterSeconds := float64(config.SleepTime) * float64(config.Jitter) / 100.0
		expectedCheckin := lastCheckin.Add(time.Duration(float64(config.SleepTime)+maxJitterSeconds) * time.Second)

		// If the beacon is still active
		if now.Before(expectedCheckin) {
			count++
		}
	}

	return count
}

// GetTotalBeaconCount gets the total number of beacons
func (m *Manager) GetTotalBeaconCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.beaconConfigs)
}

// GetFailedBeaconCount gets the number of failed beacons
func (m *Manager) GetFailedBeaconCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Count the beacons that have exceeded the maximum missed checkins
	count := 0
	for sessionID, config := range m.beaconConfigs {
		if m.missedCheckins[sessionID] > config.MaxMissedCheckins {
			count++
		}
	}

	return count
}

// ListenerInfo represents the listener information
type ListenerInfo struct {
	ID           string
	Type         string
	Address      string
	Port         int
	Protocol     string
	Status       string
	SessionCount int
}

// GetListeners gets the listener information
func (m *Manager) GetListeners() []ListenerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return the demo data
	return []ListenerInfo{
		{
			ID:           "http-listener-1",
			Type:         "http",
			Address:      "0.0.0.0",
			Port:         8080,
			Protocol:     "http",
			Status:       "active",
			SessionCount: m.GetActiveBeaconCount(),
		},
	}
}

// ListBeacons returns all beacons
func (m *Manager) ListBeacons() []*Beacon {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var beacons []*Beacon
	for _, beacon := range m.beacons {
		beacons = append(beacons, beacon)
	}
	return beacons
}

// GetBeacon returns a specific beacon by its ID
func (m *Manager) GetBeacon(beaconID string) *Beacon {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.beacons[beaconID]
}

// CreateBeacon creates a new beacon with the given name, type, and configuration
func (m *Manager) CreateBeacon(name, beaconType string, config map[string]interface{}) (*Beacon, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	beacon := NewBeacon(name, beaconType, config)
	m.beacons[beacon.ID] = beacon

	return beacon, nil
}

// DeleteBeacon deletes a beacon by its ID
func (m *Manager) DeleteBeacon(beaconID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.beacons[beaconID]; !exists {
		return fmt.Errorf("beacon not found: %s", beaconID)
	}

	delete(m.beacons, beaconID)
	return nil
}

// Count returns the total number of beacons in the manager
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.beacons)
}
