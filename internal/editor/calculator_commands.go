package editor

import (
	"sort"
	"veda/internal/buffer"
	"veda/internal/calculator"
	"veda/internal/commands"
)

type cursorTarget struct {
	line         int
	originalX    int
	isPrimary    bool
	virtualIndex int
}

func (e *Editor) collectCursorTargets() []cursorTarget {
	targets := make([]cursorTarget, 0, len(e.virtualCursors)+1)
	targets = append(targets, cursorTarget{
		line:      e.CurrentBuffer().CursorIterator.LineNumber(),
		originalX: e.CurrentBuffer().CursorX,
		isPrimary: true,
	})
	for idx, vc := range e.virtualCursors {
		targets = append(targets, cursorTarget{
			line:         vc.Y,
			originalX:    vc.X,
			virtualIndex: idx,
		})
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].line != targets[j].line {
			return targets[i].line < targets[j].line
		}
		return targets[i].originalX < targets[j].originalX
	})

	return targets
}

func snapshotVirtualCursors(vcs []VirtualCursor) []buffer.CursorPos {
	if len(vcs) == 0 {
		return nil
	}
	out := make([]buffer.CursorPos, len(vcs))
	for i, vc := range vcs {
		out[i] = buffer.CursorPos{Line: vc.Y, Char: vc.X}
	}
	return out
}

// calculateAppend evaluates the mathematical expression before each cursor and appends the result
func (e *Editor) calculateAppend(args []string) error {
	buf := e.CurrentBuffer()
	if buf == nil {
		return nil
	}

	viewLineNum := buf.ViewIterator.LineNumber()
	targets := e.collectCursorTargets()
	history := buf.History

	primaryBefore := buffer.CursorPos{Line: buf.CursorIterator.LineNumber(), Char: buf.CursorX}
	virtualBefore := snapshotVirtualCursors(e.virtualCursors)

	compositeActive := false
	snapshotSet := false
	if history != nil && len(targets) > 1 {
		history.StartComposite("Calculate append (multi-cursor)")
		compositeActive = true
	}

	defer func() {
		if compositeActive && history != nil {
			if snapshotSet {
				postPrimary := buffer.CursorPos{Line: buf.CursorIterator.LineNumber(), Char: buf.CursorX}
				postVirtual := snapshotVirtualCursors(e.virtualCursors)
				history.SetActiveCompositePostCursors(postPrimary, postVirtual)
			}
			history.EndComposite()
		}
	}()

	anySuccess := false
	primaryError := false

	for _, tgt := range targets {
		it := buffer.NewBufferIteratorAtLine(buf, tgt.line)
		line := it.Line()

		cursorX := tgt.originalX
		if cursorX > len(line) {
			cursorX = len(line)
		}
		endPos := cursorX
		if e.mode != commands.InsertMode && endPos < len(line) {
			endPos++
		}

		newLine, err := calculator.ProcessLineAppend(line, endPos)
		if err != nil {
			if tgt.isPrimary && !primaryError {
				primaryError = true
				e.ShowError("No valid expression found")
			}
			continue
		}

		anySuccess = true

		if history != nil {
			if compositeActive && !snapshotSet {
				history.SetActiveCompositeCursors(primaryBefore, virtualBefore)
				snapshotSet = true
			}
			if len(line) > 0 {
				del := buffer.NewDeleteTextEntry(tgt.line, 0, line)
				if _, err := history.Execute(buf, del); err != nil {
					return err
				}
			}
			ins := buffer.NewInsertTextEntry(tgt.line, 0, newLine)
			if executed, err := history.Execute(buf, ins); err != nil {
				return err
			} else if executed != nil {
				e.setLastChange(executed)
			}
		} else {
			if len(line) > 0 {
				del := buffer.NewDeleteTextEntry(tgt.line, 0, line)
				if err := del.Execute(buf); err != nil {
					return err
				}
			}
			ins := buffer.NewInsertTextEntry(tgt.line, 0, newLine)
			if err := ins.Execute(buf); err != nil {
				return err
			}
		}

		if tgt.isPrimary {
			buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, tgt.line)
			buf.CursorX = len(newLine)
		} else if tgt.virtualIndex >= 0 && tgt.virtualIndex < len(e.virtualCursors) {
			e.virtualCursors[tgt.virtualIndex].X = len(newLine)
			e.virtualCursors[tgt.virtualIndex].Y = tgt.line
		}
	}

	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, buf.CursorIterator.LineNumber())
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, viewLineNum)

	if len(e.virtualCursors) > 0 {
		e.syncCurrentVirtualCursors()
	}

	if anySuccess && !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}

