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

	// Add virtual cursors at the end of the remaining lines
	ed.Editor.virtualCursors = []VirtualCursor{
		{X: 6, Y: 1}, // End of "20 * 2"
		{X: 7, Y: 2}, // End of "30 - 10"
	}
	ed.Editor.syncCurrentVirtualCursors()

	// Move primary cursor to end of first line
	ed.Editor.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(ed.Editor.CurrentBuffer(), 0)
	ed.Editor.CurrentBuffer().CursorX = 6

	// Execute calculate-append command across all cursors
	if err := ed.ExecuteCommand("calculate-append"); err != nil {
		t.Fatalf("Failed to execute calculate-append: %v", err)
	}

	// Verify every line received the appended result
	expected := []string{"10 + 5 → 15", "20 * 2 → 40", "30 - 10 → 20"}
	for i := 0; i < ed.Editor.CurrentBuffer().LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(ed.Editor.CurrentBuffer(), i)
		if got := iter.Line(); got != expected[i] {
			t.Fatalf("Line %d: expected %q, got %q", i, expected[i], got)
		}
	}

	// Undo once should revert all lines in a single step
	if err := ed.Editor.CurrentBuffer().History.Undo(ed.Editor.CurrentBuffer()); err != nil {
		t.Fatalf("undo failed: %v", err)
	}

	original := []string{"10 + 5", "20 * 2", "30 - 10"}
	for i := 0; i < ed.Editor.CurrentBuffer().LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(ed.Editor.CurrentBuffer(), i)
		if got := iter.Line(); got != original[i] {
			t.Fatalf("Line %d after undo: expected %q, got %q", i, original[i], got)
		}
	}
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
