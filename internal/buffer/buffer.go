package buffer

import "strings"

const (
	maxLeafSize = 4096
	minLeafSize = 2048
)

// Node represents a node in the rope tree.
// It can be either a leaf node (containing text) or an internal node (with children).
type Node struct {
	// Common fields for all nodes
	parent *Node // Parent node (nil for root)
	weight int   // Total length of text in this subtree

	// For internal nodes
	left  *Node // Left child
	right *Node // Right child

	// For leaf nodes
	text string // The actual text content (empty for internal nodes)

	offset int
}

// NewLeaf creates a new leaf node.
func NewLeaf(text string) *Node {
	return &Node{
		text:   text,
		weight: len(text),
		offset: 0, // Will be set properly when added to rope
	}
}

// NewConcatNode creates a new internal concatenation node.
func NewConcatNode(left, right *Node) *Node {
	node := &Node{
		left:   left,
		right:  right,
		weight: 0,
		offset: 0, // Will be set when added to rope
	}
	// Calculate total weight and set child offsets
	if left != nil {
		node.weight += left.weight
		left.parent = node
		// Left child keeps its current offset
	}
	if right != nil {
		node.weight += right.weight
		right.parent = node
		// Right child's offset is left's offset + left's weight
		if left != nil {
			right.offset = left.offset + left.weight
		}
	}
	// Internal node's offset is same as left child's offset
	if left != nil {
		node.offset = left.offset
	}
	return node
}

// IsLeaf returns true if this is a leaf node
func (n *Node) IsLeaf() bool {
	return n.left == nil && n.right == nil
}

// Length returns the total length of the text stored under this node.
func (n *Node) Length() int {
	if n == nil {
		return 0
	}
	return n.weight
}

// String returns the full text of the node.
func (n *Node) String() string {
	if n == nil {
		return ""
	}
	if n.IsLeaf() {
		return n.text
	}
	var sb strings.Builder
	sb.WriteString(n.left.String())
	sb.WriteString(n.right.String())
	return sb.String()
}


// Rope is the main data structure for the editor's text content.
// It implements a rope data structure for efficient text manipulation.
type Rope struct {
	root *Node
}

// NewRopeFromString creates a new rope from a string.
// This is a simple implementation that doesn't create a balanced tree yet.
func NewRopeFromString(text string) *Rope {
	root := NewLeaf(text)
	root.offset = 0 // Root starts at offset 0
	return &Rope{
		root: root,
	}
}

func (r *Rope) String() string {
	if r.root == nil {
		return ""
	}
	return r.root.String()
}

func (r *Rope) Length() int {
	if r.root == nil {
		return 0
	}
	return r.root.Length()
}

// Lines returns all lines in the rope.
func (r *Rope) Lines() []string {
	return strings.Split(r.String(), "\n")
}

// Insert inserts text at a given position in the rope.
func (r *Rope) Insert(pos int, text string) {
	if r.root == nil {
		r.root = NewLeaf(text)
		r.root.offset = 0
		return
	}
	r.root = r.insert(r.root, pos, text)
}

// ByteAt returns the byte at the given position in the rope.
// Returns 0 if position is out of bounds.
func (r *Rope) ByteAt(pos int) byte {
	if r.root == nil || pos < 0 || pos >= r.Length() {
		return 0
	}
	return r.byteAt(r.root, pos)
}

// byteAt is a helper method that recursively finds the byte at position
func (r *Rope) byteAt(node *Node, pos int) byte {
	if node == nil {
		return 0
	}

	if node.IsLeaf() {
		if pos >= 0 && pos < len(node.text) {
			return node.text[pos]
		}
		return 0
	}

	// Internal node - check left subtree weight
	leftWeight := 0
	if node.left != nil {
		leftWeight = node.left.weight
	}

	if pos < leftWeight {
		return r.byteAt(node.left, pos)
	}
	return r.byteAt(node.right, pos-leftWeight)
}

// insert is the recursive helper for the Insert method.
func (r *Rope) insert(node *Node, pos int, text string) *Node {
	if node == nil {
		newNode := NewLeaf(text)
		newNode.offset = pos
		return newNode
	}

	if node.IsLeaf() {
		// Insert into leaf
		node.text = node.text[:pos] + text + node.text[pos:]
		node.weight = len(node.text)

		if node.weight > maxLeafSize {
			// Split the leaf
			splitPos := node.weight / 2
			leftNode := NewLeaf(node.text[:splitPos])
			rightNode := NewLeaf(node.text[splitPos:])
			
			// Set offsets for split nodes
			leftNode.offset = node.offset
			rightNode.offset = node.offset + splitPos
			
			newNode := NewConcatNode(leftNode, rightNode)
			newNode.offset = node.offset

			// Update parent pointer
			if node.parent != nil {
				newNode.parent = node.parent
				if node.parent.left == node {
					node.parent.left = newNode
				} else {
					node.parent.right = newNode
				}
			}
			return newNode
		}
		return node
	}

	// Internal node - check against left subtree weight
	leftWeight := 0
	if node.left != nil {
		leftWeight = node.left.weight
	}

	if pos < leftWeight {
		node.left = r.insert(node.left, pos, text)
		node.left.parent = node
	} else {
		node.right = r.insert(node.right, pos-leftWeight, text)
		node.right.parent = node
	}

	// Update total weight
	node.weight = 0
	if node.left != nil {
		node.weight += node.left.weight
	}
	if node.right != nil {
		node.weight += node.right.weight
		// Update right child's offset
		if node.left != nil {
			node.right.offset = node.left.offset + node.left.weight
		}
	}

	return node
}

// Extract returns a substring from the rope between start and end positions
func (r *Rope) Extract(start, end int) string {
	if start >= end || start < 0 || end > r.Length() {
		return ""
	}

	// For small extractions, use byte-by-byte
	length := end - start
	if length <= 32 {
		result := make([]byte, length)
		for i := 0; i < length; i++ {
			result[i] = r.ByteAt(start + i)
		}
		return string(result)
	}

	// For larger extractions, use optimized approach
	var result strings.Builder
	result.Grow(length)
	r.extractRange(r.root, start, end, &result)
	return result.String()
}

// extractRange recursively extracts text from start to end into the builder
func (r *Rope) extractRange(node *Node, start, end int, result *strings.Builder) {
	if node == nil || start >= end {
		return
	}

	if node.IsLeaf() {
		// Extract from leaf
		nodeEnd := len(node.text)
		if start < nodeEnd && end > 0 {
			// Calculate the range within this leaf
			extractStart := 0
			if start > 0 {
				extractStart = start
			}
			extractEnd := nodeEnd
			if end < nodeEnd {
				extractEnd = end
			}
			result.WriteString(node.text[extractStart:extractEnd])
		}
		return
	}

	// Internal node
	leftWeight := 0
	if node.left != nil {
		leftWeight = node.left.weight
	}

	// Extract from left subtree if needed
	if start < leftWeight {
		leftEnd := end
		if leftEnd > leftWeight {
			leftEnd = leftWeight
		}
		r.extractRange(node.left, start, leftEnd, result)
	}

	// Extract from right subtree if needed
	if end > leftWeight {
		rightStart := 0
		if start > leftWeight {
			rightStart = start - leftWeight
		}
		r.extractRange(node.right, rightStart, end-leftWeight, result)
	}
}
