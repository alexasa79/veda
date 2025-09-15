package editor

import "testing"

// These tests ensure forward delete operations remove a whole UTF-8 rune
// (like '→' or '✚') rather than a single byte, mirroring our Backspace test.

func TestDeleteRuneUnderCursor_NormalMode(t *testing.T) {
	t.Run("delete arrow rune with x", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "a→b")
		// Place cursor on the rune (normal mode 'x' deletes at cursor)
		te.SetCursorPosition(len("a"), 0)
		// Normal mode command to delete current character
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "ab")
		AssertCursorAt(t, te, len("a"), 0)
	})

	t.Run("delete plus rune with x", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "a✚b")
		te.SetCursorPosition(len("a"), 0)
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "ab")
		AssertCursorAt(t, te, len("a"), 0)
	})
}

func TestDeleteRuneUnderCursor_InsertMode(t *testing.T) {
	t.Run("delete arrow rune with Delete key", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "a→b")
		te.ExecuteCommand("enter-insert-mode")
		// Place cursor immediately before the rune so Delete removes it
		te.SetCursorPosition(len("a"), 0)
		te.ExecuteCommand("delete-character-insert")
		AssertBufferEquals(t, te, "ab")
		AssertCursorAt(t, te, len("a"), 0)
	})

	t.Run("delete plus rune with Delete key", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "a✚b")
		te.ExecuteCommand("enter-insert-mode")
		te.SetCursorPosition(len("a"), 0)
		te.ExecuteCommand("delete-character-insert")
		AssertBufferEquals(t, te, "ab")
		AssertCursorAt(t, te, len("a"), 0)
	})
}
