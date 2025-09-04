package editor

import (
	"veda/internal/buffer"
	"veda/internal/calculator"
	"veda/internal/commands"
)

// calculateAppend evaluates the mathematical expression before the cursor and appends the result
func (e *Editor) calculateAppend(args []string) error {
	// Get current line and cursor position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	line := e.CurrentBuffer().CursorIterator.Line()
	cursorX := e.CurrentBuffer().CursorX
	// Save view position before any buffer modifications
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// If cursor is past the end of the line, adjust to the end
	if cursorX > len(line) {
		cursorX = len(line)
	}

	// In Normal mode, the cursor sits ON the character; include it in expression end.
	// In Insert mode, the cursor is AFTER the last typed character; use position as-is.
	endPos := cursorX
	if e.mode != commands.InsertMode && endPos < len(line) {
		endPos++
	}

	// Process the line to find and evaluate the expression
	newLine, err := calculator.ProcessLineAppend(line, endPos)
	if err != nil {
		// No valid expression found
		e.ShowError("No valid expression found")
		return nil
	}

	// Replace the current line with the new line containing the result
	// First, delete the current line content
	if len(line) > 0 {
		deleteEntry := buffer.NewDeleteTextEntry(lineNum, 0, line)
		if e.CurrentBuffer().History != nil {
			if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), deleteEntry); err != nil {
				return err
			}
		}
	}

	// Then insert the new line content
	insertEntry := buffer.NewInsertTextEntry(lineNum, 0, newLine)
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), insertEntry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	}

	// Move cursor to the end of the new line
	e.CurrentBuffer().CursorX = len(newLine)

	// Recreate iterators after buffer modifications
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}

// calculateReplace evaluates the mathematical expression before the cursor and replaces it with the result
func (e *Editor) calculateReplace(args []string) error {
	// Get current line and cursor position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	line := e.CurrentBuffer().CursorIterator.Line()
	cursorX := e.CurrentBuffer().CursorX
	// Save view position before any buffer modifications
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// If cursor is past the end of the line, adjust to the end
	if cursorX > len(line) {
		cursorX = len(line)
	}

	// In Normal mode, the cursor sits ON the character; include it in expression end.
	// In Insert mode, the cursor is AFTER the last typed character; use position as-is.
	endPos := cursorX
	if e.mode != commands.InsertMode && endPos < len(line) {
		endPos++
	}

	// Process the line to find and evaluate the expression
	newLine, newCursorX, err := calculator.ProcessLineReplace(line, endPos)
	if err != nil {
		// No valid expression found
		e.ShowError("No valid expression found")
		return nil
	}

	// Replace the current line with the new line containing the result
	// First, delete the current line content
	if len(line) > 0 {
		deleteEntry := buffer.NewDeleteTextEntry(lineNum, 0, line)
		if e.CurrentBuffer().History != nil {
			if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), deleteEntry); err != nil {
				return err
			}
		}
	}

	// Then insert the new line content
	insertEntry := buffer.NewInsertTextEntry(lineNum, 0, newLine)
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), insertEntry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	}

	// Move cursor to the new position
	e.CurrentBuffer().CursorX = newCursorX

	// Recreate iterators after buffer modifications
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}
