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

func TestInfixCountWithOperators(t *testing.T) {
	t.Run("d3d deletes three lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "l1\nl2\nl3\nl4\nl5")

		te.TypeKeys("d3d")

		AssertBufferEquals(t, te, "l4\nl5")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("d12d supports multi-digit counts", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13")

		te.TypeKeys("d12d")

		AssertBufferEquals(t, te, "13")
	})

	t.Run("c3c changes three lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "l1\nl2\nl3\nl4\nl5")

		te.TypeKeys("c3c")

		if te.mode != commands.InsertMode {
			t.Fatalf("expected insert mode after c3c, got %v", te.mode)
		}
		AssertBufferEquals(t, te, "\nl4\nl5")
	})

	t.Run("y3y yanks three lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "l1\nl2\nl3\nl4\nl5")

		te.TypeKeys("'ay3y")

		if got, want := te.registers['a'], "l1\nl2\nl3\n"; got != want {
			t.Fatalf("register a = %q, want %q", got, want)
		}
	})

	t.Run("d3w deletes three words", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "one two three four five")

		te.TypeKeys("d3w")

		AssertBufferEquals(t, te, "four five")
	})

	t.Run("c2W changes through two words", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "one two three")

		te.TypeKeys("c2W")

		if te.mode != commands.InsertMode {
			t.Fatalf("expected insert mode after c2W, got %v", te.mode)
		}
		AssertBufferEquals(t, te, " three")
	})

	t.Run("y2w yanks two words", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "one two three")

		te.TypeKeys("'ay2w")

		if got, want := te.registers['a'], "one two "; got != want {
			t.Fatalf("register a = %q, want %q", got, want)
		}
	})

	t.Run("d2aw deletes two words around", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "one two three")

		te.TypeKeys("d2aw")

		AssertBufferEquals(t, te, "three")
	})

	t.Run("leading and infix counts multiply", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "1\n2\n3\n4\n5\n6\n7")

		te.TypeKeys("2d3d")

		AssertBufferEquals(t, te, "7")
	})

	t.Run("infix count is preserved when repeating", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "one two three four five six seven")

		te.TypeKeys("d3w")
		te.TypeKeys(".")

		AssertBufferEquals(t, te, "seven")
	})

	t.Run("combined count is not applied twice when repeating", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "1 2 3 4 5 6 7 8 9 10 11 12 13")

		te.TypeKeys("2d3w")
		te.TypeKeys(".")

		AssertBufferEquals(t, te, "13")
	})

	t.Run("infix count does not leak into undo", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "1\n2\n3\n4\n5\n6\n7")

		te.TypeKeys("dd")
		te.TypeKeys("d3d")
		te.TypeKeys("u")

		AssertBufferEquals(t, te, "2\n3\n4\n5\n6\n7")
	})

	t.Run("escape clears infix count", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "abcdef")

		te.TypeKeys("d3")
		te.PressKey(events.KeyEscape, 0)
		te.TypeKeys("x")

		AssertBufferEquals(t, te, "bcdef")
	})
}
