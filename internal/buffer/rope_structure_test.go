package buffer

import (
	"strings"
	"testing"
	"veda/internal/config"
)

// TestNodeWeights verifies that node weights are correctly maintained
func TestNodeWeights(t *testing.T) {
	t.Run("leaf node weight", func(t *testing.T) {
		leaf := NewLeaf("Hello")
		if leaf.weight != 5 {
			t.Errorf("Expected leaf weight to be 5, got %d", leaf.weight)
		}
	})

	t.Run("concat node weight", func(t *testing.T) {
		left := NewLeaf("Hello")
		right := NewLeaf("World")
		concat := NewConcatNode(left, right)

		if concat.weight != 10 {
			t.Errorf("Expected concat weight to be 10, got %d", concat.weight)
		}
	})

	t.Run("nested concat node weights", func(t *testing.T) {
		// Build a tree: ((Hello)(World))((Test)(Rope))
		leaf1 := NewLeaf("Hello") // 5
		leaf2 := NewLeaf("World") // 5
		leaf3 := NewLeaf("Test")  // 4
		leaf4 := NewLeaf("Rope")  // 4

		concat1 := NewConcatNode(leaf1, leaf2)  // 10
		concat2 := NewConcatNode(leaf3, leaf4)  // 8
		root := NewConcatNode(concat1, concat2) // 18

		if root.weight != 18 {
			t.Errorf("Expected root weight to be 18, got %d", root.weight)
		}
		if concat1.weight != 10 {
			t.Errorf("Expected concat1 weight to be 10, got %d", concat1.weight)
		}
		if concat2.weight != 8 {
			t.Errorf("Expected concat2 weight to be 8, got %d", concat2.weight)
		}
	})
}

// TestNodeParentPointers verifies that parent pointers are correctly maintained
func TestNodeParentPointers(t *testing.T) {
	t.Run("leaf nodes have no parent initially", func(t *testing.T) {
		leaf := NewLeaf("Hello")
		if leaf.parent != nil {
			t.Error("Expected new leaf to have nil parent")
		}
	})

	t.Run("concat node sets parent pointers", func(t *testing.T) {
		left := NewLeaf("Hello")
		right := NewLeaf("World")
		concat := NewConcatNode(left, right)

		if left.parent != concat {
			t.Error("Expected left child's parent to point to concat node")
		}
		if right.parent != concat {
			t.Error("Expected right child's parent to point to concat node")
		}
		if concat.parent != nil {
			t.Error("Expected concat node to have nil parent")
		}
	})

	t.Run("nested concat maintains parent pointers", func(t *testing.T) {
		leaf1 := NewLeaf("A")
		leaf2 := NewLeaf("B")
		leaf3 := NewLeaf("C")
		leaf4 := NewLeaf("D")

		concat1 := NewConcatNode(leaf1, leaf2)
		concat2 := NewConcatNode(leaf3, leaf4)
		root := NewConcatNode(concat1, concat2)

		// Check all parent pointers
		if concat1.parent != root {
			t.Error("concat1 parent should be root")
		}
		if concat2.parent != root {
			t.Error("concat2 parent should be root")
		}
		if leaf1.parent != concat1 {
			t.Error("leaf1 parent should be concat1")
		}
		if leaf2.parent != concat1 {
			t.Error("leaf2 parent should be concat1")
		}
		if leaf3.parent != concat2 {
			t.Error("leaf3 parent should be concat2")
		}
		if leaf4.parent != concat2 {
			t.Error("leaf4 parent should be concat2")
		}
	})
}

