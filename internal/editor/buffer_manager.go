package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/syntax"
	"veda/internal/widgets"
)

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
                    if w == nil { continue }
                    for _, b := range w.Buffers {
                        if b != nil && b.FilePath == path { bufPtr = b; break }
                    }
                    if bufPtr != nil { break }
                }
            }
            if bufPtr == nil {
                return
            }

            if openInNewWin {
                insertAt := e.activeWindow + 1
                newWin := newWindow(bufPtr)
                if insertAt >= len(e.windows) {
                    e.windows = append(e.windows, newWin)
                } else {
                    e.windows = append(e.windows[:insertAt+1], e.windows[insertAt:]...)
                    e.windows[insertAt] = newWin
                }
                e.activateWindow(insertAt)
                e.switchToBuffer(bufPtr)
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
	currentPath := e.CurrentBuffer().FilePath

	// Remove buffer from current window
    if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
		window := e.windows[e.activeWindow]
        // If the current window is showing this buffer, clear it
        shouldCloseWindow := false
        if window.Buffer != nil && window.Buffer.FilePath == currentPath {
            window.Buffer = nil
            shouldCloseWindow = true
        }

		if shouldCloseWindow {
            // No more buffers in this window - close the window (handles last-window exit)
            return e.closeWindow(e.activeWindow)
		} else {
			// Window still has buffers, switched to the next one automatically
			e.SetCurrentBuffer(window.Buffer)
		}
	} else {
		// Fallback to old behavior if windowing is not properly set up
		if len(e.buffers) <= 1 {
			return e.actuallyQuit(args)
		}

		// Find another buffer to switch to
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

	// If terminal session is attached, close it via proxy
	if ts, ok := e.terminalSessions[currentPath]; ok && ts != nil && ts.SessionID != "" {
		_ = e.proxyManager.CloseTerminal(ts.WorkingDir, ts.SessionID)
		delete(e.terminalSessions, currentPath)
	}

    // Remove buffer from global buffers (only if no other window has it)
    bufferStillInUse := false
    for _, window := range e.windows {
        if window != nil && window.Buffer != nil && window.Buffer.FilePath == currentPath {
            bufferStillInUse = true
            break
        }
    }

	if !bufferStillInUse {
		delete(e.buffers, currentPath)
		e.bufferOrder = removeFromSlice(e.bufferOrder, currentPath)
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
                insertAt := e.activeWindow + 1
                newWin := newWindow(bufPtr)
                if insertAt >= len(e.windows) {
                    e.windows = append(e.windows, newWin)
                } else {
                    e.windows = append(e.windows[:insertAt+1], e.windows[insertAt:]...)
                    e.windows[insertAt] = newWin
                }
                // Activate after the buffer is owned by the window
                e.activateWindow(insertAt)
                e.switchToBuffer(bufPtr)
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
