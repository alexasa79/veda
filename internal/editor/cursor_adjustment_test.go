package editor

import (
	"testing"
)

func TestCursorAdjustmentOnShorterLines(t *testing.T) {
	// Test the specific scenario mentioned: cursor on last '1' moving to last '2'
	te := NewTestEditorWithContent(80, 24, "11111\n222")
	
	// Move cursor to the last '1' (position 4)
	te.ExecuteCommand("right")
	te.ExecuteCommand("right")
	te.ExecuteCommand("right")
	te.ExecuteCommand("right")
	AssertCursorAt(t, te, 4, 0)
	
	// Move down - cursor should adjust to end of shorter line
	te.ExecuteCommand("down")
	AssertCursorAt(t, te, 2, 1) // Now correctly at the last '2'
	
	// Test with even more extreme case
	te2 := NewTestEditorWithContent(80, 24, "This is a very long line\nx\nAnother long line")
	
	// Move to near end of first line
	te2.cursorX = 20
	AssertCursorAt(t, te2, 20, 0)
	
	// Move down to single character line
	te2.ExecuteCommand("down")
	AssertCursorAt(t, te2, 0, 1) // Should be at the 'x'
	
	// Move down again
	te2.ExecuteCommand("down")
	AssertCursorAt(t, te2, 0, 2) // Should be at start of "Another long line"
	
	// Test with empty line
	te3 := NewTestEditorWithContent(80, 24, "Some text\n\nMore text")
	te3.cursorX = 5
	
	// Move down to empty line
	te3.ExecuteCommand("down")
	AssertCursorAt(t, te3, 0, 1) // Should be at position 0 on empty line
}