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
	parent      *Node // Parent node (nil for root)
	weight      int   // Total length of text in this subtree

	offset int // Offset of the node relative to the beginning of the buffer

	newlineCount int  // Total number of newline characters in this subtree

	// For internal nodes
	left  *Node // Left child
	right *Node // Right child

	// For leaf nodes
	text string // The actual text content (empty for internal nodes)

}

// countNewlines counts the number of newline characters in a string
func countNewlines(text string) int {
	count := 0
	for _, ch := range []byte(text) {
		if ch == '\n' {
			count++
		}
	}
	return count
}

// NewLeaf creates a new leaf node.
func NewLeaf(text string) *Node {
	return &Node{
		text:         text,
		weight:       len(text),
		newlineCount: countNewlines(text),
		offset:       0, // Will be set properly when added to rope
	}
}

// NewConcatNode creates a new internal concatenation node.
func NewConcatNode(left, right *Node) *Node {
	node := &Node{
		left:   left,
		right:  right,
		weight: 0,
		newlineCount: 0,
		offset: 0, // Will be set when added to rope
	}
	// Calculate total weight, newline count, and set child offsets
	if left != nil {
		node.weight += left.weight
		node.newlineCount += left.newlineCount
		left.parent = node
		// Left child keeps its current offset
	}
	if right != nil {
		node.weight += right.weight
		node.newlineCount += right.newlineCount
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

// findNodeAtLine finds the leaf node containing the start of the specified line number (0-based)
// Returns the node and the byte offset within that node where the line starts
// This function runs in O(log n) time by using the newlineCount fields
func (r *Rope) findNodeAtLine(lineNum int) (*Node, int) {
	if r.root == nil || lineNum < 0 {
		return nil, 0
	}

	// For empty rope, only line 0 exists (as an empty line)
	if r.Length() == 0 {
		if lineNum == 0 {
			return r.root, 0
		}
		return nil, 0
	}

	// Check if the requested line exists
	// Line N exists if N < newlineCount, or N == newlineCount and rope ends with newline
	totalNewlines := r.root.newlineCount
	endsWithNewline := r.ByteAt(r.Length()-1) == '\n'

	maxValidLine := totalNewlines
	if !endsWithNewline {
		maxValidLine = totalNewlines
	}

	if lineNum > maxValidLine {
		return nil, 0
	}

	return r.findNodeAtLineRecursive(r.root, lineNum)
}

// findNodeAtLineRecursive recursively searches for the node containing the specified line
func (r *Rope) findNodeAtLineRecursive(node *Node, targetLine int) (*Node, int) {
	if node == nil {
		return nil, 0
	}

	if node.IsLeaf() {
		// We've reached a leaf node, now find the exact position within this node
		return r.findLineStartInLeaf(node, targetLine)
	}

	// Internal node - check left subtree newline count
	leftNewlineCount := 0
	if node.left != nil {
		leftNewlineCount = node.left.newlineCount
	}

	if targetLine <= leftNewlineCount {
		// Target line is in the left subtree or at the boundary
		// Try left subtree first
		resultNode, offset := r.findNodeAtLineRecursive(node.left, targetLine)

		// If we found a valid result in left subtree, return it
		if resultNode != nil && (targetLine < leftNewlineCount || offset < len(resultNode.text)) {
			return resultNode, offset
		}

		// If targetLine == leftNewlineCount and we didn't find it in left subtree,
		// or if we're at the end of left subtree, try right subtree
		if targetLine == leftNewlineCount {
			return r.findNodeAtLineRecursive(node.right, 0)
		}
	}

	// Target line is in the right subtree
	// Adjust target line by subtracting lines in left subtree
	return r.findNodeAtLineRecursive(node.right, targetLine - leftNewlineCount)
}

// findLineStartInLeaf finds the start position of the specified line within a leaf node
// Returns the node and the byte offset within the node where the line starts
func (r *Rope) findLineStartInLeaf(node *Node, targetLine int) (*Node, int) {
	if !node.IsLeaf() || targetLine < 0 {
		return node, 0
	}

	// If targetLine is 0, we want the start of the node
	if targetLine == 0 {
		return node, 0
	}

	// Count newlines in the leaf to find the target line
	currentLine := 0
	for i, ch := range []byte(node.text) {
		if ch == '\n' {
			currentLine++
			if currentLine == targetLine {
				// Found the target line, return position after this newline
				return node, i + 1
			}
		}
	}

	// If we didn't find enough newlines, the target line is beyond this node
	// This shouldn't happen if the tree newline counts are correct
	return node, len(node.text)
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
		// Store original newline count before modification
		originalNewlineCount := node.newlineCount

		// Insert into leaf
		node.text = node.text[:pos] + text + node.text[pos:]
		node.weight = len(node.text)

		// Efficiently update newline count by only counting newlines in inserted text
		node.newlineCount = originalNewlineCount + countNewlines(text)

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

	// Update total weight and newline count
	node.weight = 0
	node.newlineCount = 0
	if node.left != nil {
		node.weight += node.left.weight
		node.newlineCount += node.left.newlineCount
	}
	if node.right != nil {
		node.weight += node.right.weight
		node.newlineCount += node.right.newlineCount
		// Update right child's offset
		if node.left != nil {
			newOffset := node.left.offset + node.left.weight
			adjustNodeOffset(node.right, newOffset - node.right.offset)
		}
	}

	return node
}

// adjustNodeOffset recursively adjusts the offset of a node and all its descendants by the given delta
func adjustNodeOffset(node *Node, delta int) {
	if node == nil || delta == 0 {
		return
	}

	// Adjust this node's offset
	node.offset += delta

	// If it's a leaf, we're done
	if node.IsLeaf() {
		return
	}

	// Recursively adjust both children
	adjustNodeOffset(node.left, delta)
	adjustNodeOffset(node.right, delta)
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

// Delete removes characters from the rope between start and end positions.
func (r *Rope) Delete(start, end int) {
	if start >= end || start < 0 || end > r.Length() {
		return
	}

	r.root = r.delete(r.root, start, end)
}

// delete recursively removes characters between start and end positions
func (r *Rope) delete(node *Node, start, end int) *Node {
	if node == nil || start >= end {
		return node
	}

	if node.IsLeaf() {
		// Delete from leaf
		nodeLen := len(node.text)

		// If deletion covers entire leaf, return nil to remove this node
		if start <= 0 && end >= nodeLen {
			return nil
		}

		// Partial deletion
		newText := ""
		if start > 0 {
			newText += node.text[:start]
		}
		if end < nodeLen {
			newText += node.text[end:]
		}

		if newText == "" {
			return nil // Empty leaf, remove it
		}

		// Create new leaf with remaining text
		newLeaf := NewLeaf(newText)
		newLeaf.offset = node.offset
		newLeaf.parent = node.parent
		return newLeaf
	}

	// Internal node
	leftWeight := 0
	if node.left != nil {
		leftWeight = node.left.weight
	}

	// Process left subtree if needed
	if start < leftWeight {
		leftEnd := end
		if leftEnd > leftWeight {
			leftEnd = leftWeight
		}
		node.left = r.delete(node.left, start, leftEnd)
		if node.left != nil {
			node.left.parent = node
		}
	}

	// Process right subtree if needed
	if end > leftWeight {
		rightStart := 0
		if start > leftWeight {
			rightStart = start - leftWeight
		}
		node.right = r.delete(node.right, rightStart, end-leftWeight)
		if node.right != nil {
			node.right.parent = node
			// Update right offset
			if node.left != nil {
				node.right.offset = node.left.offset + node.left.weight
			} else {
				node.right.offset = node.offset
			}
		}
	}

	// Update node weight and newline count
	node.weight = 0
	node.newlineCount = 0
	if node.left != nil {
		node.weight += node.left.weight
		node.newlineCount += node.left.newlineCount
	}
	if node.right != nil {
		node.weight += node.right.weight
		node.newlineCount += node.right.newlineCount
	}

	// Handle empty nodes
	if node.left == nil && node.right == nil {
		return nil
	}
	if node.left == nil {
		return node.right
	}
	if node.right == nil {
		return node.left
	}

	return node
}
