package editor

// Layout calculation methods
// These methods centralize the calculation of screen areas, making it easy to change
// the layout of the editor without modifying multiple places in the code

// getVisibleHeight returns the number of lines available for content display
func (e *Editor) getVisibleHeight() int {
	_, height := e.screen.Size()
	reserved := 1 // Status bar
	reserved += len(e.contextCache) // Context lines
	// Reserve space for signature if shown
	if e.buffer.CurrentSignature != nil {
		reserved += e.getSignatureLines()
	}
	return height - reserved
}

// getContentStartY returns the Y position where content starts
func (e *Editor) getContentStartY() int {
	return len(e.contextCache) // Start after context lines
}

// shouldShowContext determines if context should be shown
func (e *Editor) shouldShowContext() bool {
	if !e.showContext {
		return false
	}

	// Always show context if we have it
	// The scroll offset will handle limiting the number of lines
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

	// Get file path
	filePath := e.buffer.FilePath
	if filePath == "" {
		e.contextCache = nil
		return
	}

	// Calculate visible range
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

// handleResize processes screen resize events
func (e *Editor) handleResize() {
	width, _ := e.screen.Size()

	// Update the command line widget's dimensions and position
	e.commandLine.SetWidth(width)
	e.commandLine.SetPosition(0, e.getStatusBarY())

	// Also update search line widget
	e.searchLine.SetWidth(width)
	e.searchLine.SetPosition(0, e.getStatusBarY())
}