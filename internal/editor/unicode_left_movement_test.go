package editor

import (
	"testing"
	"unicode/utf8"
)

func logUnicodeLeftCursorState(t *testing.T, te *TestEditor, label string) {
	t.Helper()

	line := te.CurrentBuffer().CursorIterator.Line()
	x := te.CurrentBuffer().CursorX
	_, visualCol := te.GetCursorVisualPosition()

	atRuneBoundary := false
	if x == len(line) {
		atRuneBoundary = true
	} else if x >= 0 && x < len(line) {
		atRuneBoundary = utf8.RuneStart(line[x])
	}

	cursorRune := "(eol)"
	if x >= 0 && x < len(line) {
		r, _ := utf8.DecodeRuneInString(line[x:])
		cursorRune = string(r)
	}

	t.Logf("%s: cursorX=%d visualCol=%d atRuneBoundary=%v rune=%q line=%q",
		label, x, visualCol, atRuneBoundary, cursorRune, line)
}

func TestMoveLeftAcrossUnicodeArrow(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "a→b")

	// Start on 'b' in normal mode.
	te.SetCursorPosition(len("a→"), 0)
	logUnicodeLeftCursorState(t, te, "start")

	// First left should land on '→' (byte index 1), not inside its UTF-8 bytes.
	te.ExecuteCommand("left")
	logUnicodeLeftCursorState(t, te, "after first left")
	if te.CurrentBuffer().CursorX != len("a") {
		t.Fatalf("first left should land on '→' at byte %d, got %d", len("a"), te.CurrentBuffer().CursorX)
	}

	// Second left should land on 'a' (byte index 0).
	te.ExecuteCommand("left")
	logUnicodeLeftCursorState(t, te, "after second left")
	if te.CurrentBuffer().CursorX != 0 {
		t.Fatalf("second left should land on 'a' at byte 0, got %d", te.CurrentBuffer().CursorX)
	}
}

func TestMoveRightAcrossUnicodeArrow(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "a→b")

	logUnicodeLeftCursorState(t, te, "start")
	if te.CurrentBuffer().CursorX != 0 {
		t.Fatalf("expected initial cursor at byte 0, got %d", te.CurrentBuffer().CursorX)
	}

	te.ExecuteCommand("right")
	logUnicodeLeftCursorState(t, te, "after first right")
	if te.CurrentBuffer().CursorX != len("a") {
		t.Fatalf("first right should land on '→' at byte %d, got %d", len("a"), te.CurrentBuffer().CursorX)
	}

	te.ExecuteCommand("right")
	logUnicodeLeftCursorState(t, te, "after second right")
	if te.CurrentBuffer().CursorX != len("a→") {
		t.Fatalf("second right should land on 'b' at byte %d, got %d", len("a→"), te.CurrentBuffer().CursorX)
	}

	te.ExecuteCommand("right")
	logUnicodeLeftCursorState(t, te, "after third right")
	if te.CurrentBuffer().CursorX != len("a→") {
		t.Fatalf("third right should stay on 'b' at byte %d, got %d", len("a→"), te.CurrentBuffer().CursorX)
	}
}

func TestInsertModeArrowKeyMovementAcrossUnicodeArrow(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "a→b")
	te.ExecuteCommand("enter-insert-mode")

	te.SetCursorPosition(0, 0)
	logUnicodeLeftCursorState(t, te, "insert start")

	te.ExecuteCommand("right-insert")
	logUnicodeLeftCursorState(t, te, "insert after first right")
	if te.CurrentBuffer().CursorX != len("a") {
		t.Fatalf("insert first right should move to byte %d, got %d", len("a"), te.CurrentBuffer().CursorX)
	}

	te.ExecuteCommand("right-insert")
	logUnicodeLeftCursorState(t, te, "insert after second right")
	if te.CurrentBuffer().CursorX != len("a→") {
		t.Fatalf("insert second right should move to byte %d, got %d", len("a→"), te.CurrentBuffer().CursorX)
	}

	te.ExecuteCommand("right-insert")
	logUnicodeLeftCursorState(t, te, "insert after third right")
	if te.CurrentBuffer().CursorX != len("a→b") {
		t.Fatalf("insert third right should move to EOL byte %d, got %d", len("a→b"), te.CurrentBuffer().CursorX)
	}

	te.ExecuteCommand("left-insert")
	logUnicodeLeftCursorState(t, te, "insert after first left")
	if te.CurrentBuffer().CursorX != len("a→") {
		t.Fatalf("insert first left should move to byte %d, got %d", len("a→"), te.CurrentBuffer().CursorX)
	}

	te.ExecuteCommand("left-insert")
	logUnicodeLeftCursorState(t, te, "insert after second left")
	if te.CurrentBuffer().CursorX != len("a") {
		t.Fatalf("insert second left should move to byte %d, got %d", len("a"), te.CurrentBuffer().CursorX)
	}
}
