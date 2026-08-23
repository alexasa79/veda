package editor

import (
	"veda/internal/buffer"
	"veda/internal/commands"
)

// scroll ensures the cursor is visible within the viewport with appropriate margins
func (e *Editor) scroll() {
	t0 := e.perfTracker.Start()

	// Skip scrolling for empty windows
	if e.CurrentBuffer() == nil {
		return
	}

	// Perform wrap-aware scrolling using the active window region for sizing.
	// Temporarily compute visual metrics inside a windowScreen matching the
	// active window, so CalculateWrappedLines/GetCursorVisualPosition reflect
	// the pane width/height (important for diff/narrow panes).
	realScreen := e.screen
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
		win := e.windows[e.activeWindow]
		view := win.View
		if view == nil {
			view = &SingleBufferView{Buf: win.Buffer}
		}
		xOff, paneW := view.CursorRegion(win.Width)
		sub := newWindowScreen(e.screen, win.X+xOff, win.Y, paneW, win.Height, false)
		e.screen = sub
	}

	// Ensure we restore the real screen after computing/mutating view
	defer func() { e.screen = realScreen }()

	// Visual-space parameters for margins
	visibleHeight := e.getVisibleHeight()
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Context lines reduce the effective top scroll offset
	ctx := 0
	if e.CurrentBuffer() != nil {
		ctx = len(e.CurrentBuffer().ContextCache)
	}
	topScrollOffset := e.config.ScrollOffset - ctx
	if topScrollOffset < 0 {
		topScrollOffset = 0
	}
	bottomScrollOffset := e.config.ScrollOffset

	// For terminal buffers in terminal mode, disable scroll offsets entirely.
	// The emulator is actively driving the view (e.g., Ctrl+L clear), so editor
	// margins would fight the terminal UX. When browsing scrollback in normal
	// mode we restore the usual offsets so scrolling behaves like other buffers.
	if e.CurrentBuffer() != nil && e.mode == commands.TerminalMode {
		if _, ok := e.terminalSessions[e.CurrentBuffer().FilePath]; ok {
			topScrollOffset = 0
			bottomScrollOffset = 0
		}
	}

	// Fast-path logical alignment for the case where PageUp/PageDown jumped
	// the cursor outside the current view: if the cursor's logical line is
	// above the current top, reposition the view top near the cursor first,
	// then apply wrap-aware fine adjustments below. This prevents a frame
	// where content appears stuck while status/context move.
	topLine := e.CurrentBuffer().ViewIterator.LineNumber()
	cursorLine := e.CurrentBuffer().CursorIterator.LineNumber()
	if cursorLine < topLine {
		desiredTop := cursorLine - topScrollOffset
		if desiredTop < 0 {
			desiredTop = 0
		}
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), desiredTop)
	}

	// Compute current visual cursor row (relative to view) using wrapping
	visCursorY, _ := e.GetCursorVisualPosition()

	// Scroll up to satisfy top margin in visual rows
	if visCursorY < topScrollOffset {
		moveBatch := e.perfTracker.Start()
		// Move view up one logical line at a time until the margin is satisfied
		// or we reach the beginning of the buffer.
		for steps := 0; steps < 4096; steps++ { // safety bound
			if visCursorY >= topScrollOffset {
				break
			}
			// Do not move above the first line
			if e.CurrentBuffer().ViewIterator.LineNumber() == 0 {
				break
			}
			step := e.perfTracker.Start()
			e.CurrentBuffer().ViewIterator = e.CurrentBuffer().ViewIterator.Previous()
			_ = e.perfTracker.ReportSince("scroll.step.prev", step)
			// Recompute visual cursor
			visCursorY, _ = e.GetCursorVisualPosition()
		}
		_ = e.perfTracker.ReportSince("scroll.move.up.batch", moveBatch)
	}

	// Scroll down to satisfy bottom margin in visual rows
	if visCursorY >= visibleHeight-bottomScrollOffset {
		// Compute wrap-aware maximum view top: walk backwards from the last
		// line, counting visual rows per line, until we've accumulated enough
		// to fill the visible height. This replaces the old logical-line-based
		// maxTop which broke when lines wrapped to multiple visual rows.
		maxTop := 0
		{
			w, _ := e.screen.Size()
			gw := e.CurrentBuffer().LineNumberWidth
			wc := NewWrapCalculator(w, gw)
			isInternal := e.CurrentBuffer().IsInternal
			accum := 0
			lastLine := e.CurrentBuffer().LineCount - 1
			if lastLine < 0 {
				lastLine = 0
			}
			mIter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), lastLine)
			for {
				nRows := 1
				if !isInternal {
					nRows = len(e.wrapLineForDisplay(wc, e.CurrentBuffer(), mIter.Line(), mIter.Position(), mIter.LineNumber()))
				}
				accum += nRows
				if accum >= visibleHeight {
					maxTop = mIter.LineNumber()
					break
				}
				if mIter.LineNumber() == 0 {
					break
				}
				mIter = mIter.Previous()
			}
		}

		moveBatch := e.perfTracker.Start()
		for steps := 0; steps < 4096; steps++ { // safety bound
			if visCursorY < visibleHeight-bottomScrollOffset {
				break
			}
			if e.CurrentBuffer().ViewIterator.LineNumber() >= maxTop {
				break
			}
			step := e.perfTracker.Start()
			e.CurrentBuffer().ViewIterator = e.CurrentBuffer().ViewIterator.Next()
			_ = e.perfTracker.ReportSince("scroll.step.next", step)
			visCursorY, _ = e.GetCursorVisualPosition()
		}
		_ = e.perfTracker.ReportSince("scroll.move.down.batch", moveBatch)
	}
	e.perfTracker.ReportSince("scroll.core", t0)
}

