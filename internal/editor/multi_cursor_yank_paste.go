package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/clipboard"
)

// yankForAllCursors performs yank operation for all cursors (primary + virtual)
// Returns the yanked text as a single string with lines separated by newlines
func (e *Editor) yankForAllCursors(yankFunc func() (string, error)) error {
	if len(e.virtualCursors) == 0 {
		// No virtual cursors, just do normal yank
		text, err := yankFunc()
		if err != nil {
			return err
		}
		return clipboard.Copy(text)
	}
	
	// Save primary cursor position
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
		
		// Yank at this position
		text, err := yankFunc()
		if err != nil {
			LL.Debug("Failed to yank at virtual cursor: %v", err)
			continue
		}
		
		// Remove trailing newline for consistency
		text = strings.TrimSuffix(text, "\n")
		yankedTexts = append(yankedTexts, text)
	}
	
	// Restore primary cursor and yank from it
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primaryLineNum)
	e.buffer.CursorX = primaryCursorX
	
	text, err := yankFunc()
	if err != nil {
		return err
	}
	
	// Remove trailing newline for consistency
	text = strings.TrimSuffix(text, "\n")
	yankedTexts = append(yankedTexts, text)
	
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
	
	return nil
}

// yankLineForCursor yanks the current line
// This is a helper function for yankForAllCursors
func (e *Editor) yankLineForCursor() (string, error) {
	line := e.buffer.CursorIterator.Line()
	return line + "\n", nil
}

