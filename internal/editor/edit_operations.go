package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
)

// Edit operations handle text insertion, deletion, and manipulation

// openLineBelow opens a new line below the current line and enters insert mode
func (e *Editor) openLineBelow(args []string) error {
	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// Get the current line and move to end
	currentLine := ""
	if !e.CurrentBuffer().CursorIterator.IsEOF() {
		currentLine = e.CurrentBuffer().CursorIterator.Line()
		e.CurrentBuffer().CursorX = len(currentLine)
	}

	// Calculate auto-indent for the new line
	autoIndent := e.CurrentBuffer().CalculateAutoIndent(currentLine, e.CurrentBuffer().CursorX)

	// Create history entry for the newline and indentation
	textToInsert := "\n" + autoIndent.IndentStr
	entry := buffer.NewInsertTextEntry(lineNum, e.CurrentBuffer().CursorX, textToInsert)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		pos := e.getAbsolutePosition()
		e.CurrentBuffer().Insert(pos, textToInsert)
	}

	// Recreate iterators - cursor moves to next line
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum+1)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Move cursor to end of indentation
	e.CurrentBuffer().CursorX = len(autoIndent.IndentStr)

	// Track auto-indentation for cleanup when cursor moves away
	if len(autoIndent.IndentStr) > 0 {
		e.trackAutoIndent(lineNum+1, len(autoIndent.IndentStr))
	}

	// Enter insert mode
	e.switchToInsertMode()
	// Start tracking insert mode changes
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.StartInsertSession("o")
	}

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	// Ensure the new line is visible
	e.scroll()

	return nil
}

// openLineAbove opens a new line above the current line and enters insert mode
func (e *Editor) openLineAbove(args []string) error {
	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// Get the previous line for auto-indent calculation (if we're not on the first line)
	previousLine := ""
	if lineNum > 0 {
		// Temporarily move to previous line to get its content
		prevIter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum-1)
		if !prevIter.IsEOF() {
			previousLine = prevIter.Line()
		}
	}

	// Calculate auto-indent based on the previous line
	// Use the length of the previous line as cursor position for calculation
	autoIndent := e.CurrentBuffer().CalculateAutoIndent(previousLine, len(previousLine))

	// Move to beginning of current line
	e.CurrentBuffer().CursorX = 0

	// Create history entry for the newline
	entry := buffer.NewInsertTextEntry(lineNum, 0, "\n")

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		pos := e.getAbsolutePosition()
		e.CurrentBuffer().Insert(pos, "\n")
	}

	// The cursor stays at the same line number (which is now the new empty line)
	// because we inserted before the current position
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Insert indentation at the beginning of the new line
	if autoIndent.IndentStr != "" {
		// Create history entry for the indentation
		indentEntry := buffer.NewInsertTextEntry(lineNum, 0, autoIndent.IndentStr)

		// Execute through history
		if e.CurrentBuffer().History != nil {
			if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), indentEntry); err != nil {
				return err
			} else if executed != nil {
				e.setLastChange(executed)
			}
		} else {
			// Fallback if history is not initialized
			pos := e.getAbsolutePosition()
			e.CurrentBuffer().Insert(pos, autoIndent.IndentStr)
		}

		// Recreate iterators after insertion
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)
	}

	// Move cursor to end of indentation
	e.CurrentBuffer().CursorX = len(autoIndent.IndentStr)

	// Track auto-indentation for cleanup when cursor moves away
	if len(autoIndent.IndentStr) > 0 {
		e.trackAutoIndent(lineNum, len(autoIndent.IndentStr))
	}

	// Enter insert mode
	e.switchToInsertMode()
	// Start tracking insert mode changes
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.StartInsertSession("O")
	}

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}

// joinLines joins the current line with the next line
func (e *Editor) joinLines(args []string) error {
	// Get current line info and save view position before buffer modifications
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// Can't join if we're on the last line
	if lineNum >= e.CurrentBuffer().LineCount-1 {
		return nil
	}

	// Get current line and next line
	currentLine := e.CurrentBuffer().CursorIterator.Line()
	nextIter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum+1)
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
		if i == len(nextLine)-1 {
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
	deleteEntry := buffer.NewDeleteTextEntry(lineNum, len(currentLine)+len(insertText), "\n"+nextLine[:leadingSpaces])
	compositeEntry.Add(deleteEntry)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), compositeEntry); err != nil {
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
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Set cursor position to the join point
	e.CurrentBuffer().CursorX = cursorX

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}

// insertTab inserts appropriate indentation at the current cursor position
func (e *Editor) insertTab(args []string) error {
	// Get the buffer's indentation string
	indentStr := e.CurrentBuffer().GetIndentationString()

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX

	// Create history entry
	entry := buffer.NewInsertTextEntry(lineNum, cursorX, indentStr)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		pos := e.getAbsolutePosition()
		e.CurrentBuffer().Insert(pos, indentStr)
	}

	// Recreate iterators at the same line
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Move cursor forward by the length of the indentation
	e.CurrentBuffer().CursorX = cursorX + len(indentStr)

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	// Update completion menu filter if visible
	e.updateCompletionFilter()

	return nil
}

