package editor

import (
	"testing"
	"veda/internal/events"
)

func TestTabCharacterCursorMovement(t *testing.T) {
	t.Run("cursor moves past tab character correctly", func(t *testing.T) {
		// Create editor with content containing tabs
		te := NewTestEditorWithContent(80, 24, "hello\tworld")
		
		// Move cursor to position 5 (just before the tab)
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 5, 0)
		
		// Move right once - should skip over the tab to 'w'
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 6, 0) // Position after tab
		
		// Move left once - should go back to position before tab
		te.ExecuteCommand("left")
		AssertCursorAt(t, te, 5, 0)
	})
	
	t.Run("visual cursor position accounts for tab width", func(t *testing.T) {
		// Create editor with tab at beginning
		te := NewTestEditorWithContent(80, 24, "\thello")
		
		// Initially at position 0
		AssertCursorAt(t, te, 0, 0)
		
		// Get visual position - should be at column 0 (plus line number width)
		visualLine, visualCol := te.GetCursorVisualPosition()
		if visualLine != 0 {
			t.Errorf("Expected visual line 0, got %d", visualLine)
		}
		// Visual column should be line number width
		expectedVisualCol := te.buffer.LineNumberWidth
		if visualCol != expectedVisualCol {
			t.Errorf("Expected visual column %d, got %d", expectedVisualCol, visualCol)
		}
		
		// Move right once - cursor should be at position 1 (after tab)
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 1, 0)
		
		// Visual position should account for tab width (4 spaces)
		visualLine, visualCol = te.GetCursorVisualPosition()
		if visualLine != 0 {
			t.Errorf("Expected visual line 0, got %d", visualLine)
		}
		// Visual column should be line number width + 4 (tab width)
		expectedVisualCol = te.buffer.LineNumberWidth + 4
		if visualCol != expectedVisualCol {
			t.Errorf("Expected visual column %d, got %d", expectedVisualCol, visualCol)
		}
	})
	
	t.Run("multiple tabs render correctly", func(t *testing.T) {
		// Create editor with multiple tabs
		te := NewTestEditorWithContent(80, 24, "\t\thello\tworld")
		
		// Move to position after second tab (position 2)
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 0)
		
		// Visual position should be line number width + 8 (two tabs)
		_, visualCol := te.GetCursorVisualPosition()
		expectedVisualCol := te.buffer.LineNumberWidth + 8
		if visualCol != expectedVisualCol {
			t.Errorf("Expected visual column %d, got %d", expectedVisualCol, visualCol)
		}
		
		// Move to position after "hello" and before third tab
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 7, 0)
		
		// Visual position should account for two tabs + 5 characters
		_, visualCol = te.GetCursorVisualPosition()
		expectedVisualCol = te.buffer.LineNumberWidth + 8 + 5
		if visualCol != expectedVisualCol {
			t.Errorf("Expected visual column %d, got %d", expectedVisualCol, visualCol)
		}
	})
}

func TestTabCharacterWithMultipleCursors(t *testing.T) {
	t.Run("multiple cursors handle tabs correctly", func(t *testing.T) {
		// Create editor with tabs on multiple lines
		te := NewTestEditorWithContent(80, 24, "hello\tworld\nfoo\tbar")
		
		// Enter cursor mode and add a virtual cursor on second line
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("down")
		te.ExecuteCommand("add-virtual-cursor")
		te.ExecuteCommand("up") // Move back to first line
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode
		
		// Move both cursors to position 3
		for i := 0; i < 3; i++ {
			te.ExecuteCommand("right")
		}
		
		
		// Primary cursor should be at position 3, line 0
		AssertCursorAt(t, te, 3, 0)
		
		// Virtual cursor should be at position 3, line 1
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 1 {
			t.Fatalf("Expected 1 virtual cursor, got %d", len(virtualCursors))
		}
		// Check virtual cursor moved to position 3
		if virtualCursors[0].X != 3 {
			t.Errorf("Expected virtual cursor at x=3, got x=%d", virtualCursors[0].X)
		}
		
		// Move right three times - both cursors should skip over tabs
		for i := 0; i < 3; i++ {
			te.ExecuteCommand("right")
		}
		
		// Primary cursor should be at position 6 (after tab)
		AssertCursorAt(t, te, 6, 0)
		
		// Virtual cursor should also be at position 6 on line 1
		virtualCursors = te.GetVirtualCursors()
		if virtualCursors[0].X != 6 {
			t.Errorf("Expected virtual cursor at x=6, got x=%d", virtualCursors[0].X)
		}
	})
}