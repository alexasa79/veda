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
	
	// For change and delete operations, we need special handling to preserve cursors
	isChangeOp := false
	isDeleteOp := false
	if _, isChange := op.(ChangeOperator); isChange {
		isChangeOp = true
	}
	if _, isDelete := op.(DeleteOperator); isDelete {
		isDeleteOp = true
	}
	
    if isChangeOp || isDeleteOp {
        // Group all per-cursor history entries into a single undo unit
        if e.buffer != nil && e.buffer.History != nil {
            if isChangeOp {
                e.buffer.History.StartComposite("Change with multiple cursors")
            } else {
                e.buffer.History.StartComposite("Delete with multiple cursors")
            }
            // Snapshot cursor positions prior to modifications so undo can restore them
            primary := buffer.CursorPos{Line: e.buffer.CursorIterator.LineNumber(), Char: e.buffer.CursorX}
            v := make([]buffer.CursorPos, 0, len(e.virtualCursors))
            for _, vc := range e.virtualCursors {
                v = append(v, buffer.CursorPos{Line: vc.Y, Char: vc.X})
            }
            e.buffer.History.SetActiveCompositeCursors(primary, v)
        }
		// For change operations, we need to carefully track cursor positions
		// because deletions can affect the positions of subsequent cursors
		
		// First, collect all the deletion ranges
		type deleteRange struct {
			startLine, startChar, endLine, endChar int
			originalCursorIndex int
		}
		var deleteRanges []deleteRange
		
		for i, cursor := range cursors {
			// Get the range from the text object at this cursor position
			startLine, startChar, endLine, endChar, found := textObj.GetRange(e.buffer, cursor.y, cursor.x)
			if !found {
				continue
			}
			
			deleteRanges = append(deleteRanges, deleteRange{
				startLine: startLine,
				startChar: startChar,
				endLine: endLine,
				endChar: endChar,
				originalCursorIndex: i,
			})
		}
		
		// Execute all deletions and track final cursor positions
		deleteOp := DeleteOperator{}
		newVirtualCursors := make([]VirtualCursor, 0)
		
		// Keep track of cumulative deletions to adjust cursor positions
		cumulativeDeletedChars := 0
		
		for i, dr := range deleteRanges {
			// Execute the deletion
			if err := deleteOp.Execute(e, dr.startLine, dr.startChar, dr.endLine, dr.endChar); err != nil {
				return err
			}
			
			// Calculate how many characters were deleted
			deletedChars := dr.endChar - dr.startChar
			
			// For virtual cursors (not the primary cursor), track their final position
			if dr.originalCursorIndex < len(cursors)-1 {
				// The cursor should be positioned at the start of where the deletion occurred
				// But we need to account for previous deletions that happened later in the text
				finalX := dr.startChar
				
				// Adjust for deletions that happened to the right of this cursor
				// (which we processed first due to reverse order)
				for j := 0; j < i; j++ {
					otherDr := deleteRanges[j]
					if otherDr.startLine == dr.startLine && otherDr.startChar > dr.startChar {
						finalX -= (otherDr.endChar - otherDr.startChar)
					}
				}
				
				newVirtualCursors = append(newVirtualCursors, VirtualCursor{
					X: finalX,
					Y: dr.startLine,
				})
			}
			
			cumulativeDeletedChars += deletedChars
		}
		
        // Update virtual cursors to their new positions
		e.virtualCursors = newVirtualCursors
		
		// Only enter insert mode for change operations, not delete operations
        if isChangeOp {
			// Now enter insert mode once
			e.switchToInsertMode()
			
			// Start tracking insert mode changes
			if e.buffer != nil && e.buffer.History != nil {
				e.buffer.History.StartInsertSession("c")
			}
		}
        // Do NOT end the composite yet for change operations. The subsequent
        // insert session should be part of the same composite so a single 'u'
        // undoes both the deletions and the inserted text. We'll end the
        // composite here for delete, but keep it open for change until the
        // insert session ends elsewhere.
        if e.buffer != nil && e.buffer.History != nil {
            if !isChangeOp { // pure delete
                e.buffer.History.EndComposite()
            }
        }
	} else {
		// For non-change operations, execute normally
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