func (e *Editor) insertChar(ch rune) error {
	// Auto-pairing feature has been disabled
	// Simply insert the character without any pairing or skipping logic

	// Skip if no buffer (empty window)
	if e.CurrentBuffer() == nil {
		return nil
	}

	// Reset desired column when inserting text
	e.desiredColumn = -1

	// If user types non-whitespace content, mark the line as having user content
	if ch != ' ' && ch != '\t' {
		e.markLineHasUserContent(e.CurrentBuffer().CursorIterator.LineNumber())
	}

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX

	// Check if we're typing a closing brace and should auto-dedent (disabled during paste)
	if !e.inPasteMode && ch == '}' && e.shouldAutoDedentClosingBrace(lineNum, cursorX) {
		// Apply auto-dedent before inserting the closing brace
		if err := e.autoDedentCurrentLine(lineNum); err != nil {
			return err
		}
		// Update cursor position after dedent
		cursorX = e.CurrentBuffer().CursorX
		// Recreate iterators after buffer modification
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)
	}

	// Create history entry for the single character
	textToInsert := string(ch)
	entry := buffer.NewInsertTextEntry(lineNum, cursorX, textToInsert)

	// Execute through history (this handles both direct execution and insert mode tracking)
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.CurrentBuffer()); err != nil {
			return err
		}
	}

	// Recreate iterators at the same line
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Move cursor forward by one position
	e.CurrentBuffer().CursorX = cursorX + 1

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	// Update completion menu filter if visible
	e.updateCompletionFilter()

	// If we just typed an opening parenthesis in insert mode, trigger signature help
	if !e.inPasteMode && ch == '(' && e.mode == commands.InsertMode {
		// Check if we're now in a function call context
		if e.isLikelyInsideCallOnCurrentLine() {
			// Increment generation and request signature help
			e.CurrentBuffer().SignatureGeneration++
			e.requestSignatureHelpAsync()
		}
	}

	// Auto-trigger completion menu for member access operators in insert mode with LSP
	if !e.inPasteMode && e.mode == commands.InsertMode && e.CurrentBuffer() != nil && e.CurrentBuffer().HasLSP {
		shouldTriggerCompletion := false

		if ch == '.' {
			// Triggered by typing '.'
			shouldTriggerCompletion = true
		} else if ch == '>' {
			// Check if we just typed '->' (arrow operator)
			line := e.CurrentBuffer().CursorIterator.Line()
			if e.CurrentBuffer().CursorX >= 2 && len(line) >= 2 {
				// Check if the previous character is '-' to form '->'
				if line[e.CurrentBuffer().CursorX-2] == '-' {
					shouldTriggerCompletion = true
				}
			}
		}

		if shouldTriggerCompletion {
			// Trigger completion menu automatically
			e.showCompletions()
		}
	}

	return nil
}

// Auto-pairing functions removed - feature has been disabled

func (e *Editor) insertNewline(args []string) error {
	// Debug: entering insertNewline
	LL.Debug("insertNewline: start at line=%d, x=%d, viewTop=%d, pasteMode=%v",
		e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX, e.CurrentBuffer().ViewIterator.LineNumber(), e.inPasteMode)
	// During paste mode, insert a raw newline without auto-indentation
	if e.inPasteMode {
		LL.Trace("insertNewline: paste mode path")
		lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
		viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
		cursorX := e.CurrentBuffer().CursorX

		entry := buffer.NewInsertTextEntry(lineNum, cursorX, "\n")

		if e.CurrentBuffer().History != nil {
			if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
				return err
			} else if executed != nil {
				e.setLastChange(executed)
			}
		} else {
			if err := entry.Execute(e.CurrentBuffer()); err != nil {
				return err
			}
		}

		// Move cursor to the next line at column 0
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum+1)
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)
		e.CurrentBuffer().CursorX = 0

		// Skip highlighting during paste; it will be refreshed once at paste end
		LL.Debug("insertNewline: (paste) after insert -> line=%d, x=%d, viewTop=%d",
			e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX, e.CurrentBuffer().ViewIterator.LineNumber())
		return nil
	}
	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX

	// Get the current line for auto-indent calculation
	currentLine := ""
	if !e.CurrentBuffer().CursorIterator.IsEOF() {
		currentLine = e.CurrentBuffer().CursorIterator.Line()
	}

	// Check if cursor is between braces on the same line
	isBetweenBraces := e.isCursorBetweenBraces(currentLine, cursorX)

	// Calculate auto-indent for the new line
	autoIndent := e.CurrentBuffer().CalculateAutoIndent(currentLine, e.CurrentBuffer().CursorX)

	// Create the text to insert
	var textToInsert string
	var finalCursorX int

	if isBetweenBraces {
		// Special case: cursor between {}
		// Insert newline with indent, then another newline with base indent for closing brace
		baseIndentInfo := e.CurrentBuffer().GetLineIndentInfo(currentLine)
		baseIndentStr := strings.Repeat(e.CurrentBuffer().GetIndentationString(), baseIndentInfo.Level)

		textToInsert = "\n" + autoIndent.IndentStr + "\n" + baseIndentStr
		finalCursorX = len(autoIndent.IndentStr)
	} else {
		// Normal case
		textToInsert = "\n" + autoIndent.IndentStr
		finalCursorX = len(autoIndent.IndentStr)
	}

	entry := buffer.NewInsertTextEntry(lineNum, cursorX, textToInsert)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.CurrentBuffer()); err != nil {
			return err
		}
	}

	// Recreate iterators after buffer modification
	// Cursor moves to next line (the indented line)
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum+1)
	// View iterator stays at its current line
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Move cursor to end of indentation
	e.CurrentBuffer().CursorX = finalCursorX

	// Track auto-indentation for cleanup when cursor moves away
	if len(autoIndent.IndentStr) > 0 {
		e.trackAutoIndent(lineNum+1, len(autoIndent.IndentStr))
	}

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	// Ensure the cursor's new line is visible immediately to avoid one-frame misplacement
	e.scroll()

	LL.Debug("insertNewline: end -> line=%d, x=%d, viewTop=%d (finalX=%d, betweenBraces=%v, indent=%q)",
		e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX, e.CurrentBuffer().ViewIterator.LineNumber(), finalCursorX, isBetweenBraces, autoIndent.IndentStr)

	return nil
}

