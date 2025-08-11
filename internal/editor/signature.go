package editor


// getSignatureY returns the Y position where signature should be drawn
func (e *Editor) getSignatureY() int {
	y := e.getStatusBarY() - e.getSignatureLines()
	// Adjust for error line if present
	if e.errorManager.GetCurrentError() != "" {
		y -= 1
	}
	return y
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

	// Launch goroutine to fetch signature help
	go func(generation uint64) {
		sigHelp, err := e.proxyManager.GetSignatureHelp(e.buffer.FilePath, line, col)
		
		// Only apply response if generation matches current state
		if generation == e.buffer.SignatureGeneration {
			if err != nil {
				LL.Debug("LSP signature help error for %s: %v", e.buffer.FilePath, err)
				e.buffer.CurrentSignature = nil
			} else {
				// Parse and store the signature help
				e.parseSignatureHelpResponse(e.buffer, sigHelp)
			}
		} else {
			// Stale response - ignore it
			LL.Debug("Ignoring stale signature help response: gen=%d, current=%d",
				generation, e.buffer.SignatureGeneration)
		}
	}(e.buffer.SignatureGeneration)
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
    if e.inPasteMode {
        return
    }
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
