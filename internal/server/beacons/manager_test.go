package beacons

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager()

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.beaconConfigs == nil {
		t.Error("beaconConfigs map is nil")
	}
	if m.lastCheckins == nil {
		t.Error("lastCheckins map is nil")
	}
	if m.missedCheckins == nil {
		t.Error("missedCheckins map is nil")
	}
	if m.beacons == nil {
		t.Error("beacons map is nil")
	}
}

func TestRegisterBeacon(t *testing.T) {
	m := NewManager()
	sessionID := "test-session-123"
	config := &BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
		KillDate:          time.Now().Add(24 * time.Hour),
	}

	m.RegisterBeacon(sessionID, config)

	// Verify beacon was registered
	if _, exists := m.beaconConfigs[sessionID]; !exists {
		t.Error("Beacon config not registered")
	}

	// Verify initial check-in was recorded
	if _, exists := m.lastCheckins[sessionID]; !exists {
		t.Error("Initial check-in not recorded")
	}

	// Verify missed check-ins initialized to 0
	if count := m.missedCheckins[sessionID]; count != 0 {
		t.Errorf("Missed check-ins = %d, want 0", count)
	}
}

func TestUpdateBeaconConfig(t *testing.T) {
	m := NewManager()
	sessionID := "test-session-456"

	// Test updating non-existent beacon
	err := m.UpdateBeaconConfig(sessionID, 60, 30)
	if err == nil {
		t.Error("Expected error for non-existent beacon")
	}

	// Register beacon
	config := &BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
		KillDate:          time.Now().Add(24 * time.Hour),
	}
	m.RegisterBeacon(sessionID, config)

	// Test valid update
	err = m.UpdateBeaconConfig(sessionID, 60, 30)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify update
	updatedConfig := m.beaconConfigs[sessionID]
	if updatedConfig.SleepTime != 60 {
		t.Errorf("SleepTime = %d, want 60", updatedConfig.SleepTime)
	}
	if updatedConfig.Jitter != 30 {
		t.Errorf("Jitter = %d, want 30", updatedConfig.Jitter)
	}

	// Test invalid sleep time
	err = m.UpdateBeaconConfig(sessionID, -1, 30)
	if err == nil {
		t.Error("Expected error for negative sleep time")
	}

	// Test invalid jitter (negative)
	err = m.UpdateBeaconConfig(sessionID, 60, -1)
	if err == nil {
		t.Error("Expected error for negative jitter")
	}

	// Test invalid jitter (> 50)
	err = m.UpdateBeaconConfig(sessionID, 60, 51)
	if err == nil {
		t.Error("Expected error for jitter > 50")
	}
}

func TestRecordCheckin(t *testing.T) {
	m := NewManager()
	sessionID := "test-session-789"

	// Register beacon
	config := &BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
		KillDate:          time.Now().Add(24 * time.Hour),
	}
	m.RegisterBeacon(sessionID, config)

	// Set missed check-ins to simulate previous misses
	m.missedCheckins[sessionID] = 2
	originalCheckin := m.lastCheckins[sessionID]

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Record new check-in
	m.RecordCheckin(sessionID)

	// Verify last check-in was updated
	newCheckin := m.lastCheckins[sessionID]
	if !newCheckin.After(originalCheckin) {
		t.Error("Last check-in time not updated")
	}

	// Verify missed check-ins reset to 0
	if count := m.missedCheckins[sessionID]; count != 0 {
		t.Errorf("Missed check-ins = %d, want 0", count)
	}
}

func TestGetExpectedNextCheckin(t *testing.T) {
	m := NewManager()
	sessionID := "test-session-abc"

	// Test non-existent beacon
	_, err := m.GetExpectedNextCheckin(sessionID)
	if err == nil {
		t.Error("Expected error for non-existent beacon")
	}

	// Register beacon
	config := &BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
		KillDate:          time.Now().Add(24 * time.Hour),
	}
	m.RegisterBeacon(sessionID, config)

	// Get expected next check-in
	nextCheckin, err := m.GetExpectedNextCheckin(sessionID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Calculate expected time with max jitter
	lastCheckin := m.lastCheckins[sessionID]
	maxJitterSeconds := float64(config.SleepTime) * float64(config.Jitter) / 100.0
	expectedTime := lastCheckin.Add(time.Duration(float64(config.SleepTime)+maxJitterSeconds) * time.Second)

	// Allow small time difference for test execution
	timeDiff := nextCheckin.Sub(expectedTime).Abs()
	if timeDiff > time.Millisecond {
		t.Errorf("Next check-in time mismatch: got %v, want %v", nextCheckin, expectedTime)
	}
}