// isCursorBetweenBraces checks if the cursor is positioned between { and } on the same line
func (e *Editor) isCursorBetweenBraces(line string, cursorX int) bool {
	if cursorX <= 0 || cursorX >= len(line) {
		return false
	}

	// Look backward for {
	hasOpenBrace := false
	for i := cursorX - 1; i >= 0; i-- {
		if line[i] == '{' {
			hasOpenBrace = true
			break
		} else if line[i] != ' ' && line[i] != '\t' {
			// Found non-whitespace character that's not {
			break
		}
	}

	if !hasOpenBrace {
		return false
	}

	// Look forward for }
	for i := cursorX; i < len(line); i++ {
		if line[i] == '}' {
			return true
		} else if line[i] != ' ' && line[i] != '\t' {
			// Found non-whitespace character that's not }
			break
		}
	}

	return false
}

func (e *Editor) backspace(args []string) error {
	// If at beginning of buffer, nothing to delete
	if e.CurrentBuffer().CursorX == 0 && e.CurrentBuffer().CursorIterator.LineNumber() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX
	pos := e.getAbsolutePosition()

	if cursorX == 0 {
		// At beginning of line, join with previous line
		if pos > 0 && lineNum > 0 {
			// Store the length of the previous line before joining
			prevIter := e.CurrentBuffer().CursorIterator.Previous()
			prevLineLen := 0
			if !prevIter.Equals(e.CurrentBuffer().CursorIterator) {
				prevLineLen = len(prevIter.Line())
			}

			// We're deleting a newline character
			deletedText := "\n"
			entry := buffer.NewDeleteTextEntry(lineNum-1, prevLineLen, deletedText)

			// Execute through history
			if e.CurrentBuffer().History != nil {
				if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
					return err
				} else if executed != nil {
					e.setLastChange(executed)
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.CurrentBuffer()); err != nil {
					return err
				}
			}

			// Recreate iterators - cursor moves to previous line
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum-1)
			e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

			// Position cursor at end of previous line
			e.CurrentBuffer().CursorX = prevLineLen

			// Update syntax highlighting (skip during paste mode)
			if !e.inPasteMode {
				e.updateSyntaxHighlighting()
			}
		}
	} else {
		// Optimize backspace in leading whitespace: delete one indent chunk
		if pos > 0 {
			line := e.CurrentBuffer().CursorIterator.Line()
			// Only apply when all characters before cursor are spaces/tabs
			onlyWhitespace := true
			if cursorX > len(line) {
				cursorX = len(line)
			}
			for i := 0; i < cursorX; i++ {
				if line[i] != ' ' && line[i] != '\t' {
					onlyWhitespace = false
					break
				}
			}

			// Try chunk deletion if in leading whitespace
			if onlyWhitespace && cursorX > 0 {
				var startX int
				var deletedText string

				if e.CurrentBuffer().Indentation.Type == config.IndentTabs {
					// Prefer removing a single tab if immediately before cursor
					if line[cursorX-1] == '\t' {
						startX = cursorX - 1
						deletedText = "\t"
					} else {
						// Remove up to 4 spaces (common tab width) if present
						spaceCount := 0
						for i := cursorX - 1; i >= 0 && line[i] == ' '; i-- {
							spaceCount++
							if spaceCount >= 4 {
								break
							}
						}
						if spaceCount > 0 {
							startX = cursorX - spaceCount
							deletedText = line[startX:cursorX]
						}
					}
				} else {
					// Spaces mode: delete one indent width of spaces if present before cursor
					indentWidth := e.CurrentBuffer().Indentation.Width
					if indentWidth <= 0 {
						indentWidth = 4
					}
					if cursorX >= indentWidth {
						okChunk := true
						for i := cursorX - indentWidth; i < cursorX; i++ {
							if line[i] != ' ' {
								okChunk = false
								break
							}
						}
						if okChunk {
							startX = cursorX - indentWidth
							deletedText = line[startX:cursorX]
						}
					}
				}

				if deletedText != "" {
					entry := buffer.NewDeleteTextEntry(lineNum, startX, deletedText)
					// Execute through history
					if e.CurrentBuffer().History != nil {
						if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
							return err
						} else if executed != nil {
							e.setLastChange(executed)
						}
					} else {
						// Fallback if history is not initialized
						if err := entry.Execute(e.CurrentBuffer()); err != nil {
							return err
						}
					}

					// Recreate iterators at the same line
					e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
					e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

					// Move cursor back by chunk size
					e.CurrentBuffer().CursorX = startX

					// Update syntax highlighting (skip during paste mode)
					if !e.inPasteMode {
						e.updateSyntaxHighlighting()
					}

					// Update completion menu filter if visible
					e.updateCompletionFilter()

					return nil
				}
			}

			// Default: delete a single character before cursor
			deletedChar := e.CurrentBuffer().Rope.ByteAt(pos - 1)
			deletedText := string(deletedChar)
			entry := buffer.NewDeleteTextEntry(lineNum, cursorX-1, deletedText)

			// Execute through history
			if e.CurrentBuffer().History != nil {
				if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
					return err
				} else if executed != nil {
					e.setLastChange(executed)
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.CurrentBuffer()); err != nil {
					return err
				}
			}

			// Recreate iterators at the same line
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
			e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

			// Move cursor back one position
			e.CurrentBuffer().CursorX = cursorX - 1

			// Update syntax highlighting (skip during paste mode)
			if !e.inPasteMode {
				e.updateSyntaxHighlighting()
			}

			// Update completion menu filter if visible
			e.updateCompletionFilter()
		}
	}

	return nil
}

