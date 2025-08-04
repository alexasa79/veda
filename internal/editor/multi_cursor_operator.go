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
	
	// If no virtual cursors, just execute on primary cursor
	if len(e.virtualCursors) == 0 {
		return e.executeOperatorWithTextObject(op, textObj)
	}
	
	// For other operators with multiple cursors, we need to execute on all cursors
	// Process from last to first to avoid position shifts
	cursors := make([]struct{ x, y int }, 0, len(e.virtualCursors)+1)
	
	// Add virtual cursors
	for _, vc := range e.virtualCursors {
		cursors = append(cursors, struct{ x, y int }{vc.X, vc.Y})
	}
	
	// Add primary cursor
	primaryLine := e.buffer.CursorIterator.LineNumber()
	primaryX := e.buffer.CursorX
	cursors = append(cursors, struct{ x, y int }{primaryX, primaryLine})
	
	// Sort cursors by position (line, then char) in reverse order
	// This ensures we process from end to beginning to avoid position shifts
	for i := 0; i < len(cursors)-1; i++ {
		for j := 0; j < len(cursors)-i-1; j++ {
			if cursors[j].y < cursors[j+1].y || 
			   (cursors[j].y == cursors[j+1].y && cursors[j].x < cursors[j+1].x) {
				cursors[j], cursors[j+1] = cursors[j+1], cursors[j]
			}
		}
	}
	
	// Store the view position
	viewLine := e.buffer.ViewIterator.LineNumber()
	
	// Execute operator at each cursor position
	for _, cursor := range cursors {
		// Get the range from the text object at this cursor position
		startLine, startChar, endLine, endChar, found := textObj.GetRange(e.buffer, cursor.y, cursor.x)
		if !found {
			continue
		}
		
		// Execute the operator
		if err := op.Execute(e, startLine, startChar, endLine, endChar); err != nil {
			return err
		}
	}
	
	// For change operator, clear virtual cursors since we're entering insert mode
	if _, isChange := op.(ChangeOperator); isChange {
		e.virtualCursors = nil
	}
	
	// Restore view position
	if viewLine >= e.buffer.LineCount {
		viewLine = e.buffer.LineCount - 1
	}
	if viewLine < 0 {
		viewLine = 0
	}
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLine)
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	// Store for repeat
	e.lastCommand = &RepeatableCommand{
		Operator:   op,
		TextObject: textObj,
	}
	
	// Clear pending operation
	e.clearPendingOperation()
	
	return nil
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