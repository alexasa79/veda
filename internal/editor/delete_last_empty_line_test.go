package editor

import (
	"testing"
)

// Test that deleting the last (empty) line works when the buffer ends with a newline.
// Input buffer has two logical lines: "" and "" (content "\n"). On dd at the last line
// we should remove the trailing newline, leaving a single empty line (empty buffer content).
func TestDD_DeleteLastEmptyLine_TwoEmptyLines(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "\n")

	// Move cursor to the last line (line index 1), column 0
	te.SetCursorPosition(0, 1)

	// Delete the last empty line
	te.TypeKeys("dd")

	// Expect buffer content to be empty (single logical empty line, no trailing newline)
	got := te.GetBufferContent()
	want := ""
	if got != want {
		t.Fatalf("unexpected content after deleting last empty line.\nGot: %q\nWant: %q", got, want)
	}

	// Still one logical line remains
	if te.CurrentBuffer().LineCount != 1 {
		t.Fatalf("expected LineCount=1, got %d", te.CurrentBuffer().LineCount)
	}
}

// Test that deleting a trailing empty line after content removes the trailing newline.
func TestDD_DeleteTrailingEmptyLine_AfterContent(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello\n")

	// Cursor to the trailing empty line
	te.SetCursorPosition(0, 1)

	te.TypeKeys("dd")

	got := te.GetBufferContent()
	want := "hello"
	if got != want {
		t.Fatalf("unexpected content after deleting trailing empty line.\nGot: %q\nWant: %q", got, want)
	}

	if te.CurrentBuffer().LineCount != 1 {
		t.Fatalf("expected LineCount=1, got %d", te.CurrentBuffer().LineCount)
	}
}
