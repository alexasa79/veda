package editor

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/indent"
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
	var startedSession bool
	if e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.StartInsertSession("o")
		startedSession = true
	}
	textToInsert := "\n" + autoIndent.IndentStr
	entry := buffer.NewInsertTextEntry(lineNum, e.CurrentBuffer().CursorX, textToInsert)

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			if startedSession {
				e.CurrentBuffer().History.EndInsertSession()
			}
			return err
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

	// Enter insert mode
	e.switchToInsertMode()

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	// Refine indentation using analyzer if available
	e.adjustIndentationForLines([]int{lineNum + 1}, lineNum+1)

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

	var startedSession bool
	if e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.StartInsertSession("O")
		startedSession = true
	}

	// Create history entry for the newline
	entry := buffer.NewInsertTextEntry(lineNum, 0, "\n")

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			if startedSession {
				e.CurrentBuffer().History.EndInsertSession()
			}
			return err
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
			if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), indentEntry); err != nil {
				if startedSession {
					e.CurrentBuffer().History.EndInsertSession()
				}
				return err
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

	// Enter insert mode
	e.switchToInsertMode()

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	// Refine indentation using analyzer if available
	e.adjustIndentationForLines([]int{lineNum}, lineNum)

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
	// Close completion menu on explicit Tab in insert mode
	e.hideCompletionMenu()
	// Get the buffer's indentation string
	indentStr := e.CurrentBuffer().GetIndentationString()

	// Remember current position
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX

	// Compute absolute start before modification for incremental syntax update
	startByte := e.getAbsolutePosition()

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
		e.updateSyntaxApplyEdit(startByte, "", indentStr)
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

	// Electric opening brace in continuation-indented lines:
	// If user types '{' at the first non-whitespace on a line that appears
	// to be continuation-indented (alignment under '('), adjust indentation
	// back to the enclosing block level (L1 relative to base) before inserting '{'.
	if !e.inPasteMode && ch == '{' && e.shouldAdjustIndentForOpeningBrace(lineNum, cursorX) {
		if err := e.adjustIndentForOpeningBrace(lineNum); err != nil {
			return err
		}
		// Update cursor position after adjustment
		cursorX = e.CurrentBuffer().CursorX
		// Recreate iterators after buffer modification
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)
	}

	// Auto-dedent numbered list items in markdown files
	if !e.inPasteMode && ch == '.' && e.shouldAutoDedentMarkdownListNumber(lineNum, cursorX) {
		if err := e.autoDedentMarkdownListNumber(lineNum); err != nil {
			return err
		}
		cursorX = e.CurrentBuffer().CursorX
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)
	}

	// Create history entry for the single character
	// Compute absolute start before modification for incremental syntax update
	startByte := e.getAbsolutePosition()
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

	// Move cursor forward by the byte length of the inserted rune
	e.CurrentBuffer().CursorX = cursorX + len(textToInsert)

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxApplyEdit(startByte, "", textToInsert)
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
	if !e.inPasteMode && e.mode == commands.InsertMode && e.CurrentBuffer() != nil && e.hasLSP() {
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

	// Compute absolute start before modification for incremental syntax update
	startByte := e.getAbsolutePosition()

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

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxApplyEdit(startByte, "", textToInsert)
	}

	// Refine indentation using analyzer if available
	linesToAdjust := []int{lineNum + 1}
	if isBetweenBraces {
		linesToAdjust = append(linesToAdjust, lineNum+2)
	}
	e.adjustIndentationForLines(linesToAdjust, lineNum+1)

	finalIndentStr := autoIndent.IndentStr
	if buf := e.CurrentBuffer(); buf != nil {
		if buf.LineCount > 0 {
			if lineNum+1 >= buf.LineCount {
				buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, buf.LineCount-1)
			} else {
				buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, lineNum+1)
			}
			lineText := buf.CursorIterator.Line()
			if finalCursorX > len(lineText) {
				finalCursorX = len(lineText)
			}
			buf.CursorX = finalCursorX
			if finalCursorX > 0 {
				finalIndentStr = lineText[:finalCursorX]
			} else {
				finalIndentStr = ""
			}
		}
	}

	// Ensure the cursor's new line is visible immediately to avoid one-frame misplacement
	e.scroll()

	LL.Debug("insertNewline: end -> line=%d, x=%d, viewTop=%d (finalX=%d, betweenBraces=%v, indent=%q)",
		e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX, e.CurrentBuffer().ViewIterator.LineNumber(), finalCursorX, isBetweenBraces, finalIndentStr)

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
			// For incremental syntax: start at pos-1, delete "\n"
			incStart := pos - 1
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
				e.updateSyntaxApplyEdit(incStart, deletedText, "")
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
						// Compute absolute start for deletion using line start + startX BEFORE modification
						iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
						absLineStart := iter.Position()
						incStart := absLineStart + startX
						e.updateSyntaxApplyEdit(incStart, deletedText, "")
					}

					// Update completion menu filter if visible
					e.updateCompletionFilter()

					return nil
				}
			}

			// Default: delete a single Unicode rune before cursor
			// Determine previous rune boundaries within the current line
			linePrefix := line[:cursorX]
			_, size := utf8.DecodeLastRuneInString(linePrefix)
			if size <= 0 {
				size = 1
			}
			startX := cursorX - size
			if startX < 0 {
				startX = 0
			}
			deletedText := line[startX:cursorX]
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

			// Move cursor back by the size of the deleted rune
			e.CurrentBuffer().CursorX = startX

			// Update syntax highlighting (skip during paste mode)
			if !e.inPasteMode {
				e.updateSyntaxApplyEdit(pos-len(deletedText), deletedText, "")
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
				e.updateSyntaxApplyEdit(pos, "\n", "")
			}
			return nil
		}
		return nil
	}

	// Get cursor X position
	cursorX := e.CurrentBuffer().CursorX

	// Compute UTF-8 rune at cursor and delete the full rune slice
	pos := e.getAbsolutePosition()
	// Safeguard bounds
	if cursorX < 0 {
		cursorX = 0
	}
	if cursorX > len(line) {
		cursorX = len(line)
	}
	_, size := utf8.DecodeRuneInString(line[cursorX:])
	if size <= 0 {
		size = 1
	}
	if cursorX+size > len(line) {
		size = len(line) - cursorX
		if size < 0 {
			size = 0
		}
	}
	deletedText := line[cursorX : cursorX+size]

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
		e.updateSyntaxApplyEdit(pos, deletedText, "")
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

	// Get the UTF-8 rune under the cursor and delete the full rune slice
	cursorX := e.CurrentBuffer().CursorX
	if cursorX < 0 {
		cursorX = 0
	}
	if cursorX > len(line) {
		cursorX = len(line)
	}
	_, size := utf8.DecodeRuneInString(line[cursorX:])
	if size <= 0 {
		size = 1
	}
	if cursorX+size > len(line) {
		size = len(line) - cursorX
		if size < 0 {
			size = 0
		}
	}
	deletedText := line[cursorX : cursorX+size]

	// Create history entry
	entry := buffer.NewDeleteTextEntry(lineNum, cursorX, deletedText)

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
		// Compute absolute start using iterator based on original startX
		iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
		absStart := iter.Position() + startX
		e.updateSyntaxApplyEdit(absStart, textToDelete, "")
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
	if e.mode == commands.NormalMode {
		if e.CurrentBuffer().CursorX > 0 && e.CurrentBuffer().CursorX >= len(line) && len(line) > 0 {
			// In normal mode, cursor should be on last character if line is not empty
			e.CurrentBuffer().CursorX = len(line) - 1
		}
	} else {
		// In insert mode, keep cursor at original column if possible
		if startX <= len(line) {
			e.CurrentBuffer().CursorX = startX
		} else {
			e.CurrentBuffer().CursorX = len(line)
		}
	}

	// Update syntax highlighting
	iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	absStart := iter.Position() + startX
	e.updateSyntaxApplyEdit(absStart, deletedText, "")

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

