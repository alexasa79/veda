package buffer

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/config"
)

// TestIteratorCorruptionLargeFile reproduces the crash that occurs when editing
// a large file: press "o" early in file, type text, then backspace
func TestIteratorCorruptionLargeFile(t *testing.T) {
	// Generate a large C file similar to the 280K file that causes the issue
	// We'll create about 8000 lines with varying content
	var content strings.Builder
	
	// Add some includes and defines at the top (similar to a real C file)
	content.WriteString("#include <stdio.h>\n")
	content.WriteString("#include <stdlib.h>\n")
	content.WriteString("#include <string.h>\n")
	content.WriteString("\n")
	
	// Add function declarations and definitions
	for i := 0; i < 8000; i++ {
		if i%100 == 0 {
			// Add some variety - function definitions
			content.WriteString(fmt.Sprintf("\n/* Function group %d */\n", i/100))
		}
		
		if i%10 == 0 {
			// Function definition
			content.WriteString(fmt.Sprintf("int function_%d(int param1, int param2) {\n", i))
			content.WriteString("    int result = param1 + param2;\n")
			content.WriteString("    return result;\n")
			content.WriteString("}\n\n")
		} else {
			// Regular code line
			content.WriteString(fmt.Sprintf("    process_data_%d(); // Processing step %d\n", i, i))
		}
	}
	
	largeContent := content.String()
	t.Logf("Generated content size: %d bytes", len(largeContent))
	
	// Create buffer with the large content
	buffer := NewBuffer(largeContent, "test.c", config.IndentSpaces, 4)
	
	// Find a position early in the file (around line 50)
	targetLine := 50
	iter := NewBufferIteratorAtLine(buffer, targetLine)
	if iter.IsEOF() {
		t.Fatal("Iterator at line 50 should not be EOF")
	}
	
	// Get the position at end of line 50
	line := iter.Line()
	insertPos := iter.Position() + len(line)
	
	// Simulate pressing "o" - insert newline and move to new line
	buffer.Insert(insertPos, "\n")
	
	// Create cursor iterator at the new line
	cursorIter := NewBufferIteratorAtLine(buffer, targetLine + 1)
	
	// Simulate typing "static __threasd"
	textToType := "static __threasd"
	currentPos := cursorIter.Position()
	for i, ch := range textToType {
		buffer.Insert(currentPos + i, string(ch))
	}
	
	// Now simulate backspace - delete the 'd'
	deletePos := currentPos + len(textToType) - 1
	buffer.Delete(deletePos, deletePos + 1)
	
	// Create a view iterator at some line before the cursor (simulating the ViewIterator)
	viewLine := 30 // View starts at line 30
	viewIter := NewBufferIteratorAtLine(buffer, viewLine)
	
	// Now simulate what happens in drawEditor - iterate through visible lines
	// This is where the crash would occur
	visibleLines := 50 // Simulate 50 visible lines
	
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Caught panic as expected: %v", r)
			// Check if it's the specific assertion we expect
			if panicStr, ok := r.(string); ok {
				if strings.Contains(panicStr, "position in node must be >= 0") {
					t.Log("Successfully reproduced the iterator corruption bug!")
					return
				}
			}
			t.Errorf("Unexpected panic: %v", r)
		}
	}()
	
	// Try to iterate through visible lines (this should trigger the crash)
	currentIter := viewIter
	for i := 0; i < visibleLines && !currentIter.IsEOF(); i++ {
		// This is what drawEditor does
		_ = currentIter.Line()
		nextIter := currentIter.Next()
		
		// Validate the iterator state
		if nextIter.Offset < nextIter.Node.offset {
			t.Errorf("Iterator corruption detected at line %d: Offset=%d, Node.offset=%d",
				i + viewLine, nextIter.Offset, nextIter.Node.offset)
			
			// This would trigger the assertion in getCharAt
			// Let's force it to happen
			positionInNode := nextIter.Offset - nextIter.Node.offset
			if positionInNode < 0 {
				panic(fmt.Sprintf("assertion failed: getCharAt: position in node must be >= 0 (got %d)", positionInNode))
			}
		}
		
		currentIter = nextIter
	}
	
	t.Error("Expected iterator corruption but none occurred - the bug might be more specific to the exact file structure")
}