// TestInsertMaintainsStructure verifies that insert operations maintain weights and parent pointers
func TestInsertMaintainsStructure(t *testing.T) {
	t.Run("insert updates weights", func(t *testing.T) {
		r := NewRopeFromString("Hello")
		initialWeight := r.root.weight

		r.Insert(5, " World")

		if r.root.weight != initialWeight+6 {
			t.Errorf("Expected weight to increase by 6, got %d", r.root.weight)
		}
	})

    t.Run("split maintains parent pointers", func(t *testing.T) {
        // Create a rope that will split when we insert: ensure combined size > maxLeafSize
        baseLen := maxLeafSize - 800
        insertLen := 2000
        text := strings.Repeat("a", baseLen)
        r := NewRopeFromString(text)

        // Insert to trigger split
        r.Insert(baseLen/2, strings.Repeat("b", insertLen))

        // Root should now be internal node
        if r.root.IsLeaf() {
            t.Fatal("Expected root to be internal after split")
        }

        // Check parent pointers
        if r.root.left.parent != r.root {
            t.Error("Left child's parent should point to root")
        }
        if r.root.right.parent != r.root {
            t.Error("Right child's parent should point to root")
        }

        // Check weights
        expectedWeight := baseLen + insertLen
        if r.root.weight != expectedWeight {
            t.Errorf("Expected root weight %d, got %d", expectedWeight, r.root.weight)
        }
    })

	t.Run("insert maintains weights through tree", func(t *testing.T) {
		// Build a structured rope
		r := &Rope{root: NewConcatNode(NewLeaf("Hello"), NewLeaf("World"))}

		// Insert in left subtree
		r.Insert(2, "XXX")

		// Check that all weights are updated
		if r.root.weight != 13 { // "HeXXXllo" + "World" = 8 + 5 = 13
			t.Errorf("Expected root weight 13, got %d", r.root.weight)
		}
		if r.root.left.weight != 8 { // "HeXXXllo"
			t.Errorf("Expected left weight 8, got %d", r.root.left.weight)
		}
		if r.root.right.weight != 5 { // "World"
			t.Errorf("Expected right weight 5, got %d", r.root.right.weight)
		}
	})
}

// TestIteratorNodeField verifies that the iterator correctly tracks the node containing the current line
func TestIteratorNodeField(t *testing.T) {
	t.Run("iterator at beginning has correct node", func(t *testing.T) {
		buf := NewBuffer("Hello\nWorld", "test.txt", config.IndentSpaces, 4)
		iter := NewBufferIterator(buf)

		if iter.Node == nil {
			t.Fatal("Expected iterator to have non-nil node")
		}

		// The node should contain the text that includes the first line
		if !iter.Node.IsLeaf() {
			t.Error("Expected iterator node to be a leaf")
		}

		// The node's text should contain "Hello"
		if !strings.Contains(iter.Node.text, "Hello") {
			t.Errorf("Expected node to contain 'Hello', got %q", iter.Node.text)
		}
	})

	t.Run("iterator maintains correct node on Next", func(t *testing.T) {
		// Create a buffer that will have multiple nodes
		text := strings.Repeat("Line\n", 1000) // This should create multiple nodes
		buf := NewBuffer(text, "test.txt", config.IndentSpaces, 4)

		iter := NewBufferIterator(buf)

		// Move forward several times
		for i := 0; i < 10; i++ {
			iter = iter.Next()
			if !iter.IsEOF() && iter.Node == nil {
				t.Errorf("Iterator at line %d has nil node", iter.LineNum)
			}
		}
	})

	t.Run("iterator node points to last leaf at EOF", func(t *testing.T) {
		buf := NewBuffer("Hello", "test.txt", config.IndentSpaces, 4)
		iter := NewBufferIteratorAtEnd(buf)

		if iter.Node == nil {
			t.Fatal("Expected EOF iterator to have non-nil node")
		}

		if !iter.Node.IsLeaf() {
			t.Error("Expected EOF iterator to point to a leaf node")
		}

		// The node should be the last (rightmost) node in the rope
		if iter.Node.text != "Hello" {
			t.Errorf("Expected EOF iterator to point to node with 'Hello', got %q", iter.Node.text)
		}
	})

	t.Run("iterator finds correct node for offset", func(t *testing.T) {
		// Create a rope with known structure
		left := NewLeaf("Hello\n")
		right := NewLeaf("World\n")
		left.offset = 0
		right.offset = 6
		root := NewConcatNode(left, right)
		r := &Rope{root: root}
		buf := &Buffer{Rope: r}

		iter := NewBufferIterator(buf)

		// First line should be in left node
		if iter.Node != r.root.left {
			t.Error("Expected iterator to point to left node for first line")
		}

		// Move to second line
		iter = iter.Next()

		// Second line should be in right node
		if iter.Node != r.root.right {
			t.Error("Expected iterator to point to right node for second line")
		}
	})
}

