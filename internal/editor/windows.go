package editor

import (
	"fmt"
	"veda/internal/buffer"
	"veda/internal/events"
	"veda/internal/ui"
)

// Window represents a rectangular area of the screen showing a buffer
type Window struct {
	Buffer *buffer.Buffer
	X      int
	Y      int
	Width  int
	Height int
	// Per-window view and cursor state (independent even if Buffer is shared)
	ViewLine   int // Top visible logical line number
	CursorLine int // Cursor logical line number
	CursorX    int // Cursor X within the line
}

// windowScreen clips drawing to a window region and can dim output for inactive windows.
// It composes the base Screen without adding new interfaces.
type windowScreen struct {
	base Screen
	x, y int
	w, h int
	dim  bool
	// Optional hook to capture view/cursor changes during rendering
	onCursorUpdate func(viewLine, cursorLine, cursorX int)
}

func newWindowScreen(base Screen, x, y, w, h int, dim bool) *windowScreen {
	return &windowScreen{base: base, x: x, y: y, w: w, h: h, dim: dim}
}

// Lifecycle - unused for sub-screens
func (ws *windowScreen) Init() error { return nil }
func (ws *windowScreen) Fini()       {}

// Clear only clears within the window region
func (ws *windowScreen) Clear() { /* no-op; draw routines fill region explicitly */ }

// Show is handled by outer screen
func (ws *windowScreen) Show() {}

// Size returns the size of the window region
func (ws *windowScreen) Size() (int, int) { return ws.w, ws.h }

// SetStyle forwards to base
func (ws *windowScreen) SetStyle(style ui.Style) { ws.base.SetStyle(style) }

// SetContent applies clipping and optional dimming, then forwards to base coordinates
func (ws *windowScreen) SetContent(x, y int, mainc rune, combc []rune, style ui.Style) {
	if x < 0 || y < 0 || x >= ws.w || y >= ws.h {
		return
	}
	if ws.dim {
		style = dimStyle(style, 0.55)
	}
	ws.base.SetContent(ws.x+x, ws.y+y, mainc, combc, style)
}

// ShowCursor offsets into base coordinates
func (ws *windowScreen) ShowCursor(x, y int) { ws.base.ShowCursor(ws.x+x, ws.y+y) }

// SetCursorStyle forwards to base
func (ws *windowScreen) SetCursorStyle(style ui.CursorStyle) { ws.base.SetCursorStyle(style) }

// Event forwarding
func (ws *windowScreen) PollEvent() events.Event           { return ws.base.PollEvent() }
func (ws *windowScreen) EventChannel() <-chan events.Event { return ws.base.EventChannel() }

// activateWindow centralizes switching focus to a window index, enforcing layout and buffer focus
func (e *Editor) activateWindow(idx int) {
	if idx < 0 || idx >= len(e.windows) {
		return
	}
	if e.activeWindow != idx {
		e.lastWindow = e.activeWindow
		e.activeWindow = idx
	}
	e.recomputeWindowsLayout()
	if win := e.windows[idx]; win != nil && win.Buffer != nil {
		e.switchToBuffer(win.Buffer)
	} else {
		// When activating an empty window, clear the editor's buffer
		e.buffer = nil
	}
}

