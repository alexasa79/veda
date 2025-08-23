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
	ed.Editor.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(ed.Editor.CurrentBuffer(), 0)
	ed.Editor.CurrentBuffer().CursorX = 6

	// Test current behavior - calculator commands are NOT multi-cursor
	// This test documents the current limitation
	t.Run("calculator commands don't apply to all cursors", func(t *testing.T) {
		// Execute calculate-append command
		err := ed.ExecuteCommand("calculate-append")
		if err != nil {
			t.Fatalf("Failed to execute calculate-append: %v", err)
		}

		// Check that only the primary cursor line was modified
		lines := make([]string, ed.Editor.CurrentBuffer().LineCount)
		for i := 0; i < ed.Editor.CurrentBuffer().LineCount; i++ {
			iter := buffer.NewBufferIteratorAtLine(ed.Editor.CurrentBuffer(), i)
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

func TestCalculatorMultiCursorSequentialExecution(t *testing.T) {
	// This test verifies we can achieve multi-cursor-like results by
	// sequentially running the command at each cursor position.
	content := `10 + 5
20 * 2
30 - 10`
	ed := NewTestEditorWithContent(80, 24, content)

	// For each line, position the cursor AFTER the last character (inclusive end)
	for lineNum := 0; lineNum < ed.Editor.CurrentBuffer().LineCount; lineNum++ {
		iter := buffer.NewBufferIteratorAtLine(ed.Editor.CurrentBuffer(), lineNum)
		line := iter.Line()
		ed.Editor.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(ed.Editor.CurrentBuffer(), lineNum)
		ed.Editor.CurrentBuffer().CursorX = len(line) // inclusive end, so the full expression is considered
		if err := ed.ExecuteCommand("calculate-append"); err != nil {
			t.Fatalf("calculate-append failed at line %d: %v", lineNum, err)
		}
	}

	// Validate expected results on all lines
	expected := []string{"10 + 5 → 15", "20 * 2 → 40", "30 - 10 → 20"}
	for i := 0; i < ed.Editor.CurrentBuffer().LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(ed.Editor.CurrentBuffer(), i)
		if got := iter.Line(); got != expected[i] {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected[i], got)
		}
	}
}
