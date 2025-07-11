package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/syntax"
	"veda/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

// BufferInfo holds information about an open buffer
type BufferInfo struct {
	buffer       *buffer.Buffer
	filePath     string
	cursorX      int
	cursorY      int
	topLine      int
	lineOffsets  []int  // Cache of byte offsets for each line
	offsetsDirty bool   // Whether line offsets need recalculation
}

// saveCurrentBufferState saves the current buffer's state before switching
func (e *Editor) saveCurrentBufferState() {
	if e.currentFile != "" {
		if bufferInfo, exists := e.buffers[e.currentFile]; exists {
			bufferInfo.cursorX = e.cursorX
			bufferInfo.cursorY = e.cursorY
			bufferInfo.topLine = e.topLine
			bufferInfo.lineOffsets = e.lineOffsets
			bufferInfo.offsetsDirty = e.offsetsDirty
		}
	}
}

// switchToBuffer switches to a different buffer
func (e *Editor) switchToBuffer(bufferInfo *BufferInfo) {
	// Set the buffer and related fields
	e.buffer = bufferInfo.buffer
	e.currentFile = bufferInfo.filePath
	e.cursorX = bufferInfo.cursorX
	e.cursorY = bufferInfo.cursorY
	e.topLine = bufferInfo.topLine
	e.lineOffsets = bufferInfo.lineOffsets
	e.offsetsDirty = bufferInfo.offsetsDirty
	
	// Set up syntax highlighting
	if e.highlighter != nil && bufferInfo.filePath != "[No Name]" {
		e.highlighter.SetLanguageFromFile(bufferInfo.filePath)
		e.highlighter.ParseFull([]byte(e.buffer.String()))
	} else {
		// Reset syntax highlighting for empty buffer
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
	
	// Create list of buffer paths
	var bufferPaths []string
	for _, path := range e.bufferOrder {
		if path == e.currentFile {
			// Mark current buffer
			bufferPaths = append(bufferPaths, path + " [current]")
		} else {
			bufferPaths = append(bufferPaths, path)
		}
	}
	
	fuzzyListWidget.SetContent(bufferPaths)
	
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
		
		// Switch to selected buffer
		if bufferInfo, exists := e.buffers[selection]; exists {
			e.switchToBuffer(bufferInfo)
		}
	}
	
	return nil
}