// reindentExpression recalculates indentation for the current buffer using the
// experimental indentation engine. Unsupported file types are left untouched.
func (e *Editor) reindentExpression(args []string) error {
	buf := e.CurrentBuffer()
	if buf == nil {
		return nil
	}

	lang := detectIndentLanguage(buf.FilePath)
	if lang == nil {
		ext := strings.ToLower(filepath.Ext(buf.FilePath))
		if ext == "" {
			e.ShowInfo("reindent-expression is not yet supported for this file type")
		} else {
			e.ShowInfo("reindent-expression is not yet supported for %s files", ext)
		}
		return nil
	}

	text := buf.Rope.Extract(0, buf.Rope.Length())
	tokens, err := lang.Tokenize([]rune(text))
	if err != nil {
		return err
	}

	annotated, err := indent.NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err != nil {
		return err
	}

	indentPerLine, continuationPerLine := buildIndentMaps(annotated)

	if len(indentPerLine) == 0 {
		return nil
	}

	cursorLine := buf.CursorIterator.LineNumber()
	cursorX := buf.CursorX
	ranges := lang.ExpressionRanges(tokens, cursorLine, cursorX)
	if len(ranges) == 0 {
		ranges = []indent.Range{{StartLine: cursorLine, EndLine: cursorLine}}
	}

	lineSet := make(map[int]struct{})
	for _, r := range ranges {
		start := r.StartLine
		if start < 0 {
			start = 0
		}
		if start >= buf.LineCount {
			start = buf.LineCount - 1
		}
		end := r.EndLine
		if end >= buf.LineCount {
			end = buf.LineCount - 1
		}
		if end < start {
			end = start
		}
		for line := start; line <= end; line++ {
			lineSet[line] = struct{}{}
		}
	}
	if len(lineSet) == 0 {
		lineSet[cursorLine] = struct{}{}
	}

	lines := make([]int, 0, len(lineSet))
	for line := range lineSet {
		if line < 0 || line >= buf.LineCount {
			continue
		}
		lines = append(lines, line)
	}
	sort.Ints(lines)

	contColumns := computeContinuationColumns(tokens)

	viewLine := buf.ViewIterator.LineNumber()

	changes := buffer.NewCompositeEntry("Reindent expression")
	modified := false

	for _, line := range lines {
		iter := buffer.NewBufferIteratorAtLine(buf, line)
		if iter.IsEOF() {
			continue
		}
		lineText := iter.Line()
		if strings.TrimSpace(lineText) == "" {
			continue
		}
		level, ok := indentPerLine[line]
		if !ok {
			continue
		}
		cont := continuationPerLine[line]
		effectiveLevel := level + cont
		baseColumns := targetColumnsForLevel(buf, effectiveLevel)
		extraColumns := 0
		if contCol, hasContCol := contColumns[line]; hasContCol && contCol > baseColumns {
			extraColumns = contCol - baseColumns
		}
		desired := indentStringForLevel(buf, effectiveLevel, extraColumns)
		info := buf.GetLineIndentInfo(lineText)
		trimmed := strings.TrimLeft(lineText, " \t")
		if effectiveLevel == 0 && strings.HasPrefix(trimmed, "//") {
			continue
		}
		if info.ActualPrefix == desired {
			continue
		}
		if info.ActualPrefix != "" {
			changes.Add(buffer.NewDeleteTextEntry(line, 0, info.ActualPrefix))
		}
		if desired != "" {
			changes.Add(buffer.NewInsertTextEntry(line, 0, desired))
		}
		modified = true
	}

	if !modified {
		return nil
	}

	if buf.History != nil {
		executed, err := buf.History.Execute(buf, changes)
		if err != nil {
			return err
		}
		if executed != nil {
			e.setLastChange(executed)
			e.lastSimpleCommand = "reindent-expression"
		}
	} else {
		_ = changes.Execute(buf)
	}

	if buf.LineCount == 0 {
		e.updateSyntaxHighlighting()
		return nil
	}

	if cursorLine >= buf.LineCount {
		cursorLine = buf.LineCount - 1
	}
	if cursorLine < 0 {
		cursorLine = 0
	}
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, cursorLine)
	lineText := buf.CursorIterator.Line()
	if cursorX > len(lineText) {
		cursorX = len(lineText)
	}
	buf.CursorX = cursorX

	if viewLine >= buf.LineCount {
		viewLine = buf.LineCount - 1
		if viewLine < 0 {
			viewLine = 0
		}
	}
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, viewLine)

	e.updateSyntaxHighlighting()
	return nil
}

