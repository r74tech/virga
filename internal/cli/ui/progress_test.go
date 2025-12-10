package ui

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProgressBar_New(t *testing.T) {
	pb := NewProgressBar(100, 50)

	if pb.Total != 100 {
		t.Errorf("NewProgressBar() Total = %v, want %v", pb.Total, 100)
	}

	if pb.Width != 50 {
		t.Errorf("NewProgressBar() Width = %v, want %v", pb.Width, 50)
	}

	if pb.Current != 0 {
		t.Errorf("NewProgressBar() Current = %v, want %v", pb.Current, 0)
	}
}

func TestProgressBar_SetCurrent(t *testing.T) {
	pb := NewProgressBar(100, 50)

	pb.SetCurrent(50)
	if pb.GetCurrent() != 50 {
		t.Errorf("SetCurrent(50) resulted in Current = %v, want 50", pb.GetCurrent())
	}

	// Test clamping to max
	pb.SetCurrent(150)
	if pb.GetCurrent() != 100 {
		t.Errorf("SetCurrent(150) resulted in Current = %v, want 100", pb.GetCurrent())
	}

	// Test negative value
	pb.SetCurrent(-10)
	if pb.GetCurrent() != 0 {
		t.Errorf("SetCurrent(-10) resulted in Current = %v, want 0", pb.GetCurrent())
	}
}

func TestProgressBar_Increment(t *testing.T) {
	pb := NewProgressBar(100, 50)

	pb.SetCurrent(30)
	pb.Increment(20)
	if pb.GetCurrent() != 50 {
		t.Errorf("Increment(20) from 30 resulted in Current = %v, want 50", pb.GetCurrent())
	}

	// Test overflow
	pb.SetCurrent(90)
	pb.Increment(20)
	if pb.GetCurrent() != 100 {
		t.Errorf("Increment(20) from 90 resulted in Current = %v, want 100", pb.GetCurrent())
	}
}

func TestProgressBar_GetPercent(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		current int
		want    float64
	}{
		{
			name:    "halfway",
			total:   100,
			current: 50,
			want:    50.0,
		},
		{
			name:    "quarter",
			total:   100,
			current: 25,
			want:    25.0,
		},
		{
			name:    "complete",
			total:   100,
			current: 100,
			want:    100.0,
		},
		{
			name:    "zero",
			total:   100,
			current: 0,
			want:    0.0,
		},
		{
			name:    "non-round percentage",
			total:   3,
			current: 1,
			want:    33.33,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, 50)
			pb.SetCurrent(tt.current)
			got := pb.GetPercent()
			// Allow small floating point differences
			if got < tt.want-0.01 || got > tt.want+0.01 {
				t.Errorf("GetPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProgressBar_String(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		current int
		width   int
		check   func(string) bool
	}{
		{
			name:    "empty progress",
			total:   100,
			current: 0,
			width:   20,
			check: func(s string) bool {
				return strings.Contains(s, "[") &&
					strings.Contains(s, "]") &&
					strings.Contains(s, "0.00%") &&
					!strings.Contains(s, "=")
			},
		},
		{
			name:    "half progress",
			total:   100,
			current: 50,
			width:   20,
			check: func(s string) bool {
				return strings.Contains(s, "[") &&
					strings.Contains(s, "]") &&
					strings.Contains(s, "50.00%") &&
					strings.Contains(s, "=")
			},
		},
		{
			name:    "full progress",
			total:   100,
			current: 100,
			width:   20,
			check: func(s string) bool {
				return strings.Contains(s, "[") &&
					strings.Contains(s, "]") &&
					strings.Contains(s, "100.00%")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, tt.width)
			pb.SetCurrent(tt.current)
			got := pb.String()
			if !tt.check(got) {
				t.Errorf("String() = %v, check failed", got)
			}
		})
	}
}

func TestProgressBar_Render(t *testing.T) {
	pb := NewProgressBar(100, 30)
	pb.SetCurrent(75)

	output := pb.Render(true)

	// Should contain color codes when enabled
	if !strings.Contains(output, "\033[") {
		t.Error("Render(true) should contain ANSI color codes")
	}

	// Should contain percentage
	if !strings.Contains(output, "75.00%") {
		t.Error("Render() should contain percentage")
	}

	// Test without colors
	outputNoColor := pb.Render(false)
	if strings.Contains(outputNoColor, "\033[") {
		t.Error("Render(false) should not contain ANSI color codes")
	}
}

