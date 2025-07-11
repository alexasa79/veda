package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/syntax"
	"veda/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

// saveCurrentBufferState saves the current buffer's state before switching
func (e *Editor) saveCurrentBufferState() {
	if e.buffer != nil {
		e.buffer.SaveCursorState(e.cursorX, e.cursorY, e.topLine)
		e.buffer.LineOffsets = e.lineOffsets
		e.buffer.OffsetsDirty = e.offsetsDirty
	}
}

// switchToBuffer switches to a different buffer
func (e *Editor) switchToBuffer(buf *buffer.Buffer) {
	// Set the buffer
	e.buffer = buf
	
	// Restore cursor and view state
	e.cursorX, e.cursorY, e.topLine = buf.RestoreCursorState()
	e.lineOffsets = buf.LineOffsets
	e.offsetsDirty = buf.OffsetsDirty
	
	// Set up syntax highlighting
	if e.highlighter != nil && buf.HasFile() {
		e.highlighter.SetLanguageFromFile(buf.FilePath)
		e.highlighter.ParseFull([]byte(buf.String()))
	} else {
		// Reset syntax highlighting for unnamed buffer
		e.highlighter = syntax.NewHighlighter()
	}
}

// showBuffers shows the buffer list using the fuzzy list widget
func (e *Editor) showBuffers() error {
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
	
	// Show the widget with a background render function
	var tcellScreen tcell.Screen
	if provider, ok := e.screen.(TcellScreenProvider); ok {
		tcellScreen = provider.GetTcellScreen()
	} else {
		// For testing, we'll need to handle this differently
		return nil
	}
	
	selection := fuzzyListWidget.Show(tcellScreen, func() {
		e.drawEditor()
		e.drawStatusBar()
	})
	
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
	
	return nil
}