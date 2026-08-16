package editor

import (
	"fmt"
	"veda/internal/buffer"
)

// BufferView is the interface for rendering a window's content.
// A SingleBufferView renders one buffer; a DiffBufferView renders two panes.
type BufferView interface {
	// Render draws this view into the given sub-screen for the specified window.
	Render(e *Editor, win *Window, sub Screen)
	// ActiveBuffer returns the buffer that currently has focus.
	ActiveBuffer() *buffer.Buffer
	// AllBuffers returns all buffers owned by this view.
	AllBuffers() []*buffer.Buffer
	// IsDiff returns true if this is a diff view.
	IsDiff() bool
	// DisplayName returns a human-readable name for the buffer list.
	DisplayName() string
	// ContainsBuffer returns true if this view owns the given buffer.
	ContainsBuffer(buf *buffer.Buffer) bool
	// ViewPath returns a stable identifier path for this view (used for
	// deduplication and matching in the buffer list).
	ViewPath() string
	// SwitchPane cycles to the next pane within this view.
	// Returns true if the view has multiple panes and switched, false otherwise.
	SwitchPane() bool
	// CursorRegion returns the x-offset and width of the active pane within
	// the window.  For single-buffer views this is the full window width.
	CursorRegion(windowWidth int) (xOffset, width int)
}

// renderBufferToScreen renders a single buffer into the given sub-screen.
// It swaps the editor's screen and current buffer, calls renderPane + drawContext,
// then restores. viewLine, cursorLine, cursorX specify the viewport/cursor state.
// Returns the updated (viewLine, cursorLine, cursorX) after rendering.
func renderBufferToScreen(e *Editor, buf *buffer.Buffer, sub Screen, viewLine, cursorLine, cursorX int) (int, int, int) {
	if buf == nil {
		return viewLine, cursorLine, cursorX
	}

	// Save editor-level state
	realScreen := e.screen
	realBuffer := e.CurrentBuffer()
	e.screen = sub

	// Save buffer iterators
	savedView := buf.ViewIterator
	savedCursor := buf.CursorIterator
	savedX := buf.CursorX

	// Apply state to buffer
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, max(0, viewLine))
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, max(0, cursorLine))
	buf.CursorX = cursorX
	e.SetCurrentBuffer(buf)

	// Ensure per-buffer highlighter is initialized and parsed
	if buf.Highlighter != nil {
		if buf.IsInternal {
			if buf.Highlighter.GetTree() == nil {
				_ = buf.Highlighter.SetLanguage("markdown")
				buf.Highlighter.ParseFull([]byte(buf.Rope.String()))
			}
		} else if buf.HasFile() {
			if buf.Highlighter.GetTree() == nil {
				content := []byte(buf.Rope.String())
				_ = buf.Highlighter.SetLanguageFromFileAndContent(buf.FilePath, content)
				buf.Highlighter.ParseFull(content)
			}
		}
	}

	// Render
	e.renderPane()
	e.drawContext()

	// Capture updated state
	newViewLine := buf.ViewIterator.LineNumber()
	newCursorLine := buf.CursorIterator.LineNumber()
	newCursorX := buf.CursorX

	// Restore buffer iterators (if still valid for current buffer generation)
	if savedView != nil && savedView.Generation == buf.Generation {
		buf.ViewIterator = savedView
	}
	if savedCursor != nil && savedCursor.Generation == buf.Generation {
		buf.CursorIterator = savedCursor
	}
	buf.CursorX = savedX

	// Restore editor-level state
	e.screen = realScreen
	e.SetCurrentBuffer(realBuffer)

	return newViewLine, newCursorLine, newCursorX
}

// SingleBufferView renders a single buffer in a window.
type SingleBufferView struct {
	Buf *buffer.Buffer
}

func (v *SingleBufferView) ActiveBuffer() *buffer.Buffer {
	return v.Buf
}

func (v *SingleBufferView) AllBuffers() []*buffer.Buffer {
	return []*buffer.Buffer{v.Buf}
}

func (v *SingleBufferView) IsDiff() bool {
	return false
}

func (v *SingleBufferView) DisplayName() string {
	if v.Buf != nil {
		return v.Buf.DisplayName()
	}
	return ""
}

func (v *SingleBufferView) ContainsBuffer(buf *buffer.Buffer) bool {
	return v.Buf == buf
}

func (v *SingleBufferView) ViewPath() string {
	if v.Buf != nil {
		return v.Buf.FilePath
	}
	return ""
}

func (v *SingleBufferView) SwitchPane() bool {
	return false
}

