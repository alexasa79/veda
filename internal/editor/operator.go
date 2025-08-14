package editor

import (
    "strings"
    "veda/internal/buffer"
    "veda/internal/clipboard"
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

    // Normalize empty-block newline: if we ended up with a blank line between
    // a line ending with '{' and a following line starting with '}', remove the blank line.
    editor.normalizeEmptyBlockNewlineAround(cursorLine)

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
	
	// Check if this is a line change operation (cc command)
	// For cc, we want to clear the line but keep it (not delete the newline)
	isLineChange := startChar == 0 && startLine == endLine
	if isLineChange {
		iter := buffer.NewBufferIteratorAtLine(editor.buffer, startLine)
		if !iter.IsEOF() {
			line := iter.Line()
			if endChar >= len(line) {
				// This is a cc operation - delete the line content but keep the line
				// Store view position before making changes
				viewLine := editor.buffer.ViewIterator.LineNumber()
				
				// Create delete entry for just the line content
				if len(line) > 0 {
					entry := buffer.NewDeleteTextEntry(startLine, 0, line)
					if editor.buffer.History != nil {
						if _, err := editor.buffer.History.Execute(editor.buffer, entry); err != nil {
							return err
						}
					} else {
						if err := entry.Execute(editor.buffer); err != nil {
							return err
						}
					}
				}
				
				// Position cursor at the beginning of the now-empty line
				editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(editor.buffer, startLine)
				editor.buffer.CursorX = 0
				
				// Update view iterator to ensure it's valid
				if viewLine >= editor.buffer.LineCount {
					viewLine = editor.buffer.LineCount - 1
					if viewLine < 0 {
						viewLine = 0
					}
				}
				editor.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(editor.buffer, viewLine)
				
				// Enter insert mode
				editor.switchToInsertMode()
				
				// Start tracking insert mode changes
				if editor.buffer.History != nil {
					editor.buffer.History.StartInsertSession("c")
				}
				
				return nil
			}
		}
	}
	
	// For all other cases, use the normal delete operation
	deleteOp := DeleteOperator{}
	if err := deleteOp.Execute(editor, startLine, startChar, endLine, endChar); err != nil {
		return err
	}
	
	// Then enter insert mode
	editor.switchToInsertMode()
	
	// Start tracking insert mode changes
	if editor.buffer != nil && editor.buffer.History != nil {
		editor.buffer.History.StartInsertSession("c")
	}
	
	return nil
}

func (c ChangeOperator) GetType() string {
	return "change"
}

// YankOperator yanks (copies) text in a given range to the system clipboard
type YankOperator struct{}

func (y YankOperator) Execute(editor *Editor, startLine, startChar, endLine, endChar int) error {
	if editor.buffer == nil {
		return nil
	}
	
	// Get the text to yank
	var textToYank string
	
	if startLine == endLine {
		// Single line yank
		iter := buffer.NewBufferIteratorAtLine(editor.buffer, startLine)
		if !iter.IsEOF() {
			line := iter.Line()
			if startChar < len(line) && endChar <= len(line) {
				textToYank = line[startChar:endChar]
			}
			
			// If this is a full line yank (starts at 0 and ends at line length),
			// add a newline to make it linewise
			if startChar == 0 && endChar == len(line) {
				textToYank += "\n"
			}
		}
	} else {
		// Multi-line yank
		textToYank = editor.getTextInRange(startLine, startChar, endLine, endChar)
		
		// If this is a full line yank (starts at 0 and ends at line length),
		// add a newline to make it linewise
		if startChar == 0 {
			iter := buffer.NewBufferIteratorAtLine(editor.buffer, endLine)
			if !iter.IsEOF() {
				line := iter.Line()
				if endChar == len(line) {
					textToYank += "\n"
				}
			}
		}
	}
	
	// Copy to system clipboard
	if err := clipboard.Copy(textToYank); err != nil {
		LL.Error("Failed to copy to clipboard: %v", err)
		return err
	}
	
	LL.Debug("Yanked %d characters to clipboard", len(textToYank))
	
	// Set flash highlight to show what was yanked
	editor.setFlashHighlight(startLine, startChar, endLine, endChar)
	
	// Yank doesn't move the cursor
	return nil
}

func (y YankOperator) GetType() string {
	return "yank"
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

// normalizeEmptyBlockNewlineAround removes a whitespace-only line between a line ending with
// an opening brace '{' and the following line that starts with a closing brace '}'.
// It operates around the provided line index (typically the cursor line after a delete).
func (e *Editor) normalizeEmptyBlockNewlineAround(lineIndex int) {
    if e.buffer == nil || e.buffer.LineCount < 3 {
        return
    }
    curr := lineIndex
    if curr <= 0 || curr >= e.buffer.LineCount-1 {
        return
    }

    // Read previous, current, and next lines
    prevIter := buffer.NewBufferIteratorAtLine(e.buffer, curr-1)
    currIter := buffer.NewBufferIteratorAtLine(e.buffer, curr)
    nextIter := buffer.NewBufferIteratorAtLine(e.buffer, curr+1)
    if prevIter.IsEOF() || currIter.IsEOF() || nextIter.IsEOF() {
        return
    }

    prevLine := prevIter.Line()
    currLine := currIter.Line()
    nextLine := nextIter.Line()

    // Check pattern: prev ends with '{', current is empty/whitespace, next begins with '}'
    prevTrimRight := strings.TrimRight(prevLine, " \t")
    if !strings.HasSuffix(prevTrimRight, "{") {
        return
    }

    if len(strings.TrimSpace(currLine)) != 0 {
        return
    }

    nextTrim := strings.TrimSpace(nextLine)
    if !(len(nextTrim) > 0 && nextTrim[0] == '}') {
        return
    }

    // Delete the current blank line including its newline
    // Compose the text to delete (line content + newline)
    textToDelete := currLine + "\n"
    entry := buffer.NewDeleteTextEntry(curr, 0, textToDelete)
    if e.buffer.History != nil {
        if _, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
            return
        }
    } else {
        if err := entry.Execute(e.buffer); err != nil {
            return
        }
    }

    // Keep cursor at the same visual place (line stays the same index but now points to the next line)
    e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, curr)
    if e.buffer.CursorIterator.IsEOF() && curr > 0 {
        e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, curr-1)
    }
    e.buffer.CursorX = min(e.buffer.CursorX, len(e.buffer.CursorIterator.Line()))
}