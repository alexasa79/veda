package editor

import "testing"

// Ensure backspace deletes one full Unicode rune (like '→') and not just one byte.
func TestBackspaceDeletesSingleUnicodeRune(t *testing.T) {
	// Start with empty buffer and enter insert mode
	te := NewTestEditorWithContent(80, 24, "")
	te.ExecuteCommand("enter-insert-mode")

	// Type: ab→cd (arrow U+2192 between ascii)
	te.InsertChar('a')
	te.InsertChar('b')
	te.InsertChar('→')
	te.InsertChar('c')
	te.InsertChar('d')

	// Place cursor immediately after the arrow
	posAfterArrow := len("ab→") // byte index after the rune
	te.SetCursorPosition(posAfterArrow, 0)

	// Backspace should delete exactly the arrow character
	te.ExecuteCommand("backspace")

	// Expect: abcd (no extra bytes or corruption)
	AssertBufferEquals(t, te, "abcd")
}