func (v *SingleBufferView) CursorRegion(windowWidth int) (int, int) {
	return 0, windowWidth
}

func (v *SingleBufferView) Render(e *Editor, win *Window, sub Screen) {
	vl, cl, cx := renderBufferToScreen(e, v.Buf, sub, win.ViewLine, win.CursorLine, win.CursorX)
	win.ViewLine = vl
	win.CursorLine = cl
	win.CursorX = cx
}

// DiffBufferView renders two buffers side-by-side in a single window.
type DiffBufferView struct {
	LeftBuffer  *buffer.Buffer // HEAD, read-only
	RightBuffer *buffer.Buffer // current working copy
	ActivePane  int            // 0=left, 1=right
	RealPath    string         // original file path
}

func (v *DiffBufferView) ActiveBuffer() *buffer.Buffer {
	if v.ActivePane == 0 {
		return v.LeftBuffer
	}
	return v.RightBuffer
}

func (v *DiffBufferView) InactiveBuffer() *buffer.Buffer {
	if v.ActivePane == 0 {
		return v.RightBuffer
	}
	return v.LeftBuffer
}

func (v *DiffBufferView) AllBuffers() []*buffer.Buffer {
	return []*buffer.Buffer{v.LeftBuffer, v.RightBuffer}
}

func (v *DiffBufferView) IsDiff() bool {
	return true
}

func (v *DiffBufferView) DisplayName() string {
	return v.RealPath + " [diff]"
}

func (v *DiffBufferView) ContainsBuffer(buf *buffer.Buffer) bool {
	return v.LeftBuffer == buf || v.RightBuffer == buf
}

func (v *DiffBufferView) ViewPath() string {
	return v.RealPath
}

// SyncCursors copies cursor Y and view line from active pane to inactive pane.
func (v *DiffBufferView) SyncCursors() {
	active := v.ActiveBuffer()
	inactive := v.InactiveBuffer()
	if active == nil || inactive == nil {
		return
	}

	cursorLine := active.CursorIterator.LineNumber()
	viewLine := active.ViewIterator.LineNumber()

	inactive.CursorIterator = buffer.NewBufferIteratorAtLine(inactive, cursorLine)
	inactive.ViewIterator = buffer.NewBufferIteratorAtLine(inactive, viewLine)
}

func (v *DiffBufferView) CursorRegion(windowWidth int) (int, int) {
	leftW := windowWidth / 2
	if v.ActivePane == 0 {
		return 0, leftW
	}
	return leftW, windowWidth - leftW
}

// SwitchPane toggles the active pane between left and right.
func (v *DiffBufferView) SwitchPane() bool {
	if v.ActivePane == 0 {
		v.ActivePane = 1
	} else {
		v.ActivePane = 0
	}
	return true
}

// ResolveDiffLine maps an aligned buffer row to the real working-file line number.
// Returns the 0-based line number and nil, or -1 and an error for filler/invalid lines.
func (v *DiffBufferView) ResolveDiffLine(row int) (int, error) {
	buf := v.RightBuffer
	if buf == nil || buf.DiffHints == nil {
		return -1, fmt.Errorf("no diff hints")
	}
	if row < 0 || row >= len(buf.DiffHints) {
		return -1, fmt.Errorf("row %d out of range", row)
	}
	dl := buf.DiffHints[row]
	if dl.IsFillerLine {
		return -1, fmt.Errorf("filler line")
	}
	return dl.RightLine, nil
}

// Render draws the left and right diff panes side by side.
func (v *DiffBufferView) Render(e *Editor, win *Window, sub Screen) {
	totalW, totalH := sub.Size()
	if totalW < 2 || totalH < 1 {
		return
	}

	// Split width: left gets half (rounding down), right gets the rest
	leftW := totalW / 2
	rightW := totalW - leftW

	// Create sub-screens for each pane. Neither pane is dimmed.
	leftSub := newWindowScreen(sub, 0, 0, leftW, totalH, false)
	rightSub := newWindowScreen(sub, leftW, 0, rightW, totalH, false)

	// Both panes share the same view/cursor Y. The window state tracks the active pane.
	viewLine := win.ViewLine
	cursorLine := win.CursorLine

	// Render left pane (HEAD)
	renderBufferToScreen(e, v.LeftBuffer, leftSub, viewLine, cursorLine, 0)

	// Render right pane (current) — this is the active pane, so persist state back
	vl, cl, cx := renderBufferToScreen(e, v.RightBuffer, rightSub, viewLine, cursorLine, win.CursorX)
	win.ViewLine = vl
	win.CursorLine = cl
	win.CursorX = cx
}
