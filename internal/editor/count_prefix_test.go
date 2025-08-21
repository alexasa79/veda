package editor

import (
	"testing"
	"veda/internal/clipboard"
	"veda/internal/commands"
	"veda/internal/events"
)

// These tests verify that a numeric count prefix applies to dd, cc, and yy.
func TestCountPrefixWithOperators(t *testing.T) {
	t.Run("3dd deletes three lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "l1\nl2\nl3\nl4\nl5")

		// Type "3dd"
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '3'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'd'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'd'})

		// Expect first 3 lines removed
		AssertBufferEquals(t, te, "l4\nl5")
		AssertCursorAt(t, te, 0, 0) // Cursor at beginning of new first line
	})

	t.Run("10dd caps at available lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "a\nb\nc")

		// Type "10dd" (more than available lines)
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '1'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '0'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'd'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'd'})

		// Everything deleted
		AssertBufferEquals(t, te, "")
	})

	t.Run("2cc applies to two lines (enters insert mode)", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "L1\nL2\nL3\nL4")

		// Move to beginning (already at 0,0); apply "2cc"
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '2'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'c'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'c'})

		// Semantics: leaves a single empty line replacing the two changed lines and enters insert mode
		if te.mode != commands.InsertMode {
			t.Fatalf("expected insert mode after 2cc, got %v", te.mode)
		}
		AssertBufferEquals(t, te, "\nL3\nL4")
	})

	t.Run("4yy yanks four lines (caps at buffer end)", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "one\ntwo\nthree")

		// Type "4yy"
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '4'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})

		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("clipboard paste failed: %v", err)
		}
		if content != "one\ntwo\nthree\n" {
			t.Fatalf("unexpected yanked content: %q", content)
		}
		// Buffer unchanged; cursor stays
		AssertBufferEquals(t, te, "one\ntwo\nthree")
		AssertCursorAt(t, te, 0, 0)
	})
}
