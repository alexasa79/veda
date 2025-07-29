package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/events"
)

// handleSearchMode handles key events in search mode
func (e *Editor) handleSearchMode(ev *events.KeyEvent) {
	switch ev.Key {
	case events.KeyEscape:
		// Exit search mode without searching
		e.exitSearchMode()
	case events.KeyEnter:
		// Execute search and stay in normal mode
		e.executeSearch()
		e.exitSearchMode()
	default:
		// Pass other keys to the search line widget
		e.searchLine.HandleKey(ev)

		// Check if the user deleted all text including the slash
		// If so, exit search mode automatically
		if e.searchLine.GetText() == "" {
			e.exitSearchMode()
		}
	}
}

// enterSearchMode switches to search mode and initializes the search line
func (e *Editor) enterSearchMode() {
	e.mode = commands.SearchMode
	e.searchLine.SetText("/")
	// Position cursor after the slash
	e.searchLine.SetCursor(1)
	e.searchDirection = true // Forward search by default
}

// exitSearchMode switches back to normal mode
func (e *Editor) exitSearchMode() {
	e.mode = commands.NormalMode
	e.searchLine.SetText("")
}

// executeSearch performs the search operation
func (e *Editor) executeSearch() {
	fullSearch := e.searchLine.GetText()

	// Remove the leading "/"
	pattern := fullSearch
	if len(pattern) > 1 {
		pattern = pattern[1:]
	} else {
		// Empty search, do nothing
		return
	}

	// Store the search pattern
	e.searchPattern = pattern

	// Perform the search
	e.findNext()
}

// findNext finds the next occurrence of the search pattern
func (e *Editor) findNext() {
	if e.searchPattern == "" {
		return
	}

	// Get current position
	startPos := e.getAbsolutePosition()

	// Convert buffer content to string for searching
	content := e.buffer.Rope.String()

	// Search forward from current position + 1
	searchStart := startPos + 1
	if searchStart >= len(content) {
		searchStart = 0 // Wrap around
	}

	// Find the pattern
	foundPos := strings.Index(content[searchStart:], e.searchPattern)

	if foundPos == -1 && searchStart > 0 {
		// Not found after cursor, try from beginning
		foundPos = strings.Index(content[:startPos], e.searchPattern)
		if foundPos >= 0 {
			// Found before cursor (wrapped around)
			e.moveCursorToPosition(foundPos)
			return
		}
	} else if foundPos >= 0 {
		// Found after cursor
		e.moveCursorToPosition(searchStart + foundPos)
		return
	}

	// Pattern not found
	LL.Debug("Pattern not found: %s", e.searchPattern)
}

// findPrevious finds the previous occurrence of the search pattern
func (e *Editor) findPrevious() {
	if e.searchPattern == "" {
		return
	}

	// Get current position
	startPos := e.getAbsolutePosition()

	// Convert buffer content to string for searching
	content := e.buffer.Rope.String()

	// Search backward from current position
	searchEnd := startPos
	if searchEnd <= 0 {
		searchEnd = len(content) // Wrap around
	}

	// Find the last occurrence before current position
	foundPos := -1
	searchContent := content[:searchEnd]

	// Find all occurrences and take the last one
	lastPos := -1
	searchStart := 0
	for {
		pos := strings.Index(searchContent[searchStart:], e.searchPattern)
		if pos == -1 {
			break
		}
		lastPos = searchStart + pos
		searchStart = lastPos + 1
	}

	if lastPos != -1 {
		foundPos = lastPos
	} else if searchEnd < len(content) {
		// Not found before cursor, try after cursor (wrap around)
		afterContent := content[startPos:]
		// Find the last occurrence after cursor
		lastPos = -1
		searchStart = 0
		for {
			pos := strings.Index(afterContent[searchStart:], e.searchPattern)
			if pos == -1 {
				break
			}
			lastPos = searchStart + pos
			searchStart = lastPos + 1
		}
		if lastPos != -1 {
			foundPos = startPos + lastPos
		}
	}

	if foundPos != -1 {
		e.moveCursorToPosition(foundPos)
	} else {
		// Pattern not found
		LL.Debug("Pattern not found: %s", e.searchPattern)
	}
}

// findNextCmd is the command handler for the 'n' key
func (e *Editor) findNextCmd(args []string) error {
	e.findNext()
	return nil
}

// findPreviousCmd is the command handler for the 'N' key
func (e *Editor) findPreviousCmd(args []string) error {
	e.findPrevious()
	return nil
}

// moveCursorToPosition moves the cursor to an absolute position in the buffer
func (e *Editor) moveCursorToPosition(pos int) {
	// Remember the current view line before moving cursor (if iterator is still valid)
	oldViewLine := 0
	if e.buffer.ViewIterator != nil && e.buffer.ViewIterator.Generation == e.buffer.Generation {
		oldViewLine = e.buffer.ViewIterator.LineNumber()
	}

	// Create an iterator at the beginning
	iter := buffer.NewBufferIterator(e.buffer)
	lineNum := 0
	lineStart := 0

	// Find the line containing the position
	for !iter.IsEOF() {
		lineEnd := iter.LineEnd()
		if pos <= lineEnd {
			// Position is on this line
			break
		}
		lineStart = lineEnd + 1 // +1 for newline
		iter = iter.Next()
		lineNum++
	}

	// Set cursor position
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.CursorX = pos - lineStart

	// Ensure cursor X is within bounds
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(line) {
		e.buffer.CursorX = len(line)
		if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
			e.buffer.CursorX--
		}
	}

	// Check if the cursor is already visible
	visibleHeight := e.getVisibleHeight()
	
	// Ensure ViewIterator is valid
	if e.buffer.ViewIterator == nil || e.buffer.ViewIterator.Generation != e.buffer.Generation {
		e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, oldViewLine)
	}
	
	viewLine := e.buffer.ViewIterator.LineNumber()
	cursorLine := e.buffer.CursorIterator.LineNumber()

	// Check if cursor is within visible range
	if cursorLine >= viewLine && cursorLine < viewLine + visibleHeight {
		// Cursor is already visible, no need to scroll
		return
	}

	// Cursor is not visible, center it on screen
	e.scrollToCenter()

	// Check if the view actually changed
	newViewLine := e.buffer.ViewIterator.LineNumber()
	if newViewLine == oldViewLine {
		// View didn't change, use default scrolling instead
		e.scrollToShowAtTop()
	}
}

// enterSearchModeCmd is the command handler for entering search mode
func (e *Editor) enterSearchModeCmd(args []string) error {
	e.enterSearchMode()
	return nil
}