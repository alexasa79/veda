package editor

import (
	"testing"
	"veda/internal/events"
)

func TestAlignCursors(t *testing.T) {
	t.Run("align multiple cursors to rightmost position", func(t *testing.T) {
		// Create editor with test content
		content := "line1\nline2\nline3\nline4"
		te := NewTestEditorWithContent(80, 24, content)

		// Enter cursor mode
		te.ExecuteCommand("enter-cursor-mode")

		// Add virtual cursor at first line, position 2
		te.SetCursorPosition(2, 0)
		te.PressKey(events.KeyEnter, 0)

		// Move to second line, position 4
		te.SetCursorPosition(4, 1)
		te.PressKey(events.KeyEnter, 0)

		// Move to third line, position 1
		te.SetCursorPosition(1, 2)
		te.PressKey(events.KeyEnter, 0)

		// Move to fourth line, position 3 (primary cursor)
		te.SetCursorPosition(3, 3)

		// Exit cursor mode
		te.PressKey(events.KeyEscape, 0)

		// Execute align-cursors command
		te.ExecuteCommand("align-cursors")

		// Expected content after alignment (spaces added to align to position 4)
		// Line 0: cursor at 2, add 2 spaces -> "li  ne1"
		// Line 1: cursor at 4 (already rightmost) -> "line2"
		// Line 2: cursor at 1, add 3 spaces -> "l   ine3"
		// Line 3: cursor at 3, add 1 space -> "lin e4"
		expected := "li  ne1\nline2\nl   ine3\nlin e4"
		AssertBufferEquals(t, te, expected)

		// Check that all cursors are at position 4
		AssertCursorAt(t, te, 4, 3) // Primary cursor

		// Check virtual cursors
		virtualCursors := te.Editor.GetVirtualCursors()
		if len(virtualCursors) != 3 {
			t.Fatalf("Expected 3 virtual cursors, got %d", len(virtualCursors))
		}

		for i, cursor := range virtualCursors {
			if cursor.X != 4 {
				t.Errorf("Virtual cursor %d at X=%d, expected X=4", i, cursor.X)
			}
		}
	})

	t.Run("align cursors with rightmost cursor being primary", func(t *testing.T) {
		// Create editor with test content
		content := "abc\ndef\nghi"
		te := NewTestEditorWithContent(80, 24, content)

		// Enter cursor mode
		te.ExecuteCommand("enter-cursor-mode")

		// Add virtual cursor at first line, position 1
		te.SetCursorPosition(1, 0)
		te.PressKey(events.KeyEnter, 0)

		// Add virtual cursor at second line, position 2
		te.SetCursorPosition(2, 1)
		te.PressKey(events.KeyEnter, 0)

		// Move to third line, position 3 (primary cursor - rightmost)
		te.SetCursorPosition(3, 2)

		// Exit cursor mode
		te.PressKey(events.KeyEscape, 0)

		// Execute align-cursors command
		te.ExecuteCommand("align-cursors")

		// Expected content after alignment
		expected := "a  bc\nde f\nghi"
		AssertBufferEquals(t, te, expected)

		// Check cursor positions
		AssertCursorAt(t, te, 3, 2) // Primary cursor stays at 3

		virtualCursors := te.Editor.GetVirtualCursors()
		if virtualCursors[0].X != 3 {
			t.Errorf("Virtual cursor 0 at X=%d, expected X=3", virtualCursors[0].X)
		}
		if virtualCursors[1].X != 3 {
			t.Errorf("Virtual cursor 1 at X=%d, expected X=3", virtualCursors[1].X)
		}
	})

	t.Run("align cursors error when no multiple cursors", func(t *testing.T) {
		// Create editor with test content
		content := "test line"
		te := NewTestEditorWithContent(80, 24, content)

		// Execute align-cursors command without multiple cursors
		te.ExecuteCommand("align-cursors")

		// Should show error (check that content is unchanged)
		AssertBufferEquals(t, te, content)
	})

	t.Run("align cursors with undo/redo", func(t *testing.T) {
		// Create editor with test content
		content := "a\nb\nc"
		te := NewTestEditorWithContent(80, 24, content)

		// Enter cursor mode
		te.ExecuteCommand("enter-cursor-mode")

		// Add virtual cursors
		te.SetCursorPosition(0, 0)
		te.PressKey(events.KeyEnter, 0)
		te.SetCursorPosition(0, 1)
		te.PressKey(events.KeyEnter, 0)
		te.SetCursorPosition(1, 2) // Rightmost

		// Exit cursor mode
		te.PressKey(events.KeyEscape, 0)

		// Execute align-cursors command
		te.ExecuteCommand("align-cursors")

		// Check aligned content
		expected := " a\n b\nc"
		AssertBufferEquals(t, te, expected)

		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, content)

		// Redo
		te.ExecuteCommand("redo")
		AssertBufferEquals(t, te, expected)
	})

	t.Run("align cursors with different positions", func(t *testing.T) {
		// Create editor with test content
		content := "hello\nworld\ntest"
		te := NewTestEditorWithContent(80, 24, content)

		// Enter cursor mode
		te.ExecuteCommand("enter-cursor-mode")

		// Add virtual cursor at first line, position 2
		te.SetCursorPosition(2, 0)
		te.PressKey(events.KeyEnter, 0)

		// Add virtual cursor at second line, position 5 (rightmost)
		te.SetCursorPosition(5, 1)
		te.PressKey(events.KeyEnter, 0)

		// Primary cursor at third line, position 0
		te.SetCursorPosition(0, 2)

		// Exit cursor mode
		te.PressKey(events.KeyEscape, 0)

		// Execute align-cursors command
		te.ExecuteCommand("align-cursors")

		// Expected content after alignment
		// Line 0: cursor at 2, add 3 spaces to reach 5 -> "he   llo"
		// Line 1: cursor at 5 (already rightmost) -> "world"
		// Line 2: cursor at 0, add 5 spaces -> "     test"
		expected := "he   llo\nworld\n     test"
		AssertBufferEquals(t, te, expected)
	})
}