// recomputeWindowsLayout recalculates all window rectangles.
// It aims for equal widths, but guarantees the active window has at least
// minActiveWindowWidth columns without reducing it if it would naturally be wider.
func (e *Editor) recomputeWindowsLayout() {
	if len(e.windows) == 0 {
		return
	}
	screenW, screenH := e.screen.Size()
	// Reserve last line for status bar
	contentH := screenH - 1
	if contentH < 1 {
		contentH = 1
	}
	n := len(e.windows)

	// Single window: fill screen
	if n == 1 {
		e.windows[0].X = 0
		e.windows[0].Y = 0
		e.windows[0].Width = screenW
		e.windows[0].Height = contentH
		return
	}

	// Calculate minimum window width if configured
	var minW int
	if e.minActiveWindowWidth > 0 {
		// minActiveWindowWidth specifies buffer content columns
		// Add gutter width (line numbers + indicators) to get total window width
		gutterWidth := 0
		if e.buffer != nil {
			gutterWidth = e.buffer.LineNumberWidth
		} else {
			// Fallback: assume typical gutter width for reasonable line counts
			gutterWidth = 7 // 4 digits + 1 padding + 2 indicators
		}
		minW = e.minActiveWindowWidth + gutterWidth

		if minW > screenW-(n-1) {
			// leave at least 1 column for others total
			minW = screenW - (n - 1)
			if minW < 1 {
				minW = 1
			}
		}
	}

	activeIdx := e.activeWindow
	if activeIdx < 0 || activeIdx >= n {
		activeIdx = 0
	}

	// Start with equal distribution across all windows
	widths := make([]int, n)
	base := screenW / n
	extra := screenW % n
	for i := 0; i < n; i++ {
		w := base
		if i < extra {
			w++
		}
		widths[i] = w
	}

	// If minimum width is configured and the active window is narrower than the minimum,
	// enlarge it by borrowing columns from other windows (preferentially from widest ones).
	if minW > 0 && widths[activeIdx] < minW {
		need := minW - widths[activeIdx]
		widths[activeIdx] = minW

		// Borrow one column at a time from other windows in rounds to keep balance
		for need > 0 {
			// Find a donor window with the largest current width (excluding active)
			donor := -1
			donorWidth := -1
			for i := 0; i < n; i++ {
				if i == activeIdx {
					continue
				}
				if widths[i] > donorWidth && widths[i] > 1 { // keep at least 1 col
					donor = i
					donorWidth = widths[i]
				}
			}
			if donor == -1 {
				// No donor available (should not happen due to minW clamp), stop
				break
			}
			widths[donor]--
			need--
		}
	}

	x := 0
	for i := 0; i < n; i++ {
		e.windows[i].X = x
		e.windows[i].Y = 0
		e.windows[i].Width = widths[i]
		e.windows[i].Height = contentH
		x += widths[i]
	}
}

// dimStyle produces a dimmer version of a style by squashing RGB channels toward
// their average. This visually reduces contrast without needing new theme styles.
func dimStyle(s ui.Style, factor float32) ui.Style {
	fg, bg, attrs := s.Decompose()
	s.Fg = dimColor(fg, factor)
	s.Bg = dimColor(bg, factor)
	s.Attrs = attrs
	return s
}

func dimColor(c ui.Color, factor float32) ui.Color {
	if c == ui.ColorDefault {
		return c
	}
	r, g, b := c.RGB()
	nr := clamp255(float32(r) * factor)
	ng := clamp255(float32(g) * factor)
	nb := clamp255(float32(b) * factor)
	return ui.NewRGBColor(int32(nr), int32(ng), int32(nb))
}

