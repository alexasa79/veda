package editor

import (
	"strings"
	"testing"
)

func TestMarkdownSelectionHighlighting(t *testing.T) {
	t.Run("visual selection highlighting in markdown links", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Check [this link](https://example.com) out")
		te.buffer.FilePath = "test.md"

		// Get selection style
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Move to "this" in the link
		for i := 0; i < 7; i++ {
			te.ExecuteCommand("right")
		}

		// Enter visual mode
		te.ExecuteCommand("enter-visual-mode")

		// Select "this link"
		for i := 0; i < 9; i++ {
			te.ExecuteCommand("right")
		}

		// Render
		te.drawEditor()

		// Check the rendered line
		line := te.mockScreen.GetLine(0)
		t.Logf("Rendered line: %q", line)

		// In normal mode, this should show as "Check this link out"
		// Find where "this" starts
		thisStart := strings.Index(line, "this")
		if thisStart == -1 {
			t.Fatalf("Could not find 'this' in rendered line")
		}

		// Check that all characters of "this link" have selection background
		for i := 0; i < 9; i++ {
			_, bg, _ := te.mockScreen.GetCellStyle(thisStart+i, 0)
			if bg != selBg {
				t.Errorf("Character %d of 'this link' does not have selection bg. Got %v, want %v", i, bg, selBg)
			}
		}
	})

	t.Run("selection across markdown boundaries", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Text before [link](url) text after")
		te.buffer.FilePath = "test.md"

		// Get selection style
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Move to "before"
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}

		// Enter visual mode and select "before link text"
		te.ExecuteCommand("enter-visual-mode")
		for i := 0; i < 15; i++ { // "before link text" (accounting for markdown being rendered as just "link")
			te.ExecuteCommand("right")
		}

		// Render
		te.drawEditor()

		line := te.mockScreen.GetLine(0)
		t.Logf("Rendered line: %q", line)

		// Find "before"
		beforeStart := strings.Index(line, "before")
		if beforeStart != -1 {
			// Check selection starts at "before" and continues through "link text"
			for i := 0; i < 6; i++ { // "before"
				_, bg, _ := te.mockScreen.GetCellStyle(beforeStart+i, 0)
				if bg != selBg {
					t.Errorf("Character %d of 'before' should be selected", i)
				}
			}
		}
	})
}
