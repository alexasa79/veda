package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/syntax"
	"veda/internal/utils"
	"veda/internal/widgets"
)

// assertBufferInWindow asserts that a buffer belongs to the specified window
func (e *Editor) assertBufferInWindow(buf *buffer.Buffer, windowIdx int) {
	utils.Assert(windowIdx >= 0 && windowIdx < len(e.windows), fmt.Sprintf("Invalid window index: %d (windows: %d)", windowIdx, len(e.windows)))
	utils.Assert(e.windows[windowIdx] != nil, fmt.Sprintf("Window %d is nil", windowIdx))
	utils.Assert(buf != nil, "Buffer is nil")

	bufferFound := false
	for _, b := range e.windows[windowIdx].Buffers {
		if b == buf {
			bufferFound = true
			break
		}
	}
	utils.Assert(bufferFound, fmt.Sprintf("Buffer %s not found in window %d's buffer list", buf.FilePath, windowIdx))
}

// saveCurrentBufferState saves the current buffer's state before switching
func (e *Editor) saveCurrentBufferState() {
	if e.CurrentBuffer() != nil {
		// All state (cursorX, iterators, line offset cache) stays with the buffer
		// No saving needed
	}
}

// switchToBuffer switches to a different buffer
func (e *Editor) switchToBuffer(buf *buffer.Buffer) {
	// Save the previous buffer
	if e.CurrentBuffer() != nil && e.CurrentBuffer() != buf {
		e.previousBuffer = e.CurrentBuffer()
	}

	// Set the buffer - all state (cursorX, iterators, line offset cache) comes with it
	e.SetCurrentBuffer(buf)

	// Ensure iterators are valid w.r.t. current buffer generation
	// Iterators become invalid whenever the buffer content changes off-screen (e.g., messages buffer appends)
	if e.CurrentBuffer() != nil {
		// Validate and recreate ViewIterator if needed
		if e.CurrentBuffer().ViewIterator == nil || e.CurrentBuffer().ViewIterator.Generation != e.CurrentBuffer().Generation {
			// Preserve previous desired line without triggering validation
			desiredLine := 0
			if e.CurrentBuffer().ViewIterator != nil {
				desiredLine = e.CurrentBuffer().ViewIterator.LineNum
			}
			e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), desiredLine)
		}
		// Validate and recreate CursorIterator if needed
		if e.CurrentBuffer().CursorIterator == nil || e.CurrentBuffer().CursorIterator.Generation != e.CurrentBuffer().Generation {
			desiredCursorLine := 0
			if e.CurrentBuffer().CursorIterator != nil {
				desiredCursorLine = e.CurrentBuffer().CursorIterator.LineNum
			}
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), desiredCursorLine)
		}
	}

	// Set up LSP notification callback
	e.CurrentBuffer().OnContentChange = func() {
		e.notifyLSPFileChange()
		e.updateSearchMatchesOnContentChange()
		e.refreshReferencesInBackground()
	}

	// Set up syntax highlighting (per-buffer)
	if buf.Highlighter != nil {
		// Check if this is a special buffer that should use markdown
		if (strings.HasPrefix(buf.FilePath, "*References:") && strings.HasSuffix(buf.FilePath, "*")) ||
			buf.FilePath == "*Help*" ||
			buf.FilePath == "*Key Bindings*" ||
			buf.FilePath == "*Diagnostics*" ||
			buf.FilePath == "*Messages*" ||
			buf.FilePath == "*Project Changes*" {
			// Set markdown language for special buffers
			buf.Highlighter.SetLanguage("markdown")
			buf.Highlighter.ParseFull([]byte(buf.Rope.String()))
		} else if buf.HasFile() {
			// Use real file path for language detection when available
			filePath := buf.RealPath
			if filePath == "" {
				filePath = buf.FilePath
			}
			buf.Highlighter.SetLanguageFromFile(filePath)
			buf.Highlighter.ParseFull([]byte(buf.Rope.String()))
		} else {
			// No real file path: preserve existing highlighter (e.g., diff @HEAD buffer)
			// Re-parse content to keep tokens in sync; do not reset language.
			if buf.Highlighter != nil {
				buf.Highlighter.ParseFull([]byte(buf.Rope.String()))
			} else {
				// Fallback for truly unnamed buffers
				buf.Highlighter = syntax.NewHighlighter()
			}
		}

		// Ensure the active window owns this buffer (keeps Buffers in sync)
		if len(e.windows) > 0 {
			if e.activeWindow < 0 {
				e.activeWindow = 0
			}
			if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
				if win := e.windows[e.activeWindow]; win != nil {
					win.addBufferToWindow(buf)

					// Assert invariant: buffer must belong to the active window
					e.assertBufferInWindow(buf, e.activeWindow)

					// Sync win.View to the view containing this buffer
					if idx := win.findViewForBuffer(buf); idx >= 0 {
						win.View = win.Views[idx]
					}
				}
			}
		}
	}
}