// calculateReplace evaluates the mathematical expression before the cursor and replaces it with the result
func (e *Editor) calculateReplace(args []string) error {
	buf := e.CurrentBuffer()
	if buf == nil {
		return nil
	}

	// Save view position before modifications
	viewLineNum := buf.ViewIterator.LineNumber()

	targets := e.collectCursorTargets()
	history := buf.History

	primaryBefore := buffer.CursorPos{Line: buf.CursorIterator.LineNumber(), Char: buf.CursorX}
	virtualBefore := snapshotVirtualCursors(e.virtualCursors)

	compositeActive := false
	snapshotSet := false
	if history != nil && len(targets) > 1 {
		history.StartComposite("Calculate replace (multi-cursor)")
		compositeActive = true
	}

	var applyAt func(lineNum, endPos int) (int, error)
	applyAt = func(lineNum, endPos int) (int, error) {
		it := buffer.NewBufferIteratorAtLine(buf, lineNum)
		line := it.Line()
		if endPos > len(line) {
			endPos = len(line)
		}
		newLine, newCursorX, err := calculator.ProcessLineReplace(line, endPos)
		if err != nil {
			return 0, err
		}
		if history != nil && compositeActive && !snapshotSet {
			history.SetActiveCompositeCursors(primaryBefore, virtualBefore)
			snapshotSet = true
		}
		if len(line) > 0 {
			del := buffer.NewDeleteTextEntry(lineNum, 0, line)
			if history != nil {
				if _, err := history.Execute(buf, del); err != nil {
					return 0, err
				}
			} else {
				if err := del.Execute(buf); err != nil {
					return 0, err
				}
			}
		}
		ins := buffer.NewInsertTextEntry(lineNum, 0, newLine)
		if history != nil {
			if executed, err := history.Execute(buf, ins); err != nil {
				return 0, err
			} else if executed != nil {
				e.setLastChange(executed)
			}
		} else {
			if err := ins.Execute(buf); err != nil {
				return 0, err
			}
		}
		adjX := newCursorX
		if e.mode != commands.InsertMode && adjX > 0 {
			adjX--
		}
		if buf.CursorIterator.LineNumber() == lineNum {
			buf.CursorX = adjX
		}
		return adjX, nil
	}

	defer func() {
		if compositeActive && history != nil {
			if snapshotSet {
				postPrimary := buffer.CursorPos{Line: buf.CursorIterator.LineNumber(), Char: buf.CursorX}
				postVirtual := snapshotVirtualCursors(e.virtualCursors)
				history.SetActiveCompositePostCursors(postPrimary, postVirtual)
			}
			history.EndComposite()
		}
	}()

	for _, tgt := range targets {
		it := buffer.NewBufferIteratorAtLine(buf, tgt.line)
		line := it.Line()
		endPos := tgt.originalX
		if endPos > len(line) {
			endPos = len(line)
		}
		if e.mode != commands.InsertMode && endPos < len(line) {
			endPos++
		}
		newX, err := applyAt(tgt.line, endPos)
		if err != nil {
			e.ShowError("No valid expression found")
			continue
		}
		if tgt.isPrimary {
			buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, tgt.line)
		} else if tgt.virtualIndex >= 0 && tgt.virtualIndex < len(e.virtualCursors) {
			e.virtualCursors[tgt.virtualIndex].X = newX
			e.virtualCursors[tgt.virtualIndex].Y = tgt.line
		}
	}

	// Recreate iterators after buffer modifications
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, buf.CursorIterator.LineNumber())
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, viewLineNum)

	if len(e.virtualCursors) > 0 {
		e.syncCurrentVirtualCursors()
	}

	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}
	return nil
}
