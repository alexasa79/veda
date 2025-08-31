package editor

import "testing"

// Verify that count works with W (from cursor to end of word) for yank with register
func TestYankWWordCountWithOperatorPrefix(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello_world test more")
	// Place cursor in the middle of the first word: hel|lo_world
	te.SetCursorPosition(3, 0)

	// Yank two W-words into register 'a': 2 ' a y W
	te.TypeKeys("2")
	te.TypeKeys("'")
	te.TypeKeys("a")
	te.TypeKeys("y")
	te.TypeKeys("W")

	// Delete buffer to make verification simple, then paste from register 'a'
	te.TypeKeys("d")
	te.TypeKeys("A")
	te.TypeKeys("'")
	te.TypeKeys("a")
	te.TypeKeys("p")

	// Expect from cursor to end of current word, plus the next full word (without trailing space)
	// From hel|lo_world test more -> "lo_world test"
	AssertBufferEquals(t, te, "lo_world test")
}
