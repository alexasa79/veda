package buffer

import (
	"strings"
	"testing"
)

func TestFindNodeAtLine(t *testing.T) {
	t.Run("simple single node cases", func(t *testing.T) {
		testCases := []struct {
			name        string
			text        string
			lineNum     int
			expectNode  bool
			expectOffset int
		}{
			{"empty buffer line 0", "", 0, true, 0},
			{"empty buffer line 1", "", 1, false, 0},
			{"single line no newline", "hello", 0, true, 0},
			{"single line no newline line 1", "hello", 1, false, 0},
			{"single line with newline", "hello\n", 0, true, 0},
			{"single line with newline line 1", "hello\n", 1, true, 6},
			{"two lines", "hello\nworld", 0, true, 0},
			{"two lines line 1", "hello\nworld", 1, true, 6},
			{"two lines line 2", "hello\nworld", 2, false, 0},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rope := NewRopeFromString(tc.text)
				node, offsetInNode := rope.findNodeAtLine(tc.lineNum)

				if tc.expectNode {
					if node == nil {
						t.Errorf("Expected to find node for line %d, got nil", tc.lineNum)
						return
					}
					if offsetInNode != tc.expectOffset {
						t.Errorf("Expected offset %d in node, got %d", tc.expectOffset, offsetInNode)
					}
				} else {
					if node != nil {
						t.Errorf("Expected nil node for line %d, got node with offset %d", tc.lineNum, offsetInNode)
					}
				}
			})
		}
	})

	t.Run("multi-line cases", func(t *testing.T) {
		text := "line0\nline1\nline2\nline3\n"
		rope := NewRopeFromString(text)

		testCases := []struct {
			lineNum      int
			expectOffset int
			expectLine   string
		}{
			{0, 0, "line0"},
			{1, 6, "line1"},
			{2, 12, "line2"},
			{3, 18, "line3"},
			{4, 24, ""}, // Empty line after last newline
		}

		for _, tc := range testCases {
			t.Run(strings.Join([]string{"line", string(rune('0'+tc.lineNum))}, ""), func(t *testing.T) {
				node, offsetInNode := rope.findNodeAtLine(tc.lineNum)

				if node == nil {
					if tc.lineNum < 5 { // Lines 0-4 should exist
						t.Errorf("Expected to find node for line %d, got nil", tc.lineNum)
					}
					return
				}

				// Calculate absolute offset
				absoluteOffset := node.offset + offsetInNode
				if absoluteOffset != tc.expectOffset {
					t.Errorf("Line %d: expected absolute offset %d, got %d", tc.lineNum, tc.expectOffset, absoluteOffset)
				}

				// Verify we can extract the correct line content
				if tc.lineNum < 4 { // Don't test line content for empty line
					lineEnd := absoluteOffset
					for lineEnd < rope.Length() && rope.ByteAt(lineEnd) != '\n' {
						lineEnd++
					}
					actualLine := rope.Extract(absoluteOffset, lineEnd)
					if actualLine != tc.expectLine {
						t.Errorf("Line %d: expected content %q, got %q", tc.lineNum, tc.expectLine, actualLine)
					}
				}
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("negative line number", func(t *testing.T) {
			rope := NewRopeFromString("hello\nworld")
			node, offset := rope.findNodeAtLine(-1)
			if node != nil {
				t.Errorf("Expected nil node for negative line number, got node with offset %d", offset)
			}
		})

		t.Run("line beyond end", func(t *testing.T) {
			rope := NewRopeFromString("hello\nworld")
			node, offset := rope.findNodeAtLine(10)
			if node != nil {
				t.Errorf("Expected nil node for line beyond end, got node with offset %d", offset)
			}
		})

		t.Run("empty lines", func(t *testing.T) {
			rope := NewRopeFromString("a\n\n\nb")

			// Line 0: "a"
			node, offset := rope.findNodeAtLine(0)
			if node == nil || node.offset + offset != 0 {
				t.Errorf("Line 0: expected offset 0, got node=%v offset=%d", node, offset)
			}

			// Line 1: "" (empty)
			node, offset = rope.findNodeAtLine(1)
			if node == nil || node.offset + offset != 2 {
				t.Errorf("Line 1: expected offset 2, got node=%v offset=%d", node, offset)
			}

			// Line 2: "" (empty)
			node, offset = rope.findNodeAtLine(2)
			if node == nil || node.offset + offset != 3 {
				t.Errorf("Line 2: expected offset 3, got node=%v offset=%d", node, offset)
			}

			// Line 3: "b"
			node, offset = rope.findNodeAtLine(3)
			if node == nil || node.offset + offset != 4 {
				t.Errorf("Line 3: expected offset 4, got node=%v offset=%d", node, offset)
			}
		})
	})
}

func TestFindNodeAtLineWithComplexTree(t *testing.T) {
	// Create a rope that will have multiple nodes
	// Build it by inserting large chunks to force splits
	rope := NewRopeFromString("initial")

	// Insert large text to force tree structure
	largeText := strings.Repeat("X", 2000) + "\n" + strings.Repeat("Y", 2000) + "\n"
	rope.Insert(7, largeText) // After "initial"

	// Add more content to create a complex tree
	moreText := strings.Repeat("Z", 1500) + "\n" + "final line"
	rope.Insert(rope.Length(), moreText)

	// Verify the rope has multiple nodes
	if rope.root.IsLeaf() {
		t.Skip("Rope didn't split into multiple nodes, skipping complex tree test")
	}

	// Test finding lines in different parts of the tree
	testCases := []struct {
		lineNum int
		desc    string
	}{
		{0, "first line"},
		{1, "line in first large chunk"},
		{2, "line in second large chunk"},
		{3, "line in final chunk"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			node, offsetInNode := rope.findNodeAtLine(tc.lineNum)

			if node == nil {
				t.Errorf("Failed to find node for line %d", tc.lineNum)
				return
			}

			if !node.IsLeaf() {
				t.Errorf("findNodeAtLine should return a leaf node, got internal node")
			}

			// Verify the offset is within the node bounds
			if offsetInNode < 0 || offsetInNode > len(node.text) {
				t.Errorf("Offset %d is out of bounds for node with text length %d", offsetInNode, len(node.text))
			}

			// Verify we can extract a valid line from this position
			absoluteOffset := node.offset + offsetInNode
			if absoluteOffset > rope.Length() {
				t.Errorf("Absolute offset %d exceeds rope length %d", absoluteOffset, rope.Length())
			}
		})
	}
}

func TestFindLineStartInLeaf(t *testing.T) {
	rope := NewRopeFromString("dummy") // We just need a rope instance

	t.Run("single line leaf", func(t *testing.T) {
		node := NewLeaf("hello world")

		// Line 0 should start at offset 0
		resultNode, offset := rope.findLineStartInLeaf(node, 0)
		if resultNode != node || offset != 0 {
			t.Errorf("Line 0: expected node=%v offset=0, got node=%v offset=%d", node, resultNode, offset)
		}

		// Line 1 doesn't exist, should return end of node
		resultNode, offset = rope.findLineStartInLeaf(node, 1)
		if resultNode != node || offset != len(node.text) {
			t.Errorf("Line 1: expected node=%v offset=%d, got node=%v offset=%d",
				node, len(node.text), resultNode, offset)
		}
	})

	t.Run("multi-line leaf", func(t *testing.T) {
		node := NewLeaf("line0\nline1\nline2")

		testCases := []struct {
			lineNum      int
			expectOffset int
		}{
			{0, 0},  // Start of "line0"
			{1, 6},  // Start of "line1" (after "line0\n")
			{2, 12}, // Start of "line2" (after "line0\nline1\n")
			{3, 17}, // Beyond end (length of text)
		}

		for _, tc := range testCases {
			_, offset := rope.findLineStartInLeaf(node, tc.lineNum)
			if offset != tc.expectOffset {
				t.Errorf("Line %d: expected offset %d, got %d", tc.lineNum, tc.expectOffset, offset)
			}
		}
	})

	t.Run("leaf with empty lines", func(t *testing.T) {
		node := NewLeaf("a\n\n\nb")

		testCases := []struct {
			lineNum      int
			expectOffset int
		}{
			{0, 0}, // "a"
			{1, 2}, // First empty line (after "a\n")
			{2, 3}, // Second empty line (after "a\n\n")
			{3, 4}, // "b" (after "a\n\n\n")
		}

		for _, tc := range testCases {
			_, offset := rope.findLineStartInLeaf(node, tc.lineNum)
			if offset != tc.expectOffset {
				t.Errorf("Line %d: expected offset %d, got %d", tc.lineNum, tc.expectOffset, offset)
			}
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		node := NewLeaf("test")

		// Negative line number
		_, offset := rope.findLineStartInLeaf(node, -1)
		if offset != 0 {
			t.Errorf("Negative line: expected offset=0, got offset=%d", offset)
		}

		// Non-leaf node should return early
		internal := NewConcatNode(NewLeaf("left"), NewLeaf("right"))
		_, offset = rope.findLineStartInLeaf(internal, 0)
		if offset != 0 {
			t.Errorf("Non-leaf: expected early return with offset 0, got offset %d", offset)
		}
	})
}

func TestFindNodeAtLineRecursive(t *testing.T) {
	t.Run("tree navigation", func(t *testing.T) {
		// Create a specific tree structure: ((A\n, B\nC\n), (D, E\n))
		nodeA := NewLeaf("A\n")      // 1 newline
		nodeB := NewLeaf("B\nC\n")   // 2 newlines
		nodeD := NewLeaf("D")        // 0 newlines
		nodeE := NewLeaf("E\n")      // 1 newline

		// Set offsets manually for this test
		nodeA.offset = 0
		nodeB.offset = 2  // After "A\n"
		nodeD.offset = 6  // After "A\nB\nC\n"
		nodeE.offset = 7  // After "A\nB\nC\nD"

		leftSubtree := NewConcatNode(nodeA, nodeB)   // 3 newlines total
		rightSubtree := NewConcatNode(nodeD, nodeE)  // 1 newline total
		root := NewConcatNode(leftSubtree, rightSubtree) // 4 newlines total

		rope := &Rope{root: root}

		testCases := []struct {
			lineNum      int
			expectNode   *Node
			expectOffset int
		}{
			{0, nodeA, 0}, // Line 0: "A" in nodeA
			{1, nodeB, 0}, // Line 1: "B" in nodeB
			{2, nodeB, 2}, // Line 2: "C" in nodeB (after "B\n")
			{3, nodeD, 0}, // Line 3: "D" in nodeD
		}

		for _, tc := range testCases {
			t.Run(strings.Join([]string{"line", string(rune('0'+tc.lineNum))}, ""), func(t *testing.T) {
				resultNode, offset := rope.findNodeAtLineRecursive(root, tc.lineNum)

				if resultNode != tc.expectNode {
					t.Errorf("Line %d: expected node %v, got %v", tc.lineNum, tc.expectNode, resultNode)
				}
				if offset != tc.expectOffset {
					t.Errorf("Line %d: expected offset %d, got %d", tc.lineNum, tc.expectOffset, offset)
				}
			})
		}
	})

	t.Run("nil node handling", func(t *testing.T) {
		rope := &Rope{root: nil}
		node, offset := rope.findNodeAtLineRecursive(nil, 0)
		if node != nil || offset != 0 {
			t.Errorf("Nil node: expected nil node and 0 offset, got node=%v offset=%d", node, offset)
		}
	})
}

// Benchmark to verify the logarithmic performance improvement
func BenchmarkFindNodeAtLine(b *testing.B) {
	// Create a large buffer with many lines
	lines := make([]string, 10000)
	for i := range lines {
		lines[i] = strings.Repeat("x", 50) // 50 chars per line
	}
	content := strings.Join(lines, "\n")
	rope := NewRopeFromString(content)

	b.Run("logarithmic search", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lineNum := i % 10000
			rope.findNodeAtLine(lineNum)
		}
	})

	// For comparison, implement a linear search
	linearSearch := func(rope *Rope, targetLine int) int {
		offset := 0
		currentLine := 0
		length := rope.Length()

		for offset < length && currentLine < targetLine {
			if rope.ByteAt(offset) == '\n' {
				currentLine++
			}
			offset++
		}
		return offset
	}

	b.Run("linear search", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lineNum := i % 10000
			linearSearch(rope, lineNum)
		}
	})
}