func TestCheckMissedBeacons(t *testing.T) {
	m := NewManager()

	// Register multiple beacons
	beacons := []struct {
		sessionID string
		config    *BeaconConfig
		lastSeen  time.Duration
		missed    int
	}{
		{
			sessionID: "beacon-1",
			config: &BeaconConfig{
				SleepTime:         10,
				Jitter:            20,
				MaxMissedCheckins: 2,
			},
			lastSeen: -15 * time.Second, // Should be marked as missed
			missed:   2,
		},
		{
			sessionID: "beacon-2",
			config: &BeaconConfig{
				SleepTime:         10,
				Jitter:            20,
				MaxMissedCheckins: 2,
			},
			lastSeen: -20 * time.Second, // Should exceed max missed
			missed:   3,
		},
		{
			sessionID: "beacon-3",
			config: &BeaconConfig{
				SleepTime:         10,
				Jitter:            20,
				MaxMissedCheckins: 2,
			},
			lastSeen: -5 * time.Second, // Should be fine
			missed:   0,
		},
	}

	// Set up beacons
	now := time.Now()
	for _, b := range beacons {
		m.RegisterBeacon(b.sessionID, b.config)
		m.lastCheckins[b.sessionID] = now.Add(b.lastSeen)
		m.missedCheckins[b.sessionID] = b.missed
	}

	// Check missed beacons
	missingIDs := m.CheckMissedBeacons()

	// Should report both beacon-1 and beacon-2 as missing
	// beacon-1: was at 2, incremented to 3, exceeds max of 2
	// beacon-2: was at 3, incremented to 4, exceeds max of 2
	if len(missingIDs) != 2 {
		t.Errorf("Missing beacon count = %d, want 2", len(missingIDs))
	}

	// Verify both beacons are in the list
	foundBeacon1 := false
	foundBeacon2 := false
	for _, id := range missingIDs {
		if id == "beacon-1" {
			foundBeacon1 = true
		}
		if id == "beacon-2" {
			foundBeacon2 = true
		}
	}

	if !foundBeacon1 {
		t.Error("beacon-1 not found in missing list")
	}
	if !foundBeacon2 {
		t.Error("beacon-2 not found in missing list")
	}

	// Verify missed counts were incremented for overdue beacons
	if count := m.missedCheckins["beacon-1"]; count != 3 {
		t.Errorf("beacon-1 missed count = %d, want 3", count)
	}
	if count := m.missedCheckins["beacon-2"]; count != 4 {
		t.Errorf("beacon-2 missed count = %d, want 4", count)
	}
	if count := m.missedCheckins["beacon-3"]; count != 0 {
		t.Errorf("beacon-3 missed count = %d, want 0", count)
	}
}

func TestRemoveBeacon(t *testing.T) {
	m := NewManager()
	sessionID := "test-remove"

	// Register beacon
	config := &BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
	}
	m.RegisterBeacon(sessionID, config)

	// Verify beacon exists
	if _, exists := m.beaconConfigs[sessionID]; !exists {
		t.Fatal("Beacon not registered")
	}

	// Remove beacon
	m.RemoveBeacon(sessionID)

	// Verify beacon was removed from all maps
	if _, exists := m.beaconConfigs[sessionID]; exists {
		t.Error("Beacon config still exists")
	}
	if _, exists := m.lastCheckins[sessionID]; exists {
		t.Error("Last check-in still exists")
	}
	if _, exists := m.missedCheckins[sessionID]; exists {
		t.Error("Missed check-ins still exists")
	}
}

