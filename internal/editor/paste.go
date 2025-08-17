package editor

import (
    "fmt"
    "strings"
    "veda/internal/buffer"
    "veda/internal/clipboard"
)

// pasteAfter pastes clipboard content after the cursor (p command)
func (e *Editor) pasteAfter(args []string) error {
	// If we have virtual cursors, use multi-cursor paste
	if len(e.virtualCursors) > 0 {
		return e.pasteForAllCursors(func(content string) error {
			// Determine if this is a line-wise paste
			isLinewise := strings.HasSuffix(content, "\n")
			
			if isLinewise {
				return e.pasteAfterLine(content)
			} else {
				return e.pasteAfterCursor(content)
			}
		}, true)
	}
	
	// Single cursor paste - original implementation
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
	// If we have virtual cursors, use multi-cursor paste
	if len(e.virtualCursors) > 0 {
		return e.pasteForAllCursors(func(content string) error {
			// Determine if this is a line-wise paste
			isLinewise := strings.HasSuffix(content, "\n")
			
			if isLinewise {
				return e.pasteBeforeLine(content)
			} else {
				return e.pasteBeforeCursor(content)
			}
		}, false)
	}
	
	// Single cursor paste - original implementation
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
	// The CursorIterator points to the beginning of the line, so we need to add CursorX
	pos := e.buffer.CursorIterator.Position() + e.buffer.CursorX
	line := e.buffer.CursorIterator.Line()
	
	// Always move one character right to paste after cursor (unless line is empty)
	if len(line) > 0 {
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
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	return nil
}

// pasteBeforeCursor pastes content before the cursor position (character-wise)
func (e *Editor) pasteBeforeCursor(content string) error {
	// Save ViewIterator line number before buffer modification
	viewLine := e.buffer.ViewIterator.LineNumber()
	
	// Get current position
	// The CursorIterator points to the beginning of the line, so we need to add CursorX
	pos := e.buffer.CursorIterator.Position() + e.buffer.CursorX
	
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
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
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
	
    // Optionally auto-adjust indentation for code blocks when conditions are clear
    content = e.autoIndentPasteIfApplicable(content, lineNum, true)

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
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
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
	
    // Optionally auto-adjust indentation for code blocks when conditions are clear
    content = e.autoIndentPasteIfApplicable(content, lineNum, false)

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
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
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

// autoIndentPasteIfApplicable tries to rebase the indentation of a multi-line, line-wise
// pasted block to match the indentation level at the insertion position.
// It works only for code buffers (non-markdown) and only when indentation is obvious in both
// the pasted block (common leading whitespace across non-empty lines) and the target line
// (not a continuation indent). It returns the possibly adjusted content.
//
// Parameters:
// - content: pasted text (may end with a trailing newline for line-wise pastes)
// - currentLineNum: the cursor's current line number at the time of paste
// - after: whether this is an after-line paste (inserts at next line when possible)
func (e *Editor) autoIndentPasteIfApplicable(content string, currentLineNum int, after bool) string {
    // Only attempt for multi-line content
    if !strings.Contains(content, "\n") {
        return content
    }

    // Only when we have a code language highlighter that's not markdown
    if e.highlighter == nil {
        return content
    }
    lang := e.highlighter.GetCurrentLanguage()
    if lang == "" || lang == "markdown" {
        return content
    }

    // Determine the target line for indentation reference.
    // We align to the indentation of the current line where the user invoked paste,
    // not to the following line which may be a closing brace with smaller indent.
    targetLine := currentLineNum

    // Get indentation info for the target line; require non-continuation
    if targetLine < 0 || targetLine >= e.buffer.LineCount {
        return content
    }
    targetIter := buffer.NewBufferIteratorAtLine(e.buffer, targetLine)
    // Extract actual leading whitespace of the target line without relying on file indent settings
    lineText := targetIter.Line()
    j := 0
    for j < len(lineText) {
        ch := lineText[j]
        if ch == ' ' || ch == '\t' {
            j++
        } else {
            break
        }
    }
    targetIndent := lineText[:j]
    if targetIndent != "" {
        hasSpace := strings.Contains(targetIndent, " ")
        hasTab := strings.Contains(targetIndent, "\t")
        if hasSpace && hasTab {
            return content
        }
    }

    // Detect a common leading whitespace prefix across non-empty lines in the pasted block
    lines := strings.Split(content, "\n")
    // Exclude a possible trailing empty line from trailing newline
    if len(lines) > 0 && lines[len(lines)-1] == "" {
        lines = lines[:len(lines)-1]
    }
    if len(lines) == 0 {
        return content
    }

    common, ok := detectCommonLeadingWhitespace(lines)
    if !ok {
        return content
    }

    // If target indent equals common, nothing to do
    if common == targetIndent {
        return content
    }

    // Rebase: for each non-empty line, replace the common prefix with target indent
    rebased := make([]string, len(lines))
    for i, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed == "" {
            rebased[i] = line // keep whitespace-only or empty lines as-is
            continue
        }
        if strings.HasPrefix(line, common) {
            rebased[i] = targetIndent + strings.TrimPrefix(line, common)
        } else {
            // If a non-empty line does not have the common prefix, abort adjustment
            return content
        }
    }

    // Preserve trailing newline if original had it
    adjusted := strings.Join(rebased, "\n")
    if strings.HasSuffix(content, "\n") {
        adjusted += "\n"
    }
    return adjusted
}

// detectCommonLeadingWhitespace returns the common leading whitespace prefix shared by all
// non-empty lines. Returns ok=false if no non-empty lines, if the common prefix is empty,
// or if it mixes tabs and spaces (not obvious).
func detectCommonLeadingWhitespace(lines []string) (prefix string, ok bool) {
    first := true
    for _, line := range lines {
        // Consider only lines that have non-whitespace content
        if strings.TrimSpace(line) == "" {
            continue
        }
        // Extract this line's leading whitespace
        i := 0
        for i < len(line) {
            ch := line[i]
            if ch == ' ' || ch == '\t' {
                i++
            } else {
                break
            }
        }
        lw := line[:i]
        if first {
            prefix = lw
            first = false
        } else {
            // Reduce prefix to the common part
            max := len(prefix)
            if len(lw) < max {
                max = len(lw)
            }
            j := 0
            for j < max && prefix[j] == lw[j] {
                j++
            }
            prefix = prefix[:j]
            if prefix == "" {
                break
            }
        }
    }

    if first {
        // No non-empty lines
        return "", false
    }
    if prefix == "" {
        return "", false
    }
    // Require prefix to be all spaces or all tabs to be considered obvious
    hasSpace := strings.Contains(prefix, " ")
    hasTab := strings.Contains(prefix, "\t")
    if hasSpace && hasTab {
        return "", false
    }
    return prefix, true
}