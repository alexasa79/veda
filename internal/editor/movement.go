package editor

import (
	"veda/internal/buffer"
	"veda/internal/commands"
)

func isWrapPadding(line WrappedLine) bool {
	return !line.IsFirstLine && line.StartChar == line.EndChar && line.Text == ""
}

// Movement commands
func (e *Editor) moveUp(args []string) error {
	t0 := e.perfTracker.Start()
	defer e.perfTracker.ReportSince("moveUp", t0)

	// Get current visual position
	// Get current visual position
	visualLine, visualCol := e.GetCursorVisualPosition()

	// Set desired column if not already set
	if e.desiredColumn == -1 {
		e.desiredColumn = visualCol
	}

	wrappedLines := e.CalculateWrappedLines()

	if visualLine > 0 {
		// Map to new position one visual line up, using desired column
		newVisualLine := visualLine - 1
		for newVisualLine >= 0 && isWrapPadding(wrappedLines[newVisualLine]) {
			newVisualLine--
		}
		if newVisualLine >= 0 {
			newLogicalLine, newCharPos := e.MapVisualToLogical(newVisualLine, e.desiredColumn)

			// Update cursor position
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), newLogicalLine)
			e.CurrentBuffer().CursorX = newCharPos

			// Ensure cursor is within bounds for normal mode
			if e.mode == commands.NormalMode && !e.CurrentBuffer().CursorIterator.IsEOF() {
				line := e.CurrentBuffer().CursorIterator.Line()
				if len(line) > 0 && e.CurrentBuffer().CursorX >= len(line) {
					e.CurrentBuffer().CursorX = len(line) - 1 // Place on last character
				}
			}
		}
	}

	// Debug printout: which buffer changed and new cursor position
	if e.CurrentBuffer() != nil {
		LL.Debug("moveUp: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveDown(args []string) error {
	t0 := e.perfTracker.Start()
	defer e.perfTracker.ReportSince("moveDown", t0)

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
		for newVisualLine < len(wrappedLines) && isWrapPadding(wrappedLines[newVisualLine]) {
			newVisualLine++
		}
		if newVisualLine < len(wrappedLines) {
			newLogicalLine, newCharPos := e.MapVisualToLogical(newVisualLine, e.desiredColumn)

			// Update cursor position
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), newLogicalLine)
			e.CurrentBuffer().CursorX = newCharPos

			// Ensure cursor is within bounds for normal mode
			if e.mode == commands.NormalMode && !e.CurrentBuffer().CursorIterator.IsEOF() {
				line := e.CurrentBuffer().CursorIterator.Line()
				if len(line) > 0 && e.CurrentBuffer().CursorX >= len(line) {
					e.CurrentBuffer().CursorX = len(line) - 1 // Place on last character
				}
			}
		}
	}

	if e.CurrentBuffer() != nil {
		LL.Debug("moveDown: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	} else {
		// At last visible wrapped row: advance to the next logical line to keep moving
		if !e.CurrentBuffer().CursorIterator.IsEOF() {
			e.CurrentBuffer().CursorIterator = e.CurrentBuffer().CursorIterator.Next()
			// Clamp X for normal mode to avoid going past last character
			if e.mode == commands.NormalMode {
				line := e.CurrentBuffer().CursorIterator.Line()
				if len(line) > 0 && e.CurrentBuffer().CursorX >= len(line) {
					e.CurrentBuffer().CursorX = len(line) - 1
				}
			}
		}
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveLeft(args []string) error {
	// Auto-indent cleanup not needed for horizontal movement on same line
	line := e.CurrentBuffer().CursorIterator.Line()
	if e.CurrentBuffer().CursorX > 0 {
		e.CurrentBuffer().CursorX = prevRuneStart(line, e.CurrentBuffer().CursorX)
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	if e.CurrentBuffer() != nil {
		LL.Debug("moveLeft: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) moveRight(args []string) error {
	line := e.CurrentBuffer().CursorIterator.Line()
	if len(line) > 0 {
		cursorX := clampCursorToRuneStart(line, e.CurrentBuffer().CursorX)
		maxX := lastRuneStart(line) // Don't go past the last character in normal mode
		if cursorX < maxX {
			cursorX = nextRuneStart(line, cursorX)
		}
		e.CurrentBuffer().CursorX = cursorX
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	if e.CurrentBuffer() != nil {
		LL.Debug("moveRight: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

// wordForward moves the cursor to the beginning of the next word
// Enhanced to stop at end of line when a word extends to the line boundary
func (e *Editor) wordForward(args []string) error {
	currentLine := e.CurrentBuffer().CursorIterator.LineNumber()
	startX := e.CurrentBuffer().CursorX
	line := e.CurrentBuffer().CursorIterator.Line()

	// Check if we should stop at end of line first
	if shouldStopAtEndOfLine(e.CurrentBuffer(), currentLine, startX) {
		// Move to end of current line
		e.CurrentBuffer().CursorX = len(line) - 1
		e.desiredColumn = -1
		e.updateCursorState()
		return nil
	}

	// Use normal word boundary logic
	nextWord := FindNextWordBoundary(e.CurrentBuffer(), currentLine, startX)

	if nextWord != nil {
		// Move cursor to the next word
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), nextWord.StartLine)
		e.CurrentBuffer().CursorX = nextWord.StartChar
	}

	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	if e.CurrentBuffer() != nil {
		LL.Debug("wordForward: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

// wordBackward moves the cursor to the beginning of the previous word
func (e *Editor) wordBackward(args []string) error {
	currentLine := e.CurrentBuffer().CursorIterator.LineNumber()
	prevWord := FindPreviousWordBoundary(e.CurrentBuffer(), currentLine, e.CurrentBuffer().CursorX)

	if prevWord != nil {
		// Move cursor to the previous word
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), prevWord.StartLine)
		e.CurrentBuffer().CursorX = prevWord.StartChar
	}

	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	if e.CurrentBuffer() != nil {
		LL.Debug("wordBackward: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

// wordForwardNoUnderscore moves the cursor to the beginning of the next word (W command)
// Treats underscores as word boundaries, not as part of words
func (e *Editor) wordForwardNoUnderscore(args []string) error {
	currentLine := e.CurrentBuffer().CursorIterator.LineNumber()
	startX := e.CurrentBuffer().CursorX
	line := e.CurrentBuffer().CursorIterator.Line()

	// Check if we should stop at end of line first (using underscore-aware logic)
	if shouldStopAtEndOfLineWithOptions(e.CurrentBuffer(), currentLine, startX, false) {
		// Move to end of current line
		e.CurrentBuffer().CursorX = len(line) - 1
		e.desiredColumn = -1
		e.updateCursorState()
		return nil
	}

	// Use word boundary logic with underscores as boundaries
	nextWord := FindNextWordBoundaryWithOptions(e.CurrentBuffer(), currentLine, startX, false)

	if nextWord != nil {
		// Move cursor to the next word
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), nextWord.StartLine)
		e.CurrentBuffer().CursorX = nextWord.StartChar
	}

	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	if e.CurrentBuffer() != nil {
		LL.Debug("wordForwardNoUnderscore: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

// wordBackwardNoUnderscore moves the cursor to the beginning of the previous word (B command)
// Treats underscores as word boundaries, not as part of words
func (e *Editor) wordBackwardNoUnderscore(args []string) error {
	currentLine := e.CurrentBuffer().CursorIterator.LineNumber()
	prevWord := FindPreviousWordBoundaryWithOptions(e.CurrentBuffer(), currentLine, e.CurrentBuffer().CursorX, false)

	if prevWord != nil {
		// Move cursor to the previous word
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), prevWord.StartLine)
		e.CurrentBuffer().CursorX = prevWord.StartChar
	}

	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	if e.CurrentBuffer() != nil {
		LL.Debug("wordBackwardNoUnderscore: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

// beginningOfLine moves cursor to the beginning of the current line
func (e *Editor) beginningOfLine(args []string) error {
	e.CurrentBuffer().CursorX = 0
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	if e.CurrentBuffer() != nil {
		LL.Debug("bol: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

// endOfLine moves cursor to the end of the current line
func (e *Editor) endOfLine(args []string) error {
	// Check if we have virtual cursors AND we're not in cursor mode
	// In cursor mode, only move the primary cursor
	if len(e.virtualCursors) > 0 && e.mode != commands.CursorMode {
		return e.executeMovementForAllCursors(func() error {
			line := e.CurrentBuffer().CursorIterator.Line()
			lineLen := len(line)
			if lineLen > 0 {
				e.CurrentBuffer().CursorX = lineLen - 1
			} else {
				e.CurrentBuffer().CursorX = 0
			}
			return nil
		})
	}

	// Single cursor movement (or in cursor mode)
	line := e.CurrentBuffer().CursorIterator.Line()
	lineLen := len(line)
	if lineLen > 0 {
		e.CurrentBuffer().CursorX = lineLen - 1
	} else {
		e.CurrentBuffer().CursorX = 0
	}
	// Reset desired column when moving horizontally
	e.desiredColumn = -1
	if e.CurrentBuffer() != nil {
		LL.Debug("eol: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) pageUp(args []string) error {
	t0 := e.perfTracker.Start()
	defer e.perfTracker.ReportSince("pageUp", t0)
	visibleLines := e.getVisibleLines()

	// Move cursor up by visible lines
	for i := 0; i < visibleLines && e.CurrentBuffer().CursorIterator.LineNumber() > 0; i++ {
		e.CurrentBuffer().CursorIterator = e.CurrentBuffer().CursorIterator.Previous()
	}

	// Adjust cursor X if needed
	line := e.CurrentBuffer().CursorIterator.Line()
	if e.CurrentBuffer().CursorX > len(line) {
		e.CurrentBuffer().CursorX = len(line)
		if e.CurrentBuffer().CursorX > 0 {
			e.CurrentBuffer().CursorX--
		}
	}
	if e.CurrentBuffer() != nil {
		LL.Debug("pageUp: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

func (e *Editor) pageDown(args []string) error {
	t0 := e.perfTracker.Start()
	defer e.perfTracker.ReportSince("pageDown", t0)
	visibleLines := e.getVisibleLines()

	// Move cursor down by visible lines
	maxY := e.CurrentBuffer().LineCount - 1
	for i := 0; i < visibleLines && e.CurrentBuffer().CursorIterator.LineNumber() < maxY; i++ {
		e.CurrentBuffer().CursorIterator = e.CurrentBuffer().CursorIterator.Next()
	}

	// Adjust cursor X if needed
	line := e.CurrentBuffer().CursorIterator.Line()
	if e.CurrentBuffer().CursorX > len(line) {
		e.CurrentBuffer().CursorX = len(line)
		if e.CurrentBuffer().CursorX > 0 {
			e.CurrentBuffer().CursorX--
		}
	}
	if e.CurrentBuffer() != nil {
		LL.Debug("pageDown: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

// goToBeginning moves the cursor to the beginning of the file
func (e *Editor) goToBeginning(args []string) error {
	e.CurrentBuffer().CursorX = 0
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIterator(e.CurrentBuffer())
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(e.CurrentBuffer())
	if e.CurrentBuffer() != nil {
		LL.Debug("goToBeginning: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()
	return nil
}

// goToEnd moves the cursor to the end of the file
func (e *Editor) goToEnd(args []string) error {
	// Robust, iterator-independent logic under newline-as-boundary semantics
	// Determine target logical line index
	lineCount := e.CurrentBuffer().LineCount
	if lineCount <= 0 {
		// Defensive: fall back to origin
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(e.CurrentBuffer())
		e.CurrentBuffer().CursorX = 0
	} else {
		lastIndex := lineCount - 1
		targetLine := lastIndex
		// If buffer ends with a newline, the last line is empty; pick previous line when possible
		if e.CurrentBuffer().Rope.Length() > 0 {
			if e.CurrentBuffer().Rope.ByteAt(e.CurrentBuffer().Rope.Length()-1) == '\n' && lastIndex > 0 {
				targetLine = lastIndex - 1
			}
		}

		// Move cursor to target line
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), targetLine)
		// Move to end of line (normal mode: last character)
		lastLine := e.CurrentBuffer().CursorIterator.Line()
		e.CurrentBuffer().CursorX = len(lastLine)
		if e.CurrentBuffer().CursorX > 0 && e.mode == commands.NormalMode {
			e.CurrentBuffer().CursorX--
		}
	}

	// Scroll to show the cursor
	e.ScrollToShowAtBottom()
	if e.CurrentBuffer() != nil {
		LL.Debug("goToEnd: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
	e.updateCursorState()

	return nil
}

// goToLine moves the cursor to a specific line number (0-based)
func (e *Editor) goToLine(lineNum int) {
	// Clamp line number to valid range
	if lineNum < 0 {
		lineNum = 0
	} else if lineNum >= e.CurrentBuffer().LineCount {
		lineNum = e.CurrentBuffer().LineCount - 1
	}

	// Move cursor to beginning of the target line
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().CursorX = 0

	// Update the view to ensure the cursor is visible
	e.scrollToCenter()
	if e.CurrentBuffer() != nil {
		LL.Debug("goToLine: buffer=%s -> line=%d, cursorX=%d", e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
	}
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

	line := e.CurrentBuffer().CursorIterator.Line()
	currentX := e.CurrentBuffer().CursorX

	// Search for the character starting from the position after the cursor
	for i := currentX + 1; i < len(line); i++ {
		if string(line[i]) == char {
			e.CurrentBuffer().CursorX = i
			e.desiredColumn = -1
			if e.CurrentBuffer() != nil {
				LL.Debug("findCharacter '%s': buffer=%s -> line=%d, cursorX=%d", char, e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
			}
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

	line := e.CurrentBuffer().CursorIterator.Line()
	currentX := e.CurrentBuffer().CursorX

	// Search for the character starting from the position after the cursor
	for i := currentX + 1; i < len(line); i++ {
		if string(line[i]) == char {
			// Move to the position before the character
			if i > 0 {
				e.CurrentBuffer().CursorX = i - 1
				e.desiredColumn = -1
				if e.CurrentBuffer() != nil {
					LL.Debug("tillCharacter '%s': buffer=%s -> line=%d, cursorX=%d", char, e.CurrentBuffer().DisplayName(), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)
				}
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
