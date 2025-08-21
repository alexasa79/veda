package editor

import (
	"veda/internal/ui"
)

const (
	// Unicode characters for whitespace visualization
	tabChar   = '→' // U+2192 RIGHTWARDS ARROW
	spaceChar = '·' // U+00B7 MIDDLE DOT
)

// whitespaceStyle represents the visual style for whitespace characters
type whitespaceStyle struct {
	showTabs           bool
	showTrailingSpaces bool
	// Style for whitespace indicators (dimmed)
	style ui.Style
}

// newWhitespaceStyle creates a default whitespace style
func newWhitespaceStyle(baseStyle ui.Style) whitespaceStyle {
	// Use the provided base style which should already be configured
	// with a dim/subdued color from the theme
	return whitespaceStyle{
		showTabs:           true,
		showTrailingSpaces: true,
		style:              baseStyle,
	}
}

// detectTrailingSpaces finds the position where trailing spaces begin in a line
// Returns -1 if there are no trailing spaces
func detectTrailingSpaces(line string) int {
	if len(line) == 0 {
		return -1
	}

	// Find the last non-space character
	lastNonSpace := len(line) - 1
	for lastNonSpace >= 0 && line[lastNonSpace] == ' ' {
		lastNonSpace--
	}

	// If we found trailing spaces, return where they start
	if lastNonSpace < len(line)-1 {
		return lastNonSpace + 1
	}

	return -1
}

// renderWhitespaceChar renders a single character with whitespace visualization
// Returns the character to display and whether to use whitespace style
func (ws whitespaceStyle) renderChar(r rune, charPos int, line string, isTab bool) (rune, bool) {
	if isTab && ws.showTabs {
		return tabChar, true
	}

	if r == ' ' && ws.showTrailingSpaces {
		// Check if this space is part of trailing spaces
		trailingStart := detectTrailingSpaces(line)
		if trailingStart >= 0 && charPos >= trailingStart {
			return spaceChar, true
		}
	}

	return r, false
}
