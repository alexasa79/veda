package editor

import (
	"veda/internal/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

// updateArgumentListState checks if the cursor is inside a function's argument list
func (e *Editor) updateArgumentListState() {
	if e.highlighter == nil || e.highlighter.GetTree() == nil {
		e.buffer.InArgumentList = false
		e.buffer.CurrentSignature = nil
		return
	}

	// Get the current cursor position in the buffer
	cursorPos := e.getAbsolutePosition()

	// Find the node at cursor position
	root := e.highlighter.GetTree().RootNode()
	node := e.findNodeAtPosition(root, cursorPos)

	if node == nil {
		e.buffer.InArgumentList = false
		e.buffer.CurrentSignature = nil
		return
	}

	// Walk up the tree to find a call_expression node
	current := node
	var callExpr *sitter.Node
	for current != nil {
		if current.Type() == "call_expression" {
			callExpr = current
			break
		}
		current = current.Parent()
	}

	if callExpr == nil {
		e.buffer.InArgumentList = false
		e.buffer.CurrentSignature = nil
		return
	}

	// Find the argument_list child of the call expression
	var argList *sitter.Node
	for i := 0; i < int(callExpr.ChildCount()); i++ {
		child := callExpr.Child(i)
		if child.Type() == "argument_list" {
			argList = child
			break
		}
	}

	if argList == nil || argList.ChildCount() < 2 {
		e.buffer.InArgumentList = false
		e.buffer.CurrentSignature = nil
		return
	}

	// Get the positions of the opening and closing parentheses
	// First child is always '(', last child is always ')'
	openParen := argList.Child(0)
	closeParen := argList.Child(int(argList.ChildCount() - 1))

	if openParen == nil || closeParen == nil {
		e.buffer.InArgumentList = false
		e.buffer.CurrentSignature = nil
		return
	}

	// Check if cursor is between the parentheses (inclusive of both)
	openPos := int(openParen.StartByte())
	closePos := int(closeParen.StartByte())

	// In normal mode, being ON the opening paren or anywhere up to and including the closing paren
	// counts as being inside the argument list
	if cursorPos >= openPos && cursorPos <= closePos {
		if !e.buffer.InArgumentList {
			// Entering argument list for the first time
			e.buffer.InArgumentList = true
			LL.Debug("Transition detected: entered argument list")
			// Send async request for signature help
			e.requestSignatureHelpAsync()
		}
		// If already in argument list, don't send duplicate requests
	} else {
		if e.buffer.InArgumentList {
			// Leaving argument list
			LL.Debug("Transition detected: left argument list")
			e.buffer.InArgumentList = false
			e.buffer.CurrentSignature = nil
			// Increment generation to invalidate any pending responses
			e.buffer.SignatureGeneration++
		}
	}
}


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
	request := lsp.LSPRequest{
		Type:       lsp.RequestSignatureHelp,
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
