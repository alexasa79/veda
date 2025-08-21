package editor

import (
	"strings"
	"testing"
)

func TestVisualModeMarkdownRendering(t *testing.T) {
	t.Run("visual selection in markdown file", func(t *testing.T) {
		// Create editor with markdown content
		te := NewTestEditorWithContent(80, 24, "# Header\n\nThis is a [link](https://example.com) in markdown.\n\nAnother paragraph.")

		// Set the file path to .md to trigger markdown highlighting
		te.buffer.FilePath = "test.md"

		// Initialize markdown highlighter
		if te.highlighter != nil {
			te.highlighter.SetLanguageFromFile(te.buffer.FilePath)
			te.highlighter.ParseFull([]byte(te.buffer.Rope.String()))
		}

		// Get theme colors
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Move to the word "link" on line 3
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		for i := 0; i < 10; i++ {
			te.ExecuteCommand("right")
		}

		// Enter visual mode
		te.ExecuteCommand("enter-visual-mode")

		// Select the word "link"
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}

		// Force render
		te.drawEditor()

		// Get the third line (index 2)
		line := te.mockScreen.GetLine(2)
		t.Logf("Line 2 content: %q", line)

		// Find where "link" appears
		linkStart := strings.Index(line, "link")
		if linkStart == -1 {
			t.Fatalf("Could not find 'link' in rendered line")
		}

		// Check that all 4 characters of "link" have selection background
		for i := 0; i < 4; i++ {
			_, bg, _ := te.mockScreen.GetCellStyle(linkStart+i, 2)
			if bg != selBg {
				t.Errorf("Character %d of 'link' does not have selection bg. Got %v, want %v", i, bg, selBg)
			}
		}
	})

	t.Run("visual selection across markdown link", func(t *testing.T) {
		// Create editor with markdown link
		te := NewTestEditorWithContent(80, 24, "Check out [this link](https://example.com) for more info.")
		te.buffer.FilePath = "test.md"

		// Initialize markdown highlighter
		if te.highlighter != nil {
			te.highlighter.SetLanguageFromFile(te.buffer.FilePath)
			te.highlighter.ParseFull([]byte(te.buffer.Rope.String()))
		}

		// Get selection style
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Move to start of "this"
		for i := 0; i < 11; i++ {
			te.ExecuteCommand("right")
		}

		// Enter visual mode
		te.ExecuteCommand("enter-visual-mode")

		// Select "this link" (9 characters)
		for i := 0; i < 9; i++ {
			te.ExecuteCommand("right")
		}

		// Force render
		te.drawEditor()

		// Get the line
		line := te.mockScreen.GetLine(0)
		t.Logf("Line content: %q", line)

		// In normal mode, the link should be rendered as just "this link"
		// Find where "this" appears
		thisStart := strings.Index(line, "this")
		if thisStart == -1 {
			t.Fatalf("Could not find 'this' in rendered line")
		}

		// Check that "this link" has selection background
		for i := 0; i < 9; i++ {
			_, bg, _ := te.mockScreen.GetCellStyle(thisStart+i, 0)
			if bg != selBg {
				t.Errorf("Character %d of 'this link' does not have selection bg. Got %v, want %v", i, bg, selBg)
			}
		}
	})

	t.Run("visual mode with markdown in insert vs normal mode", func(t *testing.T) {
		// Test that markdown rendering differs between modes
		te := NewTestEditorWithContent(80, 24, "This is a [test link](https://test.com) here.")
		te.buffer.FilePath = "test.md"

		// Initialize markdown highlighter
		if te.highlighter != nil {
			te.highlighter.SetLanguageFromFile(te.buffer.FilePath)
			te.highlighter.ParseFull([]byte(te.buffer.Rope.String()))
		}

		// First check normal mode rendering
		te.drawEditor()
		normalModeLine := te.mockScreen.GetLine(0)
		t.Logf("Normal mode: %q", normalModeLine)

		// The link should show full syntax in all modes now (we no longer hide URLs)
		if !strings.Contains(normalModeLine, "](https://") {
			t.Errorf("Normal mode should show full markdown link syntax")
		}

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		te.drawEditor()
		insertModeLine := te.mockScreen.GetLine(0)
		t.Logf("Insert mode: %q", insertModeLine)

		// The link should show full syntax in insert mode
		if !strings.Contains(insertModeLine, "[test link](https://test.com)") {
			t.Errorf("Insert mode should show full markdown syntax")
		}

		// Go back to normal mode and test visual selection
		te.ExecuteCommand("exit-insert-mode")

		// Get selection style
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Move to "test"
		for i := 0; i < 10; i++ {
			te.ExecuteCommand("right")
		}

		// Enter visual mode and select "test"
		te.ExecuteCommand("enter-visual-mode")
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}

		// Render and check
		te.drawEditor()
		line := te.mockScreen.GetLine(0)

		// Find "test" in the rendered line
		testStart := strings.Index(line, "test")
		if testStart != -1 {
			// Check that "test" has selection background
			for i := 0; i < 4; i++ {
				_, bg, _ := te.mockScreen.GetCellStyle(testStart+i, 0)
				if bg != selBg {
					t.Errorf("Character %d of 'test' in markdown link does not have selection bg. Got %v, want %v", i, bg, selBg)
				}
			}
		}
	})

	t.Run("visual selection with markdown headers", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "# Main Header\n## Sub Header\nNormal text")
		te.buffer.FilePath = "test.md"

		// Initialize markdown highlighter
		if te.highlighter != nil {
			te.highlighter.SetLanguageFromFile(te.buffer.FilePath)
			te.highlighter.ParseFull([]byte(te.buffer.Rope.String()))
		}

		// Get selection style
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Move to "Main"
		for i := 0; i < 2; i++ {
			te.ExecuteCommand("right")
		}

		// Enter visual mode and select "Main"
		te.ExecuteCommand("enter-visual-mode")
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}

		// Render
		te.drawEditor()
		line := te.mockScreen.GetLine(0)
		t.Logf("Header line: %q", line)

		// Find "Main" in the line
		mainStart := strings.Index(line, "Main")
		if mainStart != -1 {
			for i := 0; i < 4; i++ {
				_, bg, _ := te.mockScreen.GetCellStyle(mainStart+i, 0)
				if bg != selBg {
					t.Errorf("Character %d of 'Main' in header does not have selection bg. Got %v, want %v", i, bg, selBg)
				}
			}
		}
	})
}
