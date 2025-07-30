package editor

import (
	"veda/internal/buffer"
	"veda/internal/commands"
)

// Edit operations handle text insertion, deletion, and manipulation

// openLineBelow opens a new line below the current line and enters insert mode
func (e *Editor) openLineBelow(args []string) error {
	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()

	// Get the current line and move to end
	currentLine := ""
	if !e.buffer.CursorIterator.IsEOF() {
		currentLine = e.buffer.CursorIterator.Line()
		e.buffer.CursorX = len(currentLine)
	}

	// Calculate auto-indent for the new line
	autoIndent := e.buffer.CalculateAutoIndent(currentLine, e.buffer.CursorX)

	// Create history entry for the newline and indentation
	textToInsert := "\n" + autoIndent.IndentStr
	entry := buffer.NewInsertTextEntry(lineNum, e.buffer.CursorX, textToInsert)

	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		pos := e.getAbsolutePosition()
		e.buffer.Insert(pos, textToInsert)
	}

	// Recreate iterators - cursor moves to next line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum + 1)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Move cursor to end of indentation
	e.buffer.CursorX = len(autoIndent.IndentStr)

	// Enter insert mode
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession("o")
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// openLineAbove opens a new line above the current line and enters insert mode
func (e *Editor) openLineAbove(args []string) error {
	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()

	// Get the previous line for auto-indent calculation (if we're not on the first line)
	previousLine := ""
	if lineNum > 0 {
		// Temporarily move to previous line to get its content
		prevIter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum - 1)
		if !prevIter.IsEOF() {
			previousLine = prevIter.Line()
		}
	}

	// Calculate auto-indent based on the previous line
	// Use the length of the previous line as cursor position for calculation
	autoIndent := e.buffer.CalculateAutoIndent(previousLine, len(previousLine))

	// Move to beginning of current line
	e.buffer.CursorX = 0

	// Create history entry for the newline
	entry := buffer.NewInsertTextEntry(lineNum, 0, "\n")

	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		pos := e.getAbsolutePosition()
		e.buffer.Insert(pos, "\n")
	}

	// The cursor stays at the same line number (which is now the new empty line)
	// because we inserted before the current position
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Insert indentation at the beginning of the new line
	if autoIndent.IndentStr != "" {
		// Create history entry for the indentation
		indentEntry := buffer.NewInsertTextEntry(lineNum, 0, autoIndent.IndentStr)

		// Execute through history
		if e.buffer.History != nil {
			if executed, err := e.buffer.History.Execute(e.buffer, indentEntry); err != nil {
				return err
			} else if executed != nil {
				e.setLastChange(executed)
			}
		} else {
			// Fallback if history is not initialized
			pos := e.getAbsolutePosition()
			e.buffer.Insert(pos, autoIndent.IndentStr)
		}

		// Recreate iterators after insertion
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	}

	// Move cursor to end of indentation
	e.buffer.CursorX = len(autoIndent.IndentStr)

	// Enter insert mode
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession("O")
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// joinLines joins the current line with the next line
func (e *Editor) joinLines(args []string) error {
	// Get current line info
	lineNum := e.buffer.CursorIterator.LineNumber()
	
	// Can't join if we're on the last line
	if lineNum >= e.buffer.LineCount - 1 {
		return nil
	}
	
	// Get current line and next line
	currentLine := e.buffer.CursorIterator.Line()
	nextIter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum + 1)
	if nextIter.IsEOF() {
		return nil
	}
	nextLine := nextIter.Line()
	
	// Determine what to delete and what to insert
	// We need to delete the newline and any leading whitespace on the next line
	leadingSpaces := 0
	for i, ch := range nextLine {
		if ch != ' ' && ch != '\t' {
			leadingSpaces = i
			break
		}
		if i == len(nextLine) - 1 {
			// Next line is all whitespace
			leadingSpaces = len(nextLine)
		}
	}
	
	// Determine what text to insert between the lines
	insertText := ""
	if len(currentLine) > 0 && currentLine[len(currentLine)-1] != ' ' && 
	   leadingSpaces < len(nextLine) && nextLine[leadingSpaces] != ' ' {
		// Add a space if current line doesn't end with space and next line content doesn't start with space
		insertText = " "
	}
	
	// Create a composite entry for the join operation
	compositeEntry := buffer.NewCompositeEntry("Join lines")
	
	// First, insert the space if needed
	if insertText != "" {
		insertEntry := buffer.NewInsertTextEntry(lineNum, len(currentLine), insertText)
		compositeEntry.Add(insertEntry)
	}
	
	// Then delete the newline and leading whitespace
	deleteEntry := buffer.NewDeleteTextEntry(lineNum, len(currentLine) + len(insertText), "\n" + nextLine[:leadingSpaces])
	compositeEntry.Add(deleteEntry)
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, compositeEntry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	}
	
	// Remember where we want the cursor to be (at the join point)
	cursorX := len(currentLine)
	if insertText != "" {
		cursorX++ // Account for the added space
	}
	
	// Recreate iterators
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, e.buffer.ViewIterator.LineNumber())
	
	// Set cursor position to the join point
	e.buffer.CursorX = cursorX
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	return nil
}

