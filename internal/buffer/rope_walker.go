package buffer

// RopeWalker provides efficient traversal of a rope without converting to string
type RopeWalker struct {
	rope      *Rope
	stack     []walkFrame  // Stack for tree traversal
	currentPos int         // Current byte position
	lineStart  int         // Start position of current line
	lineEnd    int         // End position of current line (excluding \n)
	lineNum    int         // Current line number
	atEOF      bool        // True if at end of file
}

type walkFrame struct {
	node      Node
	nodeStart int  // Byte position where this node starts
	isLeft    bool // True if we're in left subtree
}

// NewRopeWalker creates a walker positioned at the beginning
func NewRopeWalker(rope *Rope) *RopeWalker {
	w := &RopeWalker{
		rope:       rope,
		stack:      make([]walkFrame, 0, 32),
		currentPos: 0,
		lineStart:  0,
		lineNum:    0,
		atEOF:      false,
	}
	
	if rope.root == nil {
		w.atEOF = true
		return w
	}
	
	// Initialize stack with root
	w.stack = append(w.stack, walkFrame{node: rope.root, nodeStart: 0})
	
	// Find first line end
	w.findLineEnd()
	
	return w
}

// NewRopeWalkerAtEnd creates a walker positioned at EOF
func NewRopeWalkerAtEnd(rope *Rope) *RopeWalker {
	w := &RopeWalker{
		rope:       rope,
		stack:      make([]walkFrame, 0, 32),
		currentPos: 0,
		lineStart:  0,
		lineNum:    0,
		atEOF:      true,
	}
	
	if rope.root == nil {
		return w
	}
	
	// Count lines to set lineNum correctly
	ropeLen := rope.Length()
	lineCount := 0
	for i := 0; i < ropeLen; i++ {
		if w.isNewlineAt(i) {
			lineCount++
		}
	}
	
	w.lineNum = lineCount
	w.currentPos = ropeLen
	w.lineStart = ropeLen
	w.lineEnd = ropeLen
	
	return w
}

// NextLine moves to the next line
func (w *RopeWalker) NextLine() *RopeWalker {
	if w.atEOF {
		return w.clone()
	}
	
	newWalker := w.clone()
	
	// Move past current line end
	if newWalker.lineEnd < newWalker.rope.Length() {
		newWalker.currentPos = newWalker.lineEnd + 1 // Skip newline
		newWalker.lineStart = newWalker.currentPos
		newWalker.lineNum++
		
		// Rebuild stack to current position
		newWalker.rebuildStackToPosition(newWalker.currentPos)
		
		// Find next line end
		newWalker.findLineEnd()
	} else {
		newWalker.atEOF = true
	}
	
	return newWalker
}

// PreviousLine moves to the previous line
func (w *RopeWalker) PreviousLine() *RopeWalker {
	if w.lineNum == 0 {
		return w.clone()
	}
	
	newWalker := w.clone()
	
	// Move back to find previous newline
	pos := newWalker.lineStart - 1
	if pos >= 0 {
		// Skip the newline before current line
		if pos > 0 {
			pos--
		}
		
		// Find start of previous line
		lineEnd := pos + 1
		for pos > 0 && !newWalker.isNewlineAt(pos - 1) {
			pos--
		}
		
		newWalker.lineStart = pos
		newWalker.lineEnd = lineEnd - 1 // Exclude newline
		newWalker.currentPos = pos
		newWalker.lineNum--
		
		// Rebuild stack to new position
		newWalker.rebuildStackToPosition(pos)
	}
	
	return newWalker
}

// GetLine returns the current line content without converting entire rope to string
func (w *RopeWalker) GetLine() string {
	if w.atEOF {
		return ""
	}
	
	// Extract just the current line from the rope
	return w.extractRange(w.lineStart, w.lineEnd)
}

// LineNumber returns the current line number
func (w *RopeWalker) LineNumber() int {
	return w.lineNum
}

// IsEOF returns true if at end of file
func (w *RopeWalker) IsEOF() bool {
	return w.atEOF
}

// Position returns the byte position of line start
func (w *RopeWalker) Position() int {
	return w.lineStart
}

// LineEnd returns the byte position of line end
func (w *RopeWalker) LineEndPos() int {
	return w.lineEnd
}

// clone creates a copy of the walker
func (w *RopeWalker) clone() *RopeWalker {
	newStack := make([]walkFrame, len(w.stack))
	copy(newStack, w.stack)
	
	return &RopeWalker{
		rope:       w.rope,
		stack:      newStack,
		currentPos: w.currentPos,
		lineStart:  w.lineStart,
		lineEnd:    w.lineEnd,
		lineNum:    w.lineNum,
		atEOF:      w.atEOF,
	}
}

// findLineEnd finds the end of current line from currentPos
func (w *RopeWalker) findLineEnd() {
	pos := w.currentPos
	maxPos := w.rope.Length()
	
	for pos < maxPos && !w.isNewlineAt(pos) {
		pos++
	}
	
	w.lineEnd = pos
}

// isNewlineAt checks if there's a newline at the given position
func (w *RopeWalker) isNewlineAt(pos int) bool {
	if pos < 0 || pos >= w.rope.Length() {
		return false
	}
	
	// Navigate to position and check character
	w.rebuildStackToPosition(pos)
	
	// Get character from current position
	if len(w.stack) > 0 {
		frame := w.stack[len(w.stack)-1]
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
func (w *RopeWalker) extractRange(start, end int) string {
	if start >= end || w.rope.root == nil {
		return ""
	}
	
	// For now, use a simple approach that extracts from nodes
	// This can be optimized further to avoid any string concatenation
	result := make([]byte, 0, end-start)
	w.extractRangeFromNode(w.rope.root, 0, start, end, &result)
	return string(result)
}

// extractRangeFromNode extracts text from a node within the given range
func (w *RopeWalker) extractRangeFromNode(node Node, nodeStart, rangeStart, rangeEnd int, result *[]byte) {
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
		w.extractRangeFromNode(n.left, nodeStart, rangeStart, rangeEnd, result)
		w.extractRangeFromNode(n.right, nodeStart+n.weight, rangeStart, rangeEnd, result)
	}
}

// rebuildStackToPosition rebuilds the stack to reach a specific position
func (w *RopeWalker) rebuildStackToPosition(targetPos int) {
	w.stack = w.stack[:0]
	if w.rope.root == nil {
		return
	}
	
	w.navigateToPosition(w.rope.root, 0, targetPos)
}

// navigateToPosition recursively navigates to a position
func (w *RopeWalker) navigateToPosition(node Node, nodeStart int, targetPos int) {
	w.stack = append(w.stack, walkFrame{node: node, nodeStart: nodeStart})
	
	if concat, ok := node.(*ConcatNode); ok {
		leftEnd := nodeStart + concat.weight
		if targetPos < leftEnd {
			w.navigateToPosition(concat.left, nodeStart, targetPos)
		} else {
			w.navigateToPosition(concat.right, leftEnd, targetPos)
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