package buffer

import (
	"fmt"
	"veda/internal/utils"
)

// BufferIterator provides line-by-line iteration over a Buffer.
// This implementation tracks line position and caches the rope node for efficiency.
type BufferIterator struct {
	Buffer   *Buffer // Pointer to the buffer
	LineNum  int     // Current line number (0-based), or points past last line if at EOF
	Offset   int     // Byte offset in the buffer to the start of the current line
	Node     *Node   // Pointer to the rope node that contains this line (can be nil for EOF)
}

// NewBufferIterator creates a new iterator at the beginning of the buffer
func NewBufferIterator(buffer *Buffer) *BufferIterator {
	it := &BufferIterator{
		Buffer:  buffer,
		LineNum: 0,
		Offset:  0,
	}
	// Find the node containing offset 0
	if buffer.Rope != nil && buffer.Rope.root != nil {
		it.Node = it.findNodeAtOffset(0)
	}
	return it
}

// NewBufferIteratorAtEnd creates a new iterator at the end of the buffer
func NewBufferIteratorAtEnd(buffer *Buffer) *BufferIterator {
	it := &BufferIterator{
		Buffer: buffer,
		Offset: buffer.Rope.Length(),
	}

	// Find the last leaf node in the rope
	if buffer.Rope != nil && buffer.Rope.root != nil {
		it.Node = getRightmostLeaf(buffer.Rope.root)
	}

	// Calculate the line number at EOF
	// We set the line number to be one more than the actual last line
	// This way Previous() can always just do lineNum - 1
	lineNum := 0
	length := buffer.Rope.Length()

	// Count all newlines in the buffer
	for i := 0; i < length; i++ {
		if buffer.Rope.ByteAt(i) == '\n' {
			lineNum++
		}
	}

	// Add 1 to represent being past the last line
	it.LineNum = lineNum + 1

	return it
}

// getRightmostLeaf returns the rightmost leaf node in the subtree
func getRightmostLeaf(node *Node) *Node {
	if node == nil {
		return nil
	}

	current := node
	for !current.IsLeaf() {
		if current.right != nil {
			current = current.right
		} else if current.left != nil {
			current = current.left
		} else {
			// Shouldn't happen in a well-formed tree
			return nil
		}
	}

	return current
}

// findNodeAtOffset finds the leaf node containing the given offset
func (it *BufferIterator) findNodeAtOffset(offset int) *Node {
	if it.Buffer.Rope == nil || it.Buffer.Rope.root == nil {
		return nil
	}
	return it.findNodeAtOffsetRecursive(it.Buffer.Rope.root, offset)
}

// findNodeAtOffsetRecursive recursively finds the leaf node at the given offset
func (it *BufferIterator) findNodeAtOffsetRecursive(node *Node, offset int) *Node {
	if node == nil {
		return nil
	}

	if node.IsLeaf() {
		return node
	}

	// Internal node - check against left subtree weight
	leftWeight := 0
	if node.left != nil {
		leftWeight = node.left.weight
	}

	if offset < leftWeight {
		return it.findNodeAtOffsetRecursive(node.left, offset)
	}
	return it.findNodeAtOffsetRecursive(node.right, offset-leftWeight)
}

// findLineEnd finds the end of the current line starting from the given offset
func (it *BufferIterator) findLineEnd(start int) int {
	pos := start
	length := it.Buffer.Rope.Length()
	for pos < length && it.Buffer.Rope.ByteAt(pos) != '\n' {
		pos++
	}
	return pos
}

// getNextLeafNode returns the next leaf node in in-order traversal
func getNextLeafNode(node *Node) *Node {
	if node == nil {
		return nil
	}

	// Go up until we find a parent where we came from the left child
	current := node
	for current.parent != nil {
		parent := current.parent
		if parent.left == current {
			// We came from left child, go to right subtree
			return getLeftmostLeaf(parent.right)
		}
		// We came from right child, continue going up
		current = parent
	}

	// No more nodes
	return nil
}

