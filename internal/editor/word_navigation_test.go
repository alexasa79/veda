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
		
		// At last word, should go to end of "bar"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 18, 0)
	})
	
	t.Run("word forward with punctuation", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello, world! foo-bar")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Jump to comma (punctuation is a word)
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 5, 0)
		
		// Jump to "world"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 7, 0)
		
		// Jump to "!"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 12, 0)
		
		// Jump to "foo"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 14, 0)
		
		// Jump to "-"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 17, 0)
		
		// Jump to "bar"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 18, 0)
	})
	
	t.Run("word forward across lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world\nfoo bar\nbaz")
		
		// Start at "world"
		te.SetCursorPosition(6, 0)
		AssertCursorAt(t, te, 6, 0)
		
		// First jump should go to end of line (end of "world")
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 10, 0) // End of "world"
		
		// Next jump should go to "foo" on next line
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 0, 1)
		
		// Jump to "bar"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 4, 1)
		
		// Jump to end of "bar"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 6, 1) // End of "bar"
		
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
	
	t.Run("word forward on single line with two words", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Start at beginning of line (position 0)
		te.SetCursorPosition(0, 0)
		AssertCursorAt(t, te, 0, 0)
		
		// First 'w' should jump to beginning of "world" 
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 6, 0)
		
		// Second 'w' should go to end of "world" since it's the last word
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 10, 0)
	})
	
	t.Run("word forward on single word line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello")
		
		// Start at beginning of line (position 0)
		te.SetCursorPosition(0, 0)
		AssertCursorAt(t, te, 0, 0)
		
		// 'w' on single word should move to end of word (last character)
		// This is the vim behavior when there's nowhere else to go
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 4, 0) // Should be at 'o' (last character)
		
		// Another 'w' should stay at the same position
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 4, 0)
	})
	
	t.Run("word forward with code", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "func main() { return }")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Jump to "main"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 5, 0)
		
		// Jump to "("
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 9, 0)
		
		// Jump to ")"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 10, 0)
		
		// Jump to "{"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 12, 0)
		
		// Jump to "return"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 14, 0)
		
		// Jump to "}"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 21, 0)
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
	
	t.Run("word forward with punctuation like vim", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, `Enable: "false", "second"`)
		
		// Start at 'E' in Enable
		AssertCursorAt(t, te, 0, 0)
		
		// Move through each word/punctuation
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 6, 0) // Start of ":"
		
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 8, 0) // Start of first quote
		
		te.ExecuteCommand("word-forward") 
		AssertCursorAt(t, te, 9, 0) // Start of "false"
		
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 14, 0) // Start of closing quote
		
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 15, 0) // Start of comma
		
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 17, 0) // Start of second quote
		
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 18, 0) // Start of "second"
		
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 24, 0) // Start of closing quote
	})
	
	t.Run("word forward stops at end of line like user requested", func(t *testing.T) {
		// Test the exact scenario described by the user
		te := NewTestEditorWithContent(80, 24, "hello1\nhello22")
		
		// Start at beginning of "hello1"
		AssertCursorAt(t, te, 0, 0)
		
		// Press "w" - should jump to end of "hello1" (the "1")
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 5, 0) // Position of "1"
		
		// Press "w" again - should jump to "h" in second line
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 0, 1) // Position of "h" in "hello22"
		
		// Press "w" again - should stop at the last "2"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 6, 1) // Position of last "2"
	})
}