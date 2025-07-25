package editor

import (
	"testing"
)

func TestEditorUndoRedo(t *testing.T) {
	t.Run("basic undo redo with insert mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Type some text
		te.InsertChar('H')
		te.InsertChar('e')
		te.InsertChar('l')
		te.InsertChar('l')
		te.InsertChar('o')
		
		// Exit insert mode
		te.ExecuteCommand("exit-insert-mode")
		
		// Verify text
		AssertBufferEquals(t, te, "Hello")
		
		// Undo (should remove entire insert session)
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "")
		
		// Redo (should restore entire insert session)
		te.ExecuteCommand("redo")
		AssertBufferEquals(t, te, "Hello")
	})
	
	t.Run("multiple insert sessions", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// First insert session
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar('F')
		te.InsertChar('i')
		te.InsertChar('r')
		te.InsertChar('s')
		te.InsertChar('t')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "First")
		
		// Use 'o' to open a new line and add text there
		te.ExecuteCommand("open-line-below")
		te.InsertChar('S')
		te.InsertChar('e')
		te.InsertChar('c')
		te.InsertChar('o')
		te.InsertChar('n')
		te.InsertChar('d')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "First\nSecond")
		
		// Undo second session (the insert of "Second")
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "First\n")
		
		// Undo the open-line
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "First")
		
		// Undo first session
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "")
		
		// Redo all three
		te.ExecuteCommand("redo") // First session
		AssertBufferEquals(t, te, "First")
		
		te.ExecuteCommand("redo") // Open line
		AssertBufferEquals(t, te, "First\n")
		
		te.ExecuteCommand("redo") // Second session
		AssertBufferEquals(t, te, "First\nSecond")
	})
	
	t.Run("undo redo with backspace", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// Use open-line to add text on new line
		te.ExecuteCommand("open-line-below")
		te.InsertChar('W')
		te.InsertChar('o')
		te.InsertChar('r')
		te.ExecuteCommand("backspace") // Delete 'r'
		te.ExecuteCommand("backspace") // Delete 'o'
		te.InsertChar('o')
		te.InsertChar('r')
		te.InsertChar('l')
		te.InsertChar('d')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "Hello\nWorld")
		
		// Undo insert session
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Hello\n")
		
		// Undo open-line
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Hello")
	})
	
	t.Run("undo redo with newlines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "First line")
		
		// Use open-line-below which is cleaner
		te.ExecuteCommand("open-line-below")
		te.InsertChar('S')
		te.InsertChar('e')
		te.InsertChar('c')
		te.InsertChar('o')
		te.InsertChar('n')
		te.InsertChar('d')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "First line\nSecond")
		
		// Undo insert session
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "First line\n")
		
		// Undo open-line
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "First line")
		
		// Redo open-line
		te.ExecuteCommand("redo")
		AssertBufferEquals(t, te, "First line\n")
		
		// Redo insert session
		te.ExecuteCommand("redo")
		AssertBufferEquals(t, te, "First line\nSecond")
	})
	
	t.Run("undo delete character command", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// Delete 'H' with x command
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "ello")
		
		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Hello")
		
		// Redo
		te.ExecuteCommand("redo")
		AssertBufferEquals(t, te, "ello")
	})
	
	t.Run("multiple x commands", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "ABCDE")
		
		// Delete multiple characters
		te.ExecuteCommand("delete-current-character") // Delete 'A'
		te.ExecuteCommand("delete-current-character") // Delete 'B'
		te.ExecuteCommand("delete-current-character") // Delete 'C'
		
		AssertBufferEquals(t, te, "DE")
		
		// Undo all three
		te.ExecuteCommand("undo") // Restore 'C'
		AssertBufferEquals(t, te, "CDE")
		
		te.ExecuteCommand("undo") // Restore 'B'
		AssertBufferEquals(t, te, "BCDE")
		
		te.ExecuteCommand("undo") // Restore 'A'
		AssertBufferEquals(t, te, "ABCDE")
	})
	
	t.Run("undo after new operation clears redo", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// Add text
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar('O')
		te.InsertChar('n')
		te.InsertChar('e')
		te.ExecuteCommand("exit-insert-mode")
		
		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "")
		
		// Add different text (should clear redo stack)
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar('T')
		te.InsertChar('w')
		te.InsertChar('o')
		te.ExecuteCommand("exit-insert-mode")
		
		// Try redo - should do nothing
		te.ExecuteCommand("redo")
		AssertBufferEquals(t, te, "Two") // Should still be "Two", not "One"
	})
	
	t.Run("undo redo with open line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line1\nLine2")
		
		// Open line below
		te.ExecuteCommand("open-line-below")
		te.InsertChar('N')
		te.InsertChar('e')
		te.InsertChar('w')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "Line1\nNew\nLine2")
		
		// Since open-line-below creates the newline with the Enter key,
		// and we typed "New", we need to check what actually gets created
		
		// Undo insert session (removes "New")
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Line1\n\nLine2")
		
		// Undo open line (removes the empty line)
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Line1\nLine2")
	})
	
	t.Run("empty insert session", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Test")
		
		// Enter and exit insert mode without changes
		te.ExecuteCommand("enter-insert-mode")
		te.ExecuteCommand("exit-insert-mode")
		
		// Add actual change
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "est")
		
		// Undo should undo the delete, not do nothing
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Test")
	})
}

func TestUndoRedoCursorPosition(t *testing.T) {
	t.Run("cursor position after undo insert", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// Move cursor to middle
		te.SetCursorPosition(2, 0)
		
		// Insert text
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar('X')
		te.InsertChar('Y')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "HeXYllo")
		// Debug cursor position
		x, y := te.GetCursorPosition()
		t.Logf("After exit insert mode - cursor at: (%d, %d)", x, y)
		AssertCursorAt(t, te, 3, 0) // On 'Y' in normal mode (vim behavior)
		
		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Hello")
		AssertCursorAt(t, te, 2, 0) // Back to original position
	})
	
	t.Run("cursor position after undo delete", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// Move cursor to 'l'
		te.SetCursorPosition(3, 0)
		
		// Delete character
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "Helo")
		AssertCursorAt(t, te, 3, 0) // Cursor stays at same position
		
		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Hello")
		AssertCursorAt(t, te, 3, 0) // Cursor restored
	})
}

func TestUndoRedoEdgeCases(t *testing.T) {
	t.Run("undo on empty buffer", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// Should not crash
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "")
	})
	
	t.Run("redo without undo", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Test")
		
		// Should not crash
		te.ExecuteCommand("redo")
		AssertBufferEquals(t, te, "Test")
	})
	
	t.Run("delete at end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Test")
		
		// Move to last character
		te.SetCursorPosition(3, 0)
		
		// Delete
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "Tes")
		
		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Test")
	})
	
	t.Run("delete on empty line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "\n")
		
		// Try to delete on empty line
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "") // Should delete the newline
		
		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "\n")
	})
}