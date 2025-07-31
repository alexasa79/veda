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

	// Find all matches
	e.updateSearchMatches()

	// Move to the first match
	if len(e.searchMatches) > 0 {
		e.currentMatch = 0
		e.moveCursorToPosition(e.searchMatches[0])
	}
}

// findNext finds the next occurrence of the search pattern
func (e *Editor) findNext() {
	if e.searchPattern == "" || len(e.searchMatches) == 0 {
		return
	}

	// Get current position
	currentPos := e.getAbsolutePosition()

	// Find the next match position
	nextIndex := -1
	for i, matchPos := range e.searchMatches {
		if matchPos > currentPos {
			nextIndex = i
			break
		}
	}

	// If no match found after cursor, wrap to beginning
	if nextIndex == -1 {
		nextIndex = 0
	}

	// Update current match and move cursor
	e.currentMatch = nextIndex
	e.moveCursorToPosition(e.searchMatches[nextIndex])
}

// findPrevious finds the previous occurrence of the search pattern
func (e *Editor) findPrevious() {
	if e.searchPattern == "" || len(e.searchMatches) == 0 {
		return
	}

	// Get current position
	currentPos := e.getAbsolutePosition()

	// Find the previous match position
	prevIndex := -1
	for i := len(e.searchMatches) - 1; i >= 0; i-- {
		if e.searchMatches[i] < currentPos {
			prevIndex = i
			break
		}
	}

	// If no match found before cursor, wrap to end
	if prevIndex == -1 {
		prevIndex = len(e.searchMatches) - 1
	}

	// Update current match and move cursor
	e.currentMatch = prevIndex
	e.moveCursorToPosition(e.searchMatches[prevIndex])
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

// updateSearchMatches finds all occurrences of the search pattern
func (e *Editor) updateSearchMatches() {
	e.searchMatches = []int{}
	e.currentMatch = -1

	if e.searchPattern == "" {
		return
	}

	// Get buffer content
	content := e.buffer.Rope.String()

	// Find all matches
	pos := 0
	for {
		index := strings.Index(content[pos:], e.searchPattern)
		if index == -1 {
			break
		}
		e.searchMatches = append(e.searchMatches, pos+index)
		pos = pos + index + 1
	}
}

// clearSearch clears the current search pattern and matches
func (e *Editor) clearSearch() {
	e.searchPattern = ""
	e.searchMatches = []int{}
	e.currentMatch = -1
}

// getCurrentMatchIndex returns the index in searchMatches that corresponds to the given position
func (e *Editor) getCurrentMatchIndex(pos int) int {
	for i, matchPos := range e.searchMatches {
		if matchPos == pos {
			return i
		}
	}
	return -1
}

// clearSearchCmd is the command handler for clearing the search
func (e *Editor) clearSearchCmd(args []string) error {
	e.clearSearch()
	return nil
}