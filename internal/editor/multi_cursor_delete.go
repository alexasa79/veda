package editor

import (
	"sort"
	"veda/internal/buffer"
)

type trackedVirtualCursor struct {
	index  int
	offset int
}

func (e *Editor) deleteCharacterInsertForAllCursors(args []string) error {
	buf := e.CurrentBuffer()
	viewLine := buf.ViewIterator.LineNumber()
	primaryOffset := e.cursorOffset(buf.CursorIterator.LineNumber(), buf.CursorX)
	virtualOffsets := make([]int, len(e.virtualCursors))
	targets := make([]trackedVirtualCursor, len(e.virtualCursors))
	for i, cursor := range e.virtualCursors {
		offset := e.cursorOffset(cursor.Y, cursor.X)
		virtualOffsets[i] = offset
		targets[i] = trackedVirtualCursor{index: i, offset: offset}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].offset > targets[j].offset
	})

	joinedLines := false
	for _, target := range targets {
		line, x := e.byteOffsetToLineChar(virtualOffsets[target.index])
		buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, line)
		buf.CursorX = x

		beforeLength := buf.Rope.Length()
		beforeLines := buf.LineCount
		if err := e.commandRegistry.Execute("delete-character-insert", args); err != nil {
			LL.Debug("Delete failed for virtual cursor %d: %v", target.index, err)
			continue
		}
		deleteOffset := e.getAbsolutePosition()
		deleted := beforeLength - buf.Rope.Length()
		virtualOffsets[target.index] = deleteOffset
		if buf.LineCount < beforeLines {
			joinedLines = true
		}
		if deleted > 0 {
			for i := range virtualOffsets {
				if i != target.index && virtualOffsets[i] > deleteOffset {
					virtualOffsets[i] -= deleted
				}
			}
			if primaryOffset > deleteOffset {
				primaryOffset -= deleted
			}
		}
	}

	line, x := e.byteOffsetToLineChar(primaryOffset)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, line)
	buf.CursorX = x
	beforeLength := buf.Rope.Length()
	beforeLines := buf.LineCount
	if err := e.commandRegistry.Execute("delete-character-insert", args); err != nil {
		return err
	}
	primaryOffset = e.getAbsolutePosition()
	deleted := beforeLength - buf.Rope.Length()
	if buf.LineCount < beforeLines {
		joinedLines = true
	}
	if deleted > 0 {
		for i := range virtualOffsets {
			if virtualOffsets[i] > primaryOffset {
				virtualOffsets[i] -= deleted
			}
		}
	}

	primaryLine, primaryX := e.byteOffsetToLineChar(primaryOffset)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, primaryLine)
	buf.CursorX = primaryX
	virtualCursors := make([]VirtualCursor, len(virtualOffsets))
	allOnPrimaryLine := true
	for i, offset := range virtualOffsets {
		line, x := e.byteOffsetToLineChar(offset)
		virtualCursors[i] = VirtualCursor{X: x, Y: line}
		allOnPrimaryLine = allOnPrimaryLine && line == primaryLine
	}
	if joinedLines && allOnPrimaryLine {
		virtualCursors = nil
	}
	e.setVirtualCursorsFor(buf, virtualCursors)

	if viewLine >= buf.LineCount {
		viewLine = buf.LineCount - 1
	}
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, viewLine)
	return nil
}

func (e *Editor) cursorOffset(line, x int) int {
	buf := e.CurrentBuffer()
	iterator := buffer.NewBufferIteratorAtLine(buf, line)
	if iterator.IsEOF() {
		return buf.Rope.Length()
	}
	lineText := iterator.Line()
	if x > len(lineText) {
		x = len(lineText)
	}
	return iterator.Position() + x
}
