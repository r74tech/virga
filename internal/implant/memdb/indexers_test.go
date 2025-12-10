package memdb

import (
	"bytes"
	"testing"
	"time"
)

// TestTimeFieldIndexFromObject tests extracting time from objects
func TestTimeFieldIndexFromObject(t *testing.T) {
	index := &TimeFieldIndex{Field: "Timestamp"}

	type TestStruct struct {
		ID        string
		Timestamp time.Time
		Other     string
	}

	type TestStructPtr struct {
		ID        string
		Timestamp *time.Time
		Other     string
	}

	now := time.Now()

	tests := []struct {
		name        string
		obj         interface{}
		expectOk    bool
		expectError bool
	}{
		{
			name: "Valid struct with time.Time",
			obj: &TestStruct{
				ID:        "test-1",
				Timestamp: now,
				Other:     "data",
			},
			expectOk:    true,
			expectError: false,
		},
		{
			name: "Valid struct with *time.Time",
			obj: &TestStructPtr{
				ID:        "test-2",
				Timestamp: &now,
				Other:     "data",
			},
			expectOk:    true,
			expectError: false,
		},
		{
			name: "Struct with nil *time.Time",
			obj: &TestStructPtr{
				ID:        "test-3",
				Timestamp: nil,
				Other:     "data",
			},
			expectOk:    false,
			expectError: false,
		},
		{
			name:        "Non-struct type",
			obj:         "not a struct",
			expectOk:    false,
			expectError: true,
		},
		{
			name: "Struct without field",
			obj: &struct {
				ID    string
				Other string
			}{
				ID:    "test-4",
				Other: "data",
			},
			expectOk:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, val, err := index.FromObject(tt.obj)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if ok != tt.expectOk {
				t.Errorf("Expected ok=%v, got %v", tt.expectOk, ok)
			}

			if ok && val == nil {
				t.Error("Expected non-nil value when ok=true")
			}
			if ok && len(val) != 8 {
				t.Errorf("Expected 8-byte value, got %d bytes", len(val))
			}
		})
	}
}

// TestTimeFieldIndexFromArgs tests creating index value from arguments
func TestTimeFieldIndexFromArgs(t *testing.T) {
	index := &TimeFieldIndex{Field: "Timestamp"}

	now := time.Now()

	tests := []struct {
		name        string
		args        []interface{}
		expectError bool
	}{
		{
			name:        "Valid time.Time argument",
			args:        []interface{}{now},
			expectError: false,
		},
		{
			name:        "Valid *time.Time argument",
			args:        []interface{}{&now},
			expectError: false,
		},
		{
			name:        "Nil *time.Time argument",
			args:        []interface{}{(*time.Time)(nil)},
			expectError: true,
		},
		{
			name:        "Wrong number of arguments",
			args:        []interface{}{now, now},
			expectError: true,
		},
		{
			name:        "No arguments",
			args:        []interface{}{},
			expectError: true,
		},
		{
			name:        "Wrong type argument",
			args:        []interface{}{"not a time"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := index.FromArgs(tt.args...)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if val == nil {
					t.Error("Expected non-nil value")
				}
				if len(val) != 8 {
					t.Errorf("Expected 8-byte value, got %d bytes", len(val))
				}
			}
		})
	}
}

// TestTimeFieldIndexOrdering tests that time values are correctly ordered
func TestTimeFieldIndexOrdering(t *testing.T) {
	index := &TimeFieldIndex{Field: "Timestamp"}

	// Create times in order
	time1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 6, 15, 12, 30, 0, 0, time.UTC)
	time3 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Get index values
	val1, err1 := index.FromArgs(time1)
	val2, err2 := index.FromArgs(time2)
	val3, err3 := index.FromArgs(time3)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatal("Failed to create index values")
	}

	// Verify ordering
	if bytes.Compare(val1, val2) >= 0 {
		t.Error("Expected val1 < val2")
	}
	if bytes.Compare(val2, val3) >= 0 {
		t.Error("Expected val2 < val3")
	}
	if bytes.Compare(val1, val3) >= 0 {
		t.Error("Expected val1 < val3")
	}
}

// TestTimeFieldIndexConsistency tests that same time produces same index
func TestTimeFieldIndexConsistency(t *testing.T) {
	index := &TimeFieldIndex{Field: "Timestamp"}

	now := time.Now()

	// Get index value multiple times
	val1, err1 := index.FromArgs(now)
	val2, err2 := index.FromArgs(now)

	if err1 != nil || err2 != nil {
		t.Fatal("Failed to create index values")
	}

	// Values should be identical
	if !bytes.Equal(val1, val2) {
		t.Error("Same time should produce same index value")
	}
}

