package editor

import (
	"testing"
	"veda/internal/clipboard"
	"veda/internal/syntax"
)

func TestPasteAfterCursorPosition(t *testing.T) {
	t.Run("paste after equals sign", func(t *testing.T) {
		// Create editor with initial content
		te := NewTestEditorWithContent(80, 24, "A=12345\nB=")
		// Ensure no language to avoid auto-indent interfering
		te.CurrentBuffer().Highlighter = syntax.NewHighlighter()

		// Simply copy "12345" to clipboard directly for this test
		clipboard.Copy("12345")

		// Move down to second line
		te.ExecuteCommand("down")

		// Move cursor to '=' (position 1)
		te.SetCursorPosition(1, 1)

		// Paste after cursor (p command)
		te.ExecuteCommand("paste-after")

		// Check the result - should be "B=12345"
		expected := "A=12345\nB=12345"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected:\n%s\nBut got:\n%s", expected, actual)
		}

		// Cursor should be at the last pasted character
		x, y := te.GetCursorPosition()
		if x != 6 || y != 1 {
			t.Errorf("After paste, expected cursor at (6, 1), but got (%d, %d)", x, y)
		}
	})

	t.Run("paste after cursor in middle of line", func(t *testing.T) {
		// Create editor with initial content
		te := NewTestEditorWithContent(80, 24, "ABC\nDEF")
		te.CurrentBuffer().Highlighter = syntax.NewHighlighter()

		// Copy "BC" from first line
		clipboard.Copy("BC")

		// Move to 'E' in second line
		te.SetCursorPosition(1, 1)

		// Paste after cursor
		te.ExecuteCommand("paste-after")

		// Check the result - should be "DEBCF" (paste after 'E')
		expected := "ABC\nDEBCF"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected:\n%s\nBut got:\n%s", expected, actual)
		}
	})

	t.Run("paste after cursor at end of line", func(t *testing.T) {
		// Create editor with initial content
		te := NewTestEditorWithContent(80, 24, "ABC\nDEF")
		te.CurrentBuffer().Highlighter = syntax.NewHighlighter()

		// Copy "123" to clipboard
		clipboard.Copy("123")

		// Move to end of second line (on 'F')
		te.SetCursorPosition(2, 1)

		// Paste after cursor
		te.ExecuteCommand("paste-after")

		// Check the result - should be "DEF123"
		expected := "ABC\nDEF123"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected:\n%s\nBut got:\n%s", expected, actual)
		}
	})
}

func TestPasteBeforeCursorPosition(t *testing.T) {
	t.Run("paste before equals sign", func(t *testing.T) {
		// Create editor with initial content
		te := NewTestEditorWithContent(80, 24, "A=12345\nB=")
		te.CurrentBuffer().Highlighter = syntax.NewHighlighter()

		// Copy "12345" to clipboard
		clipboard.Copy("12345")

		// Move to '=' on second line (position 1)
		te.SetCursorPosition(1, 1)

		// Paste before cursor (P command)
		te.ExecuteCommand("paste-before")

		// Check the result - should be "B12345="
		expected := "A=12345\nB12345="
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected:\n%s\nBut got:\n%s", expected, actual)
		}
	})

	t.Run("paste before cursor in middle of line", func(t *testing.T) {
		// Create editor with initial content
		te := NewTestEditorWithContent(80, 24, "ABC\nDEF")
		te.CurrentBuffer().Highlighter = syntax.NewHighlighter()

		// Copy "123"
		clipboard.Copy("123")

		// Move to 'E' in second line
		te.SetCursorPosition(1, 1)

		// Paste before cursor
		te.ExecuteCommand("paste-before")

		// Check the result - should be "D123EF"
		expected := "ABC\nD123EF"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected:\n%s\nBut got:\n%s", expected, actual)
		}
	})
}
