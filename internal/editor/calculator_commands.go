package editor

import (
	"veda/internal/buffer"
	"veda/internal/calculator"
)

// calculateAppend evaluates the mathematical expression before the cursor and appends the result
func (e *Editor) calculateAppend(args []string) error {
	// Get current line and cursor position
	lineNum := e.buffer.CursorIterator.LineNumber()
	line := e.buffer.CursorIterator.Line()
	cursorX := e.buffer.CursorX
	// Save view position before any buffer modifications
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	
	// If cursor is past the end of the line, adjust to the end
	if cursorX > len(line) {
		cursorX = len(line)
	}
	
	// Process the line to find and evaluate the expression
	newLine, err := calculator.ProcessLineAppend(line, cursorX)
	if err != nil {
		// No valid expression found
		e.ShowError("No valid expression found")
		return nil
	}
	
	// Replace the current line with the new line containing the result
	// First, delete the current line content
	if len(line) > 0 {
		deleteEntry := buffer.NewDeleteTextEntry(lineNum, 0, line)
		if e.buffer.History != nil {
			if _, err := e.buffer.History.Execute(e.buffer, deleteEntry); err != nil {
				return err
			}
		}
	}
	
	// Then insert the new line content
	insertEntry := buffer.NewInsertTextEntry(lineNum, 0, newLine)
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, insertEntry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	}
	
	// Move cursor to the end of the new line
	e.buffer.CursorX = len(newLine)
	
	// Recreate iterators after buffer modifications
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	return nil
}

// calculateReplace evaluates the mathematical expression before the cursor and replaces it with the result
func (e *Editor) calculateReplace(args []string) error {
	// Get current line and cursor position
	lineNum := e.buffer.CursorIterator.LineNumber()
	line := e.buffer.CursorIterator.Line()
	cursorX := e.buffer.CursorX
	// Save view position before any buffer modifications
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	
	// If cursor is past the end of the line, adjust to the end
	if cursorX > len(line) {
		cursorX = len(line)
	}
	
	// Process the line to find and evaluate the expression
	newLine, newCursorX, err := calculator.ProcessLineReplace(line, cursorX)
	if err != nil {
		// No valid expression found
		e.ShowError("No valid expression found")
		return nil
	}
	
	// Replace the current line with the new line containing the result
	// First, delete the current line content
	if len(line) > 0 {
		deleteEntry := buffer.NewDeleteTextEntry(lineNum, 0, line)
		if e.buffer.History != nil {
			if _, err := e.buffer.History.Execute(e.buffer, deleteEntry); err != nil {
				return err
			}
		}
	}
	
	// Then insert the new line content
	insertEntry := buffer.NewInsertTextEntry(lineNum, 0, newLine)
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, insertEntry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	}
	
	// Move cursor to the new position
	e.buffer.CursorX = newCursorX
	
	// Recreate iterators after buffer modifications
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	return nil
}