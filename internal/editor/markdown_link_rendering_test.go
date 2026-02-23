package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
	"veda/internal/syntax"
	"veda/internal/ui"
)

// TestMarkdownLinkRenderingModes tests that markdown links render differently in normal vs insert mode
// TODO: Fix markdown link hiding to work with line wrapping. Currently only works without wrapping.
func TestMarkdownLinkRenderingModes(t *testing.T) {
	t.Skip("Markdown link hiding with line wrapping not yet implemented")
	// Note: Markdown link hiding currently only works without line wrapping
	// These tests use wider screens to avoid wrapping
	tests := []struct {
		name               string
		content            string
		expectedNormalMode string // What should be visible on screen in normal mode
		expectedInsertMode string // What should be visible on screen in insert mode
		width              int
	}{
		{
			name:               "simple_link",
			content:            "This is a [link](https://example.com) in text.",
			expectedNormalMode: "1 This is a link in text.",
			expectedInsertMode: "1 This is a [link](https://example.com) in text.",
			width:              80,
		},
		{
			name:               "multiple_links",
			content:            "Check [Google](https://google.com) and [GitHub](https://github.com) for more.",
			expectedNormalMode: "1 Check Google and GitHub for more.",
			expectedInsertMode: "1 Check [Google](https://google.com) and [GitHub](https://github.com) for more.",
			width:              100,
		},
		{
			name:               "link_at_line_start",
			content:            "[Home](https://home.com) is where the heart is.",
			expectedNormalMode: "1 Home is where the heart is.",
			expectedInsertMode: "1 [Home](https://home.com) is where the heart is.",
			width:              80,
		},
		{
			name:               "link_at_line_end",
			content:            "Visit our [website](https://site.com)",
			expectedNormalMode: "1 Visit our website",
			expectedInsertMode: "1 Visit our [website](https://site.com)",
			width:              80,
		},
		{
			name:               "empty_link_text",
			content:            "Empty [](https://example.com) link",
			expectedNormalMode: "1 Empty  link",
			expectedInsertMode: "1 Empty [](https://example.com) link",
			width:              80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test in Normal Mode
			t.Run("normal_mode", func(t *testing.T) {
				te := NewTestEditorWithContent(tt.width, 10, tt.content)
				te.CurrentBuffer().FilePath = "test.md"

				// Set up markdown syntax highlighting (per-buffer)
				te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
				err := te.CurrentBuffer().Highlighter.SetLanguage("markdown")
				if err != nil {
					t.Fatalf("Failed to set markdown language: %v", err)
				}
				te.CurrentBuffer().Highlighter.ParseFull([]byte(tt.content))

				// Ensure we're in normal mode
				te.mode = commands.NormalMode

				// Render the editor
				te.renderPane()
				screen := te.mockScreen.GetContent()

				// Check rendered output
				actualLine := GetScreenLine(screen, 0)
				if actualLine != tt.expectedNormalMode {
					t.Errorf("Normal mode: expected %q, got %q", tt.expectedNormalMode, actualLine)
				}
			})

			// Test in Insert Mode
			t.Run("insert_mode", func(t *testing.T) {
				te := NewTestEditorWithContent(tt.width, 10, tt.content)
				te.CurrentBuffer().FilePath = "test.md"

				// Set up markdown syntax highlighting (per-buffer)
				te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
				err := te.CurrentBuffer().Highlighter.SetLanguage("markdown")
				if err != nil {
					t.Fatalf("Failed to set markdown language: %v", err)
				}
				te.CurrentBuffer().Highlighter.ParseFull([]byte(tt.content))

				// Switch to insert mode
				te.ExecuteCommand("enter-insert-mode")

				// Verify we're in insert mode
				if te.mode != commands.InsertMode {
					t.Fatalf("Not in insert mode, mode is: %v", te.mode)
				}

				// Render the editor
				te.renderPane()
				screen := te.mockScreen.GetContent()

				// Check rendered output
				actualLine := GetScreenLine(screen, 0)
				if actualLine != tt.expectedInsertMode {
					t.Errorf("Insert mode: expected %q, got %q", tt.expectedInsertMode, actualLine)
				}
			})
		})
	}
}

