package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
)

// NavigationEntry represents a position in the navigation history
type NavigationEntry struct {
	filePath              string // Preferred buffer path to return to
	fallbackFilePath      string // Real-file fallback if preferred buffer is closed
	cursorX               int
	cursorLineNumber      int // Preferred-buffer line number
	topLineNumber         int // Preferred-buffer top visible line number
	fallbackCursorLine    int // Fallback real-file line number
	fallbackTopLineNumber int // Fallback real-file top visible line number
}

func normalizeDiffPath(path string) string {
	if strings.HasSuffix(path, "@HEAD") {
		return strings.TrimSuffix(path, "@HEAD")
	}
	if strings.HasSuffix(path, "@current") {
		return strings.TrimSuffix(path, "@current")
	}
	return path
}

func (e *Editor) isBufferForFile(buf *buffer.Buffer, filePath string) bool {
	if buf == nil || filePath == "" {
		return false
	}
	if buf.FilePath == filePath {
		return true
	}
	if buf.RealPath != "" && buf.RealPath == filePath {
		return true
	}
	return normalizeDiffPath(buf.FilePath) == filePath
}

func (e *Editor) findOpenBufferForFile(filePath string) *buffer.Buffer {
	if filePath == "" {
		return nil
	}
	normalized := normalizeDiffPath(filePath)
	var diffFallback *buffer.Buffer

	// Prefer deterministic order and prefer non-diff buffers first.
	for _, path := range e.bufferOrder {
		buf, ok := e.buffers[path]
		if !ok || buf == nil {
			continue
		}
		if !e.isBufferForFile(buf, normalized) {
			continue
		}
		if !e.isDiffBuffer(buf) {
			return buf
		}
		if diffFallback == nil {
			diffFallback = buf
		}
	}
	if diffFallback != nil {
		return diffFallback
	}

	// Fallback if bufferOrder doesn't contain all open buffers.
	for _, buf := range e.buffers {
		if buf != nil && e.isBufferForFile(buf, normalized) {
			return buf
		}
	}
	return nil
}

// navigateToBufferTarget switches to the preferred buffer if open, otherwise
// falls back to an open buffer for the same file or opens that file.
// Returns true if the preferred buffer was selected.
func (e *Editor) navigateToBufferTarget(preferredBufferPath, fallbackFilePath string) bool {
	if preferredBufferPath != "" {
		if buf, exists := e.buffers[preferredBufferPath]; exists && buf != nil {
			e.switchToBuffer(buf)
			return true
		}
	}

	targetFile := fallbackFilePath
	if targetFile == "" {
		targetFile = normalizeDiffPath(preferredBufferPath)
	}
	if targetFile == "" {
		return false
	}

	if buf := e.findOpenBufferForFile(targetFile); buf != nil {
		e.switchToBuffer(buf)
		return false
	}

	e.openFile(targetFile)
	return false
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

	preferredPath := e.CurrentBuffer().FilePath
	fallbackPath := e.CurrentBuffer().RealPath
	if fallbackPath == "" {
		fallbackPath = normalizeDiffPath(preferredPath)
	}

	preferredCursorLine := e.CurrentBuffer().CursorIterator.LineNumber()
	preferredViewLine := e.CurrentBuffer().ViewIterator.LineNumber()
	fallbackCursorLine := preferredCursorLine
	fallbackViewLine := preferredViewLine
	if e.isDiffBuffer(e.CurrentBuffer()) {
		fallbackPath = normalizeDiffPath(fallbackPath)
		if diffLines := e.CurrentBuffer().DiffHints; len(diffLines) > 0 {
			if fallbackCursorLine >= 0 && fallbackCursorLine < len(diffLines) && diffLines[fallbackCursorLine].RightLine >= 0 {
				fallbackCursorLine = diffLines[fallbackCursorLine].RightLine
			}
			if fallbackViewLine >= 0 && fallbackViewLine < len(diffLines) && diffLines[fallbackViewLine].RightLine >= 0 {
				fallbackViewLine = diffLines[fallbackViewLine].RightLine
			}
		}
	}

	entry := NavigationEntry{
		filePath:              preferredPath,
		fallbackFilePath:      fallbackPath,
		cursorX:               e.CurrentBuffer().CursorX,
		cursorLineNumber:      preferredCursorLine,
		topLineNumber:         preferredViewLine,
		fallbackCursorLine:    fallbackCursorLine,
		fallbackTopLineNumber: fallbackViewLine,
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

	usedPreferredBuffer := e.navigateToBufferTarget(entry.filePath, entry.fallbackFilePath)
	if e.CurrentBuffer() == nil {
		return nil
	}

	// Always restore cursor position from navigation entry
	// This overrides any position from switchToBuffer
	e.CurrentBuffer().CursorX = entry.cursorX

	// Restore cursor iterator to the saved line number
	targetLine := entry.fallbackCursorLine
	if usedPreferredBuffer {
		targetLine = entry.cursorLineNumber
	}
	if targetLine < 0 {
		targetLine = 0
	}
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), targetLine)

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