// findWindowContainingBuffer returns the window index that contains the given buffer path
func (e *Editor) findWindowContainingBuffer(bufferPath string) int {
	for i, window := range e.windows {
		if window != nil {
			// Search through all buffers in the window
			for _, buf := range window.Buffers {
				if buf != nil && buf.FilePath == bufferPath {
					return i
				}
			}
		}
	}
	return -1
}

// showBuffers shows the buffer list using the fuzzy list widget
func (e *Editor) showBuffers(args []string) error {
	// Debug: log current window/buffer context when opening the picker
	var activePath string
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil && e.windows[e.activeWindow].Buffer != nil {
		activePath = e.windows[e.activeWindow].Buffer.FilePath
	}
	var editorPath string
	if e.CurrentBuffer() != nil {
		editorPath = e.CurrentBuffer().FilePath
	}
	LL.Debug("showBuffers: activeWindow=%d activeWinPath=%s editorPath=%s", e.activeWindow, activePath, editorPath)
	// Check if we have any windows at all
	if len(e.windows) == 0 {
		return nil
	}

	width, height := e.screen.Size()

	// Create fuzzy list widget (centered overlay)
	fuzzyListWidth := width * 4 / 5    // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("All Buffers")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationFileIcon)

	// Create list of buffer display names by iterating Views in all windows.
	var bufferDisplays []string
	pathToDisplay := make(map[string]string)
	pathToWindow := make(map[string]int)
	seenViews := make(map[BufferView]bool)
	seenPaths := make(map[string]bool)

	// Determine current buffer path based on active window to avoid drift with e.CurrentBuffer()
	currentPath := editorPath
	if activePath != "" {
		currentPath = activePath
	}

	// Collect one entry per View from all windows.
	for windowIdx, window := range e.windows {
		if window == nil {
			continue
		}
		for _, view := range window.Views {
			if view == nil || seenViews[view] {
				continue
			}
			seenViews[view] = true

			displayName := view.DisplayName()
			viewPath := view.ViewPath()
			if viewPath == "" {
				continue
			}
			seenPaths[viewPath] = true
			for _, b := range view.AllBuffers() {
				if b != nil {
					seenPaths[b.FilePath] = true
				}
			}
			// Use the active buffer's path for selection so switchToBuffer
			// finds the correct view (important for DiffBufferView where
			// ViewPath is the real file but ActiveBuffer is @current).
			selectPath := viewPath
			if ab := view.ActiveBuffer(); ab != nil {
				selectPath = ab.FilePath
			}
			pathToWindow[selectPath] = windowIdx
			windowInfo := fmt.Sprintf("W%d", windowIdx+1)

			var fullDisplayName string
			if currentPath != "" && viewPath == currentPath {
				fullDisplayName = fmt.Sprintf("%s [current] (%s)", displayName, windowInfo)
			} else {
				fullDisplayName = fmt.Sprintf("%s (%s)", displayName, windowInfo)
			}

			bufferDisplays = append(bufferDisplays, fullDisplayName)
			pathToDisplay[fullDisplayName] = selectPath
		}
	}

	// Include global buffers that are not attached to any window's Views (e.g., *Messages*)
	for path, buf := range e.buffers {
		if seenPaths[path] {
			continue
		}
		seenPaths[path] = true
		pathToWindow[path] = -1

		displayName := buf.DisplayName()
		windowInfo := "global"

		var fullDisplayName string
		if currentPath != "" && path == currentPath {
			fullDisplayName = fmt.Sprintf("%s [current] (%s)", displayName, windowInfo)
		} else {
			fullDisplayName = fmt.Sprintf("%s (%s)", displayName, windowInfo)
		}

		bufferDisplays = append(bufferDisplays, fullDisplayName)
		pathToDisplay[fullDisplayName] = path
	}

	fuzzyListWidget.SetContent(bufferDisplays)

	// Create overlay and get result channel
	overlay := NewFuzzyListOverlay(fuzzyListWidget, nil, nil)
	resultChan := e.renderingState.SetOverlay(overlay)

	// Wait for result in a goroutine; post a lambda to the main loop
	go func() {
		selection := <-resultChan
		if selection == "" {
			return
		}
		openInNewWin := false
		if strings.HasPrefix(selection, "__SHIFT__\n") {
			openInNewWin = true
			selection = strings.TrimPrefix(selection, "__SHIFT__\n")
		}
		// String-only resolution; actual state mutations happen on main loop
		path, ok := pathToDisplay[selection]
		if !ok {
			return
		}
		e.Post(func(e *Editor) {
			// Save current buffer state before switching
			e.saveCurrentBufferState()

			// Resolve buffer pointer; check global buffers and window buffer lists
			var bufPtr *buffer.Buffer
			if b, exists := e.buffers[path]; exists {
				bufPtr = b
			} else {
				for _, w := range e.windows {
					if w == nil {
						continue
					}
					for _, b := range w.Buffers {
						if b != nil && b.FilePath == path {
							bufPtr = b
							break
						}
					}
					if bufPtr != nil {
						break
					}
				}
			}
			if bufPtr == nil {
				return
			}

			if openInNewWin {
				// Shift+Enter: always open in a new rightmost window
				// Remove buffer from its current window if it's in one
				if sourceWindow := e.findWindowContainingBuffer(path); sourceWindow >= 0 && sourceWindow < len(e.windows) {
					e.windows[sourceWindow].removeBufferFromWindow(bufPtr)
				}
				// Create and focus the new window at the far right
				e.createRightmostWindow(bufPtr)
				e.switchToBuffer(bufPtr)
				e.recomputeWindowsLayout()
				return
			}

			// Find the window containing this buffer and activate it.
			// switchToBuffer will set win.View to the correct view from Views.
			targetWindowIdx := e.findWindowContainingBuffer(path)
			if targetWindowIdx >= 0 && targetWindowIdx < len(e.windows) {
				if tw := e.windows[targetWindowIdx]; tw != nil {
					tw.addBufferToWindow(bufPtr)
					e.activateWindow(targetWindowIdx)
					e.switchToBuffer(bufPtr)
					return
				}
			}
			// Not present in any window: attach to active window and switch
			if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
				if aw := e.windows[e.activeWindow]; aw != nil {
					aw.addBufferToWindow(bufPtr)
					e.switchToBuffer(bufPtr)
				}
			}
		})
	}()

	return nil
}

