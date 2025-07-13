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
	
	// Test 1: Modification after iterator position
	// Create iterator at beginning
	iter1 := NewBufferIterator(buf)
	
	// Verify we're on line 1
	if iter1.LineNumber() != 0 || iter1.Line() != "Line 1" {
		t.Errorf("Expected to be on line 0 with content 'Line 1', got line %d with %q", 
			iter1.LineNumber(), iter1.Line())
	}
	
	// Insert text after the current line (after "Line 1\n")
	buf.Insert(7, "Inserted ") // Insert at beginning of "Line 2"
	
	// Iterator should still see the same content at its position
	if iter1.Line() != "Line 1" {
		t.Errorf("After insert after position, expected 'Line 1', got %q", iter1.Line())
	}
	
	// Move to next line - should see the modified content
	iter1 = iter1.Next()
	if iter1.Line() != "Inserted Line 2" {
		t.Errorf("After moving to next line, expected 'Inserted Line 2', got %q", iter1.Line())
	}
	
	// Test 2: To modify before an iterator, get an earlier position first
	// Get iterator at the beginning
	iter2 := NewBufferIterator(buf)
	
	// Insert at beginning of buffer
	buf.Insert(0, "New First ")
	
	// The iterator at position 0 now sees the new content
	if iter2.Line() != "New First Line 1" {
		t.Errorf("Iterator at position 0 should see 'New First Line 1', got %q", iter2.Line())
	}
	
	// Create a new iterator to see the changes
	iter3 := NewBufferIterator(buf)
	if iter3.Line() != "New First Line 1" {
		t.Errorf("New iterator should see 'New First Line 1', got %q", iter3.Line())
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