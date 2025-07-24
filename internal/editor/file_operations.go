package editor

import (
	"fmt"
	"strings"
	"time"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/lsp"
	"veda/internal/proxy"
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
			if unnamedBuffer.Rope.String() == "" {
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

	// Request folding ranges from LSP worker thread
	e.requestFoldingRanges(filePath)
}

// openFileCmd implements the open-file command
func (e *Editor) openFileCmd() error {
	e.showFuzzyListWidget()
	return nil
}

// showProjectSymbolsList shows a fuzzy list of all project symbols
func (e *Editor) showProjectSymbolsList() {
	LL.Debug("showProjectSymbolsList called")
	width, height := e.screen.Size()

	// Create fuzzy list widget (centered overlay)
	fuzzyListWidth := width * 4 / 5  // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("Project Symbols (type to search)")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationNone)

	// Start with empty content
	fuzzyListWidget.SetContent([]string{})
	
	// Cast our Screen to widgets.Screen interface
	if widgetScreen, ok := e.screen.(widgets.Screen); ok {
		// Track the last query to avoid redundant LSP calls
		var lastQuery string
		var pendingQuery string // Query that is currently being processed
		var symbols []proxy.ProjectSymbol // Store the actual symbol objects
		var symbolMap = make(map[string]int) // Map display name to index
		
		// Create a channel to receive LSP responses for this widget
		symbolResponseChan := make(chan lsp.LSPResponse, 10)
		
		// Create a channel for content updates
		contentUpdateChan := make(chan []string, 10)
		
		// Create a channel to pass symbol data back from the goroutine
		type symbolUpdate struct {
			symbols []proxy.ProjectSymbol
			symbolMap map[string]int
		}
		symbolUpdateChan := make(chan symbolUpdate, 1)
		
		// Start a goroutine to handle LSP responses
		go func() {
			for response := range symbolResponseChan {
				if response.Type == lsp.RequestProjectSymbols && response.Query == pendingQuery {
					if response.Error != nil {
						LL.Debug("LSP query error: %v", response.Error)
						contentUpdateChan <- []string{fmt.Sprintf("Error: %v", response.Error)}
						symbolUpdateChan <- symbolUpdate{nil, make(map[string]int)}
					} else {
						LL.Debug("Got %d symbols from LSP for query '%s'", len(response.ProjectSymbols), response.Query)
						
						// Convert to display strings
						var displayNames []string
						newSymbolMap := make(map[string]int)
						
						for i, sym := range response.ProjectSymbols {
							displayNames = append(displayNames, sym.Name)
							newSymbolMap[sym.Name] = i
						}
						
						// Send the symbol data update
						symbolUpdateChan <- symbolUpdate{response.ProjectSymbols, newSymbolMap}
						
						if len(displayNames) == 0 {
							contentUpdateChan <- []string{"No symbols found"}
						} else {
							// Send all symbols - the widget will apply filtering
							contentUpdateChan <- displayNames
						}
					}
				}
			}
			close(symbolUpdateChan)
		}()
		
		// Start another goroutine to handle symbol updates
		go func() {
			for update := range symbolUpdateChan {
				symbols = update.symbols
				symbolMap = update.symbolMap
			}
		}()
		
		// Define the text change handler
		onTextChanged := func(newText string) {
			if newText != lastQuery {
				lastQuery = newText
				
				if len(newText) > 0 {
					// Send async LSP request for symbols
					pendingQuery = newText
					LL.Debug("Sending async LSP query for symbols with: %s", newText)
					
					request := lsp.LSPRequest{
						Type:       lsp.RequestProjectSymbols,
						FilePath:   e.buffer.FilePath,
						Buffer:     e.buffer,
						Generation: e.buffer.SignatureGeneration,
						Query:      newText,
						ResponseTo: symbolResponseChan,
					}
					
					// Send request to LSP worker (non-blocking)
					select {
					case e.lspRequestChan <- request:
						LL.Trace("Sent project symbols request for query '%s'", newText)
						// Don't clear the list - keep current results displayed
					default:
						LL.Debug("LSP request channel full, dropping request")
					}
				} else {
					// Empty query - clear the list
					contentUpdateChan <- []string{}
					symbols = nil
					symbolMap = make(map[string]int)
					pendingQuery = ""
				}
			}
		}
		
		// Show the widget with async support
		selection := fuzzyListWidget.ShowWithCallbacks(widgetScreen, func() {
			e.drawEditor()
			e.drawStatusBar()
		}, onTextChanged, contentUpdateChan)
		
		// Close the channels
		close(symbolResponseChan)
		close(contentUpdateChan)
		
		// If a symbol was selected, navigate to it
		if selection != "" {
			// Find the selected symbol by display name
			if idx, exists := symbolMap[selection]; exists && idx < len(symbols) {
				sym := symbols[idx]
				
				// Open file if different
				if sym.Path != e.buffer.FilePath {
					e.openFile(sym.Path)
				}
				// Navigate to location
				e.navigateToLocation(sym.Line, sym.Character)
			}
		}
	}
}

// navigateToLocation moves cursor to specific line and character position
func (e *Editor) navigateToLocation(line, character int) {
	targetLine := line // LSP line numbers are 0-based
	e.buffer.CursorX = character

	// Create iterator and advance to target line
	e.buffer.CursorIterator = buffer.NewBufferIterator(e.buffer)

	// Move iterator to target line
	for i := 0; i < targetLine && !e.buffer.CursorIterator.IsEOF(); i++ {
		e.buffer.CursorIterator = e.buffer.CursorIterator.Next()
	}

	// Ensure cursor X is within bounds
	currentLine := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(currentLine) {
		e.buffer.CursorX = len(currentLine)
		if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
			e.buffer.CursorX-- // Don't go past last character in normal mode
		}
	}

	// Position the definition at ScrollOffset from the top of the screen
	e.scrollToShowAtTop()
}

