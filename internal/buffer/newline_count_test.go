package buffer

import (
	"strings"
	"testing"
)

func TestNewlineCountInfrastructure(t *testing.T) {
	t.Run("countNewlines function", func(t *testing.T) {
		testCases := []struct {
			name     string
			text     string
			expected int
		}{
			{"empty string", "", 0},
			{"no newlines", "hello world", 0},
			{"single newline", "hello\n", 1},
			{"multiple newlines", "line1\nline2\nline3\n", 3},
			{"consecutive newlines", "\n\n\n", 3},
			{"mixed content", "a\nb\n\nc\n", 4},
			{"newline at start", "\nhello", 1},
			{"newline in middle", "hel\nlo", 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := countNewlines(tc.text)
				if result != tc.expected {
					t.Errorf("countNewlines(%q) = %d, expected %d", tc.text, result, tc.expected)
				}
			})
		}
	})

	t.Run("NewLeaf newline count", func(t *testing.T) {
		testCases := []struct {
			name     string
			text     string
			expected int
		}{
			{"empty leaf", "", 0},
			{"no newlines", "hello world", 0},
			{"single newline", "hello\nworld", 1},
			{"multiple newlines", "a\nb\nc\n", 3},
			{"only newlines", "\n\n", 2},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				node := NewLeaf(tc.text)
				if node.newlineCount != tc.expected {
					t.Errorf("NewLeaf(%q).newlineCount = %d, expected %d", tc.text, node.newlineCount, tc.expected)
				}
				if node.weight != len(tc.text) {
					t.Errorf("NewLeaf(%q).weight = %d, expected %d", tc.text, node.weight, len(tc.text))
				}
			})
		}
	})

	t.Run("NewConcatNode newline count aggregation", func(t *testing.T) {
		t.Run("both children non-nil", func(t *testing.T) {
			left := NewLeaf("hello\nworld\n") // 2 newlines
			right := NewLeaf("foo\nbar")      // 1 newline

			concat := NewConcatNode(left, right)

			expectedNewlines := 3
			expectedWeight := len("hello\nworld\n") + len("foo\nbar")

			if concat.newlineCount != expectedNewlines {
				t.Errorf("NewConcatNode newlineCount = %d, expected %d", concat.newlineCount, expectedNewlines)
			}
			if concat.weight != expectedWeight {
				t.Errorf("NewConcatNode weight = %d, expected %d", concat.weight, expectedWeight)
			}
		})

		t.Run("left child nil", func(t *testing.T) {
			right := NewLeaf("foo\nbar\n") // 2 newlines

			concat := NewConcatNode(nil, right)

			if concat.newlineCount != 2 {
				t.Errorf("NewConcatNode with nil left: newlineCount = %d, expected 2", concat.newlineCount)
			}
		})

		t.Run("right child nil", func(t *testing.T) {
			left := NewLeaf("hello\nworld") // 1 newline

			concat := NewConcatNode(left, nil)

			if concat.newlineCount != 1 {
				t.Errorf("NewConcatNode with nil right: newlineCount = %d, expected 1", concat.newlineCount)
			}
		})

		t.Run("both children nil", func(t *testing.T) {
			concat := NewConcatNode(nil, nil)

			if concat.newlineCount != 0 {
				t.Errorf("NewConcatNode with both nil: newlineCount = %d, expected 0", concat.newlineCount)
			}
		})

		t.Run("nested concatenation", func(t *testing.T) {
			// Create a tree: ((A, B), (C, D))
			nodeA := NewLeaf("A\n")     // 1 newline
			nodeB := NewLeaf("B\nB2\n") // 2 newlines
			nodeC := NewLeaf("C")       // 0 newlines
			nodeD := NewLeaf("D\nD2")   // 1 newline

			leftSubtree := NewConcatNode(nodeA, nodeB)       // 3 newlines
			rightSubtree := NewConcatNode(nodeC, nodeD)      // 1 newline
			root := NewConcatNode(leftSubtree, rightSubtree) // 4 newlines total

			if leftSubtree.newlineCount != 3 {
				t.Errorf("Left subtree newlineCount = %d, expected 3", leftSubtree.newlineCount)
			}
			if rightSubtree.newlineCount != 1 {
				t.Errorf("Right subtree newlineCount = %d, expected 1", rightSubtree.newlineCount)
			}
			if root.newlineCount != 4 {
				t.Errorf("Root newlineCount = %d, expected 4", root.newlineCount)
			}
		})
	})

	t.Run("rope newline count consistency", func(t *testing.T) {
		testCases := []struct {
			name string
			text string
		}{
			{"simple text", "hello\nworld\ntest"},
			{"empty", ""},
			{"only newlines", "\n\n\n"},
			{"no newlines", "hello world"},
			{"complex text", "line1\n\nline3\nline4\n\n\nline7"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rope := NewRopeFromString(tc.text)

				// Count newlines manually
				expectedCount := strings.Count(tc.text, "\n")

				// Check rope's newline count
				if rope.root.newlineCount != expectedCount {
					t.Errorf("Rope newlineCount = %d, expected %d for text %q",
						rope.root.newlineCount, expectedCount, tc.text)
				}
			})
		}
	})

	t.Run("newline count invariants", func(t *testing.T) {
		// Test that newline counts are maintained correctly in a complex tree
		rope := NewRopeFromString("initial\ntext\n")

		// Verify initial state
		expectedInitial := 2
		if rope.root.newlineCount != expectedInitial {
			t.Errorf("Initial rope newlineCount = %d, expected %d", rope.root.newlineCount, expectedInitial)
		}

		// Insert text with newlines
		rope.Insert(0, "prefix\n")
		expectedAfterInsert := 3
		if rope.root.newlineCount != expectedAfterInsert {
			t.Errorf("After insert newlineCount = %d, expected %d", rope.root.newlineCount, expectedAfterInsert)
		}

		// Verify the actual content matches
		actualContent := rope.String()
		actualNewlines := strings.Count(actualContent, "\n")
		if rope.root.newlineCount != actualNewlines {
			t.Errorf("Newline count %d doesn't match actual newlines %d in content %q",
				rope.root.newlineCount, actualNewlines, actualContent)
		}
	})
}

