package editor

import (
	"testing"
)

// Ensure calculator includes the last character in Normal mode (cursor ON char)
// and also works in Insert mode (cursor AFTER char).
func TestCalculatorExpressionEndPositionByMode(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "1+2")
	te.CurrentBuffer().FilePath = "calc.txt"

	// Place cursor ON the last character ('2') to simulate Normal mode semantics
	te.CurrentBuffer().CursorIterator = te.CurrentBuffer().CursorIterator // same line
	te.CurrentBuffer().CursorX = 2                                        // index of '2'

	if err := te.ExecuteCommand("calculate-append"); err != nil {
		t.Fatalf("calculate-append failed in normal mode case: %v", err)
	}
	got := te.CurrentBuffer().Rope.String()
	if got != "1+2 → 3" {
		t.Fatalf("normal mode end position: expected '1+2 → 3', got %q", got)
	}

	// Reset buffer to original and test Insert mode semantics (cursor AFTER '2')
	te = NewTestEditorWithContent(80, 24, "3*4")
	te.CurrentBuffer().FilePath = "calc2.txt"

	// Enter insert mode then move cursor to end (after '4')
	te.ExecuteCommand("enter-insert-mode")
	te.CurrentBuffer().CursorIterator = te.CurrentBuffer().CursorIterator
	te.CurrentBuffer().CursorX = 3 // after last char

	if err := te.ExecuteCommand("calculate-append"); err != nil {
		t.Fatalf("calculate-append failed in insert mode case: %v", err)
	}
	got2 := te.CurrentBuffer().Rope.String()
	if got2 != "3*4 → 12" {
		t.Fatalf("insert mode end position: expected '3*4 → 12', got %q", got2)
	}
}
