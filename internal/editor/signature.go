package editor

import (
	"encoding/json"
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"veda/internal/buffer"
)

// updateArgumentListState checks if the cursor is inside a function's argument list
func (e *Editor) updateArgumentListState() {
	if e.highlighter == nil || e.highlighter.GetTree() == nil {
		e.buffer.InArgumentList = false
		return
	}

	// Store previous state to detect transitions
	wasInArgumentList := e.buffer.InArgumentList

	// Get the current cursor position in the buffer
	cursorPos := e.getAbsolutePosition()
	
	// Find the node at cursor position
	root := e.highlighter.GetTree().RootNode()
	node := e.findNodeAtPosition(root, cursorPos)
	
	if node == nil {
		e.buffer.InArgumentList = false
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
		return
	}
	
	// Get the positions of the opening and closing parentheses
	// First child is always '(', last child is always ')'
	openParen := argList.Child(0)
	closeParen := argList.Child(int(argList.ChildCount() - 1))
	
	if openParen == nil || closeParen == nil {
		e.buffer.InArgumentList = false
		return
	}
	
	// Check if cursor is between the parentheses (inclusive of opening paren)
	openPos := int(openParen.StartByte())
	closePos := int(closeParen.StartByte())
	
	// In normal mode, being ON the opening paren or anywhere up to (but not including) the closing paren
	// counts as being inside the argument list
	if cursorPos >= openPos && cursorPos < closePos {
		e.buffer.InArgumentList = true
	} else {
		e.buffer.InArgumentList = false
	}
	
	// Detect transition from false to true
	if !wasInArgumentList && e.buffer.InArgumentList {
		LL.Debug("Transition detected: entered argument list")
		// Query LSP for signature help only if we don't already have one
		if e.buffer.CurrentSignature == nil {
			e.querySignatureHelp()
		}
	}
	
	// Clear signature when leaving argument list
	if wasInArgumentList && !e.buffer.InArgumentList {
		LL.Debug("Transition detected: left argument list")
		e.buffer.CurrentSignature = nil
	}
}

// querySignatureHelp queries the LSP for signature help at the current cursor position
func (e *Editor) querySignatureHelp() {
	// Only works if we have a file open
	if !e.buffer.HasFile() {
		return
	}
	
	// Get cursor position
	line := e.buffer.CursorIterator.LineNumber()
	col := e.buffer.CursorX
	
	LL.Debug("Querying signature help for %s at %d:%d", e.buffer.FilePath, line, col)
	
	// Query the LSP
	result, err := e.proxyClient.GetSignatureHelp(e.buffer.FilePath, line, col)
	if err != nil {
		LL.Debug("Failed to get signature help: %v", err)
		return
	}
	
	// Log the result for debugging
	LL.Debug("Signature help result: %+v", result)
	
	// Parse the signature help result
	if result == nil {
		LL.Debug("No signature help available")
		return
	}
	
	// Log the type of result for debugging
	LL.Debug("Signature help result type: %T", result)
	
	// Marshal to JSON for better debugging
	if jsonBytes, err := json.Marshal(result); err == nil {
		LL.Debug("Signature help JSON: %s", string(jsonBytes))
	}
	
	// Convert the result to SignatureHelp structure
	if sigHelp, ok := result.(map[string]interface{}); ok {
		if signatures, ok := sigHelp["signatures"].([]interface{}); ok && len(signatures) > 0 {
			// Take the first signature
			if sig, ok := signatures[0].(map[string]interface{}); ok {
				help := &buffer.SignatureHelp{}
				
				// Extract label
				if label, ok := sig["label"].(string); ok {
					help.Label = label
				}
				
				// Extract documentation
				if doc, ok := sig["documentation"].(string); ok {
					help.Documentation = doc
				}
				
				// Extract active parameter
				if activeParam, ok := sigHelp["activeParameter"].(float64); ok {
					help.ActiveParameter = int(activeParam)
				}
				
				// Extract parameters
				if params, ok := sig["parameters"].([]interface{}); ok {
					for _, p := range params {
						if param, ok := p.(map[string]interface{}); ok {
							paramInfo := buffer.ParameterInfo{}
							if label, ok := param["label"].(string); ok {
								paramInfo.Label = label
							}
							if doc, ok := param["documentation"].(string); ok {
								paramInfo.Documentation = doc
							}
							help.Parameters = append(help.Parameters, paramInfo)
						}
					}
				}
				
				// Store the signature
				e.buffer.CurrentSignature = help
				LL.Info("Stored signature: %s", help.Label)
			}
		}
	}
}

// drawSignature renders the function signature above the status bar
func (e *Editor) drawSignature() {
	if e.buffer.CurrentSignature == nil {
		return
	}

	width, _ := e.screen.Size()
	signatureY := e.getSignatureY()
	signatureStyle := e.config.Theme.GetStyle(e.config.Theme.Context) // Use context style for signature
	
	// Fill signature background
	for i := 0; i < width; i++ {
		e.screen.SetContent(i, signatureY, ' ', nil, signatureStyle)
	}
	
	// Format the signature for display
	sig := e.buffer.CurrentSignature
	displayText := sig.Label
	
	// Highlight the active parameter if available
	if len(sig.Parameters) > 0 && sig.ActiveParameter >= 0 && sig.ActiveParameter < len(sig.Parameters) {
		// This is a simple display - we could enhance it later to highlight the active parameter
		displayText = fmt.Sprintf("%s  [%d/%d]", sig.Label, sig.ActiveParameter+1, len(sig.Parameters))
	}
	
	// Log for debugging
	LL.Debug("Drawing signature at Y=%d: %s", signatureY, displayText)
	
	// Draw the signature text (truncate if too long)
	maxLen := width - 2
	if len(displayText) > maxLen {
		displayText = displayText[:maxLen-3] + "..."
	}
	e.drawText(1, signatureY, signatureStyle, displayText)
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