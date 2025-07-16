package buffer

import (
	"testing"
)

func TestIteratorCorruption(t *testing.T) {
	t.Run("negative position in node", func(t *testing.T) {
		// Create a buffer with some content
		buf := NewBuffer("Hello\nWorld\nTest", "", 0, 4)
		
		// Create an iterator at the start
		iter := NewBufferIterator(buf)
		
		// Manually corrupt the iterator to have an offset less than the node's offset
		// This simulates what happens when rope rebalancing changes node offsets
		// Save original values
		originalOffset := iter.Offset
		originalNode := iter.Node
		
		// Force a situation where it.Offset < it.Node.offset
		iter.Offset = 2  // Iterator thinks it's at position 2
		iter.Node.offset = 5  // But the node starts at position 5
		
		// This should cause the crash with "position in node must be >= 0"
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(string); ok && err == "assertion failed: getCharAt: position in node must be >= 0" {
					t.Logf("Successfully reproduced crash: %v", err)
					t.Logf("Iterator offset: %d, Node offset: %d, Difference: %d", 
						iter.Offset, iter.Node.offset, iter.Offset - iter.Node.offset)
				} else {
					t.Errorf("Got unexpected panic: %v", r)
				}
			} else {
				t.Error("Expected panic but none occurred")
			}
		}()
		
		// Try to use the iterator - this should crash
		// Next() calls getCharAt to check for newlines
		_ = iter.Next()
		
		t.Logf("Original offset: %d, node: %v", originalOffset, originalNode)
	})
	
	t.Run("rope rebalancing corruption", func(t *testing.T) {
		// Try to reproduce the corruption through actual rope operations
		buf := NewBuffer("line1\nline2\nline3", "", 0, 4)
		
		// Get an iterator at line 2
		iter := NewBufferIteratorAtLine(buf, 1)
		
		// Save the node reference
		nodeBeforeInsert := iter.Node
		offsetBeforeInsert := iter.Node.offset
		
		// Insert a large amount of text at the beginning to trigger rebalancing
		largeText := ""
		for i := 0; i < 100; i++ {
			largeText += "X"
		}
		buf.Insert(0, largeText)
		
		// The node reference might still be valid but its offset could have changed
		// Check if the node's offset changed
		if nodeBeforeInsert.offset != offsetBeforeInsert {
			t.Logf("Node offset changed from %d to %d after insert", 
				offsetBeforeInsert, nodeBeforeInsert.offset)
		}
		
		// Create a new iterator and see if it works correctly
		newIter := NewBufferIteratorAtLine(buf, 1)
		if !newIter.IsEOF() {
			line := newIter.Line()
			t.Logf("New iterator successfully read line: %q", line)
		}
	})
}