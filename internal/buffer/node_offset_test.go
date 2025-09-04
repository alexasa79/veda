package buffer

import (
	"strings"
	"testing"
)

func TestNodeOffsets(t *testing.T) {
	t.Run("single node", func(t *testing.T) {
		// Create a rope with single leaf node
		rope := NewRopeFromString("Hello")

		// Verify root node has offset 0
		if rope.root == nil {
			t.Fatal("Expected root to be non-nil")
		}

		// Verify it's a leaf node
		if !rope.root.IsLeaf() {
			t.Error("Expected root to be a leaf node")
		}

		// Verify offset is 0 (root should start at position 0)
		if rope.root.offset != 0 {
			t.Errorf("Expected root offset to be 0, got %d", rope.root.offset)
		}
	})

	t.Run("concatenation", func(t *testing.T) {
		// Create two leaf nodes
		leftNode := NewLeaf("Hello")
		rightNode := NewLeaf(" World")

		// Set initial offsets
		leftNode.offset = 0
		rightNode.offset = 5

		// Create concatenation node
		concatNode := NewConcatNode(leftNode, rightNode)

		// Verify left child has offset 0
		if leftNode.offset != 0 {
			t.Errorf("Expected left child offset to be 0, got %d", leftNode.offset)
		}

		// Verify right child has offset = left.weight (5)
		expectedRightOffset := 5
		if rightNode.offset != expectedRightOffset {
			t.Errorf("Expected right child offset to be %d, got %d", expectedRightOffset, rightNode.offset)
		}

		// Verify internal node has offset = left child's offset (0)
		if concatNode.offset != 0 {
			t.Errorf("Expected internal node offset to be 0, got %d", concatNode.offset)
		}

		// Verify tree structure
		if concatNode.left != leftNode {
			t.Error("Expected concat node left child to be leftNode")
		}
		if concatNode.right != rightNode {
			t.Error("Expected concat node right child to be rightNode")
		}
	})

	t.Run("sequential insertions", func(t *testing.T) {
		// Start with "Hello"
		rope := NewRopeFromString("Hello")

		// Verify initial state
		if rope.root.offset != 0 {
			t.Errorf("Expected initial root offset 0, got %d", rope.root.offset)
		}

		// Insert " World" at position 5
		rope.Insert(5, " World")

		// Verify the rope content is correct
		if rope.String() != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", rope.String())
		}

		// After insertion, we should have nodes at:
		// - "Hello" at offset 0
		// - " World" at offset 5
		verifyTreeOffsets(t, rope.root, 0)

		// Insert "Beautiful " at position 6
		rope.Insert(6, "Beautiful ")

		// Verify the rope content is correct
		expected := "Hello Beautiful World"
		if rope.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, rope.String())
		}

		// After second insertion, nodes should be at:
		// - "Hello " at offset 0
		// - "Beautiful " at offset 6
		// - "World" at offset 16
		verifyTreeOffsets(t, rope.root, 0)
	})

	t.Run("insertion at beginning", func(t *testing.T) {
		// Start with "World"
		rope := NewRopeFromString("World")

		// Verify initial state - "World" should be at offset 0
		if rope.root.offset != 0 {
			t.Errorf("Expected initial root offset 0, got %d", rope.root.offset)
		}

		// Insert "Hello " at position 0 (beginning)
		rope.Insert(0, "Hello ")

		// Verify the rope content is correct
		expected := "Hello World"
		if rope.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, rope.String())
		}

		// After insertion at beginning, the root node should still be at offset 0
		// because it contains the insertion point
		if rope.root.offset != 0 {
			t.Errorf("Expected root offset to remain 0 after insertion, got %d", rope.root.offset)
		}

		// Let's also verify that we can find the original "World" text
		// by checking that text starting at position 6 is "World"
		worldPart := rope.Extract(6, 11) // positions 6-10 should be "World"
		if worldPart != "World" {
			t.Errorf("Expected 'World' at position 6-10, got '%s'", worldPart)
		}
	})

	t.Run("node splitting", func(t *testing.T) {
		// Create a string close to maxLeafSize then insert to exceed it and force a split
		baseLen := maxLeafSize - 500
		largeText := ""
		for i := 0; i < baseLen; i++ {
			largeText += "a"
		}

		// Create rope with large text
		rope := NewRopeFromString(largeText)

		// Insert more text to trigger splitting (ensure we go over maxLeafSize)
		rope.Insert(baseLen/2, strings.Repeat("X", 2000))

		// After splitting, we should have multiple nodes
		if rope.root.IsLeaf() {
			t.Error("Expected root to be split into multiple nodes, but it's still a leaf")
		}

		// Verify that both split nodes have correct offsets
		if rope.root.left == nil || rope.root.right == nil {
			t.Fatal("Expected root to have both left and right children after splitting")
		}

		// Left node should start at offset 0
		if rope.root.left.offset != 0 {
			t.Errorf("Expected left node offset 0, got %d", rope.root.left.offset)
		}

		// Debug the actual weights and offsets
		t.Logf("Left node weight: %d, offset: %d", rope.root.left.weight, rope.root.left.offset)
		t.Logf("Right node weight: %d, offset: %d", rope.root.right.weight, rope.root.right.offset)
		t.Logf("Total length: %d", rope.Length())

		// Right node should start after left node
		expectedRightOffset := rope.root.left.weight
		if rope.root.right.offset != expectedRightOffset {
			t.Errorf("Expected right node offset %d, got %d", expectedRightOffset, rope.root.right.offset)
		}

		// Internal node should have same offset as left child
		if rope.root.offset != rope.root.left.offset {
			t.Errorf("Expected internal node offset to match left child (%d), got %d", rope.root.left.offset, rope.root.offset)
		}

		// Verify content is still correct
		expectedContent := largeText[:baseLen/2] + strings.Repeat("X", 2000) + largeText[baseLen/2:]
		if rope.String() != expectedContent {
			t.Error("Content mismatch after splitting")
		}
	})

	t.Run("complex tree structure", func(t *testing.T) {
		// Build a tree with multiple levels of concatenation
		// Start with individual leaf nodes
		node1 := NewLeaf("Hello")
		node2 := NewLeaf(" ")
		node3 := NewLeaf("World")
		node4 := NewLeaf("!")

		// Set initial offsets manually
		node1.offset = 0
		node2.offset = 5
		node3.offset = 6
		node4.offset = 11

		// Create first level: ("Hello" + " ") and ("World" + "!")
		left := NewConcatNode(node1, node2)
		right := NewConcatNode(node3, node4)

		// Create root level: (("Hello" + " ") + ("World" + "!"))
		root := NewConcatNode(left, right)

		// Verify offsets throughout the entire tree structure

		// Root should have offset 0 (same as leftmost child)
		if root.offset != 0 {
			t.Errorf("Expected root offset 0, got %d", root.offset)
		}

		// Left subtree should have offset 0
		if left.offset != 0 {
			t.Errorf("Expected left subtree offset 0, got %d", left.offset)
		}

		// Right subtree should have offset = left subtree weight (6)
		expectedRightOffset := left.weight
		if right.offset != expectedRightOffset {
			t.Errorf("Expected right subtree offset %d, got %d", expectedRightOffset, right.offset)
		}

		// Verify leaf nodes have correct offsets
		if node1.offset != 0 {
			t.Errorf("Expected node1 offset 0, got %d", node1.offset)
		}
		if node2.offset != 5 {
			t.Errorf("Expected node2 offset 5, got %d", node2.offset)
		}
		if node3.offset != 6 {
			t.Errorf("Expected node3 offset 6, got %d", node3.offset)
		}
		if node4.offset != 11 {
			t.Errorf("Expected node4 offset 11, got %d", node4.offset)
		}

		// Create a rope with this complex structure
		rope := &Rope{root: root}

		// Verify content is correct
		expected := "Hello World!"
		if rope.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, rope.String())
		}

		// Verify Extract works correctly with complex offsets
		if rope.Extract(0, 5) != "Hello" {
			t.Error("Extract(0,5) failed")
		}
		if rope.Extract(6, 11) != "World" {
			t.Error("Extract(6,11) failed")
		}
	})
}

// Helper function to verify offsets are consistent with tree structure
func verifyTreeOffsets(t *testing.T, node *Node, expectedOffset int) {
	if node == nil {
		return
	}

	if node.offset != expectedOffset {
		t.Errorf("Node offset mismatch: expected %d, got %d", expectedOffset, node.offset)
	}

	if node.IsLeaf() {
		return
	}

	// For internal nodes, verify children have correct offsets
	if node.left != nil {
		verifyTreeOffsets(t, node.left, expectedOffset)
	}
	if node.right != nil {
		leftWeight := 0
		if node.left != nil {
			leftWeight = node.left.weight
		}
		verifyTreeOffsets(t, node.right, expectedOffset+leftWeight)
	}
}