func TestGetStatus(t *testing.T) {
	m := NewManager()
	sessionID := "test-status"

	// Test non-existent beacon
	_, err := m.GetStatus(sessionID)
	if err == nil {
		t.Error("Expected error for non-existent beacon")
	}

	// Register beacon
	killDate := time.Now().Add(24 * time.Hour)
	config := &BeaconConfig{
		SleepTime:         45,
		Jitter:            25,
		MaxMissedCheckins: 5,
		KillDate:          killDate,
	}
	m.RegisterBeacon(sessionID, config)
	m.missedCheckins[sessionID] = 2

	// Get status
	status, err := m.GetStatus(sessionID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify status fields
	if sleepTime, ok := status["sleep_time"].(int); !ok || sleepTime != 45 {
		t.Errorf("sleep_time = %v, want 45", status["sleep_time"])
	}
	if jitter, ok := status["jitter"].(int); !ok || jitter != 25 {
		t.Errorf("jitter = %v, want 25", status["jitter"])
	}
	if missed, ok := status["missed_checkins"].(int); !ok || missed != 2 {
		t.Errorf("missed_checkins = %v, want 2", status["missed_checkins"])
	}
	if kd, ok := status["kill_date"].(time.Time); !ok || !kd.Equal(killDate) {
		t.Errorf("kill_date = %v, want %v", status["kill_date"], killDate)
	}
	if _, ok := status["last_checkin"].(time.Time); !ok {
		t.Error("last_checkin missing or wrong type")
	}
}

func TestBeaconCounts(t *testing.T) {
	m := NewManager()

	// Initially should be 0
	if count := m.GetTotalBeaconCount(); count != 0 {
		t.Errorf("Initial total count = %d, want 0", count)
	}
	if count := m.GetActiveBeaconCount(); count != 0 {
		t.Errorf("Initial active count = %d, want 0", count)
	}
	if count := m.GetFailedBeaconCount(); count != 0 {
		t.Errorf("Initial failed count = %d, want 0", count)
	}

	// Add beacons
	now := time.Now()

	// Active beacon
	m.RegisterBeacon("active-1", &BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
	})

	// Timed out beacon (not failed yet)
	m.RegisterBeacon("timeout-1", &BeaconConfig{
		SleepTime:         10,
		Jitter:            0,
		MaxMissedCheckins: 3,
	})
	m.lastCheckins["timeout-1"] = now.Add(-20 * time.Second)
	m.missedCheckins["timeout-1"] = 2

	// Failed beacon
	m.RegisterBeacon("failed-1", &BeaconConfig{
		SleepTime:         10,
		Jitter:            0,
		MaxMissedCheckins: 2,
	})
	m.lastCheckins["failed-1"] = now.Add(-30 * time.Second)
	m.missedCheckins["failed-1"] = 3

	// Verify counts
	if count := m.GetTotalBeaconCount(); count != 3 {
		t.Errorf("Total count = %d, want 3", count)
	}
	if count := m.GetActiveBeaconCount(); count != 1 {
		t.Errorf("Active count = %d, want 1", count)
	}
	if count := m.GetFailedBeaconCount(); count != 1 {
		t.Errorf("Failed count = %d, want 1", count)
	}
}

func TestCreateAndManageBeacons(t *testing.T) {
	m := NewManager()

	// Create beacon
	name := "Test HTTP Beacon"
	beaconType := "http"
	config := map[string]interface{}{
		"url":     "http://example.com",
		"timeout": 30,
	}

	beacon, err := m.CreateBeacon(name, beaconType, config)
	if err != nil {
		t.Fatalf("Failed to create beacon: %v", err)
	}

	// Verify beacon was created
	if beacon.Name != name {
		t.Errorf("Beacon name = %s, want %s", beacon.Name, name)
	}
	if beacon.Type != beaconType {
		t.Errorf("Beacon type = %s, want %s", beacon.Type, beaconType)
	}

	// Verify beacon can be retrieved
	retrieved := m.GetBeacon(beacon.ID)
	if retrieved == nil {
		t.Fatal("Failed to retrieve beacon")
	}
	if retrieved.ID != beacon.ID {
		t.Error("Retrieved beacon has different ID")
	}

	// List beacons
	beacons := m.ListBeacons()
	if len(beacons) != 1 {
		t.Errorf("Beacon count = %d, want 1", len(beacons))
	}

	// Verify count
	if count := m.Count(); count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}

	// Delete beacon
	err = m.DeleteBeacon(beacon.ID)
	if err != nil {
		t.Errorf("Failed to delete beacon: %v", err)
	}

	// Verify beacon was deleted
	if retrieved := m.GetBeacon(beacon.ID); retrieved != nil {
		t.Error("Beacon still exists after deletion")
	}
	if count := m.Count(); count != 0 {
		t.Errorf("Count() = %d after deletion, want 0", count)
	}

	// Try to delete non-existent beacon
	err = m.DeleteBeacon("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent beacon")
	}
}

