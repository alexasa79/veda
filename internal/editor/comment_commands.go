package editor

import (
	"strings"
	"veda/internal/buffer"
)

// toggleComment toggles '//' comments on the current line.
// Count is handled in executeWithRepeatCount to operate on multiple lines.
func (e *Editor) toggleComment(args []string) error {
	return e.toggleCommentWithCount(1)
}

// toggleCommentWithCount toggles '//' comments on N lines starting at the
// cursor line. For multiple cursors, applies to each cursor independently.
func (e *Editor) toggleCommentWithCount(n int) error {
	if e.CurrentBuffer() == nil || n <= 0 {
		return nil
	}

	buf := e.CurrentBuffer()

	// Determine language-aware line comment prefix
	lang := ""
	if buf.Highlighter != nil {
		lang = buf.Highlighter.GetCurrentLanguage()
		if lang == "" {
			// Best effort: set from file path
			_ = buf.Highlighter.SetLanguageFromFile(buf.FilePath)
			lang = buf.Highlighter.GetCurrentLanguage()
		}
	}

	prefix := lineCommentPrefixFor(lang)
	if prefix == "" {
		// No known line comment style for this language; do nothing
		e.ShowInfo("No line comment style for %s", lang)
		return nil
	}

	// Save view and cursor positions prior to edits
	savedViewLine := buf.ViewIterator.LineNumber()
	savedCursorLine := buf.CursorIterator.LineNumber()
	savedCursorX := buf.CursorX

	// Group into a single undo unit
	if buf.History != nil {
		buf.History.StartComposite("toggle comment")
	}

	// Helper to toggle a single line by index
	toggleLine := func(lineNum int) {
		if lineNum < 0 || lineNum >= buf.LineCount {
			return
		}
		it := buffer.NewBufferIteratorAtLine(buf, lineNum)
		line := it.Line()

		// Determine leading indentation
		indent := 0
		for indent < len(line) {
			if line[indent] == ' ' || line[indent] == '\t' {
				indent++
			} else {
				break
			}
		}

		// Decide whether to comment or uncomment based on language prefix
		if strings.HasPrefix(line[indent:], prefix) {
			// Remove '<prefix>' and an optional single following space
			del := prefix
			nextIdx := indent + len(prefix)
			if nextIdx < len(line) && line[nextIdx] == ' ' {
				del = prefix + " "
			}
			entry := buffer.NewDeleteTextEntry(lineNum, indent, del)
			if buf.History != nil {
				if executed, err := buf.History.Execute(buf, entry); err == nil && executed != nil {
					e.setLastChange(executed)
					e.lastSimpleCommand = "toggle-comment"
				}
			} else {
				// Fallback (tests should always have History)
				pos := it.Offset + indent
				buf.Delete(pos, pos+len(del))
			}
		} else {
			// Add comment prefix after indentation
			ins := prefix + " "
			entry := buffer.NewInsertTextEntry(lineNum, indent, ins)
			if buf.History != nil {
				if executed, err := buf.History.Execute(buf, entry); err == nil && executed != nil {
					e.setLastChange(executed)
					e.lastSimpleCommand = "toggle-comment"
				}
			} else {
				pos := it.Offset + indent
				buf.Insert(pos, ins)
			}
		}
	}

	// Apply to primary cursor and all virtual cursors
	startLines := []int{}
	// Primary
	startLines = append(startLines, buf.CursorIterator.LineNumber())
	// Virtual cursors (if any)
	if vcs := e.GetVirtualCursors(); len(vcs) > 0 {
		for _, vc := range vcs {
			startLines = append(startLines, vc.Y)
		}
	}

	// Toggle per-cursor blocks
	for _, start := range startLines {
		for i := 0; i < n; i++ {
			lineIdx := start + i
			toggleLine(lineIdx)
		}
	}

	// End composite
	if buf.History != nil {
		buf.History.EndComposite()
	}

	// Recreate iterators using saved positions
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, savedViewLine)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, savedCursorLine)
	buf.CursorX = savedCursorX

	// Update syntax highlighting to reflect comment changes
	e.updateSyntaxHighlighting()

	return nil
}

// lineCommentPrefixFor returns the single-line comment prefix for a language name
// known by our highlighter (e.g., "go", "c", "cpp", "javascript", "typescript", "tsx", "python", "bash", "make").
// Returns empty string if no line comment is supported or unknown.
func lineCommentPrefixFor(lang string) string {
	switch lang {
	case "go", "c", "cpp", "javascript", "typescript", "tsx":
		return "//"
	case "python", "bash", "make":
		return "#"
	default:
		return ""
	}
}
