package editor

import (
	"sort"
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
	// Helper to apply calculate-replace at a given (lineNum, endPos). Returns new cursor X for that line.
	applyAt := func(lineNum, endPos int) (int, error) {
		it := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
		line := it.Line()
		if endPos > len(line) {
			endPos = len(line)
		}
		// Compute replacement
		newLine, newCursorX, err := calculator.ProcessLineReplace(line, endPos)
		if err != nil {
			return 0, err
		}
		// Replace line content
		if len(line) > 0 {
			del := buffer.NewDeleteTextEntry(lineNum, 0, line)
			if e.CurrentBuffer().History != nil {
				if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), del); err != nil {
					return 0, err
				}
			}
		}
		ins := buffer.NewInsertTextEntry(lineNum, 0, newLine)
		if e.CurrentBuffer().History != nil {
			if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), ins); err != nil {
				return 0, err
			} else if executed != nil {
				e.setLastChange(executed)
			}
		}
		// Compute cursor X based on mode: Normal -> on last char; Insert -> after last char
		adjX := newCursorX
		if e.mode != commands.InsertMode && adjX > 0 {
			adjX--
		}
		// Update primary cursor when operating on its line
		if e.CurrentBuffer().CursorIterator.LineNumber() == lineNum {
			e.CurrentBuffer().CursorX = adjX
		}
		return adjX, nil
	}

	// Save view position before modifications
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()

	// If there are virtual cursors, apply sequentially top-to-bottom
	if len(e.virtualCursors) > 0 {
		// Collect all cursor positions including primary
		all := make([]VirtualCursor, 0, len(e.virtualCursors)+1)
		all = append(all, VirtualCursor{X: e.CurrentBuffer().CursorX, Y: e.CurrentBuffer().CursorIterator.LineNumber()})
		all = append(all, e.virtualCursors...)
		// Sort by Y then X
		sort.Slice(all, func(i, j int) bool {
			if all[i].Y != all[j].Y {
				return all[i].Y < all[j].Y
			}
			return all[i].X < all[j].X
		})
		// Apply in order; let $? propagate between lines
		for _, c := range all {
			it := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), c.Y)
			line := it.Line()
			endPos := c.X
			if endPos > len(line) {
				endPos = len(line)
			}
			if e.mode != commands.InsertMode && endPos < len(line) {
				endPos++
			}
			newX, err := applyAt(c.Y, endPos)
			if err != nil {
				e.ShowError("No valid expression found")
				continue
			}
			// Update matching virtual cursor (if any) to the new X (already adjusted for mode)
			for i := range e.virtualCursors {
				if e.virtualCursors[i].Y == c.Y && e.virtualCursors[i].X == c.X {
					e.virtualCursors[i].X = newX
					break
				}
			}
		}
	} else {
		// Single-cursor behavior (original)
		lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
		line := e.CurrentBuffer().CursorIterator.Line()
		cursorX := e.CurrentBuffer().CursorX
		if cursorX > len(line) {
			cursorX = len(line)
		}
		endPos := cursorX
		if e.mode != commands.InsertMode && endPos < len(line) {
			endPos++
		}
		if _, err := applyAt(lineNum, endPos); err != nil {
			e.ShowError("No valid expression found")
		}
	}

	// Recreate iterators after buffer modifications
	// Keep view stable
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), e.CurrentBuffer().CursorIterator.LineNumber())
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}
	return nil
}