// TestByteAtNavigation verifies that navigation through the rope structure works correctly
func TestByteAtNavigation(t *testing.T) {
	t.Run("byteAt navigates correctly through tree", func(t *testing.T) {
		// Create rope: "Hello" + "World"
		left := NewLeaf("Hello")
		right := NewLeaf("World")
		root := NewConcatNode(left, right)
		r := &Rope{root: root}

		// Test positions in left subtree
		if r.ByteAt(0) != 'H' {
			t.Errorf("Expected 'H' at position 0, got %c", r.ByteAt(0))
		}
		if r.ByteAt(4) != 'o' {
			t.Errorf("Expected 'o' at position 4, got %c", r.ByteAt(4))
		}

		// Test positions in right subtree
		if r.ByteAt(5) != 'W' {
			t.Errorf("Expected 'W' at position 5, got %c", r.ByteAt(5))
		}
		if r.ByteAt(9) != 'd' {
			t.Errorf("Expected 'd' at position 9, got %c", r.ByteAt(9))
		}
	})
}

// TestExtract verifies that the Extract method works correctly
func TestExtract(t *testing.T) {
	t.Run("extract from single leaf", func(t *testing.T) {
		r := NewRopeFromString("Hello World")

		// Extract whole string
		if got := r.Extract(0, 11); got != "Hello World" {
			t.Errorf("Expected 'Hello World', got %q", got)
		}

		// Extract substring
		if got := r.Extract(6, 11); got != "World" {
			t.Errorf("Expected 'World', got %q", got)
		}

		// Extract middle
		if got := r.Extract(3, 8); got != "lo Wo" {
			t.Errorf("Expected 'lo Wo', got %q", got)
		}
	})

	t.Run("extract from split rope", func(t *testing.T) {
		// Create rope with structure
		left := NewLeaf("Hello ")
		right := NewLeaf("World!")
		root := NewConcatNode(left, right)
		r := &Rope{root: root}

		// Extract spanning both nodes
		if got := r.Extract(3, 9); got != "lo Wor" {
			t.Errorf("Expected 'lo Wor', got %q", got)
		}

		// Extract from left only
		if got := r.Extract(0, 5); got != "Hello" {
			t.Errorf("Expected 'Hello', got %q", got)
		}

		// Extract from right only
		if got := r.Extract(6, 12); got != "World!" {
			t.Errorf("Expected 'World!', got %q", got)
		}
	})

	t.Run("extract with complex structure", func(t *testing.T) {
		// Build a deeper tree
		leaf1 := NewLeaf("The ")
		leaf2 := NewLeaf("quick ")
		leaf3 := NewLeaf("brown ")
		leaf4 := NewLeaf("fox")

		concat1 := NewConcatNode(leaf1, leaf2)
		concat2 := NewConcatNode(leaf3, leaf4)
		root := NewConcatNode(concat1, concat2)
		r := &Rope{root: root}

		// Extract across multiple nodes
		if got := r.Extract(4, 16); got != "quick brown " {
			t.Errorf("Expected 'quick brown ', got %q", got)
		}
	})

	t.Run("extract edge cases", func(t *testing.T) {
		r := NewRopeFromString("Test")

		// Empty range
		if got := r.Extract(2, 2); got != "" {
			t.Errorf("Expected empty string, got %q", got)
		}

		// Invalid range
		if got := r.Extract(3, 1); got != "" {
			t.Errorf("Expected empty string for invalid range, got %q", got)
		}

		// Out of bounds
		if got := r.Extract(10, 20); got != "" {
			t.Errorf("Expected empty string for out of bounds, got %q", got)
		}
	})
}

