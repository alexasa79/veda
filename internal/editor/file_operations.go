package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/widgets"
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
	// Cast our Screen to widgets.Screen interface
	if widgetScreen, ok := e.screen.(widgets.Screen); ok {
		selection := fuzzyListWidget.Show(widgetScreen, func() {
			e.drawEditor()
			e.drawStatusBar()
		})
		
		// If a file was selected, open it
		if selection != "" {
			e.openFile(selection)
		}
	} else {
		// Screen doesn't implement widgets.Screen, return
		return
	}
}

// OpenFile loads a file's content into the editor buffer (exported for main.go)
func (e *Editor) OpenFile(filePath string) {
	e.openFile(filePath)
}

// openFile loads a file's content into the editor buffer
func (e *Editor) openFile(filePath string) {
	// First save the current buffer state
	e.saveCurrentBufferState()
	
	// Check if this file is already open
	if buf, exists := e.buffers[filePath]; exists {
		// Switch to existing buffer
		e.switchToBuffer(buf)
		return
	}

	// Read file content using proxy
	content, err := e.proxyClient.GetFile(filePath)
	if err != nil {
		// TODO: Show error to user
		return
	}

	// Create a new buffer with the file content
	newBuffer := buffer.NewBuffer(content, filePath, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
	
	// Add to buffers map and order
	e.buffers[filePath] = newBuffer
	
	// If this is replacing the empty buffer AND it's still empty, remove it
	// We should keep the unnamed buffer if the user has typed anything into it
	if !e.buffer.HasFile() {
		if unnamedBuffer, exists := e.buffers[""]; exists {
			// Only remove if the buffer is truly empty
			if unnamedBuffer.String() == "" {
				delete(e.buffers, "")
				for i, path := range e.bufferOrder {
					if path == "" {
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
	e.switchToBuffer(newBuffer)
	
	// Detect indentation for the new file
	newBuffer.Indentation = e.detectIndentation()
	
	// Check if LSP is available for this file
	// We do a quick check by trying to get folding ranges
	_, err = e.proxyClient.GetFoldingRanges(filePath)
	newBuffer.HasLSP = (err == nil)
}

// openFileCmd implements the open-file command
func (e *Editor) openFileCmd() error {
	e.showFuzzyListWidget()
	return nil
}