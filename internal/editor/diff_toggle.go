package editor

import "fmt"

func (e *Editor) diffCurrentBuffer(args []string) error {
	v := e.findDiffViewForBuffer(e.CurrentBuffer())
	if v == nil {
		return e.StartDiffViewForCurrentBuffer()
	}
	path := v.RightBuffer.RealPath
	if path == "" {
		return fmt.Errorf("no working file for this diff")
	}
	buf := v.ActiveBuffer()
	row := buf.CursorIterator.LineNumber()
	line := 0
	// Deleted rows have no working-file line. Prefer the next surviving
	// line, or the preceding one when the deletion is at the end.
	for i, hint := range v.RightBuffer.DiffHints {
		if hint.IsFillerLine || hint.RightLine < 0 {
			continue
		}
		line = hint.RightLine
		if i >= row {
			break
		}
	}
	if row >= len(v.RightBuffer.DiffHints) {
		line++ // The synthetic empty line after a trailing newline.
	}
	x := buf.CursorX
	visualLine, _ := e.GetCursorVisualPosition()
	e.openFile(path)
	e.goToLine(line)
	e.CurrentBuffer().CursorX = clampCursorToRuneStart(e.CurrentBuffer().CursorIterator.Line(), x)
	e.preserveCursorVisualLine(visualLine)
	e.updateCursorState()
	return nil
}