// pasteForAllCursors handles paste operation for multiple cursors
func (e *Editor) pasteForAllCursors(pasteFunc func(string) error, _ bool) error {
	// Get clipboard content
	content, err := clipboard.Paste()
	if err != nil {
		LL.Error("Failed to paste from clipboard: %v", err)
		return nil
	}
	
	if content == "" {
		return nil
	}
	
	// Check if we have virtual cursors
	if len(e.virtualCursors) == 0 {
		// No virtual cursors, just do normal paste
		return pasteFunc(content)
	}
	
	// Split content into lines
	lines := strings.Split(content, "\n")
	
	// Determine if this is multi-line content
	// Multi-line content has more than one line, regardless of trailing newline
	isMultiLine := len(lines) > 1
	
	// Save positions
	primaryLineNum := e.buffer.CursorIterator.LineNumber()
	primaryCursorX := e.buffer.CursorX
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	
	// If single line (or character paste), paste the same content at each cursor
	if !isMultiLine {
		// First paste at virtual cursors (in reverse order to maintain positions)
		for i := len(e.virtualCursors) - 1; i >= 0; i-- {
			cursor := e.virtualCursors[i]
			
			// Set cursor to virtual cursor position
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, cursor.Y)
			e.buffer.CursorX = cursor.X
			
			// Paste at this position
			if err := pasteFunc(content); err != nil {
				LL.Debug("Failed to paste at virtual cursor: %v", err)
			}
			
			// Update virtual cursor position after paste
			e.virtualCursors[i].Y = e.buffer.CursorIterator.LineNumber()
			e.virtualCursors[i].X = e.buffer.CursorX
		}
		
		// Restore and paste at primary cursor
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primaryLineNum)
		e.buffer.CursorX = primaryCursorX
		
		// Execute paste
		if err := pasteFunc(content); err != nil {
			return err
		}
		
		// Restore view iterator
		e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
		
		return nil
	}
	
	// Multi-line paste: distribute lines among cursors
	// We need to collect all cursor positions first, then paste in document order
	type cursorInfo struct {
		x, y      int
		isVirtual bool
		index     int // for virtual cursors
	}
	
	var cursors []cursorInfo
	
	// Add virtual cursors
	for i, vc := range e.virtualCursors {
		cursors = append(cursors, cursorInfo{
			x:         vc.X,
			y:         vc.Y,
			isVirtual: true,
			index:     i,
		})
	}
	
	// Add primary cursor
	cursors = append(cursors, cursorInfo{
		x:         primaryCursorX,
		y:         primaryLineNum,
		isVirtual: false,
		index:     -1,
	})
	// Sort cursors by line number (y), then by column (x)
	// This ensures we paste in document order
	for i := 0; i < len(cursors); i++ {
		for j := i + 1; j < len(cursors); j++ {
			if cursors[i].y > cursors[j].y || 
			   (cursors[i].y == cursors[j].y && cursors[i].x > cursors[j].x) {
				cursors[i], cursors[j] = cursors[j], cursors[i]
			}
		}
	}
	
	// Determine how many lines each cursor should get
	// If we have more lines than cursors, the last cursor gets all remaining lines
	linesToPaste := make(map[int][]string) // map from cursor index in sorted order to lines
	for i := 0; i < len(cursors); i++ {
		if i < len(lines) {
			if i == len(cursors)-1 && len(lines) > len(cursors) {
				// Last cursor gets all remaining lines
				linesToPaste[i] = lines[i:]
			} else {
				// Each cursor gets one line
				linesToPaste[i] = []string{lines[i]}
			}
		}
	}
	
	// Now paste lines in reverse order to maintain positions
	for i := len(cursors) - 1; i >= 0; i-- {
		cursor := cursors[i]
		
		// Find which sorted position this cursor is at
		sortedIndex := -1
		for j := 0; j < len(cursors); j++ {
			if cursors[j].y == cursor.y && cursors[j].x == cursor.x {
				sortedIndex = j
				break
			}
		}
		
		if sortedIndex == -1 || len(linesToPaste[sortedIndex]) == 0 {
			continue
		}
		
		// Set buffer cursor to this position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, cursor.y)
		e.buffer.CursorX = cursor.x
		
		// Paste the lines for this cursor
		textToPaste := strings.Join(linesToPaste[sortedIndex], "\n")
		
		// Paste at this position
		if err := pasteFunc(textToPaste); err != nil {
			LL.Debug("Failed to paste at cursor: %v", err)
		}
		
		// Update cursor position if it's a virtual cursor
		if cursor.isVirtual {
			e.virtualCursors[cursor.index].Y = e.buffer.CursorIterator.LineNumber()
			e.virtualCursors[cursor.index].X = e.buffer.CursorX
		}
	}
	
	// Restore primary cursor to its final position
	if cursors[len(cursors)-1].isVirtual {
		// Primary cursor wasn't the last, restore its position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primaryLineNum)
		e.buffer.CursorX = primaryCursorX
	}
	
	// Restore view iterator
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	
	return nil
}

// executeMovementForAllCursors executes a movement command for all cursors
func (e *Editor) executeMovementForAllCursors(moveFunc func() error) error {
	if len(e.virtualCursors) == 0 {
		// No virtual cursors, just execute normally
		return moveFunc()
	}
	
	// Save primary cursor position
	primaryLineNum := e.buffer.CursorIterator.LineNumber()
	primaryCursorX := e.buffer.CursorX
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	
	// Move virtual cursors
	for i := range e.virtualCursors {
		// Set cursor to virtual cursor position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, e.virtualCursors[i].Y)
		e.buffer.CursorX = e.virtualCursors[i].X
		
		// Execute movement
		if err := moveFunc(); err != nil {
			LL.Debug("Failed to move virtual cursor: %v", err)
			continue
		}
		
		// Update virtual cursor position
		e.virtualCursors[i].Y = e.buffer.CursorIterator.LineNumber()
		e.virtualCursors[i].X = e.buffer.CursorX
	}
	
	// Restore and move primary cursor
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primaryLineNum)
	e.buffer.CursorX = primaryCursorX
	
	// Execute movement for primary cursor
	if err := moveFunc(); err != nil {
		return err
	}
	
	// Restore view iterator
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	
	return nil
}