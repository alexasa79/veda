package editor

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/proxy"
	"veda/internal/proxyworker"
	"veda/internal/utils"
	"veda/internal/widgets"
)

// Package-level variables for efficient file filtering
var (
	binaryExtensions = map[string]bool{
		".o": true, ".a": true, ".so": true, ".dylib": true, ".dll": true, ".exe": true,
		".bin": true, ".out": true, ".class": true, ".pyc": true, ".pyo": true,
		".jar": true, ".war": true, ".ear": true, ".zip": true, ".tar": true, ".gz": true,
		".bz2": true, ".xz": true, ".7z": true, ".rar": true, ".deb": true, ".rpm": true,
		".dmg": true, ".pkg": true, ".msi": true, ".iso": true, ".img": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
		".svg": true, ".ico": true, ".tiff": true, ".webp": true, ".heic": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mkv": true, ".mov": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4a": true, ".flac": true,
		".wav": true, ".ogg": true, ".aac": true, ".wma": true, ".opus": true,
		".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
		".sqlite": true, ".db": true, ".mdb": true, ".accdb": true,
	}

	binaryNames = map[string]bool{
		"a.out": true, "core": true, "vgcore": true, "vd": true,
	}
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
	fuzzyListWidth := width * 4 / 5    // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("Open File")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationFileIcon)
	// Show loading message - actual connection status is shown in the status bar
	fuzzyListWidget.SetContent([]string{"Loading..."})

	return fuzzyListWidget
}

