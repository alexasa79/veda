package editor

import (
	"testing"
)

func TestDesiredColumnTracking(t *testing.T) {
	t.Run("basic desired column behavior", func(t *testing.T) {
		// Create editor with content
		content := `111
22
333`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move cursor to the last "3" (line 2, position 2)
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 2) // Line 2, position 2
		
		// Move up - should go to end of "22" (line 1, position 1)
		te.ExecuteCommand("up")
		AssertCursorAt(t, te, 1, 1) // Line 1, position 1 (last char of "22")
		
		// Move up again - should remember column 2 and go to last "1" (line 0, position 2)
		te.ExecuteCommand("up")
		AssertCursorAt(t, te, 2, 0) // Line 0, position 2 (last "1")
	})
	
	t.Run("desired column resets on horizontal movement", func(t *testing.T) {
		content := `12345
1
12345`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move to position 4 on first line
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 4, 0) // Line 0, position 4
		
		// Move down - should go to end of short line
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 1) // Line 1, position 0 (only char)
		
		// Move down again - should remember column 4
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 4, 2) // Line 2, position 4
		
		// Move left (horizontal movement) - resets desired column
		te.ExecuteCommand("left")
		AssertCursorAt(t, te, 3, 2) // Line 2, position 3
		
		// Move up - should now use column 3 instead of 4
		te.ExecuteCommand("up")
		AssertCursorAt(t, te, 0, 1) // Line 1, position 0 (short line)
		
		// Move up again - should use column 3
		te.ExecuteCommand("up")
		AssertCursorAt(t, te, 3, 0) // Line 0, position 3
	})
	
	t.Run("desired column in insert mode", func(t *testing.T) {
		content := `12345
1
12345`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Move to end of first line
		te.ExecuteCommand("end-of-line-insert")
		AssertCursorAt(t, te, 5, 0) // In insert mode, cursor can be after last char
		
		// Move down - should go to end of short line
		te.ExecuteCommand("down-insert")
		AssertCursorAt(t, te, 1, 1) // Line 1, position 1 (after only char)
		
		// Move down again - should remember column 5
		te.ExecuteCommand("down-insert")
		AssertCursorAt(t, te, 5, 2) // Line 2, position 5
	})
	
	t.Run("desired column resets on text insertion", func(t *testing.T) {
		content := `12345
1
12345`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move to position 4 on first line
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 4, 0)
		
		// Move down to short line
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 1)
		
		// Enter insert mode and type a character
		te.ExecuteCommand("enter-insert-mode")
		te.TypeKeys("x")
		
		// Exit insert mode
		te.ExecuteCommand("exit-insert-mode")
		AssertCursorAt(t, te, 0, 1) // After exit-insert-mode, cursor adjusts back one position
		
		// Move down - should use current column (0) not original desired column (4)
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 2) // Line 2, position 0
	})
	
	t.Run("word movement resets desired column", func(t *testing.T) {
		content := `hello world test
short
hello world test`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move to end of "test" on first line
		te.ExecuteCommand("end-of-line")
		AssertCursorAt(t, te, 15, 0) // Last char of "test"
		
		// Move down - should go to end of short line
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 4, 1) // Last char of "short"
		
		// Move down again - should remember column 15
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 15, 2) // Last char of "test" on line 2
		
		// Move word backward - resets desired column
		te.ExecuteCommand("word-backward")
		
		// Move up - should use new column position
		te.ExecuteCommand("up")
		te.ExecuteCommand("up")
		// Should be at beginning of "world" on first line (not at column 15)
		x, _ := te.GetCursorPosition()
		if x == 15 {
			t.Error("Desired column should have been reset by word movement")
		}
	})
}