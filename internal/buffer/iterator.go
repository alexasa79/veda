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

// getCharAt returns the character at the iterator's current position
func (it *BufferIterator) getCharAt() byte {
	utils.Assert(it.Node != nil && it.Node.IsLeaf(), "getCharAt: iterator must point to a leaf node")
	positionInNode := it.Offset - it.Node.offset
	utils.Assert(positionInNode >= 0, "getCharAt: position in node must be >= 0")
	utils.Assert(positionInNode < len(it.Node.text), "getCharAt: position in node must be < text length")
	return it.Node.text[positionInNode]
}

// nextChar returns a new iterator moved forward one character
// Returns the new iterator and true if successful, or current iterator and false if at end of buffer
func (it *BufferIterator) nextChar() (BufferIterator, bool) {
	// Check if we're at the end of the buffer
	if it.Offset >= it.Buffer.Rope.Length() {
		return *it, false
	}

	// First increment the offset
	newOffset := it.Offset + 1

	// If the new offset is beyond the current node, move to next node
	if newOffset >= it.Node.offset + len(it.Node.text) {
		nextNode := getNextLeafNode(it.Node)
		if nextNode == nil {
			// At end of nodes but maybe not at end of buffer
			return BufferIterator{
				Buffer:  it.Buffer,
				LineNum: it.LineNum,
				Offset:  newOffset,
				Node:    it.Node,
			}, newOffset <= it.Buffer.Rope.Length()
		}
		// Move to the next node
		return BufferIterator{
			Buffer:  it.Buffer,
			LineNum: it.LineNum,
			Offset:  newOffset,
			Node:    nextNode,
		}, true
	}

	// Stay in current node
	return BufferIterator{
		Buffer:  it.Buffer,
		LineNum: it.LineNum,
		Offset:  newOffset,
		Node:    it.Node,
	}, true
}

// previousChar returns a new iterator moved backwards one character
// Returns the new iterator and true if successful, or current iterator and false if at beginning of buffer
func (it *BufferIterator) previousChar() (BufferIterator, bool) {
	nodeStartOffset := it.Node.offset
	positionInNode := it.Offset - nodeStartOffset

	// If we're at the beginning of current node, move to previous node
	if positionInNode <= 0 {
		prevNode := getPreviousLeafNode(it.Node)
		if prevNode == nil {
			// Reached beginning of buffer
			return BufferIterator{
				Buffer:  it.Buffer,
				LineNum: it.LineNum,
				Offset:  0,
				Node:    it.Node,
			}, false
		}
		return BufferIterator{
			Buffer:  it.Buffer,
			LineNum: it.LineNum,
			Offset:  prevNode.offset + len(prevNode.text) - 1,
			Node:    prevNode,
		}, true
	}

	// Move back one character in current node
	return BufferIterator{
		Buffer:  it.Buffer,
		LineNum: it.LineNum,
		Offset:  it.Offset - 1,
		Node:    it.Node,
	}, true
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

	// An iterator must always have a node
	utils.Assert(it.Node != nil, "BufferIterator.Next: iterator has nil node but is not at EOF")
	// Iterator must always point to a leaf node
	utils.Assert(it.Node.IsLeaf(), "BufferIterator.Next: iterator node must be a leaf")

	// Create a working copy of the iterator
	workingIter := *it
	var ok bool

	// Search for newline character by character
	for {
		// Check character at current position
		if workingIter.Offset < it.Buffer.Rope.Length() && workingIter.getCharAt() == '\n' {
			// Found newline, move past it
			workingIter, ok = workingIter.nextChar()
			if !ok {
				// Reached end of buffer after newline
				return &BufferIterator{
					Buffer:  it.Buffer,
					LineNum: it.LineNum + 1,
					Offset:  it.Buffer.Rope.Length(),
					Node:    workingIter.Node,
				}
			}
			workingIter.LineNum = it.LineNum + 1
			return &workingIter
		}

		// Move to next character
		workingIter, ok = workingIter.nextChar()
		if !ok {
			// Reached end of buffer without finding newline
			return &BufferIterator{
				Buffer:  it.Buffer,
				LineNum: it.LineNum + 1,
				Offset:  it.Buffer.Rope.Length(),
				Node:    workingIter.Node,
			}
		}
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

	// Create a working copy of the iterator
	workingIter := *it
	var ok bool

	// Step 1: Handle EOF case
	if workingIter.IsEOF() {
		workingIter, ok = workingIter.previousChar()
		if !ok {
			// At beginning of buffer
			return &BufferIterator{
				Buffer:  it.Buffer,
				LineNum: 0,
				Offset:  0,
				Node:    workingIter.Node,
			}
		}
		// If the character before EOF is \n, move back one more
		if workingIter.getCharAt() == '\n' {
			workingIter, ok = workingIter.previousChar()
			if !ok {
				// At beginning of buffer
				return &BufferIterator{
					Buffer:  it.Buffer,
					LineNum: 0,
					Offset:  0,
					Node:    workingIter.Node,
				}
			}
		}
	} else {
		// Step 2: Not at EOF, previous character must be \n
		workingIter, ok = workingIter.previousChar()
		if !ok {
			// At beginning of buffer
			return &BufferIterator{
				Buffer:  it.Buffer,
				LineNum: 0,
				Offset:  0,
				Node:    workingIter.Node,
			}
		}
		utils.Assert(workingIter.getCharAt() == '\n',
			"Previous: expected newline before current line start")
	}

	// Step 3: Search backwards for the previous \n
	for {
		// Move to previous character
		workingIter, ok = workingIter.previousChar()
		if !ok {
			// Reached beginning, return start of buffer
			return &BufferIterator{
				Buffer:  it.Buffer,
				LineNum: 0,
				Offset:  0,
				Node:    workingIter.Node,
			}
		}

		// Check if this character is a newline
		if workingIter.getCharAt() == '\n' {
			// Found the newline, move forward one character to get start of line
			nextIter, ok := workingIter.nextChar()
			if !ok {
				// At end of buffer after newline, return position after the newline
				nextIter = BufferIterator{
					Buffer:  it.Buffer,
					LineNum: it.LineNum - 1,
					Offset:  workingIter.Offset + 1,
					Node:    workingIter.Node,
				}
			} else {
				nextIter.LineNum = it.LineNum - 1
			}
			return &nextIter
		}
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
