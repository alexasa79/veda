package editor

import (
	"fmt"
	"strings"
	"time"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/proxyworker"
	"veda/internal/proxy"
	"veda/internal/widgets"
)

// fileListingResult contains the processed file paths
type fileListingResult struct {
	paths     []string
	filePaths map[string]string
}

// createFuzzyListWidget creates and configures the fuzzy list widget
func (e *Editor) createFuzzyListWidget() *widgets.FuzzyListWidget {
	width, height := e.screen.Size()

	// Create fuzzy list widget (centered overlay)
	fuzzyListWidth := width * 4 / 5  // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("Open File")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationFileIcon)
	// Show appropriate loading message
	host, _, isRemote := parsePathComponents(e.workingDirectory)
	if isRemote {
		fuzzyListWidget.SetContent([]string{fmt.Sprintf("Connecting to %s...", host)})
	} else {
		fuzzyListWidget.SetContent([]string{"Loading directory..."})
	}

	return fuzzyListWidget
}

// sendDirectoryListingRequest sends an async request for directory listing
func (e *Editor) sendDirectoryListingRequest(fileResponseChan chan proxyworker.ProxyResponse, contentUpdateChan chan []string) {
	// Start async operation tracking
	operationID := fmt.Sprintf("dir-list-%s", e.workingDirectory)
	e.startAsyncOperation(operationID)
	
	request := proxyworker.ProxyRequest{
		Type:          proxyworker.RequestListDirectoryRecursive,
		DirectoryPath: e.workingDirectory,
		ResponseTo:    fileResponseChan,
	}

	select {
	case e.lspRequestChan <- request:
		LL.Debug("Sent directory listing request for %s", e.workingDirectory)
	default:
		LL.Debug("Proxy request channel full, dropping request")
		contentUpdateChan <- []string{"Error: Could not send request"}
		e.endAsyncOperation(operationID)
	}
}

// shouldSkipFile checks if a file should be filtered out
func shouldSkipFile(file proxy.FileInfo) bool {
	// Filter out hidden files and directories (starting with .)
	pathComponents := strings.Split(file.Path, "/")
	for _, component := range pathComponents {
		if len(component) > 0 && component[0] == '.' {
			return true
		}
	}

	// Only include files, not directories
	return file.Type != "file"
}

// calculateDisplayPath converts a file path to its display representation
func (e *Editor) calculateDisplayPath(filePath string, basePath string) string {
	displayPath := filePath

	if basePath == "~/" || basePath == "~" {
		// For home directory, just show paths relative to home
		// The proxy returns absolute paths like /home/user/file.txt
		// We want to show them as file.txt or subdir/file.txt
		homePrefix := "/home/"
		if idx := strings.Index(filePath, homePrefix); idx >= 0 {
			// Find the next slash after /home/username/
			afterHome := filePath[idx+len(homePrefix):]
			if slashIdx := strings.Index(afterHome, "/"); slashIdx >= 0 {
				displayPath = afterHome[slashIdx+1:]
			}
		}
	} else if strings.HasPrefix(filePath, basePath) {
		// For other paths, make relative as before
		displayPath = strings.TrimPrefix(filePath, basePath)
		displayPath = strings.TrimPrefix(displayPath, "/")
	}

	return displayPath
}

// processFileList processes the raw file list from proxy into display paths
func (e *Editor) processFileList(files []proxy.FileInfo) fileListingResult {
	var paths []string
	filePaths := make(map[string]string)

	// Extract the base path for making relative paths
	_, basePath, isRemote := parsePathComponents(e.workingDirectory)
	if !isRemote {
		basePath = e.workingDirectory
	}
	LL.Debug("File browser: workingDirectory=%s, basePath=%s, isRemote=%v, fileCount=%d", e.workingDirectory, basePath, isRemote, len(files))

	for _, file := range files {
		if shouldSkipFile(file) {
			continue
		}

		displayPath := e.calculateDisplayPath(file.Path, basePath)
		paths = append(paths, displayPath)
		filePaths[displayPath] = file.Path
	}

	return fileListingResult{
		paths:     paths,
		filePaths: filePaths,
	}
}

