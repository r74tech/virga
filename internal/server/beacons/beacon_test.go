package beacons

import (
	"testing"
	"time"
)

func TestNewBeacon(t *testing.T) {
	name := "test-beacon"
	beaconType := "http"
	config := map[string]interface{}{
		"host": "example.com",
		"port": 8080,
	}

	beacon := NewBeacon(name, beaconType, config)

	if beacon == nil {
		t.Fatal("Expected beacon, got nil")
	}

	// Check fields
	if beacon.ID == "" {
		t.Error("Expected beacon ID to be set")
	}

	if beacon.Name != name {
		t.Errorf("Expected name %s, got %s", name, beacon.Name)
	}

	if beacon.Type != beaconType {
		t.Errorf("Expected type %s, got %s", beaconType, beacon.Type)
	}

	if !beacon.Active {
		t.Error("Expected beacon to be active by default")
	}

	if beacon.Config["host"] != "example.com" {
		t.Error("Config not set correctly")
	}

	// Check timestamps
	if beacon.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if beacon.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	if beacon.LastActive.IsZero() {
		t.Error("LastActive should be set")
	}

	// Timestamps should be equal at creation
	if !beacon.CreatedAt.Equal(beacon.UpdatedAt) {
		t.Error("CreatedAt and UpdatedAt should be equal at creation")
	}
}

func TestBeaconUpdate(t *testing.T) {
	beacon := NewBeacon("old-name", "http", map[string]interface{}{
		"host": "old.com",
	})

	oldUpdatedAt := beacon.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	newName := "new-name"
	newConfig := map[string]interface{}{
		"host": "new.com",
		"port": 9090,
	}

	beacon.Update(newName, newConfig)

	// Check updates
	if beacon.Name != newName {
		t.Errorf("Expected name %s, got %s", newName, beacon.Name)
	}

	if beacon.Config["host"] != "new.com" {
		t.Error("Config not updated correctly")
	}

	if beacon.Config["port"] != 9090 {
		t.Error("Config port not updated correctly")
	}

	// UpdatedAt should be newer
	if !beacon.UpdatedAt.After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}

	// CreatedAt should not change
	if !beacon.CreatedAt.Equal(oldUpdatedAt) {
		t.Error("CreatedAt should not change on update")
	}
}

func TestSetActive(t *testing.T) {
	beacon := NewBeacon("test", "http", nil)

	// Initially active
	if !beacon.Active {
		t.Error("Beacon should be active initially")
	}

	oldUpdatedAt := beacon.UpdatedAt
	oldLastActive := beacon.LastActive
	time.Sleep(10 * time.Millisecond)

	// Deactivate
	beacon.SetActive(false)

	if beacon.Active {
		t.Error("Beacon should be inactive")
	}

	if !beacon.UpdatedAt.After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}

	// LastActive should not change when deactivating
	if !beacon.LastActive.Equal(oldLastActive) {
		t.Error("LastActive should not change when deactivating")
	}

	time.Sleep(10 * time.Millisecond)

	// Reactivate
	beacon.SetActive(true)

	if !beacon.Active {
		t.Error("Beacon should be active")
	}

	// LastActive should be updated when activating
	if !beacon.LastActive.After(oldLastActive) {
		t.Error("LastActive should be updated when activating")
	}
}

func TestUpdateLastActive(t *testing.T) {
	beacon := NewBeacon("test", "http", nil)

	oldLastActive := beacon.LastActive
	time.Sleep(10 * time.Millisecond)

	beacon.UpdateLastActive()

	if !beacon.LastActive.After(oldLastActive) {
		t.Error("LastActive should be updated")
	}

	// Other timestamps should not change
	if !beacon.CreatedAt.Equal(beacon.UpdatedAt) {
		t.Error("Other timestamps should not change")
	}
}

func TestBeaconTypes(t *testing.T) {
	types := []string{"http", "https", "dns", "smb"}

	for _, beaconType := range types {
		beacon := NewBeacon("test", beaconType, nil)
		if beacon.Type != beaconType {
			t.Errorf("Expected type %s, got %s", beaconType, beacon.Type)
		}
	}
}

func TestBeaconConfigVariations(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]interface{}
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name:   "empty config",
			config: map[string]interface{}{},
		},
		{
			name: "complex config",
			config: map[string]interface{}{
				"host":    "example.com",
				"port":    8080,
				"secure":  true,
				"headers": []string{"X-Custom-Header"},
				"nested": map[string]interface{}{
					"key": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beacon := NewBeacon("test", "http", tt.config)

			if tt.config == nil && beacon.Config != nil {
				t.Error("Expected nil config")
			}

			if tt.config != nil && len(tt.config) != len(beacon.Config) {
				t.Error("Config not copied correctly")
			}
		})
	}
}
