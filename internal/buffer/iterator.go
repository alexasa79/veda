package buffer

import "fmt"

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
	// Can't go before line 0
	if it.lineNum == 0 && !it.atEOF {
		return it.clone()
	}

	newIt := it.clone()
	
	// Special handling for EOF
	if it.atEOF {
		newIt.atEOF = false
		
		// Position at the last actual line based on lineNum
		ropeLen := newIt.rope.Length()
		
		// If empty buffer, can't go back
		if ropeLen == 0 {
			return newIt
		}
		
		// If lineNum is 0, position at line 0
		if it.lineNum == 0 {
			newIt.lineStart = 0
			newIt.lineEnd = ropeLen
			if ropeLen > 0 && newIt.isNewlineAt(ropeLen - 1) {
				newIt.lineEnd = ropeLen - 1
			}
			newIt.currentPos = 0
			newIt.rebuildStackToPosition(0)
			return newIt
		}
		
		// Otherwise, position at the line indicated by lineNum
		// Check if buffer ends with newline
		if newIt.isNewlineAt(ropeLen - 1) {
			// Buffer ends with newline, so the last line is empty
			newIt.lineStart = ropeLen
			newIt.lineEnd = ropeLen
			newIt.currentPos = ropeLen
		} else {
			// Find the start of the last line
			pos := ropeLen - 1
			for pos > 0 && !newIt.isNewlineAt(pos - 1) {
				pos--
			}
			newIt.lineStart = pos
			newIt.lineEnd = ropeLen
			newIt.currentPos = pos
		}
		
		// Rebuild stack to new position
		newIt.rebuildStackToPosition(newIt.currentPos)
		return newIt
	}
	
	// Regular Previous() logic for non-EOF case
	pos := newIt.lineStart - 1
	if pos >= 0 {
		// We're now at the last character of the previous line (or at a newline if prev line is empty)
		
		// Find the end of the previous line (before any newline)
		lineEnd := pos
		if pos >= 0 && newIt.isNewlineAt(pos) {
			// If we're at a newline, the previous line ends before it
			lineEnd = pos
		} else {
			// Otherwise, find the next newline
			lineEnd = pos + 1
		}
		
		// Now find the start of the previous line
		for pos > 0 && !newIt.isNewlineAt(pos - 1) {
			pos--
		}
		
		newIt.lineStart = pos
		newIt.lineEnd = lineEnd
		if lineEnd > pos && newIt.isNewlineAt(lineEnd - 1) {
			newIt.lineEnd = lineEnd - 1 // Exclude the newline
		}
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

// String implements the fmt.Stringer interface for BufferIterator
// This allows the iterator to be used with %v in printf and other formatting functions
func (it *BufferIterator) String() string {
	if it == nil {
		return "BufferIterator(nil)"
	}

	// Include key information about the iterator's state
	return fmt.Sprintf("BufferIterator(line=%d, eof=%v, pos=%d, start=%d, end=%d)",
		it.lineNum, it.atEOF, it.currentPos, it.lineStart, it.lineEnd)
}

// UpdateBuffer updates the iterator's buffer reference after the buffer has been modified
// This is necessary because the rope structure changes when content is inserted/deleted
func (it *BufferIterator) UpdateBuffer(buffer *Buffer) {
	if it == nil || buffer == nil {
		return
	}
	
	// Update buffer and rope references
	it.buffer = buffer
	it.rope = buffer.Rope
	
	// If we're at a valid position, rebuild the stack to maintain our position
	if !it.atEOF && it.currentPos >= 0 && it.currentPos <= it.rope.Length() {
		// We need to adjust currentPos if content was inserted before our position
		// For now, we'll recreate the iterator at the same line
		savedLine := it.lineNum
		
		// Reset to beginning
		it.currentPos = 0
		it.lineStart = 0
		it.lineNum = 0
		it.atEOF = false
		it.stack = make([]walkFrame, 0, 32)
		
		if it.rope.root != nil {
			it.stack = append(it.stack, walkFrame{node: it.rope.root, nodeStart: 0})
			it.findLineEnd()
			
			// Move to the saved line
			for i := 0; i < savedLine && !it.atEOF; i++ {
				// Move past current line end
				if it.lineEnd < it.rope.Length() {
					it.currentPos = it.lineEnd + 1 // Skip newline
					it.lineStart = it.currentPos
					it.lineNum++
					it.rebuildStackToPosition(it.currentPos)
					it.findLineEnd()
				} else {
					it.atEOF = true
					break
				}
			}
		} else {
			it.atEOF = true
		}
	} else if it.currentPos > it.rope.Length() {
		// Position is beyond the new buffer length, move to EOF
		it.atEOF = true
		it.currentPos = it.rope.Length()
		it.lineStart = it.currentPos
		it.lineEnd = it.currentPos
	}
}