// closeBuffer closes the current buffer
func (e *Editor) closeBuffer(args []string) error {
	cur := e.CurrentBuffer()
	if cur == nil {
		return nil
	}
	currentPath := cur.FilePath

	// Check if this is a diff buffer - if so, close both buffers in the pair
	if cur.DiffHints != nil {
		return e.closeDiffBufferPair(currentPath)
	}

	// Remove buffer from current window (close the window if it becomes empty)
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
		w := e.windows[e.activeWindow]
		// If this is the last buffer in the window, close the window first so cleanup can see the buffer
		if len(w.Buffers) == 1 && w.Buffers[0] == cur {
			// Close the window; closeWindow will handle global/terminal cleanup if no other window has the buffer
			_ = e.closeWindow(e.activeWindow)
			return nil
		}
		// Otherwise, remove from per-window list and adjust active buffer
		w.removeBufferFromWindow(cur)
		// Sync editor's current buffer to the window's active buffer (may be nil)
		e.SetCurrentBuffer(w.Buffer)
	} else {
		// Fallback: if no proper windowing, switch to some other buffer (keep editor alive)
		for path, buf := range e.buffers {
			if path != currentPath {
				e.switchToBuffer(buf)
				break
			}
		}
	}

	// Clean up auto-save file since the buffer is being closed without saving
	if e.autoSaveManager != nil && currentPath != "" {
		e.autoSaveManager.CleanupAutoSave(currentPath)
	}

	// If terminal session is attached, close it via proxy (only if buffer no longer shown anywhere)
	// Determine if buffer is still in use in any window (scan window.Buffers)
	stillInUse := false
	for _, win := range e.windows {
		if win == nil {
			continue
		}
		for _, b := range win.Buffers {
			if b == cur {
				stillInUse = true
				break
			}
		}
		if stillInUse {
			break
		}
	}

	if !stillInUse {
		// Clear lastActiveSpecialBuffer if we're closing it
		if e.lastActiveSpecialBuffer == currentPath {
			e.lastActiveSpecialBuffer = ""
			// Check if there's another special buffer still open
			for path := range e.buffers {
				if e.isSpecialBuffer(path) {
					e.lastActiveSpecialBuffer = path
					break
				}
			}
		}

		if ts, ok := e.terminalSessions[currentPath]; ok && ts != nil && ts.SessionID != "" {
			_ = e.proxyManager.CloseTerminal(ts.WorkingDir, ts.SessionID)
			delete(e.terminalSessions, currentPath)
		}
		// Clean up associated data for the buffer
		if buf := e.buffers[currentPath]; buf != nil {
			buf.Highlighter = nil
			buf.DiffHints = nil
		}
		// Remove from global buffers registry
		delete(e.buffers, currentPath)
		e.bufferOrder = removeFromSlice(e.bufferOrder, currentPath)
		// Drop any per-buffer virtual cursors for this buffer
		delete(e.virtualCursorsByBuffer, cur)
	}

	return nil
}

