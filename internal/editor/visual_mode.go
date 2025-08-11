package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/clipboard"
	"veda/internal/commands"
)

// deleteSelection deletes the current visual selection
func (e *Editor) deleteSelection(args []string) error {
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		return nil
	}
	
	// Get selection range
	start, end, hasSelection := e.getSelectionRange()
	if !hasSelection {
		return nil
	}
	
	// Calculate line and character position for history entry
	// Find the line containing the start position
	iter := buffer.NewBufferIterator(e.buffer)
	lineNum := 0
	lineStart := 0
	
	for !iter.IsEOF() {
		lineEnd := iter.LineEnd()
		if start <= lineEnd {
			// Position is on this line
			break
		}
		lineStart = lineEnd + 1 // +1 for newline
		iter = iter.Next()
		lineNum++
	}
	
	charPos := start - lineStart
	
	// Extract text that will be deleted
	deletedText := e.buffer.Rope.Extract(start, end)
	
	// Create delete history entry
	deleteEntry := buffer.NewDeleteTextEntry(lineNum, charPos, deletedText)
	
	// Execute the delete through history
	if _, err := e.buffer.History.Execute(e.buffer, deleteEntry); err != nil {
		return err
	}
	
	// Move cursor to start of deletion
	e.moveCursorToPosition(start)
	
	// Exit visual mode
	e.exitVisualMode(nil)
	
	// Mark buffer as modified
	e.buffer.IsModified = true
	
	// Track this as a repeatable change
	e.lastChange = deleteEntry
	
    // Update syntax highlighting (skip during paste mode)
    if !e.inPasteMode {
        e.updateSyntaxHighlighting()
    }
	
	return nil
}

// yankSelection yanks (copies) the current visual selection to system clipboard
func (e *Editor) yankSelection(args []string) error {
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		return nil
	}
	
	// Get selection range
	start, end, hasSelection := e.getSelectionRange()
	if !hasSelection {
		return nil
	}
	
	// Extract selected text
	selectedText := e.buffer.Rope.Extract(start, end)
	
	// For visual line mode, ensure the text ends with a newline
	if e.mode == commands.VisualLineMode && !strings.HasSuffix(selectedText, "\n") {
		selectedText += "\n"
	}
	
	// Copy to system clipboard
	if err := clipboard.Copy(selectedText); err != nil {
		LL.Error("Failed to copy to clipboard: %v", err)
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	
	// Move cursor to start of selection
	e.moveCursorToPosition(start)
	
	// Exit visual mode
	e.exitVisualMode(nil)
	
	LL.Debug("Yanked %d characters to clipboard", len(selectedText))
	
	return nil
}

// changeSelection deletes the selection and enters insert mode
func (e *Editor) changeSelection(args []string) error {
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		return nil
	}
	
	// Get selection range
	start, end, hasSelection := e.getSelectionRange()
	if !hasSelection {
		return nil
	}
	
	// Calculate line and character position for history entry
	// Find the line containing the start position
	iter := buffer.NewBufferIterator(e.buffer)
	lineNum := 0
	lineStart := 0
	
	for !iter.IsEOF() {
		lineEnd := iter.LineEnd()
		if start <= lineEnd {
			// Position is on this line
			break
		}
		lineStart = lineEnd + 1 // +1 for newline
		iter = iter.Next()
		lineNum++
	}
	
	charPos := start - lineStart
	
	// Extract text that will be deleted
	deletedText := e.buffer.Rope.Extract(start, end)
	
	// Create delete history entry
	deleteEntry := buffer.NewDeleteTextEntry(lineNum, charPos, deletedText)
	
	// Execute the delete through history
	if _, err := e.buffer.History.Execute(e.buffer, deleteEntry); err != nil {
		return err
	}
	
	// Move cursor to start of deletion
	e.moveCursorToPosition(start)
	
	// Exit visual mode and enter insert mode
	e.exitVisualMode(nil)
	e.enterInsertMode(nil)
	
	// Mark buffer as modified
	e.buffer.IsModified = true
	
	// Track this as part of the change (delete + upcoming insert)
	e.lastChange = deleteEntry
	
    // Update syntax highlighting (skip during paste mode)
    if !e.inPasteMode {
        e.updateSyntaxHighlighting()
    }
	
	return nil
}