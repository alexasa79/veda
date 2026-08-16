package editor

import (
	"veda/internal/buffer"
	"veda/internal/utils"
)

// rightDiffRow maps a working-file line to its row in the aligned diff.
func rightDiffRow(lines []buffer.DiffLine, rightLine int) int {
	for row, line := range lines {
		if line.RightLine == rightLine {
			return row
		}
	}

	// A trailing newline can put the cursor on the synthetic line after the
	// last diff row. In that case its source and aligned line numbers match.
	utils.Assert(rightLine == len(lines), "working-file cursor line missing from diff")
	return rightLine
}

// preserveDiffCursorVisualLine chooses the aligned view top that keeps the
// cursor closest to its previous terminal row.
func (e *Editor) preserveDiffCursorVisualLine(targetVisualLine int) {
	buf := e.CurrentBuffer()
	utils.Assert(buf != nil, "diff buffer is nil")

	cursorLine := buf.CursorIterator.LineNumber()
	bestViewLine := cursorLine
	bestDistance := targetVisualLine
	for viewLine := cursorLine; viewLine >= 0; viewLine-- {
		buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, viewLine)
		visualLine, _ := e.GetCursorVisualPosition()
		distance := visualLine - targetVisualLine
		if distance < 0 {
			distance = -distance
		}
		if distance < bestDistance {
			bestViewLine = viewLine
			bestDistance = distance
		}
		if visualLine >= targetVisualLine {
			break
		}
	}

	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, bestViewLine)
}
