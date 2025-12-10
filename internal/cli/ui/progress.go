package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/shared/logger"
)

// ProgressBar represents a progress bar
type ProgressBar struct {
	mu             sync.RWMutex
	Total          int
	Current        int
	Width          int
	ShowPercentage bool
	ShowTime       bool
	StartTime      time.Time
	noColor        bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total, width int) *ProgressBar {
	return &ProgressBar{
		Total:          total,
		Width:          width,
		ShowPercentage: true,
		ShowTime:       true,
		StartTime:      time.Now(),
		noColor:        shouldDisableColors(),
	}
}

// Update updates the progress bar
func (p *ProgressBar) Update(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Current = current
	p.draw()
}

// SetCurrent sets the current progress value
func (p *ProgressBar) SetCurrent(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if current < 0 {
		p.Current = 0
	} else if current > p.Total {
		p.Current = p.Total
	} else {
		p.Current = current
	}
}

// GetCurrent returns the current progress value
func (p *ProgressBar) GetCurrent() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Current
}

// Increment increments the current progress by the given amount
func (p *ProgressBar) Increment(amount int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	newCurrent := p.Current + amount
	if newCurrent > p.Total {
		p.Current = p.Total
	} else if newCurrent < 0 {
		p.Current = 0
	} else {
		p.Current = newCurrent
	}
}

// GetPercent returns the current progress as a percentage
func (p *ProgressBar) GetPercent() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Total == 0 {
		return 0
	}
	return float64(p.Current) / float64(p.Total) * 100
}

// String returns a string representation of the progress bar
func (p *ProgressBar) String() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Total == 0 {
		return "[0.00%]"
	}

	percent := float64(p.Current) / float64(p.Total)
	filled := int(percent * float64(p.Width))

	bar := strings.Repeat("=", filled)
	if filled < p.Width {
		bar += ">"
		bar += strings.Repeat("-", p.Width-filled-1)
	}

	return fmt.Sprintf("[%s] %.2f%%", bar, percent*100)
}

// Render returns a colored string representation of the progress bar
func (p *ProgressBar) Render(useColor bool) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Total == 0 {
		return "[0.00%]"
	}

	percent := float64(p.Current) / float64(p.Total)
	filled := int(percent * float64(p.Width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.Width-filled)

	// Color the bar
	if useColor {
		if percent < 0.5 {
			bar = colorize(ColorYellow, bar, false)
		} else if percent < 1.0 {
			bar = colorize(ColorBlue, bar, false)
		} else {
			bar = colorize(ColorGreen, bar, false)
		}
	}

	return fmt.Sprintf("[%s] %.2f%%", bar, percent*100)
}

// draw renders the progress bar
func (p *ProgressBar) draw() {
	if p.Total == 0 {
		return
	}

	percent := float64(p.Current) / float64(p.Total)
	filled := int(percent * float64(p.Width))

	// Build the bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.Width-filled)

	// Color the bar
	if !p.noColor {
		if percent < 0.5 {
			bar = colorize(ColorYellow, bar, p.noColor)
		} else if percent < 1.0 {
			bar = colorize(ColorBlue, bar, p.noColor)
		} else {
			bar = colorize(ColorGreen, bar, p.noColor)
		}
	}

	// Use logger for progress output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("\r[%s] ", bar))

	if p.ShowPercentage {
		output.WriteString(fmt.Sprintf("%.1f%% ", percent*100))
	}

	if p.ShowTime {
		elapsed := time.Since(p.StartTime)
		output.WriteString(fmt.Sprintf("(%s) ", formatDuration(elapsed)))
	}

	if p.Current >= p.Total {
		logger.Info(output.String())
	} else {
		// For partial updates, we still need to use fmt.Printf for carriage return to work
		fmt.Print(output.String())
	}
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.Update(p.Total)
}

// Spinner represents a loading spinner
type Spinner struct {
	mu      sync.Mutex
	Frames  []string
	Current int
	message string
	stop    chan bool
	stopped bool
	noColor bool
}

// NewSpinner creates a new spinner
func NewSpinner() *Spinner {
	return &Spinner{
		Frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		message: "",
		stop:    make(chan bool, 1),
		noColor: shouldDisableColors(),
	}
}

// Next returns the next frame and advances the spinner
func (s *Spinner) Next() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	frame := s.Frames[s.Current]
	s.Current = (s.Current + 1) % len(s.Frames)
	return frame
}

// String returns the current frame
func (s *Spinner) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.Frames[s.Current]
}

// Render returns the spinner with a message
func (s *Spinner) Render(message string, useColor bool) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	frame := s.Frames[s.Current]
	if useColor {
		frame = colorize(ColorCyan, frame, false)
	}
	return fmt.Sprintf("%s %s", frame, message)
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				// Clear the line
				logger.Info(fmt.Sprintf("\r%s\r", strings.Repeat(" ", len(s.message)+10)))
				return
			case <-ticker.C:
				s.mu.Lock()
				frame := s.Frames[s.Current]
				s.Current = (s.Current + 1) % len(s.Frames)
				s.mu.Unlock()

				color := ColorCyan
				if s.noColor {
					color = ""
				}

				// For spinner animation, we need to use fmt.Printf for carriage return to work
				fmt.Printf("\r%s %s %s", colorize(color, frame, s.noColor), s.message, strings.Repeat(" ", 10))
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.stopped {
		s.stopped = true
		close(s.stop)
	}
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// FormatDuration formats a duration for display with extended format
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// LoadingIndicator represents a loading indicator with message
type LoadingIndicator struct {
	mu      sync.Mutex
	message string
	spinner *Spinner
	running bool
	stop    chan bool
}

// NewLoadingIndicator creates a new loading indicator
func NewLoadingIndicator(message string) *LoadingIndicator {
	return &LoadingIndicator{
		message: message,
		spinner: NewSpinner(),
		running: true,
		stop:    make(chan bool, 1),
	}
}

// IsRunning returns whether the indicator is running
func (l *LoadingIndicator) IsRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.running
}

// Stop stops the loading indicator
func (l *LoadingIndicator) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.running {
		l.running = false
		close(l.stop)
	}
}

// BytesProgressBar represents a progress bar for byte operations
type BytesProgressBar struct {
	*ProgressBar
	totalBytes   int64
	currentBytes int64
}

// NewBytesProgressBar creates a new bytes progress bar
func NewBytesProgressBar(totalBytes int64, width int) *BytesProgressBar {
	return &BytesProgressBar{
		ProgressBar:  NewProgressBar(100, width),
		totalBytes:   totalBytes,
		currentBytes: 0,
	}
}

// SetCurrentBytes sets the current byte count
func (b *BytesProgressBar) SetCurrentBytes(bytes int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.currentBytes = bytes
	if b.totalBytes > 0 {
		percent := int(float64(bytes) / float64(b.totalBytes) * 100)
		b.Current = percent
	}
}

// String returns a string representation with byte counts
func (b *BytesProgressBar) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	percent := float64(b.currentBytes) / float64(b.totalBytes)
	filled := int(percent * float64(b.Width))

	bar := strings.Repeat("=", filled)
	if filled < b.Width {
		bar += ">"
		bar += strings.Repeat("-", b.Width-filled-1)
	}

	return fmt.Sprintf("[%s] %.2f%% %s / %s",
		bar,
		percent*100,
		FormatBytes(b.currentBytes),
		FormatBytes(b.totalBytes))
}