func (e *Editor) insertChar(ch rune) error {
	// Check if we should skip over a closing character
	if e.shouldSkipClosingChar(ch) {
		// Just move cursor forward without inserting
		e.buffer.CursorX++
		return nil
	}

	// Check for auto-pair characters
	pairedChar, isPaired := e.getPairedChar(ch)

	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	cursorX := e.buffer.CursorX

	// Create history entry
	textToInsert := string(ch)
	if isPaired {
		textToInsert = string(ch) + string(pairedChar)
	}

	entry := buffer.NewInsertTextEntry(lineNum, cursorX, textToInsert)

	// Execute through history (this handles both direct execution and insert mode tracking)
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.buffer); err != nil {
			return err
		}
	}

	// Recreate iterators at the same line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Move cursor appropriately
	if isPaired {
		// Move cursor between the pair (only forward by 1)
		e.buffer.CursorX = cursorX + 1
	} else {
		// Move cursor forward
		e.buffer.CursorX = cursorX + 1
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	// Update completion menu filter if visible
	e.updateCompletionFilter()

	return nil
}

// shouldSkipClosingChar checks if we should skip over a closing character
// This happens when the character at the cursor position matches what we're trying to type
// and it's a closing character from auto-pairing
func (e *Editor) shouldSkipClosingChar(ch rune) bool {
	// Check if this is a closing character
	isClosing := false
	switch ch {
	case ')', ']', '}', '"', '\'', '`':
		isClosing = true
	}
	
	if !isClosing {
		return false
	}
	
	// Check if cursor is at a position where we have this character
	if e.buffer.CursorIterator.IsEOF() {
		return false
	}
	
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX >= len(line) {
		return false
	}
	
	// Check if the character at cursor position matches what we're typing
	return rune(line[e.buffer.CursorX]) == ch
}

// getPairedChar returns the closing character for auto-pairing and whether the character should be paired
func (e *Editor) getPairedChar(ch rune) (rune, bool) {
	switch ch {
	case '(':
		return ')', true
	case '[':
		return ']', true
	case '{':
		return '}', true
	case '"':
		return '"', true
	case '\'':
		return '\'', true
	case '`':
		return '`', true
	default:
		return 0, false
	}
}

func (e *Editor) insertNewline(args []string) error {
	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	cursorX := e.buffer.CursorX

	// Get the current line for auto-indent calculation
	currentLine := ""
	if !e.buffer.CursorIterator.IsEOF() {
		currentLine = e.buffer.CursorIterator.Line()
	}

	// Calculate auto-indent for the new line
	autoIndent := e.buffer.CalculateAutoIndent(currentLine, e.buffer.CursorX)

	// Create history entry for newline + indentation
	textToInsert := "\n" + autoIndent.IndentStr
	entry := buffer.NewInsertTextEntry(lineNum, cursorX, textToInsert)

	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.buffer); err != nil {
			return err
		}
	}

	// Recreate iterators after buffer modification
	// Cursor moves to next line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum + 1)
	// View iterator stays at its current line
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Move cursor to end of indentation
	e.buffer.CursorX = len(autoIndent.IndentStr)

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

