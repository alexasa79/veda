package buffer

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/config"
)

// TestIteratorNodeOffsetBug reproduces the specific bug where Next() creates
// an iterator with inconsistent offset and node values
func TestIteratorNodeOffsetBug(t *testing.T) {
	// Create a buffer structure that will trigger the bug
	// We need a large enough buffer to create multiple nodes

	// First, create initial content
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("Line %d with some content to make it longer", i))
	}
	content := strings.Join(lines, "\n")

	buffer := NewBuffer(content, "test.c", "test.c", config.IndentSpaces, 4)

	// Perform insertions at various positions to create a complex tree
	// Insert at position that will cause rebalancing
	for i := 0; i < 20; i++ {
		pos := (i * 1000) % buffer.Rope.Length()
		buffer.Insert(pos, fmt.Sprintf("/* Insert %d */", i))
	}

	// Now perform the specific sequence that triggers the bug
	targetLine := 50
	iter := NewBufferIteratorAtLine(buffer, targetLine)
	insertPos := iter.Position() + len(iter.Line())

	// Insert newline (simulate "o")
	buffer.Insert(insertPos, "\n    ")

	// Type text
	textPos := insertPos + 5
	buffer.Insert(textPos, "static __threasd")

	// Delete last character
	buffer.Delete(textPos+15, textPos+16)

	// Now test iterator creation and Next() behavior
	t.Run("iterator Next() consistency", func(t *testing.T) {
		// Create an iterator at line 30
		iter := NewBufferIteratorAtLine(buffer, 30)

		// Track the iterations
		for i := 0; i < 100 && !iter.IsEOF(); i++ {
			// Before calling Next, verify the current iterator is valid
			if iter.Offset < iter.Node.offset {
				t.Fatalf("Iterator invalid before Next() at iteration %d: Offset=%d < Node.offset=%d",
					i, iter.Offset, iter.Node.offset)
			}

			// Verify position is within node bounds
			posInNode := iter.Offset - iter.Node.offset
			if posInNode < 0 || (iter.Node.IsLeaf() && posInNode >= len(iter.Node.text)) {
				t.Fatalf("Iterator position out of bounds at iteration %d: posInNode=%d, node text len=%d",
					i, posInNode, len(iter.Node.text))
			}

			// Get the line (this might panic)
			line := iter.Line()
			_ = line // use it

			// Call Next
			nextIter := iter.Next()

			// Check if Next returned a valid iterator
			if !nextIter.IsEOF() {
				if nextIter.Offset < nextIter.Node.offset {
					t.Errorf("Next() returned invalid iterator at iteration %d:", i)
					t.Errorf("  Current: Offset=%d, Node.offset=%d", iter.Offset, iter.Node.offset)
					t.Errorf("  Next: Offset=%d, Node.offset=%d", nextIter.Offset, nextIter.Node.offset)
					t.Errorf("  Next position in node: %d", nextIter.Offset-nextIter.Node.offset)

					// Log node details
					if nextIter.Node.IsLeaf() {
						t.Errorf("  Next node text length: %d", len(nextIter.Node.text))
						t.Errorf("  Next node starts with: %q", truncateString(nextIter.Node.text, 50))
					}

					// Try to trigger the panic
					defer func() {
						if r := recover(); r != nil {
							t.Logf("Expected panic: %v", r)
						}
					}()
					_ = nextIter.Line() // This should panic
					break
				}
			}

			iter = nextIter
		}
	})
}

// TestNextMethodLogic examines the Next() method implementation
func TestNextMethodLogic(t *testing.T) {
	// Create a simple buffer where we can control the structure
	content := "Line1\nLine2\nLine3\n"
	buffer := NewBuffer(content, "test.txt", "test.txt", config.IndentSpaces, 4)

	// Create iterator at line 0
	iter := NewBufferIteratorAtLine(buffer, 0)

	t.Logf("Initial iterator: Offset=%d, Node.offset=%d, LineNum=%d",
		iter.Offset, iter.Node.offset, iter.LineNum)

	// Call Next and log what happens
	next1 := iter.Next()
	t.Logf("After 1st Next: Offset=%d, Node.offset=%d, LineNum=%d",
		next1.Offset, next1.Node.offset, next1.LineNum)

	next2 := next1.Next()
	t.Logf("After 2nd Next: Offset=%d, Node.offset=%d, LineNum=%d",
		next2.Offset, next2.Node.offset, next2.LineNum)

	// Now let's create a more complex scenario
	// Insert a large amount of text to potentially create multiple nodes
	largeText := strings.Repeat("X", 5000)
	buffer.Insert(10, largeText)

	t.Logf("\nAfter inserting large text:")

	// Recreate iterator
	iter2 := NewBufferIteratorAtLine(buffer, 0)
	t.Logf("Recreated iterator: Offset=%d, Node.offset=%d, LineNum=%d",
		iter2.Offset, iter2.Node.offset, iter2.LineNum)

	// Try to navigate
	for i := 0; i < 3 && !iter2.IsEOF(); i++ {
		iter2 = iter2.Next()
		t.Logf("After Next %d: Offset=%d, Node.offset=%d, LineNum=%d, posInNode=%d",
			i+1, iter2.Offset, iter2.Node.offset, iter2.LineNum,
			iter2.Offset-iter2.Node.offset)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
