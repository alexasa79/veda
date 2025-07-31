package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/clipboard"
)

// pasteAfter pastes clipboard content after the cursor (p command)
func (e *Editor) pasteAfter(args []string) error {
	// Get clipboard content
	content, err := clipboard.Paste()
	if err != nil {
		LL.Error("Failed to paste from clipboard: %v", err)
		return nil // Don't return error to avoid showing error to user
	}
	
	if content == "" {
		return nil
	}
	
	// Determine if this is a line-wise paste (content ends with newline)
	isLinewise := strings.HasSuffix(content, "\n")
	
	if isLinewise {
		// Line-wise paste: insert after current line
		return e.pasteAfterLine(content)
	} else {
		// Character-wise paste: insert after cursor
		return e.pasteAfterCursor(content)
	}
}

// pasteBefore pastes clipboard content before the cursor (P command)
func (e *Editor) pasteBefore(args []string) error {
	// Get clipboard content
	content, err := clipboard.Paste()
	if err != nil {
		LL.Error("Failed to paste from clipboard: %v", err)
		return nil // Don't return error to avoid showing error to user
	}
	
	if content == "" {
		return nil
	}
	
	// Determine if this is a line-wise paste (content ends with newline)
	isLinewise := strings.HasSuffix(content, "\n")
	
	if isLinewise {
		// Line-wise paste: insert before current line
		return e.pasteBeforeLine(content)
	} else {
		// Character-wise paste: insert before cursor
		return e.pasteBeforeCursor(content)
	}
}

// pasteAfterCursor pastes content after the cursor position (character-wise)
func (e *Editor) pasteAfterCursor(content string) error {
	// Save ViewIterator line number before buffer modification
	viewLine := e.buffer.ViewIterator.LineNumber()
	
	// Get current position
	pos := e.buffer.CursorIterator.Position()
	line := e.buffer.CursorIterator.Line()
	
	// If we're not at the end of the line, move one character right
	if e.buffer.CursorX < len(line) {
		pos++
	}
	
	// Create paste command for undo/redo
	pasteCmd := &PasteCommand{
		buffer:      e.buffer,
		position:    pos,
		content:     content,
		isLinewise:  false,
		oldCursorX:  e.buffer.CursorX,
		oldCursorY:  e.buffer.CursorIterator.LineNumber(),
	}
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, pasteCmd); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		pasteCmd.Execute(e.buffer)
	}
	
	// Recreate ViewIterator after buffer modification
	if viewLine >= e.buffer.LineCount {
		viewLine = e.buffer.LineCount - 1
	}
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLine)
	
	return nil
}

// pasteBeforeCursor pastes content before the cursor position (character-wise)
func (e *Editor) pasteBeforeCursor(content string) error {
	// Save ViewIterator line number before buffer modification
	viewLine := e.buffer.ViewIterator.LineNumber()
	
	// Get current position
	pos := e.buffer.CursorIterator.Position()
	
	// Create paste command for undo/redo
	pasteCmd := &PasteCommand{
		buffer:      e.buffer,
		position:    pos,
		content:     content,
		isLinewise:  false,
		oldCursorX:  e.buffer.CursorX,
		oldCursorY:  e.buffer.CursorIterator.LineNumber(),
	}
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, pasteCmd); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		pasteCmd.Execute(e.buffer)
	}
	
	// Recreate ViewIterator after buffer modification
	if viewLine >= e.buffer.LineCount {
		viewLine = e.buffer.LineCount - 1
	}
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLine)
	
	return nil
}

// pasteAfterLine pastes content after the current line (line-wise)
func (e *Editor) pasteAfterLine(content string) error {
	// Save ViewIterator line number before buffer modification
	viewLine := e.buffer.ViewIterator.LineNumber()
	
	// Get current line number
	lineNum := e.buffer.CursorIterator.LineNumber()
	
	// Find the start of the next line
	var pos int
	if lineNum < e.buffer.LineCount - 1 {
		// Not the last line, insert at start of next line
		iter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum + 1)
		pos = iter.Position()
	} else {
		// Last line, insert at end of buffer
		iter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		pos = iter.LineEnd()
		// Add newline before pasted content if buffer doesn't end with one
		if pos > 0 && e.buffer.Rope.ByteAt(pos-1) != '\n' {
			content = "\n" + content
		}
	}
	
	// Create paste command for undo/redo
	pasteCmd := &PasteCommand{
		buffer:      e.buffer,
		position:    pos,
		content:     content,
		isLinewise:  true,
		oldCursorX:  e.buffer.CursorX,
		oldCursorY:  e.buffer.CursorIterator.LineNumber(),
	}
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, pasteCmd); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		pasteCmd.Execute(e.buffer)
	}
	
	// Recreate ViewIterator after buffer modification
	// Adjust view line if we added lines
	newLineCount := strings.Count(content, "\n")
	if viewLine > lineNum {
		viewLine += newLineCount
	}
	if viewLine >= e.buffer.LineCount {
		viewLine = e.buffer.LineCount - 1
	}
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLine)
	
	return nil
}

