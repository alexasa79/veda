package editor

import (
	"veda/internal/buffer"
	"veda/internal/commands"
)

// RepeatableCommand stores an operator and text object combination that can be repeated
type RepeatableCommand struct {
	Operator   Operator
	TextObject TextObject
}

// Operator represents an action that can be performed on a text object
type Operator interface {
	// Execute performs the operation on the given range
	Execute(editor *Editor, startLine, startChar, endLine, endChar int) error
	
	// GetType returns the operator type
	GetType() string
}

// DeleteOperator deletes text in a given range
type DeleteOperator struct{}

func (d DeleteOperator) Execute(editor *Editor, startLine, startChar, endLine, endChar int) error {
	if editor.buffer == nil {
		return nil
	}
	
	// Remember the view position before we make changes
	viewLine := editor.buffer.ViewIterator.LineNumber()
	
	// Calculate the text to delete
	var textToDelete string
	
	if startLine == endLine {
		// Single line deletion
		iter := buffer.NewBufferIteratorAtLine(editor.buffer, startLine)
		if !iter.IsEOF() {
			line := iter.Line()
			if startChar < len(line) && endChar <= len(line) {
				textToDelete = line[startChar:endChar]
			}
		}
	} else {
		// Multi-line deletion
		textToDelete = editor.getTextInRange(startLine, startChar, endLine, endChar)
	}
	
	// Check if this is a line deletion (dd command)
	isLineDeletion := false
	if startChar == 0 && startLine == endLine {
		iter := buffer.NewBufferIteratorAtLine(editor.buffer, startLine)
		line := iter.Line()
		if endChar >= len(line) && startLine < editor.buffer.LineCount-1 {
			// Include the newline character
			textToDelete = line + "\n"
			endChar = 0
			endLine = startLine + 1
			isLineDeletion = true
		} else if endChar >= len(line) && startLine == editor.buffer.LineCount-1 && startLine > 0 {
			// Last line - delete the newline from the previous line
			prevIter := buffer.NewBufferIteratorAtLine(editor.buffer, startLine-1)
			prevLine := prevIter.Line()
			startLine = startLine - 1
			startChar = len(prevLine)
			textToDelete = "\n" + line
			isLineDeletion = true
		}
	}
	
	// Create delete entry for history
	entry := buffer.NewDeleteTextEntry(startLine, startChar, textToDelete)
	
	// Execute through history
	if editor.buffer.History != nil {
		if executed, err := editor.buffer.History.Execute(editor.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			// Don't set lastChange here - it's handled by executeOperatorWithTextObject
			// which sets lastCommand instead
		}
	} else {
		if err := entry.Execute(editor.buffer); err != nil {
			return err
		}
	}
	
	// After delete, cursor should be at the start position
	// But we need to check bounds first
	cursorLine := startLine
	cursorChar := startChar
	
	// Handle special cases
	if editor.buffer.LineCount == 0 {
		// Buffer is now empty
		editor.buffer.CursorIterator = buffer.NewBufferIterator(editor.buffer)
		editor.buffer.ViewIterator = buffer.NewBufferIterator(editor.buffer)
		editor.buffer.CursorX = 0
		return nil
	}
	
	// Adjust cursor position if we deleted entire lines
	if startChar == 0 && endLine > startLine {
		// We deleted entire lines
		if cursorLine >= editor.buffer.LineCount {
			cursorLine = editor.buffer.LineCount - 1
		}
	}
	
	// Recreate cursor iterator at the new position
	editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(editor.buffer, cursorLine)
	
	// Recreate view iterator to ensure it's valid after buffer changes
	if viewLine >= editor.buffer.LineCount {
		viewLine = editor.buffer.LineCount - 1
		if viewLine < 0 {
			viewLine = 0
		}
	}
	editor.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(editor.buffer, viewLine)
	
	// Keep view where it was (don't change the view position)
	// The view should only be adjusted if cursor goes out of visible range
	
	// Set cursor X position
	if isLineDeletion {
		// For line deletions (dd), cursor should go to beginning of line
		editor.buffer.CursorX = 0
	} else {
		editor.buffer.CursorX = cursorChar
	}
	
	// Ensure cursor X is within bounds
	line := editor.buffer.CursorIterator.Line()
	if editor.buffer.CursorX > len(line) {
		editor.buffer.CursorX = len(line)
		if editor.buffer.CursorX > 0 && editor.mode == commands.NormalMode {
			editor.buffer.CursorX--
		}
	}
	
	// Adjust scroll to ensure cursor is visible
	editor.scroll()
	
	// TODO: Save deleted text to register for paste operations
	
	return nil
}

func (d DeleteOperator) GetType() string {
	return "delete"
}

// ChangeOperator deletes text in a given range and enters insert mode
type ChangeOperator struct{}

func (c ChangeOperator) Execute(editor *Editor, startLine, startChar, endLine, endChar int) error {
	if editor.buffer == nil {
		return nil
	}
	
	// First, execute the delete operation (reuse delete logic)
	deleteOp := DeleteOperator{}
	if err := deleteOp.Execute(editor, startLine, startChar, endLine, endChar); err != nil {
		return err
	}
	
	// Then enter insert mode
	editor.mode = commands.InsertMode
	
	// Start tracking insert mode changes
	if editor.buffer != nil && editor.buffer.History != nil {
		editor.buffer.History.StartInsertSession("c")
	}
	
	return nil
}

func (c ChangeOperator) GetType() string {
	return "change"
}

// getTextInRange extracts text from a range spanning multiple lines
func (e *Editor) getTextInRange(startLine, startChar, endLine, endChar int) string {
	var result string
	
	for line := startLine; line <= endLine; line++ {
		iter := buffer.NewBufferIteratorAtLine(e.buffer, line)
		if iter.IsEOF() {
			break
		}
		
		lineStr := iter.Line()
		
		if line == startLine && line == endLine {
			// Single line
			if startChar < len(lineStr) && endChar <= len(lineStr) {
				result += lineStr[startChar:endChar]
			}
		} else if line == startLine {
			// First line
			if startChar < len(lineStr) {
				result += lineStr[startChar:] + "\n"
			}
		} else if line == endLine {
			// Last line
			if endChar <= len(lineStr) {
				result += lineStr[:endChar]
			}
		} else {
			// Middle lines
			result += lineStr + "\n"
		}
	}
	
	return result
}