// TestIteratorCorruptionWithRebalancing tests a more targeted scenario
// focusing on operations that trigger rope rebalancing
func TestIteratorCorruptionWithRebalancing(t *testing.T) {
	// Create a buffer with content that will force rebalancing
	// We'll create many small segments to force a deep tree
	var segments []string
	segmentSize := 100 // Small segments to force many nodes
	numSegments := 3000 // This creates a ~300K file
	
	for i := 0; i < numSegments; i++ {
		segment := fmt.Sprintf("/* Segment %d */ function_%d() { return %d; }\n", i, i, i)
		// Pad to segmentSize
		for len(segment) < segmentSize {
			segment += " "
		}
		segments = append(segments, segment)
	}
	
	// Build the content by inserting segments at different positions
	// This should create a complex tree structure
	content := strings.Join(segments[:1000], "")
	buffer := NewBuffer(content, "test.c", config.IndentSpaces, 4)
	
	// Insert remaining segments at various positions to force rebalancing
	for i := 1000; i < numSegments; i++ {
		// Insert at calculated positions to spread throughout the buffer
		pos := (i * 1000) % buffer.Rope.Length()
		buffer.Insert(pos, segments[i])
	}
	
	t.Logf("Created buffer with %d bytes", buffer.Rope.Length())
	
	// Now perform the problematic sequence
	// Find line 50
	targetLine := 50
	iter := NewBufferIteratorAtLine(buffer, targetLine)
	
	// Insert newline (simulate "o")
	insertPos := iter.Position() + len(iter.Line())
	buffer.Insert(insertPos, "\n    ")
	
	// Type some text
	textPos := insertPos + 5 // After newline and indent
	typeText := "static __threasd"
	for i := range typeText {
		buffer.Insert(textPos + i, string(typeText[i]))
	}
	
	// Delete last character (backspace)
	buffer.Delete(textPos + len(typeText) - 1, textPos + len(typeText))
	
	// Create view iterator
	viewIter := NewBufferIteratorAtLine(buffer, 30)
	
	// Set up panic recovery to catch the exact error
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Successfully reproduced the crash: %v", r)
			// Log the exact state that caused the crash
			t.Logf("ViewIterator state at crash:")
			t.Logf("  Offset: %d", viewIter.Offset)
			t.Logf("  Node.offset: %d", viewIter.Node.offset)
			t.Logf("  Difference: %d", viewIter.Offset - viewIter.Node.offset)
			t.Logf("  LineNum: %d", viewIter.LineNum)
			t.Logf("  Generation: %d", viewIter.Generation)
			t.Logf("  Buffer.Generation: %d", viewIter.Buffer.Generation)
			if viewIter.Node.IsLeaf() {
				t.Logf("  Node text length: %d", len(viewIter.Node.text))
				if len(viewIter.Node.text) < 100 {
					t.Logf("  Node text: %q", viewIter.Node.text)
				}
			}
		}
	}()
	
	// Check for corruption
	for i := 0; i < 100 && !viewIter.IsEOF(); i++ {
		// Log current state before potential crash
		if i < 5 || (i > 25 && i < 35) { // Log around where we expect issues
			t.Logf("Iteration %d: Offset=%d, Node.offset=%d, LineNum=%d", 
				i, viewIter.Offset, viewIter.Node.offset, viewIter.LineNum)
		}
		
		// Check iterator consistency
		if viewIter.Offset < viewIter.Node.offset {
			t.Errorf("Iterator corruption detected at iteration %d: Offset=%d < Node.offset=%d",
				i, viewIter.Offset, viewIter.Node.offset)
			
			// This should trigger the panic
			_ = viewIter.Line()
		}
		
		viewIter = viewIter.Next()
	}
}

// TestRopeNodeOffsetConsistency verifies that node offsets remain consistent
// after insertions and deletions in a large rope
func TestRopeNodeOffsetConsistency(t *testing.T) {
	// Create a large rope
	content := strings.Repeat("This is a line of text.\n", 10000)
	buffer := NewBuffer(content, "test.txt", config.IndentSpaces, 4)
	
	// Perform many operations
	for i := 0; i < 100; i++ {
		// Insert at various positions
		pos := (i * 1000) % buffer.Rope.Length()
		buffer.Insert(pos, fmt.Sprintf("INSERT%d", i))
		
		// Delete at various positions
		if buffer.Rope.Length() > 1000 {
			delPos := (i * 500) % (buffer.Rope.Length() - 10)
			buffer.Delete(delPos, delPos + 5)
		}
		
		// After each operation, verify all node offsets
		verifyNodeOffsets(t, buffer.Rope.root, 0, i)
	}
}

// verifyNodeOffsets recursively checks that all node offsets are correct
func verifyNodeOffsets(t *testing.T, node *Node, expectedOffset int, iteration int) int {
	if node == nil {
		return expectedOffset
	}
	
	if node.offset != expectedOffset {
		t.Errorf("Iteration %d: Node offset mismatch: expected %d, got %d",
			iteration, expectedOffset, node.offset)
	}
	
	if node.IsLeaf() {
		return expectedOffset + len(node.text)
	}
	
	// Internal node
	nextOffset := expectedOffset
	if node.left != nil {
		nextOffset = verifyNodeOffsets(t, node.left, expectedOffset, iteration)
	}
	if node.right != nil {
		nextOffset = verifyNodeOffsets(t, node.right, nextOffset, iteration)
	}
	
	return nextOffset
}