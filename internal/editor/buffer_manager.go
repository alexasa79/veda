package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/syntax"
	"veda/internal/widgets"
)

// saveCurrentBufferState saves the current buffer's state before switching
func (e *Editor) saveCurrentBufferState() {
	if e.buffer != nil {
		// All state (cursorX, iterators, line offset cache) stays with the buffer
		// No saving needed
	}
}

// switchToBuffer switches to a different buffer
func (e *Editor) switchToBuffer(buf *buffer.Buffer) {
	// Save the previous buffer
	if e.buffer != nil && e.buffer != buf {
		e.previousBuffer = e.buffer
	}
	
	// Set the buffer - all state (cursorX, iterators, line offset cache) comes with it
	e.buffer = buf
	
	// Set up syntax highlighting
	if e.highlighter != nil && buf.HasFile() {
		e.highlighter.SetLanguageFromFile(buf.FilePath)
		e.highlighter.ParseFull([]byte(buf.Rope.String()))
	} else {
		// Reset syntax highlighting for unnamed buffer
		e.highlighter = syntax.NewHighlighter()
	}
}

// showBuffers shows the buffer list using the fuzzy list widget
func (e *Editor) showBuffers(args []string) error {
	if len(e.buffers) == 0 {
		return nil
	}
	
	width, height := e.screen.Size()
	
	// Create fuzzy list widget (centered overlay)
	fuzzyListWidth := width * 4 / 5  // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2
	
	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("Open Buffers")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationFileIcon)
	
	// Create list of buffer display names
	var bufferDisplays []string
	pathToDisplay := make(map[string]string)
	
	for _, path := range e.bufferOrder {
		if buf, exists := e.buffers[path]; exists {
			displayName := buf.DisplayName()
			if path == e.buffer.FilePath {
				// Mark current buffer
				bufferDisplays = append(bufferDisplays, displayName + " [current]")
			} else {
				bufferDisplays = append(bufferDisplays, displayName)
			}
			pathToDisplay[displayName] = path
		}
	}
	
	fuzzyListWidget.SetContent(bufferDisplays)
	
	// Create overlay and get result channel
	overlay := NewFuzzyListOverlay(fuzzyListWidget, nil, nil)
	resultChan := e.renderingState.SetOverlay(overlay)
	
	// Wait for result in a goroutine
	go func() {
		selection := <-resultChan
		
		// If a buffer was selected, switch to it
		if selection != "" {
			// Remove the " [current]" suffix if present
			selection = strings.TrimSuffix(selection, " [current]")
			
			// Save current buffer state before switching
			e.saveCurrentBufferState()
			
			// Look up the actual path from the display name
			if path, ok := pathToDisplay[selection]; ok {
				// Switch to selected buffer
				if buf, exists := e.buffers[path]; exists {
					e.switchToBuffer(buf)
				}
			}
		}
	}()
	
	return nil
}

// closeBuffer closes the current buffer
func (e *Editor) closeBuffer(args []string) error {
	// Don't close if it's the only buffer
	if len(e.buffers) <= 1 {
		// Create a new empty buffer instead
		emptyBuffer := buffer.NewBuffer("", "", e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		e.buffers[""] = emptyBuffer
		if !contains(e.bufferOrder, "") {
			e.bufferOrder = append(e.bufferOrder, "")
		}
		e.switchToBuffer(emptyBuffer)
		
		// Remove the old buffer if it wasn't the empty buffer
		if e.previousBuffer != nil && e.previousBuffer.FilePath != "" {
			delete(e.buffers, e.previousBuffer.FilePath)
			e.bufferOrder = removeFromSlice(e.bufferOrder, e.previousBuffer.FilePath)
		}
		return nil
	}
	
	// Find the buffer to switch to
	currentPath := e.buffer.FilePath
	currentIndex := -1
	for i, path := range e.bufferOrder {
		if path == currentPath {
			currentIndex = i
			break
		}
	}
	
	// Switch to the next buffer (or previous if we're at the end)
	var nextBuffer *buffer.Buffer
	if currentIndex >= 0 {
		nextIndex := currentIndex + 1
		if nextIndex >= len(e.bufferOrder) {
			nextIndex = currentIndex - 1
		}
		if nextIndex >= 0 && nextIndex < len(e.bufferOrder) {
			nextBuffer = e.buffers[e.bufferOrder[nextIndex]]
		}
	}
	
	// If we couldn't find a next buffer, just pick the first one that's not current
	if nextBuffer == nil {
		for path, buf := range e.buffers {
			if path != currentPath {
				nextBuffer = buf
				break
			}
		}
	}
	
	// Switch to the next buffer
	if nextBuffer != nil {
		e.switchToBuffer(nextBuffer)
	}
	
	// Remove the closed buffer
	delete(e.buffers, currentPath)
	e.bufferOrder = removeFromSlice(e.bufferOrder, currentPath)
	
	return nil
}

// switchToPreviousBuffer switches to the previously active buffer
func (e *Editor) switchToPreviousBuffer(args []string) error {
	if e.previousBuffer != nil {
		e.switchToBuffer(e.previousBuffer)
	}
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