// TestMarkdownLinkHighlighting tests that syntax highlighting works correctly with mode-aware rendering
func TestMarkdownLinkHighlighting(t *testing.T) {
	t.Skip("Markdown link hiding not working properly in tests")
	content := "Visit [our site](https://example.com) for more info."

	te := NewTestEditorWithContent(80, 10, content)
	te.CurrentBuffer().FilePath = "test.md"

	// Set up markdown syntax highlighting (per-buffer)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	err := te.CurrentBuffer().Highlighter.SetLanguage("markdown")
	if err != nil {
		t.Fatalf("Failed to set markdown language: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(content))

	// Test that link text is highlighted in normal mode
	te.mode = commands.NormalMode
	te.renderPane()
	screen := te.mockScreen.GetContent()

	// Find the link text "our site" and check it has link styling
	line0 := screen[0]
	if line0 == nil {
		t.Fatal("No content on line 0")
	}

	// Look for "our site" starting after "Visit "
	visitEnd := 8 // "1 Visit " = 8 chars

	// Check if the link text has the expected style (not default)
	hasLinkStyle := false
	for x := visitEnd; x < visitEnd+8; x++ { // "our site" = 8 chars
		if cell, ok := line0[x]; ok && cell.Style != ui.StyleDefault {
			hasLinkStyle = true
			break
		}
	}

	if !hasLinkStyle {
		t.Error("Link text should have syntax highlighting in normal mode")
	}
}

// TestMarkdownLinkCursorPositioning tests cursor positioning with mode-aware rendering
func TestMarkdownLinkCursorPositioning(t *testing.T) {
	t.Skip("Markdown link hiding not working properly in tests")
	content := "Click [here](https://example.com) to continue."

	te := NewTestEditorWithContent(80, 10, content)
	te.CurrentBuffer().FilePath = "test.md"

	// Set up markdown syntax highlighting (per-buffer)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	err := te.CurrentBuffer().Highlighter.SetLanguage("markdown")
	if err != nil {
		t.Fatalf("Failed to set markdown language: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(content))

	// In normal mode, move cursor to where the link URL would be
	te.mode = commands.NormalMode

	// Move cursor to position 12 (middle of URL in buffer)
	// "Click [here]" = 12 chars, so cursor is at the start of URL
	te.CurrentBuffer().CursorX = 12

	// Render and check cursor position
	te.renderPane()
	screenX, _ := te.mockScreen.GetCursorPosition()

	// In normal mode, the URL is hidden, so cursor should be adjusted
	// "Click here" = 10 chars, + line number "1 " = 2 chars = 12
	// But cursor at position 12 is past "here", so it should be at end of "here"
	expectedX := 12 // "1 Click here"
	if screenX != expectedX {
		t.Errorf("Normal mode cursor X: expected %d, got %d", expectedX, screenX)
	}

	// Switch to insert mode
	te.ExecuteCommand("insert")

	// Render again and check cursor position
	te.renderPane()
	screenX, _ = te.mockScreen.GetCursorPosition()

	// In insert mode, full text is shown, so cursor should be at original position
	// "1 Click [here]" = 14 chars (line number + content up to cursor)
	expectedX = 14
	if screenX != expectedX {
		t.Errorf("Insert mode cursor X: expected %d, got %d", expectedX, screenX)
	}
}

// TestMarkdownLinkEditing tests editing behavior with mode-aware rendering
func TestMarkdownLinkEditing(t *testing.T) {
	t.Skip("Markdown link hiding not working properly in tests")
	content := "Check [this](https://example.com) out."

	te := NewTestEditorWithContent(80, 10, content)
	te.CurrentBuffer().FilePath = "test.md"

	// Set up markdown syntax highlighting (per-buffer)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	err := te.CurrentBuffer().Highlighter.SetLanguage("markdown")
	if err != nil {
		t.Fatalf("Failed to set markdown language: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(content))

	// Move cursor to the link text in normal mode
	te.CurrentBuffer().CursorX = 7 // Position at 't' in "this"

	// Enter insert mode and add text
	te.ExecuteCommand("insert")
	te.InsertChar('!')

	// Buffer should now contain "Check [t!his](https://example.com) out."
	expected := "Check [t!his](https://example.com) out."
	actual := te.CurrentBuffer().Rope.String()
	if actual != expected {
		t.Errorf("After insert: expected %q, got %q", expected, actual)
	}

	// Exit insert mode
	te.ExecuteCommand("normal_mode")

	// Render and check display
	te.renderPane()
	screen := te.mockScreen.GetContent()

	// In normal mode, should show "Check t!his out."
	expectedLine := "1 Check t!his out."
	actualLine := GetScreenLine(screen, 0)
	if actualLine != expectedLine {
		t.Errorf("Normal mode display: expected %q, got %q", expectedLine, actualLine)
	}
}

// Helper to get visible line content without trailing spaces
func GetVisibleLine(screen map[int]map[int]Cell, y int) string {
	row, ok := screen[y]
	if !ok {
		return ""
	}

	var line strings.Builder
	maxX := 0
	for x := range row {
		if x > maxX {
			maxX = x
		}
	}

	for x := 0; x <= maxX; x++ {
		if cell, ok := row[x]; ok {
			line.WriteRune(cell.Rune)
		} else {
			line.WriteRune(' ')
		}
	}

	return strings.TrimRight(line.String(), " ")
}