// scrollToShowAtTop positions the view so the cursor line appears at ScrollOffset from the top
func (e *Editor) scrollToShowAtTop() {
	// Create a new iterator at the beginning
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIterator(e.CurrentBuffer())

	// Adjust scroll offset for top margin only (to account for context lines)
	ctx := 0
	if e.CurrentBuffer() != nil {
		ctx = len(e.CurrentBuffer().ContextCache)
	}
	topScrollOffset := e.config.ScrollOffset - ctx
	if topScrollOffset < 0 {
		topScrollOffset = 0
	}

	// Move down to position cursor at ScrollOffset from the top
	targetLine := e.CurrentBuffer().CursorIterator.LineNumber() - topScrollOffset
	if targetLine < 0 {
		targetLine = 0
	}

	// Move iterator to target line
	for i := 0; i < targetLine && !e.CurrentBuffer().ViewIterator.IsEOF(); i++ {
		e.CurrentBuffer().ViewIterator = e.CurrentBuffer().ViewIterator.Next()
	}
}

// ScrollToShowAtBottom positions the view so the cursor line appears at the bottom of the screen.
// It walks backwards from the cursor line, counting visual rows consumed by
// wrapped lines, to find the view top that places the cursor near the bottom.
func (e *Editor) ScrollToShowAtBottom() {
	visibleHeight := e.getVisibleHeight()
	cursorLine := e.CurrentBuffer().CursorIterator.LineNumber()

	// Determine effective width for wrap calculations.
	width, _ := e.screen.Size()
	if _, isWindowScreen := e.screen.(*windowScreen); !isWindowScreen {
		if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
			width = e.windows[e.activeWindow].Width
			if _, isDiff := e.windows[e.activeWindow].View.(*DiffBufferView); isDiff {
				width = width / 2
			}
		}
	}
	gutterWidth := e.CurrentBuffer().LineNumberWidth
	wc := NewWrapCalculator(width, gutterWidth)

	// Walk backwards from the cursor line, accumulating visual rows,
	// until we've filled the visible height.
	visualRows := 0
	iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), cursorLine)
	desiredViewLine := 0

	for {
		lineText := iter.Line()
		wrapped := wc.WrapLine(lineText, iter.LineNumber())
		visualRows += len(wrapped)

		if visualRows >= visibleHeight || iter.LineNumber() == 0 {
			desiredViewLine = iter.LineNumber()
			break
		}

		iter = iter.Previous()
	}

	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), desiredViewLine)

	LL.Debug("After going to the end, the view iterator is at %+v, cursor iterator at %+v, visible height is %d",
		e.CurrentBuffer().ViewIterator, e.CurrentBuffer().CursorIterator, visibleHeight)
}

// scrollToCenter positions the view so the cursor line appears in the middle of the screen
func (e *Editor) scrollToCenter() {
	visibleHeight := e.getVisibleHeight()
	cursorLine := e.CurrentBuffer().CursorIterator.LineNumber()

	// Calculate the desired view start line to center the cursor
	desiredViewLine := cursorLine - visibleHeight/2

	// Handle edge cases
	if desiredViewLine < 0 {
		desiredViewLine = 0
	} else {
		// Don't scroll past the end of the buffer
		maxViewLine := e.CurrentBuffer().LineCount - visibleHeight
		if maxViewLine < 0 {
			maxViewLine = 0
		}
		if desiredViewLine > maxViewLine {
			desiredViewLine = maxViewLine
		}
	}

	// Create iterator at the desired view line
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), desiredViewLine)
}
