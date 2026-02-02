package editor

// Insert mode movement commands - allow cursor to go past last character
func (e *Editor) moveUpInsert(args []string) error {
	if e.CurrentBuffer().CursorIterator.LineNumber() > 0 {
		// Store current X position as desired column if not set
		if e.desiredColumn == -1 {
			e.desiredColumn = e.CurrentBuffer().CursorX
		}

		e.CurrentBuffer().CursorIterator = e.CurrentBuffer().CursorIterator.Previous()

		// Try to maintain desired column position
		line := e.CurrentBuffer().CursorIterator.Line()
		if e.desiredColumn <= len(line) {
			e.CurrentBuffer().CursorX = e.desiredColumn
		} else {
			e.CurrentBuffer().CursorX = len(line)
		}
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveDownInsert(args []string) error {
	if e.CurrentBuffer().CursorIterator.LineNumber() < e.CurrentBuffer().LineCount-1 {
		// Store current X position as desired column if not set
		if e.desiredColumn == -1 {
			e.desiredColumn = e.CurrentBuffer().CursorX
		}

		e.CurrentBuffer().CursorIterator = e.CurrentBuffer().CursorIterator.Next()

		// Try to maintain desired column position
		line := e.CurrentBuffer().CursorIterator.Line()
		if e.desiredColumn <= len(line) {
			e.CurrentBuffer().CursorX = e.desiredColumn
		} else {
			e.CurrentBuffer().CursorX = len(line)
		}
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveLeftInsert(args []string) error {
	// Close any active completion menu when moving with arrows in insert mode
	e.hideCompletionMenu()
	if e.CurrentBuffer().CursorX > 0 {
		e.CurrentBuffer().CursorX--
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
	return nil
}

func (e *Editor) moveRightInsert(args []string) error {
	// Close any active completion menu when moving with arrows in insert mode
	e.hideCompletionMenu()
	line := e.CurrentBuffer().CursorIterator.Line()
	// In insert mode, allow cursor to go past the last character
	if e.CurrentBuffer().CursorX < len(line) {
		e.CurrentBuffer().CursorX++
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
	return nil
}

// beginningOfLineInsert is the same as beginningOfLine but for insert mode
func (e *Editor) beginningOfLineInsert(args []string) error {
	return e.beginningOfLine(args)
}

// endOfLineInsert moves cursor past the end of line for insert mode
func (e *Editor) endOfLineInsert(args []string) error {
	line := e.CurrentBuffer().CursorIterator.Line()
	e.CurrentBuffer().CursorX = len(line)
	e.updateCursorState()
	return nil
}