func computeContinuationColumns(tokens []indent.Token) map[int]int {
	columns := make(map[int]int)
	type entry struct {
		openLine int
		column   int
	}
	var stack []entry
	for _, tok := range tokens {
		switch tok.Kind {
		case indent.TokenOpenParen, indent.TokenOpenBracket:
			stack = append(stack, entry{openLine: tok.Start.Line, column: -1})
		case indent.TokenCloseParen, indent.TokenCloseBracket:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case indent.TokenWhitespace, indent.TokenNewline, indent.TokenCommentEnd, indent.TokenLineCommentStart:
			// ignore
		default:
			if len(stack) == 0 {
				continue
			}
			top := &stack[len(stack)-1]
			if top.column < 0 {
				top.column = tok.Start.Column
			}
			if top.column >= 0 && tok.Start.Line != top.openLine {
				if existing, ok := columns[tok.Start.Line]; !ok || existing < top.column {
					columns[tok.Start.Line] = top.column
				}
			}
		}
	}
	return columns
}

func buildIndentMaps(annotated []indent.AnnotatedToken) (map[int]int, map[int]int) {
	indentPerLine := make(map[int]int)
	continuationPerLine := make(map[int]int)
	for _, at := range annotated {
		switch at.Token.Kind {
		case indent.TokenWhitespace, indent.TokenNewline, indent.TokenIndent, indent.TokenDedent:
			continue
		}
		if _, ok := indentPerLine[at.Token.Start.Line]; !ok {
			indentPerLine[at.Token.Start.Line] = at.IndentLevel
			continuationPerLine[at.Token.Start.Line] = at.ContinuationLevel
		}
	}
	return indentPerLine, continuationPerLine
}

