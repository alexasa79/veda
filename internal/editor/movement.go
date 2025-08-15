package editor

import (
	"veda/internal/buffer"
	"veda/internal/commands"
)

// Movement commands
func (e *Editor) moveUp(args []string) error {
	// Check for auto-indent cleanup before moving
	e.checkAndCleanupAutoIndent()
	
	// Get current visual position
	visualLine, visualCol := e.GetCursorVisualPosition()
	
	// Set desired column if not already set
	if e.desiredColumn == -1 {
		e.desiredColumn = visualCol
	}

	if visualLine > 0 {
		// Map to new position one visual line up, using desired column
		newVisualLine := visualLine - 1
		newLogicalLine, newCharPos := e.MapVisualToLogical(newVisualLine, e.desiredColumn)

		// Update cursor position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, newLogicalLine)
		e.buffer.CursorX = newCharPos

		// Ensure cursor is within bounds for normal mode
		if e.mode == commands.NormalMode && !e.buffer.CursorIterator.IsEOF() {
			line := e.buffer.CursorIterator.Line()
			if len(line) > 0 && e.buffer.CursorX >= len(line) {
				e.buffer.CursorX = len(line) - 1 // Place on last character
			}
		}
	}

	e.updateCursorState()
	return nil
}

func (e *Editor) moveDown(args []string) error {
	// Check for auto-indent cleanup before moving
	e.checkAndCleanupAutoIndent()
	
	// Get current visual position
	visualLine, visualCol := e.GetCursorVisualPosition()
	
	// Set desired column if not already set
	if e.desiredColumn == -1 {
		e.desiredColumn = visualCol
	}

	// Calculate wrapped lines to check bounds
	wrappedLines := e.CalculateWrappedLines()

	if visualLine < len(wrappedLines)-1 {
		// Map to new position one visual line down, using desired column
		newVisualLine := visualLine + 1
		newLogicalLine, newCharPos := e.MapVisualToLogical(newVisualLine, e.desiredColumn)

		// Update cursor position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, newLogicalLine)
		e.buffer.CursorX = newCharPos

		// Ensure cursor is within bounds for normal mode
		if e.mode == commands.NormalMode && !e.buffer.CursorIterator.IsEOF() {
			line := e.buffer.CursorIterator.Line()
			if len(line) > 0 && e.buffer.CursorX >= len(line) {
				e.buffer.CursorX = len(line) - 1 // Place on last character
			}
		}
	}

	e.updateCursorState()
	return nil
}

func (e *Editor) moveLeft(args []string) error {
	// Auto-indent cleanup not needed for horizontal movement on same line
	if e.buffer.CursorX > 0 {
		e.buffer.CursorX--
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
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
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
	return nil
}

// wordForward moves the cursor to the beginning of the next word
// Enhanced to stop at end of line when a word extends to the line boundary
func (e *Editor) wordForward(args []string) error {
	currentLine := e.buffer.CursorIterator.LineNumber()
	startX := e.buffer.CursorX
	line := e.buffer.CursorIterator.Line()
	
	// Check if we should stop at end of line first
	if shouldStopAtEndOfLine(e.buffer, currentLine, startX) {
		// Move to end of current line
		e.buffer.CursorX = len(line) - 1
		e.desiredColumn = -1
		e.updateCursorState()
		return nil
	}
	
	// Use normal word boundary logic
	nextWord := FindNextWordBoundary(e.buffer, currentLine, startX)

	if nextWord != nil {
		// Move cursor to the next word
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, nextWord.StartLine)
		e.buffer.CursorX = nextWord.StartChar
	}

	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
	return nil
}

// wordBackward moves the cursor to the beginning of the previous word
func (e *Editor) wordBackward(args []string) error {
	currentLine := e.buffer.CursorIterator.LineNumber()
	prevWord := FindPreviousWordBoundary(e.buffer, currentLine, e.buffer.CursorX)

	if prevWord != nil {
		// Move cursor to the previous word
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, prevWord.StartLine)
		e.buffer.CursorX = prevWord.StartChar
	}

	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
	return nil
}

// wordForwardNoUnderscore moves the cursor to the beginning of the next word (W command)
// Treats underscores as word boundaries, not as part of words
func (e *Editor) wordForwardNoUnderscore(args []string) error {
	currentLine := e.buffer.CursorIterator.LineNumber()
	startX := e.buffer.CursorX
	line := e.buffer.CursorIterator.Line()
	
	// Check if we should stop at end of line first (using underscore-aware logic)
	if shouldStopAtEndOfLineWithOptions(e.buffer, currentLine, startX, false) {
		// Move to end of current line
		e.buffer.CursorX = len(line) - 1
		e.desiredColumn = -1
		e.updateCursorState()
		return nil
	}
	
	// Use word boundary logic with underscores as boundaries
	nextWord := FindNextWordBoundaryWithOptions(e.buffer, currentLine, startX, false)

	if nextWord != nil {
		// Move cursor to the next word
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, nextWord.StartLine)
		e.buffer.CursorX = nextWord.StartChar
	}

	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
	return nil
}

