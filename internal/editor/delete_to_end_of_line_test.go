package editor

import (
	"testing"
	"veda/internal/events"
)

func TestDeleteToEndOfLine(t *testing.T) {
	t.Run("delete from middle of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello, World!")

		// Position cursor at position 7 (after comma and space)
		te.buffer.CursorX = 7
		AssertCursorAt(t, te, 7, 0)

		// Execute D command
		te.ExecuteCommand("delete-to-end-of-line")

		// Should have deleted "World!"
		AssertBufferEquals(t, te, "Hello, ")

		// Cursor should be at position 6 (last character in normal mode)
		AssertCursorAt(t, te, 6, 0)
	})

	t.Run("delete from beginning of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello, World!")

		// Cursor at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute D command
		te.ExecuteCommand("delete-to-end-of-line")

		// Should have deleted entire line content
		AssertBufferEquals(t, te, "")

		// Cursor should be at position 0
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("delete on empty line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")

		// Execute D command on empty buffer
		te.ExecuteCommand("delete-to-end-of-line")

		// Buffer should remain empty
		AssertBufferEquals(t, te, "")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("delete at end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")

		// Position cursor at end (position 4, last character)
		te.buffer.CursorX = 4
		AssertCursorAt(t, te, 4, 0)

		// Execute D command
		te.ExecuteCommand("delete-to-end-of-line")

		// Should delete just the last character
		AssertBufferEquals(t, te, "Hell")

		// Cursor should be at position 3 (last character)
		AssertCursorAt(t, te, 3, 0)
	})

	t.Run("delete preserves newlines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Position cursor at "1" in first line
		te.buffer.CursorX = 5

		// Execute D command
		te.ExecuteCommand("delete-to-end-of-line")

		// Should only delete "1" from first line, preserving newline
		AssertBufferEquals(t, te, "Line \nLine 2\nLine 3")

		// Cursor should be at position 4 (space after "Line")
		AssertCursorAt(t, te, 4, 0)
	})

	t.Run("undo delete to end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello, World!")

		// Position cursor at position 7
		te.buffer.CursorX = 7

		// Execute D command
		te.ExecuteCommand("delete-to-end-of-line")
		AssertBufferEquals(t, te, "Hello, ")

		// Undo
		te.ExecuteCommand("undo")

		// Should restore original text
		AssertBufferEquals(t, te, "Hello, World!")
		AssertCursorAt(t, te, 7, 0)
	})

	t.Run("delete with tabs", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "\tHello\tWorld")

		// Position after first tab
		te.buffer.CursorX = 1

		// Execute D command
		te.ExecuteCommand("delete-to-end-of-line")

		// Should delete from position 1 to end
		AssertBufferEquals(t, te, "\t")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("repeat delete to end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1 text\nLine 2 text\nLine 3 text")

		// Position at "1" in first line
		te.buffer.CursorX = 5

		// Execute D command
		te.ExecuteCommand("delete-to-end-of-line")
		AssertBufferEquals(t, te, "Line \nLine 2 text\nLine 3 text")

		// Move to next line
		te.ExecuteCommand("down")
		te.buffer.CursorX = 5 // Position at "2"

		// Repeat last command with '.'
		te.ExecuteCommand("repeat-last-change")

		// Should have deleted "2 text" from second line
		AssertBufferEquals(t, te, "Line \nLine \nLine 3 text")
	})

	t.Run("multiple cursors delete to end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "First line text\nSecond line text\nThird line text")

		// Enter cursor mode
		te.ExecuteCommand("enter-cursor-mode")

		// Add virtual cursor at beginning of first line
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (0,0), moves to line 1

		// Add virtual cursor at beginning of second line
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (0,1), moves to line 2

		// Exit cursor mode (press Escape)
		te.PressKey(events.KeyEscape, 0)

		// Now we have cursors at beginning of lines 0, 1, and primary cursor at line 2
		// Execute D command - should delete from beginning to end on all three lines
		te.ExecuteCommand("delete-to-end-of-line")

		// All three lines should be empty
		AssertBufferEquals(t, te, "\n\n")

		// Virtual cursors should still exist after the operation (they persist until cleared)
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors to remain after operation, but found %d", len(virtualCursors))
		}

		// Clear virtual cursors by pressing Escape
		te.PressKey(events.KeyEscape, 0)

		// Now virtual cursors should be cleared
		virtualCursors = te.GetVirtualCursors()
		if len(virtualCursors) != 0 {
			t.Errorf("Expected virtual cursors to be cleared after Escape, but found %d", len(virtualCursors))
		}
	})
}