func (e *Editor) computeIndentAdjustments(lines []int) (map[int]string, map[int]string, error) {
	buf := e.CurrentBuffer()
	if buf == nil {
		return nil, nil, nil
	}
	if len(lines) == 0 {
		return nil, nil, nil
	}
	lang := detectIndentLanguage(buf.FilePath)
	if lang == nil {
		return nil, nil, nil
	}
	unique := make(map[int]struct{}, len(lines))
	for _, line := range lines {
		unique[line] = struct{}{}
	}
	if len(unique) == 0 {
		return nil, nil, nil
	}
	sortedLines := make([]int, 0, len(unique))
	for line := range unique {
		sortedLines = append(sortedLines, line)
	}
	sort.Ints(sortedLines)

	text := buf.Rope.Extract(0, buf.Rope.Length())
	tokens, err := lang.Tokenize([]rune(text))
	if err != nil {
		return nil, nil, err
	}
	annotated, err := indent.NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err != nil {
		return nil, nil, err
	}
	indentPerLine, continuationPerLine := buildIndentMaps(annotated)
	contColumns := computeContinuationColumns(tokens)

	desired := make(map[int]string)
	current := make(map[int]string)
	for _, line := range sortedLines {
		if line < 0 || line >= buf.LineCount {
			continue
		}
		iter := buffer.NewBufferIteratorAtLine(buf, line)
		if iter.IsEOF() {
			continue
		}
		lineText := iter.Line()
		if strings.TrimSpace(lineText) == "" {
			continue
		}
		level, ok := indentPerLine[line]
		if !ok {
			continue
		}
		cont := continuationPerLine[line]
		effectiveLevel := level + cont
		baseColumns := targetColumnsForLevel(buf, effectiveLevel)
		extraColumns := 0
		if col, ok := contColumns[line]; ok && col > baseColumns {
			extraColumns = col - baseColumns
		}
		desiredStr := indentStringForLevel(buf, effectiveLevel, extraColumns)
		info := buf.GetLineIndentInfo(lineText)
		trimmed := strings.TrimLeft(lineText, " \t")
		if effectiveLevel == 0 && strings.HasPrefix(trimmed, "//") {
			continue
		}
		if info.ActualPrefix == desiredStr {
			continue
		}
		desired[line] = desiredStr
		current[line] = info.ActualPrefix
	}

	if len(desired) == 0 {
		return nil, nil, nil
	}
	return desired, current, nil
}

