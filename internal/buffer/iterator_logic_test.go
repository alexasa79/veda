package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestIteratorLogic(t *testing.T) {
	t.Run("construction", func(t *testing.T) {
		t.Run("NewBufferIterator at beginning", func(t *testing.T) {
			// Test with simple single-line buffer
			buf := NewBuffer("Hello World", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)
			
			// Verify iterator is at beginning
			if iter.LineNum != 0 {
				t.Errorf("Expected line number 0, got %d", iter.LineNum)
			}
			if iter.Offset != 0 {
				t.Errorf("Expected offset 0, got %d", iter.Offset)
			}
			
			// Verify iterator points to a leaf node
			if iter.Node == nil {
				t.Fatal("Expected iterator to have non-nil node")
			}
			if !iter.Node.IsLeaf() {
				t.Error("Expected iterator to point to a leaf node")
			}
			
			// Verify node offset matches iterator offset
			if iter.Node.offset != 0 {
				t.Errorf("Expected node offset 0, got %d", iter.Node.offset)
			}
			
			// Verify content is correct
			if iter.Line() != "Hello World" {
				t.Errorf("Expected 'Hello World', got '%s'", iter.Line())
			}
		})
		
		t.Run("NewBufferIterator with multi-line", func(t *testing.T) {
			// Test with multi-line buffer
			buf := NewBuffer("Line 1\nLine 2\nLine 3", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)
			
			// Verify iterator is at beginning
			if iter.LineNum != 0 {
				t.Errorf("Expected line number 0, got %d", iter.LineNum)
			}
			if iter.Offset != 0 {
				t.Errorf("Expected offset 0, got %d", iter.Offset)
			}
			
			// Verify iterator points to a leaf node
			if iter.Node == nil {
				t.Fatal("Expected iterator to have non-nil node")
			}
			if !iter.Node.IsLeaf() {
				t.Error("Expected iterator to point to a leaf node")
			}
			
			// Verify content is first line
			if iter.Line() != "Line 1" {
				t.Errorf("Expected 'Line 1', got '%s'", iter.Line())
			}
		})
		
		t.Run("NewBufferIteratorAtEnd", func(t *testing.T) {
			// Test with multi-line buffer
			buf := NewBuffer("Line 1\nLine 2\nLine 3", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIteratorAtEnd(buf)
			
			// Verify iterator is at EOF
			if !iter.IsEOF() {
				t.Error("Expected iterator to be at EOF")
			}
			
			// Verify offset is at buffer length
			expectedOffset := buf.Rope.Length()
			if iter.Offset != expectedOffset {
				t.Errorf("Expected offset %d, got %d", expectedOffset, iter.Offset)
			}
			
			// Verify iterator points to a leaf node (the last one)
			if iter.Node == nil {
				t.Fatal("Expected iterator to have non-nil node")
			}
			if !iter.Node.IsLeaf() {
				t.Error("Expected iterator to point to a leaf node")
			}
			
			// For EOF iterator, line number should be past the last line
			// "Line 1\nLine 2\nLine 3" has 2 newlines, so lineNum = 2 + 1 = 3
			expectedLineNum := 3
			if iter.LineNum != expectedLineNum {
				t.Errorf("Expected line number %d, got %d", expectedLineNum, iter.LineNum)
			}
		})
		
		t.Run("empty buffer", func(t *testing.T) {
			// Test with empty buffer
			buf := NewBuffer("", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)
			
			// Verify iterator is at beginning (which is also EOF for empty buffer)
			if iter.LineNum != 0 {
				t.Errorf("Expected line number 0, got %d", iter.LineNum)
			}
			if iter.Offset != 0 {
				t.Errorf("Expected offset 0, got %d", iter.Offset)
			}
			if !iter.IsEOF() {
				t.Error("Expected empty buffer iterator to be at EOF")
			}
			
			// Verify iterator points to a leaf node
			if iter.Node == nil {
				t.Fatal("Expected iterator to have non-nil node")
			}
			if !iter.Node.IsLeaf() {
				t.Error("Expected iterator to point to a leaf node")
			}
		})
		
		t.Run("buffer ending with newline", func(t *testing.T) {
			// Test buffer that ends with \n
			buf := NewBuffer("Line 1\nLine 2\nLine 3\n", "test.txt", config.IndentSpaces, 4)
			
			// Test NewBufferIterator
			iter := NewBufferIterator(buf)
			if iter.LineNum != 0 {
				t.Errorf("Expected line number 0, got %d", iter.LineNum)
			}
			if iter.Line() != "Line 1" {
				t.Errorf("Expected 'Line 1', got '%s'", iter.Line())
			}
			
			// Test NewBufferIteratorAtEnd
			iterEnd := NewBufferIteratorAtEnd(buf)
			if !iterEnd.IsEOF() {
				t.Error("Expected iterator to be at EOF")
			}
			
			// For "Line 1\nLine 2\nLine 3\n" there are 3 newlines
			// So lineNum should be 3 + 1 = 4 (representing empty line after last \n)
			expectedLineNum := 4
			if iterEnd.LineNum != expectedLineNum {
				t.Errorf("Expected EOF line number %d, got %d", expectedLineNum, iterEnd.LineNum)
			}
			
			// Verify EOF offset is at buffer length
			expectedOffset := buf.Rope.Length() // Should be 21 chars
			if iterEnd.Offset != expectedOffset {
				t.Errorf("Expected EOF offset %d, got %d", expectedOffset, iterEnd.Offset)
			}
			
		})
	})
	
	t.Run("basic navigation", func(t *testing.T) {
		t.Run("Next moves forward one line", func(t *testing.T) {
			buf := NewBuffer("Line 1\nLine 2\nLine 3", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)
			
			// Start at line 0
			if iter.LineNum != 0 || iter.Line() != "Line 1" {
				t.Errorf("Expected line 0 with 'Line 1', got line %d with '%s'", iter.LineNum, iter.Line())
			}
			
			// Move to line 1
			iter = iter.Next()
			if iter.LineNum != 1 || iter.Line() != "Line 2" {
				t.Errorf("Expected line 1 with 'Line 2', got line %d with '%s'", iter.LineNum, iter.Line())
			}
			
			// Move to line 2
			iter = iter.Next()
			if iter.LineNum != 2 || iter.Line() != "Line 3" {
				t.Errorf("Expected line 2 with 'Line 3', got line %d with '%s'", iter.LineNum, iter.Line())
			}
			
			// Move to EOF
			iter = iter.Next()
			if !iter.IsEOF() {
				t.Error("Expected to be at EOF after last line")
			}
		})
		
		t.Run("Previous moves backward one line", func(t *testing.T) {
			buf := NewBuffer("Line 1\nLine 2\nLine 3", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIteratorAtEnd(buf)
			
			// Start at EOF
			if !iter.IsEOF() {
				t.Error("Expected to start at EOF")
			}
			
			// Move to line 2
			iter = iter.Previous()
			t.Logf("After 1st Previous(): LineNum=%d, Offset=%d, Line='%s'", iter.LineNum, iter.Offset, iter.Line())
			if iter.LineNum != 2 || iter.Line() != "Line 3" {
				t.Errorf("Expected line 2 with 'Line 3', got line %d with '%s'", iter.LineNum, iter.Line())
			}
			
			// Move to line 1
			iter = iter.Previous()
			t.Logf("After 2nd Previous(): LineNum=%d, Offset=%d, Line='%s'", iter.LineNum, iter.Offset, iter.Line())
			if iter.LineNum != 1 || iter.Line() != "Line 2" {
				t.Errorf("Expected line 1 with 'Line 2', got line %d with '%s'", iter.LineNum, iter.Line())
			}
			
			// Move to line 0
			iter = iter.Previous()
			t.Logf("After 3rd Previous(): LineNum=%d, Offset=%d, Line='%s'", iter.LineNum, iter.Offset, iter.Line())
			if iter.LineNum != 0 || iter.Line() != "Line 1" {
				t.Errorf("Expected line 0 with 'Line 1', got line %d with '%s'", iter.LineNum, iter.Line())
			}
		})
		
		t.Run("navigation maintains correct offsets", func(t *testing.T) {
			buf := NewBuffer("Hello\nWorld\nTest", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)
			
			// Line 0: "Hello" should start at offset 0
			if iter.Offset != 0 {
				t.Errorf("Expected offset 0, got %d", iter.Offset)
			}
			
			// Move to line 1: "World" should start at offset 6
			iter = iter.Next()
			expectedOffset := 6 // "Hello\n" = 6 chars
			if iter.Offset != expectedOffset {
				t.Errorf("Expected offset %d, got %d", expectedOffset, iter.Offset)
			}
			
			// Move to line 2: "Test" should start at offset 12
			iter = iter.Next()
			expectedOffset = 12 // "Hello\nWorld\n" = 12 chars
			if iter.Offset != expectedOffset {
				t.Errorf("Expected offset %d, got %d", expectedOffset, iter.Offset)
			}
		})
		
		t.Run("navigation maintains correct node pointers", func(t *testing.T) {
			buf := NewBuffer("Line1\nLine2", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)
			
			// Verify initial node is a leaf
			if iter.Node == nil || !iter.Node.IsLeaf() {
				t.Error("Expected iterator to point to a leaf node")
			}
			initialNode := iter.Node
			
			// Move to next line
			iter = iter.Next()
			
			// Verify new node is also a leaf
			if iter.Node == nil || !iter.Node.IsLeaf() {
				t.Error("Expected iterator to point to a leaf node after Next()")
			}
			
			// Node might be the same (if content fits in one leaf) or different
			// but should always be a valid leaf
			if iter.Node == initialNode {
				t.Logf("Iterator stayed on same node (content fits in one leaf)")
			} else {
				t.Logf("Iterator moved to different node")
			}
		})
	})
}