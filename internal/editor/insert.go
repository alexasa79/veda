package editor

// Insert mode movement commands - allow cursor to go past last character
func (e *Editor) moveUpInsert(args []string) error {
	if e.buffer.CursorIterator.LineNumber() > 0 {
		// Store current X position as desired column if not set
		if e.desiredColumn == -1 {
			e.desiredColumn = e.buffer.CursorX
		}

		e.buffer.CursorIterator = e.buffer.CursorIterator.Previous()

		// Try to maintain desired column position
		line := e.buffer.CursorIterator.Line()
		if e.desiredColumn <= len(line) {
			e.buffer.CursorX = e.desiredColumn
		} else {
			e.buffer.CursorX = len(line)
		}
	}
	return nil
}

func (e *Editor) moveDownInsert(args []string) error {
	if e.buffer.CursorIterator.LineNumber() < e.buffer.LineCount-1 {
		// Store current X position as desired column if not set
		if e.desiredColumn == -1 {
			e.desiredColumn = e.buffer.CursorX
		}

		e.buffer.CursorIterator = e.buffer.CursorIterator.Next()

		// Try to maintain desired column position
		line := e.buffer.CursorIterator.Line()
		if e.desiredColumn <= len(line) {
			e.buffer.CursorX = e.desiredColumn
		} else {
			e.buffer.CursorX = len(line)
		}
	}
	return nil
}

func (e *Editor) moveLeftInsert(args []string) error {
	if e.buffer.CursorX > 0 {
		e.buffer.CursorX--
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	return nil
}

func (e *Editor) moveRightInsert(args []string) error {
	line := e.buffer.CursorIterator.Line()
	// In insert mode, allow cursor to go past the last character
	if e.buffer.CursorX < len(line) {
		e.buffer.CursorX++
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	return nil
}

// beginningOfLineInsert is the same as beginningOfLine but for insert mode
func (e *Editor) beginningOfLineInsert(args []string) error {
	return e.beginningOfLine(args)
}

// endOfLineInsert moves cursor past the end of line for insert mode
func (e *Editor) endOfLineInsert(args []string) error {
	line := e.buffer.CursorIterator.Line()
	e.buffer.CursorX = len(line)
	e.updateCursorState()
	return nil
}