func (e *Editor) applyIndentAdjustments(desired, current map[int]string) error {
	if len(desired) == 0 {
		return nil
	}
	buf := e.CurrentBuffer()
	if buf == nil {
		return nil
	}
	lines := make([]int, 0, len(desired))
	for line := range desired {
		lines = append(lines, line)
	}
	sort.Ints(lines)

	changes := buffer.NewCompositeEntry("Auto indentation adjustment")
	modified := false
	for _, line := range lines {
		if cur, ok := current[line]; ok && cur != "" {
			changes.Add(buffer.NewDeleteTextEntry(line, 0, cur))
			modified = true
		}
		des := desired[line]
		if des != "" {
			changes.Add(buffer.NewInsertTextEntry(line, 0, des))
			modified = true
		}
	}
	if !modified {
		return nil
	}

	cursorLine := buf.CursorIterator.LineNumber()
	cursorX := buf.CursorX
	viewLine := buf.ViewIterator.LineNumber()

	if buf.History != nil {
		if _, err := buf.History.Execute(buf, changes); err != nil {
			return err
		}
	} else {
		if err := changes.Execute(buf); err != nil {
			return err
		}
	}

	if buf.LineCount == 0 {
		buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 0)
		buf.CursorX = 0
		buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, 0)
		e.updateSyntaxHighlighting()
		return nil
	}

	if cursorLine >= buf.LineCount {
		cursorLine = buf.LineCount - 1
	}
	if cursorLine < 0 {
		cursorLine = 0
	}
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, cursorLine)
	lineText := buf.CursorIterator.Line()
	if cursorX > len(lineText) {
		cursorX = len(lineText)
	}
	buf.CursorX = cursorX

	if viewLine >= buf.LineCount {
		viewLine = buf.LineCount - 1
	}
	if viewLine < 0 {
		viewLine = 0
	}
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, viewLine)

	e.updateSyntaxHighlighting()
	return nil
}

func (e *Editor) adjustIndentationForLines(lines []int, focusLine int) {
	desired, current, err := e.computeIndentAdjustments(lines)
	if err != nil {
		LL.Warning("indent analyzer failed: %v", err)
		return
	}
	if len(desired) == 0 {
		return
	}
	if err := e.applyIndentAdjustments(desired, current); err != nil {
		LL.Warning("indent adjustment apply failed: %v", err)
		return
	}
	buf := e.CurrentBuffer()
	if buf == nil {
		return
	}
	if focusLine < 0 {
		return
	}
	if buf.LineCount == 0 {
		buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 0)
		buf.CursorX = 0
		return
	}
	if focusLine >= buf.LineCount {
		focusLine = buf.LineCount - 1
	}
	if focusLine < 0 {
		focusLine = 0
	}
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, focusLine)
	lineText := buf.CursorIterator.Line()
	indentInfo := buf.GetLineIndentInfo(lineText)
	buf.CursorX = len(indentInfo.ActualPrefix)
}

func findReindentRange(tokens []indent.Token, cursorLine, cursorCol int) (int, int) {
	bestOpen := -1
	bestClose := -1
	bestSpan := 1 << 30
	type frame struct {
		kind indent.TokenKind
		idx  int
	}
	var stack []frame
	for idx, tok := range tokens {
		switch tok.Kind {
		case indent.TokenOpenBrace, indent.TokenOpenParen, indent.TokenOpenBracket:
			stack = append(stack, frame{kind: tok.Kind, idx: idx})
		case indent.TokenCloseBrace, indent.TokenCloseParen, indent.TokenCloseBracket:
			if len(stack) == 0 {
				continue
			}
			frame := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			op := tokens[frame.idx]
			closeTok := tok
			if positionWithinRange(cursorLine, cursorCol, op.Start, closeTok.End) {
				span := closeTok.Start.Line - op.Start.Line
				if span < 0 {
					span = 0
				}
				if bestOpen == -1 || span < bestSpan || (span == bestSpan && prefersKind(frame.kind, tokens[bestOpen].Kind)) {
					bestOpen = frame.idx
					bestClose = idx
					bestSpan = span
				}
			}
		}
	}
	if bestOpen != -1 {
		start := tokens[bestOpen].Start.Line
		end := tokens[bestClose].Start.Line
		return start, end
	}

	lastSemi := -1
	nextSemi := -1
	for _, tok := range tokens {
		if tok.Kind != indent.TokenSemicolon {
			continue
		}
		if positionLessEqual(tok.End, cursorLine, cursorCol) {
			lastSemi = tok.Start.Line
			continue
		}
		if positionLess(cursorLine, cursorCol, tok.Start) {
			nextSemi = tok.Start.Line
			break
		}
	}
	start := cursorLine
	if lastSemi >= 0 {
		start = lastSemi
		if start < cursorLine {
			start++
		}
	}
	end := cursorLine
	if nextSemi >= 0 {
		end = nextSemi
	}
	return start, end
}

