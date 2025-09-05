package editor

import (
	"unicode"
	"veda/internal/buffer"
	"veda/internal/commands"
)

// toUpperCmd transforms current word (Normal) or selection (Visual) to upper-case.
func (e *Editor) toUpperCmd(args []string) error { return e.caseTransform(true) }

// toLowerCmd transforms current word (Normal) or selection (Visual) to lower-case.
func (e *Editor) toLowerCmd(args []string) error { return e.caseTransform(false) }

func (e *Editor) caseTransform(toUpper bool) error {
	if e.CurrentBuffer() == nil {
		return nil
	}
	// Capture current view iterator line to restore after buffer mutations,
	// since any change invalidates iterators.
	preViewLine := e.CurrentBuffer().ViewIterator.LineNumber()
	// Visual modes: operate on selection range
	if e.mode == commands.VisualMode || e.mode == commands.VisualLineMode {
		startByte, endByte, has := e.getSelectionRange()
		if !has || startByte >= endByte {
			return nil
		}
		// Extract the selected text directly by absolute offsets
		old := e.CurrentBuffer().Rope.Extract(startByte, endByte)
		if old == "" {
			return nil
		}
		transformed := transformStringASCII(old, toUpper)
		// Compute (line, char) for the start byte using robust logic
		iter := buffer.NewBufferIterator(e.CurrentBuffer())
		lineNum := 0
		lineStart := 0
		for !iter.IsEOF() {
			lineEnd := iter.LineEnd()
			if startByte <= lineEnd {
				break
			}
			lineStart = lineEnd + 1 // include newline
			iter = iter.Next()
			lineNum++
		}
		charPos := startByte - lineStart

		// Replace old with transformed as a single composite unit and mark as lastChange
        if e.CurrentBuffer().History != nil {
            // Mark this as a repeatable simple command prior to recording lastChange
            if toUpper {
                e.lastSimpleCommand = "to-upper"
            } else {
                e.lastSimpleCommand = "to-lower"
            }
            composite := buffer.NewCompositeEntry("case-transform")
            composite.Add(buffer.NewDeleteTextEntry(lineNum, charPos, old))
            composite.Add(buffer.NewInsertTextEntry(lineNum, charPos, transformed))
            if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), composite); err != nil {
                return err
            } else if executed != nil {
                e.setLastChange(executed)
            }
		} else {
			// Fallback without history
			_ = buffer.NewDeleteTextEntry(lineNum, charPos, old).Execute(e.CurrentBuffer())
			_ = buffer.NewInsertTextEntry(lineNum, charPos, transformed).Execute(e.CurrentBuffer())
		}
		// Incremental highlight update
		e.updateSyntaxApplyEdit(startByte, old, transformed)
		// Restore view iterator after mutations
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), preViewLine)
		return nil
	}

	// Normal mode: transform word under cursor
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	it := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lineNum)
	line := it.Line()
	if len(line) == 0 {
		return nil
	}
	cx := e.CurrentBuffer().CursorX
	if cx < 0 {
		cx = 0
	}
	if cx >= len(line) {
		cx = len(line) - 1
	}
	// Find word boundaries
	if !isWordByte(line[cx]) {
		// If on non-word, try next to the right if within line
		if cx+1 < len(line) && isWordByte(line[cx+1]) {
			cx = cx + 1
		} else {
			return nil
		}
	}
	start := cx
	for start > 0 && isWordByte(line[start-1]) {
		start--
	}
	end := cx
	for end < len(line) && isWordByte(line[end]) {
		end++
	}
	if start >= end {
		return nil
	}

	// Compute absolute start byte
	absStart := it.Position() + start
	old := line[start:end]
	transformed := transformStringASCII(old, toUpper)
	if transformed == old {
		return nil
	}

	// Replace using history
    if e.CurrentBuffer().History != nil {
        // Mark this as a repeatable simple command prior to recording lastChange
        if toUpper {
            e.lastSimpleCommand = "to-upper"
        } else {
            e.lastSimpleCommand = "to-lower"
        }
        composite := buffer.NewCompositeEntry("case-transform-word")
        composite.Add(buffer.NewDeleteTextEntry(lineNum, start, old))
        composite.Add(buffer.NewInsertTextEntry(lineNum, start, transformed))
        if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), composite); err != nil {
            return err
        } else if executed != nil {
            e.setLastChange(executed)
        }
	} else {
		_ = buffer.NewDeleteTextEntry(lineNum, start, old).Execute(e.CurrentBuffer())
		_ = buffer.NewInsertTextEntry(lineNum, start, transformed).Execute(e.CurrentBuffer())
	}

	e.updateSyntaxApplyEdit(absStart, old, transformed)
	// Restore view iterator after mutations
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), preViewLine)
	return nil
}

func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func transformStringASCII(s string, toUpper bool) string {
	// Use unicode to be safe, though ASCII-only rules for speed
	r := []rune(s)
	for i, ch := range r {
		if toUpper {
			r[i] = unicode.ToUpper(ch)
		} else {
			r[i] = unicode.ToLower(ch)
		}
	}
	return string(r)
}

// byteOffsetToLineChar converts an absolute byte offset into (line,char)
func (e *Editor) byteOffsetToLineChar(offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if e.CurrentBuffer() == nil {
		return 0, 0
	}
	// Iterate lines accumulating lengths
	it := buffer.NewBufferIterator(e.CurrentBuffer())
	for !it.IsEOF() {
		start := it.Position()
		end := it.LineEnd()
		if offset <= end { // within this line (end is last byte index of line)
			char := offset - start
			if char < 0 {
				char = 0
			}
			line := it.LineNumber()
			// clamp char
			lstr := it.Line()
			if char > len(lstr) {
				char = len(lstr)
			}
			return line, char
		}
		it = it.Next()
	}
	// Past end -> last line
	last := e.CurrentBuffer().LineCount - 1
	if last < 0 {
		return 0, 0
	}
	it = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), last)
	return last, len(it.Line())
}