// handleDirectoryListingResponse processes the async response from proxy
func (e *Editor) handleDirectoryListingResponse(fileResponseChan chan proxyworker.ProxyResponse, contentUpdateChan chan []string) map[string]string {
	var filePaths map[string]string
	
	// End async operation tracking when done
	operationID := fmt.Sprintf("dir-list-%s", e.workingDirectory)
	defer e.endAsyncOperation(operationID)
	
	LL.Debug("Waiting for directory listing response...")

	select {
	case response := <-fileResponseChan:
		LL.Debug("Received response: type=%v, error=%v", response.Type, response.Error)
		if response.Type == proxyworker.RequestListDirectoryRecursive {
			if response.Error != nil {
				LL.Debug("Directory listing error: %v", response.Error)
				contentUpdateChan <- []string{fmt.Sprintf("Error: %v", response.Error)}
			} else {
				LL.Debug("Got %d files from proxy", len(response.Files))

				result := e.processFileList(response.Files)
				filePaths = result.filePaths

				if len(result.paths) == 0 {
					contentUpdateChan <- []string{"No files found"}
				} else {
					contentUpdateChan <- result.paths
				}
			}
		}
	case <-time.After(10 * time.Second):
		LL.Debug("Timeout waiting for directory listing response")
		contentUpdateChan <- []string{"Error: Timeout waiting for response"}
	}

	return filePaths
}

// resolveSelectedPath converts the selected display path back to full path
func (e *Editor) resolveSelectedPath(selectedPath string, filePaths map[string]string) string {
	// Get the full path from our mapping
	fullPath := ""
	if filePaths != nil {
		if mapped, ok := filePaths[selectedPath]; ok {
			fullPath = mapped
		}
	}

	// If no mapping found (shouldn't happen), reconstruct path
	if fullPath == "" {
		fullPath = selectedPath
		if !strings.HasPrefix(selectedPath, "/") {
			// Make it absolute again
			_, basePath, isRemote := parsePathComponents(e.workingDirectory)
			if !isRemote {
				basePath = e.workingDirectory
			}
			fullPath = basePath + "/" + selectedPath
		}
	}

	// If it's a remote path, prepend the host
	host, _, isRemote := parsePathComponents(e.workingDirectory)
	if isRemote {
		// Only prepend host if not already present
		if !strings.Contains(fullPath, ":") {
			fullPath = host + ":" + fullPath
		}
	}

	return fullPath
}

// ShowFuzzyListWidget displays a fuzzy search widget for file selection
func (e *Editor) ShowFuzzyListWidget() {
	fuzzyListWidget := e.createFuzzyListWidget()

	// Create a channel to receive proxy responses for this widget
	fileResponseChan := make(chan proxyworker.ProxyResponse, 1)

	// Create a channel for content updates
	contentUpdateChan := make(chan []string, 1)

	// Send async proxy request for directory listing
	e.sendDirectoryListingRequest(fileResponseChan, contentUpdateChan)

	// Store file paths for mapping back from display names
	var filePaths map[string]string

	// Start a goroutine to handle proxy response
	go func() {
		filePaths = e.handleDirectoryListingResponse(fileResponseChan, contentUpdateChan)
		close(fileResponseChan)
	}()

	// Create overlay and get result channel
	overlay := NewFuzzyListOverlay(fuzzyListWidget, nil, contentUpdateChan)
	resultChan := e.renderingState.SetOverlay(overlay)

	// Wait for result in a goroutine
	go func() {
		selectedPath := <-resultChan
		if selectedPath != "" {
			fullPath := e.resolveSelectedPath(selectedPath, filePaths)
			LL.Debug("Selected file: %s (full path: %s)", selectedPath, fullPath)
			e.openFile(fullPath)
		}
		// Close channel
		close(contentUpdateChan)
	}()
}

// OpenFile loads a file's content into the editor buffer (exported for main.go)
func (e *Editor) OpenFile(filePath string) {
	e.openFile(filePath)
}

// OpenPath opens either a file or shows the fuzzy list for a directory
func (e *Editor) OpenPath(path string) {
	// Store the path to be processed after the UI is ready
	e.pendingPath = path
	LL.Info("Deferred opening of path: %s", path)
}

// openFile loads a file's content into the editor buffer
func (e *Editor) openFile(filePath string) {
	// Resolve relative paths
	resolvedPath := e.resolvePath(filePath)

	// First save the current buffer state
	e.saveCurrentBufferState()

	// Check if this file is already open
	if buf, exists := e.buffers[resolvedPath]; exists {
		// Switch to existing buffer
		e.switchToBuffer(buf)
		return
	}

	// Read file content using proxy
	content, err := e.proxyManager.GetFile(resolvedPath)
	if err != nil {
		// TODO: Show error to user
		return
	}

	// Create a new buffer with the file content
	newBuffer := buffer.NewBuffer(content, resolvedPath, e.config.DefaultIndentType, e.config.DefaultIndentWidth)

	// Add to buffers map and order
	e.buffers[resolvedPath] = newBuffer

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
	e.bufferOrder = append(e.bufferOrder, resolvedPath)

	// Switch to the new buffer
	e.switchToBuffer(newBuffer)

	// Detect indentation for the new file
	newBuffer.Indentation = e.detectIndentation()

	// Request folding ranges from LSP worker thread
	e.requestFoldingRanges(resolvedPath)
}

