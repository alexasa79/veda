package buffer

import "fmt"

// BufferIterator provides line-by-line iteration over a Buffer.
// This implementation only tracks the line number, computing everything else on-demand.
type BufferIterator struct {
	buffer  *Buffer
	lineNum int // Current line number (0-based), or points past last line if at EOF
}

// NewBufferIterator creates a new iterator at the beginning of the buffer
func NewBufferIterator(buffer *Buffer) *BufferIterator {
	return &BufferIterator{
		buffer:  buffer,
		lineNum: 0,
	}
}

// NewBufferIteratorAtEnd creates a new iterator at the end of the buffer
func NewBufferIteratorAtEnd(buffer *Buffer) *BufferIterator {
	it := &BufferIterator{
		buffer: buffer,
	}
	// Set lineNum to the total number of lines, which makes IsEOF() return true
	it.lineNum = it.countLines()
	return it
}

// countLines returns the total number of lines in the buffer
func (it *BufferIterator) countLines() int {
	length := it.buffer.Rope.Length()
	if length == 0 {
		return 0
	}

	// Start with one line
	lineCount := 1

	// Count newlines
	for i := 0; i < length; i++ {
		if it.buffer.Rope.ByteAt(i) == '\n' {
			lineCount++
		}
	}

	// If buffer ends with newline, last line is empty (already counted)
	// If buffer doesn't end with newline, we already counted it
	return lineCount
}

// getLineInfo computes the start and end positions of the current line
func (it *BufferIterator) getLineInfo() (start, end int, exists bool) {
	// Check if we're at EOF
	totalLines := it.countLines()
	if it.lineNum >= totalLines {
		return 0, 0, false
	}

	// Find the start of our line by counting newlines
	pos := 0
	currentLine := 0
	length := it.buffer.Rope.Length()

	// Find start of target line
	for pos < length && currentLine < it.lineNum {
		if it.buffer.Rope.ByteAt(pos) == '\n' {
			currentLine++
		}
		pos++
	}

	start = pos

	// Find end of line
	for pos < length && it.buffer.Rope.ByteAt(pos) != '\n' {
		pos++
	}
	end = pos

	return start, end, true
}

// Next returns a new iterator pointing to the next line
func (it *BufferIterator) Next() *BufferIterator {
	if it.IsEOF() {
		return &BufferIterator{
			buffer:  it.buffer,
			lineNum: it.lineNum,
		}
	}

	return &BufferIterator{
		buffer:  it.buffer,
		lineNum: it.lineNum + 1,
	}
}

// Previous returns a new iterator pointing to the previous line
func (it *BufferIterator) Previous() *BufferIterator {
	if it.lineNum == 0 {
		return &BufferIterator{
			buffer:  it.buffer,
			lineNum: 0,
		}
	}

	newLineNum := it.lineNum - 1

	// Make sure we don't go below 0
	if newLineNum < 0 {
		newLineNum = 0
	}

	return &BufferIterator{
		buffer:  it.buffer,
		lineNum: newLineNum,
	}
}

// Line returns the current line content (without trailing newline)
func (it *BufferIterator) Line() string {
	start, end, exists := it.getLineInfo()
	if !exists {
		return ""
	}

	lineLen := end - start
	if lineLen <= 0 {
		return ""
	}

	// Extract line content
	result := make([]byte, lineLen)
	for i := 0; i < lineLen; i++ {
		result[i] = it.buffer.Rope.ByteAt(start + i)
	}

	return string(result)
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
	return it.lineNum >= it.countLines()
}

// Position returns the current byte position in the buffer
func (it *BufferIterator) Position() int {
	start, _, exists := it.getLineInfo()
	if !exists {
		return it.buffer.Rope.Length()
	}
	return start
}

// LineStart returns the byte position of the start of the current line
func (it *BufferIterator) LineStart() int {
	start, _, exists := it.getLineInfo()
	if !exists {
		return it.buffer.Rope.Length()
	}
	return start
}

// LineEnd returns the byte position of the end of the current line (excluding newline)
func (it *BufferIterator) LineEnd() int {
	_, end, exists := it.getLineInfo()
	if !exists {
		return it.buffer.Rope.Length()
	}
	return end
}

// Equals returns true if two iterators point to the same position
func (it *BufferIterator) Equals(other *BufferIterator) bool {
	if it == nil || other == nil {
		return it == other
	}
	return it.lineNum == other.lineNum && it.buffer == other.buffer
}

// String implements the fmt.Stringer interface for BufferIterator
func (it *BufferIterator) String() string {
	if it == nil {
		return "BufferIterator(nil)"
	}

	start, end, _ := it.getLineInfo()
	return fmt.Sprintf("BufferIterator(line=%d, eof=%v, pos=%d, start=%d, end=%d)",
		it.lineNum, it.IsEOF(), it.Position(), start, end)
}