// TestDelete tests the rope Delete operation
func TestDelete(t *testing.T) {
	t.Run("delete from single leaf", func(t *testing.T) {
		r := NewRopeFromString("Hello World")

		// Delete "World"
		r.Delete(6, 11)
		if got := r.String(); got != "Hello " {
			t.Errorf("Expected 'Hello ', got %q", got)
		}

		// Delete space
		r.Delete(5, 6)
		if got := r.String(); got != "Hello" {
			t.Errorf("Expected 'Hello', got %q", got)
		}

		// Delete from beginning
		r.Delete(0, 2)
		if got := r.String(); got != "llo" {
			t.Errorf("Expected 'llo', got %q", got)
		}
	})

	t.Run("delete entire content", func(t *testing.T) {
		r := NewRopeFromString("Test String")
		r.Delete(0, r.Length())
		if got := r.String(); got != "" {
			t.Errorf("Expected empty string, got %q", got)
		}
	})

	t.Run("delete from split rope", func(t *testing.T) {
		// Create a rope that will split into multiple nodes
		content := strings.Repeat("Hello World ", 100)
		r := NewRopeFromString(content)

		// Delete from middle
		r.Delete(50, 100)
		result := r.String()
		expected := content[:50] + content[100:]
		if result != expected {
			t.Errorf("Delete from middle failed")
		}
	})

	t.Run("delete across nodes", func(t *testing.T) {
		// Create a large rope to ensure multiple nodes
		part1 := strings.Repeat("A", 2000)
		part2 := strings.Repeat("B", 2000)
		part3 := strings.Repeat("C", 2000)
		content := part1 + part2 + part3
		r := NewRopeFromString(content)

		// Delete from middle of part1 to middle of part3
		r.Delete(1000, 5000)
		result := r.String()
		expected := part1[:1000] + part3[1000:]
		if result != expected {
			t.Errorf("Delete across nodes failed, got length %d, expected %d", len(result), len(expected))
		}
	})

	t.Run("delete edge cases", func(t *testing.T) {
		r := NewRopeFromString("Test")

		// Invalid range (start >= end)
		r.Delete(2, 2)
		if got := r.String(); got != "Test" {
			t.Errorf("Expected 'Test' for empty range, got %q", got)
		}

		// Invalid range (start > end)
		r.Delete(3, 1)
		if got := r.String(); got != "Test" {
			t.Errorf("Expected 'Test' for invalid range, got %q", got)
		}

		// Out of bounds
		r.Delete(10, 20)
		if got := r.String(); got != "Test" {
			t.Errorf("Expected 'Test' for out of bounds, got %q", got)
		}

		// Negative start
		r.Delete(-1, 2)
		if got := r.String(); got != "Test" {
			t.Errorf("Expected 'Test' for negative start, got %q", got)
		}
	})

	t.Run("delete preserves structure", func(t *testing.T) {
		// Create a rope properly with NewRopeFromString to ensure offsets are set
		r := NewRopeFromString("Hello World!")

		// Delete part of the string
		r.Delete(8, 11) // Delete "rld"
		if got := r.String(); got != "Hello Wo!" {
			t.Errorf("Expected 'Hello Wo!', got %q", got)
		}

		// Verify the length is correct
		if r.Length() != 9 {
			t.Errorf("Expected length 9, got %d", r.Length())
		}
	})

	t.Run("delete with newlines", func(t *testing.T) {
		r := NewRopeFromString("Line1\nLine2\nLine3")

		// Delete middle line with its newline
		r.Delete(5, 12) // Delete "\nLine2\n"
		if got := r.String(); got != "Line1Line3" {
			t.Errorf("Expected 'Line1Line3', got %q", got)
		}
	})

	t.Run("delete updates offsets correctly", func(t *testing.T) {
		// Create a larger rope to test offset updates
		content := "AAABBBCCC"
		r := NewRopeFromString(content)

		// Force split by inserting at multiple positions
		r.Insert(3, "XXX")
		r.Insert(9, "YYY")
		// Now we have "AAAXXXBBBYYYCCC"

		// Delete the insertions
		r.Delete(3, 6) // Remove XXX
		// Now we have "AAABBBYYYCCC"
		r.Delete(6, 9) // Remove YYY
		// Should be back to "AAABBBCCC"

		if got := r.String(); got != content {
			t.Errorf("Expected %q, got %q", content, got)
		}
	})

	t.Run("delete entire nodes", func(t *testing.T) {
		// Create rope with multiple nodes
		texts := []string{
			strings.Repeat("A", 1000),
			strings.Repeat("B", 1000),
			strings.Repeat("C", 1000),
		}
		content := strings.Join(texts, "")
		r := NewRopeFromString(content)

		// Delete entire middle section
		r.Delete(1000, 2000)
		expected := texts[0] + texts[2]
		if got := r.String(); got != expected {
			t.Errorf("Expected length %d, got %d", len(expected), len(got))
		}
	})
}
