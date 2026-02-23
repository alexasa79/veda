package editor

import (
	"testing"
	"veda/internal/ui"
)

func TestDetectTrailingSpaces(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "No trailing spaces",
			line:     "hello world",
			expected: -1,
		},
		{
			name:     "Single trailing space",
			line:     "hello world ",
			expected: 11,
		},
		{
			name:     "Multiple trailing spaces",
			line:     "hello world   ",
			expected: 11,
		},
		{
			name:     "Only spaces",
			line:     "   ",
			expected: 0,
		},
		{
			name:     "Empty line",
			line:     "",
			expected: -1,
		},
		{
			name:     "Spaces in middle",
			line:     "hello   world",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectTrailingSpaces(tt.line)
			if result != tt.expected {
				t.Errorf("detectTrailingSpaces(%q) = %d, want %d", tt.line, result, tt.expected)
			}
		})
	}
}

func TestWhitespaceStyleRenderChar(t *testing.T) {
	// Create a test whitespace style with a default style
	ws := whitespaceStyle{
		showTabs:           true,
		showTrailingSpaces: true,
		style:              ui.StyleDefault,
	}

	tests := []struct {
		name             string
		char             rune
		charPos          int
		line             string
		isTab            bool
		expectedChar     rune
		expectedUseStyle bool
	}{
		{
			name:             "Tab character",
			char:             '\t',
			charPos:          0,
			line:             "\thello",
			isTab:            true,
			expectedChar:     '→',
			expectedUseStyle: true,
		},
		{
			name:             "Regular space",
			char:             ' ',
			charPos:          5,
			line:             "hello world",
			isTab:            false,
			expectedChar:     ' ',
			expectedUseStyle: false,
		},
		{
			name:             "Trailing space",
			char:             ' ',
			charPos:          11,
			line:             "hello world ",
			isTab:            false,
			expectedChar:     '·',
			expectedUseStyle: true,
		},
		{
			name:             "Multiple trailing spaces",
			char:             ' ',
			charPos:          12,
			line:             "hello world   ",
			isTab:            false,
			expectedChar:     '·',
			expectedUseStyle: true,
		},
		{
			name:             "Regular character",
			char:             'a',
			charPos:          0,
			line:             "abc",
			isTab:            false,
			expectedChar:     'a',
			expectedUseStyle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChar, gotUseStyle := ws.renderChar(tt.char, tt.charPos, tt.line, tt.isTab)
			if gotChar != tt.expectedChar {
				t.Errorf("renderChar() char = %c, want %c", gotChar, tt.expectedChar)
			}
			if gotUseStyle != tt.expectedUseStyle {
				t.Errorf("renderChar() useStyle = %v, want %v", gotUseStyle, tt.expectedUseStyle)
			}
		})
	}
}

func TestWhitespaceRenderingIntegration(t *testing.T) {
	// Create test editor
	te := NewTestEditorWithContent(80, 24, "\thello world   ")

	// Draw the editor
	te.renderPane()

	// Get the rendered output
	mockScreen := te.screen.(*MockScreen)

	// Line numbers are shown by default; use buffer's configured width
	lineNumberWidth := te.CurrentBuffer().LineNumberWidth

	// Check that tab is rendered as arrow
	if cell, ok := mockScreen.cells[0][lineNumberWidth]; ok {
		if cell.Rune != '→' {
			t.Errorf("Expected tab to be rendered as '→', got %c", cell.Rune)
		}
	} else {
		t.Error("No character rendered at tab position")
	}

	// Check that trailing spaces are rendered as dots
	// Tab = 4 visual chars (arrow + 3 spaces), "hello world" = 11 chars
	// So trailing spaces start at lineNumberWidth + 4 + 11 = 17
	trailingStart := lineNumberWidth + 4 + 11
	for i := 0; i < 3; i++ {
		if cell, ok := mockScreen.cells[0][trailingStart+i]; ok {
			if cell.Rune != '·' {
				t.Errorf("Expected trailing space at position %d to be rendered as '·', got %c", trailingStart+i, cell.Rune)
			}
		} else {
			t.Errorf("No character rendered at position (%d,0)", trailingStart+i)
		}
	}
}
