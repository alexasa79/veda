package buffer

// BufferIterator provides efficient line-by-line iteration over a Buffer
type BufferIterator struct {
	buffer     *Buffer
	rope       *Rope        // Direct reference to the rope
	stack      []walkFrame  // Stack for tree traversal
	currentPos int          // Current byte position
	lineStart  int          // Start position of current line
	lineEnd    int          // End position of current line (excluding \n)
	lineNum    int          // Current line number
	atEOF      bool         // True if at end of file
}

type walkFrame struct {
	node      Node
	nodeStart int  // Byte position where this node starts
	isLeft    bool // True if we're in left subtree
}

// NewBufferIterator creates a new iterator at the beginning of the buffer
func NewBufferIterator(buffer *Buffer) *BufferIterator {
	it := &BufferIterator{
		buffer:     buffer,
		rope:       buffer.Rope,
		stack:      make([]walkFrame, 0, 32),
		currentPos: 0,
		lineStart:  0,
		lineNum:    0,
		atEOF:      false,
	}
	
	if it.rope.root == nil {
		it.atEOF = true
		return it
	}
	
	// Initialize stack with root
	it.stack = append(it.stack, walkFrame{node: it.rope.root, nodeStart: 0})
	
	// Find first line end
	it.findLineEnd()
	
	return it
}

// NewBufferIteratorAtEnd creates a new iterator at the end of the buffer
func NewBufferIteratorAtEnd(buffer *Buffer) *BufferIterator {
	it := &BufferIterator{
		buffer:     buffer,
		rope:       buffer.Rope,
		stack:      make([]walkFrame, 0, 32),
		currentPos: 0,
		lineStart:  0,
		lineNum:    0,
		atEOF:      true,
	}
	
	if it.rope.root == nil {
		return it
	}
	
	// Count lines to set lineNum correctly
	ropeLen := it.rope.Length()
	lineCount := 0
	for i := 0; i < ropeLen; i++ {
		if it.isNewlineAt(i) {
			lineCount++
		}
	}
	
	it.lineNum = lineCount
	it.currentPos = ropeLen
	it.lineStart = ropeLen
	it.lineEnd = ropeLen
	
	return it
}

// Next returns a new iterator pointing to the next line
func (it *BufferIterator) Next() *BufferIterator {
	if it.atEOF {
		return it.clone()
	}
	
	newIt := it.clone()
	
	// Move past current line end
	if newIt.lineEnd < newIt.rope.Length() {
		newIt.currentPos = newIt.lineEnd + 1 // Skip newline
		newIt.lineStart = newIt.currentPos
		newIt.lineNum++
		
		// Rebuild stack to current position
		newIt.rebuildStackToPosition(newIt.currentPos)
		
		// Find next line end
		newIt.findLineEnd()
	} else {
		newIt.atEOF = true
	}
	
	return newIt
}

// Previous returns a new iterator pointing to the previous line
func (it *BufferIterator) Previous() *BufferIterator {
	if it.lineNum == 0 {
		return it.clone()
	}
	
	newIt := it.clone()
	
	// Move back to find previous newline
	pos := newIt.lineStart - 1
	if pos >= 0 {
		// Skip the newline before current line
		if pos > 0 {
			pos--
		}
		
		// Find start of previous line
		lineEnd := pos + 1
		for pos > 0 && !newIt.isNewlineAt(pos - 1) {
			pos--
		}
		
		newIt.lineStart = pos
		newIt.lineEnd = lineEnd - 1 // Exclude newline
		newIt.currentPos = pos
		newIt.lineNum--
		
		// Rebuild stack to new position
		newIt.rebuildStackToPosition(pos)
	}
	
	return newIt
}

// Line returns the content of the current line (without newline)
func (it *BufferIterator) Line() string {
	if it.atEOF {
		return ""
	}
	
	// Extract just the current line from the rope
	return it.extractRange(it.lineStart, it.lineEnd)
}

// LineNumber returns the current line number (0-based)
func (it *BufferIterator) LineNumber() int {
	return it.lineNum
}

// IsEOF returns true if the iterator is past the last line
func (it *BufferIterator) IsEOF() bool {
	return it.atEOF
}

