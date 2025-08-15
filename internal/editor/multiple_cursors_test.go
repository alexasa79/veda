package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestMultipleCursorsTextEditing(t *testing.T) {
	t.Run("insert character at multiple cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello\nWorld\nTest")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)
		
		// Move down and add another virtual cursor at (0,1)
		te.PressKey(events.KeyEnter, 0)
		
		// Exit cursor mode
		te.TypeKeys("m")
		
		// Enter insert mode
		te.TypeKeys("i")
		
		// Type "X" - should insert at all 3 cursors
		te.TypeKeys("X")
		
		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)
		
		// Check buffer content
		expected := "XHello\nXWorld\nXTest"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
	})
	
	t.Run("backspace at multiple cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello\nWorld\nTest")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at end of "Hello" (5,0)
		te.TypeKeys("$")
		te.PressKey(events.KeyEnter, 0)
		
		// Now at line 1 after auto-move, go to end of "World" and add cursor
		te.TypeKeys("$")
		te.PressKey(events.KeyEnter, 0)
		
		// Now at line 2 after auto-move, go to end of "Test"
		te.TypeKeys("$")
		
		// Exit cursor mode
		te.TypeKeys("m")
		
		// Enter insert mode with 'a' (append) to position cursor after last char
		te.TypeKeys("a")
		
		// Backspace - should delete last char from each line
		te.ExecuteCommand("backspace")
		
		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)
		
		// Check buffer content
		expected := "Hell\nWorl\nTes"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
	})
	
	t.Run("movement commands apply to all cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "abc def\nabc def\nabc def")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)
		
		// Move down and add another virtual cursor at (0,1)
		te.PressKey(events.KeyEnter, 0)
		
		// Exit cursor mode
		te.TypeKeys("m")
		
		// Move all cursors to next word
		te.TypeKeys("w")
		
		// Enter insert mode
		te.TypeKeys("i")
		
		// Type "X" - should insert at position 4 on all lines
		te.TypeKeys("X")
		
		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)
		
		// Check buffer content
		expected := "abc Xdef\nabc Xdef\nabc Xdef"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
		
		// Verify virtual cursors are at correct positions
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors, got %d", len(virtualCursors))
		}
		if len(virtualCursors) >= 2 {
			// After inserting X and exiting insert mode, cursor should be at position 4
			if virtualCursors[0].X != 4 || virtualCursors[0].Y != 0 {
				t.Errorf("Expected first virtual cursor at (4,0), got (%d,%d)", 
					virtualCursors[0].X, virtualCursors[0].Y)
			}
			if virtualCursors[1].X != 4 || virtualCursors[1].Y != 1 {
				t.Errorf("Expected second virtual cursor at (4,1), got (%d,%d)", 
					virtualCursors[1].X, virtualCursors[1].Y)
			}
		}
	})
	
	t.Run("cursor mode movement only affects primary cursor", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)
		
		// Move down (should only move primary cursor)
		te.TypeKeys("down")
		
		// Add another virtual cursor
		te.PressKey(events.KeyEnter, 0)
		
		// Exit cursor mode
		te.TypeKeys("m")
		
		// Verify virtual cursors
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors, got %d", len(virtualCursors))
		}
		if len(virtualCursors) >= 2 {
			// First cursor should still be at line 0
			if virtualCursors[0].Y != 0 {
				t.Errorf("Expected first virtual cursor on line 0, got line %d", 
					virtualCursors[0].Y)
			}
			// Second cursor should be at line 2 (where we added it after moving down twice)
			// First move is automatic after adding first cursor, second is explicit "down"
			if virtualCursors[1].Y != 2 {
				t.Errorf("Expected second virtual cursor on line 2, got line %d", 
					virtualCursors[1].Y)
			}
		}
	})
	
	t.Run("cw should work with multiple cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world\nhello world\nhello world")
		
		// First, move to the 'w' in "world" on first line (position 6)
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at current position (6,0)
		te.PressKey(events.KeyEnter, 0)
		
		// Move down and add another virtual cursor at (6,1)
		te.PressKey(events.KeyEnter, 0)
		
		// Exit cursor mode - now we have 3 cursors all pointing to 'w' in "world"
		te.TypeKeys("m")
		
		// Execute cw to change "world" to something else
		te.TypeKeys("cw")
		
		// Should be in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'cw', got %v", te.mode)
		}
		
		// Verify that words were deleted and we're in insert mode with virtual cursors preserved
		afterCw := te.GetBufferContent()
		expectedAfterCw := "hello \nhello \nhello "
		if afterCw != expectedAfterCw {
			t.Errorf("Expected buffer after cw:\n%q\nBut got:\n%q", expectedAfterCw, afterCw)
		}
		
		// Check that virtual cursors are preserved
		virtualCursorsAfterCw := te.GetVirtualCursors()
		if len(virtualCursorsAfterCw) != 2 {
			t.Errorf("Expected 2 virtual cursors after cw, got %d", len(virtualCursorsAfterCw))
		}
		
		// Type "hello" to replace "world" at all cursor positions
		te.TypeKeys("hello")
		
		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)
		
		// Expected result: all three lines should say "hello hello"
		expected := "hello hello\nhello hello\nhello hello"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
	})
	
	t.Run("dw followed by insert should work with multiple cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world\nhello world\nhello world")
		
		// First, move to the 'w' in "world" on first line (position 6)
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at current position (6,0)
		te.PressKey(events.KeyEnter, 0)
		
		// Move down and add another virtual cursor at (6,1)
		te.PressKey(events.KeyEnter, 0)
		
		// Exit cursor mode - now we have 3 cursors all pointing to 'w' in "world"
		te.TypeKeys("m")
		
		// Execute dw to delete "world" from all positions
		te.TypeKeys("dw")
		
		// Should be in normal mode after dw
		if te.mode != commands.NormalMode {
			t.Errorf("Expected normal mode after 'dw', got %v", te.mode)
		}
		
		// Verify that "world" was deleted from all lines
		afterDw := te.GetBufferContent()
		expectedAfterDw := "hello \nhello \nhello "
		if afterDw != expectedAfterDw {
			t.Errorf("Expected buffer after dw:\n%q\nBut got:\n%q", expectedAfterDw, afterDw)
		}
		
		// Check virtual cursor state after dw  
		virtualCursorsAfterDw := te.GetVirtualCursors()
		if len(virtualCursorsAfterDw) == 0 {
			t.Errorf("Expected virtual cursors to be preserved after dw, but they were cleared")
		}
		
		// Now enter insert mode
		te.TypeKeys("i")
		
		// Should be in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'i', got %v", te.mode)
		}
		
		// Type "hello" - this should work at all cursor positions
		te.TypeKeys("hello")
		
		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)
		
		// Expected result: all three lines should say "hello hello"
		expected := "hello hello\nhello hello\nhello hello"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
	})
}