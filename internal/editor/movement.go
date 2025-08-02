package editor

import (
	"veda/internal/buffer"
	"veda/internal/commands"
)

// Movement commands
func (e *Editor) moveUp(args []string) error {
	// Get current visual position
	visualLine, visualCol := e.GetCursorVisualPosition()

	if visualLine > 0 {
		// Map to new position one visual line up
		newVisualLine := visualLine - 1
		newLogicalLine, newCharPos := e.MapVisualToLogical(newVisualLine, visualCol)

		// Update cursor position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, newLogicalLine)
		e.buffer.CursorX = newCharPos

		// Ensure cursor is within bounds for normal mode
		if e.mode == commands.NormalMode && !e.buffer.CursorIterator.IsEOF() {
			line := e.buffer.CursorIterator.Line()
			if e.buffer.CursorX > len(line) {
				e.buffer.CursorX = len(line)
			}
			if e.buffer.CursorX > 0 && len(line) > 0 {
				e.buffer.CursorX-- // Don't go past last character in normal mode
			}
		}
	}

	e.updateCursorState()
	return nil
}

func (e *Editor) moveDown(args []string) error {
	// Get current visual position
	visualLine, visualCol := e.GetCursorVisualPosition()

	// Calculate wrapped lines to check bounds
	wrappedLines := e.CalculateWrappedLines()

	if visualLine < len(wrappedLines)-1 {
		// Map to new position one visual line down
		newVisualLine := visualLine + 1
		newLogicalLine, newCharPos := e.MapVisualToLogical(newVisualLine, visualCol)

		// Update cursor position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, newLogicalLine)
		e.buffer.CursorX = newCharPos

		// Ensure cursor is within bounds for normal mode
		if e.mode == commands.NormalMode && !e.buffer.CursorIterator.IsEOF() {
			line := e.buffer.CursorIterator.Line()
			if e.buffer.CursorX > len(line) {
				e.buffer.CursorX = len(line)
			}
			if e.buffer.CursorX > 0 && len(line) > 0 {
				e.buffer.CursorX-- // Don't go past last character in normal mode
			}
		}
	}

	e.updateCursorState()
	return nil
}

