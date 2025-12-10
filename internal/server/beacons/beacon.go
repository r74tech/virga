package beacons

import (
	"time"

	"github.com/google/uuid"
)

// Beacon represents a beacon configuration
type Beacon struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // http, https, dns, smb
	Active     bool                   `json:"active"`
	Config     map[string]interface{} `json:"config"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	LastActive time.Time              `json:"last_active"`
}

// NewBeacon creates a new beacon
func NewBeacon(name, beaconType string, config map[string]interface{}) *Beacon {
	now := time.Now()
	return &Beacon{
		ID:         uuid.New().String(),
		Name:       name,
		Type:       beaconType,
		Active:     true,
		Config:     config,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastActive: now,
	}
}

// Update updates beacon fields
func (b *Beacon) Update(name string, config map[string]interface{}) {
	b.Name = name
	b.Config = config
	b.UpdatedAt = time.Now()
}

// SetActive sets the beacon active status
func (b *Beacon) SetActive(active bool) {
	b.Active = active
	b.UpdatedAt = time.Now()
	if active {
		b.LastActive = time.Now()
	}
}

// UpdateLastActive updates the last active timestamp
func (b *Beacon) UpdateLastActive() {
	b.LastActive = time.Now()
}
