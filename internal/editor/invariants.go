package editor

import (
	"fmt"

	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/utils"
)

// runInvariants checks editor invariants for the active window.
func (e *Editor) runInvariants() {
	if e.activeWindow < 0 || e.activeWindow >= len(e.windows) {
		return
	}
	w := e.windows[e.activeWindow]
	if w == nil {
		return
	}

	if len(w.Buffers) == 0 {
		if w.Buffer != nil {
			utils.Assert(false, fmt.Sprintf("active window has no buffers but active Buffer is set to %s", w.Buffer.DisplayName()))
		}
		return
	}

	if w.Buffer != nil {
		count := 0
		for _, b := range w.Buffers {
			if b == w.Buffer {
				count++
			}
		}
		if count == 0 {
			utils.Assert(false, fmt.Sprintf("active window active Buffer %s not present in Buffers list", w.Buffer.DisplayName()))
		}
		if count > 1 {
			utils.Assert(false, fmt.Sprintf("active window active Buffer %s appears multiple times in Buffers list", w.Buffer.DisplayName()))
		}
	}

	buf := e.CurrentBuffer()
	if buf == nil {
		return
	}

	// Check diff buffer cursor sync
	if dv := e.findDiffViewForBuffer(buf); dv != nil {
		left := dv.LeftBuffer
		right := dv.RightBuffer
		utils.Assert(left != nil && right != nil, "DiffBufferView has nil buffer")
		if left != nil && right != nil {
			leftLine := left.CursorIterator.LineNumber()
			rightLine := right.CursorIterator.LineNumber()
			utils.Assert(leftLine == rightLine,
				fmt.Sprintf("diff view cursor lines out of sync: left=%d right=%d (left=%s, right=%s)",
					leftLine, rightLine, left.DisplayName(), right.DisplayName()))

			leftViewLine := left.ViewIterator.LineNumber()
			rightViewLine := right.ViewIterator.LineNumber()
			utils.Assert(leftViewLine == rightViewLine,
				fmt.Sprintf("diff view view lines out of sync: left=%d right=%d (left=%s, right=%s)",
					leftViewLine, rightViewLine, left.DisplayName(), right.DisplayName()))

			// Check that cursor appears at the same physical Y in both panes.
			e.checkDiffCursorVisualSync(dv, w)
		}
	} else if e.isDiffBuffer(buf) {
		// Legacy two-window fallback
		pair := e.getDiffPairBuffer(buf)
		utils.Assert(pair != nil, fmt.Sprintf("diff buffer %s missing paired buffer", buf.DisplayName()))
		if pair != nil {
			currentLine := buf.CursorIterator.LineNumber()
			pairLine := pair.CursorIterator.LineNumber()
			utils.Assert(currentLine == pairLine,
				fmt.Sprintf("diff view cursor lines out of sync: current=%d pair=%d (buffer=%s, pair=%s)",
					currentLine, pairLine, buf.DisplayName(), pair.DisplayName()))

			currentViewLine := buf.ViewIterator.LineNumber()
			pairViewLine := pair.ViewIterator.LineNumber()
			utils.Assert(currentViewLine == pairViewLine,
				fmt.Sprintf("diff view view lines out of sync: current=%d pair=%d (buffer=%s, pair=%s)",
					currentViewLine, pairViewLine, buf.DisplayName(), pair.DisplayName()))
		}
	}

	if e.mode == commands.CommandMode || e.mode == commands.SearchMode {
		return
	}

	if e.mode == commands.TerminalMode && buf.IsInternal {
		if ts := e.terminalSessions[buf.FilePath]; ts != nil && ts.term != nil {
			return
		}
	}

	visualLine, visualCol := e.GetCursorVisualPosition()
	cursorY := visualLine + e.getContentStartY()
	width, _ := e.screen.Size()
	if visualCol >= width {
		visualCol = width - 1
	}
	contentStartY := e.getContentStartY()
	statusBarY := e.getStatusBarY()
	visibleHeight := e.getVisibleHeight()
	logicalLine := buf.CursorIterator.LineNumber()
	cursorX := buf.CursorX
	viewTop := buf.ViewIterator.LineNumber()
	lineNumberWidth := buf.LineNumberWidth

	utils.Assert(visualLine >= 0,
		fmt.Sprintf("visualLine=%d is negative (logicalLine=%d, cursorX=%d, viewTop=%d)",
			visualLine, logicalLine, cursorX, viewTop))

	utils.Assert(cursorY >= contentStartY,
		fmt.Sprintf("cursorY=%d < contentStartY=%d (visualLine=%d, logicalLine=%d, cursorX=%d, viewTop=%d)",
			cursorY, contentStartY, visualLine, logicalLine, cursorX, viewTop))

	utils.Assert(cursorY < statusBarY,
		fmt.Sprintf("cursorY=%d >= statusBarY=%d (visualLine=%d, contentStartY=%d, logicalLine=%d, cursorX=%d, viewTop=%d, visibleHeight=%d)",
			cursorY, statusBarY, visualLine, contentStartY, logicalLine, cursorX, viewTop, visibleHeight))

	utils.Assert(visualCol >= 0,
		fmt.Sprintf("visualCol=%d is negative (logicalLine=%d, cursorX=%d)",
			visualCol, logicalLine, cursorX))

	utils.Assert(visualCol < width,
		fmt.Sprintf("visualCol=%d >= screenWidth=%d (logicalLine=%d, cursorX=%d, lineNumberWidth=%d)",
			visualCol, width, logicalLine, cursorX, lineNumberWidth))

	utils.Assert(visualLine < visibleHeight,
		fmt.Sprintf("visualLine=%d >= visibleHeight=%d (logicalLine=%d, cursorX=%d, viewTop=%d)",
			visualLine, visibleHeight, logicalLine, cursorX, viewTop))
}

// checkDiffCursorVisualSync asserts that the cursor line occupies the same
// physical terminal Y in both diff panes.
func (e *Editor) checkDiffCursorVisualSync(dv *DiffBufferView, win *Window) {
	leftW := win.Width / 2
	rightW := win.Width - leftW
	if leftW < 1 || rightW < 1 {
		return
	}

	// Compute visual cursor Y for a given buffer inside a pane-sized sub-screen.
	visualY := func(buf *buffer.Buffer, paneW int) int {
		realScreen := e.screen
		realBuf := e.CurrentBuffer()

		sub := newWindowScreen(e.screen, 0, 0, paneW, win.Height, false)
		e.screen = sub
		e.SetCurrentBuffer(buf)

		y, _ := e.GetCursorVisualPosition()

		e.screen = realScreen
		e.SetCurrentBuffer(realBuf)
		return y
	}

	leftY := visualY(dv.LeftBuffer, leftW)
	rightY := visualY(dv.RightBuffer, rightW)

	utils.Assert(leftY == rightY,
		fmt.Sprintf("diff view cursor physical Y out of sync: leftY=%d rightY=%d (cursorLine=%d, viewLine=%d, left=%s, right=%s)",
			leftY, rightY,
			dv.LeftBuffer.CursorIterator.LineNumber(),
			dv.LeftBuffer.ViewIterator.LineNumber(),
			dv.LeftBuffer.DisplayName(), dv.RightBuffer.DisplayName()))
}