// pasteBeforeLine pastes content before the current line (line-wise)
func (e *Editor) pasteBeforeLine(content string) error {
	// Save ViewIterator line number before buffer modification
	viewLine := e.buffer.ViewIterator.LineNumber()
	
	// Get current line number
	lineNum := e.buffer.CursorIterator.LineNumber()
	
	// Find the start of the current line
	iter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	pos := iter.Position()
	
	// Create paste command for undo/redo
	pasteCmd := &PasteCommand{
		buffer:      e.buffer,
		position:    pos,
		content:     content,
		isLinewise:  true,
		oldCursorX:  e.buffer.CursorX,
		oldCursorY:  e.buffer.CursorIterator.LineNumber(),
	}
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, pasteCmd); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		pasteCmd.Execute(e.buffer)
	}
	
	// Recreate ViewIterator after buffer modification
	// Adjust view line if we added lines before the view
	newLineCount := strings.Count(content, "\n")
	if viewLine >= lineNum {
		viewLine += newLineCount
	}
	if viewLine >= e.buffer.LineCount {
		viewLine = e.buffer.LineCount - 1
	}
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLine)
	
	return nil
}

// PasteCommand represents a paste operation for undo/redo
type PasteCommand struct {
	buffer      *buffer.Buffer
	position    int
	content     string
	isLinewise  bool
	oldCursorX  int
	oldCursorY  int
}

func (p *PasteCommand) Execute(buf *buffer.Buffer) error {
	// Insert the content
	buf.Insert(p.position, p.content)
	
	// Update cursor position
	if p.isLinewise {
		// For line-wise paste, move cursor to first non-whitespace character of pasted line
		lineNum := p.oldCursorY
		
		// Determine if we need to adjust line number based on paste position
		// We need to convert position to line number to compare
		posLineNum := 0
		for i := 0; i < buf.LineCount; i++ {
			iter := buffer.NewBufferIteratorAtLine(buf, i)
			lineEnd := iter.LineEnd()
			if p.position <= lineEnd {
				posLineNum = i
				break
			}
		}
		
		if posLineNum <= p.oldCursorY {
			// Pasted at or before current line, adjust line number
			lineNum += strings.Count(p.content, "\n")
		}
		
		buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, lineNum)
		
		// Move to first non-whitespace character
		line := buf.CursorIterator.Line()
		buf.CursorX = 0
		for i, ch := range line {
			if ch != ' ' && ch != '\t' {
				buf.CursorX = i
				break
			}
		}
	} else {
		// For character-wise paste, move cursor to end of pasted text
		newPos := p.position + len(p.content) - 1
		if newPos < 0 {
			newPos = 0
		}
		
		// Find the line and column for the new position
		iter := buffer.NewBufferIterator(buf)
		lineNum := 0
		lineStart := 0
		
		for !iter.IsEOF() {
			lineEnd := iter.LineEnd()
			if newPos <= lineEnd {
				// Position is on this line
				break
			}
			lineStart = lineEnd + 1
			lineNum++
			iter = iter.Next()
		}
		
		// Set cursor position
		buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, lineNum)
		buf.CursorX = newPos - lineStart
		if buf.CursorX < 0 {
			buf.CursorX = 0
		}
	}
	
	return nil
}

func (p *PasteCommand) Undo(buf *buffer.Buffer) error {
	// Delete the pasted content
	buf.Delete(p.position, p.position + len(p.content))
	
	// Restore cursor position
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, p.oldCursorY)
	buf.CursorX = p.oldCursorX
	
	return nil
}

func (p *PasteCommand) Description() string {
	if p.isLinewise {
		lines := strings.Count(p.content, "\n")
		return fmt.Sprintf("Paste %d line(s)", lines)
	}
	return fmt.Sprintf("Paste %d character(s)", len(p.content))
}