// deleteCurrentCharacter deletes the character at the current cursor position
func (e *Editor) deleteCurrentCharacter(args []string) error {
	// If at EOF or empty buffer, nothing to delete
	if e.CurrentBuffer().CursorIterator.IsEOF() || e.CurrentBuffer().Rope.Length() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// Get current line to check bounds
	line := e.CurrentBuffer().CursorIterator.Line()

	// If cursor is past the end of the line, check if we can delete newline
	if e.CurrentBuffer().CursorX >= len(line) {
		// Check if there's a newline character at the end of this line
		pos := e.getAbsolutePosition()
		if pos < e.CurrentBuffer().Rope.Length() && e.CurrentBuffer().Rope.ByteAt(pos) == '\n' {
			// We can delete the newline
			deletedText := "\n"
			entry := buffer.NewDeleteTextEntry(lineNum, e.CurrentBuffer().CursorX, deletedText)

			// Execute through history
			if e.CurrentBuffer().History != nil {
				if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
					return err
				} else if executed != nil {
					e.setLastChange(executed)
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.CurrentBuffer()); err != nil {
					return err
				}
			}

			// Recreate iterators at the same position
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
			e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

			// Update syntax highlighting (skip during paste mode)
			if !e.inPasteMode {
				e.updateSyntaxHighlighting()
			}
			return nil
		}
		return nil
	}

	// Get cursor X position
	cursorX := e.CurrentBuffer().CursorX

	// Get the character being deleted
	pos := e.getAbsolutePosition()
	deletedChar := e.CurrentBuffer().Rope.ByteAt(pos)
	deletedText := string(deletedChar)

	// Create history entry
	entry := buffer.NewDeleteTextEntry(lineNum, cursorX, deletedText)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.CurrentBuffer()); err != nil {
			return err
		}
	}

	// Recreate iterators at the same line
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Cursor position stays the same, but we need to ensure it's not past the end of the line
	newLine := e.CurrentBuffer().CursorIterator.Line()
	if e.CurrentBuffer().CursorX > len(newLine) {
		if len(newLine) > 0 {
			e.CurrentBuffer().CursorX = len(newLine) - 1 // Stay on last character in normal mode
		} else {
			e.CurrentBuffer().CursorX = 0
		}
	}

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}

// deleteCharacterInsert deletes the character under the cursor in insert mode
func (e *Editor) deleteCharacterInsert(args []string) error {
	// If at EOF or empty buffer, nothing to delete
	if e.CurrentBuffer().CursorIterator.IsEOF() || e.CurrentBuffer().Rope.Length() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// Get current line to check bounds
	line := e.CurrentBuffer().CursorIterator.Line()

	// If cursor is at or past the end of the line, check if we can delete newline
	if e.CurrentBuffer().CursorX >= len(line) {
		// Check if there's a newline character at the end of this line
		pos := e.getAbsolutePosition()
		if pos < e.CurrentBuffer().Rope.Length() && e.CurrentBuffer().Rope.ByteAt(pos) == '\n' {
			// We can delete the newline
			deletedText := "\n"
			entry := buffer.NewDeleteTextEntry(lineNum, e.CurrentBuffer().CursorX, deletedText)

			// Execute through history
			if e.CurrentBuffer().History != nil {
				if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
					return err
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.CurrentBuffer()); err != nil {
					return err
				}
			}

			// Recreate iterators at the same position
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
			e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

			// Update syntax highlighting (skip during paste mode)
			if !e.inPasteMode {
				e.updateSyntaxHighlighting()
			}
			return nil
		}
		return nil
	}

	// Get the character being deleted
	pos := e.getAbsolutePosition()
	deletedChar := e.CurrentBuffer().Rope.ByteAt(pos)
	deletedText := string(deletedChar)

	// Create history entry
	entry := buffer.NewDeleteTextEntry(lineNum, e.CurrentBuffer().CursorX, deletedText)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.CurrentBuffer()); err != nil {
			return err
		}
	}

	// Recreate iterators at the same line
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// In insert mode, cursor position stays the same
	// But ensure it's not past the end of the line
	newLine := e.CurrentBuffer().CursorIterator.Line()
	if e.CurrentBuffer().CursorX > len(newLine) {
		e.CurrentBuffer().CursorX = len(newLine)
	}

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	// Update completion filter if menu is visible
	e.updateCompletionFilter()

	return nil
}

// deleteWordForward deletes from cursor to end of current word (Alt+Delete)
func (e *Editor) deleteWordForward(args []string) error {
	// If at EOF or empty buffer, nothing to delete
	if e.CurrentBuffer().CursorIterator.IsEOF() || e.CurrentBuffer().Rope.Length() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	startX := e.CurrentBuffer().CursorX

	// Find the next word boundary
	nextWord := FindNextWordBoundary(e.CurrentBuffer(), lineNum, startX)

	var endLine, endX int
	if nextWord != nil {
		endLine = nextWord.StartLine
		endX = nextWord.StartChar
	} else {
		// No next word found, delete to end of current line
		line := e.CurrentBuffer().CursorIterator.Line()
		endLine = lineNum
		endX = len(line)
	}

	// Calculate text to delete
	var textToDelete string
	if endLine == lineNum {
		// Same line deletion
		line := e.CurrentBuffer().CursorIterator.Line()
		if startX < len(line) && endX <= len(line) && endX > startX {
			textToDelete = line[startX:endX]
		}
	} else {
		// Multi-line deletion (including newline)
		line := e.CurrentBuffer().CursorIterator.Line()
		if startX < len(line) {
			textToDelete = line[startX:] + "\n"
		} else {
			textToDelete = "\n"
		}
	}

	if textToDelete == "" {
		return nil
	}

	// Create history entry
	entry := buffer.NewDeleteTextEntry(lineNum, startX, textToDelete)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.CurrentBuffer()); err != nil {
			return err
		}
	}

	// Recreate iterators at the same position
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	// Update completion filter if menu is visible
	e.updateCompletionFilter()

	return nil
}

