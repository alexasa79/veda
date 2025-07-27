package editor

// Insert mode movement commands - allow cursor to go past last character
func (e *Editor) moveUpInsert(args []string) error {
	if e.buffer.CursorIterator.LineNumber() > 0 {
		e.buffer.CursorIterator = e.buffer.CursorIterator.Previous()
		// Adjust cursor X if the new line is shorter
		line := e.buffer.CursorIterator.Line()
		if e.buffer.CursorX > len(line) {
			e.buffer.CursorX = len(line)
		}
	}
	return nil
}

func (e *Editor) moveDownInsert(args []string) error {
	if e.buffer.CursorIterator.LineNumber() < e.buffer.LineCount-1 {
		e.buffer.CursorIterator = e.buffer.CursorIterator.Next()
		// Adjust cursor X if the new line is shorter
		line := e.buffer.CursorIterator.Line()
		if e.buffer.CursorX > len(line) {
			e.buffer.CursorX = len(line)
		}
	}
	return nil
}

func (e *Editor) moveLeftInsert(args []string) error {
	if e.buffer.CursorX > 0 {
		e.buffer.CursorX--
	}
	return nil
}

func (e *Editor) moveRightInsert(args []string) error {
	line := e.buffer.CursorIterator.Line()
	// In insert mode, allow cursor to go past the last character
	if e.buffer.CursorX < len(line) {
		e.buffer.CursorX++
	}
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

