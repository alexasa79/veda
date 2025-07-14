package buffer

import (
	"strconv"
	"strings"
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

	t.Run("edge case navigation", func(t *testing.T) {
		t.Run("Next at EOF stays at EOF", func(t *testing.T) {
			buf := NewBuffer("Line 1\nLine 2", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIteratorAtEnd(buf)

			// Verify we're at EOF
			if !iter.IsEOF() {
				t.Error("Expected iterator to be at EOF")
			}
			initialOffset := iter.Offset
			initialLineNum := iter.LineNum

			// Call Next() multiple times
			iter2 := iter.Next()
			iter3 := iter2.Next()

			// Should stay at same position
			if iter2.Offset != initialOffset || iter2.LineNum != initialLineNum {
				t.Errorf("Expected Next() at EOF to stay at same position, got offset %d->%d, line %d->%d",
					initialOffset, iter2.Offset, initialLineNum, iter2.LineNum)
			}
			if iter3.Offset != initialOffset || iter3.LineNum != initialLineNum {
				t.Errorf("Expected multiple Next() at EOF to stay at same position")
			}
		})

		t.Run("Previous at beginning stays at beginning", func(t *testing.T) {
			buf := NewBuffer("Line 1\nLine 2", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// Verify we're at beginning
			if iter.Offset != 0 || iter.LineNum != 0 {
				t.Error("Expected iterator to be at beginning")
			}

			// Call Previous() multiple times
			iter2 := iter.Previous()
			iter3 := iter2.Previous()

			// Should stay at line 0, offset 0
			if iter2.Offset != 0 || iter2.LineNum != 0 {
				t.Errorf("Expected Previous() at beginning to stay at line 0, offset 0")
			}
			if iter3.Offset != 0 || iter3.LineNum != 0 {
				t.Errorf("Expected multiple Previous() at beginning to stay at line 0, offset 0")
			}
		})

		t.Run("navigation through empty lines", func(t *testing.T) {
			// Buffer with empty lines: "Line1\n\n\nLine4"
			buf := NewBuffer("Line1\n\n\nLine4", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// Line 0: "Line1"
			if iter.Line() != "Line1" || iter.LineNum != 0 {
				t.Errorf("Expected line 0 'Line1', got line %d '%s'", iter.LineNum, iter.Line())
			}

			// Line 1: "" (empty)
			iter = iter.Next()
			if iter.Line() != "" || iter.LineNum != 1 {
				t.Errorf("Expected line 1 '', got line %d '%s'", iter.LineNum, iter.Line())
			}

			// Line 2: "" (empty)
			iter = iter.Next()
			if iter.Line() != "" || iter.LineNum != 2 {
				t.Errorf("Expected line 2 '', got line %d '%s'", iter.LineNum, iter.Line())
			}

			// Line 3: "Line4"
			iter = iter.Next()
			if iter.Line() != "Line4" || iter.LineNum != 3 {
				t.Errorf("Expected line 3 'Line4', got line %d '%s'", iter.LineNum, iter.Line())
			}

			// Navigate backwards through empty lines
			iter = iter.Previous() // back to line 2 (empty)
			if iter.Line() != "" || iter.LineNum != 2 {
				t.Errorf("Previous from line 3: expected line 2 '', got line %d '%s'", iter.LineNum, iter.Line())
			}

			iter = iter.Previous() // back to line 1 (empty)
			if iter.Line() != "" || iter.LineNum != 1 {
				t.Errorf("Previous from line 2: expected line 1 '', got line %d '%s'", iter.LineNum, iter.Line())
			}
		})

		t.Run("navigation in single-line buffer", func(t *testing.T) {
			// Buffer with no newlines
			buf := NewBuffer("Single line only", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// Should be at line 0
			if iter.LineNum != 0 || iter.Line() != "Single line only" {
				t.Errorf("Expected line 0 'Single line only', got line %d '%s'", iter.LineNum, iter.Line())
			}

			// Next() should go to EOF
			iter2 := iter.Next()
			if !iter2.IsEOF() {
				t.Error("Expected Next() on single line to reach EOF")
			}
			if iter2.LineNum != 1 {
				t.Errorf("Expected EOF line number to be 1, got %d", iter2.LineNum)
			}

			// Previous() from EOF should go back to line 0
			iter3 := iter2.Previous()
			if iter3.LineNum != 0 || iter3.Line() != "Single line only" {
				t.Errorf("Expected Previous() from EOF to return to line 0, got line %d '%s'", iter3.LineNum, iter3.Line())
			}
		})

		t.Run("buffer starting with newline", func(t *testing.T) {
			// Buffer starting with newline: "\nLine2\nLine3"
			buf := NewBuffer("\nLine2\nLine3", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// Line 0 should be empty
			if iter.Line() != "" || iter.LineNum != 0 {
				t.Errorf("Expected line 0 '', got line %d '%s'", iter.LineNum, iter.Line())
			}

			// Line 1 should be "Line2"
			iter = iter.Next()
			if iter.Line() != "Line2" || iter.LineNum != 1 {
				t.Errorf("Expected line 1 'Line2', got line %d '%s'", iter.LineNum, iter.Line())
			}
		})

		t.Run("multiple consecutive newlines", func(t *testing.T) {
			// Buffer with many consecutive newlines
			buf := NewBuffer("A\n\n\n\n\nB", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// Count lines forward
			lineCount := 0
			for !iter.IsEOF() {
				lineCount++
				iter = iter.Next()
			}

			// Should have 6 lines: "A", "", "", "", "", "B"
			if lineCount != 6 {
				t.Errorf("Expected 6 lines, got %d", lineCount)
			}

			// Navigate backward and verify
			iter = iter.Previous() // Line 5: "B"
			if iter.Line() != "B" || iter.LineNum != 5 {
				t.Errorf("Expected line 5 'B', got line %d '%s'", iter.LineNum, iter.Line())
			}

			// Lines 4,3,2,1 should all be empty
			for expectedLine := 4; expectedLine >= 1; expectedLine-- {
				iter = iter.Previous()
				if iter.Line() != "" || iter.LineNum != expectedLine {
					t.Errorf("Expected line %d '', got line %d '%s'", expectedLine, iter.LineNum, iter.Line())
				}
			}

			// Line 0: "A"
			iter = iter.Previous()
			if iter.Line() != "A" || iter.LineNum != 0 {
				t.Errorf("Expected line 0 'A', got line %d '%s'", iter.LineNum, iter.Line())
			}
		})
	})

	t.Run("node traversal", func(t *testing.T) {
		t.Run("iterator moves between leaf nodes correctly", func(t *testing.T) {
			// Create a buffer that will have multiple nodes
			// Start with small buffer and insert to trigger splits
			buf := NewBuffer("initial", "test.txt", config.IndentSpaces, 4)

			// Insert large text to force splits
			largeText := strings.Repeat("X", 5000)
			buf.Rope.Insert(7, largeText) // Insert after "initial"

			// Verify the buffer has multiple nodes
			if buf.Rope.root.IsLeaf() {
				t.Skip("Buffer didn't split into multiple nodes, skipping multi-node test")
			}

			// Track unique nodes visited by traversing character by character
			visitedNodes := make(map[*Node]bool)
			iter := *NewBufferIterator(buf)

			// Navigate through the buffer character by character to ensure we visit all nodes
			charCount := 0
			for iter.Offset < buf.Rope.Length() {
				visitedNodes[iter.Node] = true
				charCount++
				var ok bool
				iter, ok = iter.nextChar()
				if !ok {
					break
				}
			}

			// Should have visited multiple nodes
			if len(visitedNodes) < 2 {
				t.Errorf("Expected to visit multiple nodes, but only visited %d", len(visitedNodes))
			}
			t.Logf("Visited %d unique nodes while traversing %d characters", len(visitedNodes), charCount)
		})

		t.Run("node pointers updated correctly during forward traversal", func(t *testing.T) {
			// Create a rope with exactly two nodes
			left := NewLeaf("First part\n")
			right := NewLeaf("Second part\n")
			left.offset = 0
			right.offset = 11 // "First part\n" is 11 bytes
			root := NewConcatNode(left, right)
			buf := &Buffer{Rope: &Rope{root: root}}

			iter := NewBufferIterator(buf)

			// Should start in left node
			if iter.Node != left {
				t.Error("Expected iterator to start in left node")
			}
			if iter.Offset != 0 {
				t.Errorf("Expected initial offset 0, got %d", iter.Offset)
			}

			// Move to second line (should be in right node)
			iter = iter.Next()
			if iter.Node != right {
				t.Error("Expected iterator to move to right node after Next()")
			}
			if iter.Offset != 11 {
				t.Errorf("Expected offset 11 after Next(), got %d", iter.Offset)
			}
		})

		t.Run("node pointers updated correctly during backward traversal", func(t *testing.T) {
			// Create a rope with exactly two nodes
			left := NewLeaf("First part\n")
			right := NewLeaf("Second part")
			left.offset = 0
			right.offset = 11
			root := NewConcatNode(left, right)
			buf := &Buffer{Rope: &Rope{root: root}}

			// Start at end
			iter := NewBufferIteratorAtEnd(buf)

			// Should be in right node at EOF
			if iter.Node != right {
				t.Error("Expected EOF iterator to be in right node")
			}

			// Move to last line (should still be in right node)
			iter = iter.Previous()
			if iter.Node != right {
				t.Error("Expected iterator to stay in right node for last line")
			}
			if iter.Offset != 11 {
				t.Errorf("Expected offset 11 for last line, got %d", iter.Offset)
			}

			// Move to first line (should be in left node)
			iter = iter.Previous()
			if iter.Node != left {
				t.Error("Expected iterator to move to left node after Previous()")
			}
			if iter.Offset != 0 {
				t.Errorf("Expected offset 0 after Previous(), got %d", iter.Offset)
			}
		})

		t.Run("offset values match node offset during traversal", func(t *testing.T) {
			// Create a complex tree structure
			node1 := NewLeaf("Part1\n") // 6 bytes
			node2 := NewLeaf("Part2\n") // 6 bytes
			node3 := NewLeaf("Part3\n") // 6 bytes
			node4 := NewLeaf("Part4")   // 5 bytes

			// Set offsets
			node1.offset = 0
			node2.offset = 6
			node3.offset = 12
			node4.offset = 18

			// Build tree: ((Part1, Part2), (Part3, Part4))
			left := NewConcatNode(node1, node2)
			right := NewConcatNode(node3, node4)
			root := NewConcatNode(left, right)

			buf := &Buffer{Rope: &Rope{root: root}}
			iter := NewBufferIterator(buf)

			// Verify offset invariant: iter.Offset should be within [node.offset, node.offset + len(node.text))
			positions := []struct {
				expectedNode   *Node
				expectedOffset int
				expectedLine   string
			}{
				{node1, 0, "Part1"},
				{node2, 6, "Part2"},
				{node3, 12, "Part3"},
				{node4, 18, "Part4"},
			}

			for i, pos := range positions {
				if iter.Node != pos.expectedNode {
					t.Errorf("Line %d: expected node %v, got %v", i, pos.expectedNode, iter.Node)
				}
				if iter.Offset != pos.expectedOffset {
					t.Errorf("Line %d: expected offset %d, got %d", i, pos.expectedOffset, iter.Offset)
				}
				if iter.Line() != pos.expectedLine {
					t.Errorf("Line %d: expected content %q, got %q", i, pos.expectedLine, iter.Line())
				}

				// Verify offset is within node bounds
				nodeStart := iter.Node.offset
				nodeEnd := nodeStart + len(iter.Node.text)
				if iter.Offset < nodeStart || iter.Offset >= nodeEnd {
					t.Errorf("Line %d: offset %d not within node bounds [%d, %d)",
						i, iter.Offset, nodeStart, nodeEnd)
				}

				if i < len(positions)-1 {
					iter = iter.Next()
				}
			}
		})

		t.Run("character-by-character traversal across nodes", func(t *testing.T) {
			// Create rope with two nodes
			left := NewLeaf("ABC")
			right := NewLeaf("DEF")
			left.offset = 0
			right.offset = 3
			root := NewConcatNode(left, right)
			buf := &Buffer{Rope: &Rope{root: root}}

			iter := *NewBufferIterator(buf)

			// Move character by character using nextChar
			expectedChars := []byte{'A', 'B', 'C', 'D', 'E', 'F'}
			expectedNodes := []*Node{left, left, left, right, right, right}

			for i, expectedChar := range expectedChars {
				// Verify current character
				if iter.getCharAt() != expectedChar {
					t.Errorf("Position %d: expected char %c, got %c", i, expectedChar, iter.getCharAt())
				}

				// Verify current node
				if iter.Node != expectedNodes[i] {
					t.Errorf("Position %d: expected node %v, got %v", i, expectedNodes[i], iter.Node)
				}

				// Move to next character
				if i < len(expectedChars)-1 {
					var ok bool
					iter, ok = iter.nextChar()
					if !ok {
						t.Errorf("nextChar() failed at position %d", i)
					}
				}
			}

			// Try one more nextChar, should succeed and reach EOF
			iter2, ok := iter.nextChar()
			if !ok {
				t.Error("Expected nextChar() to succeed when moving to EOF")
			}
			if iter2.Offset != 6 {
				t.Errorf("Expected offset 6 at EOF, got %d", iter2.Offset)
			}

			// Now try another nextChar, should fail
			_, ok = iter2.nextChar()
			if ok {
				t.Error("Expected nextChar() to fail when already at EOF")
			}
		})

		t.Run("backward character traversal across nodes", func(t *testing.T) {
			// Create rope with two nodes
			left := NewLeaf("ABC")
			right := NewLeaf("DEF")
			left.offset = 0
			right.offset = 3
			root := NewConcatNode(left, right)
			buf := &Buffer{Rope: &Rope{root: root}}

			// Start at last character
			iter := BufferIterator{
				Buffer:  buf,
				LineNum: 0,
				Offset:  5, // Position at 'F'
				Node:    right,
			}

			// Move character by character backwards using previousChar
			expectedChars := []byte{'F', 'E', 'D', 'C', 'B', 'A'}
			expectedNodes := []*Node{right, right, right, left, left, left}

			for i, expectedChar := range expectedChars {
				// Verify current character
				if iter.getCharAt() != expectedChar {
					t.Errorf("Position %d: expected char %c, got %c", i, expectedChar, iter.getCharAt())
				}

				// Verify current node
				if iter.Node != expectedNodes[i] {
					t.Errorf("Position %d: expected node %v, got %v", i, expectedNodes[i], iter.Node)
				}

				// Move to previous character
				if i < len(expectedChars)-1 {
					var ok bool
					iter, ok = iter.previousChar()
					if !ok {
						t.Errorf("previousChar() failed at position %d", i)
					}
				}
			}

			// Try one more previousChar, should fail
			_, ok := iter.previousChar()
			if ok {
				t.Error("Expected previousChar() to fail at beginning of buffer")
			}
		})
	})

	t.Run("content verification", func(t *testing.T) {
		t.Run("Line() returns correct content at each position", func(t *testing.T) {
			buf := NewBuffer("First line\nSecond line\nThird line", "test.txt", config.IndentSpaces, 4)

			testCases := []struct {
				lineNum      int
				expectedLine string
			}{
				{0, "First line"},
				{1, "Second line"},
				{2, "Third line"},
			}

			// Test forward navigation
			iter := NewBufferIterator(buf)
			for _, tc := range testCases {
				if iter.LineNum != tc.lineNum {
					t.Errorf("Expected line number %d, got %d", tc.lineNum, iter.LineNum)
				}
				if iter.Line() != tc.expectedLine {
					t.Errorf("Line %d: expected %q, got %q", tc.lineNum, tc.expectedLine, iter.Line())
				}
				iter = iter.Next()
			}

			// Test backward navigation
			iter = NewBufferIteratorAtEnd(buf)
			for i := len(testCases) - 1; i >= 0; i-- {
				iter = iter.Previous()
				tc := testCases[i]
				if iter.Line() != tc.expectedLine {
					t.Errorf("Previous to line %d: expected %q, got %q", tc.lineNum, tc.expectedLine, iter.Line())
				}
			}
		})

		t.Run("LineNumber() returns correct values", func(t *testing.T) {
			buf := NewBuffer("A\nB\nC\nD\nE", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// Verify LineNumber() matches LineNum field
			for expected := 0; expected < 5; expected++ {
				if iter.LineNumber() != expected {
					t.Errorf("Expected LineNumber() to return %d, got %d", expected, iter.LineNumber())
				}
				if iter.LineNum != expected {
					t.Errorf("Expected LineNum field to be %d, got %d", expected, iter.LineNum)
				}
				iter = iter.Next()
			}

			// At EOF
			if !iter.IsEOF() {
				t.Error("Expected to be at EOF")
			}
			if iter.LineNumber() != 5 {
				t.Errorf("Expected LineNumber() at EOF to be 5, got %d", iter.LineNumber())
			}
		})

		t.Run("Position() returns correct byte offsets", func(t *testing.T) {
			// Use specific content where we know exact byte positions
			buf := NewBuffer("AB\nCDE\nF", "test.txt", config.IndentSpaces, 4)

			testCases := []struct {
				lineNum  int
				expected int
				content  string
			}{
				{0, 0, "AB"},    // Line 0 starts at byte 0
				{1, 3, "CDE"},   // Line 1 starts at byte 3 (after "AB\n")
				{2, 7, "F"},     // Line 2 starts at byte 7 (after "AB\nCDE\n")
			}

			iter := NewBufferIterator(buf)
			for _, tc := range testCases {
				if iter.Position() != tc.expected {
					t.Errorf("Line %d: expected Position() %d, got %d", tc.lineNum, tc.expected, iter.Position())
				}
				if iter.Offset != tc.expected {
					t.Errorf("Line %d: expected Offset %d, got %d", tc.lineNum, tc.expected, iter.Offset)
				}
				if iter.Line() != tc.content {
					t.Errorf("Line %d: expected content %q, got %q", tc.lineNum, tc.content, iter.Line())
				}
				iter = iter.Next()
			}
		})

		t.Run("LineEnd() returns correct end positions", func(t *testing.T) {
			buf := NewBuffer("Hello\nWorld\nTest", "test.txt", config.IndentSpaces, 4)

			testCases := []struct {
				lineNum     int
				lineStart   int
				lineEnd     int
				content     string
			}{
				{0, 0, 5, "Hello"},   // "Hello" ends at position 5 (before \n)
				{1, 6, 11, "World"},  // "World" starts at 6, ends at 11
				{2, 12, 16, "Test"},  // "Test" starts at 12, ends at 16
			}

			iter := NewBufferIterator(buf)
			for _, tc := range testCases {
				if iter.LineNum != tc.lineNum {
					t.Errorf("Expected line %d, got %d", tc.lineNum, iter.LineNum)
				}
				if iter.Position() != tc.lineStart {
					t.Errorf("Line %d: expected start position %d, got %d", tc.lineNum, tc.lineStart, iter.Position())
				}
				if iter.LineEnd() != tc.lineEnd {
					t.Errorf("Line %d: expected LineEnd() %d, got %d", tc.lineNum, tc.lineEnd, iter.LineEnd())
				}
				if iter.Line() != tc.content {
					t.Errorf("Line %d: expected content %q, got %q", tc.lineNum, tc.content, iter.Line())
				}
				iter = iter.Next()
			}

			// Test LineEnd() at EOF
			if !iter.IsEOF() {
				t.Error("Expected to be at EOF")
			}
			if iter.LineEnd() != 16 {
				t.Errorf("Expected LineEnd() at EOF to be 16, got %d", iter.LineEnd())
			}
		})

		t.Run("content consistency after navigation", func(t *testing.T) {
			// Create buffer with known content
			content := "Line 1: Hello\nLine 2: World\nLine 3: Test\nLine 4: End"
			buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)

			// Navigate forward and collect all lines
			forwardLines := []string{}
			iter := NewBufferIterator(buf)
			for !iter.IsEOF() {
				forwardLines = append(forwardLines, iter.Line())
				iter = iter.Next()
			}

			// Navigate backward and collect all lines
			backwardLines := []string{}
			iter = NewBufferIteratorAtEnd(buf)
			for iter.LineNum > 0 {
				iter = iter.Previous()
				backwardLines = append([]string{iter.Line()}, backwardLines...)
			}

			// Verify both directions produced the same content
			if len(forwardLines) != len(backwardLines) {
				t.Errorf("Forward navigation found %d lines, backward found %d",
					len(forwardLines), len(backwardLines))
			}

			for i := range forwardLines {
				if forwardLines[i] != backwardLines[i] {
					t.Errorf("Line %d mismatch: forward=%q, backward=%q",
						i, forwardLines[i], backwardLines[i])
				}
			}

			// Verify we got the expected lines
			expectedLines := []string{
				"Line 1: Hello",
				"Line 2: World",
				"Line 3: Test",
				"Line 4: End",
			}

			for i, expected := range expectedLines {
				if i >= len(forwardLines) {
					t.Errorf("Missing line %d: expected %q", i, expected)
					continue
				}
				if forwardLines[i] != expected {
					t.Errorf("Line %d: expected %q, got %q", i, expected, forwardLines[i])
				}
			}
		})

		t.Run("LineBytes() returns correct byte slices", func(t *testing.T) {
			buf := NewBuffer("Hello\nWorld", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// First line
			bytes1 := iter.LineBytes()
			expected1 := []byte("Hello")
			if len(bytes1) != len(expected1) {
				t.Errorf("Line 0: expected %d bytes, got %d", len(expected1), len(bytes1))
			}
			for i, b := range expected1 {
				if i >= len(bytes1) || bytes1[i] != b {
					t.Errorf("Line 0: byte %d mismatch", i)
				}
			}

			// Second line
			iter = iter.Next()
			bytes2 := iter.LineBytes()
			expected2 := []byte("World")
			if len(bytes2) != len(expected2) {
				t.Errorf("Line 1: expected %d bytes, got %d", len(expected2), len(bytes2))
			}
			for i, b := range expected2 {
				if i >= len(bytes2) || bytes2[i] != b {
					t.Errorf("Line 1: byte %d mismatch", i)
				}
			}
		})

		t.Run("content with unicode characters", func(t *testing.T) {
			// Test with multi-byte unicode characters
			buf := NewBuffer("Hello 世界\nΓεια σου\n👋 Emoji", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			expectedLines := []string{
				"Hello 世界",
				"Γεια σου",
				"👋 Emoji",
			}

			for i, expected := range expectedLines {
				if iter.Line() != expected {
					t.Errorf("Line %d: expected %q, got %q", i, expected, iter.Line())
				}

				// Verify byte positions are correct (unicode chars take multiple bytes)
				lineBytes := []byte(expected)
				if iter.LineEnd() - iter.Position() != len(lineBytes) {
					t.Errorf("Line %d: byte length mismatch, expected %d bytes, got %d",
						i, len(lineBytes), iter.LineEnd() - iter.Position())
				}

				iter = iter.Next()
			}
		})

		t.Run("empty line content", func(t *testing.T) {
			buf := NewBuffer("\n\n\n", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// All lines should be empty
			for i := 0; i < 3; i++ {
				if iter.Line() != "" {
					t.Errorf("Line %d: expected empty string, got %q", i, iter.Line())
				}
				if len(iter.LineBytes()) != 0 {
					t.Errorf("Line %d: expected empty byte slice, got %d bytes", i, len(iter.LineBytes()))
				}
				iter = iter.Next()
			}
		})
	})

	t.Run("complex buffer tests", func(t *testing.T) {
		t.Run("navigation through buffer with multiple insertions", func(t *testing.T) {
			// Start with a simple buffer
			buf := NewBuffer("Line 1\nLine 2\nLine 3", "test.txt", config.IndentSpaces, 4)

			// Perform multiple insertions at different positions
			buf.Rope.Insert(7, "INSERTED-") // After "Line 1\n"
			buf.Rope.Insert(22, "-MIDDLE")  // In middle of "Line 2"
			buf.Rope.Insert(0, "START-")    // At beginning

			// Expected content: "START-Line 1\nINSERTED-Line 2-MIDDLE\nLine 3"
			expectedLines := []string{
				"START-Line 1",
				"INSERTED-Line 2-MIDDLE",
				"Line 3",
			}

			// Navigate forward and verify
			iter := NewBufferIterator(buf)
			for i, expected := range expectedLines {
				if iter.Line() != expected {
					t.Errorf("Line %d after insertions: expected %q, got %q", i, expected, iter.Line())
				}
				iter = iter.Next()
			}

			// Navigate backward and verify
			iter = NewBufferIteratorAtEnd(buf)
			for i := len(expectedLines) - 1; i >= 0; i-- {
				iter = iter.Previous()
				if iter.Line() != expectedLines[i] {
					t.Errorf("Previous to line %d: expected %q, got %q", i, expectedLines[i], iter.Line())
				}
			}
		})

		t.Run("navigation through split nodes (large buffer)", func(t *testing.T) {
			// Create a buffer that will definitely split into multiple nodes
			lines := []string{}
			for i := 0; i < 100; i++ {
				// Each line is about 60 chars, total > 6000 chars (exceeds maxLeafSize)
				lines = append(lines, strings.Repeat("X", 50) + " Line " + strconv.Itoa(i))
			}
			content := strings.Join(lines, "\n")
			buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)

			// Verify buffer has split
			if buf.Rope.root.IsLeaf() {
				t.Skip("Buffer didn't split into multiple nodes")
			}

			// Count nodes in the tree
			nodeCount := countNodes(buf.Rope.root)
			if nodeCount < 3 {
				t.Skip("Buffer didn't split enough for this test")
			}

			// Navigate forward through all lines
			iter := NewBufferIterator(buf)
			lineNum := 0
			for !iter.IsEOF() {
				expectedLine := lines[lineNum]
				if iter.Line() != expectedLine {
					t.Errorf("Line %d: expected %q, got %q", lineNum, expectedLine, iter.Line())
				}
				lineNum++
				iter = iter.Next()
			}

			// Verify we visited all lines
			if lineNum != len(lines) {
				t.Errorf("Expected to visit %d lines, visited %d", len(lines), lineNum)
			}

			// Navigate backward through all lines
			iter = NewBufferIteratorAtEnd(buf)
			for i := len(lines) - 1; i >= 0; i-- {
				iter = iter.Previous()
				if iter.Line() != lines[i] {
					t.Errorf("Previous to line %d: expected %q, got %q", i, lines[i], iter.Line())
				}
			}
		})

		t.Run("navigation through complex tree structure", func(t *testing.T) {
			// Build a complex tree by strategic insertions
			buf := NewBuffer("Initial", "test.txt", config.IndentSpaces, 4)

			// Insert at various positions to create a complex tree
			insertions := []struct {
				pos  int
				text string
			}{
				{7, strings.Repeat("A", 2000)},  // Large insertion
				{0, strings.Repeat("B", 1500)},   // At beginning
				{2500, strings.Repeat("C", 1800)}, // In middle
				{5000, strings.Repeat("D", 2200)}, // Near end
			}

			for _, ins := range insertions {
				buf.Rope.Insert(ins.pos, ins.text)
			}

			// Add newlines to create multiple lines
			// Insert newlines at regular intervals
			totalLen := buf.Rope.Length()
			newlinePositions := []int{}
			for pos := 1000; pos < totalLen; pos += 1000 {
				newlinePositions = append(newlinePositions, pos)
			}

			// Insert newlines from end to beginning to maintain positions
			for i := len(newlinePositions) - 1; i >= 0; i-- {
				buf.Rope.Insert(newlinePositions[i], "\n")
			}

			// Navigate and count lines
			iter := NewBufferIterator(buf)
			lineCount := 0
			totalBytes := 0
			for !iter.IsEOF() {
				lineLen := len(iter.Line())
				totalBytes += lineLen
				if lineCount < len(newlinePositions) {
					totalBytes++ // Account for newline
				}
				lineCount++
				iter = iter.Next()
			}

			// Verify backward navigation visits same number of lines
			backwardCount := 0
			iter = NewBufferIteratorAtEnd(buf)
			for iter.LineNum > 0 {
				iter = iter.Previous()
				backwardCount++
			}

			if backwardCount != lineCount {
				t.Errorf("Forward found %d lines, backward found %d", lineCount, backwardCount)
			}
		})

		t.Run("iterator stability after buffer modifications", func(t *testing.T) {
			// Create initial buffer
			buf := NewBuffer("Line 1\nLine 2\nLine 3", "test.txt", config.IndentSpaces, 4)

			// Create iterators at different positions
			iter1 := NewBufferIterator(buf)           // At line 0
			iter2 := NewBufferIterator(buf).Next()    // At line 1
			iter3 := NewBufferIteratorAtEnd(buf)      // At EOF

			// Store original values
			orig1 := struct{ line string; offset int }{iter1.Line(), iter1.Offset}
			orig2 := struct{ line string; offset int }{iter2.Line(), iter2.Offset}
			_ = iter3 // We're just testing iter1 and iter2 for now

			// Perform insertion after all iterators
			buf.Rope.Insert(buf.Rope.Length(), "\nLine 4")

			// Verify iterators still return same content
			// (They should be unaffected by insertions after their position)
			if iter1.Line() != orig1.line {
				t.Errorf("Iterator 1 line changed: was %q, now %q", orig1.line, iter1.Line())
			}
			if iter2.Line() != orig2.line {
				t.Errorf("Iterator 2 line changed: was %q, now %q", orig2.line, iter2.Line())
			}

			// Note: Node pointers might change due to tree rebalancing,
			// but the content and offsets should remain valid
		})

		t.Run("mixed navigation patterns", func(t *testing.T) {
			buf := NewBuffer("A\nB\nC\nD\nE\nF\nG", "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)

			// Complex navigation pattern: forward, forward, back, forward, back, back
			expected := []struct {
				op       string
				lineNum  int
				content  string
			}{
				{"start", 0, "A"},
				{"next", 1, "B"},
				{"next", 2, "C"},
				{"prev", 1, "B"},
				{"next", 2, "C"},
				{"prev", 1, "B"},
				{"prev", 0, "A"},
			}

			for _, exp := range expected {
				if exp.op == "next" {
					iter = iter.Next()
				} else if exp.op == "prev" {
					iter = iter.Previous()
				}

				if iter.LineNum != exp.lineNum {
					t.Errorf("After %s: expected line %d, got %d", exp.op, exp.lineNum, iter.LineNum)
				}
				if iter.Line() != exp.content {
					t.Errorf("After %s: expected content %q, got %q", exp.op, exp.content, iter.Line())
				}
			}
		})

		t.Run("navigation with very long lines", func(t *testing.T) {
			// Create buffer with very long lines that might span multiple nodes
			longLine1 := strings.Repeat("A", 3000)
			longLine2 := strings.Repeat("B", 3500)
			longLine3 := strings.Repeat("C", 2800)

			content := longLine1 + "\n" + longLine2 + "\n" + longLine3
			buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)

			// Navigate forward
			iter := NewBufferIterator(buf)

			// Line 0
			if len(iter.Line()) != 3000 {
				t.Errorf("Line 0: expected length 3000, got %d", len(iter.Line()))
			}
			if iter.Line() != longLine1 {
				t.Error("Line 0 content mismatch")
			}

			// Line 1
			iter = iter.Next()
			if len(iter.Line()) != 3500 {
				t.Errorf("Line 1: expected length 3500, got %d", len(iter.Line()))
			}
			if iter.Line() != longLine2 {
				t.Error("Line 1 content mismatch")
			}

			// Line 2
			iter = iter.Next()
			if len(iter.Line()) != 2800 {
				t.Errorf("Line 2: expected length 2800, got %d", len(iter.Line()))
			}
			if iter.Line() != longLine3 {
				t.Error("Line 2 content mismatch")
			}

			// Navigate backward and verify
			iter = iter.Previous()
			if iter.Line() != longLine2 {
				t.Error("Previous from line 2: content mismatch")
			}

			iter = iter.Previous()
			if iter.Line() != longLine1 {
				t.Error("Previous from line 1: content mismatch")
			}
		})
	})
}

// Helper function to count nodes in the tree
func countNodes(node *Node) int {
	if node == nil {
		return 0
	}
	return 1 + countNodes(node.left) + countNodes(node.right)
}

