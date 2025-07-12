package editor

import (
	"testing"
)

func TestCursorMovement(t *testing.T) {
	t.Run("basic left and right movement", func(t *testing.T) {
		// Create editor with simple content
		te := NewTestEditorWithContent(80, 24, "Hello, World!")
		
		// Start at position 0,0
		AssertCursorAt(t, te, 0, 0)
		
		// Move right
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 1, 0)
		
		// Move right again
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 0)
		
		// Move left
		te.ExecuteCommand("left")
		AssertCursorAt(t, te, 1, 0)
		
		// Move left to beginning
		te.ExecuteCommand("left")
		AssertCursorAt(t, te, 0, 0)
		
		// Try to move left past beginning (should stay at 0)
		te.ExecuteCommand("left")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("vertical movement between lines of different lengths", func(t *testing.T) {
		// Create editor with the test case you mentioned
		te := NewTestEditorWithContent(80, 24, "11111\n222")
		
		// Move to last '1' (position 4,0)
		te.SetCursorPosition(4, 0)
		AssertCursorAt(t, te, 4, 0)
		
		// Move down - cursor should move to end of shorter line
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 2, 1) // Should be at last '2'
		
		// Move back up
		te.ExecuteCommand("up")
		AssertCursorAt(t, te, 2, 0) // Cursor stays at position 2
	})
	
	t.Run("boundary conditions", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line1\nLine2\nLine3")
		
		// Try to move up from first line
		AssertCursorAt(t, te, 0, 0)
		te.ExecuteCommand("up")
		AssertCursorAt(t, te, 0, 0) // Should stay at top
		
		// Move to last line
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 2)
		
		// Try to move down from last line
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 2) // Should stay at bottom
		
		// Move right a few times
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 2)
	})
	
	t.Run("empty lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line1\n\nLine3")
		
		// Move to empty line
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 1)
		
		// Cursor should be at position 0 on empty line
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 0, 1) // Can't move right on empty line
		
		// Move down to Line3
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 2)
	})
}