package editor

import (
	"testing"
	"veda/internal/buffer"
)

// Scenario: lines 1, $?+1, $?+1, $?+1 with cursors on each line.
// Expect results 1,2,3,4 and cursors at the end of each number.
func TestCalculatorPlusOneSequentialMultiCursor(t *testing.T) {
	content := "1\n$?+1\n$?+1\n$?+1"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().FilePath = "calc_plus1.txt"

	// Primary cursor: last line end
	last := te.Editor.CurrentBuffer().LineCount - 1
	it := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), last)
	te.Editor.CurrentBuffer().CursorIterator = it
	te.Editor.CurrentBuffer().CursorX = len(it.Line())

	// Virtual cursors: ends of previous lines
	te.Editor.virtualCursors = nil
	for y := 0; y < last; y++ {
		it2 := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), y)
		te.Editor.virtualCursors = append(te.Editor.virtualCursors, VirtualCursor{X: len(it2.Line()), Y: y})
	}

	if err := te.ExecuteCommand("calculate-replace"); err != nil {
		t.Fatalf("calculate-replace failed: %v", err)
	}

	// Validate lines
	expected := []string{"1", "2", "3", "4"}
	for i := 0; i < te.Editor.CurrentBuffer().LineCount; i++ {
		it3 := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), i)
		if got := it3.Line(); got != expected[i] {
			t.Fatalf("line %d: expected %q, got %q", i, expected[i], got)
		}
	}

	// Validate primary cursor position (Normal mode): on last digit of last line
	if te.Editor.CurrentBuffer().CursorIterator.LineNumber() != last {
		t.Fatalf("primary cursor Y expected %d, got %d", last, te.Editor.CurrentBuffer().CursorIterator.LineNumber())
	}
	if te.Editor.CurrentBuffer().CursorX != 0 {
		t.Fatalf("primary cursor X expected 0, got %d", te.Editor.CurrentBuffer().CursorX)
	}

	// Validate virtual cursor positions updated to on-digit (Normal mode)
	if len(te.Editor.virtualCursors) != 3 {
		t.Fatalf("expected 3 virtual cursors, got %d", len(te.Editor.virtualCursors))
	}
	for i, vc := range te.Editor.virtualCursors {
		if vc.Y != i { // we appended in Y order 0..2
			t.Fatalf("virtual cursor %d Y expected %d, got %d", i, i, vc.Y)
		}
		if vc.X != 0 { // on single-digit
			t.Fatalf("virtual cursor %d X expected 0, got %d (Y=%d)", i, vc.X, vc.Y)
		}
	}
}

// Same scenario but in Insert mode: cursors should end up AFTER the number.
func TestCalculatorPlusOneSequentialMultiCursor_InsertMode(t *testing.T) {
	content := "1\n$?+1\n$?+1\n$?+1"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().FilePath = "calc_plus1_insert.txt"

	// Enter insert mode to set insert semantics for cursor placement
	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatalf("failed to enter insert mode: %v", err)
	}

	// Primary cursor: last line end (after last char)
	last := te.Editor.CurrentBuffer().LineCount - 1
	it := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), last)
	te.Editor.CurrentBuffer().CursorIterator = it
	te.Editor.CurrentBuffer().CursorX = len(it.Line())

	// Virtual cursors: ends of previous lines
	te.Editor.virtualCursors = nil
	for y := 0; y < last; y++ {
		it2 := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), y)
		te.Editor.virtualCursors = append(te.Editor.virtualCursors, VirtualCursor{X: len(it2.Line()), Y: y})
	}

	if err := te.ExecuteCommand("calculate-replace"); err != nil {
		t.Fatalf("calculate-replace failed: %v", err)
	}

	// Validate lines
	expected := []string{"1", "2", "3", "4"}
	for i := 0; i < te.Editor.CurrentBuffer().LineCount; i++ {
		it3 := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), i)
		if got := it3.Line(); got != expected[i] {
			t.Fatalf("line %d: expected %q, got %q", i, expected[i], got)
		}
	}

	// Validate primary cursor position (Insert mode): after last digit of last line
	if te.Editor.CurrentBuffer().CursorIterator.LineNumber() != last {
		t.Fatalf("primary cursor Y expected %d, got %d", last, te.Editor.CurrentBuffer().CursorIterator.LineNumber())
	}
	if te.Editor.CurrentBuffer().CursorX != 1 { // after single-digit
		t.Fatalf("primary cursor X expected 1, got %d", te.Editor.CurrentBuffer().CursorX)
	}

	// Validate virtual cursor positions updated to after-digit (Insert mode)
	if len(te.Editor.virtualCursors) != 3 {
		t.Fatalf("expected 3 virtual cursors, got %d", len(te.Editor.virtualCursors))
	}
	for i, vc := range te.Editor.virtualCursors {
		if vc.Y != i {
			t.Fatalf("virtual cursor %d Y expected %d, got %d", i, i, vc.Y)
		}
		if vc.X != 1 { // after single-digit
			t.Fatalf("virtual cursor %d X expected 1, got %d (Y=%d)", i, vc.X, vc.Y)
		}
	}
}
