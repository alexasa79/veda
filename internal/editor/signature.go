package editor

import (
	"veda/internal/commands"
)

// getSignatureY returns the Y position where signature should be drawn
func (e *Editor) getSignatureY() int {
	// Signature should sit directly above the status bar when present.
	return e.getStatusBarY() - e.getSignatureLines()
}

// getErrorOverlayLines returns how many lines are currently occupied by
// banners/diagnostics above the status bar.
func (e *Editor) getErrorOverlayLines() int {
	// Transient info banner has highest priority and uses one line
	if e.transientInfoActive && e.transientInfoMessage != "" {
		return 1
	}
	// Current banner (info or error) uses one line while visible
	if msg, _ := e.errorManager.GetCurrentBanner(); msg != "" {
		return 1
	}
	// Otherwise, diagnostics for the current line may take multiple lines
	if e.CurrentBuffer() != nil {
		cursorLine := e.CurrentBuffer().CursorIterator.LineNumber() + 1
		if diag, ok := e.CurrentBuffer().GetDiagnosticForLine(cursorLine); ok {
			return e.getDiagnosticLineCount(diag.Message)
		}
	}
	return 0
}

// getSignatureLines returns the number of lines needed for signature display
func (e *Editor) getSignatureLines() int {
	if e.CurrentBuffer().CurrentSignature == nil {
		return 0
	}
	// For now, we'll use 1 line for the signature
	// Later we can expand this if we need multi-line signatures
	return 1
}

// requestSignatureHelpAsync sends an asynchronous request for signature help
func (e *Editor) requestSignatureHelpAsync() {
	// Only works if we have a file open
	if !e.CurrentBuffer().HasFile() {
		return
	}

	// Get cursor position
	line := e.CurrentBuffer().CursorIterator.LineNumber()
	col := e.CurrentBuffer().CursorX

	LL.Debug("Requesting async signature help for %s at %d:%d", e.CurrentBuffer().FilePath, line, col)

	// Launch goroutine to fetch signature help
	go func(generation uint64) {
		sigHelp, err := e.proxyManager.GetSignatureHelp(e.CurrentBuffer().FilePath, line, col)

		// Only apply response if generation matches current state
		if generation == e.CurrentBuffer().SignatureGeneration {
			if err != nil {
				LL.Debug("LSP signature help error for %s: %v", e.CurrentBuffer().FilePath, err)
				e.CurrentBuffer().CurrentSignature = nil
			} else {
				// Parse and store the signature help
				e.parseSignatureHelpResponse(e.CurrentBuffer(), sigHelp)
			}
		} else {
			// Stale response - ignore it
			LL.Debug("Ignoring stale signature help response: gen=%d, current=%d",
				generation, e.CurrentBuffer().SignatureGeneration)
		}
	}(e.CurrentBuffer().SignatureGeneration)
}

// showSignature initiates signature help display
func (e *Editor) showSignature(args []string) error {
	// Increment generation for this new request
	e.CurrentBuffer().SignatureGeneration++

	// Request signature help from LSP
	e.requestSignatureHelpAsync()

	return nil
}

// checkSignatureUpdate checks if we should re-request signature help based on cursor position
func (e *Editor) checkSignatureUpdate() {
	if e.inPasteMode {
		return
	}

	// In insert mode, check if we should show signature help automatically
	if e.mode == commands.InsertMode {
		if e.isLikelyInsideCallOnCurrentLine() {
			// We're in insert mode and between parentheses
			if e.CurrentBuffer().CurrentSignature == nil {
				// We don't have signature help but should - request it
				e.CurrentBuffer().SignatureGeneration++
				e.requestSignatureHelpAsync()
				return
			}
			// We already have signature help - check if we should update it
			if !e.CurrentBuffer().CursorIterator.IsEOF() {
				line := e.CurrentBuffer().CursorIterator.Line()
				if e.CurrentBuffer().CursorX < len(line) {
					ch := line[e.CurrentBuffer().CursorX]
					if ch == ',' || ch == ')' {
						// Increment generation and re-request
						e.CurrentBuffer().SignatureGeneration++
						e.requestSignatureHelpAsync()
					}
				}
			}
		} else {
			// We're in insert mode but not between parentheses - hide signature
			if e.CurrentBuffer().CurrentSignature != nil {
				e.CurrentBuffer().CurrentSignature = nil
			}
		}
		return
	}

	// In non-insert modes, only check if we have an active signature
	if e.CurrentBuffer().CurrentSignature == nil {
		return
	}

	// If we've moved away from a likely function call context, clear the signature.
	// Heuristic: on the current line, there should be an unmatched '(' to the left
	// of the cursor (accounting for immediate ')' balancing on the same line).
	if !e.isLikelyInsideCallOnCurrentLine() {
		e.CurrentBuffer().CurrentSignature = nil
		return
	}

	// Check if current character is comma or closing paren
	if !e.CurrentBuffer().CursorIterator.IsEOF() {
		line := e.CurrentBuffer().CursorIterator.Line()
		if e.CurrentBuffer().CursorX < len(line) {
			ch := line[e.CurrentBuffer().CursorX]
			if ch == ',' || ch == ')' {
				// Increment generation and re-request
				e.CurrentBuffer().SignatureGeneration++
				e.requestSignatureHelpAsync()
			}
		}
	}
}

// isLikelyInsideCallOnCurrentLine checks for an unmatched '(' to the left of the
// cursor on the current line, accounting for same-line paren balancing.
// This is a lightweight heuristic to decide whether signature help should remain visible.
func (e *Editor) isLikelyInsideCallOnCurrentLine() bool {
	if e.CurrentBuffer() == nil || e.CurrentBuffer().CursorIterator.IsEOF() {
		return false
	}

	line := e.CurrentBuffer().CursorIterator.Line()
	if len(line) == 0 {
		return false
	}

	// Clamp cursor position within the line bounds (treat positions beyond end as end)
	cursorX := e.CurrentBuffer().CursorX
	if cursorX < 0 {
		cursorX = 0
	}
	if cursorX >= len(line) {
		cursorX = len(line)
	}

	// Walk left from cursor, tracking paren depth on this line.
	// depth counts unmatched ')' we've seen while scanning left.
	depth := 0
	for i := cursorX - 1; i >= 0; i-- {
		switch line[i] {
		case ')':
			depth++
		case '(':
			if depth > 0 {
				depth--
			} else {
				// Found an unmatched '('
				return true
			}
		}
	}
	return false
}