func (e *Editor) moveLeft(args []string) error {
	if e.buffer.CursorX > 0 {
		e.buffer.CursorX--
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveRight(args []string) error {
	line := e.buffer.CursorIterator.Line()
	maxX := len(line)
	if maxX > 0 {
		maxX-- // Don't go past the last character
	}
	if e.buffer.CursorX < maxX {
		// Simply move to the next character - the visual rendering handles tab width
		e.buffer.CursorX++
	}
	e.updateCursorState()
	return nil
}

// wordForward moves the cursor to the beginning of the next word
func (e *Editor) wordForward(args []string) error {
	currentLine := e.buffer.CursorIterator.LineNumber()
	nextWord := FindNextWordBoundary(e.buffer, currentLine, e.buffer.CursorX)

	if nextWord != nil {
		// Move cursor to the next word
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, nextWord.StartLine)
		e.buffer.CursorX = nextWord.StartChar
	}

	e.updateCursorState()
	return nil
}

// wordBackward moves the cursor to the beginning of the previous word
func (e *Editor) wordBackward(args []string) error {
	// If we're at the very beginning, can't go back
	if e.buffer.CursorIterator.LineNumber() == 0 && e.buffer.CursorX == 0 {
		return nil
	}

	// Start from current position
	iter := e.buffer.CursorIterator
	x := e.buffer.CursorX
	currentLine := iter.Line()


	// Move left one position
	x--
	if x < 0 {
		// Move to previous line
		iter = iter.Previous()
		if iter.LineNumber() == e.buffer.CursorIterator.LineNumber() {
			// Couldn't move back, we're at beginning
			return nil
		}
		currentLine = iter.Line()
		x = len(currentLine) - 1
		if x < 0 {
			x = 0
		}
	}

	// State: are we looking for a word start?
	lookingForWord := true

	// If we're on a word character, we need to skip to the beginning of this word
	if x < len(currentLine) && isWordChar(rune(currentLine[x])) {
		lookingForWord = false
	}

	// Search backwards
	for {
		line := iter.Line()

		for x >= 0 {
			if x < len(line) {
				ch := rune(line[x])
				if lookingForWord {
					// Looking for a word character
					if isWordChar(ch) {
						// Found end of a word, now find its beginning
						lookingForWord = false
					}
				} else {
					// In a word, looking for its beginning
					if !isWordChar(ch) {
						// Found non-word character, word starts at x+1
						e.buffer.CursorX = x + 1
						e.buffer.CursorIterator = iter
						return nil
					}
				}
			}

			x--
		}

		// If we were in a word and reached beginning of line
		if !lookingForWord {
			e.buffer.CursorX = 0
			e.buffer.CursorIterator = iter
			return nil
		}

		// Move to previous line
		prevIter := iter.Previous()
		if prevIter.LineNumber() == iter.LineNumber() {
			// Can't go back further
			break
		}
		iter = prevIter
		x = len(iter.Line()) - 1
		lookingForWord = true // Line break is a word boundary
	}

	// Reached beginning of buffer
	return nil
}

// beginningOfLine moves cursor to the beginning of the current line
func (e *Editor) beginningOfLine(args []string) error {
	e.buffer.CursorX = 0
	e.updateCursorState()
	return nil
}

// endOfLine moves cursor to the end of the current line
func (e *Editor) endOfLine(args []string) error {
	line := e.buffer.CursorIterator.Line()
	lineLen := len(line)
	if lineLen > 0 {
		e.buffer.CursorX = lineLen - 1
	} else {
		e.buffer.CursorX = 0
	}
	return nil
}

func (e *Editor) pageUp(args []string) error {
	visibleLines := e.getVisibleLines()

	// Move cursor up by visible lines
	for i := 0; i < visibleLines && e.buffer.CursorIterator.LineNumber() > 0; i++ {
		e.buffer.CursorIterator = e.buffer.CursorIterator.Previous()
	}

	// Adjust cursor X if needed
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(line) {
		e.buffer.CursorX = len(line)
		if e.buffer.CursorX > 0 {
			e.buffer.CursorX--
		}
	}
	return nil
}

func (e *Editor) pageDown(args []string) error {
	visibleLines := e.getVisibleLines()

	// Move cursor down by visible lines
	maxY := e.buffer.LineCount - 1
	for i := 0; i < visibleLines && e.buffer.CursorIterator.LineNumber() < maxY; i++ {
		e.buffer.CursorIterator = e.buffer.CursorIterator.Next()
	}

	// Adjust cursor X if needed
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(line) {
		e.buffer.CursorX = len(line)
		if e.buffer.CursorX > 0 {
			e.buffer.CursorX--
		}
	}
	return nil
}

// goToBeginning moves the cursor to the beginning of the file
func (e *Editor) goToBeginning(args []string) error {
	e.buffer.CursorX = 0
	e.buffer.ViewIterator = buffer.NewBufferIterator(e.buffer)
	e.buffer.CursorIterator = buffer.NewBufferIterator(e.buffer)
	e.updateCursorState()
	return nil
}

// goToEnd moves the cursor to the end of the file
func (e *Editor) goToEnd(args []string) error {
	// Create iterator at EOF
	endIter := buffer.NewBufferIteratorAtEnd(e.buffer)

	// Go back one line to the last actual line
	prevIter := endIter.Previous()

	// Check if we actually moved (not empty buffer)
	if !prevIter.Equals(endIter) {
		e.buffer.CursorIterator = prevIter

		// Move to end of last line (but stay on last character in normal mode)
		lastLine := e.buffer.CursorIterator.Line()
		e.buffer.CursorX = len(lastLine)
		if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
			e.buffer.CursorX--
		}
	} else {
		// Empty buffer
		e.buffer.CursorIterator = buffer.NewBufferIterator(e.buffer)
		e.buffer.CursorX = 0
	}

	// Scroll to show the cursor
	e.ScrollToShowAtBottom()

	return nil
}

// goToLine moves the cursor to a specific line number (0-based)
func (e *Editor) goToLine(lineNum int) {
	// Clamp line number to valid range
	if lineNum < 0 {
		lineNum = 0
	} else if lineNum >= e.buffer.LineCount {
		lineNum = e.buffer.LineCount - 1
	}

	// Move cursor to beginning of the target line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.CursorX = 0

	// Update the view to ensure the cursor is visible
	e.scrollToCenter()
	e.updateCursorState()
}
