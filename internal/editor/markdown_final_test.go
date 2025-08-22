package editor

import (
    "strings"
    "testing"
    "veda/internal/syntax"
)

func TestMarkdownRenderingWithSelection(t *testing.T) {
	t.Run("markdown link renders correctly and selection works", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Check [this link](https://example.com) out")
		te.buffer.FilePath = "test.md"

        // Initialize highlighter for markdown (per-buffer)
        te.buffer.Highlighter = syntax.NewHighlighter()
        if te.buffer.Highlighter != nil {
            te.buffer.Highlighter.SetLanguageFromFile(te.buffer.FilePath)
            te.buffer.Highlighter.ParseFull([]byte(te.buffer.Rope.String()))
            t.Logf("Highlighter language after setup: %s", te.buffer.Highlighter.GetCurrentLanguage())
        }

		// Get selection style
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// First render to check markdown processing
		t.Logf("Editor mode: %v", te.mode)
		te.drawEditor()
		line := te.mockScreen.GetLine(0)
		t.Logf("Initial render: %q", line)

		// Should show full markdown syntax (we no longer hide URLs)
		if !strings.Contains(line, "[this link](https://example.com)") {
			t.Errorf("Markdown link should be shown in full in normal mode")
		}

		// Now test selection
		// Move to "this" (after "Check ")
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}

		// Enter visual mode
		te.ExecuteCommand("enter-visual-mode")

		// Select "this link"
		for i := 0; i < 9; i++ {
			te.ExecuteCommand("right")
		}

		// Render with selection
		te.drawEditor()
		line = te.mockScreen.GetLine(0)
		t.Logf("With selection: %q", line)

		// Find where "this" starts
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

	t.Run("insert mode shows full markdown syntax", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Text [link](url) here")
		te.buffer.FilePath = "test.md"

        // Initialize highlighter (per-buffer)
        te.buffer.Highlighter = syntax.NewHighlighter()
        if te.buffer.Highlighter != nil {
            te.buffer.Highlighter.SetLanguageFromFile(te.buffer.FilePath)
            te.buffer.Highlighter.ParseFull([]byte(te.buffer.Rope.String()))
        }

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Render
		te.drawEditor()
		line := te.mockScreen.GetLine(0)
		t.Logf("Insert mode: %q", line)

		// Should show full syntax in insert mode
		if !strings.Contains(line, "[link](url)") {
			t.Errorf("Insert mode should show full markdown syntax")
		}
	})
}
