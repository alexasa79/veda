package editor

import (
	"strings"
	"testing"
)

// Verifies that typing '{' on a continuation-indented line dedents to L1.
func TestOpeningBraceAdjustsFromContinuationIndentToBlockLevel(t *testing.T) {
	// Construct a multi-line if condition with continuation indentation (9 spaces)
	content := "if (argc == 3\n" +
		"         && !strcmp(argv[1], \"hello\")\n" +
		"         && !strcmp(argv[2], \"world\"))"

	te := NewTestEditorWithContent(80, 24, content)

	// Place cursor at end of the third line
	te.CurrentBuffer().CursorIterator = te.CurrentBuffer().CursorIterator.Next().Next()
	te.CurrentBuffer().CursorX = len(te.CurrentBuffer().CursorIterator.Line())

	// Enter insert mode, press Enter to get the continuation-indent line
	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatalf("enter-insert-mode failed: %v", err)
	}
	if err := te.ExecuteCommand("insert-newline"); err != nil {
		t.Fatalf("insert-newline failed: %v", err)
	}

	// Now type '{' – should auto-adjust indentation back to block level (L1)
	if err := te.InsertChar('{'); err != nil {
		t.Fatalf("insert '{' failed: %v", err)
	}

	// Expected: the new line begins with one indent level (4 spaces) then '{'
	// In this scenario the header line ("if ...") is at L0, so the brace
	// should align with that header level (L0), not one deeper.
	expected := "if (argc == 3\n" +
		"         && !strcmp(argv[1], \"hello\")\n" +
		"         && !strcmp(argv[2], \"world\"))\n" +
		"{"

	actual := te.CurrentBuffer().Rope.String()
	if actual != expected {
		t.Errorf("content mismatch after '{' adjustment\nexpected:\n%q\nactual:\n%q", expected, actual)
	}
}

func TestOpeningBraceAdjustsAfterOpenLineBelow(t *testing.T) {
	content := "int main()\n" +
		"{\n" +
		"    if (argc == 3 \n" +
		"        && !strcmp(argv[1], \"hello\")\n" +
		"        && !strcmp(argv[2], \"world\"))\n" +
		"\n" +
		"    return 0;\n" +
		"}\n"

	te := NewTestEditorWithContent(80, 24, content)
	// Move cursor to the continuation line (the one ending with ")") before the blank line
	te.CurrentBuffer().CursorIterator = te.CurrentBuffer().CursorIterator.Next().Next().Next().Next() // to the ")" line
	te.CurrentBuffer().CursorX = len(te.CurrentBuffer().CursorIterator.Line())

	// Normal mode: open a new line below (this enters insert mode and indents to L2)
	if err := te.ExecuteCommand("open-line-below"); err != nil {
		t.Fatalf("open-line-below failed: %v", err)
	}

	// Type '{' – should dedent to L1 and insert the brace
	if err := te.InsertChar('{'); err != nil {
		t.Fatalf("insert '{' failed: %v", err)
	}

	expectedSuffix := "        && !strcmp(argv[2], \"world\"))\n" +
		"    {\n"
	got := te.CurrentBuffer().Rope.String()
	if !strings.Contains(got, expectedSuffix) {
		t.Errorf("expected content to contain suffix:\n%q\nactual:\n%q", expectedSuffix, got)
	}
}
