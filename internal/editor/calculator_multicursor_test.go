package editor

import (
	"testing"
	"veda/internal/buffer"
)

func TestCalculatorWithMultipleCursors(t *testing.T) {
	// Create editor with test content
	content := `10 + 5
20 * 2
30 - 10`
	ed := NewTestEditorWithContent(80, 24, content)
	
	// Add virtual cursors at the end of each line
	ed.Editor.virtualCursors = []VirtualCursor{
		{X: 6, Y: 0}, // End of "10 + 5"
		{X: 6, Y: 1}, // End of "20 * 2"
		{X: 7, Y: 2}, // End of "30 - 10"
	}
	
	// Move primary cursor to end of first line
	ed.Editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(ed.Editor.buffer, 0)
	ed.Editor.buffer.CursorX = 6
	
	// Test current behavior - calculator commands are NOT multi-cursor
	// This test documents the current limitation
	t.Run("calculator commands don't apply to all cursors", func(t *testing.T) {
		// Execute calculate-append command
		err := ed.ExecuteCommand("calculate-append")
		if err != nil {
			t.Fatalf("Failed to execute calculate-append: %v", err)
		}
		
		// Check that only the primary cursor line was modified
		lines := make([]string, ed.Editor.buffer.LineCount)
		for i := 0; i < ed.Editor.buffer.LineCount; i++ {
			iter := buffer.NewBufferIteratorAtLine(ed.Editor.buffer, i)
			lines[i] = iter.Line()
		}
		
		// First line should have the result appended
		if lines[0] != "10 + 5 → 15" {
			t.Errorf("Line 0: expected '10 + 5 → 15', got '%s'", lines[0])
		}
		
		// Other lines should be unchanged (since multi-cursor is not supported)
		if lines[1] != "20 * 2" {
			t.Errorf("Line 1: expected '20 * 2', got '%s'", lines[1])
		}
		
		if lines[2] != "30 - 10" {
			t.Errorf("Line 2: expected '30 - 10', got '%s'", lines[2])
		}
	})
}

func TestDesiredCalculatorMultiCursorBehavior(t *testing.T) {
	t.Skip("This test demonstrates the desired behavior - currently not implemented")
	
	// This test shows what SHOULD happen if calculator commands
	// supported multiple cursors
	
	content := `10 + 5
20 * 2
30 - 10`
	ed := NewTestEditorWithContent(80, 24, content)
	
	// Add virtual cursors at the end of each line
	ed.Editor.virtualCursors = []VirtualCursor{
		{X: 6, Y: 0},
		{X: 6, Y: 1},
		{X: 7, Y: 2},
	}
	
	// If calculate-append supported multi-cursor, it would:
	// 1. Process each cursor position
	// 2. Evaluate the expression at each cursor
	// 3. Append results to each line
	
	// Expected result after multi-cursor calculate-append:
	expectedLines := []string{
		"10 + 5 → 15",
		"20 * 2 → 40",
		"30 - 10 → 20",
	}
	
	// This would be the implementation if it worked
	err := ed.ExecuteCommand("calculate-append")
	if err != nil {
		t.Fatalf("Failed to execute calculate-append: %v", err)
	}
	
	lines := make([]string, ed.Editor.buffer.LineCount)
	for i := 0; i < ed.Editor.buffer.LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(ed.Editor.buffer, i)
		lines[i] = iter.Line()
	}
	for i, expected := range expectedLines {
		if lines[i] != expected {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected, lines[i])
		}
	}
}