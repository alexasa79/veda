package editor

import (
	"testing"
)

func TestWordBackward(t *testing.T) {
	t.Run("basic word backward", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world foo bar")
		
		// Start at beginning of "bar"
		te.SetCursorPosition(16, 0)
		AssertCursorAt(t, te, 16, 0)
		
		// Jump back to "foo"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 12, 0)
		
		// Jump back to "world"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 6, 0)
		
		// Jump back to "hello"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 0)
		
		// At first word, should stay at current position
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("word backward with punctuation", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello, world! foo-bar")
		
		// Start at "bar"
		te.buffer.CursorX = 18
		AssertCursorAt(t, te, 18, 0)
		
		// Jump back to "foo"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 14, 0)
		
		// Jump back to "world"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 7, 0)
		
		// Jump back to "hello"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("word backward across lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world\nfoo bar\nbaz")
		
		// Start at "baz"
		te.SetCursorPosition(0, 2)
		AssertCursorAt(t, te, 0, 2)
		
		// Jump back to "bar" on previous line
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 4, 1)
		
		// Jump back to "foo"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 1)
		
		// Jump back to "world" on previous line
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 6, 0)
	})
	
	t.Run("word backward with multiple spaces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello    world     foo")
		
		// Start at "foo"
		te.buffer.CursorX = 19
		AssertCursorAt(t, te, 19, 0)
		
		// Jump back to "world" (skip multiple spaces)
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 9, 0)
		
		// Jump back to "hello"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("word backward from middle of word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world foo")
		
		// Start in middle of "world"
		te.buffer.CursorX = 8
		AssertCursorAt(t, te, 8, 0)
		
		// Should jump to beginning of "world"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 6, 0)
		
		// Now jump to beginning of "hello"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("word backward with empty lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello\n\nworld")
		
		// Start at "world"
		te.SetCursorPosition(0, 2)
		AssertCursorAt(t, te, 0, 2)
		
		// Jump back to "hello" (skip empty line)
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("word backward with underscores", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "foo_bar baz_qux some_long_variable_name")
		
		// Start at beginning of "some_long_variable_name"
		te.buffer.CursorX = 16
		AssertCursorAt(t, te, 16, 0)
		
		// Jump back to "baz_qux" (treating it as one word)
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 8, 0)
		
		// Jump back to "foo_bar"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("word backward from non-word character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello , world")
		
		// Start on the comma
		te.buffer.CursorX = 6
		AssertCursorAt(t, te, 6, 0)
		
		// Should jump back to "hello"
		te.ExecuteCommand("word-backward")
		AssertCursorAt(t, te, 0, 0)
	})
}