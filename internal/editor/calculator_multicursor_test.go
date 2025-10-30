package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/commands"
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

func TestCalculatorMultiCursorAppendCursorPlacement(t *testing.T) {
	content := "1+1\n2+1\n3+1"
	te := NewTestEditorWithContent(80, 24, content)
	te.mode = commands.NormalMode

	buf := te.Editor.CurrentBuffer()

	// Place primary cursor on the final character of line 0 (Normal-mode semantics)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 0)
	buf.CursorX = len("1+1") - 1

	// Add virtual cursors on the other lines, also on the final character
	te.Editor.virtualCursors = []VirtualCursor{
		{X: len("2+1") - 1, Y: 1},
		{X: len("3+1") - 1, Y: 2},
	}
	te.Editor.syncCurrentVirtualCursors()

	if err := te.ExecuteCommand("calculate-append"); err != nil {
		t.Fatalf("calculate-append failed: %v", err)
	}

	// Verify the text results
	expected := []string{"1+1 → 2", "2+1 → 3", "3+1 → 4"}
	for i := 0; i < buf.LineCount; i++ {
		line := buffer.NewBufferIteratorAtLine(buf, i).Line()
		if line != expected[i] {
			t.Fatalf("line %d: expected %q, got %q", i, expected[i], line)
		}
	}

	// Render and verify the virtual cursor highlight appears over the final glyph (column 6)
	te.Editor.render()

	lineNumberWidth := buf.LineNumberWidth
	virtualStyle := te.config.Theme.GetStyle(te.config.Theme.VirtualCursor)
	_, expectedBg, _ := virtualStyle.Decompose()

	column := lineNumberWidth + 6 // zero-based column for the 7th glyph
	screenWidth, _ := te.screen.Size()
	linesToCheck := make([]int, 0, len(te.Editor.virtualCursors))
	for _, vc := range te.Editor.virtualCursors {
		linesToCheck = append(linesToCheck, vc.Y)
	}

	for _, line := range linesToCheck {
		_, style := te.mockScreen.GetCell(column, line)
		_, actualBg, _ := style.Decompose()
		if actualBg != expectedBg {
			t.Errorf("line %d: expected virtual cursor background at column %d, got %v (expected %v)", line, column, actualBg, expectedBg)
		}

		if column+2 < screenWidth {
			_, neighborStyle := te.mockScreen.GetCell(column+2, line)
			_, neighborBg, _ := neighborStyle.Decompose()
			if neighborBg == expectedBg {
				t.Errorf("line %d: unexpected virtual cursor background at column %d", line, column+2)
			}
		}

		if testing.Verbose() {
			for dx := -2; dx <= 4; dx++ {
				target := column + dx
				if target < 0 || target >= screenWidth {
					continue
				}
				_, dbgStyle := te.mockScreen.GetCell(target, line)
				_, dbgBg, _ := dbgStyle.Decompose()
				t.Logf("line %d column %d bg=%v", line, target, dbgBg)
			}
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
