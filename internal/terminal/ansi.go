package terminal

import (
	"fmt"
	"strings"
)

// ANSI escape sequences for styling and colors

// Text attributes
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Dim        = "\033[2m"
	Italic     = "\033[3m"
	Underline  = "\033[4m"
	Blink      = "\033[5m"
	Reverse    = "\033[7m"
	Hidden     = "\033[8m"
	Strikethrough = "\033[9m"
)

// Basic 16 colors (0-15)
const (
	FgBlack   = "\033[30m"
	FgRed     = "\033[31m"
	FgGreen   = "\033[32m"
	FgYellow  = "\033[33m"
	FgBlue    = "\033[34m"
	FgMagenta = "\033[35m"
	FgCyan    = "\033[36m"
	FgWhite   = "\033[37m"
	
	FgBrightBlack   = "\033[90m"
	FgBrightRed     = "\033[91m"
	FgBrightGreen   = "\033[92m"
	FgBrightYellow  = "\033[93m"
	FgBrightBlue    = "\033[94m"
	FgBrightMagenta = "\033[95m"
	FgBrightCyan    = "\033[96m"
	FgBrightWhite   = "\033[97m"
	
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
	
	BgBrightBlack   = "\033[100m"
	BgBrightRed     = "\033[101m"
	BgBrightGreen   = "\033[102m"
	BgBrightYellow  = "\033[103m"
	BgBrightBlue    = "\033[104m"
	BgBrightMagenta = "\033[105m"
	BgBrightCyan    = "\033[106m"
	BgBrightWhite   = "\033[107m"
)

// Cursor operations
const (
	CursorSave    = "\033[s"
	CursorRestore = "\033[u"
	CursorHide    = "\033[?25l"
	CursorShow    = "\033[?25h"
)

// Screen operations
const (
	ClearScreen = "\033[2J"
	ClearLine   = "\033[2K"
	ClearToEOL  = "\033[K"
	ClearToEOS  = "\033[J"
)

// Color256 returns ANSI escape sequence for 256-color mode
func Color256(fg, bg int) string {
	var result strings.Builder
	if fg >= 0 && fg <= 255 {
		fmt.Fprintf(&result, "\033[38;5;%dm", fg)
	}
	if bg >= 0 && bg <= 255 {
		fmt.Fprintf(&result, "\033[48;5;%dm", bg)
	}
	return result.String()
}

// ColorRGB returns ANSI escape sequence for true color (24-bit) mode
func ColorRGB(fgR, fgG, fgB, bgR, bgG, bgB int) string {
	var result strings.Builder
	if fgR >= 0 && fgG >= 0 && fgB >= 0 {
		fmt.Fprintf(&result, "\033[38;2;%d;%d;%dm", fgR, fgG, fgB)
	}
	if bgR >= 0 && bgG >= 0 && bgB >= 0 {
		fmt.Fprintf(&result, "\033[48;2;%d;%d;%dm", bgR, bgG, bgB)
	}
	return result.String()
}

// MoveCursor returns ANSI escape sequence to move cursor to position
func MoveCursor(x, y int) string {
	return fmt.Sprintf("\033[%d;%dH", y+1, x+1)
}

// MoveCursorUp returns ANSI escape sequence to move cursor up n lines
func MoveCursorUp(n int) string {
	return fmt.Sprintf("\033[%dA", n)
}

// MoveCursorDown returns ANSI escape sequence to move cursor down n lines
func MoveCursorDown(n int) string {
	return fmt.Sprintf("\033[%dB", n)
}

// MoveCursorRight returns ANSI escape sequence to move cursor right n columns
func MoveCursorRight(n int) string {
	return fmt.Sprintf("\033[%dC", n)
}

// MoveCursorLeft returns ANSI escape sequence to move cursor left n columns
func MoveCursorLeft(n int) string {
	return fmt.Sprintf("\033[%dD", n)
}

// SetCursorStyle sets the cursor appearance
func SetCursorStyle(style CursorStyle) string {
	switch style {
	case CursorStyleBar:
		return "\033[6 q" // Steady bar
	case CursorStyleUnderline:
		return "\033[4 q" // Steady underline
	default:
		return "\033[2 q" // Steady block
	}
}

// StyleBuilder helps build complex styled strings
type StyleBuilder struct {
	parts []string
}

// NewStyleBuilder creates a new style builder
func NewStyleBuilder() *StyleBuilder {
	return &StyleBuilder{
		parts: make([]string, 0, 10),
	}
}

// Add adds text with optional styling
func (sb *StyleBuilder) Add(text string, styles ...string) *StyleBuilder {
	if len(styles) > 0 {
		sb.parts = append(sb.parts, strings.Join(styles, ""))
		sb.parts = append(sb.parts, text)
		sb.parts = append(sb.parts, Reset)
	} else {
		sb.parts = append(sb.parts, text)
	}
	return sb
}

// AddColor256 adds text with 256-color styling
func (sb *StyleBuilder) AddColor256(text string, fg, bg int) *StyleBuilder {
	sb.parts = append(sb.parts, Color256(fg, bg))
	sb.parts = append(sb.parts, text)
	sb.parts = append(sb.parts, Reset)
	return sb
}

// AddRGB adds text with true color styling
func (sb *StyleBuilder) AddRGB(text string, fgR, fgG, fgB, bgR, bgG, bgB int) *StyleBuilder {
	sb.parts = append(sb.parts, ColorRGB(fgR, fgG, fgB, bgR, bgG, bgB))
	sb.parts = append(sb.parts, text)
	sb.parts = append(sb.parts, Reset)
	return sb
}

// String returns the built string
func (sb *StyleBuilder) String() string {
	return strings.Join(sb.parts, "")
}

// Helper function to convert hex color to RGB
func HexToRGB(hex string) (r, g, b int) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) == 6 {
		fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	}
	return
}