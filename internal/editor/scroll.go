package editor

import (
	"veda/internal/buffer"
)

// scroll ensures the cursor is visible within the viewport with appropriate margins
func (e *Editor) scroll() {
	t0 := e.perfTracker.Start()

	// Skip scrolling for empty windows
	if e.buffer == nil {
		return
	}

	visibleHeight := e.getVisibleHeight()
	bufferLines := e.buffer.LineCount
	topLine := e.buffer.ViewIterator.LineNumber()

	// Adjust scroll offset for top margin only (to account for context lines)
    ctx := 0
    if e.buffer != nil {
        ctx = len(e.buffer.ContextCache)
    }
    topScrollOffset := e.config.ScrollOffset - ctx
	if topScrollOffset < 0 {
		topScrollOffset = 0
	}

	// Scroll up when cursor is too close to top
	cursorLine := e.buffer.CursorIterator.LineNumber()
	if cursorLine < topLine+topScrollOffset {
		linesToMove := topLine - (cursorLine - topScrollOffset)
		if linesToMove > 0 {
			for i := 0; i < linesToMove; i++ {
				e.buffer.ViewIterator = e.buffer.ViewIterator.Previous()
			}
		}
	}

	// Scroll down when cursor is too close to bottom, but allow cursor to reach
	// the bottom of the screen when at the end of the buffer
	// Bottom scroll offset remains unchanged
	if cursorLine >= topLine+visibleHeight-e.config.ScrollOffset {
		// If we're near the end of the buffer, allow the last line to be at the bottom
		maxTopLine := bufferLines - visibleHeight
		if maxTopLine < 0 {
			maxTopLine = 0
		}

		desiredTopLine := cursorLine - visibleHeight + e.config.ScrollOffset
		linesToMove := desiredTopLine - topLine

		// Don't scroll past the maximum allowed top line
		if topLine+linesToMove > maxTopLine {
			linesToMove = maxTopLine - topLine
		}

		if linesToMove > 0 {
			for i := 0; i < linesToMove; i++ {
				e.buffer.ViewIterator = e.buffer.ViewIterator.Next()
			}
		}
	}
	e.perfTracker.ReportSince("scroll.core", t0)
}

// scrollToShowAtTop positions the view so the cursor line appears at ScrollOffset from the top
func (e *Editor) scrollToShowAtTop() {
	// Create a new iterator at the beginning
	e.buffer.ViewIterator = buffer.NewBufferIterator(e.buffer)

	// Adjust scroll offset for top margin only (to account for context lines)
    ctx := 0
    if e.buffer != nil {
        ctx = len(e.buffer.ContextCache)
    }
    topScrollOffset := e.config.ScrollOffset - ctx
	if topScrollOffset < 0 {
		topScrollOffset = 0
	}

	// Move down to position cursor at ScrollOffset from the top
	targetLine := e.buffer.CursorIterator.LineNumber() - topScrollOffset
	if targetLine < 0 {
		targetLine = 0
	}

	// Move iterator to target line
	for i := 0; i < targetLine && !e.buffer.ViewIterator.IsEOF(); i++ {
		e.buffer.ViewIterator = e.buffer.ViewIterator.Next()
	}
}

// ScrollToShowAtBottom positions the view so the cursor line appears at the bottom of the screen
// This method calculates the view position so that the cursor is on the last visible line
func (e *Editor) ScrollToShowAtBottom() {
	visibleHeight := e.getVisibleHeight()
	cursorLine := e.buffer.CursorIterator.LineNumber()

	// Calculate the desired view start line
	// If cursor is below screen, we need to move view up by 1
	// View should start at: cursorLine - visibleHeight + 1 - 1
	desiredViewLine := cursorLine - visibleHeight + 2

	// Handle edge case where desired view would be negative
	if desiredViewLine < 0 {
		desiredViewLine = 0
	}

	// Create iterator at the desired view line
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, desiredViewLine)

	LL.Debug("After going to the end, the view iterator is at %+v, cursor iterator at %+v, visible height is %d",
		e.buffer.ViewIterator, e.buffer.CursorIterator, visibleHeight)
}

// scrollToCenter positions the view so the cursor line appears in the middle of the screen
func (e *Editor) scrollToCenter() {
	visibleHeight := e.getVisibleHeight()
	cursorLine := e.buffer.CursorIterator.LineNumber()

	// Calculate the desired view start line to center the cursor
	desiredViewLine := cursorLine - visibleHeight/2

	// Handle edge cases
	if desiredViewLine < 0 {
		desiredViewLine = 0
	} else {
		// Don't scroll past the end of the buffer
		maxViewLine := e.buffer.LineCount - visibleHeight
		if maxViewLine < 0 {
			maxViewLine = 0
		}
		if desiredViewLine > maxViewLine {
			desiredViewLine = maxViewLine
		}
	}

	// Create iterator at the desired view line
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, desiredViewLine)
}
