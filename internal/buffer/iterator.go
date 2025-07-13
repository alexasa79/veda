package buffer

import "fmt"

// BufferIterator provides line-by-line iteration over a Buffer.
// This implementation tracks line position and caches the rope node for efficiency.
type BufferIterator struct {
	buffer   *Buffer // Pointer to the buffer
	lineNum  int     // Current line number (0-based), or points past last line if at EOF
	offset   int     // Byte offset in the buffer to the start of the current line
	node     *Node   // Pointer to the rope node that contains this line (can be nil for EOF)
}

// NewBufferIterator creates a new iterator at the beginning of the buffer
func NewBufferIterator(buffer *Buffer) *BufferIterator {
	it := &BufferIterator{
		buffer:  buffer,
		lineNum: 0,
		offset:  0,
	}
	// Find the node containing offset 0
	if buffer.Rope != nil && buffer.Rope.root != nil {
		it.node = it.findNodeAtOffset(0)
	}
	return it
}

// NewBufferIteratorAtEnd creates a new iterator at the end of the buffer
func NewBufferIteratorAtEnd(buffer *Buffer) *BufferIterator {
	it := &BufferIterator{
		buffer: buffer,
		offset: buffer.Rope.Length(),
	}
	
	// Find the last leaf node in the rope
	if buffer.Rope != nil && buffer.Rope.root != nil {
		it.node = getRightmostLeaf(buffer.Rope.root)
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
	it.lineNum = lineNum + 1
	
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
	if it.buffer.Rope == nil || it.buffer.Rope.root == nil {
		return nil
	}
	return it.findNodeAtOffsetRecursive(it.buffer.Rope.root, offset)
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
	length := it.buffer.Rope.Length()
	for pos < length && it.buffer.Rope.ByteAt(pos) != '\n' {
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

// calculateNodeOffset calculates the global offset of the start of a node
func calculateNodeOffset(node *Node) int {
	if node == nil {
		return 0
	}
	
	offset := 0
	current := node
	
	// Walk up the tree, adding offsets as we go
	for current.parent != nil {
		parent := current.parent
		if parent.right == current && parent.left != nil {
			// We came from right child, add left subtree weight
			offset += parent.left.weight
		}
		current = parent
	}
	
	return offset
}

// Next returns a new iterator pointing to the next line
func (it *BufferIterator) Next() *BufferIterator {
	if it.IsEOF() {
		return &BufferIterator{
			buffer:  it.buffer,
			lineNum: it.lineNum,
			offset:  it.offset,
			node:    it.node,
		}
	}

	// An iterator must always have a node (unless at EOF)
	if it.node == nil {
		panic("BufferIterator.Next: iterator has nil node but is not at EOF")
	}

	// Start from current position
	currentNode := it.node
	nodeStartOffset := calculateNodeOffset(currentNode)
	positionInNode := it.offset - nodeStartOffset
	currentOffset := it.offset

	// Search for newline character by character
	for {
		// If we're at the end of current leaf node
		if currentNode.IsLeaf() && positionInNode >= len(currentNode.text) {
			// Move to next node
			nextNode := getNextLeafNode(currentNode)
			if nextNode == nil {
				// Reached end of buffer - stay on last node
				return &BufferIterator{
					buffer:  it.buffer,
					lineNum: it.lineNum + 1,
					offset:  it.buffer.Rope.Length(),
					node:    currentNode, // Keep the last node
				}
			}
			currentNode = nextNode
			nodeStartOffset = calculateNodeOffset(currentNode)
			positionInNode = 0
			currentOffset = nodeStartOffset
		}

		// Check character at current position
		if currentNode.IsLeaf() && positionInNode < len(currentNode.text) {
			if currentNode.text[positionInNode] == '\n' {
				// Move past the newline
				positionInNode++
				currentOffset++
				break
			}
			// Move to next character
			positionInNode++
			currentOffset++
		}
	}

	// Determine the node for the new position
	var nextNode *Node
	if positionInNode < len(currentNode.text) {
		// Still within the same node
		nextNode = currentNode
	} else {
		// Need to move to next node
		nextNode = getNextLeafNode(currentNode)
		if nextNode != nil {
			currentOffset = calculateNodeOffset(nextNode)
		} else {
			// At EOF - stay on current (last) node
			nextNode = currentNode
		}
	}

	return &BufferIterator{
		buffer:  it.buffer,
		lineNum: it.lineNum + 1,
		offset:  currentOffset,
		node:    nextNode,
	}
}

// Previous returns a new iterator pointing to the previous line
func (it *BufferIterator) Previous() *BufferIterator {
	// An iterator must always have a node
	if it.node == nil {
		panic("BufferIterator.Previous: iterator has nil node")
	}
	
	
	if it.lineNum == 0 {
		// Already at beginning, can't go back further
		return &BufferIterator{
			buffer:  it.buffer,
			lineNum: 0,
			offset:  0,
			node:    it.node,
		}
	}

	// We need to find the start of the previous line
	// Strategy: search backwards for the start of the current line first,
	// then continue to find the start of the previous line
	
	currentNode := it.node
	nodeStartOffset := calculateNodeOffset(currentNode)
	positionInNode := it.offset - nodeStartOffset
	currentOffset := it.offset
	targetLineStart := 0
	
	// First, find the start of the current line by searching backwards for a newline
	// or reaching the beginning of the buffer
	currentLineStart := it.offset
	tempNode := currentNode
	tempNodeStart := nodeStartOffset
	tempPosInNode := positionInNode
	tempOffset := currentOffset
	
	// Search backwards to find the start of current line
	for tempOffset > 0 {
		// Move back one position
		if tempPosInNode > 0 {
			tempPosInNode--
			tempOffset--
		} else {
			// Need to move to previous node
			prevNode := getPreviousLeafNode(tempNode)
			if prevNode == nil {
				// Reached beginning of buffer
				currentLineStart = 0
				break
			}
			tempNode = prevNode
			tempNodeStart = calculateNodeOffset(tempNode)
			tempPosInNode = len(tempNode.text) - 1
			tempOffset = tempNodeStart + tempPosInNode
		}
		
		// Check if we found a newline
		if tempNode.IsLeaf() && tempPosInNode >= 0 && tempPosInNode < len(tempNode.text) {
			if tempNode.text[tempPosInNode] == '\n' {
				// Found newline, current line starts after it
				currentLineStart = tempOffset + 1
				break
			}
		}
	}
	
	// If current line starts at 0, we can't go to previous line
	if currentLineStart == 0 {
		targetLineStart = 0
	} else {
		// Now find the start of the previous line
		// Start from one position before the current line start
		tempOffset = currentLineStart - 1
		// We need to recalculate which node this position is in
		tempNode = getLeftmostLeaf(it.buffer.Rope.root)
		tempNodeStart = 0
		
		// Find the node containing tempOffset
		for tempNode != nil && tempNodeStart + len(tempNode.text) <= tempOffset {
			tempNodeStart += len(tempNode.text)
			tempNode = getNextLeafNode(tempNode)
		}
		
		if tempNode != nil {
			tempPosInNode = tempOffset - tempNodeStart
			
			// Now search backwards for the previous newline
			targetLineStart = 0
			for tempOffset > 0 {
				// Move back one position
				if tempPosInNode > 0 {
					tempPosInNode--
					tempOffset--
				} else {
					// Need to move to previous node
					prevNode := getPreviousLeafNode(tempNode)
					if prevNode == nil {
						// Reached beginning of buffer
						break
					}
					tempNode = prevNode
					tempNodeStart = calculateNodeOffset(tempNode)
					tempPosInNode = len(tempNode.text) - 1
					tempOffset = tempNodeStart + tempPosInNode
				}
				
				// Check if we found a newline
				if tempNode.IsLeaf() && tempPosInNode >= 0 && tempPosInNode < len(tempNode.text) {
					if tempNode.text[tempPosInNode] == '\n' {
						// Found newline, previous line starts after it
						targetLineStart = tempOffset + 1
						break
					}
				}
			}
		}
	}
	
	// Now we need to find the node containing targetLineStart
	// We'll start from the beginning and traverse forward
	var targetNode *Node
	if targetLineStart == 0 {
		// At beginning of buffer - get leftmost leaf
		targetNode = getLeftmostLeaf(it.buffer.Rope.root)
	} else {
		// We need to traverse forward from the start to find the node at targetLineStart
		// Start from the leftmost leaf
		targetNode = getLeftmostLeaf(it.buffer.Rope.root)
		targetOffset := 0
		
		for targetNode != nil && targetOffset + len(targetNode.text) <= targetLineStart {
			targetOffset += len(targetNode.text)
			targetNode = getNextLeafNode(targetNode)
		}
	}
	
	if targetNode == nil {
		panic("BufferIterator.Previous: failed to find node for previous line")
	}
	
	return &BufferIterator{
		buffer:  it.buffer,
		lineNum: it.lineNum - 1,
		offset:  targetLineStart,
		node:    targetNode,
	}
}

// Line returns the current line content (without trailing newline)
func (it *BufferIterator) Line() string {
	if it.IsEOF() {
		return ""
	}

	// Use cached offset and find line end
	end := it.findLineEnd(it.offset)
	
	// Use the rope's efficient extraction method
	return it.buffer.Rope.Extract(it.offset, end)
}

// LineBytes returns the current line content as a byte slice (without trailing newline)
func (it *BufferIterator) LineBytes() []byte {
	return []byte(it.Line())
}

// LineNumber returns the current line number (0-based)
func (it *BufferIterator) LineNumber() int {
	return it.lineNum
}

// IsEOF returns true if the iterator is at end of file
func (it *BufferIterator) IsEOF() bool {
	return it.offset >= it.buffer.Rope.Length()
}

// Position returns the current byte position in the buffer
func (it *BufferIterator) Position() int {
	return it.offset
}

// LineEnd returns the byte position of the end of the current line (excluding newline)
func (it *BufferIterator) LineEnd() int {
	if it.IsEOF() {
		return it.buffer.Rope.Length()
	}
	return it.findLineEnd(it.offset)
}

// Equals returns true if two iterators point to the same position
func (it *BufferIterator) Equals(other *BufferIterator) bool {
	if it == nil || other == nil {
		return it == other
	}
	return it.lineNum == other.lineNum && it.buffer == other.buffer && it.offset == other.offset
}

// String implements the fmt.Stringer interface for BufferIterator
func (it *BufferIterator) String() string {
	if it == nil {
		return "BufferIterator(nil)"
	}

	nodeInfo := "nil"
	if it.node != nil {
		if it.node.IsLeaf() {
			nodeInfo = fmt.Sprintf("leaf(len=%d)", len(it.node.text))
		} else {
			nodeInfo = "internal"
		}
	}

	end := it.offset
	if !it.IsEOF() {
		end = it.findLineEnd(it.offset)
	}
	return fmt.Sprintf("BufferIterator(line=%d, eof=%v, offset=%d, node=%s, start=%d, end=%d)",
		it.lineNum, it.IsEOF(), it.offset, nodeInfo, it.offset, end)
}