// getLeftmostLeaf returns the leftmost leaf node in the subtree
func getLeftmostLeaf(node *Node) *Node {
	if node == nil {
		return nil
	}

	current := node
	for !current.IsLeaf() {
		if current.left != nil {
			current = current.left
		} else if current.right != nil {
			current = current.right
		} else {
			// Shouldn't happen in a well-formed tree
			return nil
		}
	}

	return current
}

// getPreviousLeafNode returns the previous leaf node in in-order traversal
func getPreviousLeafNode(node *Node) *Node {
	if node == nil {
		return nil
	}

	// Go up until we find a parent where we came from the right child
	current := node
	for current.parent != nil {
		parent := current.parent
		if parent.right == current {
			// We came from right child, go to left subtree
			return getRightmostLeaf(parent.left)
		}
		// We came from left child, continue going up
		current = parent
	}

	// No more nodes
	return nil
}

// Next returns a new iterator pointing to the next line
func (it *BufferIterator) Next() *BufferIterator {
	if it.IsEOF() {
		return &BufferIterator{
			Buffer:  it.Buffer,
			LineNum: it.LineNum,
			Offset:  it.Offset,
			Node:    it.Node,
		}
	}

	// An iterator must always have a node (unless at EOF)
	if it.Node == nil {
		panic("BufferIterator.Next: iterator has nil node but is not at EOF")
	}

	// Start from current position
	currentNode := it.Node
	// Iterator must always point to a leaf node
	utils.Assert(currentNode.IsLeaf(), "BufferIterator.Next: iterator node must be a leaf")
	nodeStartOffset := currentNode.offset
	positionInNode := it.Offset - nodeStartOffset
	currentOffset := it.Offset

	// Search for newline character by character
	for {
		// If we're at the end of current node
		if positionInNode >= len(currentNode.text) {
			// Move to next node
			nextNode := getNextLeafNode(currentNode)
			if nextNode == nil {
				// Assert that we are at the end of the buffer
				utils.Assert(currentOffset == it.Buffer.Rope.Length(), "reached end of nodes but not at end of buffer")

				// Reached end of buffer - stay on last node
				return &BufferIterator{
					Buffer:  it.Buffer,
					LineNum: it.LineNum + 1,
					Offset:  it.Buffer.Rope.Length(),
					Node:    currentNode, // Keep the last node
				}
			}
			currentNode = nextNode
			nodeStartOffset = currentNode.offset
			positionInNode = 0
			currentOffset = nodeStartOffset
		}

		// Check character at current position
		if positionInNode < len(currentNode.text) {
			if currentNode.text[positionInNode] == '\n' {
				// Move past the newline
				positionInNode++
				currentOffset++

				// Check if we need to move to the next node after the newline
				if positionInNode >= len(currentNode.text) {
					nextNode := getNextLeafNode(currentNode)
					if nextNode != nil {
						currentNode = nextNode
						// Keep the currentOffset we calculated, don't reset it
					}
					// If nextNode is nil, we're at EOF but stay on current node
				}
				break
			}
			// Move to next character
			positionInNode++
			currentOffset++
		}
	}

	return &BufferIterator{
		Buffer:  it.Buffer,
		LineNum: it.LineNum + 1,
		Offset:  currentOffset,
		Node:    currentNode,
	}
}