func clamp255(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// createUnnamedBuffer creates and registers a new empty buffer without switching to it.
func (e *Editor) createUnnamedBuffer() *buffer.Buffer {
	// Generate a unique buffer name
	bufferName := ""
	counter := 2 // Start from 2 since [No Name] is already used
	for {
		bufferName = fmt.Sprintf("[No Name %d]", counter)
		if _, exists := e.buffers[bufferName]; !exists {
			break
		}
		counter++
	}
	newBuf := buffer.NewBuffer("", bufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
	e.buffers[bufferName] = newBuf
	e.bufferOrder = append(e.bufferOrder, bufferName)
	return newBuf
}

// vsplit splits the current window into two and opens a new empty buffer in the new window.
func (e *Editor) vsplit(args []string) error {
	// Create new buffer for the new window
	rightBuf := e.createUnnamedBuffer()

	// Insert new window to the right of the active window
	insertAt := e.activeWindow + 1
	newWin := newWindow(rightBuf)
	if insertAt >= len(e.windows) {
		e.windows = append(e.windows, newWin)
	} else {
		e.windows = append(e.windows[:insertAt+1], e.windows[insertAt:]...)
		e.windows[insertAt] = newWin
	}

	// Record last window as the one we are splitting from, then make new window active
	e.lastWindow = e.activeWindow
	e.activeWindow = insertAt
	e.switchToBuffer(rightBuf)

	// Recompute layout
	e.recomputeWindowsLayout()

	return nil
}

// switchToWindow switches focus to window at 1-based index n
func (e *Editor) switchToWindow(n int) error {
	if len(e.windows) == 0 {
		return nil
	}
	idx := n - 1
	e.activateWindow(idx)
	return nil
}

// switchToLastWindow switches to the last window (rightmost)
func (e *Editor) switchToLastWindow(args []string) error {
	if len(e.windows) == 0 {
		return nil
	}
	if e.lastWindow < 0 || e.lastWindow >= len(e.windows) {
		return nil
	}
	// Toggle to last window without updating lastWindow to itself
	current := e.activeWindow
	target := e.lastWindow
	if target == current {
		return nil
	}
	// Perform switch and set lastWindow to previous current
	e.activeWindow = target
	e.lastWindow = current
	// Enforce min width for the newly active window
	e.recomputeWindowsLayout()
	if win := e.windows[target]; win != nil && win.Buffer != nil {
		e.switchToBuffer(win.Buffer)
	}
	return nil
}

// switchToNextWindow cycles focus to the next window (wraps around)
func (e *Editor) switchToNextWindow(args []string) error {
	n := len(e.windows)
	if n <= 1 {
		return nil
	}
	next := (e.activeWindow + 1) % n
	e.activateWindow(next)
	return nil
}

// moveBufferLeft moves the active window's buffer to the immediate left window,
// creating a new window if none exists to the left
func (e *Editor) moveBufferLeft(args []string) error {
	cur := e.activeWindow
	currentBuffer := e.windows[cur].Buffer

	// If no window to the left, create one
	if cur <= 0 {
		// Create new window to the left
		newWin := newWindow(currentBuffer)

		// Copy view state to new window
		newWin.ViewLine = e.windows[cur].ViewLine
		newWin.CursorLine = e.windows[cur].CursorLine
		newWin.CursorX = e.windows[cur].CursorX

		// Insert at the beginning
		e.windows = append([]*Window{newWin}, e.windows...)

		// Leave original window empty
		// (cur is now at index cur+1 after insertion)
		e.windows[cur+1].Buffer = nil
		e.windows[cur+1].ViewLine = 0
		e.windows[cur+1].CursorLine = 0
		e.windows[cur+1].CursorX = 0

		// Activate the new left window (index 0)
		e.activateWindow(0)
	} else {
		// Move to existing left window
		left := cur - 1

		// Store the buffer from the left window (to swap if needed)
		leftBuffer := e.windows[left].Buffer

		// Move current buffer to left window
		e.windows[left].Buffer = currentBuffer
		e.windows[left].ViewLine = e.windows[cur].ViewLine
		e.windows[left].CursorLine = e.windows[cur].CursorLine
		e.windows[left].CursorX = e.windows[cur].CursorX

		// Put the left window's buffer in current window (or leave empty if it was the same)
		if leftBuffer == currentBuffer {
			e.windows[cur].Buffer = nil
		} else {
			e.windows[cur].Buffer = leftBuffer
		}
		e.windows[cur].ViewLine = 0
		e.windows[cur].CursorLine = 0
		e.windows[cur].CursorX = 0

		// Activate the left window
		e.activateWindow(left)
	}

	e.recomputeWindowsLayout()
	return nil
}

// moveBufferRight moves the active window's buffer to the immediate right window,
// creating a new window if none exists to the right
func (e *Editor) moveBufferRight(args []string) error {
	cur := e.activeWindow
	currentBuffer := e.windows[cur].Buffer

	// If no window to the right, create one
	if cur >= len(e.windows)-1 {
		// Create new window to the right
		newWin := newWindow(currentBuffer)
		e.windows = append(e.windows, newWin)

		// Copy view state to new window
		newWin.ViewLine = e.windows[cur].ViewLine
		newWin.CursorLine = e.windows[cur].CursorLine
		newWin.CursorX = e.windows[cur].CursorX

		// Leave current window empty
		e.windows[cur].Buffer = nil
		e.windows[cur].ViewLine = 0
		e.windows[cur].CursorLine = 0
		e.windows[cur].CursorX = 0

		// Activate the new right window
		e.activateWindow(len(e.windows) - 1)
	} else {
		// Move to existing right window
		right := cur + 1

		// Store the buffer from the right window (to swap if needed)
		rightBuffer := e.windows[right].Buffer

		// Move current buffer to right window
		e.windows[right].Buffer = currentBuffer
		e.windows[right].ViewLine = e.windows[cur].ViewLine
		e.windows[right].CursorLine = e.windows[cur].CursorLine
		e.windows[right].CursorX = e.windows[cur].CursorX

		// Put the right window's buffer in current window (or leave empty if it was the same)
		if rightBuffer == currentBuffer {
			e.windows[cur].Buffer = nil
		} else {
			e.windows[cur].Buffer = rightBuffer
		}
		e.windows[cur].ViewLine = 0
		e.windows[cur].CursorLine = 0
		e.windows[cur].CursorX = 0

		// Activate the right window
		e.activateWindow(right)
	}

	e.recomputeWindowsLayout()
	return nil
}

// moveBufferCenter moves the active buffer to the center window index
// For N windows, center = N/2 (0-based). For 2 windows, that's 1 -> which is visually middle-right,
// but per request we want the left window; so we special-case N==2 -> index 0.
func (e *Editor) moveBufferCenter(args []string) error {
	n := len(e.windows)
	if n == 0 {
		return nil
	}
	var center int
	if n == 2 {
		center = 0
	} else {
		center = n / 2
	}
	cur := e.activeWindow
	if cur == center {
		return nil
	}

	currentBuffer := e.windows[cur].Buffer
	centerBuffer := e.windows[center].Buffer

	// Move current buffer to center window
	e.windows[center].Buffer = currentBuffer
	e.windows[center].ViewLine = e.windows[cur].ViewLine
	e.windows[center].CursorLine = e.windows[cur].CursorLine
	e.windows[center].CursorX = e.windows[cur].CursorX

	// Put the center window's buffer in current window (or leave empty if it was the same)
	if centerBuffer == currentBuffer {
		e.windows[cur].Buffer = nil
	} else {
		e.windows[cur].Buffer = centerBuffer
	}
	e.windows[cur].ViewLine = 0
	e.windows[cur].CursorLine = 0
	e.windows[cur].CursorX = 0

	e.activateWindow(center)
	e.recomputeWindowsLayout()
	return nil
}

// newWindow creates a new window with initialized buffer management
func newWindow(buf *buffer.Buffer) *Window {
	window := &Window{
		Buffer:            buf,
		ViewLine:          0,
		CursorLine:        0,
		CursorX:           0,
	}

	return window
}

// closeWindow removes a window and adjusts the active window index
func (e *Editor) closeWindow(windowIndex int) error {
	if windowIndex < 0 || windowIndex >= len(e.windows) {
		return nil
	}

	// Don't close the last window
	if len(e.windows) <= 1 {
		return nil
	}

	// Remove the window from the slice
	e.windows = append(e.windows[:windowIndex], e.windows[windowIndex+1:]...)

	// Adjust active window index
	if e.activeWindow >= windowIndex {
		e.activeWindow--
		if e.activeWindow < 0 {
			e.activeWindow = 0
		}
	}

	// Make sure we have a valid active window
	if e.activeWindow >= len(e.windows) {
		e.activeWindow = len(e.windows) - 1
	}

	// Update the editor's current buffer to match the new active window
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
		if e.windows[e.activeWindow] != nil {
			e.buffer = e.windows[e.activeWindow].Buffer
		}
	}

	return nil
}

// closeCurrentWindow closes the currently active window
func (e *Editor) closeCurrentWindow(args []string) error {
	return e.closeWindow(e.activeWindow)
}
