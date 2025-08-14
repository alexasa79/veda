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

func TestWordNavigationWithoutUnderscore(t *testing.T) {
	t.Run("W command treats underscores as word boundaries", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello_world foo_bar")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// W should jump to underscore (treating it as boundary)
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 5, 0) // At '_'
		
		// Next W should jump to "world"
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 6, 0) // At 'w' in "world"
		
		// Next W should jump to "foo" (skipping whitespace like normal word navigation)
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 12, 0) // At 'f' in "foo"
		
		// Next W should jump to underscore in "foo_bar"
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 15, 0) // At '_' in "foo_bar"
	})
	
	t.Run("w command includes underscores in words", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello_world foo_bar")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// w should jump over entire "hello_world" to "foo_bar"
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 12, 0) // At 'f' in "foo_bar"
		
		// Next w should stay at same position (last word)
		te.ExecuteCommand("word-forward")
		AssertCursorAt(t, te, 18, 0) // At end of "foo_bar"
	})
	
	t.Run("B command treats underscores as word boundaries", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello_world foo_bar")
		
		// Start at end of line  
		te.SetCursorPosition(18, 0) // End of "foo_bar"
		AssertCursorAt(t, te, 18, 0)
		
		// B should jump to "bar" (start of current word)
		te.ExecuteCommand("word-backward-no-underscore")
		AssertCursorAt(t, te, 16, 0) // At 'b' in "bar"
		
		// Next B should jump to underscore before "bar"
		te.ExecuteCommand("word-backward-no-underscore")
		AssertCursorAt(t, te, 15, 0) // At '_' in "foo_bar"
		
		// Next B should jump to "foo"
		te.ExecuteCommand("word-backward-no-underscore")
		AssertCursorAt(t, te, 12, 0) // At 'f' in "foo"
		
		// Next B should jump to "world"
		te.ExecuteCommand("word-backward-no-underscore")
		AssertCursorAt(t, te, 6, 0) // At 'w' in "world"
		
		// Next B should jump to underscore in "hello_world"
		te.ExecuteCommand("word-backward-no-underscore")
		AssertCursorAt(t, te, 5, 0) // At '_' in "hello_world"
		
		// Next B should jump to "hello"
		te.ExecuteCommand("word-backward-no-underscore")
		AssertCursorAt(t, te, 0, 0) // At 'h' in "hello"
	})
	
	t.Run("W and B with mixed punctuation and underscores", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "func_name(arg_value)")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// W should jump to underscore
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 4, 0) // At '_'
		
		// W should jump to "name"
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 5, 0) // At 'n' in "name"
		
		// W should jump to "("
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 9, 0) // At '('
		
		// W should jump to "arg"
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 10, 0) // At 'a' in "arg"
		
		// W should jump to underscore in "arg_value"
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 13, 0) // At '_'
		
		// W should jump to "value"
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 14, 0) // At 'v' in "value"
	})
	
	t.Run("W command on single word with underscores", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello_world_test")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// W should jump to first underscore
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 5, 0) // At first '_'
		
		// W should jump to "world"
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 6, 0) // At 'w' in "world"
		
		// W should jump to second underscore
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 11, 0) // At second '_'
		
		// W should jump to "test"
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 12, 0) // At 't' in "test"
		
		// W should jump to end of line
		te.ExecuteCommand("word-forward-no-underscore")
		AssertCursorAt(t, te, 15, 0) // At end of "test"
	})
}