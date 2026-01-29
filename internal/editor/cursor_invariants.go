package editor

import (
	"fmt"

	"veda/internal/utils"
)

// AssertCursorInvariants validates that cursor position values are consistent and within bounds.
// visualLine and visualCol are the computed visual cursor position.
// cursorY is the screen Y coordinate where the cursor will be placed.
// This function should be called after computing cursor position to catch bugs early.
func (e *Editor) AssertCursorInvariants(visualLine, visualCol, cursorY int) {
	width, _ := e.screen.Size()
	contentStartY := e.getContentStartY()
	statusBarY := e.getStatusBarY()
	visibleHeight := e.getVisibleHeight()
	logicalLine := e.CurrentBuffer().CursorIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX
	viewTop := e.CurrentBuffer().ViewIterator.LineNumber()
	lineNumberWidth := e.CurrentBuffer().LineNumberWidth

	// Invariant 1: visualLine must be non-negative
	utils.Assert(visualLine >= 0,
		fmt.Sprintf("visualLine=%d is negative (logicalLine=%d, cursorX=%d, viewTop=%d)",
			visualLine, logicalLine, cursorX, viewTop))

	// Invariant 2: cursorY must be at or after content start
	utils.Assert(cursorY >= contentStartY,
		fmt.Sprintf("cursorY=%d < contentStartY=%d (visualLine=%d, logicalLine=%d, cursorX=%d, viewTop=%d)",
			cursorY, contentStartY, visualLine, logicalLine, cursorX, viewTop))

	// Invariant 3: cursorY must be within content area (not on or past status bar)
	utils.Assert(cursorY < statusBarY,
		fmt.Sprintf("cursorY=%d >= statusBarY=%d (visualLine=%d, contentStartY=%d, logicalLine=%d, cursorX=%d, viewTop=%d, visibleHeight=%d)",
			cursorY, statusBarY, visualLine, contentStartY, logicalLine, cursorX, viewTop, visibleHeight))

	// Invariant 4: visualCol must be non-negative
	utils.Assert(visualCol >= 0,
		fmt.Sprintf("visualCol=%d is negative (logicalLine=%d, cursorX=%d)",
			visualCol, logicalLine, cursorX))

	// Invariant 5: visualCol must be within screen width (can be at width when cursor is at end of line)
	utils.Assert(visualCol <= width,
		fmt.Sprintf("visualCol=%d > screenWidth=%d (logicalLine=%d, cursorX=%d, lineNumberWidth=%d)",
			visualCol, width, logicalLine, cursorX, lineNumberWidth))

	// Invariant 6: visualLine must be less than visibleHeight
	utils.Assert(visualLine < visibleHeight,
		fmt.Sprintf("visualLine=%d >= visibleHeight=%d (logicalLine=%d, cursorX=%d, viewTop=%d)",
			visualLine, visibleHeight, logicalLine, cursorX, viewTop))
}
