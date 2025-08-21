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
			buf.Highlighter.SetLanguageFromFile(buf.FilePath)
			buf.Highlighter.ParseFull([]byte(buf.Rope.String()))
		} else {
			// Reset syntax highlighting for unnamed buffer
			buf.Highlighter = syntax.NewHighlighter()
		}

		// Ensure the active window owns this buffer (keeps Buffers in sync)
		if len(e.windows) > 0 {
			if e.activeWindow < 0 {
				e.activeWindow = 0
			}
			if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
				if e.windows[e.activeWindow] != nil {
					e.windows[e.activeWindow].addBufferToWindow(buf)

					// Assert invariant: buffer must belong to the active window
					e.assertBufferInWindow(buf, e.activeWindow)
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

	// Create list of buffer display names from all windows
	var bufferDisplays []string
	pathToDisplay := make(map[string]string)
	pathToWindow := make(map[string]int) // Maps buffer path to window index
	seenBuffers := make(map[string]bool) // Track already seen buffers to avoid duplicates

	// Determine current buffer path based on active window to avoid drift with e.CurrentBuffer()
	currentPath := editorPath
	if activePath != "" {
		currentPath = activePath
	}

	// Collect buffers from all windows: include all buffers in each window's buffer list
	for windowIdx, window := range e.windows {
		if window == nil {
			continue
		}
		// Include all buffers in this window
		for _, buf := range window.Buffers {
			if buf == nil {
				continue
			}
			path := buf.FilePath
			if path == "" || seenBuffers[path] {
				continue
			}
			seenBuffers[path] = true
			pathToWindow[path] = windowIdx

			displayName := buf.DisplayName()
			windowInfo := fmt.Sprintf("W%d", windowIdx+1) // 1-based window numbering for display

			var fullDisplayName string
			if currentPath != "" && path == currentPath {
				fullDisplayName = fmt.Sprintf("%s [current] (%s)", displayName, windowInfo)
			} else {
				fullDisplayName = fmt.Sprintf("%s (%s)", displayName, windowInfo)
			}

			bufferDisplays = append(bufferDisplays, fullDisplayName)
			pathToDisplay[fullDisplayName] = path
		}
	}

	// Include global buffers that are not attached to any window yet (e.g., *Messages*)
	for path, buf := range e.buffers {
		if seenBuffers[path] {
			continue
		}
		seenBuffers[path] = true
		pathToWindow[path] = -1 // not in any window

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

			// Resolve buffer pointer; fallback: scan window lists
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
				// Shift+Enter: open buffer in window to the right
				cur := e.activeWindow
				utils.Assert(cur >= 0 && cur < len(e.windows), fmt.Sprintf("Invalid active window index: %d (windows: %d)", cur, len(e.windows)))

				// Check if there's a window to the right
				if cur+1 < len(e.windows) {
					// Use existing right window
					right := cur + 1

					// Remove buffer from its current window if it's in one
					sourceWindow := e.findWindowContainingBuffer(path)
					if sourceWindow >= 0 && sourceWindow < len(e.windows) {
						e.windows[sourceWindow].removeBufferFromWindow(bufPtr)
					}

					// Add buffer to right window
					e.windows[right].addBufferToWindow(bufPtr)
					e.assertBufferInWindow(bufPtr, right)

					// Activate the right window
					e.activateWindow(right)
					e.switchToBuffer(bufPtr)
				} else {
					// Create new window to the right
					newWin := newWindow(bufPtr)

					// Remove buffer from its current window if it's in one
					sourceWindow := e.findWindowContainingBuffer(path)
					if sourceWindow >= 0 && sourceWindow < len(e.windows) {
						e.windows[sourceWindow].removeBufferFromWindow(bufPtr)
					}

					// Append new window
					e.windows = append(e.windows, newWin)

					// Activate the new window
					e.activateWindow(len(e.windows) - 1)
					e.switchToBuffer(bufPtr)
				}
				e.recomputeWindowsLayout()
				return
			}

			// Recompute current window for this buffer dynamically
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
	if isDiffBuffer(currentPath) {
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
		if ts, ok := e.terminalSessions[currentPath]; ok && ts != nil && ts.SessionID != "" {
			_ = e.proxyManager.CloseTerminal(ts.WorkingDir, ts.SessionID)
			delete(e.terminalSessions, currentPath)
		}
		// Clean up associated data for the buffer
		if buf := e.buffers[currentPath]; buf != nil {
			delete(e.bufferHighlighters, buf)
			delete(e.diffLineInfo, buf)
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

	// Create list of buffer display names from all windows
	var bufferDisplays []string
	pathToDisplay := make(map[string]string)
	pathToWindow := make(map[string]int) // Maps buffer path to window index
	seenBuffers := make(map[string]bool) // Track already seen buffers to avoid duplicates
	previousBufferIndex := -1

	// Collect buffers from all windows using each window's buffer list
	for windowIdx, window := range e.windows {
		if window == nil {
			continue
		}
		for _, buf := range window.Buffers {
			if buf == nil {
				continue
			}
			path := buf.FilePath
			if path == "" || seenBuffers[path] {
				continue
			}
			seenBuffers[path] = true
			pathToWindow[path] = windowIdx

			displayName := buf.DisplayName()
			windowInfo := fmt.Sprintf("W%d", windowIdx+1)

			var fullDisplayName string
			if e.CurrentBuffer() != nil && path == e.CurrentBuffer().FilePath {
				fullDisplayName = fmt.Sprintf("%s [current] (%s)", displayName, windowInfo)
			} else {
				fullDisplayName = fmt.Sprintf("%s (%s)", displayName, windowInfo)
				if e.previousBuffer != nil && path == e.previousBuffer.FilePath {
					previousBufferIndex = len(bufferDisplays)
				}
			}

			bufferDisplays = append(bufferDisplays, fullDisplayName)
			pathToDisplay[fullDisplayName] = path
		}
	}

	// Include global buffers not attached to any window (e.g., *Messages*)
	for path, buf := range e.buffers {
		if seenBuffers[path] {
			continue
		}
		seenBuffers[path] = true
		pathToWindow[path] = -1 // not in any window

		displayName := buf.DisplayName()
		windowInfo := "global"

		var fullDisplayName string
		if e.CurrentBuffer() != nil && path == e.CurrentBuffer().FilePath {
			fullDisplayName = fmt.Sprintf("%s [current] (%s)", displayName, windowInfo)
		} else {
			fullDisplayName = fmt.Sprintf("%s (%s)", displayName, windowInfo)
		}

		bufferDisplays = append(bufferDisplays, fullDisplayName)
		pathToDisplay[fullDisplayName] = path
		// If this is the previous buffer, track index for selection
		if e.previousBuffer != nil && path == e.previousBuffer.FilePath {
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
				// Shift+Enter: open buffer in window to the right
				cur := e.activeWindow
				utils.Assert(cur >= 0 && cur < len(e.windows), fmt.Sprintf("Invalid active window index: %d (windows: %d)", cur, len(e.windows)))

				// Check if there's a window to the right
				if cur+1 < len(e.windows) {
					// Use existing right window
					right := cur + 1

					// Remove buffer from its current window if it's in one
					sourceWindow := e.findWindowContainingBuffer(path)
					if sourceWindow >= 0 && sourceWindow < len(e.windows) {
						e.windows[sourceWindow].removeBufferFromWindow(bufPtr)
					}

					// Add buffer to right window
					e.windows[right].addBufferToWindow(bufPtr)
					e.assertBufferInWindow(bufPtr, right)

					// Activate the right window
					e.activateWindow(right)
					e.switchToBuffer(bufPtr)
				} else {
					// Create new window to the right
					newWin := newWindow(bufPtr)

					// Remove buffer from its current window if it's in one
					sourceWindow := e.findWindowContainingBuffer(path)
					if sourceWindow >= 0 && sourceWindow < len(e.windows) {
						e.windows[sourceWindow].removeBufferFromWindow(bufPtr)
					}

					// Append new window
					e.windows = append(e.windows, newWin)

					// Activate the new window
					e.activateWindow(len(e.windows) - 1)
					e.switchToBuffer(bufPtr)
				}
				e.recomputeWindowsLayout()
				return
			}
			// Recompute target window dynamically to avoid stale snapshot
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
	newBuffer := buffer.NewBuffer("", bufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)

	// Add to buffers map and order
	e.buffers[bufferName] = newBuffer
	e.bufferOrder = append(e.bufferOrder, bufferName)

	// Switch to the new buffer
	e.switchToBuffer(newBuffer)

	return nil
}

// isDiffBuffer checks if a buffer path represents a diff buffer
func isDiffBuffer(bufferPath string) bool {
	return strings.HasSuffix(bufferPath, "@HEAD") || strings.HasSuffix(bufferPath, "@current")
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
	pairedPath := getPairedDiffBuffer(bufferPath)
	if pairedPath == "" {
		// Not a diff buffer, handle normally
		return nil
	}

	// Find both buffers
	currentBuffer := e.buffers[bufferPath]
	pairedBuffer := e.buffers[pairedPath]

	// Clean up diff line info for both buffers
	if currentBuffer != nil {
		delete(e.bufferHighlighters, currentBuffer)
		delete(e.diffLineInfo, currentBuffer)
	}
	if pairedBuffer != nil {
		delete(e.bufferHighlighters, pairedBuffer)
		delete(e.diffLineInfo, pairedBuffer)
	}

	// Remove both buffers from all windows
	windowsToClose := make([]int, 0)
	
	for i, window := range e.windows {
		if window == nil {
			continue
		}
		
		shouldCloseWindow := false
		
		// Remove both diff buffers from this window
		if _, exists := window.Buffers[bufferPath]; exists {
			removed := window.removeBuffer(bufferPath)
			if removed {
				shouldCloseWindow = true
			}
		}
		if _, exists := window.Buffers[pairedPath]; exists {
			removed := window.removeBuffer(pairedPath)
			if removed && !shouldCloseWindow {
				shouldCloseWindow = true
			}
		}
		
		if shouldCloseWindow {
			windowsToClose = append(windowsToClose, i)
		}
	}

	// Remove both buffers from global buffers
	delete(e.buffers, bufferPath)
	delete(e.buffers, pairedPath)
	e.bufferOrder = removeFromSlice(e.bufferOrder, bufferPath)
	e.bufferOrder = removeFromSlice(e.bufferOrder, pairedPath)

	// Close windows that no longer have buffers, starting from the highest index
	// to avoid index shifting issues
	for i := len(windowsToClose) - 1; i >= 0; i-- {
		windowIndex := windowsToClose[i]
		if windowIndex < len(e.windows) {
			err := e.closeWindow(windowIndex)
			if err != nil {
				return err
			}
		}
	}

	// If no windows left but we still have other buffers, create a new window
	if len(e.windows) == 0 && len(e.buffers) > 0 {
		// Find any remaining buffer and create a window for it
		for _, buf := range e.buffers {
			newWin := newWindow(buf)
			e.windows = append(e.windows, newWin)
			e.activeWindow = 0
			e.switchToBuffer(buf)
			break
		}
	}

	// If no buffers left at all, create an unnamed buffer
	if len(e.buffers) == 0 {
		newBuf := e.createUnnamedBuffer()
		e.buffers[newBuf.FilePath] = newBuf
		e.bufferOrder = append(e.bufferOrder, newBuf.FilePath)
		
		if len(e.windows) == 0 {
			newWin := newWindow(newBuf)
			e.windows = append(e.windows, newWin)
			e.activeWindow = 0
		} else {
			e.windows[0].addBuffer(newBuf)
		}
		e.switchToBuffer(newBuf)
	}

	return nil
}