func prefersKind(candidate, current indent.TokenKind) bool {
	if candidate == indent.TokenOpenBrace && current != indent.TokenOpenBrace {
		return true
	}
	return false
}

func positionWithinRange(cursorLine, cursorCol int, startLoc, endLoc indent.Location) bool {
	if positionLess(cursorLine, cursorCol, startLoc) {
		return false
	}
	if !positionLess(cursorLine, cursorCol, endLoc) && !positionEqual(cursorLine, cursorCol, endLoc) {
		return false
	}
	return true
}

func positionLess(line1, col1 int, loc indent.Location) bool {
	if line1 != loc.Line {
		return line1 < loc.Line
	}
	return col1 < loc.Column
}

func positionLessEqual(loc indent.Location, line1, col1 int) bool {
	if loc.Line != line1 {
		return loc.Line < line1
	}
	return loc.Column <= col1
}

func positionLess2(line1, col1 int, loc indent.Location) bool {
	if line1 != loc.Line {
		return line1 < loc.Line
	}
	return col1 < loc.Column
}

func positionEqual(line1, col1 int, loc indent.Location) bool {
	return line1 == loc.Line && col1 == loc.Column
}

func targetColumnsForLevel(buf *buffer.Buffer, level int) int {
	if level <= 0 {
		return 0
	}
	if buf.Indentation.Type == config.IndentTabs {
		return level * 8
	}
	width := buf.Indentation.Width
	if width <= 0 {
		width = 4
	}
	return level * width
}

func indentStringForLevel(buf *buffer.Buffer, level int, extraColumns int) string {
	if level < 0 {
		level = 0
	}
	if extraColumns < 0 {
		extraColumns = 0
	}
	if buf.Indentation.Type == config.IndentTabs {
		totalColumns := level*8 + extraColumns
		if totalColumns <= 0 {
			return ""
		}
		tabs := totalColumns / 8
		spaces := totalColumns % 8
		return strings.Repeat("\t", tabs) + strings.Repeat(" ", spaces)
	}
	width := buf.Indentation.Width
	if width <= 0 {
		width = 4
	}
	totalColumns := level*width + extraColumns
	if totalColumns <= 0 {
		return ""
	}
	return strings.Repeat(" ", totalColumns)
}

func detectIndentLanguage(path string) indent.Language {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".c", ".h", ".m", ".cc", ".cpp", ".cxx", ".hpp", ".hh":
		return indent.NewCLanguage()
	case ".py":
		return indent.NewPythonLanguage()
	case ".go":
		return indent.NewGoLanguage()
	default:
		return nil
	}
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

// shouldAdjustIndentForOpeningBrace returns true if we should auto-adjust indentation
// when typing an opening brace '{'. Heuristic:
//   - Only whitespace before cursor on the current line (typing brace at BOL)
//   - The line appears to be continuation-indented (alignment under '(') or otherwise
//     has non-standard indentation (LineIndentInfo.IsContinuation)
func (e *Editor) shouldAdjustIndentForOpeningBrace(lineNum int, cursorX int) bool {
	// Get current line
	line := e.CurrentBuffer().CursorIterator.Line()

	// Ensure only whitespace before cursor (typing brace at BOL visually)
	for i := 0; i < cursorX && i < len(line); i++ {
		if line[i] != ' ' && line[i] != '\t' {
			return false
		}
	}

	// Compare current indent level to the nearest less-indented line above
	curInfo := e.CurrentBuffer().GetLineIndentInfo(line)
	baseLevel := 0
	nearestNonEmptyTrimmed := ""
	for ln := lineNum - 1; ln >= 0; ln-- {
		iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), ln)
		if iter.IsEOF() {
			continue
		}
		raw := iter.Line()
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if nearestNonEmptyTrimmed == "" {
			nearestNonEmptyTrimmed = strings.TrimSpace(raw)
		}
		ii := e.CurrentBuffer().GetLineIndentInfo(raw)
		if ii.Level < curInfo.Level {
			baseLevel = ii.Level
			break
		}
	}

	// Only trigger when previous non-empty line likely ends a header/condition
	// e.g., ends with ')', to avoid interfering with nested blocks inside bodies.
	if nearestNonEmptyTrimmed == "" || !strings.HasSuffix(nearestNonEmptyTrimmed, ")") {
		return false
	}

	return curInfo.Level > baseLevel
}

