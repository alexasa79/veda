package editor

import (
	"testing"
	"veda/internal/buffer"
)

// Verifies sequential calculate-replace across multiple cursors updates $? between lines.
func TestCalculatorSequentialWithLastResultMultiCursor(t *testing.T) {
	content := "1\n$?+1\n$?+2\n$?+3"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().FilePath = "calc_seq.txt"

	// Set primary cursor on the last line at end
	last := te.Editor.CurrentBuffer().LineCount - 1
	it := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), last)
	te.Editor.CurrentBuffer().CursorIterator = it
	te.Editor.CurrentBuffer().CursorX = len(it.Line())

	// Add virtual cursors at end of the preceding lines
	te.Editor.virtualCursors = []VirtualCursor{}
	for y := 0; y < last; y++ {
		it2 := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), y)
		te.Editor.virtualCursors = append(te.Editor.virtualCursors, VirtualCursor{X: len(it2.Line()), Y: y})
	}

	// Execute calculate-replace once; should process all cursors top-to-bottom
	if err := te.ExecuteCommand("calculate-replace"); err != nil {
		t.Fatalf("calculate-replace failed: %v", err)
	}

	// Validate that each line has been replaced by 1,2,4,7 respectively
	// because $? is updated after each line (sequential semantics):
	// line1: 1 -> $?=1; line2: $?+1 -> 2 -> $?=2; line3: $?+2 -> 4 -> $?=4; line4: $?+3 -> 7
	expected := []string{"1", "2", "4", "7"}
	for i := 0; i < te.Editor.CurrentBuffer().LineCount; i++ {
		it3 := buffer.NewBufferIteratorAtLine(te.Editor.CurrentBuffer(), i)
		if got := it3.Line(); got != expected[i] {
			t.Fatalf("line %d: expected %q, got %q", i, expected[i], got)
		}
	}
}