// Previous returns a new iterator pointing to the previous line
func (it *BufferIterator) Previous() *BufferIterator {
	// An iterator must always have a node
	utils.Assert(it.Node != nil, "BufferIterator.Previous: iterator has nil node")

	if it.LineNum == 0 {
		// Already at beginning, can't go back further
		return &BufferIterator{
			Buffer:  it.Buffer,
			LineNum: 0,
			Offset:  0,
			Node:    it.Node,
		}
	}

	// Start from current position
	currentNode := it.Node
	// Iterator must always point to a leaf node
	utils.Assert(currentNode.IsLeaf(), "BufferIterator.Previous: iterator node must be a leaf")
	nodeStartOffset := currentNode.offset
	positionInNode := it.Offset - nodeStartOffset
	currentOffset := it.Offset

	// Search backwards for newline character by character
	for {
		// If we're at the beginning of current node
		if positionInNode <= 0 {
			// Move to previous node
			prevNode := getPreviousLeafNode(currentNode)
			if prevNode == nil {
				// Assert that we are at the beginning of the buffer
				utils.Assert(currentOffset == 0, "reached beginning of nodes but not at beginning of buffer")

				// Reached beginning of buffer - stay at position 0
				return &BufferIterator{
					Buffer:  it.Buffer,
					LineNum: 0,
					Offset:  0,
					Node:    currentNode,
				}
			}
			currentNode = prevNode
			nodeStartOffset = currentNode.offset
			positionInNode = len(currentNode.text)
			currentOffset = nodeStartOffset + positionInNode
		}

		// Move back one character
		positionInNode--
		currentOffset--

		// Check character at current position
		if positionInNode >= 0 && positionInNode < len(currentNode.text) {
			if currentNode.text[positionInNode] == '\n' {
				// Found newline! The previous line starts after this newline
				positionInNode++
				currentOffset++

				// Check if we need to move to the next node after the newline
				if positionInNode >= len(currentNode.text) {
					nextNode := getNextLeafNode(currentNode)
					if nextNode != nil {
						currentNode = nextNode
						// Keep the currentOffset we calculated, don't reset it
					}
					// If nextNode is nil, we stay on current node
				}
				break
			}
		}
	}

	return &BufferIterator{
		Buffer:  it.Buffer,
		LineNum: it.LineNum - 1,
		Offset:  currentOffset,
		Node:    currentNode,
	}
}

// Line returns the current line content (without trailing newline)
func (it *BufferIterator) Line() string {
	if it.IsEOF() {
		return ""
	}

	// Use cached offset and find line end
	end := it.findLineEnd(it.Offset)

	// Use the rope's efficient extraction method
	return it.Buffer.Rope.Extract(it.Offset, end)
}

// LineBytes returns the current line content as a byte slice (without trailing newline)
func (it *BufferIterator) LineBytes() []byte {
	return []byte(it.Line())
}

// LineNumber returns the current line number (0-based)
func (it *BufferIterator) LineNumber() int {
	return it.LineNum
}

// IsEOF returns true if the iterator is at end of file
func (it *BufferIterator) IsEOF() bool {
	return it.Offset >= it.Buffer.Rope.Length()
}

// Position returns the current byte position in the buffer
func (it *BufferIterator) Position() int {
	return it.Offset
}

// LineEnd returns the byte position of the end of the current line (excluding newline)
func (it *BufferIterator) LineEnd() int {
	if it.IsEOF() {
		return it.Buffer.Rope.Length()
	}
	return it.findLineEnd(it.Offset)
}

// Equals returns true if two iterators point to the same position
func (it *BufferIterator) Equals(other *BufferIterator) bool {
	if it == nil || other == nil {
		return it == other
	}
	return it.LineNum == other.LineNum && it.Buffer == other.Buffer && it.Offset == other.Offset
}

// String implements the fmt.Stringer interface for BufferIterator
func (it *BufferIterator) String() string {
	if it == nil {
		return "BufferIterator(nil)"
	}

	nodeInfo := "nil"
	if it.Node != nil {
		if it.Node.IsLeaf() {
			nodeInfo = fmt.Sprintf("leaf(len=%d)", len(it.Node.text))
		} else {
			nodeInfo = "internal"
		}
	}

	end := it.Offset
	if !it.IsEOF() {
		end = it.findLineEnd(it.Offset)
	}
	return fmt.Sprintf("BufferIterator(line=%d, eof=%v, offset=%d, node=%s, start=%d, end=%d)",
		it.LineNum, it.IsEOF(), it.Offset, nodeInfo, it.Offset, end)
}
