package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

func (e *Editor) showFuzzyListWidget() {
	width, height := e.screen.Size()

	// Create fuzzy list widget (centered overlay)
	fuzzyListWidth := width * 4 / 5  // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("Open File")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationFileIcon)

	// Get recursive directory listing from proxy
	files, err := e.proxyClient.ListDirectoryRecursive(".")
	if err != nil {
		fuzzyListWidget.SetContent([]string{"Error: " + err.Error()})
	} else {
		var paths []string
		for _, file := range files {
			// Filter out hidden files and directories (starting with .)
			// Check if any component of the path starts with .
			skipFile := false
			pathComponents := strings.Split(file.Path, "/")
			for _, component := range pathComponents {
				if len(component) > 0 && component[0] == '.' {
					skipFile = true
					break
				}
			}
			if skipFile {
				continue
			}

			// Only include files, not directories
			if file.Type != "file" {
				continue
			}

			paths = append(paths, file.Path)
		}
		fuzzyListWidget.SetContent(paths)
	}

	// Show the widget with a background render function
	// The fuzzy list widget needs the full tcell.Screen interface
	var tcellScreen tcell.Screen
	if provider, ok := e.screen.(TcellScreenProvider); ok {
		tcellScreen = provider.GetTcellScreen()
	} else {
		// For testing, we'll need to handle this differently
		// For now, just return empty selection
		return
	}

	selection := fuzzyListWidget.Show(tcellScreen, func() {
		e.drawEditor()
		e.drawStatusBar()
	})

	// If a file was selected, open it
	if selection != "" {
		e.openFile(selection)
	}
}

// openFile loads a file's content into the editor buffer
func (e *Editor) openFile(filePath string) {
	// First save the current buffer state
	if e.currentFile != "" {
		e.saveCurrentBufferState()
	}
	
	// Check if this file is already open
	if bufferInfo, exists := e.buffers[filePath]; exists {
		// Switch to existing buffer
		e.switchToBuffer(bufferInfo)
		return
	}

	// Read file content using proxy
	content, err := e.proxyClient.GetFile(filePath)
	if err != nil {
		// TODO: Show error to user
		return
	}

	// Create a new buffer with the file content
	newBuffer := buffer.NewBufferFromString(content)
	
	// Create buffer info
	bufferInfo := &BufferInfo{
		buffer:       newBuffer,
		filePath:     filePath,
		cursorX:      0,
		cursorY:      0,
		topLine:      0,
		lineOffsets:  []int{},
		offsetsDirty: true,
	}
	
	// Add to buffers map and order
	e.buffers[filePath] = bufferInfo
	
	// If this is replacing the empty buffer AND it's still empty, remove it
	// We should keep the [No Name] buffer if the user has typed anything into it
	if e.currentFile == "[No Name]" {
		if noNameBuffer, exists := e.buffers["[No Name]"]; exists {
			// Only remove if the buffer is truly empty
			if noNameBuffer.buffer.String() == "" {
				delete(e.buffers, "[No Name]")
				for i, path := range e.bufferOrder {
					if path == "[No Name]" {
						e.bufferOrder = append(e.bufferOrder[:i], e.bufferOrder[i+1:]...)
						break
					}
				}
			}
		}
	}
	
	// Add new buffer to order
	e.bufferOrder = append(e.bufferOrder, filePath)
	
	// Switch to the new buffer
	e.switchToBuffer(bufferInfo)
	
	// Detect indentation for the new file
	e.indentation = e.detectIndentation()
}

// openFileCmd implements the open-file command
func (e *Editor) openFileCmd() error {
	e.showFuzzyListWidget()
	return nil
}