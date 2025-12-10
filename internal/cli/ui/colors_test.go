package ui

import (
	"testing"
)

func TestColorize(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		color    string
		enabled  bool
		expected string
	}{
		{
			name:     "red with colors enabled",
			text:     "error",
			color:    ColorRed,
			enabled:  true,
			expected: ColorRed + "error" + ColorReset,
		},
		{
			name:     "green with colors enabled",
			text:     "success",
			color:    ColorGreen,
			enabled:  true,
			expected: ColorGreen + "success" + ColorReset,
		},
		{
			name:     "colors disabled",
			text:     "text",
			color:    ColorBlue,
			enabled:  false,
			expected: "text",
		},
		{
			name:     "empty text",
			text:     "",
			color:    ColorYellow,
			enabled:  true,
			expected: "",
		},
		{
			name:     "multiline text",
			text:     "line1\nline2",
			color:    ColorMagenta,
			enabled:  true,
			expected: ColorMagenta + "line1\nline2" + ColorReset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Colorize(tt.text, tt.color, tt.enabled)
			if got != tt.expected {
				t.Errorf("Colorize() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRed(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "error message",
			enabled:  true,
			expected: ColorRed + "error message" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "error message",
			enabled:  false,
			expected: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Red(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Red() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGreen(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "success",
			enabled:  true,
			expected: ColorGreen + "success" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "success",
			enabled:  false,
			expected: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Green(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Green() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestYellow(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "warning",
			enabled:  true,
			expected: ColorYellow + "warning" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "warning",
			enabled:  false,
			expected: "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Yellow(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Yellow() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBlue(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "info",
			enabled:  true,
			expected: ColorBlue + "info" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "info",
			enabled:  false,
			expected: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Blue(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Blue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMagenta(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "special",
			enabled:  true,
			expected: ColorMagenta + "special" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "special",
			enabled:  false,
			expected: "special",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Magenta(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Magenta() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCyan(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "highlight",
			enabled:  true,
			expected: ColorCyan + "highlight" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "highlight",
			enabled:  false,
			expected: "highlight",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Cyan(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Cyan() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGray(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "muted",
			enabled:  true,
			expected: ColorGray + "muted" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "muted",
			enabled:  false,
			expected: "muted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Gray(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Gray() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBold(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "important",
			enabled:  true,
			expected: ColorBold + "important" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "important",
			enabled:  false,
			expected: "important",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Bold(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Bold() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDim(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "subtle",
			enabled:  true,
			expected: ColorDim + "subtle" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "subtle",
			enabled:  false,
			expected: "subtle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Dim(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Dim() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUnderline(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enabled  bool
		expected string
	}{
		{
			name:     "with colors",
			text:     "link",
			enabled:  true,
			expected: ColorUnderline + "link" + ColorReset,
		},
		{
			name:     "without colors",
			text:     "link",
			enabled:  false,
			expected: "link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Underline(tt.text, tt.enabled)
			if got != tt.expected {
				t.Errorf("Underline() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "simple color",
			text:     ColorRed + "error" + ColorReset,
			expected: "error",
		},
		{
			name:     "multiple colors",
			text:     ColorRed + "error " + ColorGreen + "success" + ColorReset,
			expected: "error success",
		},
		{
			name:     "no colors",
			text:     "plain text",
			expected: "plain text",
		},
		{
			name:     "complex ANSI codes",
			text:     "\033[1;32mBold Green\033[0m \033[4mUnderlined\033[0m",
			expected: "Bold Green Underlined",
		},
		{
			name:     "256 color codes",
			text:     "\033[38;5;196mRed Text\033[0m",
			expected: "Red Text",
		},
		{
			name:     "RGB color codes",
			text:     "\033[38;2;255;0;0mRGB Red\033[0m",
			expected: "RGB Red",
		},
		{
			name:     "cursor movement",
			text:     "Text\033[2Awith\033[2Bcursor\033[5Dmovement",
			expected: "Textwithcursormovement",
		},
		{
			name:     "empty string",
			text:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.text)
			if got != tt.expected {
				t.Errorf("StripANSI() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHasANSI(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "has color code",
			text: ColorRed + "error" + ColorReset,
			want: true,
		},
		{
			name: "no ANSI codes",
			text: "plain text",
			want: false,
		},
		{
			name: "has escape but not ANSI",
			text: "text with \\ backslash",
			want: false,
		},
		{
			name: "has cursor movement",
			text: "text\033[2A",
			want: true,
		},
		{
			name: "empty string",
			text: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasANSI(tt.text)
			if got != tt.want {
				t.Errorf("HasANSI() = %v, want %v", got, tt.want)
			}
		})
	}
}
