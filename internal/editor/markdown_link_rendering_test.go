package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
	"veda/internal/syntax"
	"veda/internal/ui"
)

// TestMarkdownLinkRenderingModes tests that markdown links render differently in normal vs insert mode.
func TestMarkdownLinkRenderingModes(t *testing.T) {
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
			expectedNormalMode: "This is a link in text.",
			expectedInsertMode: "This is a [link](https://example.com) in text.",
			width:              80,
		},
		{
			name:               "multiple_links",
			content:            "Check [Google](https://google.com) and [GitHub](https://github.com) for more.",
			expectedNormalMode: "Check Google and GitHub for more.",
			expectedInsertMode: "Check [Google](https://google.com) and [GitHub](https://github.com) for more.",
			width:              100,
		},
		{
			name:               "link_at_line_start",
			content:            "[Home](https://home.com) is where the heart is.",
			expectedNormalMode: "Home is where the heart is.",
			expectedInsertMode: "[Home](https://home.com) is where the heart is.",
			width:              80,
		},
		{
			name:               "link_at_line_end",
			content:            "Visit our [website](https://site.com)",
			expectedNormalMode: "Visit our website",
			expectedInsertMode: "Visit our [website](https://site.com)",
			width:              80,
		},
		{
			name:               "empty_link_text",
			content:            "Empty [](https://example.com) link",
			expectedNormalMode: "Empty  link",
			expectedInsertMode: "Empty [](https://example.com) link",
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
				// Check rendered output
				actualLine := strings.TrimSpace(markdownPresentationScreenLine(te, 0))
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
				// Check rendered output
				actualLine := strings.TrimSpace(markdownPresentationScreenLine(te, 0))
				if actualLine != tt.expectedInsertMode {
					t.Errorf("Insert mode: expected %q, got %q", tt.expectedInsertMode, actualLine)
				}
			})
		})
	}
}

// TestMarkdownLinkHighlighting tests that syntax highlighting works correctly with mode-aware rendering
func TestMarkdownLinkHighlighting(t *testing.T) {
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
	visitEnd := te.CurrentBuffer().LineNumberWidth + len("Visit ")

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
	te.Editor.render()
	screenX, _ := te.mockScreen.GetCursorPosition()

	// In normal mode, the URL is hidden, so cursor should be adjusted
	// "Click here" = 10 chars, + line number "1 " = 2 chars = 12
	// But cursor at position 12 is past "here", so it should be at end of "here"
	expectedX := te.CurrentBuffer().LineNumberWidth + len("Click here")
	if screenX != expectedX {
		t.Errorf("Normal mode cursor X: expected %d, got %d", expectedX, screenX)
	}

	// Switch to insert mode
	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatalf("enter-insert-mode failed: %v", err)
	}

	// Render again and check cursor position
	te.Editor.render()
	screenX, _ = te.mockScreen.GetCursorPosition()

	// In insert mode, full text is shown, so cursor should be at original position
	// "1 Click [here]" = 14 chars (line number + content up to cursor)
	expectedX = te.CurrentBuffer().LineNumberWidth + 12
	if screenX != expectedX {
		t.Errorf("Insert mode cursor X: expected %d, got %d", expectedX, screenX)
	}
}

// TestMarkdownLinkEditing tests editing behavior with mode-aware rendering
func TestMarkdownLinkEditing(t *testing.T) {
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
	te.CurrentBuffer().CursorX = 8 // Position after 't' in "this"

	// Enter insert mode and add text
	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatalf("enter-insert-mode failed: %v", err)
	}
	te.InsertChar('!')

	// Buffer should now contain "Check [t!his](https://example.com) out."
	expected := "Check [t!his](https://example.com) out."
	actual := te.CurrentBuffer().Rope.String()
	if actual != expected {
		t.Errorf("After insert: expected %q, got %q", expected, actual)
	}

	// Exit insert mode
	if err := te.ExecuteCommand("exit-insert-mode"); err != nil {
		t.Fatalf("exit-insert-mode failed: %v", err)
	}

	// Render and check display
	te.renderPane()
	screen := te.mockScreen.GetContent()

	// In normal mode, should show "Check t!his out."
	expectedLine := "Check t!his out."
	actualLine := strings.TrimSpace(GetScreenLine(screen, 0)[te.CurrentBuffer().LineNumberWidth:])
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