func (e *Editor) backspace(args []string) error {
	// If at beginning of buffer, nothing to delete
	if e.buffer.CursorX == 0 && e.buffer.CursorIterator.LineNumber() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	cursorX := e.buffer.CursorX
	pos := e.getAbsolutePosition()

	if cursorX == 0 {
		// At beginning of line, join with previous line
		if pos > 0 && lineNum > 0 {
			// Store the length of the previous line before joining
			prevIter := e.buffer.CursorIterator.Previous()
			prevLineLen := 0
			if !prevIter.Equals(e.buffer.CursorIterator) {
				prevLineLen = len(prevIter.Line())
			}

			// We're deleting a newline character
			deletedText := "\n"
			entry := buffer.NewDeleteTextEntry(lineNum-1, prevLineLen, deletedText)

			// Execute through history
			if e.buffer.History != nil {
				if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
					return err
				} else if executed != nil {
					e.setLastChange(executed)
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.buffer); err != nil {
					return err
				}
			}

			// Recreate iterators - cursor moves to previous line
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum-1)
			e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

			// Position cursor at end of previous line
			e.buffer.CursorX = prevLineLen

			// Update syntax highlighting
			e.updateSyntaxHighlighting()
		}
	} else {
		// Delete character before cursor
		if pos > 0 {
			// Get the character being deleted
			deletedChar := e.buffer.Rope.ByteAt(pos - 1)
			deletedText := string(deletedChar)

			entry := buffer.NewDeleteTextEntry(lineNum, cursorX-1, deletedText)

			// Execute through history
			if e.buffer.History != nil {
				if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
					return err
				} else if executed != nil {
					e.setLastChange(executed)
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.buffer); err != nil {
					return err
				}
			}

			// Recreate iterators at the same line
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
			e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

			// Move cursor back one position
			e.buffer.CursorX = cursorX - 1

			// Update syntax highlighting
			e.updateSyntaxHighlighting()
			
			// Update completion menu filter if visible
			e.updateCompletionFilter()
		}
	}

	return nil
}

// deleteCurrentCharacter deletes the character at the current cursor position
func (e *Editor) deleteCurrentCharacter(args []string) error {
	// If at EOF or empty buffer, nothing to delete
	if e.buffer.CursorIterator.IsEOF() || e.buffer.Rope.Length() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()

	// Get current line to check bounds
	line := e.buffer.CursorIterator.Line()

	// If cursor is past the end of the line, check if we can delete newline
	if e.buffer.CursorX >= len(line) {
		// Check if there's a newline character at the end of this line
		pos := e.getAbsolutePosition()
		if pos < e.buffer.Rope.Length() && e.buffer.Rope.ByteAt(pos) == '\n' {
			// We can delete the newline
			deletedText := "\n"
			entry := buffer.NewDeleteTextEntry(lineNum, e.buffer.CursorX, deletedText)

			// Execute through history
			if e.buffer.History != nil {
				if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
					return err
				} else if executed != nil {
					e.setLastChange(executed)
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.buffer); err != nil {
					return err
				}
			}

			// Recreate iterators at the same position
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
			e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

			// Update syntax highlighting
			e.updateSyntaxHighlighting()
			return nil
		}
		return nil
	}

	// Get cursor X position
	cursorX := e.buffer.CursorX

	// Get the character being deleted
	pos := e.getAbsolutePosition()
	deletedChar := e.buffer.Rope.ByteAt(pos)
	deletedText := string(deletedChar)

	// Create history entry
	entry := buffer.NewDeleteTextEntry(lineNum, cursorX, deletedText)

	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.buffer); err != nil {
			return err
		}
	}

	// Recreate iterators at the same line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Cursor position stays the same, but we need to ensure it's not past the end of the line
	newLine := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(newLine) {
		if len(newLine) > 0 {
			e.buffer.CursorX = len(newLine) - 1 // Stay on last character in normal mode
		} else {
			e.buffer.CursorX = 0
		}
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}