// Helper function to verify newline count consistency in a tree
func verifyNewlineCountConsistency(t *testing.T, node *Node) {
	if node == nil {
		return
	}

	if node.IsLeaf() {
		// For leaf nodes, verify count matches actual newlines in text
		actualCount := countNewlines(node.text)
		if node.newlineCount != actualCount {
			t.Errorf("Leaf node newlineCount = %d, but actual count in text %q is %d",
				node.newlineCount, node.text, actualCount)
		}
	} else {
		// For internal nodes, verify count equals sum of children
		leftCount := 0
		rightCount := 0

		if node.left != nil {
			leftCount = node.left.newlineCount
			verifyNewlineCountConsistency(t, node.left)
		}
		if node.right != nil {
			rightCount = node.right.newlineCount
			verifyNewlineCountConsistency(t, node.right)
		}

		expectedCount := leftCount + rightCount
		if node.newlineCount != expectedCount {
			t.Errorf("Internal node newlineCount = %d, but sum of children is %d (left=%d, right=%d)",
				node.newlineCount, expectedCount, leftCount, rightCount)
		}
	}
}

func TestNewlineCountConsistencyAfterOperations(t *testing.T) {
	t.Run("consistency after insertions", func(t *testing.T) {
		rope := NewRopeFromString("line1\nline2\n")

		insertions := []struct {
			pos  int
			text string
		}{
			{0, "start\n"},
			{6, "mid\n"},
			{rope.Length(), "end\n"},
		}

		for i, ins := range insertions {
			rope.Insert(ins.pos, ins.text)

			// Verify consistency after each insertion
			verifyNewlineCountConsistency(t, rope.root)

			// Verify total count matches actual content
			actualContent := rope.String()
			actualNewlines := strings.Count(actualContent, "\n")
			if rope.root.newlineCount != actualNewlines {
				t.Errorf("After insertion %d: newlineCount = %d, actual = %d, content = %q",
					i, rope.root.newlineCount, actualNewlines, actualContent)
			}
		}
	})

	t.Run("consistency after deletions", func(t *testing.T) {
		rope := NewRopeFromString("line1\nline2\nline3\nline4\n")

		deletions := []struct {
			start int
			end   int
		}{
			{6, 12}, // Delete "line2\n"
			{0, 6},  // Delete "line1\n"
			{6, 12}, // Delete "line4\n"
		}

		for i, del := range deletions {
			if del.start < rope.Length() && del.end <= rope.Length() {
				rope.Delete(del.start, del.end)

				// Verify consistency after each deletion
				verifyNewlineCountConsistency(t, rope.root)

				// Verify total count matches actual content
				actualContent := rope.String()
				actualNewlines := strings.Count(actualContent, "\n")
				if rope.root.newlineCount != actualNewlines {
					t.Errorf("After deletion %d: newlineCount = %d, actual = %d, content = %q",
						i, rope.root.newlineCount, actualNewlines, actualContent)
				}
			}
		}
	})
}
