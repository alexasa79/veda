package editor

import (
	"veda/internal/buffer"
	"veda/internal/commands"
)

// moveLinesUp moves the current line or visual line selection up by one line.
func (e *Editor) moveLinesUp(args []string) error {
	return e.moveLines(-1)
}

// moveLinesDown moves the current line or visual line selection down by one line.
func (e *Editor) moveLinesDown(args []string) error {
	return e.moveLines(1)
}

func (e *Editor) moveLines(direction int) error {
	buf := e.CurrentBuffer()
	if buf == nil || buf.LineCount == 0 {
		return nil
	}

	startLine, endLine, ok := e.lineRangeForMovement()
	if !ok {
		return nil
	}

	// Guard boundaries
	if direction < 0 && startLine == 0 {
		return nil
	}
	if direction > 0 && endLine >= buf.LineCount-1 {
		return nil
	}

	origCursorLine := buf.CursorIterator.LineNumber()
	origCursorX := buf.CursorX
	origViewLine := buf.ViewIterator.LineNumber()
	origMode := e.mode
	origAnchorLine := buf.VisualAnchorLine
	origAnchorOffset := buf.VisualAnchorOffset

	// Track whether we started a composite history entry
	startedComposite := false
	if buf.History != nil && !buf.History.HasActiveComposite() {
		desc := "Move lines"
		buf.History.StartComposite(desc)
		startedComposite = true
		primary := buffer.CursorPos{Line: origCursorLine, Char: origCursorX}
		buf.History.SetActiveCompositeCursors(primary, nil)
	}

	// Swap the target lines with the adjacent line in the desired direction.
	if err := e.swapLineBlock(startLine, endLine, direction); err != nil {
		if startedComposite {
			buf.History.EndComposite()
		}
		return err
	}

	// Refresh view iterator to the same logical line but current generation.
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, clampLine(origViewLine, buf.LineCount))

	// Re-indent only the lines that moved.
	newStart := startLine + direction
	newEnd := endLine + direction
	if err := e.reindentLineRange(newStart, newEnd); err != nil {
		if startedComposite {
			buf.History.EndComposite()
		}
		return err
	}

	// Update cursor and view positions after movement.
	newCursorLine := clampLine(origCursorLine+direction, buf.LineCount)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, newCursorLine)
	buf.CursorX = clampColumn(origCursorX, buf.CursorIterator.Line(), e.mode)

	if origMode == commands.VisualLineMode {
		buf.VisualAnchorLine = clampLine(origAnchorLine+direction, buf.LineCount)
		buf.VisualAnchorOffset = lineStartOffset(buf, buf.VisualAnchorLine)
	}

	// Allow the normal scrolling logic to adjust the viewport if needed.
	e.scroll()

	// Finalize history composite if we started one and capture undo unit.
	var executed buffer.HistoryEntry
	if startedComposite {
		primary := buffer.CursorPos{Line: buf.CursorIterator.LineNumber(), Char: buf.CursorX}
		buf.History.SetActiveCompositePostCursors(primary, nil)
		if entry := buf.History.EndComposite(); entry != nil {
			executed = entry
		}
	}

	if executed != nil {
		e.setLastChange(executed)
	}

	if direction < 0 {
		e.lastSimpleCommand = "move-lines-up"
	} else {
		e.lastSimpleCommand = "move-lines-down"
	}

	buf.MarkLinesChanged(newStart, newEnd)

	// Restore anchor offset for non-visual modes if it moved implicitly.
	if origMode != commands.VisualLineMode {
		buf.VisualAnchorLine = origAnchorLine
		buf.VisualAnchorOffset = origAnchorOffset
	}

	e.updateSyntaxHighlighting()
	return nil
}

