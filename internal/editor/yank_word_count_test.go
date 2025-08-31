package editor

import "testing"

func TestYankWordCountWithOperatorPrefix(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "one two three four")
	// Move cursor to beginning
	te.SetCursorPosition(0, 0)

	// Yank two words into register 'a': 2 ' a y w
	te.TypeKeys("2")
	te.TypeKeys("'")
	te.TypeKeys("a")
	te.TypeKeys("y")
	te.TypeKeys("w")

	// Delete to clear, then paste from register 'a'
	te.TypeKeys("d")
	te.TypeKeys("A")
	te.TypeKeys("'")
	te.TypeKeys("a")
	te.TypeKeys("p")

	// For our 'w' semantics, yanking a word includes trailing space; for two words, expect "one two "
	AssertBufferEquals(t, te, "one two ")
}
