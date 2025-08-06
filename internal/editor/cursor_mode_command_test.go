package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestCursorModeCommandExecution(t *testing.T) {
	t.Run("D command in cursor mode applies to all cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "First line text\nSecond line text\nThird line text")
		
		// Enter cursor mode
		te.ExecuteCommand("enter-cursor-mode")
		
		// Add virtual cursor at beginning of first line
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (0,0), moves to line 1
		
		// Add virtual cursor at beginning of second line  
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (0,1), moves to line 2
		
		// Now we're in cursor mode with cursors at lines 0, 1, and primary cursor at line 2
		// Press 'D' while still in cursor mode - it should exit cursor mode and apply to all cursors
		te.TypeKeys("D")
		
		// All three lines should be empty
		AssertBufferEquals(t, te, "\n\n")
		
		// Should be back in normal mode
		if te.mode != commands.NormalMode {
			t.Errorf("Expected to be in normal mode after D command, but mode is %s", te.mode)
		}
		
		// Virtual cursors should still exist (they persist until cleared)
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors to remain, but found %d", len(virtualCursors))
		}
	})
	
	t.Run("x command in cursor mode applies to all cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "ABC\nDEF\nGHI")
		
		// Enter cursor mode
		te.ExecuteCommand("enter-cursor-mode")
		
		// Add cursors at beginning of each line
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (0,0), moves to line 1
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (0,1), moves to line 2
		
		// Press 'x' while in cursor mode - should delete first character of each line
		te.TypeKeys("x")
		
		// First character of each line should be deleted
		AssertBufferEquals(t, te, "BC\nEF\nHI")
		
		// Should be back in normal mode
		if te.mode != commands.NormalMode {
			t.Errorf("Expected to be in normal mode after x command, but mode is %s", te.mode)
		}
	})
	
	t.Run("movement commands in cursor mode don't exit cursor mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Enter cursor mode
		te.ExecuteCommand("enter-cursor-mode")
		
		// Add a virtual cursor
		te.ExecuteCommand("add-virtual-cursor")
		
		// Move right with arrow key - should stay in cursor mode
		te.PressKey(events.KeyRight, 0)
		
		// Should still be in cursor mode
		if te.mode != commands.CursorMode {
			t.Errorf("Expected to remain in cursor mode after movement, but mode is %s", te.mode)
		}
		
		// Cursor should have moved
		x, _ := te.GetCursorPosition()
		if x != 1 {
			t.Errorf("Expected cursor to move to position 1, but it's at %d", x)
		}
	})
	
	t.Run("entering insert mode from cursor mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")
		
		// Enter cursor mode and add cursors
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("add-virtual-cursor")
		te.ExecuteCommand("add-virtual-cursor")
		
		// Press 'i' to enter insert mode - should exit cursor mode and enter insert mode
		te.TypeKeys("i")
		
		// Should be in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected to be in insert mode, but mode is %s", te.mode)
		}
		
		// Virtual cursors should still exist for multi-cursor editing
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors for multi-cursor insert, but found %d", len(virtualCursors))
		}
		
		// Type some text - should be inserted at all cursor positions
		te.TypeKeys("X")
		
		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)
		
		// Check that 'X' was inserted at the beginning of each line
		AssertBufferEquals(t, te, "XLine 1\nXLine 2\nXLine 3")
	})
}