// deleteToEndOfLine deletes from cursor position to end of line (D in vim)
func (e *Editor) deleteToEndOfLine(args []string) error {
	// If at EOF or empty buffer, nothing to delete
	if e.CurrentBuffer().CursorIterator.IsEOF() || e.CurrentBuffer().Rope.Length() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	startX := e.CurrentBuffer().CursorX

	// Get current line
	line := e.CurrentBuffer().CursorIterator.Line()

	// If cursor is already at or past end of line, nothing to delete
	if startX >= len(line) {
		return nil
	}

	// Calculate text to delete (from cursor to end of line, not including newline)
	deletedText := line[startX:]

	// Create history entry
	entry := buffer.NewDeleteTextEntry(lineNum, startX, deletedText)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	} else {
		// Fallback if history is not initialized
		pos := e.getAbsolutePosition()
		e.CurrentBuffer().Delete(pos, len(deletedText))
	}

	// Recreate iterators
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Adjust cursor position if needed
	line = e.CurrentBuffer().CursorIterator.Line()
	if e.CurrentBuffer().CursorX > 0 && e.CurrentBuffer().CursorX >= len(line) && len(line) > 0 {
		// In normal mode, cursor should be on last character if line is not empty
		e.CurrentBuffer().CursorX = len(line) - 1
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	// Note: Multi-cursor support will be handled automatically by the
	// command registry since we marked this command with appliestoAllCursors=true

	return nil
}

// deleteWordBackward deletes from beginning of current word to cursor (Alt+Backspace)
func (e *Editor) deleteWordBackward(args []string) error {
	// If at beginning of buffer, nothing to delete
	if e.CurrentBuffer().CursorIterator.LineNumber() == 0 && e.CurrentBuffer().CursorX == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	endX := e.CurrentBuffer().CursorX

	// Find the previous word boundary
	// We need to move back one character first, then find word start
	tempLine := lineNum
	tempX := endX - 1

	if tempX < 0 {
		// Move to previous line
		if lineNum > 0 {
			tempLine = lineNum - 1
			tempIter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), tempLine)
			tempX = len(tempIter.Line())
		} else {
			return nil
		}
	}

	// Find the start of the current/previous word
	var startLine, startX int

	// Use a temporary buffer position to find word start
	tempIter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), tempLine)
	currentLine := tempIter.Line()

	// If we're in a word, find the start of this word
	if tempX < len(currentLine) && isWordChar(rune(currentLine[tempX])) {
		// Find start of current word
		startX = tempX
		for startX > 0 && isWordChar(rune(currentLine[startX-1])) {
			startX--
		}
		startLine = tempLine
	} else {
		// We're not in a word, find the previous word
		// Skip non-word characters backward
		for tempX >= 0 && (tempX >= len(currentLine) || !isWordChar(rune(currentLine[tempX]))) {
			tempX--
			if tempX < 0 {
				// Need to go to previous line
				if tempLine > 0 {
					tempLine--
					tempIter = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), tempLine)
					currentLine = tempIter.Line()
					tempX = len(currentLine) - 1
				} else {
					// At beginning of buffer
					startLine = 0
					startX = 0
					break
				}
			}
		}

		if tempX >= 0 {
			// Now find the start of this word
			startX = tempX
			for startX > 0 && isWordChar(rune(currentLine[startX-1])) {
				startX--
			}
			startLine = tempLine
		}
	}

	// Calculate text to delete
	var textToDelete string
	var deleteLineNum, deleteX int

	if startLine == lineNum {
		// Same line deletion
		line := e.CurrentBuffer().CursorIterator.Line()
		if startX < endX && endX <= len(line) {
			textToDelete = line[startX:endX]
			deleteLineNum = lineNum
			deleteX = startX
		}
	} else {
		// Multi-line deletion
		// Delete from start position on previous line to cursor
		prevIter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), startLine)
		prevLine := prevIter.Line()

		textToDelete = prevLine[startX:] + "\n"
		if endX > 0 {
			line := e.CurrentBuffer().CursorIterator.Line()
			textToDelete += line[:endX]
		}
		deleteLineNum = startLine
		deleteX = startX
	}

	if textToDelete == "" {
		return nil
	}

	// Create history entry
	entry := buffer.NewDeleteTextEntry(deleteLineNum, deleteX, textToDelete)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.CurrentBuffer()); err != nil {
			return err
		}
	}

	// Update cursor position to where deletion started
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), startLine)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)
	e.CurrentBuffer().CursorX = startX

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	// Update completion filter if menu is visible
	e.updateCompletionFilter()

	return nil
}

// wordForwardInsert moves cursor forward by one word in insert mode
func (e *Editor) wordForwardInsert(args []string) error {
	// Use the same logic as normal mode
	return e.wordForward(args)
}