// wordBackwardNoUnderscore moves the cursor to the beginning of the previous word (B command)
// Treats underscores as word boundaries, not as part of words
func (e *Editor) wordBackwardNoUnderscore(args []string) error {
	currentLine := e.buffer.CursorIterator.LineNumber()
	prevWord := FindPreviousWordBoundaryWithOptions(e.buffer, currentLine, e.buffer.CursorX, false)

	if prevWord != nil {
		// Move cursor to the previous word
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, prevWord.StartLine)
		e.buffer.CursorX = prevWord.StartChar
	}

	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
	return nil
}

// beginningOfLine moves cursor to the beginning of the current line
func (e *Editor) beginningOfLine(args []string) error {
	e.buffer.CursorX = 0
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
	return nil
}

// endOfLine moves cursor to the end of the current line
func (e *Editor) endOfLine(args []string) error {
	// Check if we have virtual cursors AND we're not in cursor mode
	// In cursor mode, only move the primary cursor
	if len(e.virtualCursors) > 0 && e.mode != commands.CursorMode {
		return e.executeMovementForAllCursors(func() error {
			line := e.buffer.CursorIterator.Line()
			lineLen := len(line)
			if lineLen > 0 {
				e.buffer.CursorX = lineLen - 1
			} else {
				e.buffer.CursorX = 0
			}
			return nil
		})
	}
	
	// Single cursor movement (or in cursor mode)
	line := e.buffer.CursorIterator.Line()
	lineLen := len(line)
	if lineLen > 0 {
		e.buffer.CursorX = lineLen - 1
	} else {
		e.buffer.CursorX = 0
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	e.updateCursorState()
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
    // Robust, iterator-independent logic under newline-as-boundary semantics
    // Determine target logical line index
    lineCount := e.buffer.LineCount
    if lineCount <= 0 {
        // Defensive: fall back to origin
        e.buffer.CursorIterator = buffer.NewBufferIterator(e.buffer)
        e.buffer.CursorX = 0
    } else {
        lastIndex := lineCount - 1
        targetLine := lastIndex
        // If buffer ends with a newline, the last line is empty; pick previous line when possible
        if e.buffer.Rope.Length() > 0 {
            if e.buffer.Rope.ByteAt(e.buffer.Rope.Length()-1) == '\n' && lastIndex > 0 {
                targetLine = lastIndex - 1
            }
        }

        // Move cursor to target line
        e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, targetLine)
        // Move to end of line (normal mode: last character)
        lastLine := e.buffer.CursorIterator.Line()
        e.buffer.CursorX = len(lastLine)
        if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
            e.buffer.CursorX--
        }
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

// findCharacter moves cursor to the next occurrence of the specified character on the current line
func (e *Editor) findCharacter(args []string) error {
	if len(args) == 0 {
		return nil
	}
	
	char := args[0]
	if len(char) == 0 {
		return nil
	}
	
	line := e.buffer.CursorIterator.Line()
	currentX := e.buffer.CursorX
	
	// Search for the character starting from the position after the cursor
	for i := currentX + 1; i < len(line); i++ {
		if string(line[i]) == char {
			e.buffer.CursorX = i
			e.desiredColumn = -1
			e.updateCursorState()
			return nil
		}
	}
	
	// Character not found
	return nil
}

// tillCharacter moves cursor to the character before the next occurrence of the specified character on the current line
func (e *Editor) tillCharacter(args []string) error {
	if len(args) == 0 {
		return nil
	}
	
	char := args[0]
	if len(char) == 0 {
		return nil
	}
	
	line := e.buffer.CursorIterator.Line()
	currentX := e.buffer.CursorX
	
	// Search for the character starting from the position after the cursor
	for i := currentX + 1; i < len(line); i++ {
		if string(line[i]) == char {
			// Move to the position before the character
			if i > 0 {
				e.buffer.CursorX = i - 1
				e.desiredColumn = -1
				e.updateCursorState()
			}
			return nil
		}
	}
	
	// Character not found
	return nil
}

// startFindCharacterOperation begins a find character operation, waiting for character input
func (e *Editor) startFindCharacterOperation(args []string) error {
	e.pendingKeys = []string{"f"}
	return nil
}

// startTillCharacterOperation begins a till character operation, waiting for character input
func (e *Editor) startTillCharacterOperation(args []string) error {
	e.pendingKeys = []string{"t"}
	return nil
}
