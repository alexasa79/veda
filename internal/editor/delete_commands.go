package editor

// startDeleteOperation begins a delete operation, waiting for a text object
func (e *Editor) startDeleteOperation(args []string) error {
	e.pendingOperator = DeleteOperator{}
	e.pendingModifier = ModifierNone
	e.pendingKeys = []string{"d"}
	return nil
}

// startChangeOperation begins a change operation, waiting for a text object
func (e *Editor) startChangeOperation(args []string) error {
	e.pendingOperator = ChangeOperator{}
	e.pendingModifier = ModifierNone
	e.pendingKeys = []string{"c"}
	return nil
}

// startYankOperation begins a yank operation, waiting for a text object
func (e *Editor) startYankOperation(args []string) error {
	e.pendingOperator = YankOperator{}
	e.pendingModifier = ModifierNone
	e.pendingKeys = []string{"y"}
	return nil
}

// handlePendingOperation processes keys when there's a pending operation
func (e *Editor) handlePendingOperation(key string) error {
	if e.pendingOperator == nil {
		return nil
	}
	
	// Add key to pending keys
	e.pendingKeys = append(e.pendingKeys, key)
	
	// Check for special cases first
	if len(e.pendingKeys) == 2 && e.pendingKeys[0] == "d" && e.pendingKeys[1] == "d" {
		// dd - delete line(s)
		if e.pendingCount > 1 {
			return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, MultiLineTextObject{Count: e.pendingCount})
		}
		return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, LineTextObject{})
	}
	
	if len(e.pendingKeys) == 2 && e.pendingKeys[0] == "c" && e.pendingKeys[1] == "c" {
		// cc - change line(s)
		if e.pendingCount > 1 {
			return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, MultiLineTextObject{Count: e.pendingCount})
		}
		return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, LineTextObject{})
	}
	
	if len(e.pendingKeys) == 2 && e.pendingKeys[0] == "y" && e.pendingKeys[1] == "y" {
		// yy - yank line(s)
		if e.pendingCount > 1 {
			return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, MultiLineTextObject{Count: e.pendingCount})
		}
		return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, LineTextObject{})
	}
	
	// Check for modifier keys
	if len(e.pendingKeys) == 2 && (key == "a" || key == "i") {
		e.pendingModifier = TextObjectModifier(key)
		return nil // Wait for text object
	}
	
	// Try to find a text object
	textObj := e.getTextObjectForKey(key)
	if textObj != nil {
		return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, textObj)
	}
	
	// Check for brackets after modifier
	if len(e.pendingKeys) == 3 && (e.pendingModifier == ModifierAround || e.pendingModifier == ModifierInside) {
		bracket := rune(key[0])
		if bracketObj := NewBracketTextObject(bracket, e.pendingModifier); bracketObj != nil {
			return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, bracketObj)
		}
	}
	
	// If we get here and have accumulated too many keys, cancel the operation
	if len(e.pendingKeys) > 3 {
		e.clearPendingOperation()
	}
	
	return nil
}

// getTextObjectForKey returns the appropriate text object for a key
func (e *Editor) getTextObjectForKey(key string) TextObject {
	switch key {
	case "w":
		return WordTextObject{modifier: e.pendingModifier}
	case "W":
		return WORDTextObject{}
	case "b":
		// 'b' for blocks through LSP folding ranges
		if e.pendingModifier != ModifierNone {
			return NewBlockTextObject(e.pendingModifier)
		}
	case "a":
		// 'a' for argument
		if e.pendingModifier != ModifierNone {
			return NewArgumentTextObject(e.pendingModifier, e.proxyManager)
		}
	case "f":
		// 'f' for function
		if e.pendingModifier != ModifierNone {
			return NewFunctionTextObject(e.pendingModifier, e.proxyManager)
		}
	}
	
	return nil
}

// executeOperatorWithTextObject executes the pending operator on the given text object
func (e *Editor) executeOperatorWithTextObject(op Operator, textObj TextObject) error {
	defer e.clearPendingOperation()
	
	if e.buffer == nil {
		return nil
	}
	
	// Store the operator+text object combination for repeat
	e.lastCommand = &RepeatableCommand{
		Operator:   op,
		TextObject: textObj,
	}
	
	// Get current cursor position
	cursorLine := e.buffer.CursorIterator.LineNumber()
	cursorChar := e.buffer.CursorX
	
	// Get the range from the text object
	startLine, startChar, endLine, endChar, found := textObj.GetRange(e.buffer, cursorLine, cursorChar)
	if !found {
		return nil
	}
	
	// Execute the operator on the range
	if err := op.Execute(e, startLine, startChar, endLine, endChar); err != nil {
		return err
	}
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	return nil
}

// clearPendingOperation resets the pending operation state
func (e *Editor) clearPendingOperation() {
	e.pendingOperator = nil
	e.pendingModifier = ModifierNone
	e.pendingKeys = nil
	e.pendingCount = 0
}

// cancelPendingOperation cancels any pending operation (called on Escape)
func (e *Editor) cancelPendingOperation() error {
	e.clearPendingOperation()
	return nil
}

// hasPendingOperation returns true if there's a pending operation
func (e *Editor) hasPendingOperation() bool {
	return e.pendingOperator != nil
}