// wordBackwardInsert moves cursor backward by one word in insert mode
func (e *Editor) wordBackwardInsert(args []string) error {
	// Use the same logic as normal mode
	return e.wordBackward(args)
}

// reindentLine recalculates indentation of the current line based on context (vim '==')
func (e *Editor) reindentLine(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}
	// Preserve view position
	curLine := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLine := e.CurrentBuffer().ViewIterator.LineNumber()

	// Get current line text and its current indent info
	line := e.CurrentBuffer().CursorIterator.Line()
	indentInfo := e.CurrentBuffer().GetLineIndentInfo(line)

	// Compute desired indentation string using buffer's auto-indent context for empty line
	// We reuse calculateIndentForEmptyLine-like logic via Buffer.CalculateAutoIndent on an empty line context.
	// Use beginning of line as cursor column.
	auto := e.CurrentBuffer().CalculateAutoIndent("", 0)
	desired := auto.IndentStr

	// If desired equals current actual prefix, nothing to do
	if desired == indentInfo.ActualPrefix {
		return nil
	}

	// Build composite: replace current leading whitespace with desired
	// First, delete current prefix (if any), then insert desired (if any)
	composite := buffer.NewCompositeEntry("Reindent line")
	if indentInfo.ActualPrefix != "" {
		composite.Add(buffer.NewDeleteTextEntry(curLine, 0, indentInfo.ActualPrefix))
	}
	if desired != "" {
		composite.Add(buffer.NewInsertTextEntry(curLine, 0, desired))
	}

	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), composite); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
			// Treat as simple command for '.' repeat compatibility similar to indent/unindent
			e.lastSimpleCommand = "reindent-line"
		}
	} else {
		// Fallback direct edits without history
		lineStart := e.CurrentBuffer().CursorIterator.Position() - e.CurrentBuffer().CursorX
		if indentInfo.ActualPrefix != "" {
			e.CurrentBuffer().Delete(lineStart, len(indentInfo.ActualPrefix))
		}
		if desired != "" {
			e.CurrentBuffer().Insert(lineStart, desired)
		}
	}

	// Restore iterators
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), curLine)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLine)

	// Place cursor at first non-whitespace character
	newLine := e.CurrentBuffer().CursorIterator.Line()
	e.CurrentBuffer().CursorX = 0
	for i := 0; i < len(newLine); i++ {
		if newLine[i] != ' ' && newLine[i] != '\t' {
			e.CurrentBuffer().CursorX = i
			break
		}
	}

	e.updateSyntaxHighlighting()
	return nil
}

// reindentExpression recalculates indentation for the entire enclosing expression or block.
// It attempts to find a structural region (LSP folding block, then {..}, then (..)),
// and replaces leading whitespace on each line within that region using the editor's
// auto-indent heuristics. Repeated invocations at the same position will cycle the
// target width index (stored in lastRewrap* fields) to mirror Alt+Q behavior, so we
// can later extend line-breaking by rulers. For now, width cycling is recorded but
// does not change indentation.
func (e *Editor) reindentExpression(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}

	// Determine width cycling index similar to rewrapParagraph
	cursorLine := e.CurrentBuffer().CursorIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX
	width := 0
	if len(e.config.VerticalRulers) > 0 {
		if e.lastRewrapLine == cursorLine && e.lastRewrapChar == cursorX {
			e.lastRewrapIndex = (e.lastRewrapIndex + 1) % len(e.config.VerticalRulers)
		} else {
			e.lastRewrapIndex = 0
		}
		width = e.config.VerticalRulers[e.lastRewrapIndex]
	}
	_ = width // Reserved for future line-breaking based on rulers

	// Find expression range:
	startLine, _, endLine, _, ok := e.findExpressionRange()
	if !ok {
		// Fallback to current line only
		startLine, endLine = cursorLine, cursorLine
	}

	// Build a composite history edit
	composite := buffer.NewCompositeEntry("Reindent expression")

	// Preserve view position
	viewLine := e.CurrentBuffer().ViewIterator.LineNumber()

	// Helper: unit indentation string
	unit := e.CurrentBuffer().GetIndentationString()

	for ln := startLine; ln <= endLine; ln++ {
		iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), ln)
		if iter.IsEOF() {
			continue
		}
		line := iter.Line()
		// Place CursorIterator at this line to let CalculateAutoIndent analyze context
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), ln)
		// Desired indentation for an empty line at this position (block-level)
		auto := e.CurrentBuffer().CalculateAutoIndent("", 0)
		desired := auto.IndentStr

		// If this line begins with a closing brace, align to its matching open brace's indent
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) > 0 && trimmed[0] == '}' {
			if openLine, openIndent := e.findMatchingOpenBrace(ln); openLine >= 0 {
				desired = strings.Repeat(unit, openIndent)
			}
		}

		// Current leading whitespace
		i := 0
		for i < len(line) {
			ch := line[i]
			if ch == ' ' || ch == '\t' {
				i++
			} else {
				break
			}
		}
		current := line[:i]
		if current == desired {
			continue
		}
		// Apply: delete current, insert desired
		if current != "" {
			composite.Add(buffer.NewDeleteTextEntry(ln, 0, current))
		}
		if desired != "" {
			composite.Add(buffer.NewInsertTextEntry(ln, 0, desired))
		}
	}

	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), composite); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
			e.lastSimpleCommand = "reindent-line" // repeat-friendly; treat like simple reindent
		}
	} else {
		// Fallback: execute entries directly
		_ = composite.Execute(e.CurrentBuffer())
	}

	// Restore iterators
	if viewLine >= e.CurrentBuffer().LineCount {
		viewLine = e.CurrentBuffer().LineCount - 1
		if viewLine < 0 {
			viewLine = 0
		}
	}
	if cursorLine >= e.CurrentBuffer().LineCount {
		cursorLine = e.CurrentBuffer().LineCount - 1
		if cursorLine < 0 {
			cursorLine = 0
		}
	}
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLine)
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), cursorLine)

	// Move cursor to first non-whitespace of its line
	newLine := e.CurrentBuffer().CursorIterator.Line()
	e.CurrentBuffer().CursorX = 0
	for i := 0; i < len(newLine); i++ {
		if newLine[i] != ' ' && newLine[i] != '\t' {
			e.CurrentBuffer().CursorX = i
			break
		}
	}

	// Remember position for width cycling
	e.lastRewrapLine = cursorLine
	e.lastRewrapChar = e.CurrentBuffer().CursorX

	e.updateSyntaxHighlighting()
	return nil
}

