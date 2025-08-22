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
    if e.buffer == nil {
        return
    }
    if !e.buffer.HasFile() {
		return
	}

	// Skip special buffers - don't add them to navigation history
	if (strings.HasPrefix(e.buffer.FilePath, "*References:") && strings.HasSuffix(e.buffer.FilePath, "*")) ||
		e.buffer.FilePath == "*Diagnostics*" {
		return
	}

	entry := NavigationEntry{
		filePath:         e.buffer.FilePath,
		cursorX:          e.buffer.CursorX,
		cursorLineNumber: e.buffer.CursorIterator.LineNumber(),
		topLineNumber:    e.buffer.ViewIterator.LineNumber(),
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
	if entry.filePath != e.buffer.FilePath {
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
	e.buffer.CursorX = entry.cursorX

	// Restore cursor iterator to the saved line number
	e.buffer.CursorIterator = buffer.NewBufferIterator(e.buffer)
	for i := 0; i < entry.cursorLineNumber && !e.buffer.CursorIterator.IsEOF(); i++ {
		e.buffer.CursorIterator = e.buffer.CursorIterator.Next()
	}

	// Ensure cursor is within bounds
	if !e.buffer.CursorIterator.IsEOF() {
		line := e.buffer.CursorIterator.Line()
		if e.buffer.CursorX > len(line) {
			e.buffer.CursorX = len(line)
			if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
				e.buffer.CursorX-- // Don't go past last character in normal mode
			}
		}
	}

	// Position at the center of the screen for better visibility
	e.scrollToCenter()

	return nil
}
