package editor

import (
	"veda/internal/proxyworker"
)


// getSignatureY returns the Y position where signature should be drawn
func (e *Editor) getSignatureY() int {
	return e.getStatusBarY() - e.getSignatureLines()
}

// getSignatureLines returns the number of lines needed for signature display
func (e *Editor) getSignatureLines() int {
	if e.buffer.CurrentSignature == nil {
		return 0
	}
	// For now, we'll use 1 line for the signature
	// Later we can expand this if we need multi-line signatures
	return 1
}

// requestSignatureHelpAsync sends an asynchronous request for signature help
func (e *Editor) requestSignatureHelpAsync() {
	// Only works if we have a file open
	if !e.buffer.HasFile() {
		return
	}

	// Get cursor position
	line := e.buffer.CursorIterator.LineNumber()
	col := e.buffer.CursorX

	LL.Debug("Requesting async signature help for %s at %d:%d", e.buffer.FilePath, line, col)

	// Create LSP request with current generation
	request := proxyworker.ProxyRequest{
		Type:       proxyworker.RequestSignatureHelp,
		FilePath:   e.buffer.FilePath,
		Buffer:     e.buffer,
		Generation: e.buffer.SignatureGeneration,
		Line:       line,
		Character:  col,
	}

	// Send request to LSP worker (non-blocking)
	select {
	case e.lspRequestChan <- request:
		LL.Trace("Sent async signature help request for %s at %d:%d (gen=%d)",
			e.buffer.FilePath, line, col, e.buffer.SignatureGeneration)
	default:
		LL.Debug("LSP request channel full, dropping signature help request for %s", e.buffer.FilePath)
	}
}

// showSignature initiates signature help display
func (e *Editor) showSignature(args []string) error {
	// Increment generation for this new request
	e.buffer.SignatureGeneration++
	
	// Request signature help from LSP
	e.requestSignatureHelpAsync()
	
	return nil
}

// checkSignatureUpdate checks if we should re-request signature help based on cursor position
func (e *Editor) checkSignatureUpdate() {
	// Only check if we have an active signature
	if e.buffer.CurrentSignature == nil {
		return
	}
	
	// Check if current character is comma or closing paren
	if !e.buffer.CursorIterator.IsEOF() {
		line := e.buffer.CursorIterator.Line()
		if e.buffer.CursorX < len(line) {
			ch := line[e.buffer.CursorX]
			if ch == ',' || ch == ')' {
				// Increment generation and re-request
				e.buffer.SignatureGeneration++
				e.requestSignatureHelpAsync()
			}
		}
	}
}
