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
		node:   nil, // EOF has no node
	}
	
	// Calculate the line number at EOF
	// The line number at EOF should be the same as the line number
	// of the last character in the buffer
	lineNum := 0
	length := buffer.Rope.Length()
	
	// Count newlines up to (but not including) the last character
	// This gives us the line number of the last character
	for i := 0; i < length-1; i++ {
		if buffer.Rope.ByteAt(i) == '\n' {
			lineNum++
		}
	}
	
	// Special case: if buffer ends with \n, we need to count it
	// because EOF is on the empty line after the newline
	if length > 0 && buffer.Rope.ByteAt(length-1) == '\n' {
		lineNum++
	}
	
	it.lineNum = lineNum
	
	return it
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

	// Calculate the offset for the next line
	end := it.findLineEnd(it.offset)
	nextOffset := end
	if nextOffset < it.buffer.Rope.Length() && it.buffer.Rope.ByteAt(nextOffset) == '\n' {
		nextOffset++ // Skip the newline
	}

	next := &BufferIterator{
		buffer:  it.buffer,
		lineNum: it.lineNum + 1,
		offset:  nextOffset,
	}

	// Update the node if we need to
	if nextOffset < it.buffer.Rope.Length() {
		next.node = next.findNodeAtOffset(nextOffset)
	} else {
		next.node = nil // EOF
	}

	return next
}

// Previous returns a new iterator pointing to the previous line
func (it *BufferIterator) Previous() *BufferIterator {
	if it.lineNum == 0 {
		return &BufferIterator{
			buffer:  it.buffer,
			lineNum: 0,
			offset:  0,
			node:    it.node,
		}
	}

	newLineNum := it.lineNum - 1

	// Make sure we don't go below 0
	if newLineNum < 0 {
		newLineNum = 0
	}

	prev := &BufferIterator{
		buffer:  it.buffer,
		lineNum: newLineNum,
	}

	// Find the offset for the previous line by searching backwards
	// We need to find where line (newLineNum) starts
	pos := 0
	currentLine := 0
	length := it.buffer.Rope.Length()
	
	// Count newlines until we reach the target line
	for pos < length && currentLine < newLineNum {
		if it.buffer.Rope.ByteAt(pos) == '\n' {
			currentLine++
		}
		pos++
	}
	
	prev.offset = pos
	if prev.offset < it.buffer.Rope.Length() {
		prev.node = prev.findNodeAtOffset(prev.offset)
	} else {
		prev.node = nil
	}

	return prev
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
