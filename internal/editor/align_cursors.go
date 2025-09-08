package editor

import (
	"strings"
	"veda/internal/buffer"
)

// alignCursors aligns all virtual cursors to the rightmost cursor position by adding spaces
func (e *Editor) alignCursors(args []string) error {
	// Only works when there are multiple cursors
	if len(e.virtualCursors) == 0 {
		e.ShowError("Align cursors requires multiple cursors")
		return nil
	}

	// Find the rightmost cursor position (maximum X coordinate)
	maxX := e.CurrentBuffer().CursorX
	for _, cursor := range e.virtualCursors {
		if cursor.X > maxX {
			maxX = cursor.X
		}
	}

	// Save view position and primary cursor line and position
	viewLineNum := e.CurrentBuffer().ViewIterator.LineNumber()
	primaryLineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	primaryCursorX := e.CurrentBuffer().CursorX

	// Start a composite operation for undo/redo
	if e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.StartComposite("Align cursors")
	}

	// Align virtual cursors first (in reverse order to maintain positions)
	for i := len(e.virtualCursors) - 1; i >= 0; i-- {
		cursor := e.virtualCursors[i]

		// Calculate how many spaces to add
		spacesToAdd := maxX - cursor.X
		if spacesToAdd > 0 {
			// Create spaces string
			spaces := strings.Repeat(" ", spacesToAdd)

			// Insert spaces at cursor position
			entry := buffer.NewInsertTextEntry(cursor.Y, cursor.X, spaces)
			if e.CurrentBuffer().History != nil {
				if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
					e.CurrentBuffer().History.EndComposite()
					return err
				}
			}

			// Update virtual cursor position
			e.virtualCursors[i].X = maxX
		}
	}

	// After modifying buffer for virtual cursors, recreate primary cursor iterator
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), primaryLineNum)
	// Restore the saved cursor X position
	e.CurrentBuffer().CursorX = primaryCursorX

	// Align primary cursor
	spacesToAdd := maxX - primaryCursorX
	if spacesToAdd > 0 {
		spaces := strings.Repeat(" ", spacesToAdd)
		entry := buffer.NewInsertTextEntry(primaryLineNum, primaryCursorX, spaces)
		if e.CurrentBuffer().History != nil {
			if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
				e.CurrentBuffer().History.EndComposite()
				return err
			}
		}
	}

	// End composite operation
	if e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.EndComposite()
	}

	// Restore iterators and update cursor X position
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), primaryLineNum)
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLineNum)
	e.CurrentBuffer().CursorX = maxX

	// Sync virtual cursors back to storage
	e.syncCurrentVirtualCursors()

	// Update syntax highlighting
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}
