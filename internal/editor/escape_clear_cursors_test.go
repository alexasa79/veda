package editor

import (
	"testing"
	"veda/internal/events"
)

func TestEscapeClearsVirtualCursors(t *testing.T) {
	t.Run("escape in normal mode clears virtual cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
		
		// Create multiple cursors
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("add-virtual-cursor")
		te.ExecuteCommand("add-virtual-cursor")
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode
		
		// Verify we have virtual cursors
		if len(te.virtualCursors) != 2 {
			t.Fatalf("Expected 2 virtual cursors, got %d", len(te.virtualCursors))
		}
		
		// Press escape in normal mode
		te.PressKey(events.KeyEscape, 0)
		
		// Verify virtual cursors are cleared
		if len(te.virtualCursors) != 0 {
			t.Errorf("Expected 0 virtual cursors after escape, got %d", len(te.virtualCursors))
		}
	})
	
	t.Run("escape with pending operation cancels operation not cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
		
		// Create multiple cursors
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("add-virtual-cursor")
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode
		
		// Verify we have virtual cursor
		if len(te.virtualCursors) != 1 {
			t.Fatalf("Expected 1 virtual cursor, got %d", len(te.virtualCursors))
		}
		
		// Start a pending operation (e.g., 'd' for delete)
		te.TypeKeys("d")
		
		// Press escape - should cancel operation but keep cursors
		te.PressKey(events.KeyEscape, 0)
		
		// Verify virtual cursors are still there
		if len(te.virtualCursors) != 1 {
			t.Errorf("Expected 1 virtual cursor after canceling operation, got %d", len(te.virtualCursors))
		}
		
		// Press escape again - now should clear cursors
		te.PressKey(events.KeyEscape, 0)
		
		// Verify virtual cursors are cleared
		if len(te.virtualCursors) != 0 {
			t.Errorf("Expected 0 virtual cursors after second escape, got %d", len(te.virtualCursors))
		}
	})
	
	t.Run("escape in insert mode doesn't clear cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
		
		// Create multiple cursors
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("add-virtual-cursor")
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode
		
		// Verify we have virtual cursor
		if len(te.virtualCursors) != 1 {
			t.Fatalf("Expected 1 virtual cursor, got %d", len(te.virtualCursors))
		}
		
		// Enter insert mode
		te.TypeKeys("i")
		
		// Press escape - should exit insert mode but keep cursors
		te.PressKey(events.KeyEscape, 0)
		
		// Verify virtual cursors are still there
		if len(te.virtualCursors) != 1 {
			t.Errorf("Expected 1 virtual cursor after exiting insert mode, got %d", len(te.virtualCursors))
		}
	})
}