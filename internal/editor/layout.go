package editor

import (
	"strings"
	"veda/internal/commands"
	"veda/internal/syntax"
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
	// Terminal/internal buffers or empty windows should not reserve extra lines for context/diagnostics/signature
	if e.mode == commands.TerminalMode || e.CurrentBuffer() == nil || (e.CurrentBuffer() != nil && e.CurrentBuffer().IsInternal) {
		// Reserve only the status bar line
		return height - 1
	}
	reserved := 1 // Status bar
	if e.shouldShowContext() {
		if e.CurrentBuffer() != nil {
			reserved += len(e.CurrentBuffer().ContextCache) // Context lines
		}
	}
	// Reserve space for signature if shown
	if e.CurrentBuffer().CurrentSignature != nil {
		reserved += e.getSignatureLines()
	}
	// Reserve space for banners/diagnostics:
	// - Error banners reserve a line (critical). Info banners do NOT reserve space (overlay-only).
	// - If no banner reserves space, diagnostics may reserve lines.
	if msg, sev := e.errorManager.GetCurrentBanner(); msg != "" {
		if sev != "info" {
			reserved += 1
		}
		// If an info banner is active, we still draw it as an overlay without changing layout;
		// diagnostics are suppressed by drawErrorLine() when any banner is present.
	} else {
		// Reserve space for diagnostic lines if we have a diagnostic for current line
		cursorLine := e.CurrentBuffer().CursorIterator.LineNumber() + 1
		if diag, ok := e.CurrentBuffer().GetDiagnosticForLine(cursorLine); ok {
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
	if e.mode == commands.TerminalMode || (e.CurrentBuffer() != nil && e.CurrentBuffer().IsInternal) {
		return 0
	}
	if e.shouldShowContext() {
		if e.CurrentBuffer() != nil {
			return len(e.CurrentBuffer().ContextCache)
		}
		return 0
	}
	return 0
}

// getActiveWindowSizeForTerminal returns the width and height for the active window's
// content region, suitable for sizing a PTY-backed terminal.
// When no windows are present, it falls back to the full screen width and screen height minus status bar.
func (e *Editor) getActiveWindowSizeForTerminal() (int, int) {
	// Determine base width/height from active window or full-screen fallback
	var w, h int
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
		win := e.windows[e.activeWindow]
		w, h = win.Width, win.Height
	} else {
		// Fallback: single-window mode using the whole screen content area
		w, h = e.screen.Size()
		// Reserve status bar line in fallback
		if h > 0 {
			h = h - 1
		}
		if h < 1 {
			h = 1
		}
	}

	// Subtract gutter (line numbers + indicators) from width so PTY columns match
	// the visible content area where terminal characters are rendered.
	if e.CurrentBuffer() != nil {
		gutter := e.CurrentBuffer().LineNumberWidth
		if gutter >= 0 && gutter < w {
			w = w - gutter
		}
	}
	if w < 1 {
		w = 1
	}
	return w, h
}

// shouldShowContext determines if context should be shown
func (e *Editor) shouldShowContext() bool {
	if !e.showContext {
		return false
	}
	// Never show context for terminal/internal buffers
	if e.mode == commands.TerminalMode || (e.CurrentBuffer() != nil && e.CurrentBuffer().IsInternal) {
		return false
	}
	return true
}

// updateContext updates the cached context for the current cursor position
func (e *Editor) updateContext() {
	// Get current cursor line
	cursorIter := e.CurrentBuffer().CursorIterator
	currentLine := cursorIter.LineNumber()

	// Only update if line changed
	if e.CurrentBuffer() != nil && currentLine == e.CurrentBuffer().ContextLine && len(e.CurrentBuffer().ContextCache) > 0 {
		return
	}

	// Suppress context for internal/terminal buffers
	if e.mode == commands.TerminalMode || (e.CurrentBuffer() != nil && e.CurrentBuffer().IsInternal) {
		if e.CurrentBuffer() != nil {
			e.CurrentBuffer().ContextCache = nil
			e.CurrentBuffer().ContextLine = currentLine
		}
		return
	}
	// Get file path
	filePath := e.CurrentBuffer().FilePath
	if filePath == "" {
		if e.CurrentBuffer() != nil {
			e.CurrentBuffer().ContextCache = nil
		}
		return
	}

	// Calculate visible range (per-window; entire region height)
	viewStartLine := e.CurrentBuffer().ViewIterator.LineNumber()
	viewEndLine := viewStartLine + e.getVisibleHeight() - 1

	// Try to use tree-sitter for context if available (per-buffer highlighter)
	if e.CurrentBuffer() != nil && e.CurrentBuffer().Highlighter != nil && e.CurrentBuffer().Highlighter.GetTree() != nil {
		// Use tree-sitter based context detection
		e.CurrentBuffer().ContextCache = getTreeSitterContext(e.CurrentBuffer(), e.CurrentBuffer().Highlighter, currentLine, viewStartLine, viewEndLine, e.config.ScrollOffset)
	} else {
		// No tree-sitter available, no context
		if e.CurrentBuffer() != nil {
			e.CurrentBuffer().ContextCache = nil
		}
		return
	}

	if e.CurrentBuffer() != nil {
		e.CurrentBuffer().ContextLine = currentLine
	}
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
	pos := e.CurrentBuffer().CursorIterator.Position()

	// Add current X position
	// Need to get current line length to validate cursorX
	if !e.CurrentBuffer().CursorIterator.IsEOF() {
		line := e.CurrentBuffer().CursorIterator.Line()
		if e.CurrentBuffer().CursorX > len(line) {
			pos += len(line)
		} else {
			pos += e.CurrentBuffer().CursorX
		}
	}

	return pos
}

// updateSyntaxHighlighting re-parses the buffer for syntax highlighting
func (e *Editor) updateSyntaxHighlighting() {
	if e.CurrentBuffer() == nil || !e.CurrentBuffer().HasFile() {
		return
	}
	// Update the buffer's own highlighter in place
	if e.CurrentBuffer().Highlighter == nil {
		e.CurrentBuffer().Highlighter = syntax.NewHighlighter()
		_ = e.CurrentBuffer().Highlighter.SetLanguageFromFile(e.CurrentBuffer().FilePath)
	}
	content := e.CurrentBuffer().Rope.String()
	_ = e.CurrentBuffer().Highlighter.ReparseIncremental([]byte(content))
}

// updateSyntaxApplyEdit applies an incremental syntax update using the buffer's highlighter.
// startByte must be computed against the pre-edit content. oldText/newText are the exact bytes
// that were replaced/inserted at that position.
func (e *Editor) updateSyntaxApplyEdit(startByte int, oldText, newText string) {
	if e.CurrentBuffer() == nil || !e.CurrentBuffer().HasFile() {
		return
	}
	if e.CurrentBuffer().Highlighter == nil {
		e.CurrentBuffer().Highlighter = syntax.NewHighlighter()
		_ = e.CurrentBuffer().Highlighter.SetLanguageFromFile(e.CurrentBuffer().FilePath)
		// Initialize the tree from full content; subsequent calls can be incremental
		_ = e.CurrentBuffer().Highlighter.ParseFull([]byte(e.CurrentBuffer().Rope.String()))
		return
	}
	_ = e.CurrentBuffer().Highlighter.ApplyEdit(startByte, oldText, newText)
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
	// When a terminal buffer is active, propagate resize to PTY via proxy using the active window size
	if e.CurrentBuffer() != nil {
		if ts := e.terminalSessions[e.CurrentBuffer().FilePath]; ts != nil && ts.SessionID != "" {
			cols, rows := e.getActiveWindowSizeForTerminal()
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
