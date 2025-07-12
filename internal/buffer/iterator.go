package buffer

// BufferIterator provides efficient line-by-line iteration over a Buffer
type BufferIterator struct {
	buffer *Buffer
	walker *RopeWalker // Use RopeWalker for efficient traversal
}

// NewBufferIterator creates a new iterator at the beginning of the buffer
func NewBufferIterator(buffer *Buffer) *BufferIterator {
	return &BufferIterator{
		buffer: buffer,
		walker: NewRopeWalker(buffer.Rope),
	}
}

// NewBufferIteratorAtEnd creates a new iterator at the end of the buffer
func NewBufferIteratorAtEnd(buffer *Buffer) *BufferIterator {
	return &BufferIterator{
		buffer: buffer,
		walker: NewRopeWalkerAtEnd(buffer.Rope),
	}
}

// Next returns a new iterator pointing to the next line
func (it *BufferIterator) Next() *BufferIterator {
	return &BufferIterator{
		buffer: it.buffer,
		walker: it.walker.NextLine(),
	}
}

// Previous returns a new iterator pointing to the previous line
func (it *BufferIterator) Previous() *BufferIterator {
	return &BufferIterator{
		buffer: it.buffer,
		walker: it.walker.PreviousLine(),
	}
}

// Line returns the content of the current line (without newline)
func (it *BufferIterator) Line() string {
	return it.walker.GetLine()
}

// LineNumber returns the current line number (0-based)
func (it *BufferIterator) LineNumber() int {
	return it.walker.LineNumber()
}

// IsEOF returns true if the iterator is past the last line
func (it *BufferIterator) IsEOF() bool {
	return it.walker.IsEOF()
}

// Position returns the byte position of the current line start
func (it *BufferIterator) Position() int {
	return it.walker.Position()
}

// LineEnd returns the byte position of the current line end (excluding newline)
func (it *BufferIterator) LineEnd() int {
	return it.walker.LineEndPos()
}

// Equals returns true if two iterators point to the same position
func (it *BufferIterator) Equals(other *BufferIterator) bool {
	return it.buffer == other.buffer && 
	       it.walker.Position() == other.walker.Position() && 
	       it.walker.LineNumber() == other.walker.LineNumber()
}