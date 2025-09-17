package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
)

// TestRewrapCursorWhitespaceSplit reproduces the case where the cursor sits on
// whitespace that turns into a newline after wrapping. The cursor should remain
// adjacent to the same textual boundary rather than jumping to the beginning of
// the paragraph.
func TestRewrapCursorWhitespaceSplit(t *testing.T) {
	te := NewTestEditorWithContent(120, 40, "alpha beta  gamma delta")
	te.config.VerticalRulers = []int{12}

	line := te.CurrentBuffer().CursorIterator.Line()
	cursorPos := strings.Index(line, " gamma")
	if cursorPos == -1 {
		t.Fatalf("test setup failure: could not locate wrap split in %q", line)
	}
	te.CurrentBuffer().CursorX = cursorPos

	if err := te.Editor.rewrapParagraphWithWidth(12); err != nil {
		t.Fatalf("rewrapParagraphWithWidth failed: %v", err)
	}

	// After wrapping we expect the cursor to remain at the end of the previous
	// word (same logical boundary), not at the start of the wrapped paragraph.
	expectedLine := 0
	expectedX := len("alpha beta")
	actualLine := te.CurrentBuffer().CursorIterator.LineNumber()
	actualX := te.CurrentBuffer().CursorX
	if actualLine != expectedLine || actualX != expectedX {
		t.Fatalf("cursor moved to unexpected location: want line %d col %d, got line %d col %d", expectedLine, expectedX, actualLine, actualX)
	}
}

// TestRewrapCursorTrailingWhitespace reproduces the scenario where the cursor
// starts beyond the final non-space text (on trailing spaces). The cursor should
// land on the end of the wrapped paragraph, not jump to the beginning.
func TestRewrapCursorTrailingWhitespace(t *testing.T) {
	te := NewTestEditorWithContent(120, 40, "alpha beta gamma delta epsilon     ")
	te.config.VerticalRulers = []int{12}

	buf := te.CurrentBuffer()
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 0)
	buf.CursorX = len(buf.CursorIterator.Line())

	if err := te.Editor.rewrapParagraphWithWidth(12); err != nil {
		t.Fatalf("rewrapParagraphWithWidth failed: %v", err)
	}

	expectedLine := 2
	expectedX := len("epsilon")
	actualLine := te.CurrentBuffer().CursorIterator.LineNumber()
	actualX := te.CurrentBuffer().CursorX
	if actualLine != expectedLine || actualX != expectedX {
		t.Fatalf("cursor moved to unexpected location: want line %d col %d, got line %d col %d", expectedLine, expectedX, actualLine, actualX)
	}
}
