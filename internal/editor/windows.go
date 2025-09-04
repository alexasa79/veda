package editor

import (
	"fmt"
	"veda/internal/buffer"
	"veda/internal/events"
	"veda/internal/ui"
)

// Window represents a rectangular area of the screen showing a buffer
type Window struct {
	Buffer  *buffer.Buffer   // Current active buffer in the window
	Buffers []*buffer.Buffer // All buffers in this window
	X       int
	Y       int
	Width   int
	Height  int
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
		e.SetCurrentBuffer(nil)
	}
}

// recomputeWindowsLayout recalculates all window rectangles.
// It aims for equal widths, but guarantees the active window has at least
// minActiveWindowWidth columns without reducing it if it would naturally be wider.
// When windowFullscreen is true, the active window gets most of the screen width.
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

	// Handle fullscreen mode
	if e.windowFullscreen {
		activeIdx := e.activeWindow
		if activeIdx < 0 || activeIdx >= n {
			activeIdx = 0
		}

		// Give 10 characters to each non-active window
		minOtherWidth := 10
		otherWindowsTotal := minOtherWidth * (n - 1)

		// Active window gets the rest
		activeWidth := screenW - otherWindowsTotal
		if activeWidth < 20 { // Ensure active window gets at least 20 columns
			activeWidth = 20
			// Recalculate other windows if needed
			otherWindowsTotal = screenW - activeWidth
			if otherWindowsTotal > 0 && n > 1 {
				minOtherWidth = otherWindowsTotal / (n - 1)
				if minOtherWidth < 1 {
					minOtherWidth = 1
				}
			}
		}

		// Set window positions and sizes
		x := 0
		for i := 0; i < n; i++ {
			e.windows[i].X = x
			e.windows[i].Y = 0
			e.windows[i].Height = contentH

			if i == activeIdx {
				e.windows[i].Width = activeWidth
				x += activeWidth
			} else {
				width := minOtherWidth
				// Distribute any remaining space among non-active windows
				if i == n-1 {
					// Last window gets any remaining space
					width = screenW - x
				}
				e.windows[i].Width = width
				x += width
			}
		}
		return
	}

	// Check if we're in diff view mode
	diffLeftIdx, diffRightIdx := e.findDiffWindowPair()
	isDiffMode := diffLeftIdx >= 0 && diffRightIdx >= 0

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

	if isDiffMode {
		// In diff mode, make the two diff windows equal and wider than others
		// Calculate how much extra space we can give to diff windows
		minOtherWidth := 20   // Minimum width for non-diff windows
		nonDiffCount := n - 2 // Number of non-diff windows

		if nonDiffCount > 0 {
			// Calculate space available for diff windows after giving minimum to others
			spaceForOthers := minOtherWidth * nonDiffCount
			spaceForDiff := screenW - spaceForOthers

			if spaceForDiff >= 40 { // Ensure diff windows get at least 20 cols each
				// Each diff window gets half of the remaining space
				diffWidth := spaceForDiff / 2
				extraDiff := spaceForDiff % 2

				// Set diff window widths
				widths[diffLeftIdx] = diffWidth
				widths[diffRightIdx] = diffWidth
				if extraDiff > 0 {
					// Give extra column to one of the diff windows
					widths[diffLeftIdx]++
				}

				// Distribute remaining space among non-diff windows
				remainingSpace := screenW - widths[diffLeftIdx] - widths[diffRightIdx]
				if nonDiffCount > 0 {
					otherBase := remainingSpace / nonDiffCount
					otherExtra := remainingSpace % nonDiffCount
					otherIdx := 0
					for i := 0; i < n; i++ {
						if i != diffLeftIdx && i != diffRightIdx {
							widths[i] = otherBase
							if otherIdx < otherExtra {
								widths[i]++
							}
							otherIdx++
						}
					}
				}
			}
			// If not enough space, fall back to equal distribution (already set above)
		} else {
			// Only two windows and both are diff windows - make them equal
			widths[diffLeftIdx] = screenW / 2
			widths[diffRightIdx] = screenW - widths[diffLeftIdx]
		}
	} else {
		// Not in diff mode - apply minimum width for active window if configured
		activeIdx := e.activeWindow
		if activeIdx < 0 || activeIdx >= n {
			activeIdx = 0
		}

		var minW int
		if e.minActiveWindowWidth > 0 {
			// minActiveWindowWidth specifies buffer content columns
			// Add gutter width (line numbers + indicators) to get total window width
			gutterWidth := 0
			if e.CurrentBuffer() != nil {
				gutterWidth = e.CurrentBuffer().LineNumberWidth
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
	}

	x := 0
	for i := 0; i < n; i++ {
		e.windows[i].X = x
		e.windows[i].Y = 0
		e.windows[i].Width = widths[i]
		e.windows[i].Height = contentH
		x += widths[i]
	}

	// If an active terminal buffer exists, ensure the PTY matches the active window size
	if e.CurrentBuffer() != nil {
		if ts := e.terminalSessions[e.CurrentBuffer().FilePath]; ts != nil && ts.SessionID != "" {
			cols, rows := e.getActiveWindowSizeForTerminal()
			_ = e.proxyManager.ResizeTerminal(ts.WorkingDir, ts.SessionID, cols, rows)
			ts.Cols, ts.Rows = cols, rows
			if ts.term != nil {
				ts.term.Resize(cols, rows)
			}
		}
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
	// Prefer the base name "[No Name]" if not taken
	base := "[No Name]"
	if _, exists := e.buffers[base]; !exists {
		newBuf := buffer.NewBuffer("", base, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		e.buffers[base] = newBuf
		e.bufferOrder = append(e.bufferOrder, base)
		return newBuf
	}
	// Otherwise, find the next available suffix starting from 2
	counter := 2
	for {
		name := fmt.Sprintf("[No Name %d]", counter)
		if _, exists := e.buffers[name]; !exists {
			newBuf := buffer.NewBuffer("", name, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
			e.buffers[name] = newBuf
			e.bufferOrder = append(e.bufferOrder, name)
			return newBuf
		}
		counter++
	}
}

// vsplit splits the current window into two and opens a new empty buffer in the new window.
func (e *Editor) vsplit(args []string) error {
	// Insert a new empty window to the right of the active window (no auto-created buffer)
	insertAt := e.activeWindow + 1
	newWin := newWindow(nil)
	if insertAt >= len(e.windows) {
		e.windows = append(e.windows, newWin)
	} else {
		e.windows = append(e.windows[:insertAt+1], e.windows[insertAt:]...)
		e.windows[insertAt] = newWin
	}

	// Record last window as the one we are splitting from, then make new window active
	e.lastWindow = e.activeWindow
	e.activeWindow = insertAt

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

// createRightmostWindow appends a new window to the right of the rightmost window
// and makes it the active window. If seed is non-nil, it becomes the window's buffer.
func (e *Editor) createRightmostWindow(seed *buffer.Buffer) int {
	newWin := newWindow(seed)
	e.windows = append(e.windows, newWin)
	e.activateWindow(len(e.windows) - 1)
	return len(e.windows) - 1
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

		// Remove buffer from original window
		// (cur is now at index cur+1 after insertion)
		e.windows[cur+1].removeBufferFromWindow(currentBuffer)

		// Activate the new left window (index 0)
		e.activateWindow(0)
	} else {
		// Move to existing left window
		left := cur - 1

		// Add current buffer to left window
		e.windows[left].addBufferToWindow(currentBuffer)
		e.windows[left].ViewLine = e.windows[cur].ViewLine
		e.windows[left].CursorLine = e.windows[cur].CursorLine
		e.windows[left].CursorX = e.windows[cur].CursorX

		// Remove buffer from current window
		e.windows[cur].removeBufferFromWindow(currentBuffer)
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

		// Remove buffer from current window
		e.windows[cur].removeBufferFromWindow(currentBuffer)

		// Activate the new right window
		e.activateWindow(len(e.windows) - 1)
	} else {
		// Move to existing right window
		right := cur + 1

		// Add current buffer to right window
		e.windows[right].addBufferToWindow(currentBuffer)
		e.windows[right].ViewLine = e.windows[cur].ViewLine
		e.windows[right].CursorLine = e.windows[cur].CursorLine
		e.windows[right].CursorX = e.windows[cur].CursorX

		// Remove buffer from current window
		e.windows[cur].removeBufferFromWindow(currentBuffer)
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

	// Add current buffer to center window
	e.windows[center].addBufferToWindow(currentBuffer)
	e.windows[center].ViewLine = e.windows[cur].ViewLine
	e.windows[center].CursorLine = e.windows[cur].CursorLine
	e.windows[center].CursorX = e.windows[cur].CursorX

	// Remove buffer from current window
	e.windows[cur].removeBufferFromWindow(currentBuffer)
	e.windows[cur].ViewLine = 0
	e.windows[cur].CursorLine = 0
	e.windows[cur].CursorX = 0

	e.activateWindow(center)
	e.recomputeWindowsLayout()
	return nil
}

// toggleWindowFullscreen toggles fullscreen mode for the active window
func (e *Editor) toggleWindowFullscreen(args []string) error {
	// Only makes sense with multiple windows
	if len(e.windows) <= 1 {
		return nil
	}

	// Toggle the fullscreen state
	e.windowFullscreen = !e.windowFullscreen

	// Recompute layout with new fullscreen state
	e.recomputeWindowsLayout()

	return nil
}

// addBufferToWindow adds a buffer to a window's buffer list and makes it active
func (w *Window) addBufferToWindow(buf *buffer.Buffer) {
	if buf == nil {
		return
	}

	// Check if buffer is already in the window
	for _, existing := range w.Buffers {
		if existing == buf {
			w.Buffer = buf // Just make it active
			return
		}
	}

	// Add new buffer to the list and make it active
	w.Buffers = append(w.Buffers, buf)
	w.Buffer = buf
}

// removeBufferFromWindow removes a buffer from a window's buffer list
func (w *Window) removeBufferFromWindow(buf *buffer.Buffer) {
	if buf == nil {
		return
	}

	// Find and remove the buffer
	for i, existing := range w.Buffers {
		if existing == buf {
			// Remove from slice
			w.Buffers = append(w.Buffers[:i], w.Buffers[i+1:]...)

			// If this was the active buffer, switch to another or set to nil
			if w.Buffer == buf {
				if len(w.Buffers) > 0 {
					w.Buffer = w.Buffers[len(w.Buffers)-1] // Use the last buffer
				} else {
					w.Buffer = nil // No buffers left
				}
			}
			return
		}
	}
}

// newWindow creates a new window with initialized buffer management
func newWindow(buf *buffer.Buffer) *Window {
	var buffers []*buffer.Buffer
	if buf != nil {
		buffers = []*buffer.Buffer{buf}
	}

	window := &Window{
		Buffer:     buf,
		Buffers:    buffers,
		ViewLine:   0,
		CursorLine: 0,
		CursorX:    0,
	}

	return window
}

// closeWindow removes a window and adjusts the active window index
func (e *Editor) closeWindow(windowIndex int) error {
	if windowIndex < 0 || windowIndex >= len(e.windows) {
		return nil
	}

	// If this is the last window, close the editor
	if len(e.windows) <= 1 {
		return e.actuallyQuit(nil)
	}
	// Capture buffer associated with the window to determine if it should be closed
	var closedBuf *buffer.Buffer
	if e.windows[windowIndex] != nil {
		closedBuf = e.windows[windowIndex].Buffer
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
			// Buffer already synced via window
		}
	}

	// Recompute layout so remaining windows expand to fill freed space
	e.recomputeWindowsLayout()

	// If the closed window's buffer is not shown in any remaining window, close it globally
	if closedBuf != nil {
		stillInUse := false
		for _, w := range e.windows {
			if w == nil {
				continue
			}
			for _, b := range w.Buffers {
				if b == closedBuf {
					stillInUse = true
					break
				}
			}
			if stillInUse {
				break
			}
		}
		if !stillInUse {
			path := closedBuf.FilePath
			// Close terminal session if any
			if ts, ok := e.terminalSessions[path]; ok && ts != nil && ts.SessionID != "" {
				_ = e.proxyManager.CloseTerminal(ts.WorkingDir, ts.SessionID)
				delete(e.terminalSessions, path)
			}
			// Remove from global buffers registry
			delete(e.buffers, path)
			e.bufferOrder = removeFromSlice(e.bufferOrder, path)
		}
	}

	return nil
}

// closeCurrentWindow closes the currently active window
func (e *Editor) closeCurrentWindow(args []string) error {
	return e.closeWindow(e.activeWindow)
}