// nextBufferInWindow switches to the next buffer in the current window
func (e *Editor) nextBufferInWindow(args []string) error {
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
		// No per-window list anymore; nothing to cycle within a window
		// Could later implement MRU or global cycling; for now, no-op
	}
	return nil
}

// previousBufferInWindow switches to the previous buffer in the current window
func (e *Editor) previousBufferInWindow(args []string) error {
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
		// No per-window list anymore; for now, no-op
	}
	return nil
}

// showBufferListWithPrevious shows the buffer list with cursor on the previous buffer
// This replaces the old switchToPreviousBuffer functionality
func (e *Editor) showBufferListWithPrevious(args []string) error {
	// Check if we have any windows at all
	if len(e.windows) == 0 {
		return nil
	}

	width, height := e.screen.Size()

	// Create fuzzy list widget (centered overlay)
	fuzzyListWidth := width * 4 / 5    // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("All Buffers")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationFileIcon)

	// Create list of buffer display names by iterating Views in all windows.
	var bufferDisplays []string
	pathToDisplay := make(map[string]string)
	pathToWindow := make(map[string]int)
	seenViews := make(map[BufferView]bool)
	seenPaths := make(map[string]bool)
	previousBufferIndex := -1

	// Determine the "current" path from the active window's view.
	currentPath := ""
	if e.CurrentBuffer() != nil {
		currentPath = e.CurrentBuffer().FilePath
	}
	// Use the active view's ViewPath so it matches the view entry.
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
		if win := e.windows[e.activeWindow]; win != nil && win.View != nil {
			currentPath = win.View.ViewPath()
		}
	}

	// Determine the "previous" path.
	previousPath := ""
	if e.previousBuffer != nil {
		previousPath = e.previousBuffer.FilePath
		if e.previousBuffer.RealPath != "" {
			// If previousBuffer has a RealPath, use it (handles diff internal buffers)
			previousPath = e.previousBuffer.RealPath
		}
	}

	// Collect one entry per View from all windows.
	for windowIdx, window := range e.windows {
		if window == nil {
			continue
		}
		for _, view := range window.Views {
			if view == nil || seenViews[view] {
				continue
			}
			seenViews[view] = true

			displayName := view.DisplayName()
			viewPath := view.ViewPath()
			if viewPath == "" {
				continue
			}
			seenPaths[viewPath] = true
			for _, b := range view.AllBuffers() {
				if b != nil {
					seenPaths[b.FilePath] = true
				}
			}
			selectPath := viewPath
			if ab := view.ActiveBuffer(); ab != nil {
				selectPath = ab.FilePath
			}
			pathToWindow[selectPath] = windowIdx
			windowInfo := fmt.Sprintf("W%d", windowIdx+1)

			var fullDisplayName string
			if currentPath != "" && viewPath == currentPath {
				fullDisplayName = fmt.Sprintf("%s [current] (%s)", displayName, windowInfo)
			} else {
				fullDisplayName = fmt.Sprintf("%s (%s)", displayName, windowInfo)
				if previousPath != "" && (viewPath == previousPath || selectPath == previousPath) {
					previousBufferIndex = len(bufferDisplays)
				}
			}

			bufferDisplays = append(bufferDisplays, fullDisplayName)
			pathToDisplay[fullDisplayName] = selectPath
		}
	}

	// Include global buffers not attached to any window's Views (e.g., *Messages*)
	for path, buf := range e.buffers {
		if seenPaths[path] {
			continue
		}
		seenPaths[path] = true
		pathToWindow[path] = -1

		displayName := buf.DisplayName()
		windowInfo := "global"

		var fullDisplayName string
		if currentPath != "" && path == currentPath {
			fullDisplayName = fmt.Sprintf("%s [current] (%s)", displayName, windowInfo)
		} else {
			fullDisplayName = fmt.Sprintf("%s (%s)", displayName, windowInfo)
		}

		bufferDisplays = append(bufferDisplays, fullDisplayName)
		pathToDisplay[fullDisplayName] = path
		if previousPath != "" && path == previousPath {
			previousBufferIndex = len(bufferDisplays) - 1
		}
	}

	fuzzyListWidget.SetContent(bufferDisplays)

	// Set cursor on previous buffer if available
	if previousBufferIndex >= 0 {
		fuzzyListWidget.SetSelectedIndex(previousBufferIndex)
	} else if len(bufferDisplays) > 1 {
		// If no previous buffer, select the first non-current buffer
		fuzzyListWidget.SetSelectedIndex(1)
	}

	// Create overlay and get result channel
	overlay := NewFuzzyListOverlay(fuzzyListWidget, nil, nil)
	resultChan := e.renderingState.SetOverlay(overlay)

	// Wait for result in a goroutine; post a lambda to the main loop
	go func() {
		selection := <-resultChan
		if selection == "" {
			return
		}
		openInNewWin := false
		if strings.HasPrefix(selection, "__SHIFT__\n") {
			openInNewWin = true
			selection = strings.TrimPrefix(selection, "__SHIFT__\n")
		}
		// Resolve path from display name snapshot (string-only data), but do
		// all state mutations in the main loop lambda below.
		path, ok := pathToDisplay[selection]
		if !ok {
			return
		}
		e.Post(func(e *Editor) {
			// Save current buffer state before switching
			e.saveCurrentBufferState()

			// Resolve buffer pointer from registry; fallback: scan windows
			var bufPtr *buffer.Buffer
			if b, exists := e.buffers[path]; exists {
				bufPtr = b
			} else {
				for _, w := range e.windows {
					if w == nil {
						continue
					}
					for _, b := range w.Buffers {
						if b != nil && b.FilePath == path {
							bufPtr = b
							break
						}
					}
					if bufPtr != nil {
						break
					}
				}
			}
			if bufPtr == nil {
				return
			}
			if openInNewWin {
				// Shift+Enter: always open in a new rightmost window
				if sourceWindow := e.findWindowContainingBuffer(path); sourceWindow >= 0 && sourceWindow < len(e.windows) {
					e.windows[sourceWindow].removeBufferFromWindow(bufPtr)
				}
				e.createRightmostWindow(bufPtr)
				e.switchToBuffer(bufPtr)
				e.recomputeWindowsLayout()
				return
			}
			// Find the window containing this buffer and activate it.
			// switchToBuffer will set win.View to the correct view from Views.
			targetWindowIdx := e.findWindowContainingBuffer(path)
			if targetWindowIdx >= 0 && targetWindowIdx < len(e.windows) {
				if tw := e.windows[targetWindowIdx]; tw != nil {
					tw.addBufferToWindow(bufPtr)
					e.activateWindow(targetWindowIdx)
					e.switchToBuffer(bufPtr)
					return
				}
			}
			// Not present in any window: attach to active window and switch
			if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
				if aw := e.windows[e.activeWindow]; aw != nil {
					aw.addBufferToWindow(bufPtr)
					e.switchToBuffer(bufPtr)
				}
			}
		})
	}()

	return nil
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// newBuffer creates a new empty buffer
func (e *Editor) newBuffer(args []string) error {
	// Generate a unique buffer name for the new empty buffer
	bufferName := "[No Name]"
	if _, exists := e.buffers[bufferName]; exists {
		counter := 2
		for {
			candidate := fmt.Sprintf("[No Name %d]", counter)
			if _, exists := e.buffers[candidate]; !exists {
				bufferName = candidate
				break
			}
			counter++
		}
	}

	// Create a new empty buffer
	newBuffer := buffer.NewBuffer("", bufferName, "", e.config.DefaultIndentType, e.config.DefaultIndentWidth)

	// Add to buffers map and order
	e.buffers[bufferName] = newBuffer
	e.bufferOrder = append(e.bufferOrder, bufferName)

	// Switch to the new buffer
	e.switchToBuffer(newBuffer)

	return nil
}