// TestTimeFieldIndexEdgeCases tests edge cases
func TestTimeFieldIndexEdgeCases(t *testing.T) {
	index := &TimeFieldIndex{Field: "Timestamp"}

	// Test with zero time
	zeroTime := time.Time{}
	val, err := index.FromArgs(zeroTime)
	if err != nil {
		t.Errorf("Failed to index zero time: %v", err)
	}
	if val == nil {
		t.Error("Expected non-nil value for zero time")
	}

	// Test with far future time
	futureTime := time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
	val, err = index.FromArgs(futureTime)
	if err != nil {
		t.Errorf("Failed to index future time: %v", err)
	}
	if val == nil {
		t.Error("Expected non-nil value for future time")
	}

	// Test with Unix epoch
	epochTime := time.Unix(0, 0)
	val, err = index.FromArgs(epochTime)
	if err != nil {
		t.Errorf("Failed to index epoch time: %v", err)
	}
	if val == nil {
		t.Error("Expected non-nil value for epoch time")
	}
}

// TestTimeFieldIndexByteConversion tests the byte conversion logic
func TestTimeFieldIndexByteConversion(t *testing.T) {
	index := &TimeFieldIndex{Field: "Timestamp"}

	// Test specific time value
	testTime := time.Unix(1234567890, 123456789) // Known Unix timestamp

	val, err := index.FromArgs(testTime)
	if err != nil {
		t.Fatalf("Failed to create index value: %v", err)
	}

	// Convert back to verify
	var reconstructed int64
	for i := 0; i < 8; i++ {
		reconstructed |= int64(val[i]) << (56 - i*8)
	}

	originalNano := testTime.UnixNano()
	if reconstructed != originalNano {
		t.Errorf("Byte conversion mismatch: expected %d, got %d", originalNano, reconstructed)
	}
}

// TestTimeFieldIndexDifferentFields tests indexing different field names
func TestTimeFieldIndexDifferentFields(t *testing.T) {
	type MultiTimeStruct struct {
		CreatedAt time.Time
		UpdatedAt time.Time
		DeletedAt *time.Time
	}

	now := time.Now()
	later := now.Add(1 * time.Hour)

	obj := &MultiTimeStruct{
		CreatedAt: now,
		UpdatedAt: later,
		DeletedAt: nil,
	}

	// Test different fields
	fields := []struct {
		name      string
		expectOk  bool
		fieldName string
	}{
		{"CreatedAt field", true, "CreatedAt"},
		{"UpdatedAt field", true, "UpdatedAt"},
		{"DeletedAt field (nil)", false, "DeletedAt"},
		{"NonExistent field", false, "NonExistent"},
	}

	for _, f := range fields {
		t.Run(f.name, func(t *testing.T) {
			index := &TimeFieldIndex{Field: f.fieldName}
			ok, val, err := index.FromObject(obj)

			if f.expectOk {
				if !ok || err != nil {
					t.Errorf("Expected successful indexing for field %s", f.fieldName)
				}
				if val == nil || len(val) != 8 {
					t.Errorf("Expected 8-byte value for field %s", f.fieldName)
				}
			} else {
				if ok && err == nil && f.fieldName != "DeletedAt" {
					t.Errorf("Expected failure for field %s", f.fieldName)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkTimeFieldIndexFromObject(b *testing.B) {
	index := &TimeFieldIndex{Field: "Timestamp"}

	type TestStruct struct {
		ID        string
		Timestamp time.Time
		Data      string
	}

	obj := &TestStruct{
		ID:        "bench-1",
		Timestamp: time.Now(),
		Data:      "benchmark data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = index.FromObject(obj)
	}
}

func BenchmarkTimeFieldIndexFromArgs(b *testing.B) {
	index := &TimeFieldIndex{Field: "Timestamp"}
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = index.FromArgs(now)
	}
}

func BenchmarkTimeFieldIndexComparison(b *testing.B) {
	index := &TimeFieldIndex{Field: "Timestamp"}

	time1 := time.Now()
	time2 := time1.Add(1 * time.Hour)

	val1, _ := index.FromArgs(time1)
	val2, _ := index.FromArgs(time2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bytes.Compare(val1, val2)
	}
}
