package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/clipboard"
	"veda/internal/commands"
	"veda/internal/config"
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
	iter := buffer.NewBufferIterator(e.CurrentBuffer())
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
	deletedText := e.CurrentBuffer().Rope.Extract(start, end)

	// Store deleted text into 'd' register for later paste
	if e.registers == nil {
		e.registers = make(map[rune]string)
	}
	e.registers['d'] = deletedText
	// We are about to paste deleted lines back frequently; ensure next line-wise paste
	// does not auto-rebase indentation when coming right after a visual delete
	e.skipAutoIndentForNextPaste = true

	// Create delete history entry
	deleteEntry := buffer.NewDeleteTextEntry(lineNum, charPos, deletedText)

	// Execute the delete through history
	if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), deleteEntry); err != nil {
		return err
	}

	// Move cursor to start of deletion
	e.moveCursorToPosition(start)

	// Exit visual mode
	e.exitVisualMode(nil)

	// Mark buffer as modified
	e.CurrentBuffer().IsModified = true

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
	selectedText := e.CurrentBuffer().Rope.Extract(start, end)

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
	iter := buffer.NewBufferIterator(e.CurrentBuffer())
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
	deletedText := e.CurrentBuffer().Rope.Extract(start, end)

	// Create delete history entry
	deleteEntry := buffer.NewDeleteTextEntry(lineNum, charPos, deletedText)

	// Execute the delete through history
	if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), deleteEntry); err != nil {
		return err
	}

	// Move cursor to start of deletion
	e.moveCursorToPosition(start)

	// Exit visual mode and enter insert mode
	e.exitVisualMode(nil)
	e.enterInsertMode(nil)

	// Mark buffer as modified
	e.CurrentBuffer().IsModified = true

	// Track this as part of the change (delete + upcoming insert)
	e.lastChange = deleteEntry

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}

// indentSelection indents all lines covered by the current visual selection
func (e *Editor) indentSelection(args []string) error { return e.indentSelectionWithCount(1) }

func (e *Editor) indentSelectionWithCount(count int) error {
	if count <= 0 {
		count = 1
	}
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		return nil
	}

	start, end, hasSelection := e.getSelectionRange()
	if !hasSelection || start >= end {
		return nil
	}

	// Determine line range
	startLine := 0
	{
		iter := buffer.NewBufferIterator(e.CurrentBuffer())
		for !iter.IsEOF() {
			if start <= iter.LineEnd() {
				break
			}
			iter = iter.Next()
			startLine++
		}
	}
	endLine := 0
	{
		target := end - 1
		if target < 0 {
			target = 0
		}
		iter := buffer.NewBufferIterator(e.CurrentBuffer())
		for !iter.IsEOF() {
			if target <= iter.LineEnd() {
				break
			}
			iter = iter.Next()
			endLine++
		}
	}
	if endLine < startLine {
		return nil
	}

	// Build composite: insert (indentStr * count) at BOL for each line
	unit := e.CurrentBuffer().GetIndentationString()
	insertStr := strings.Repeat(unit, count)
	composite := buffer.NewCompositeEntry("Indent selection")
	for ln := startLine; ln <= endLine; ln++ {
		if insertStr != "" {
			composite.Add(buffer.NewInsertTextEntry(ln, 0, insertStr))
		}
	}
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), composite); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	}

	e.moveCursorToPosition(start)
	e.exitVisualMode(nil)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}
	return nil
}

// unindentSelection unindents all lines covered by the current visual selection
func (e *Editor) unindentSelection(args []string) error { return e.unindentSelectionWithCount(1) }

func (e *Editor) unindentSelectionWithCount(count int) error {
	if count <= 0 {
		count = 1
	}
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		return nil
	}

	start, end, hasSelection := e.getSelectionRange()
	if !hasSelection || start >= end {
		return nil
	}

	// Determine line range
	startLine := 0
	{
		iter := buffer.NewBufferIterator(e.CurrentBuffer())
		for !iter.IsEOF() {
			if start <= iter.LineEnd() {
				break
			}
			iter = iter.Next()
			startLine++
		}
	}
	endLine := 0
	{
		target := end - 1
		if target < 0 {
			target = 0
		}
		iter := buffer.NewBufferIterator(e.CurrentBuffer())
		for !iter.IsEOF() {
			if target <= iter.LineEnd() {
				break
			}
			iter = iter.Next()
			endLine++
		}
	}
	if endLine < startLine {
		return nil
	}

	composite := buffer.NewCompositeEntry("Unindent selection")
	useTabs := e.CurrentBuffer().Indentation.Type == config.IndentTabs
	width := e.CurrentBuffer().Indentation.Width
	if width <= 0 {
		width = 4
	}

	for ln := startLine; ln <= endLine; ln++ {
		iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), ln)
		if iter.IsEOF() {
			continue
		}
		line := iter.Line()

		// Simulate removing up to 'count' levels
		remaining := line
		removed := 0
		var delBuilder strings.Builder
		for i := 0; i < count; i++ {
			if len(remaining) == 0 {
				break
			}
			if useTabs {
				if remaining[0] == '\t' {
					delBuilder.WriteByte('\t')
					remaining = remaining[1:]
					removed++
					continue
				}
				// Remove up to 4 leading spaces as single dedent level
				spaceCount := 0
				for _, ch := range remaining {
					if ch == ' ' {
						spaceCount++
					} else {
						break
					}
					if spaceCount >= 4 {
						break
					}
				}
				if spaceCount == 0 {
					break
				}
				delBuilder.WriteString(remaining[:spaceCount])
				remaining = remaining[spaceCount:]
				removed += spaceCount
			} else {
				// Spaces mode: remove up to 'width' leading spaces per level
				spaceCount := 0
				for _, ch := range remaining {
					if ch == ' ' {
						spaceCount++
					} else {
						break
					}
					if spaceCount >= width {
						break
					}
				}
				if spaceCount == 0 {
					break
				}
				delBuilder.WriteString(remaining[:spaceCount])
				remaining = remaining[spaceCount:]
				removed += spaceCount
			}
		}
		if delBuilder.Len() > 0 {
			composite.Add(buffer.NewDeleteTextEntry(ln, 0, delBuilder.String()))
		}
	}

	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), composite); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	}

	e.moveCursorToPosition(start)
	e.exitVisualMode(nil)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}
	return nil
}

// startVisualLeft enters visual mode if not already active and moves left
func (e *Editor) startVisualLeft(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		if err := e.enterVisualMode(nil); err != nil {
			return err
		}
	}
	return e.moveLeft(nil)
}

// startVisualRight enters visual mode if not already active and moves right
func (e *Editor) startVisualRight(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		if err := e.enterVisualMode(nil); err != nil {
			return err
		}
	}
	return e.moveRight(nil)
}

// startVisualUp enters visual mode if not already active and moves up
func (e *Editor) startVisualUp(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		if err := e.enterVisualMode(nil); err != nil {
			return err
		}
	}
	return e.moveUp(nil)
}

// startVisualDown enters visual mode if not already active and moves down
func (e *Editor) startVisualDown(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}
	if e.mode != commands.VisualMode && e.mode != commands.VisualLineMode {
		if err := e.enterVisualMode(nil); err != nil {
			return err
		}
	}
	return e.moveDown(nil)
}