// Position returns the byte position of the current line start
func (it *BufferIterator) Position() int {
	return it.lineStart
}

// LineEnd returns the byte position of the current line end (excluding newline)
func (it *BufferIterator) LineEnd() int {
	return it.lineEnd
}

// Equals returns true if two iterators point to the same position
func (it *BufferIterator) Equals(other *BufferIterator) bool {
	return it.buffer == other.buffer && 
	       it.Position() == other.Position() && 
	       it.LineNumber() == other.LineNumber()
}

// clone creates a copy of the iterator
func (it *BufferIterator) clone() *BufferIterator {
	newStack := make([]walkFrame, len(it.stack))
	copy(newStack, it.stack)
	
	return &BufferIterator{
		buffer:     it.buffer,
		rope:       it.rope,
		stack:      newStack,
		currentPos: it.currentPos,
		lineStart:  it.lineStart,
		lineEnd:    it.lineEnd,
		lineNum:    it.lineNum,
		atEOF:      it.atEOF,
	}
}

// findLineEnd finds the end of current line from currentPos
func (it *BufferIterator) findLineEnd() {
	pos := it.currentPos
	maxPos := it.rope.Length()
	
	for pos < maxPos && !it.isNewlineAt(pos) {
		pos++
	}
	
	it.lineEnd = pos
}

// isNewlineAt checks if there's a newline at the given position
func (it *BufferIterator) isNewlineAt(pos int) bool {
	if pos < 0 || pos >= it.rope.Length() {
		return false
	}
	
	// Navigate to position and check character
	it.rebuildStackToPosition(pos)
	
	// Get character from current position
	if len(it.stack) > 0 {
		frame := it.stack[len(it.stack)-1]
		if leaf, ok := frame.node.(*Leaf); ok {
			localPos := pos - frame.nodeStart
			if localPos >= 0 && localPos < len(leaf.text) {
				return leaf.text[localPos] == '\n'
			}
		}
	}
	
	return false
}

// extractRange extracts text from start to end position
func (it *BufferIterator) extractRange(start, end int) string {
	if start >= end || it.rope.root == nil {
		return ""
	}
	
	// For now, use a simple approach that extracts from nodes
	// This can be optimized further to avoid any string concatenation
	result := make([]byte, 0, end-start)
	it.extractRangeFromNode(it.rope.root, 0, start, end, &result)
	return string(result)
}

// extractRangeFromNode extracts text from a node within the given range
func (it *BufferIterator) extractRangeFromNode(node Node, nodeStart, rangeStart, rangeEnd int, result *[]byte) {
	nodeEnd := nodeStart + node.Length()
	
	// Skip if node is outside range
	if nodeEnd <= rangeStart || nodeStart >= rangeEnd {
		return
	}
	
	switch n := node.(type) {
	case *Leaf:
		// Extract the relevant part of the leaf
		leafStart := max(0, rangeStart-nodeStart)
		leafEnd := min(len(n.text), rangeEnd-nodeStart)
		if leafStart < leafEnd {
			*result = append(*result, n.text[leafStart:leafEnd]...)
		}
		
	case *ConcatNode:
		// Recursively extract from children
		it.extractRangeFromNode(n.left, nodeStart, rangeStart, rangeEnd, result)
		it.extractRangeFromNode(n.right, nodeStart+n.weight, rangeStart, rangeEnd, result)
	}
}

// rebuildStackToPosition rebuilds the stack to reach a specific position
func (it *BufferIterator) rebuildStackToPosition(targetPos int) {
	it.stack = it.stack[:0]
	if it.rope.root == nil {
		return
	}
	
	it.navigateToPosition(it.rope.root, 0, targetPos)
}

// navigateToPosition recursively navigates to a position
func (it *BufferIterator) navigateToPosition(node Node, nodeStart int, targetPos int) {
	it.stack = append(it.stack, walkFrame{node: node, nodeStart: nodeStart})
	
	if concat, ok := node.(*ConcatNode); ok {
		leftEnd := nodeStart + concat.weight
		if targetPos < leftEnd {
			it.navigateToPosition(concat.left, nodeStart, targetPos)
		} else {
			it.navigateToPosition(concat.right, leftEnd, targetPos)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}