func (e *Editor) lineRangeForMovement() (int, int, bool) {
	buf := e.CurrentBuffer()
	if buf == nil {
		return 0, 0, false
	}

	switch e.mode {
	case commands.VisualLineMode:
		startOff, endOff, ok := e.getSelectionRange()
		if !ok {
			return buf.CursorIterator.LineNumber(), buf.CursorIterator.LineNumber(), true
		}
		if endOff > startOff {
			endOff--
		}
		if endOff < startOff {
			endOff = startOff
		}
		startLine := clampLine(buf.GetLineAtPosition(startOff), buf.LineCount)
		endLine := clampLine(buf.GetLineAtPosition(endOff), buf.LineCount)
		if startLine > endLine {
			startLine, endLine = endLine, startLine
		}
		return startLine, endLine, true
	default:
		line := buf.CursorIterator.LineNumber()
		return line, line, true
	}
}

func (e *Editor) swapLineBlock(startLine, endLine, direction int) error {
	buf := e.CurrentBuffer()
	if buf == nil {
		return nil
	}

	var regionStartLine int
	var regionStart int
	var regionEnd int
	var newText string

	if direction < 0 {
		regionStartLine = startLine - 1
		regionStart = lineStartOffset(buf, regionStartLine)
		blockStart := lineStartOffset(buf, startLine)
		regionEnd = lineEndExclusive(buf, endLine)
		prevText := buf.Rope.Extract(regionStart, blockStart)
		blockText := buf.Rope.Extract(blockStart, regionEnd)
		newText = blockText + prevText
	} else {
		regionStartLine = startLine
		regionStart = lineStartOffset(buf, startLine)
		nextLineStart := lineStartOffset(buf, endLine+1)
		regionEnd = lineEndExclusive(buf, endLine+1)
		blockText := buf.Rope.Extract(regionStart, nextLineStart)
		nextText := buf.Rope.Extract(nextLineStart, regionEnd)
		newText = nextText + blockText
	}

	regionText := buf.Rope.Extract(regionStart, regionEnd)
	if regionText == newText {
		return nil
	}

	deleteEntry := buffer.NewDeleteTextEntry(regionStartLine, 0, regionText)
	insertEntry := buffer.NewInsertTextEntry(regionStartLine, 0, newText)
	composite := buffer.NewCompositeEntry("Swap lines")
	composite.Add(deleteEntry)
	composite.Add(insertEntry)

	if buf.History != nil {
		_, err := buf.History.Execute(buf, composite)
		return err
	}

	// Fallback without history (should not normally happen).
	buf.Delete(regionStart, regionEnd)
	buf.Insert(regionStart, newText)
	return nil
}

func (e *Editor) reindentLineRange(startLine, endLine int) error {
	buf := e.CurrentBuffer()
	if buf == nil {
		return nil
	}

	if startLine < 0 {
		startLine = 0
	}
	if endLine >= buf.LineCount {
		endLine = buf.LineCount - 1
	}
	for ln := startLine; ln <= endLine; ln++ {
		it := buffer.NewBufferIteratorAtLine(buf, ln)
		buf.CursorIterator = it
		buf.CursorX = 0
		if err := e.reindentExpression(nil); err != nil {
			return err
		}
	}
	return nil
}

func lineStartOffset(buf *buffer.Buffer, line int) int {
	if line < 0 {
		return 0
	}
	if line >= buf.LineCount {
		return buf.Rope.Length()
	}
	return buffer.NewBufferIteratorAtLine(buf, line).Position()
}

func lineEndExclusive(buf *buffer.Buffer, line int) int {
	if line < 0 {
		return 0
	}
	if line >= buf.LineCount {
		return buf.Rope.Length()
	}
	iter := buffer.NewBufferIteratorAtLine(buf, line)
	end := iter.LineEnd()
	if end < buf.Rope.Length() && buf.Rope.ByteAt(end) == '\n' {
		end++
	}
	return end
}

func clampLine(line, lineCount int) int {
	if line < 0 {
		return 0
	}
	max := lineCount - 1
	if max < 0 {
		max = 0
	}
	if line > max {
		return max
	}
	return line
}

func clampColumn(column int, line string, mode commands.Mode) int {
	if column < 0 {
		return 0
	}
	limit := len(line)
	if mode == commands.NormalMode {
		if limit > 0 {
			limit--
		} else {
			limit = 0
		}
	}
	if column > limit {
		return limit
	}
	return column
}
