package editor

import (
	"testing"
)

func TestWordNavigation(t *testing.T) {
	t.Run("basic word forward", func(t *testing.T) {
		// Test with simple words separated by spaces
		te := NewTestEditorWithContent(80, 24, "hello world foo bar")
		
		// Start at beginning of "hello"
		AssertCursorAt(t, te, 0, 0)
		
		// Jump to "world"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 6, 0)
		
		// Jump to "foo"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 12, 0)
		
		// Jump to "bar"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 16, 0)
		
		// At last word, should stay at current position
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 16, 0)
	})
	
	t.Run("word forward with punctuation", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello, world! foo-bar")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Jump to "world" (skip comma)
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 7, 0)
		
		// Jump to "foo"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 14, 0)
		
		// Jump to "bar" (after hyphen)
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 18, 0)
	})
	
	t.Run("word forward across lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world\nfoo bar\nbaz")
		
		// Start at "world"
		te.SetCursorPosition(6, 0)
		AssertCursorAt(t, te, 6, 0)
		
		// Jump to "foo" on next line
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 0, 1)
		
		// Jump to "bar"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 4, 1)
		
		// Jump to "baz" on next line
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 0, 2)
	})
	
	t.Run("word forward with multiple spaces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello    world     foo")
		
		// Jump from "hello" to "world" (skip multiple spaces)
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 9, 0)
		
		// Jump to "foo"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 19, 0)
	})
	
	t.Run("word forward from middle of word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world foo")
		
		// Start in middle of "hello"
		te.buffer.CursorX = 2
		AssertCursorAt(t, te, 2, 0)
		
		// Should jump to beginning of "world"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 6, 0)
	})
	
	t.Run("word forward with empty lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello\n\nworld")
		
		// Jump from "hello" should skip empty line and go to "world"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 0, 2)
	})
	
	t.Run("word forward with code", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "func main() { return }")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Jump to "main"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 5, 0)
		
		// Jump to "return"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 14, 0)
	})
	
	t.Run("word forward with underscores", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "foo_bar baz_qux some_long_variable_name")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Jump to "baz_qux" (skipping foo_bar as one word)
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 8, 0)
		
		// Jump to "some_long_variable_name"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 16, 0)
	})
}