// listProjectSymbolsCmd implements the list-project-symbols command
func (e *Editor) listProjectSymbolsCmd() error {
	LL.Debug("listProjectSymbolsCmd called")
	e.showProjectSymbolsList()
	return nil
}

// showDocumentSymbolsList shows a fuzzy list of symbols in the current document
func (e *Editor) showDocumentSymbolsList() {
	LL.Debug("showDocumentSymbolsList called")
	width, height := e.screen.Size()

	// Create fuzzy list widget (centered overlay)
	fuzzyListWidth := width * 4 / 5  // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("Document Symbols")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationNone)

	// Start with loading message
	fuzzyListWidget.SetContent([]string{"Loading symbols..."})
	
	// Cast our Screen to widgets.Screen interface
	if widgetScreen, ok := e.screen.(widgets.Screen); ok {
		var symbols []proxy.DocumentSymbol // Store the actual symbol objects
		var symbolMap = make(map[string]int) // Map display name to index
		
		// Create a channel to receive LSP responses for this widget
		symbolResponseChan := make(chan lsp.LSPResponse, 1)
		
		// Create a channel for content updates
		contentUpdateChan := make(chan []string, 1)
		
		// Send async LSP request for document symbols
		request := lsp.LSPRequest{
			Type:       lsp.RequestDocumentSymbols,
			FilePath:   e.buffer.FilePath,
			Buffer:     e.buffer,
			Generation: e.buffer.SignatureGeneration,
			ResponseTo: symbolResponseChan,
		}
		
		// Send request to LSP worker (non-blocking)
		select {
		case e.lspRequestChan <- request:
			LL.Trace("Sent document symbols request for %s", e.buffer.FilePath)
		default:
			LL.Debug("LSP request channel full, dropping request")
			contentUpdateChan <- []string{"Error: Could not send request"}
		}
		
		// Start a goroutine to handle LSP response
		go func() {
			select {
			case response := <-symbolResponseChan:
				if response.Type == lsp.RequestDocumentSymbols {
					if response.Error != nil {
						LL.Debug("LSP document symbols error: %v", response.Error)
						contentUpdateChan <- []string{fmt.Sprintf("Error: %v", response.Error)}
					} else {
						LL.Debug("Got %d document symbols from LSP", len(response.DocumentSymbols))
						
						// Convert to display strings
						var displayNames []string
						newSymbolMap := make(map[string]int)
						
						// Symbol kind names
						kindNames := map[int]string{
							1: "File", 2: "Module", 3: "Namespace", 4: "Package",
							5: "Class", 6: "Method", 7: "Property", 8: "Field",
							9: "Constructor", 10: "Enum", 11: "Interface", 12: "Function",
							13: "Variable", 14: "Constant", 15: "String", 16: "Number",
							17: "Boolean", 18: "Array", 19: "Object", 20: "Key",
							21: "Null", 22: "EnumMember", 23: "Struct", 24: "Event",
							25: "Operator", 26: "TypeParameter",
						}
						
						for i, sym := range response.DocumentSymbols {
							kindName := kindNames[sym.Kind]
							if kindName == "" {
								kindName = fmt.Sprintf("Unknown(%d)", sym.Kind)
							}
							displayName := fmt.Sprintf("%s [%s] (line %d)", sym.Name, kindName, sym.Range.Start.Line+1)
							displayNames = append(displayNames, displayName)
							newSymbolMap[displayName] = i
						}
						
						// Update symbol data
						symbols = response.DocumentSymbols
						symbolMap = newSymbolMap
						
						if len(displayNames) == 0 {
							contentUpdateChan <- []string{"No symbols found"}
						} else {
							contentUpdateChan <- displayNames
						}
					}
				}
			case <-time.After(5 * time.Second):
				contentUpdateChan <- []string{"Error: Timeout waiting for symbols"}
			}
			close(symbolResponseChan)
		}()
		
		// Show the widget with async support (no text change handler needed)
		selection := fuzzyListWidget.ShowWithCallbacks(widgetScreen, func() {
			e.drawEditor()
			e.drawStatusBar()
		}, nil, contentUpdateChan)
		
		// Close the channel
		close(contentUpdateChan)
		
		// If a symbol was selected, navigate to it
		if selection != "" {
			// Find the selected symbol by display name
			if idx, exists := symbolMap[selection]; exists && idx < len(symbols) {
				sym := symbols[idx]
				// Navigate to location (document symbols are in the current file)
				e.navigateToLocation(sym.Range.Start.Line, sym.Range.Start.Character)
			}
		}
	}
}

// listDocumentSymbolsCmd implements the list-document-symbols command
func (e *Editor) listDocumentSymbolsCmd() error {
	LL.Debug("listDocumentSymbolsCmd called")
	e.showDocumentSymbolsList()
	return nil
}
