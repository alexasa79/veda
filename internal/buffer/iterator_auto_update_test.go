package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestIteratorAutomaticallySeesBufferChanges(t *testing.T) {
	// Create a buffer with initial content
	content := "Hello World"
	buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)
	
	// Create an iterator at the beginning
	iter := NewBufferIterator(buf)
	
	// Verify initial state
	if iter.Line() != "Hello World" {
		t.Errorf("Expected line 'Hello World', got %q", iter.Line())
	}
	
	// Modify the buffer by inserting text at position 5
	buf.Insert(5, " Beautiful")
	
	// The iterator should automatically see the updated content
	// without needing any refresh or update calls
	if iter.Line() != "Hello Beautiful World" {
		t.Errorf("Expected line 'Hello Beautiful World', got %q", iter.Line())
	}
	
	// Test insertion before current position
	buf.Insert(0, "Oh ")
	if iter.Line() != "Oh Hello Beautiful World" {
		t.Errorf("Expected line 'Oh Hello Beautiful World', got %q", iter.Line())
	}
}

func TestIteratorPositionAfterBufferModification(t *testing.T) {
	// Create multi-line buffer
	content := "Line 1\nLine 2\nLine 3"
	buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)
	
	// Create iterator and move to line 2
	iter := NewBufferIterator(buf)
	iter = iter.Next()
	
	// Verify we're on line 2
	if iter.LineNumber() != 1 || iter.Line() != "Line 2" {
		t.Errorf("Expected to be on line 1 with content 'Line 2', got line %d with %q", 
			iter.LineNumber(), iter.Line())
	}
	
	// Insert text on line 1 (before current line)
	buf.Insert(3, "XXX") // Insert in middle of "Line 1"
	
	// Iterator should still be on line 2 with updated content visible
	if iter.LineNumber() != 1 || iter.Line() != "Line 2" {
		t.Errorf("After insert before, expected line 1 'Line 2', got line %d with %q", 
			iter.LineNumber(), iter.Line())
	}
	
	// Insert newline at end of first line (after "LinXXXe 1\n" which ends at position 10)
	buf.Insert(10, "New Line\n") // Insert before "Line 2"
	
	// With a line-number-based iterator, when lines are inserted before us,
	// our line number stays the same but now points to different content
	// Line 0: "LinXXXe 1"  
	// Line 1: "New Line" (this is where our iterator now points)
	// Line 2: "Line 2" (the original content moved down)
	// Line 3: "Line 3"
	if iter.Line() != "New Line" {
		t.Errorf("After newline insert, expected 'New Line', got %q", iter.Line())
	}
}

func TestIteratorWithMultipleModifications(t *testing.T) {
	// Start with empty buffer
	buf := NewBuffer("", "test.txt", config.IndentSpaces, 4)
	
	// Create iterator
	iter := NewBufferIterator(buf)
	
	// Should be at EOF with empty buffer
	if !iter.IsEOF() {
		t.Error("Expected iterator to be at EOF for empty buffer")
	}
	
	// Add content
	buf.Insert(0, "First line")
	
	// Iterator should no longer be at EOF
	if iter.IsEOF() {
		t.Error("Expected iterator to not be at EOF after insert")
	}
	
	if iter.Line() != "First line" {
		t.Errorf("Expected 'First line', got %q", iter.Line())
	}
	
	// Add a newline and second line
	buf.Insert(10, "\nSecond line")
	
	// Current iterator should still show first line
	if iter.Line() != "First line" {
		t.Errorf("Expected 'First line', got %q", iter.Line())
	}
	
	// Move to next line
	iter2 := iter.Next()
	if iter2.Line() != "Second line" {
		t.Errorf("Expected 'Second line', got %q", iter2.Line())
	}
	
	// Original iterator should still see first line
	if iter.Line() != "First line" {
		t.Errorf("Original iterator changed, expected 'First line', got %q", iter.Line())
	}
}