package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestLineNavigation(t *testing.T) {
	t.Run("move to beginning of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello, World!")

		// Start in middle of line
		te.CurrentBuffer().CursorX = 7
		AssertCursorAt(t, te, 7, 0)

		// Move to beginning
		te.ExecuteCommand("beginning-of-line")
		AssertCursorAt(t, te, 0, 0)

		// Already at beginning, should stay there
		te.ExecuteCommand("beginning-of-line")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("move to end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello, World!")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Move to end (position 12, last character)
		te.ExecuteCommand("end-of-line")
		AssertCursorAt(t, te, 12, 0)

		// Already at end, should stay there
		te.ExecuteCommand("end-of-line")
		AssertCursorAt(t, te, 12, 0)
	})

	t.Run("line navigation with multiple lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "First line\nSecond line\nThird")

		// Start on second line, middle
		te.SetCursorPosition(6, 1)

		// Move to beginning of current line
		te.ExecuteCommand("beginning-of-line")
		AssertCursorAt(t, te, 0, 1)

		// Move to end of current line
		te.ExecuteCommand("end-of-line")
		AssertCursorAt(t, te, 10, 1) // "Second line" has 11 chars, cursor at position 10

		// Move to third line
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 4, 2) // Cursor adjusts to shorter line

		// Move to beginning
		te.ExecuteCommand("beginning-of-line")
		AssertCursorAt(t, te, 0, 2)

		// Move to end
		te.ExecuteCommand("end-of-line")
		AssertCursorAt(t, te, 4, 2) // "Third" has 5 chars
	})

	t.Run("empty line handling", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line1\n\nLine3")

		// Move to empty line
		te.SetCursorPosition(0, 1)

		// Beginning and end should both be at position 0
		te.ExecuteCommand("beginning-of-line")
		AssertCursorAt(t, te, 0, 1)

		te.ExecuteCommand("end-of-line")
		AssertCursorAt(t, te, 0, 1)
	})

	t.Run("line navigation in insert mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello, World!")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Start in middle
		te.CurrentBuffer().CursorX = 7

		// Move to beginning in insert mode
		te.ExecuteCommand("beginning-of-line-insert")
		AssertCursorAt(t, te, 0, 0)

		// Move to end in insert mode (past last character)
		te.ExecuteCommand("end-of-line-insert")
		AssertCursorAt(t, te, 13, 0) // One position past the last character

		// Should still be in insert mode
		if te.mode != commands.InsertMode {
			t.Error("Should remain in insert mode after line navigation")
		}
	})
}