func TestConcurrentBeaconOperations(t *testing.T) {
	m := NewManager()
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 4) // 4 types of operations

	// Concurrent beacon registration
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sessionID := fmt.Sprintf("session-%d-%d", id, j)
				config := &BeaconConfig{
					SleepTime:         30,
					Jitter:            20,
					MaxMissedCheckins: 3,
				}
				m.RegisterBeacon(sessionID, config)
			}
		}(i)
	}

	// Concurrent check-ins
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sessionID := fmt.Sprintf("session-%d-%d", id, j)
				m.RecordCheckin(sessionID)
			}
		}(i)
	}

	// Concurrent status checks
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sessionID := fmt.Sprintf("session-%d-%d", id, j)
				m.GetStatus(sessionID)
			}
		}(i)
	}

	// Concurrent missed beacon checks
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				m.CheckMissedBeacons()
			}
		}()
	}

	wg.Wait()

	// Verify final state
	expectedTotal := numGoroutines * numOperations
	if total := m.GetTotalBeaconCount(); total != expectedTotal {
		t.Errorf("Total beacon count = %d, want %d", total, expectedTotal)
	}
}

func TestGetConfiguration(t *testing.T) {
	m := NewManager()
	config := m.GetConfiguration()

	// Verify default values
	if config.Interval != 30 {
		t.Errorf("Interval = %d, want 30", config.Interval)
	}
	if config.Jitter != 20 {
		t.Errorf("Jitter = %d, want 20", config.Jitter)
	}
	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}
	if config.RetryDelay != 5 {
		t.Errorf("RetryDelay = %d, want 5", config.RetryDelay)
	}
	if config.LastUpdate.IsZero() {
		t.Error("LastUpdate is zero")
	}
}

func TestGetListeners(t *testing.T) {
	m := NewManager()

	// Add some beacons to have active count
	for i := 0; i < 5; i++ {
		sessionID := fmt.Sprintf("test-session-%d", i)
		config := &BeaconConfig{
			SleepTime:         30,
			Jitter:            20,
			MaxMissedCheckins: 3,
		}
		m.RegisterBeacon(sessionID, config)
	}

	listeners := m.GetListeners()

	// Should have at least one listener
	if len(listeners) == 0 {
		t.Fatal("No listeners returned")
	}

	// Verify listener info
	listener := listeners[0]
	if listener.ID == "" {
		t.Error("Listener ID is empty")
	}
	if listener.Type != "http" {
		t.Errorf("Listener type = %s, want http", listener.Type)
	}
	if listener.Port != 8080 {
		t.Errorf("Listener port = %d, want 8080", listener.Port)
	}
	if listener.Status != "active" {
		t.Errorf("Listener status = %s, want active", listener.Status)
	}
	if listener.SessionCount != 5 {
		t.Errorf("Session count = %d, want 5", listener.SessionCount)
	}
}

// Benchmark tests
func BenchmarkRegisterBeacon(b *testing.B) {
	m := NewManager()
	config := &BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
		KillDate:          time.Now().Add(24 * time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := fmt.Sprintf("bench-session-%d", i)
		m.RegisterBeacon(sessionID, config)
	}
}

func BenchmarkRecordCheckin(b *testing.B) {
	m := NewManager()
	sessionID := "bench-session"
	config := &BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
	}
	m.RegisterBeacon(sessionID, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordCheckin(sessionID)
	}
}

func BenchmarkCheckMissedBeacons(b *testing.B) {
	m := NewManager()

	// Register many beacons
	for i := 0; i < 1000; i++ {
		sessionID := fmt.Sprintf("bench-session-%d", i)
		config := &BeaconConfig{
			SleepTime:         30,
			Jitter:            20,
			MaxMissedCheckins: 3,
		}
		m.RegisterBeacon(sessionID, config)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.CheckMissedBeacons()
	}
}

func BenchmarkConcurrentOperations(b *testing.B) {
	m := NewManager()

	// Pre-register some beacons
	for i := 0; i < 100; i++ {
		sessionID := fmt.Sprintf("bench-session-%d", i)
		config := &BeaconConfig{
			SleepTime:         30,
			Jitter:            20,
			MaxMissedCheckins: 3,
		}
		m.RegisterBeacon(sessionID, config)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sessionID := fmt.Sprintf("bench-session-%d", i%100)

			// Mix of operations
			switch i % 4 {
			case 0:
				m.RecordCheckin(sessionID)
			case 1:
				m.GetStatus(sessionID)
			case 2:
				m.GetExpectedNextCheckin(sessionID)
			case 3:
				m.CheckMissedBeacons()
			}
			i++
		}
	})
}
