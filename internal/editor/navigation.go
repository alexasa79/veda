package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
)

// NavigationEntry represents a position in the navigation history
type NavigationEntry struct {
	filePath         string
	cursorX          int
	cursorLineNumber int // Line number of the cursor position
	topLineNumber    int // Line number of the top visible line
}

// pushNavigationHistory saves the current position to navigation history
func (e *Editor) pushNavigationHistory() {
	// Guard against cases where no buffer is active yet
	if e.CurrentBuffer() == nil {
		return
	}
	if !e.CurrentBuffer().HasFile() {
		return
	}

	// Skip special buffers - don't add them to navigation history
	if (strings.HasPrefix(e.CurrentBuffer().FilePath, "*References:") && strings.HasSuffix(e.CurrentBuffer().FilePath, "*")) ||
		e.CurrentBuffer().FilePath == "*Diagnostics*" {
		return
	}

	entry := NavigationEntry{
		filePath:         e.CurrentBuffer().FilePath,
		cursorX:          e.CurrentBuffer().CursorX,
		cursorLineNumber: e.CurrentBuffer().CursorIterator.LineNumber(),
		topLineNumber:    e.CurrentBuffer().ViewIterator.LineNumber(),
	}

	// Add to history
	e.navHistory = append(e.navHistory, entry)

	// Trim history if it exceeds max size
	if len(e.navHistory) > e.maxNavHistory {
		e.navHistory = e.navHistory[len(e.navHistory)-e.maxNavHistory:]
	}
}

// goBack navigates to the previous position in history
func (e *Editor) goBack(args []string) error {
	if len(e.navHistory) == 0 {
		return nil
	}

	// Pop the last entry
	lastIndex := len(e.navHistory) - 1
	entry := e.navHistory[lastIndex]
	e.navHistory = e.navHistory[:lastIndex]

	// Save current position before jumping back
	// (but don't add to history to avoid circular navigation)
	e.saveCurrentBufferState()

	// Check if we need to switch files
	if entry.filePath != e.CurrentBuffer().FilePath {
		// Check if the file is already open
		if buf, exists := e.buffers[entry.filePath]; exists {
			e.switchToBuffer(buf)
		} else {
			// File is not open, open it
			e.openFile(entry.filePath)
		}
	}

	// Always restore cursor position from navigation entry
	// This overrides any position from switchToBuffer
	e.CurrentBuffer().CursorX = entry.cursorX

	// Restore cursor iterator to the saved line number
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(e.CurrentBuffer())
	for i := 0; i < entry.cursorLineNumber && !e.CurrentBuffer().CursorIterator.IsEOF(); i++ {
		e.CurrentBuffer().CursorIterator = e.CurrentBuffer().CursorIterator.Next()
	}

	// Ensure cursor is within bounds
	if !e.CurrentBuffer().CursorIterator.IsEOF() {
		line := e.CurrentBuffer().CursorIterator.Line()
		if e.CurrentBuffer().CursorX > len(line) {
			e.CurrentBuffer().CursorX = len(line)
			if e.CurrentBuffer().CursorX > 0 && e.mode == commands.NormalMode {
				e.CurrentBuffer().CursorX-- // Don't go past last character in normal mode
			}
		}
	}

	// Position at the center of the screen for better visibility
	e.scrollToCenter()

	return nil
}
