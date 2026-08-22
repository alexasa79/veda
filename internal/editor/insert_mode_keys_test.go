package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

// TestDeleteKeyInInsert tests the Delete key functionality in insert mode
func TestDeleteKeyInInsert(t *testing.T) {
	t.Run("delete character under cursor", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Move cursor to 'w' in "world"
		te.SetCursorPosition(6, 0)

		// Press delete key
		te.ExecuteCommand("delete-character-insert")

		// Should have deleted 'w', leaving "hello orld"
		AssertBufferEquals(t, te, "hello orld")
		AssertCursorAt(t, te, 6, 0) // Cursor should stay at same position
	})

	t.Run("delete at end of line joins lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello\nworld")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Move cursor to end of first line
		te.SetCursorPosition(5, 0)

		// Press delete key
		te.ExecuteCommand("delete-character-insert")

		// Should join lines
		AssertBufferEquals(t, te, "helloworld")
		AssertCursorAt(t, te, 5, 0)
	})

	t.Run("delete at end of buffer does nothing", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Move cursor to end of buffer
		te.SetCursorPosition(5, 0)

		// Press delete key
		te.ExecuteCommand("delete-character-insert")

		// Should not change anything
		AssertBufferEquals(t, te, "hello")
		AssertCursorAt(t, te, 5, 0)
	})
}

func TestCtrlKDeleteToEndOfLineInInsertMode(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello world")

	te.ExecuteCommand("enter-insert-mode")
	te.SetCursorPosition(6, 0) // on 'w'

	te.ExecuteCommand("delete-to-end-of-line")

	AssertBufferEquals(t, te, "hello ")
	AssertCursorAt(t, te, 6, 0)
}

func TestPageKeysInInsertMode(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, strings.Repeat("line\n", 80)+"line")
	te.ExecuteCommand("enter-insert-mode")

	te.handleEvent(events.NewKeyEvent(events.KeyPgDn, 0, events.ModNone))

	lineAfterPageDown := te.CurrentBuffer().CursorIterator.LineNumber()
	if lineAfterPageDown == 0 {
		t.Fatalf("expected PageDown to move cursor in insert mode")
	}
	if te.mode != commands.InsertMode {
		t.Fatalf("expected PageDown to keep insert mode, got %v", te.mode)
	}

	te.handleEvent(events.NewKeyEvent(events.KeyPgUp, 0, events.ModNone))

	if te.CurrentBuffer().CursorIterator.LineNumber() >= lineAfterPageDown {
		t.Fatalf("expected PageUp to move cursor up in insert mode")
	}
	if te.mode != commands.InsertMode {
		t.Fatalf("expected PageUp to keep insert mode, got %v", te.mode)
	}
}

// TestAltArrowKeysInInsert tests Alt+Left/Right word navigation in insert mode
func TestAltArrowKeysInInsert(t *testing.T) {
	t.Run("alt+right moves forward by word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world foo bar")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Start at beginning
		te.SetCursorPosition(0, 0)

		// Alt+Right to "world"
		te.ExecuteCommand("word-forward-insert")
		AssertCursorAt(t, te, 6, 0)

		// Alt+Right to "foo"
		te.ExecuteCommand("word-forward-insert")
		AssertCursorAt(t, te, 12, 0)

		// Alt+Right to "bar"
		te.ExecuteCommand("word-forward-insert")
		AssertCursorAt(t, te, 16, 0)
	})

	t.Run("alt+right moves past final character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")
		te.ExecuteCommand("enter-insert-mode")
		te.SetCursorPosition(6, 0)

		te.PressKey(events.KeyRight, events.ModAlt)

		AssertCursorAt(t, te, 11, 0)
	})

	t.Run("alt+left moves backward by word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world foo bar")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Start at end
		te.SetCursorPosition(19, 0)

		// Alt+Left to start of "bar"
		te.ExecuteCommand("word-backward-insert")
		AssertCursorAt(t, te, 16, 0)

		// Alt+Left to start of "foo"
		te.ExecuteCommand("word-backward-insert")
		AssertCursorAt(t, te, 12, 0)

		// Alt+Left to start of "world"
		te.ExecuteCommand("word-backward-insert")
		AssertCursorAt(t, te, 6, 0)

		// Alt+Left to start of "hello"
		te.ExecuteCommand("word-backward-insert")
		AssertCursorAt(t, te, 0, 0)
	})
}

// TestAltDeleteBackspaceInInsert tests Alt+Delete/Backspace word deletion
func TestAltDeleteBackspaceInInsert(t *testing.T) {
	t.Run("alt+delete deletes word forward", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world foo bar")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Position at start of "world"
		te.SetCursorPosition(6, 0)

		// Alt+Delete should delete "world" and the space after it
		te.ExecuteCommand("delete-word-forward")

		// Should have "hello foo bar"
		AssertBufferEquals(t, te, "hello foo bar")
		AssertCursorAt(t, te, 6, 0)
	})

	t.Run("alt+backspace deletes word backward", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world foo bar")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Position at end of "world"
		te.SetCursorPosition(11, 0)

		// Alt+Backspace should delete "world"
		te.ExecuteCommand("delete-word-backward")

		// Should have "hello  foo bar"
		AssertBufferEquals(t, te, "hello  foo bar")
		AssertCursorAt(t, te, 6, 0)
	})

	t.Run("alt+delete at middle of word deletes to word end", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Position in middle of "world" (at 'r')
		te.SetCursorPosition(8, 0)

		// Alt+Delete should delete "rld"
		te.ExecuteCommand("delete-word-forward")

		AssertBufferEquals(t, te, "hello wo")
		AssertCursorAt(t, te, 8, 0)
	})

	t.Run("alt+backspace at middle of word deletes to word start", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Position in middle of "world" (at 'r')
		te.SetCursorPosition(8, 0)

		// Alt+Backspace should delete "wo"
		te.ExecuteCommand("delete-word-backward")

		AssertBufferEquals(t, te, "hello rld")
		AssertCursorAt(t, te, 6, 0)
	})

	t.Run("alt+delete across line boundary", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello\nworld")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Position at end of first line
		te.SetCursorPosition(5, 0)

		// Alt+Delete should delete newline and stop at "world"
		te.ExecuteCommand("delete-word-forward")

		AssertBufferEquals(t, te, "helloworld")
		AssertCursorAt(t, te, 5, 0)
	})

	t.Run("alt+backspace across line boundary", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello\nworld")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")

		// Position at beginning of second line
		te.SetCursorPosition(0, 1)

		// Alt+Backspace should delete back including "hello" and newline
		te.ExecuteCommand("delete-word-backward")

		AssertBufferEquals(t, te, "world")
		AssertCursorAt(t, te, 0, 0)
	})
}

// TestInsertModeStatePreservation tests that insert mode is maintained during operations
func TestInsertModeStatePreservation(t *testing.T) {
	t.Run("mode stays in insert after delete operations", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")

		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		if te.Editor.mode != commands.InsertMode {
			t.Fatal("Should be in insert mode")
		}

		// Delete character
		te.ExecuteCommand("delete-character-insert")
		if te.Editor.mode != commands.InsertMode {
			t.Fatal("Should still be in insert mode after delete")
		}

		// Alt+Delete
		te.ExecuteCommand("delete-word-forward")
		if te.Editor.mode != commands.InsertMode {
			t.Fatal("Should still be in insert mode after alt+delete")
		}

		// Alt+Backspace
		te.ExecuteCommand("delete-word-backward")
		if te.Editor.mode != commands.InsertMode {
			t.Fatal("Should still be in insert mode after alt+backspace")
		}
	})
}