// adjustIndentForOpeningBrace replaces leading whitespace of the current line so that
// an opening brace '{' will be at the block level (L1 relative to the nearest
// non-continuation structural line above).
func (e *Editor) adjustIndentForOpeningBrace(lineNum int) error {
	// Preserve view position
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// Determine target indent level.
	// Strategy:
	// 1) Prefer the indent of the nearest control header line above (if/for/while/switch).
	// 2) Otherwise, use the nearest non-empty line above with strictly less indent than current.
	baseLevel := 0
	// Current line's indentation level
	curLine := e.CurrentBuffer().CursorIterator.Line()
	curInfo := e.CurrentBuffer().GetLineIndentInfo(curLine)
	foundControl := false
	firstLessSet := false
	firstLess := 0
	for ln := lineNum - 1; ln >= 0; ln-- {
		iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), ln)
		if iter.IsEOF() {
			continue
		}
		raw := iter.Line()
		if strings.TrimSpace(raw) == "" {
			continue
		}
		ii := e.CurrentBuffer().GetLineIndentInfo(raw)
		trimmedLower := strings.ToLower(strings.TrimSpace(raw))
		// Detect common control headers
		if strings.HasPrefix(trimmedLower, "if ") || strings.HasPrefix(trimmedLower, "if(") ||
			strings.HasPrefix(trimmedLower, "for ") || strings.HasPrefix(trimmedLower, "for(") ||
			strings.HasPrefix(trimmedLower, "while ") || strings.HasPrefix(trimmedLower, "while(") ||
			strings.HasPrefix(trimmedLower, "switch ") || strings.HasPrefix(trimmedLower, "switch(") {
			baseLevel = ii.Level
			foundControl = true
			break
		}
		// Remember the first less-indented line as fallback
		if !firstLessSet && ii.Level < curInfo.Level {
			firstLessSet = true
			firstLess = ii.Level
			// Don't break; we might still find a control header above
		}
	}
	if !foundControl && firstLessSet {
		baseLevel = firstLess
	}
	// Opening brace should align to the base level (same as the control line)
	targetLevel := baseLevel
	desired := strings.Repeat(e.CurrentBuffer().GetIndentationString(), targetLevel)

	// Current line's indentation
	line := e.CurrentBuffer().CursorIterator.Line()
	i := 0
	for i < len(line) {
		if line[i] == ' ' || line[i] == '\t' {
			i++
		} else {
			break
		}
	}
	current := line[:i]
	if current == desired {
		return nil
	}

	// Build composite history edit: replace current leading whitespace with desired
	composite := buffer.NewCompositeEntry("Adjust indent for opening brace")
	if current != "" {
		composite.Add(buffer.NewDeleteTextEntry(lineNum, 0, current))
	}
	if desired != "" {
		composite.Add(buffer.NewInsertTextEntry(lineNum, 0, desired))
	}
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), composite); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
			e.lastSimpleCommand = "reindent-expression" // treat like reindent for '.' repeat
		}
	} else {
		// Fallback direct edit
		lineStart := e.CurrentBuffer().CursorIterator.Position() - e.CurrentBuffer().CursorX
		if current != "" {
			e.CurrentBuffer().Delete(lineStart, len(current))
		}
		if desired != "" {
			e.CurrentBuffer().Insert(lineStart, desired)
		}
	}

	// Restore iterators and move cursor to end of the desired indentation
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)
	e.CurrentBuffer().CursorX = len(desired)

	// Update highlighting
	e.updateSyntaxHighlighting()
	return nil
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

// isMarkdownFile returns true if the current buffer is a markdown file.
func (e *Editor) isMarkdownFile() bool {
	ext := strings.ToLower(filepath.Ext(e.CurrentBuffer().FilePath))
	return ext == ".md" || ext == ".markdown"
}

