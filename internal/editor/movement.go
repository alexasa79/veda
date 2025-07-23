package editor

import (
	"unicode"
	"veda/internal/buffer"
	"veda/internal/commands"
)

// Movement commands
func (e *Editor) moveUp() error {
	if e.buffer.CursorIterator.LineNumber() > 0 {
		e.buffer.CursorIterator = e.buffer.CursorIterator.Previous()
		// Adjust cursor X if the new line is shorter
		line := e.buffer.CursorIterator.Line()
		maxX := len(line)
		if maxX > 0 {
			maxX-- // Don't go past the last character
		}
		if e.buffer.CursorX > maxX {
			e.buffer.CursorX = maxX
		}
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveDown() error {
	if e.buffer.CursorIterator.LineNumber() < e.buffer.LineCount-1 {
		e.buffer.CursorIterator = e.buffer.CursorIterator.Next()
		// Adjust cursor X if the new line is shorter
		line := e.buffer.CursorIterator.Line()
		maxX := len(line)
		if maxX > 0 {
			maxX-- // Don't go past the last character
		}
		if e.buffer.CursorX > maxX {
			e.buffer.CursorX = maxX
		}
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveLeft() error {
	if e.buffer.CursorX > 0 {
		e.buffer.CursorX--
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveRight() error {
	line := e.buffer.CursorIterator.Line()
	maxX := len(line)
	if maxX > 0 {
		maxX-- // Don't go past the last character
	}
	if e.buffer.CursorX < maxX {
		e.buffer.CursorX++
	}
	e.updateCursorState()
	return nil
}

// wordForward moves the cursor to the beginning of the next word
func (e *Editor) wordForward() error {
	// Start from current position
	iter := e.buffer.CursorIterator
	x := e.buffer.CursorX + 1 // Start searching from next character
	currentLine := iter.Line()

	// Track if we've seen a non-alphanumeric character
	seenNonAlnum := false

	// If we start on a non-alphanumeric character, we're ready to find the next word
	if e.buffer.CursorX < len(currentLine) {
		ch := rune(currentLine[e.buffer.CursorX])
		seenNonAlnum = !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_'
	}

	// Loop through characters until we find a transition from non-alnum to alnum
	for !iter.IsEOF() {
		line := iter.Line()

		for x < len(line) {
			ch := rune(line[x])
			isAlnum := unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'

			if seenNonAlnum && isAlnum {
				// Found transition from non-alnum to alnum
				e.buffer.CursorX = x
				e.buffer.CursorIterator = iter
				return nil
			}

			if !isAlnum {
				seenNonAlnum = true
			}

			x++
		}

		// Move to next line
		iter = iter.Next()
		if !iter.IsEOF() {
			x = 0
			seenNonAlnum = true // Line break counts as non-alnum
		}
	}

	// No more words found, stay at current position
	return nil
}

// wordBackward moves the cursor to the beginning of the previous word
func (e *Editor) wordBackward() error {
	// If we're at the very beginning, can't go back
	if e.buffer.CursorIterator.LineNumber() == 0 && e.buffer.CursorX == 0 {
		return nil
	}

	// Start from current position
	iter := e.buffer.CursorIterator
	x := e.buffer.CursorX
	currentLine := iter.Line()

	// Helper to check if character is word character
	isWordChar := func(ch rune) bool {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
	}

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
func (e *Editor) beginningOfLine() error {
	e.buffer.CursorX = 0
	e.updateCursorState()
	return nil
}

// endOfLine moves cursor to the end of the current line
func (e *Editor) endOfLine() error {
	line := e.buffer.CursorIterator.Line()
	lineLen := len(line)
	if lineLen > 0 {
		e.buffer.CursorX = lineLen - 1
	} else {
		e.buffer.CursorX = 0
	}
	return nil
}

func (e *Editor) pageUp() error {
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

func (e *Editor) pageDown() error {
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
func (e *Editor) goToBeginning() error {
	e.buffer.CursorX = 0
	e.buffer.ViewIterator = buffer.NewBufferIterator(e.buffer)
	e.buffer.CursorIterator = buffer.NewBufferIterator(e.buffer)
	e.updateCursorState()
	return nil
}

// goToEnd moves the cursor to the end of the file
func (e *Editor) goToEnd() error {
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