// sendDirectoryListingRequest sends an async request for directory listing
func (e *Editor) sendDirectoryListingRequest(contentUpdateChan chan []string, useCache bool) chan fileListingResult {
	// Start async operation tracking
	operationID := fmt.Sprintf("dir-list-%s", e.workingDirectory)
	e.startAsyncOperation(operationID, "listing directory: "+e.workingDirectory)

	// Set connection status
	host, _, isRemote := parsePathComponents(e.workingDirectory)
	if isRemote {
		e.connectionStatus = fmt.Sprintf("Listing files on %s", host)
	} else {
		e.connectionStatus = "Listing files"
	}

	// Create a channel for the result
	resultChan := make(chan fileListingResult, 1)

	// If we can satisfy from cache, do so synchronously
	if files, ok := e.getCachedDirectoryListing(e.workingDirectory); ok && useCache {
		LL.Debug("Using cached directory listing for %s", e.workingDirectory)
		result := e.processFileList(files)
		if len(result.paths) == 0 {
			contentUpdateChan <- []string{"No files found"}
		} else {
			contentUpdateChan <- result.paths
		}
		// End the async operation since we're serving from cache
		e.endAsyncOperation(operationID)
		e.connectionStatus = "" // Clear status when done
		// Return result immediately
		resultChan <- result
		return resultChan
	}

	// Start goroutine to fetch directory listing (and populate cache)
	go func() {
		defer e.endAsyncOperation(operationID)
		defer func() { e.connectionStatus = "" }() // Clear status when done

		LL.Debug("Fetching directory listing for %s", e.workingDirectory)
		files, err := e.proxyManager.ListDirectoryRecursive(e.workingDirectory)

		if err != nil {
			LL.Debug("Directory listing error: %v", err)
			contentUpdateChan <- []string{fmt.Sprintf("Error: %v", err)}
			resultChan <- fileListingResult{paths: []string{}, filePaths: nil}
			return
		}

		LL.Debug("Got %d files from proxy", len(files))
		// Store in cache
		e.setCachedDirectoryListing(e.workingDirectory, files)
		result := e.processFileList(files)

		if len(result.paths) == 0 {
			contentUpdateChan <- []string{"No files found"}
		} else {
			contentUpdateChan <- result.paths
		}

		resultChan <- result
	}()

	return resultChan
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
	if file.Type != "file" {
		return true
	}

	// Get the filename for extension checking
	filename := pathComponents[len(pathComponents)-1]

	// Filter out .d files (dependency files)
	if strings.HasSuffix(filename, ".d") {
		return true
	}

	// Check for binary file extensions
	dotIndex := strings.LastIndex(filename, ".")
	if dotIndex >= 0 {
		ext := strings.ToLower(filename[dotIndex:])
		if binaryExtensions[ext] {
			return true
		}
	}

	// Filter out files without extensions that are commonly binaries
	if !strings.Contains(filename, ".") {
		if binaryNames[filename] {
			return true
		}

		// Check if it might be an executable by looking at the name
		// Many compiled binaries don't have extensions
		// But we don't want to filter out all files without extensions
		// (like README, LICENSE, Makefile, etc.)
		// So we'll be conservative here
	}

	return false
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

// getCachedDirectoryListing returns cached list if available for a path
func (e *Editor) getCachedDirectoryListing(path string) ([]proxy.FileInfo, bool) {
	if e == nil || e.dirCache == nil {
		return nil, false
	}
	if files, ok := e.dirCache[path]; ok {
		return files, true
	}
	return nil, false
}

// setCachedDirectoryListing stores a directory listing in cache
func (e *Editor) setCachedDirectoryListing(path string, files []proxy.FileInfo) {
	if e == nil {
		return
	}
	if e.dirCache == nil {
		e.dirCache = make(map[string][]proxy.FileInfo)
	}
	// Copy slice to avoid external mutation
	copied := make([]proxy.FileInfo, len(files))
	copy(copied, files)
	e.dirCache[path] = copied
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

	// Create a channel for content updates
	contentUpdateChan := make(chan []string, 1)

	// Send async request for directory listing and get result channel
	resultChan := e.sendDirectoryListingRequest(contentUpdateChan, true)

	// Store file paths for mapping back from display names
	var filePaths map[string]string

	// Start a goroutine to handle the result
	go func() {
		result := <-resultChan
		filePaths = result.filePaths
		close(resultChan)
	}()

	// Create overlay and get result channel
	overlay := NewFuzzyListOverlay(fuzzyListWidget, nil, contentUpdateChan)
	overlayResultChan := e.renderingState.SetOverlay(overlay)

	// Wait for result in a goroutine
	go func() {
		selectedPath := <-overlayResultChan
		if selectedPath != "" {
			openInNewWin := false
			if strings.HasPrefix(selectedPath, "__SHIFT__\n") {
				openInNewWin = true
				selectedPath = strings.TrimPrefix(selectedPath, "__SHIFT__\n")
			}
			fullPath := e.resolveSelectedPath(selectedPath, filePaths)
			LL.Debug("Selected file: %s (full path: %s)", selectedPath, fullPath)
			if openInNewWin {
				// Create a new window and open file into it
				insertAt := e.activeWindow + 1
				// Temporarily switch active window to new one for openFile side effects
				if insertAt >= len(e.windows) {
					e.windows = append(e.windows, newWindow(e.buffer))
				} else {
					e.windows = append(e.windows[:insertAt+1], e.windows[insertAt:]...)
					e.windows[insertAt] = newWindow(e.buffer)
				}
				e.activateWindow(insertAt)
				e.openFile(fullPath)
			} else {
				e.openFile(fullPath)
			}
		}
		// Don't close contentUpdateChan - the sending goroutine might still be running
		// The channel will be garbage collected when no longer referenced
	}()
}

// startFuzzyFileListRefresh issues a fresh directory request bypassing cache
func (e *Editor) startFuzzyFileListRefresh(contentUpdateChan chan []string) chan fileListingResult {
	// Clear any cached entry for this directory to force fetch
	if e.dirCache != nil {
		delete(e.dirCache, e.workingDirectory)
	}
	return e.sendDirectoryListingRequest(contentUpdateChan, false)
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
	LL.Debug("openFile: Called with filePath=%s", filePath)
	// Resolve relative paths
	resolvedPath := e.resolvePath(filePath)
	LL.Debug("openFile: resolvedPath=%s", resolvedPath)

	// First save the current buffer state
	e.saveCurrentBufferState()

	// Check if this file is already open
	if buf, exists := e.buffers[resolvedPath]; exists {
		// Switch to existing buffer
		e.switchToBuffer(buf)
		return
	}

	// Read file content using proxy
	fileContent, err := e.proxyManager.GetFile(resolvedPath)
	var content string
	var changes *proxy.FileChanges
	var branch string
	var diagnostics []proxy.Diagnostic

	if err != nil {
		// Check if it's a file not found error
		if strings.Contains(err.Error(), "File does not exist") || strings.Contains(err.Error(), "file does not exist") {
			// Create an empty buffer for a new file
			content = ""
			changes = &proxy.FileChanges{Added: []int{}, Deleted: []int{}, Modified: []int{}}
			branch = "main"
			LL.Info("Creating new file: %s", resolvedPath)
		} else {
			// Other errors (permission denied, etc.) - show error and return
			LL.Error("Failed to open file %s: %v", resolvedPath, err)
			e.ShowError("Failed to open file: %v", err)
			return
		}
	} else {
		content = fileContent.Content
		changes = &fileContent.Changes
		branch = fileContent.Branch
		diagnostics = fileContent.Diagnostics

		// Check for auto-save recovery if we have a file
		if e.autoSaveManager != nil {
			// For now, check if auto-save exists without time comparison
			// TODO: Get file modification time from proxy for better recovery detection
			if exists, _, _ := e.autoSaveManager.GetAutoSaveInfo(resolvedPath); exists {
				// Present non-blocking recovery dialog using overlay, then finish open based on result
				width, height := e.screen.Size()
				dialogWidth := 60
				dialogHeight := 8
				dialogX := (width - dialogWidth) / 2
				dialogY := (height - dialogHeight) / 2

				message := fmt.Sprintf("Auto-save found for %s. Recover unsaved changes?", filepath.Base(resolvedPath))
				dialog := widgets.NewYesNoDialog(dialogX, dialogY, dialogWidth, dialogHeight, "Recovery", message, e.config.Theme)

				overlay := NewYesNoDialogOverlay(dialog)
				resultChan := e.renderingState.SetOverlay(overlay)

				// Defer finishing the open based on user's choice, then return immediately
				go func(origContent string, origChanges *proxy.FileChanges, origBranch string, origDiagnostics []proxy.Diagnostic) {
					res := <-resultChan
					// Close overlay handled by renderingState when result arrives
					finalContent := origContent
					if res == string(widgets.DialogResultYes) {
						if recoveredContent, err := e.autoSaveManager.GetAutoSaveContent(resolvedPath); err == nil {
							finalContent = recoveredContent
							LL.Info("Recovered content from auto-save for: %s", resolvedPath)
						} else {
							LL.Error("Failed to read auto-save content: %v", err)
						}
					} else if res == string(widgets.DialogResultNo) {
						// Clean up auto-save on explicit No
						e.autoSaveManager.CleanupAutoSave(resolvedPath)
					}
					e.finishOpenFile(resolvedPath, finalContent, origChanges, origBranch, origDiagnostics)
				}(content, changes, branch, diagnostics)

				return
			}
		}
	}

	// Finish opening immediately (no recovery dialog shown)
	e.finishOpenFile(resolvedPath, content, changes, branch, diagnostics)
}

// finishOpenFile finalizes opening a file into a buffer and switching to it
func (e *Editor) finishOpenFile(resolvedPath string, content string, changes *proxy.FileChanges, branch string, diagnostics []proxy.Diagnostic) {
	// Create a new buffer with the file content (empty for new files)
	newBuffer := buffer.NewBuffer(content, resolvedPath, e.config.DefaultIndentType, e.config.DefaultIndentWidth)

	LL.Debug("Setting buffer changes to %+v", changes)

	// Store the git changes, branch, and diagnostics in the buffer
	newBuffer.GitChanges = changes
	newBuffer.GitBranch = branch
	newBuffer.Diagnostics = diagnostics

	// Add to buffers map and order
	e.buffers[resolvedPath] = newBuffer

	// If this is replacing the empty buffer AND it's still empty, remove it
	// We should keep the unnamed buffer if the user has typed anything into it
	if e.buffer == nil || !e.buffer.HasFile() {
		if unnamedBuffer, exists := e.buffers["[No Name]"]; exists {
			// Only remove if the buffer is truly empty
			if unnamedBuffer.Rope.String() == "" {
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
	e.bufferOrder = append(e.bufferOrder, resolvedPath)

	// Switch to the new buffer
	e.switchToBuffer(newBuffer)

	// Detect indentation for the new file
	newBuffer.Indentation = e.detectIndentation()

	// Request folding ranges from LSP worker thread
	e.requestFoldingRanges(resolvedPath)

	// Enable LSP support and request a non-blocking diagnostics refresh
	if newBuffer.HasFile() {
		newBuffer.HasLSP = true
		e.requestDiagnosticsRefresh(resolvedPath)
	}
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
	fuzzyListWidth := width * 4 / 5    // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("Project Symbols")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationSymbol)

	// Start with empty content
	fuzzyListWidget.SetContent([]string{})

	var pendingQuery string              // Query that is currently being processed
	var symbols []proxy.ProjectSymbol    // Store the actual symbol objects
	var symbolMap = make(map[string]int) // Map display name to index

	// Create a channel to receive LSP responses for this widget
	symbolResponseChan := make(chan proxyworker.ProxyResponse, 10)

	// Create a channel for content updates
	contentUpdateChan := make(chan []string, 10)

	// Create a channel to pass symbol data back from the goroutine
	type symbolUpdate struct {
		symbols   []proxy.ProjectSymbol
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
					// Report error through editor's error system
					e.ShowError("LSP error: %v", response.Error)
					// Clear the list but don't show error in picker
					contentUpdateChan <- []string{}
					symbolUpdateChan <- symbolUpdate{nil, make(map[string]int)}
				} else {
					LL.Debug("Got %d symbols from LSP for query '%s'", len(response.ProjectSymbols), response.Query)

					// Convert to display strings with icons
					var displayNames []string
					newSymbolMap := make(map[string]int)

					for i, sym := range response.ProjectSymbols {
						// Get symbol icon based on kind
						iconInfo := utils.GetSymbolIconInfo(sym.Kind)
						// Format: icon symbol_name - file_path
						displayName := fmt.Sprintf("%s %s - %s", iconInfo.Icon, sym.RawName, sym.Path)
						displayNames = append(displayNames, displayName)
						newSymbolMap[displayName] = i
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
		// Send async LSP request for symbols (including empty queries)
		pendingQuery = newText
		LL.Debug("Sending async LSP query for symbols with: %s", newText)

		// Start async operation tracking
		operationID := fmt.Sprintf("project-symbols-%s", newText)
		e.startAsyncOperation(operationID, "searching project symbols: "+newText)

		// Set connection status
		if len(newText) > 0 {
			e.connectionStatus = fmt.Sprintf("Searching for \"%s\"", newText)
		} else {
			e.connectionStatus = "Loading all symbols"
		}

		// Launch goroutine to fetch symbols
		go func(query string) {
			defer e.endAsyncOperation(operationID)
			defer func() { e.connectionStatus = "" }() // Clear status when done

			projectSymbols, err := e.proxyManager.ListProjectSymbols(e.workingDirectory, query)

			response := proxyworker.ProxyResponse{
				Type:           proxyworker.RequestProjectSymbols,
				FilePath:       e.buffer.FilePath,
				Buffer:         e.buffer,
				Generation:     e.buffer.SignatureGeneration,
				ProjectSymbols: projectSymbols,
				Query:          query,
				Error:          err,
			}

			// Send response to the channel
			select {
			case symbolResponseChan <- response:
				LL.Trace("Sent project symbols response for query '%s'", query)
			default:
				LL.Debug("Symbol response channel full, dropping response")
			}
		}(newText)
	}

	// Create overlay and get result channel
	overlay := NewFuzzyListOverlay(fuzzyListWidget, onTextChanged, contentUpdateChan)
	resultChan := e.renderingState.SetOverlay(overlay)

	// Send initial empty query to load all symbols
	onTextChanged("")

	// Wait for result in a goroutine
	go func() {
		// Don't close any channels - the sending goroutines might still be running
		// The channels will be garbage collected when no longer referenced

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
	fuzzyListWidth := width * 4 / 5    // 80% of screen width
	fuzzyListHeight := height * 7 / 10 // 70% of screen height
	fuzzyListX := (width - fuzzyListWidth) / 2
	fuzzyListY := (height - fuzzyListHeight) / 2

	fuzzyListWidget := widgets.NewFuzzyListWidget(fuzzyListX, fuzzyListY, fuzzyListWidth, fuzzyListHeight, e.config.Theme)
	fuzzyListWidget.SetTitle("Document Symbols")
	fuzzyListWidget.SetDecorationMode(widgets.DecorationSymbol)

	// Start with loading message
	fuzzyListWidget.SetContent([]string{"Loading symbols..."})

	var symbols []proxy.DocumentSymbol   // Store the actual symbol objects
	var symbolMap = make(map[string]int) // Map display name to index

	// Create a channel to receive LSP responses for this widget
	symbolResponseChan := make(chan proxyworker.ProxyResponse, 1)

	// Create a channel for content updates
	contentUpdateChan := make(chan []string, 1)

	// Send async LSP request for document symbols
	operationID := fmt.Sprintf("doc-symbols-%s", e.buffer.FilePath)
	e.startAsyncOperation(operationID, "loading document symbols for: "+e.buffer.FilePath)

	// Set connection status
	e.connectionStatus = "Loading document symbols"

	// Launch goroutine to fetch document symbols
	go func() {
		documentSymbols, err := e.proxyManager.GetDocumentSymbols(e.buffer.FilePath)

		response := proxyworker.ProxyResponse{
			Type:            proxyworker.RequestDocumentSymbols,
			FilePath:        e.buffer.FilePath,
			Buffer:          e.buffer,
			Generation:      e.buffer.SignatureGeneration,
			DocumentSymbols: documentSymbols,
			Error:           err,
		}

		// Send response to the channel
		select {
		case symbolResponseChan <- response:
			LL.Trace("Sent document symbols response for %s", e.buffer.FilePath)
		default:
			LL.Debug("Symbol response channel full, dropping response")
		}
	}()

	// Start a goroutine to handle LSP response
	go func() {
		defer e.endAsyncOperation(operationID)
		defer func() { e.connectionStatus = "" }() // Clear status when done

		select {
		case response := <-symbolResponseChan:
			if response.Type == proxyworker.RequestDocumentSymbols {
				if response.Error != nil {
					LL.Debug("LSP document symbols error: %v", response.Error)
					// Report error through editor's error system
					e.ShowError("LSP error: %v", response.Error)
					// Clear the list but don't show error in picker
					contentUpdateChan <- []string{}
				} else {
					LL.Debug("Got %d document symbols from LSP", len(response.DocumentSymbols))

					// Convert to display strings with icons
					var displayNames []string
					newSymbolMap := make(map[string]int)

					for i, sym := range response.DocumentSymbols {
						// Get symbol icon based on kind
						iconInfo := utils.GetSymbolIconInfo(sym.Kind)
						// Format: icon symbol_name (line N)
						displayName := fmt.Sprintf("%s %s (line %d)", iconInfo.Icon, sym.Name, sym.Range.Start.Line+1)
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
		// Don't close any channels - the sending goroutines might still be running
		// The channels will be garbage collected when no longer referenced

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

// writeFile writes the current buffer to disk asynchronously with spinner
func (e *Editor) writeFile(args []string) error {
	var filePath string
	var saveAs bool

	// Determine the file path to use
	if len(args) > 0 {
		// Save as - use the provided filename
		filePath = e.resolvePath(args[0])
		saveAs = true
	} else if e.buffer.HasFile() {
		// Save to current file
		filePath = e.buffer.FilePath
		saveAs = false
	} else {
		// No filename and buffer has no file
		return fmt.Errorf("no file name")
	}

	// Get the content from the buffer
	content := e.buffer.Rope.String()

	// Save to local history first (don't fail the save if history fails)
	if e.localHistory != nil {
		if err := e.localHistory.SaveFileHistory(filePath, content); err != nil {
			LL.Error("Failed to save file history: %v", err)
			// Continue with the actual save even if history fails
		}
	}

	// Start async operation tracking
	operationID := fmt.Sprintf("save-file-%s", filePath)
	e.startAsyncOperation(operationID, "saving file: "+filePath)

	// Set connection status
	host, _, isRemote := parsePathComponents(filePath)
	if isRemote {
		e.connectionStatus = fmt.Sprintf("Saving to %s", host)
	} else {
		e.connectionStatus = "Saving file"
	}

	// Perform the save operation asynchronously
	go func() {
		defer e.endAsyncOperation(operationID)
		defer func() { e.connectionStatus = "" }() // Clear status when done

		// Write the file and get updated diagnostics
		diagnostics, err := e.proxyManager.WriteFile(filePath, content)
		if err != nil {
			LL.Error("Failed to write file %s: %v", filePath, err)
			e.ShowError("Failed to save file: %v", err)
			return
		}

		// Update buffer diagnostics with the returned diagnostics
		e.buffer.Diagnostics = diagnostics

		// If this is a "save as" operation, update the buffer's file path
		if saveAs {
			// Remove old buffer from maps if it had a file
			if e.buffer.HasFile() {
				delete(e.buffers, e.buffer.FilePath)
				// Update buffer order
				for i, path := range e.bufferOrder {
					if path == e.buffer.FilePath {
						e.bufferOrder[i] = filePath
						break
					}
				}
			} else {
				// This was an unnamed buffer, remove it and add the new one
				delete(e.buffers, "")
				for i, path := range e.bufferOrder {
					if path == "" {
						e.bufferOrder[i] = filePath
						break
					}
				}
			}

			// Update buffer's file path
			e.buffer.FilePath = filePath

			// Add buffer with new path
			e.buffers[filePath] = e.buffer
		}

		// Mark buffer as saved
		e.buffer.MarkSaved()

		// Clean up auto-save since file was successfully saved
		if e.autoSaveManager != nil {
			e.autoSaveManager.CleanupAutoSave(filePath)
		}

		// Show success message
		LL.Info("Wrote %s", filePath)
		e.ShowInfo("File saved: %s", filePath)
	}()

	return nil
}

// writeAndQuit implements the :wq command to write the file and close the buffer
func (e *Editor) writeAndQuit(args []string) error {
	// First write the file
	if err := e.writeFile(args); err != nil {
		return err
	}

	// Then close the buffer (which will quit if it's the last buffer)
	return e.closeBuffer(args)
}

// editFile implements the :edit/:e command to open a file or directory
func (e *Editor) editFile(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no file name")
	}

	path := args[0]

	// Check if it's a directory by attempting to list it
	_, err := e.proxyManager.ListDirectory(path)
	if err == nil {
		// It's a directory - update working directory and show file browser
		e.workingDirectory = path
		LL.Info("Changed directory to: %s", e.workingDirectory)
		e.ShowFuzzyListWidget()
		return nil
	}

	// It's a file - open it
	e.openFile(path)
	return nil
}

// showRecoveryDialog displays a dialog asking the user whether to recover from auto-save
func (e *Editor) showRecoveryDialog(filePath string) bool {
	width, height := e.screen.Size()

	// Create a centered dialog
	dialogWidth := 60
	dialogHeight := 8
	dialogX := (width - dialogWidth) / 2
	dialogY := (height - dialogHeight) / 2

	message := fmt.Sprintf("Auto-save found for %s. Recover unsaved changes?", filepath.Base(filePath))

	dialog := widgets.NewYesNoDialog(dialogX, dialogY, dialogWidth, dialogHeight,
		"Recovery", message, e.config.Theme)

	// Show the dialog and wait for user input
	for !dialog.IsClosed() {
		// Clear screen and render dialog
		e.screen.Clear()
		dialog.Render(e.screen)
		e.screen.Show()

		// Handle events
		event := e.screen.PollEvent()
		dialog.HandleEvent(event)
	}

	result := dialog.GetResult()
	return result == widgets.DialogResultYes
}

// performAutoSave performs auto-save for a specific file
func (e *Editor) performAutoSave(filePath string) {
	// Find the buffer for this file
	if buf, exists := e.buffers[filePath]; exists {
		content := buf.Rope.String()
		if err := e.autoSaveManager.saveFile(filePath, content); err != nil {
			LL.Error("Failed to auto-save file %s: %v", filePath, err)
		} else {
			// Update last save time
			e.autoSaveManager.mutex.Lock()
			e.autoSaveManager.lastSaveTimes[filePath] = time.Now()
			e.autoSaveManager.mutex.Unlock()
		}
	}
}

// markBufferDirty marks a buffer as having unsaved changes for auto-save
func (e *Editor) markBufferDirty() {
	if e.autoSaveManager != nil && e.buffer.HasFile() {
		e.autoSaveManager.MarkDirty(e.buffer.FilePath)
	}
}
