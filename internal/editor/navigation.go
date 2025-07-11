package editor

import (
	"veda/internal/commands"
)

// NavigationEntry represents a position in the navigation history
type NavigationEntry struct {
	filePath string
	cursorX  int
	cursorY  int
	topLine  int
}

// pushNavigationHistory saves the current position to navigation history
func (e *Editor) pushNavigationHistory() {
	if e.currentFile == "" {
		return
	}
	
	entry := NavigationEntry{
		filePath: e.currentFile,
		cursorX:  e.cursorX,
		cursorY:  e.cursorY,
		topLine:  e.topLine,
	}
	
	// Add to history
	e.navHistory = append(e.navHistory, entry)
	
	// Trim history if it exceeds max size
	if len(e.navHistory) > e.maxNavHistory {
		e.navHistory = e.navHistory[len(e.navHistory)-e.maxNavHistory:]
	}
}

// goBack navigates to the previous position in history
func (e *Editor) goBack() error {
	if len(e.navHistory) == 0 {
		return nil // No history to go back to
	}
	
	// Pop the last entry
	lastIndex := len(e.navHistory) - 1
	entry := e.navHistory[lastIndex]
	e.navHistory = e.navHistory[:lastIndex]
	
	// Save current position before jumping back
	// (but don't add to history to avoid circular navigation)
	e.saveCurrentBufferState()
	
	// Check if we need to switch files
	if entry.filePath != e.currentFile {
		// Check if the file is already open
		if bufferInfo, exists := e.buffers[entry.filePath]; exists {
			e.switchToBuffer(bufferInfo)
		} else {
			// File is not open, open it
			e.openFile(entry.filePath)
		}
	}
	
	// Always restore cursor position from navigation entry
	// This overrides any position from switchToBuffer
	e.cursorX = entry.cursorX
	e.cursorY = entry.cursorY
	// Don't restore topLine here - let scrollToShowAtTop handle it
	
	// Ensure cursor is within bounds
	lines := e.buffer.Lines()
	if e.cursorY >= len(lines) {
		e.cursorY = len(lines) - 1
	}
	if e.cursorY >= 0 && e.cursorY < len(lines) {
		line := lines[e.cursorY]
		if e.cursorX > len(line) {
			e.cursorX = len(line)
			if e.cursorX > 0 && e.mode == commands.NormalMode {
				e.cursorX-- // Don't go past last character in normal mode
			}
		}
	}
	
	// Position at ScrollOffset from the top for better visibility
	e.scrollToShowAtTop()
	
	return nil
}