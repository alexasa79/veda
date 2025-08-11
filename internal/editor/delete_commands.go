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
	// Handle character search operations (f/t) when no operator is pending
	if e.pendingOperator == nil && len(e.pendingKeys) > 0 {
		if len(e.pendingKeys) == 1 {
			if e.pendingKeys[0] == "f" {
				// Execute find character operation
				e.pendingKeys = nil
				return e.findCharacter([]string{key})
			} else if e.pendingKeys[0] == "t" {
				// Execute till character operation
				e.pendingKeys = nil
				return e.tillCharacter([]string{key})
			}
		}
	}
	
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
	
	// Check for 'f' or 't' motions - need to wait for character
	if len(e.pendingKeys) == 2 && (key == "f" || key == "t") {
		// Don't consume the key yet, we need one more character
		return nil // Wait for the character to find/till
	}
	
	// Check for 'f' or 't' with character (3 keys total: operator + f/t + character)
	if len(e.pendingKeys) == 3 && (e.pendingKeys[1] == "f" || e.pendingKeys[1] == "t") {
		character := key
		if e.pendingKeys[1] == "f" {
			return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, FindCharacterTextObject{character: character})
		} else if e.pendingKeys[1] == "t" {
			return e.executeOperatorWithTextObjectForAllCursors(e.pendingOperator, TillCharacterTextObject{character: character})
		}
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
	if len(e.pendingKeys) > 4 {
		e.clearPendingOperation()
	}
	
	return nil
}

// getTextObjectForKey returns the appropriate text object for a key
func (e *Editor) getTextObjectForKey(key string) TextObject {
	switch key {
	case "w":
		// For 'cw', exclude trailing space (like 'ce' in Vim)
		excludeTrailing := false
		if _, isChange := e.pendingOperator.(ChangeOperator); isChange && e.pendingModifier == ModifierNone {
			excludeTrailing = true
		}
		return WordTextObject{modifier: e.pendingModifier, excludeTrailingSpace: excludeTrailing}
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
	case "s":
		// 's' for search result
		return NewSearchResultTextObject(e)
	case "left":
		return CharacterTextObject{Direction: "left"}
	case "right":
		return CharacterTextObject{Direction: "right"}
	case "up":
		return LineMotionTextObject{Direction: "up"}
	case "down":
		return LineMotionTextObject{Direction: "down"}
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
	
    // Update syntax highlighting (skip during paste mode)
    if !e.inPasteMode {
        e.updateSyntaxHighlighting()
    }
	
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
	return e.pendingOperator != nil || len(e.pendingKeys) > 0
}