// getPairedDiffBuffer returns the path of the paired diff buffer, or empty string if not found
func getPairedDiffBuffer(bufferPath string) string {
	if strings.HasSuffix(bufferPath, "@HEAD") {
		// Return the @current version
		baseName := strings.TrimSuffix(bufferPath, "@HEAD")
		return baseName + "@current"
	} else if strings.HasSuffix(bufferPath, "@current") {
		// Return the @HEAD version
		baseName := strings.TrimSuffix(bufferPath, "@current")
		return baseName + "@HEAD"
	}
	return ""
}

// closeDiffBufferPair closes both diff buffers in a pair and handles window management
func (e *Editor) closeDiffBufferPair(bufferPath string) error {
	buf := e.buffers[bufferPath]

	// Check if this buffer is in a DiffBufferView
	if buf != nil {
		if dv := e.findDiffViewForBuffer(buf); dv != nil {
			// Found DiffBufferView — use exitDiffViewIfActive style cleanup
			for _, w := range e.windows {
				if w == nil {
					continue
				}
				// Check if this window's Views list contains the DiffBufferView
				found := false
				for vi, v := range w.Views {
					if v == dv {
						w.Views = append(w.Views[:vi], w.Views[vi+1:]...)
						found = true
						break
					}
				}
				if !found {
					continue
				}
				// Remove diff buffers from editor
				for _, b := range dv.AllBuffers() {
					if b != nil {
						b.DiffHints = nil
						b.Highlighter = nil
						delete(e.buffers, b.FilePath)
						e.bufferOrder = removeFromSlice(e.bufferOrder, b.FilePath)
						w.removeBufferFromWindow(b)
					}
				}
				// Restore SingleBufferView for original file
				w.View = nil
				if origBuf := e.buffers[dv.RealPath]; origBuf != nil {
					if w.findViewForBuffer(origBuf) < 0 {
						w.Views = append(w.Views, &SingleBufferView{Buf: origBuf})
					}
					w.setActiveBuffer(origBuf)
					e.switchToBuffer(origBuf)
				} else if w.Buffer != nil {
					e.switchToBuffer(w.Buffer)
				} else if len(e.buffers) > 0 {
					for _, b := range e.buffers {
						w.setActiveBuffer(b)
						e.switchToBuffer(b)
						break
					}
				} else {
					newBuf := e.createUnnamedBuffer()
					w.setActiveBuffer(newBuf)
					e.switchToBuffer(newBuf)
				}
				e.recomputeWindowsLayout()
				return nil
			}
		}
	}

	// Legacy fallback: string-suffix matching for two-window diff
	pairedPath := getPairedDiffBuffer(bufferPath)
	if pairedPath == "" {
		return nil
	}

	currentBuffer := e.buffers[bufferPath]
	pairedBuffer := e.buffers[pairedPath]

	if currentBuffer != nil {
		currentBuffer.Highlighter = nil
		currentBuffer.DiffHints = nil
	}
	if pairedBuffer != nil {
		pairedBuffer.Highlighter = nil
		pairedBuffer.DiffHints = nil
	}

	windowsToClose := make([]int, 0)
	for i, window := range e.windows {
		if window == nil {
			continue
		}
		shouldCloseWindow := false
		for _, b := range window.Buffers {
			if b != nil && (b.FilePath == bufferPath || b.FilePath == pairedPath) {
				window.removeBufferFromWindow(b)
				if len(window.Buffers) == 0 {
					shouldCloseWindow = true
				}
			}
		}
		if shouldCloseWindow {
			windowsToClose = append(windowsToClose, i)
		}
	}

	delete(e.buffers, bufferPath)
	delete(e.buffers, pairedPath)
	e.bufferOrder = removeFromSlice(e.bufferOrder, bufferPath)
	e.bufferOrder = removeFromSlice(e.bufferOrder, pairedPath)

	for i := len(windowsToClose) - 1; i >= 0; i-- {
		windowIndex := windowsToClose[i]
		if windowIndex < len(e.windows) {
			if err := e.closeWindow(windowIndex); err != nil {
				return err
			}
		}
	}

	if len(e.windows) == 0 && len(e.buffers) > 0 {
		for _, b := range e.buffers {
			newWin := newWindow(b)
			e.windows = append(e.windows, newWin)
			e.activeWindow = 0
			e.switchToBuffer(b)
			break
		}
	}

	if len(e.buffers) == 0 {
		newBuf := e.createUnnamedBuffer()
		if len(e.windows) == 0 {
			newWin := newWindow(newBuf)
			e.windows = append(e.windows, newWin)
			e.activeWindow = 0
		} else {
			e.windows[0].addBufferToWindow(newBuf)
		}
		e.switchToBuffer(newBuf)
	}

	return nil
}