func TestProgressBar_Concurrent(t *testing.T) {
	pb := NewProgressBar(1000, 50)

	// Run concurrent increments
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				pb.Increment(1)
			}
		}()
	}

	wg.Wait()

	// Should have incremented to 1000
	if pb.GetCurrent() != 1000 {
		t.Errorf("Concurrent increments resulted in Current = %v, want 1000", pb.GetCurrent())
	}
}

func TestSpinner_New(t *testing.T) {
	spinner := NewSpinner()

	if spinner.Current != 0 {
		t.Errorf("NewSpinner() Current = %v, want 0", spinner.Current)
	}

	if len(spinner.Frames) == 0 {
		t.Error("NewSpinner() should have frames")
	}
}

func TestSpinner_Next(t *testing.T) {
	spinner := NewSpinner()

	first := spinner.Next()
	if first == "" {
		t.Error("Next() should return a non-empty frame")
	}

	// Advance through all frames
	frames := make([]string, 0)
	for i := 0; i < len(spinner.Frames)*2; i++ {
		frame := spinner.Next()
		if len(frames) < len(spinner.Frames) {
			frames = append(frames, frame)
		}
	}

	// Should have collected all unique frames
	if len(frames) != len(spinner.Frames) {
		t.Errorf("Expected %d unique frames, got %d", len(spinner.Frames), len(frames))
	}
}

func TestSpinner_String(t *testing.T) {
	spinner := NewSpinner()

	str := spinner.String()
	if str == "" {
		t.Error("String() should return a non-empty frame")
	}

	// Should be one of the frames
	found := false
	for _, frame := range spinner.Frames {
		if str == frame {
			found = true
			break
		}
	}

	if !found {
		t.Error("String() should return one of the spinner frames")
	}
}

func TestSpinner_Render(t *testing.T) {
	spinner := NewSpinner()

	// With colors
	rendered := spinner.Render("Loading", true)
	if !strings.Contains(rendered, "Loading") {
		t.Error("Render() should contain the message")
	}
	if !strings.Contains(rendered, "\033[") {
		t.Error("Render(true) should contain ANSI color codes")
	}

	// Without colors
	renderedNoColor := spinner.Render("Loading", false)
	if !strings.Contains(renderedNoColor, "Loading") {
		t.Error("Render() should contain the message")
	}
	if strings.Contains(renderedNoColor, "\033[") {
		t.Error("Render(false) should not contain ANSI color codes")
	}
}

func TestSpinner_Concurrent(t *testing.T) {
	spinner := NewSpinner()

	// Run concurrent Next() calls
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = spinner.Next()
			}
		}()
	}

	wg.Wait()

	// Should still be able to get frames
	frame := spinner.Next()
	if frame == "" {
		t.Error("Next() should still work after concurrent access")
	}
}

func TestLoadingIndicator(t *testing.T) {
	indicator := NewLoadingIndicator("Processing")

	if !indicator.IsRunning() {
		t.Error("NewLoadingIndicator() should start in running state")
	}

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	indicator.Stop()

	if indicator.IsRunning() {
		t.Error("Stop() should set running to false")
	}
}

func TestBytesProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		total   int64
		current int64
		width   int
		check   func(string) bool
	}{
		{
			name:    "bytes display",
			total:   1024 * 1024, // 1MB
			current: 512 * 1024,  // 512KB
			width:   50,
			check: func(s string) bool {
				return strings.Contains(s, "512.0 KB") &&
					strings.Contains(s, "1.0 MB") &&
					strings.Contains(s, "50.00%")
			},
		},
		{
			name:    "gigabytes display",
			total:   5 * 1024 * 1024 * 1024, // 5GB
			current: 1024 * 1024 * 1024,     // 1GB
			width:   50,
			check: func(s string) bool {
				return strings.Contains(s, "1.0 GB") &&
					strings.Contains(s, "5.0 GB")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewBytesProgressBar(tt.total, tt.width)
			pb.SetCurrentBytes(tt.current)
			got := pb.String()
			if !tt.check(got) {
				t.Errorf("String() = %v, check failed", got)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{90 * time.Second, "1m 30s"},
		{3600 * time.Second, "1h 0m 0s"},
		{3665 * time.Second, "1h 1m 5s"},
		{86400 * time.Second, "1d 0h 0m 0s"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}
