package editor

import (
	"testing"
)

func TestFindCharacterMotion(t *testing.T) {
	t.Run("basic find character", func(t *testing.T) {
		// Test with a line containing various characters
		te := NewTestEditorWithContent(80, 24, "hello world test line")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Find 'w' - should move to "world"
		te.TypeKeys("fw")
		AssertCursorAt(t, te, 6, 0)
		
		// Find 's' - should move to "test"
		te.TypeKeys("fs")
		AssertCursorAt(t, te, 14, 0)
		
		// Find 'e' - should move to "line"
		te.TypeKeys("fe")
		AssertCursorAt(t, te, 20, 0)
	})
	
	t.Run("find character not found", func(t *testing.T) {
		// Test with a line not containing the character
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Find 'x' - should not move cursor
		te.TypeKeys("fx")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("find from middle of line", func(t *testing.T) {
		// Test starting from middle of line
		te := NewTestEditorWithContent(80, 24, "abcdefghijk")
		
		// Move to 'd'
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 3, 0)
		
		// Find 'g' - should move forward
		te.TypeKeys("fg")
		AssertCursorAt(t, te, 6, 0)
		
		// Find 'b' - should not move (b is before current position)
		te.TypeKeys("fb")
		AssertCursorAt(t, te, 6, 0)
	})
	
	t.Run("find duplicate characters", func(t *testing.T) {
		// Test with duplicate characters
		te := NewTestEditorWithContent(80, 24, "hello hello hello")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Find first 'e'
		te.TypeKeys("fe")
		AssertCursorAt(t, te, 1, 0)
		
		// Find next 'e'
		te.TypeKeys("fe")
		AssertCursorAt(t, te, 7, 0)
		
		// Find third 'e'
		te.TypeKeys("fe")
		AssertCursorAt(t, te, 13, 0)
	})
}

func TestTillCharacterMotion(t *testing.T) {
	t.Run("basic till character", func(t *testing.T) {
		// Test with a line containing various characters
		te := NewTestEditorWithContent(80, 24, "hello world test line")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Till 'w' - should move to position before "world"
		te.TypeKeys("tw")
		AssertCursorAt(t, te, 5, 0)
		
		// Till 's' - should move to position before "test"
		te.TypeKeys("ts")
		AssertCursorAt(t, te, 13, 0)
	})
	
	t.Run("till character not found", func(t *testing.T) {
		// Test with a line not containing the character
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Till 'x' - should not move cursor
		te.TypeKeys("tx")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("till from middle of line", func(t *testing.T) {
		// Test starting from middle of line
		te := NewTestEditorWithContent(80, 24, "abcdefghijk")
		
		// Move to 'd'
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 3, 0)
		
		// Till 'h' - should move to position before 'h'
		te.TypeKeys("th")
		AssertCursorAt(t, te, 6, 0)
	})
	
	t.Run("till at beginning of line", func(t *testing.T) {
		// Test till when character is at the beginning
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Move to position 2
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 0)
		
		// Till first character should not move (can't move before position 0)
		te.TypeKeys("th")
		AssertCursorAt(t, te, 2, 0)
	})
	
	t.Run("till duplicate characters", func(t *testing.T) {
		// Test with duplicate characters
		te := NewTestEditorWithContent(80, 24, "hello hello hello")
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Till first 'e' - move to position before first 'e'
		te.TypeKeys("te")
		AssertCursorAt(t, te, 0, 0) // Already at position before first 'e'
		
		// Move forward a bit
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 0)
		
		// Till next 'e' - should move to position before second 'e'
		te.TypeKeys("te")
		AssertCursorAt(t, te, 6, 0)
	})
}