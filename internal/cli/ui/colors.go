package ui

import (
	"os"
	"regexp"
	"runtime"
	"strings"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// ANSI color codes
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorPurple  = "\033[35m" // Alias for Magenta
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[37m"
	ColorWhite   = "\033[97m"

	// Styles
	ColorBold      = "\033[1m"
	ColorDim       = "\033[2m"
	ColorUnderline = "\033[4m"
	StyleBold      = "\033[1m"
	StyleDim       = "\033[2m"
	StyleItalic    = "\033[3m"
	StyleUnderline = "\033[4m"
)

// ColorScheme defines the colors used for different message types
type ColorScheme struct {
	Info    string
	Success string
	Warning string
	Error   string
	Debug   string
	Reset   string
}

// DefaultColorScheme provides the default color scheme
var DefaultColorScheme = ColorScheme{
	Info:    ColorBlue,
	Success: ColorGreen,
	Warning: ColorYellow,
	Error:   ColorRed,
	Debug:   ColorGray,
	Reset:   ColorReset,
}

// NoColorScheme provides a color scheme with no colors
var NoColorScheme = ColorScheme{
	Info:    "",
	Success: "",
	Warning: "",
	Error:   "",
	Debug:   "",
	Reset:   "",
}

// shouldDisableColors determines if colors should be disabled
func shouldDisableColors() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return true
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "dumb" {
		return true
	}

	// Disable colors on Windows unless in Windows Terminal
	if runtime.GOOS == "windows" && os.Getenv("WT_SESSION") == "" {
		return true
	}

	// Check if output is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil || (fileInfo.Mode()&os.ModeCharDevice) == 0 {
		return true
	}

	return false
}

// colorize applies color to text if colors are enabled
func colorize(color, text string, noColor bool) string {
	if noColor || color == "" {
		return text
	}
	return color + text + ColorReset
}

// Colorize is the public version of colorize
func Colorize(text, color string, enabled bool) string {
	if text == "" {
		return ""
	}
	return colorize(color, text, !enabled)
}

// Color helper functions
func Red(text string, enabled bool) string {
	return Colorize(text, ColorRed, enabled)
}

func Green(text string, enabled bool) string {
	return Colorize(text, ColorGreen, enabled)
}

func Yellow(text string, enabled bool) string {
	return Colorize(text, ColorYellow, enabled)
}

func Blue(text string, enabled bool) string {
	return Colorize(text, ColorBlue, enabled)
}

func Magenta(text string, enabled bool) string {
	return Colorize(text, ColorMagenta, enabled)
}

func Cyan(text string, enabled bool) string {
	return Colorize(text, ColorCyan, enabled)
}

func Gray(text string, enabled bool) string {
	return Colorize(text, ColorGray, enabled)
}

func Bold(text string, enabled bool) string {
	return Colorize(text, ColorBold, enabled)
}

func Dim(text string, enabled bool) string {
	return Colorize(text, ColorDim, enabled)
}

func Underline(text string, enabled bool) string {
	return Colorize(text, ColorUnderline, enabled)
}

// StripANSI removes all ANSI escape sequences from a string
func StripANSI(text string) string {
	return ansiRegex.ReplaceAllString(text, "")
}

// HasANSI checks if a string contains ANSI escape sequences
func HasANSI(text string) bool {
	return strings.Contains(text, "\033[") || strings.Contains(text, "\x1b[")
}