// findExpressionRange tries to find a structural range around the cursor:
// 1) LSP folding block (around), 2) braces {..}, 3) parentheses (..).
func (e *Editor) findExpressionRange() (int, int, int, int, bool) {
	if e.CurrentBuffer() == nil {
		return 0, 0, 0, 0, false
	}
	line := e.CurrentBuffer().CursorIterator.LineNumber()
	x := e.CurrentBuffer().CursorX

	// Prefer LSP folding range if available
	if len(e.CurrentBuffer().FoldingRanges) > 0 {
		tb := NewBlockTextObject(ModifierAround)
		if sL, sC, eL, eC, ok := tb.GetRange(e.CurrentBuffer(), line, x); ok {
			return sL, sC, eL, eC, true
		}
	}

	// Try braces
	bt := NewBracketTextObject('{', ModifierAround)
	if sL, sC, eL, eC, ok := bt.GetRange(e.CurrentBuffer(), line, x); ok {
		return sL, sC, eL, eC, true
	}
	// Try parentheses
	pt := NewBracketTextObject('(', ModifierAround)
	if sL, sC, eL, eC, ok := pt.GetRange(e.CurrentBuffer(), line, x); ok {
		return sL, sC, eL, eC, true
	}
	return 0, 0, 0, 0, false
}

