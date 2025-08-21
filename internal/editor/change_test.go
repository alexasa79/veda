package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestChangeOperations(t *testing.T) {
	t.Run("c left changes character to the left", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")

		// Move cursor to position 5 (space after "Hello")
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 5, 0)

		// Execute "c left"
		te.TypeKeys("c")
		te.TypeKeys("left")

		// Should be in insert mode after deleting 'o'
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'c left', got %v", te.mode)
		}

		// Type replacement text
		te.InsertChar('a')
		te.ExecuteCommand("exit-insert-mode")

		// Should have replaced 'o' with 'a'
		AssertBufferEquals(t, te, "Hella World")
		AssertCursorAt(t, te, 4, 0) // Cursor on 'a'
	})

	t.Run("C changes to end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello, World!")
		// Position cursor at position 7 (after comma and space)
		te.buffer.CursorX = 7
		AssertCursorAt(t, te, 7, 0)

		// Execute C
		te.TypeKeys("C")

		// Should be in insert mode after deleting to EOL
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'C', got %v", te.mode)
		}

		// Type replacement text and exit insert mode
		te.InsertChar('X')
		te.ExecuteCommand("exit-insert-mode")

		// Should have replaced tail of line with 'X'
		AssertBufferEquals(t, te, "Hello, X")
	})

	t.Run("C at end of line enters insert mode with no deletion", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		// Place at end (on last visible char index 4 in normal mode)
		te.buffer.CursorX = 4
		AssertCursorAt(t, te, 4, 0)

		// Execute C
		te.TypeKeys("C")
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'C' at EOL, got %v", te.mode)
		}
		// Add text and finish
		te.InsertChar('!')
		te.ExecuteCommand("exit-insert-mode")
		AssertBufferEquals(t, te, "Hell!")
	})

	t.Run("c right changes character to the right", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")

		// Cursor starts at position 0
		AssertCursorAt(t, te, 0, 0)

		// Execute "c right"
		te.TypeKeys("c")
		te.TypeKeys("right")

		// Should be in insert mode after deleting 'H'
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'c right', got %v", te.mode)
		}

		// Type replacement text
		te.InsertChar('J')
		te.ExecuteCommand("exit-insert-mode")

		// Should have replaced 'H' with 'J'
		AssertBufferEquals(t, te, "Jello World")
		AssertCursorAt(t, te, 0, 0) // Cursor on 'J'
	})

	t.Run("c left at beginning of line does nothing", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")

		// Cursor at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "c left"
		te.TypeKeys("c")
		te.TypeKeys("left")

		// Should still be in normal mode (no change happened)
		if te.mode != commands.NormalMode {
			t.Errorf("Expected normal mode when 'c left' fails, got %v", te.mode)
		}

		// Buffer unchanged
		AssertBufferEquals(t, te, "Hello World")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("c right at end of line does nothing", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")

		// Move to last character (position 4 for "Hello")
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 4, 0) // On 'o'

		// Execute "c right"
		te.TypeKeys("c")
		te.TypeKeys("right")

		// This should work and change the 'o'
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'c right', got %v", te.mode)
		}

		// Type replacement
		te.InsertChar('!')
		te.ExecuteCommand("exit-insert-mode")

		// Should have replaced 'o' with '!'
		AssertBufferEquals(t, te, "Hell!")
		AssertCursorAt(t, te, 4, 0) // Cursor on '!'
	})

	t.Run("c right on empty line does nothing", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "\nHello")

		// Stay on first line which is empty
		AssertCursorAt(t, te, 0, 0)

		// Execute "c right"
		te.TypeKeys("c")
		te.TypeKeys("right")

		// Should still be in normal mode (no change happened)
		if te.mode != commands.NormalMode {
			t.Errorf("Expected normal mode when 'c right' fails, got %v", te.mode)
		}

		// Buffer unchanged
		AssertBufferEquals(t, te, "\nHello")
	})

	t.Run("cc changes entire line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Move to line 2
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 1)

		// Execute cc
		te.TypeKeys("cc")

		// Should be in insert mode with line 2 cleared
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'cc', got %v", te.mode)
		}

		// Type replacement text
		te.InsertChar('N')
		te.InsertChar('e')
		te.InsertChar('w')
		te.ExecuteCommand("exit-insert-mode")

		// Line 2 should be replaced
		AssertBufferEquals(t, te, "Line 1\nNew\nLine 3")
	})

	t.Run("cw changes word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello world test")

		// Execute cw at beginning
		te.TypeKeys("cw")

		// Should be in insert mode after deleting "Hello"
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'cw', got %v", te.mode)
		}

		// Type replacement text
		te.InsertChar('H')
		te.InsertChar('i')
		te.ExecuteCommand("exit-insert-mode")

		// Should have replaced "Hello" with "Hi"
		AssertBufferEquals(t, te, "Hi world test")
		AssertCursorAt(t, te, 1, 0) // Cursor on 'i'
	})

	t.Run("repeat c left with dot", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")

		// Move to position 5
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}

		// Execute "c left" and replace with 'a'
		te.TypeKeys("c")
		te.TypeKeys("left")
		te.InsertChar('a')
		te.ExecuteCommand("exit-insert-mode")

		// Should have "Hella World"
		AssertBufferEquals(t, te, "Hella World")

		// Move right a few positions
		for i := 0; i < 3; i++ {
			te.ExecuteCommand("right")
		}

		// Press dot to repeat
		te.TypeKeys(".")

		// Should have changed 'W' to 'a'
		AssertBufferEquals(t, te, "Hella aorld")
	})

	t.Run("change with multiple cursors deletes at all positions", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "foo bar\nfoo baz\nfoo end")

		// Add virtual cursors at each "foo" on different lines
		te.virtualCursors = []VirtualCursor{
			{X: 0, Y: 1}, // Second line foo
			{X: 0, Y: 2}, // Third line foo
		}

		// Execute cw to change all "foo" occurrences
		te.TypeKeys("cw")

		// Should be in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'cw', got %v", te.mode)
		}

		// All "foo" should be deleted from each line
		AssertBufferEquals(t, te, " bar\n baz\n end")

		// Check virtual cursor state after cw
		virtualCursorsAfterCw := te.GetVirtualCursors()
		t.Logf("Virtual cursors after cw: %d", len(virtualCursorsAfterCw))
		for i, vc := range virtualCursorsAfterCw {
			t.Logf("Virtual cursor %d: (%d, %d)", i, vc.X, vc.Y)
		}

		// Type replacement at all cursors (fixed behavior)
		te.TypeKeys("bar")
		te.ExecuteCommand("exit-insert-mode")

		// All cursor positions should get the inserted text
		AssertBufferEquals(t, te, "bar bar\nbar baz\nbar end")

		// Virtual cursors should be preserved during change operations
		if len(te.virtualCursors) == 0 {
			t.Errorf("Expected virtual cursors to be preserved, but they were cleared")
		}
	})

	t.Run("c up changes current line and line above", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Move to line 2 (index 1)
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 1)

		// Execute "c up"
		te.TypeKeys("c")
		te.TypeKeys("up")

		// Should be in insert mode after deleting lines 1 and 2
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'c up', got %v", te.mode)
		}

		// Type replacement text
		te.InsertChar('N')
		te.InsertChar('e')
		te.InsertChar('w')
		te.ExecuteCommand("exit-insert-mode")

		// Lines 1 and 2 should be replaced with "New"
		AssertBufferEquals(t, te, "New\nLine 3")
		AssertCursorAt(t, te, 2, 0) // Cursor on 'w' of "New"
	})

	t.Run("c down changes current line and line below", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Stay on line 1
		AssertCursorAt(t, te, 0, 0)

		// Execute "c down"
		te.TypeKeys("c")
		te.TypeKeys("down")

		// Should be in insert mode after deleting lines 1 and 2
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'c down', got %v", te.mode)
		}

		// Type replacement text
		te.InsertChar('M')
		te.InsertChar('e')
		te.InsertChar('r')
		te.InsertChar('g')
		te.InsertChar('e')
		te.InsertChar('d')
		te.ExecuteCommand("exit-insert-mode")

		// Lines 1 and 2 should be replaced with "Merged"
		AssertBufferEquals(t, te, "Merged\nLine 3")
		AssertCursorAt(t, te, 5, 0) // Cursor on 'd' of "Merged"
	})

	t.Run("c up at first line does nothing", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Already at first line
		AssertCursorAt(t, te, 0, 0)

		// Execute "c up"
		te.TypeKeys("c")
		te.TypeKeys("up")

		// Should still be in normal mode (no change happened)
		if te.mode != commands.NormalMode {
			t.Errorf("Expected normal mode when 'c up' fails, got %v", te.mode)
		}

		// Buffer unchanged
		AssertBufferEquals(t, te, "Line 1\nLine 2\nLine 3")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("c down at last line does nothing", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Move to last line
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 2)

		// Execute "c down"
		te.TypeKeys("c")
		te.TypeKeys("down")

		// Should still be in normal mode (no change happened)
		if te.mode != commands.NormalMode {
			t.Errorf("Expected normal mode when 'c down' fails, got %v", te.mode)
		}

		// Buffer unchanged
		AssertBufferEquals(t, te, "Line 1\nLine 2\nLine 3")
		AssertCursorAt(t, te, 0, 2)
	})
}