// openFileCmd implements the open-file command
func (e *Editor) openFileCmd(args []string) error {
	e.ShowFuzzyListWidget()
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
	
	// Track the last query to avoid redundant LSP calls
	var lastQuery string
	var pendingQuery string // Query that is currently being processed
	var symbols []proxy.ProjectSymbol // Store the actual symbol objects
	var symbolMap = make(map[string]int) // Map display name to index

	// Create a channel to receive LSP responses for this widget
	symbolResponseChan := make(chan proxyworker.ProxyResponse, 10)

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
			if response.Type == proxyworker.RequestProjectSymbols && response.Query == pendingQuery {
				// End async operation for this query
				operationID := fmt.Sprintf("project-symbols-%s", response.Query)
				e.endAsyncOperation(operationID)
				
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
				
				// Start async operation tracking
				operationID := fmt.Sprintf("project-symbols-%s", newText)
				e.startAsyncOperation(operationID)

				request := proxyworker.ProxyRequest{
					Type:       proxyworker.RequestProjectSymbols,
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
					e.endAsyncOperation(operationID)
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

	// Create overlay and get result channel
	overlay := NewFuzzyListOverlay(fuzzyListWidget, onTextChanged, contentUpdateChan)
	resultChan := e.renderingState.SetOverlay(overlay)

	// Wait for result in a goroutine
	go func() {
		defer func() {
			// Close the channels when done
			close(symbolResponseChan)
			close(contentUpdateChan)
		}()
		
		selection := <-resultChan

		// If a symbol was selected, navigate to it
		if selection != "" {
			// Find the selected symbol by display name
			if idx, exists := symbolMap[selection]; exists && idx < len(symbols) {
				sym := symbols[idx]

				// Push current position to navigation history before jumping
				e.pushNavigationHistory()

				// Open file if different
				if sym.Path != e.buffer.FilePath {
					e.openFile(sym.Path)
				}
				// Navigate to location
				e.navigateToLocation(sym.Line, sym.Character)
			}
		}
	}()
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
func (e *Editor) listProjectSymbolsCmd(args []string) error {
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

	var symbols []proxy.DocumentSymbol // Store the actual symbol objects
	var symbolMap = make(map[string]int) // Map display name to index

	// Create a channel to receive LSP responses for this widget
	symbolResponseChan := make(chan proxyworker.ProxyResponse, 1)

	// Create a channel for content updates
	contentUpdateChan := make(chan []string, 1)

	// Send async LSP request for document symbols
	operationID := fmt.Sprintf("doc-symbols-%s", e.buffer.FilePath)
	e.startAsyncOperation(operationID)
	
	request := proxyworker.ProxyRequest{
		Type:       proxyworker.RequestDocumentSymbols,
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
		e.endAsyncOperation(operationID)
	}

	// Start a goroutine to handle LSP response
	go func() {
		defer e.endAsyncOperation(operationID)
		
		select {
		case response := <-symbolResponseChan:
			if response.Type == proxyworker.RequestDocumentSymbols {
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
		// Don't close symbolResponseChan here - it's managed by the caller
	}()

	// Create overlay and get result channel
	overlay := NewFuzzyListOverlay(fuzzyListWidget, nil, contentUpdateChan)
	resultChan := e.renderingState.SetOverlay(overlay)

	// Wait for result in a goroutine
	go func() {
		defer func() {
			// Close the channels when done
			close(contentUpdateChan)
			close(symbolResponseChan)
		}()
		
		selection := <-resultChan

		// If a symbol was selected, navigate to it
		if selection != "" {
			// Find the selected symbol by display name
			if idx, exists := symbolMap[selection]; exists && idx < len(symbols) {
				sym := symbols[idx]
				// Push current position to navigation history before jumping
				e.pushNavigationHistory()
				// Navigate to location (document symbols are in the current file)
				e.navigateToLocation(sym.Range.Start.Line, sym.Range.Start.Character)
			}
		}
	}()
}

// listDocumentSymbolsCmd implements the list-document-symbols command
func (e *Editor) listDocumentSymbolsCmd(args []string) error {
	LL.Debug("listDocumentSymbolsCmd called")
	e.showDocumentSymbolsList()
	return nil
}
