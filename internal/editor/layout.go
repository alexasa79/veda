package editor

import (
    "strings"
    "veda/internal/commands"
)

// Layout calculation methods
// These methods centralize the calculation of screen areas, making it easy to change
// the layout of the editor without modifying multiple places in the code

// getVisibleHeight returns the number of lines available for content display
func (e *Editor) getVisibleHeight() int {
    _, height := e.screen.Size()
    // For sub-window screens, the region already excludes the global status bar
    if _, ok := e.screen.(*windowScreen); ok {
        return height
    }
    // Terminal/internal buffers should not reserve extra lines for context/diagnostics/signature
    if e.mode == commands.TerminalMode || (e.buffer != nil && e.buffer.IsInternal) {
        // Reserve only the status bar line
        return height - 1
    }
    reserved := 1 // Status bar
    if e.shouldShowContext() {
        reserved += len(e.contextCache) // Context lines
    }
    // Reserve space for signature if shown
    if e.buffer.CurrentSignature != nil {
        reserved += e.getSignatureLines()
    }
    // Reserve space for error/diagnostic lines if shown
    if e.errorManager.GetCurrentError() != "" {
        reserved += 1
    } else {
        // Reserve space for diagnostic lines if we have a diagnostic for current line
        cursorLine := e.buffer.CursorIterator.LineNumber() + 1
        if diag, ok := e.buffer.GetDiagnosticForLine(cursorLine); ok {
            reserved += e.getDiagnosticLineCount(diag.Message)
        }
    }
    return height - reserved
}

// getVisibleWidth returns the number of columns available for content display
func (e *Editor) getVisibleWidth() int {
    width, _ := e.screen.Size()
    // Width is unaffected by status bar/overlays; subtract nothing
    return width
}

// getContentStartY returns the Y position where content starts
func (e *Editor) getContentStartY() int {
    // Window sub-screens always start at 0 within their region
    if _, ok := e.screen.(*windowScreen); ok {
        return 0
    }
    // Terminal/internal buffers do not reserve context lines
    if e.mode == commands.TerminalMode || (e.buffer != nil && e.buffer.IsInternal) {
        return 0
    }
    if e.shouldShowContext() {
        return len(e.contextCache)
    }
    return 0
}

// shouldShowContext determines if context should be shown
func (e *Editor) shouldShowContext() bool {
    if !e.showContext {
        return false
    }
    // Never show context for terminal/internal buffers
    if e.mode == commands.TerminalMode || (e.buffer != nil && e.buffer.IsInternal) {
        return false
    }
    return true
}

// updateContext updates the cached context for the current cursor position
func (e *Editor) updateContext() {
	// Get current cursor line
	cursorIter := e.buffer.CursorIterator
	currentLine := cursorIter.LineNumber()

	// Only update if line changed
	if currentLine == e.contextLine && len(e.contextCache) > 0 {
		return
	}

    // Suppress context for internal/terminal buffers
    if e.mode == commands.TerminalMode || (e.buffer != nil && e.buffer.IsInternal) {
        e.contextCache = nil
        e.contextLine = currentLine
        return
    }
    // Get file path
    filePath := e.buffer.FilePath
    if filePath == "" {
        e.contextCache = nil
        return
    }

    // Calculate visible range (per-window; entire region height)
    viewStartLine := e.buffer.ViewIterator.LineNumber()
    viewEndLine := viewStartLine + e.getVisibleHeight() - 1

	// Try to use tree-sitter for context if available
	if e.highlighter != nil && e.highlighter.GetTree() != nil {
		// Use tree-sitter based context detection
		e.contextCache = getTreeSitterContext(e.buffer, e.highlighter, currentLine, viewStartLine, viewEndLine, e.config.ScrollOffset)
	} else {
		// No tree-sitter available, no context
		e.contextCache = nil
		return
	}

	e.contextLine = currentLine
}

// getStatusBarY returns the Y position of the status bar
func (e *Editor) getStatusBarY() int {
	_, height := e.screen.Size()
	return height - 1
}

// getVisibleLines returns the number of lines that can be displayed in the editor
func (e *Editor) getVisibleLines() int {
	return e.getVisibleHeight()
}

// getAbsolutePosition returns the absolute position in the buffer for the current cursor
func (e *Editor) getAbsolutePosition() int {
	// Get the position at the start of the current line
	pos := e.buffer.CursorIterator.Position()

	// Add current X position
	// Need to get current line length to validate cursorX
	if !e.buffer.CursorIterator.IsEOF() {
		line := e.buffer.CursorIterator.Line()
		if e.buffer.CursorX > len(line) {
			pos += len(line)
		} else {
			pos += e.buffer.CursorX
		}
	}

	return pos
}

// updateSyntaxHighlighting re-parses the buffer for syntax highlighting
func (e *Editor) updateSyntaxHighlighting() {
	if e.highlighter != nil && e.buffer.HasFile() {
		content := e.buffer.Rope.String()
		// Force a full reparse by not reusing the old tree
		e.highlighter.ParseFull([]byte(content))
	}
}

// getDiagnosticLineCount returns the number of lines needed to display a diagnostic message
func (e *Editor) getDiagnosticLineCount(message string) int {
	// Remove carriage returns and split by newlines
	cleanMessage := strings.ReplaceAll(message, "\r", "")
	allLines := strings.Split(cleanMessage, "\n")
	
	// Filter out empty lines
	var lines []string
	for _, line := range allLines {
		cleanLine := strings.TrimSpace(line)
		if cleanLine != "" {
			lines = append(lines, cleanLine)
		}
	}
	
	// Limit to reasonable number of lines to avoid taking up too much screen space
	maxLines := 3
	if len(lines) > maxLines {
		return maxLines
	}
	return len(lines)
}

// handleResize processes screen resize events
func (e *Editor) handleResize() {
    width, _ := e.screen.Size()

	// Update the command line widget's dimensions and position
	e.commandLine.SetWidth(width)
	e.commandLine.SetPosition(0, e.getStatusBarY())

	// Also update search line widget
	e.searchLine.SetWidth(width)
	e.searchLine.SetPosition(0, e.getStatusBarY())

	// Clear and redraw the screen to handle the new dimensions
    // Recompute window layout if using windows
    if len(e.windows) > 0 {
        e.recomputeWindowsLayout()
    }
    // When a terminal buffer is active, propagate resize to PTY via proxy
    if e.mode == commands.TerminalMode && e.buffer != nil {
        if ts := e.terminalSessions[e.buffer.FilePath]; ts != nil && ts.SessionID != "" {
            cols := e.getVisibleWidth()
            rows := e.getVisibleHeight()
            _ = e.proxyManager.ResizeTerminal(ts.WorkingDir, ts.SessionID, cols, rows)
            ts.Cols, ts.Rows = cols, rows
            if ts.term != nil {
                ts.term.Resize(cols, rows)
            }
        }
    }
    e.screen.Clear()
	e.render()
	e.screen.Show()
}