// shouldAutoDedentMarkdownListNumber checks if typing '.' would complete a
// numbered list marker that should be auto-dedented to match the previous item.
// Called BEFORE '.' is inserted.
func (e *Editor) shouldAutoDedentMarkdownListNumber(lineNum int, cursorX int) bool {
	if !e.isMarkdownFile() {
		return false
	}

	line := e.CurrentBuffer().CursorIterator.Line()
	prefix := line[:cursorX]

	// Line up to cursor must be whitespace then digits (e.g. "   3")
	digitStart := -1
	for i := 0; i < len(prefix); i++ {
		if prefix[i] == ' ' || prefix[i] == '\t' {
			if digitStart >= 0 {
				return false // whitespace after digits
			}
			continue
		}
		if prefix[i] >= '0' && prefix[i] <= '9' {
			if digitStart < 0 {
				digitStart = i
			}
			continue
		}
		return false // non-whitespace, non-digit
	}
	if digitStart < 0 {
		return false // no digits found
	}

	currentIndent := prefix[:digitStart]
	currentNum, err := strconv.Atoi(prefix[digitStart:])
	if err != nil {
		return false
	}

	// Nothing should follow the cursor on this line (only whitespace allowed)
	for i := cursorX; i < len(line); i++ {
		if line[i] != ' ' && line[i] != '\t' {
			return false
		}
	}

	// Scan backward for the previous numbered list item
	prevNum, prevIndent, found := e.findPrevMarkdownListItem(lineNum)
	if !found {
		return false
	}

	// Current number must be previous + 1
	if currentNum != prevNum+1 {
		return false
	}

	// Current line must have more indent than previous item
	if len(currentIndent) <= len(prevIndent) {
		return false
	}

	return true
}

// findPrevMarkdownListItem scans backward from lineNum to find the previous
// numbered list item (a line matching \s*\d+\.\s). It skips blank lines and
// continuation lines (non-list-item text that belongs to a previous item).
// Returns the item number, its leading whitespace, and whether one was found.
func (e *Editor) findPrevMarkdownListItem(lineNum int) (int, string, bool) {
	for prev := lineNum - 1; prev >= 0; prev-- {
		iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), prev)
		if iter.IsEOF() {
			continue
		}
		prevLine := iter.Line()
		if strings.TrimSpace(prevLine) == "" {
			continue
		}

		// Try to parse as a numbered list item: \s*\d+\.\s
		digitStart := -1
		digitEnd := -1
		for i := 0; i < len(prevLine); i++ {
			if prevLine[i] == ' ' || prevLine[i] == '\t' {
				if digitStart >= 0 {
					break
				}
				continue
			}
			if prevLine[i] >= '0' && prevLine[i] <= '9' {
				if digitStart < 0 {
					digitStart = i
				}
				digitEnd = i + 1
				continue
			}
			break
		}
		if digitStart < 0 || digitEnd >= len(prevLine) || prevLine[digitEnd] != '.' {
			continue // not a list item, keep scanning
		}
		if digitEnd+1 >= len(prevLine) || prevLine[digitEnd+1] != ' ' {
			continue
		}

		num, err := strconv.Atoi(prevLine[digitStart:digitEnd])
		if err != nil {
			continue
		}

		return num, prevLine[:digitStart], true
	}

	return 0, "", false
}

// autoDedentMarkdownListNumber replaces leading whitespace on the current line
// with the whitespace from the previous numbered list item.
func (e *Editor) autoDedentMarkdownListNumber(lineNum int) error {
	line := e.CurrentBuffer().CursorIterator.Line()
	cursorX := e.CurrentBuffer().CursorX

	// Find where digits start on current line
	digitStart := 0
	for digitStart < len(line) && (line[digitStart] == ' ' || line[digitStart] == '\t') {
		digitStart++
	}
	currentIndent := line[:digitStart]

	// Find the previous numbered list item's indent
	_, prevIndent, found := e.findPrevMarkdownListItem(lineNum)
	if !found {
		return nil
	}

	if currentIndent == prevIndent {
		return nil // nothing to do
	}

	// Build composite entry: delete old indent, insert new indent
	composite := buffer.NewCompositeEntry("auto-dedent markdown list number")
	composite.Entries = append(composite.Entries, buffer.NewDeleteTextEntry(lineNum, 0, currentIndent))
	if len(prevIndent) > 0 {
		composite.Entries = append(composite.Entries, buffer.NewInsertTextEntry(lineNum, 0, prevIndent))
	}

	if e.CurrentBuffer().History != nil {
		if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), composite); err != nil {
			return err
		}
	} else {
		for _, entry := range composite.Entries {
			if err := entry.Execute(e.CurrentBuffer()); err != nil {
				return err
			}
		}
	}

	// Adjust cursor position: shift by the indent difference
	indentDiff := len(currentIndent) - len(prevIndent)
	e.CurrentBuffer().CursorX = cursorX - indentDiff

	return nil
}
