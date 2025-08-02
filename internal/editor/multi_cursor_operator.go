package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/clipboard"
)

// executeOperatorWithTextObjectForAllCursors executes an operator+textobject for all cursors
func (e *Editor) executeOperatorWithTextObjectForAllCursors(op Operator, textObj TextObject) error {
	// Special handling for yank operator
	if _, isYank := op.(YankOperator); isYank && len(e.virtualCursors) > 0 {
		return e.executeYankForAllCursors(textObj)
	}
	
	// For other operators, execute them individually at each cursor
	// (this maintains the existing behavior for delete, change, etc.)
	return e.executeOperatorWithTextObject(op, textObj)
}

// executeYankForAllCursors performs yank operation on all cursors
func (e *Editor) executeYankForAllCursors(textObj TextObject) error {
	if e.buffer == nil {
		return nil
	}
	
	// Save positions
	primaryLineNum := e.buffer.CursorIterator.LineNumber()
	primaryCursorX := e.buffer.CursorX
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	
	// Collect yanked text from all cursors
	var yankedTexts []string
	
	// First yank from virtual cursors
	for _, cursor := range e.virtualCursors {
		// Set cursor to virtual cursor position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, cursor.Y)
		e.buffer.CursorX = cursor.X
		
		// Get the range from the text object
		startLine, startChar, endLine, endChar, found := textObj.GetRange(e.buffer, cursor.Y, cursor.X)
		if !found {
			continue
		}
		
		// Get the text to yank
		var text string
		if startLine == endLine {
			// Single line yank
			iter := buffer.NewBufferIteratorAtLine(e.buffer, startLine)
			if !iter.IsEOF() {
				line := iter.Line()
				if startChar < len(line) && endChar <= len(line) {
					text = line[startChar:endChar]
				}
			}
		} else {
			// Multi-line yank
			text = e.getTextInRange(startLine, startChar, endLine, endChar)
		}
		
		if text != "" {
			// Remove trailing newline for consistency
			text = strings.TrimSuffix(text, "\n")
			yankedTexts = append(yankedTexts, text)
		}
	}
	
	// Restore primary cursor and yank from it
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primaryLineNum)
	e.buffer.CursorX = primaryCursorX
	
	// Get the range from the text object
	startLine, startChar, endLine, endChar, found := textObj.GetRange(e.buffer, primaryLineNum, primaryCursorX)
	if found {
		// Get the text to yank
		var text string
		if startLine == endLine {
			// Single line yank
			iter := buffer.NewBufferIteratorAtLine(e.buffer, startLine)
			if !iter.IsEOF() {
				line := iter.Line()
				if startChar < len(line) && endChar <= len(line) {
					text = line[startChar:endChar]
				}
			}
		} else {
			// Multi-line yank
			text = e.getTextInRange(startLine, startChar, endLine, endChar)
		}
		
		if text != "" {
			// Remove trailing newline for consistency
			text = strings.TrimSuffix(text, "\n")
			yankedTexts = append(yankedTexts, text)
		}
		
		// Flash highlight at primary cursor
		e.setFlashHighlight(startLine, startChar, endLine, endChar)
	}
	
	// Join all yanked texts with newlines
	combinedText := strings.Join(yankedTexts, "\n")
	
	// Copy to clipboard
	if err := clipboard.Copy(combinedText); err != nil {
		LL.Error("Failed to copy to clipboard: %v", err)
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	
	// Restore view iterator
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	
	LL.Debug("Yanked %d items to clipboard (%d characters total)", len(yankedTexts), len(combinedText))
	
	// Store the operator+text object combination for repeat
	e.lastCommand = &RepeatableCommand{
		Operator:   YankOperator{},
		TextObject: textObj,
	}
	
	return nil
}