// indentLine indents the current line by one level (>> in vim)
func (e *Editor) indentLine(args []string) error {
	// Get current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// Get the indentation string for this buffer
	indentStr := e.CurrentBuffer().GetIndentationString()

	// Create history entry to insert indentation at beginning of line
	entry := buffer.NewInsertTextEntry(lineNum, 0, indentStr)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
			// Mark this as an indent command for proper repeat
			e.lastSimpleCommand = "indent-line"
		}
	} else {
		// Fallback if history is not initialized
		lineStart := e.CurrentBuffer().CursorIterator.Position() - e.CurrentBuffer().CursorX
		e.CurrentBuffer().Insert(lineStart, indentStr)
	}

	// Recreate iterators
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Move cursor to first non-whitespace character
	line := e.CurrentBuffer().CursorIterator.Line()
	foundNonWhitespace := false
	for i, ch := range line {
		if ch != ' ' && ch != '\t' {
			e.CurrentBuffer().CursorX = i
			foundNonWhitespace = true
			break
		}
	}
	// If line is all whitespace, leave cursor at current position
	if !foundNonWhitespace && len(line) > 0 {
		e.CurrentBuffer().CursorX = len(line)
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// unindentLine unindents the current line by one level (<< in vim)
func (e *Editor) unindentLine(args []string) error {
	// Get current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// Get current line
	line := e.CurrentBuffer().CursorIterator.Line()

	// Calculate how much indentation to remove
	indentInfo := e.CurrentBuffer().GetLineIndentInfo(line)
	if indentInfo.Tabs == 0 && indentInfo.Spaces == 0 {
		// No indentation to remove
		return nil
	}

	// Determine how much to remove based on buffer's indentation settings
	var toRemove int
	deletedText := ""

	if e.CurrentBuffer().Indentation.Type == config.IndentTabs {
		// Remove one tab if present at the beginning
		if len(line) > 0 && line[0] == '\t' {
			toRemove = 1
			deletedText = "\t"
		} else {
			// Count leading spaces only and remove up to 4
			spaceCount := 0
			for _, ch := range line {
				if ch == ' ' {
					spaceCount++
				} else {
					break
				}
				if spaceCount >= 4 {
					break
				}
			}
			toRemove = spaceCount
			if toRemove > 0 {
				deletedText = line[:toRemove]
			}
		}
	} else {
		// In spaces mode, remove up to indent width of leading spaces
		spaceCount := 0
		for _, ch := range line {
			if ch == ' ' {
				spaceCount++
			} else {
				break
			}
			if spaceCount >= e.CurrentBuffer().Indentation.Width {
				break
			}
		}
		toRemove = spaceCount
		if toRemove > 0 {
			deletedText = line[:toRemove]
		}
	}

	// Create history entry to delete indentation
	if toRemove > 0 && deletedText != "" {
		entry := buffer.NewDeleteTextEntry(lineNum, 0, deletedText)

		// Execute through history
		if e.CurrentBuffer().History != nil {
			if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
				return err
			} else if executed != nil {
				e.setLastChange(executed)
				// Mark this as an unindent command for proper repeat
				e.lastSimpleCommand = "unindent-line"
			}
		} else {
			// Fallback if history is not initialized
			lineStart := e.CurrentBuffer().CursorIterator.Position() - e.CurrentBuffer().CursorX
			e.CurrentBuffer().Delete(lineStart, toRemove)
		}
	}

	// Recreate iterators
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Move cursor to first non-whitespace character
	line = e.CurrentBuffer().CursorIterator.Line()
	e.CurrentBuffer().CursorX = 0 // Start at beginning
	for i, ch := range line {
		if ch != ' ' && ch != '\t' {
			e.CurrentBuffer().CursorX = i
			break
		}
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// shouldAutoDedentClosingBrace checks if we should auto-dedent when typing a closing brace
func (e *Editor) shouldAutoDedentClosingBrace(lineNum int, cursorX int) bool {
	// Get current line
	line := e.CurrentBuffer().CursorIterator.Line()

	// Check if line only contains whitespace before cursor
	for i := 0; i < cursorX && i < len(line); i++ {
		if line[i] != ' ' && line[i] != '\t' {
			return false // Non-whitespace before cursor, don't auto-dedent
		}
	}

	// Find the matching open brace by looking backwards
	openBraceLineNum, openBraceIndentLevel := e.findMatchingOpenBrace(lineNum)
	if openBraceLineNum == -1 {
		return false // No matching open brace found
	}

	// Get current line's indentation level
	currentIndentInfo := e.CurrentBuffer().GetLineIndentInfo(line)

	// We should dedent if current indentation is greater than the open brace line's indentation
	return currentIndentInfo.Level > openBraceIndentLevel
}

// findMatchingOpenBrace looks backwards for the matching open brace
func (e *Editor) findMatchingOpenBrace(startLine int) (lineNum int, indentLevel int) {
	braceCount := 1 // We're starting from a closing brace position

	// Search backwards from the previous line
	for lineNum := startLine - 1; lineNum >= 0; lineNum-- {
		iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
		if iter.IsEOF() {
			continue
		}

		line := iter.Line()

		// Count braces in this line (backwards)
		for i := len(line) - 1; i >= 0; i-- {
			if line[i] == '}' {
				braceCount++
			} else if line[i] == '{' {
				braceCount--
				if braceCount == 0 {
					// Found the matching open brace
					indentInfo := e.CurrentBuffer().GetLineIndentInfo(line)
					return lineNum, indentInfo.Level
				}
			}
		}
	}

	return -1, 0 // No matching brace found
}

// autoDedentCurrentLine removes one level of indentation from the current line
func (e *Editor) autoDedentCurrentLine(lineNum int) error {
	// Get current line
	line := e.CurrentBuffer().CursorIterator.Line()
	indentInfo := e.CurrentBuffer().GetLineIndentInfo(line)

	if indentInfo.Level == 0 {
		return nil // No indentation to remove
	}

	// First, check if the line only contains whitespace
	// If so, we need to clear the entire line's whitespace after dedenting
	isOnlyWhitespace := true
	for _, ch := range line {
		if ch != ' ' && ch != '\t' {
			isOnlyWhitespace = false
			break
		}
	}

	// Calculate how much indentation to remove (one level)
	var toRemove int
	var deletedText string

	if e.CurrentBuffer().Indentation.Type == config.IndentTabs {
		// Remove one tab if present
		if indentInfo.Tabs > 0 {
			toRemove = 1
			deletedText = "\t"
		}
	} else {
		// Remove one indent width of spaces
		if indentInfo.Spaces >= e.CurrentBuffer().Indentation.Width {
			toRemove = e.CurrentBuffer().Indentation.Width
			deletedText = strings.Repeat(" ", toRemove)
		}
	}

	if toRemove == 0 || deletedText == "" {
		return nil // Nothing to remove
	}

	// If the line only contains whitespace, we need to delete ALL the whitespace
	// and replace it with the dedented amount
	if isOnlyWhitespace && len(line) > toRemove {
		// Delete all whitespace
		allWhitespace := line
		entry := buffer.NewDeleteTextEntry(lineNum, 0, allWhitespace)

		// Execute through history
		if e.CurrentBuffer().History != nil {
			if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
				return err
			}
		} else {
			lineStart := e.CurrentBuffer().CursorIterator.Position() - e.CurrentBuffer().CursorX
			e.CurrentBuffer().Delete(lineStart, len(allWhitespace))
		}

		// Now insert the dedented whitespace
		dedentedWhitespace := line[toRemove:]
		if len(dedentedWhitespace) > 0 {
			insertEntry := buffer.NewInsertTextEntry(lineNum, 0, dedentedWhitespace)
			if e.CurrentBuffer().History != nil {
				if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), insertEntry); err != nil {
					return err
				}
			} else {
				lineStart := e.CurrentBuffer().CursorIterator.Position() - e.CurrentBuffer().CursorX
				e.CurrentBuffer().Insert(lineStart, dedentedWhitespace)
			}
		}

		// Set cursor to end of dedented whitespace (where brace will go)
		e.CurrentBuffer().CursorX = len(dedentedWhitespace)
	} else {
		// Normal case: just remove indentation from beginning
		entry := buffer.NewDeleteTextEntry(lineNum, 0, deletedText)

		// Execute through history
		if e.CurrentBuffer().History != nil {
			if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
				return err
			}
		} else {
			lineStart := e.CurrentBuffer().CursorIterator.Position() - e.CurrentBuffer().CursorX
			e.CurrentBuffer().Delete(lineStart, toRemove)
		}

		// Update cursor position after dedent
		if e.CurrentBuffer().CursorX >= toRemove {
			e.CurrentBuffer().CursorX -= toRemove
		} else {
			e.CurrentBuffer().CursorX = 0
		}
	}

	return nil
}
