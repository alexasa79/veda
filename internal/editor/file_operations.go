package editor

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/events"
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
	return e.sendDirectoryListingRequestFiltered(contentUpdateChan, useCache, "supported")
}

// sendDirectoryListingRequestFiltered sends an async request for directory listing with filter mode
func (e *Editor) sendDirectoryListingRequestFiltered(contentUpdateChan chan []string, useCache bool, filterMode string) chan fileListingResult {
	// Start async operation tracking
	operationID := fmt.Sprintf("dir-list-%s-%s", e.workingDirectory, filterMode)
	e.startAsyncOperation(operationID, "listing directory: "+e.workingDirectory)

	// Create a channel for the result
	resultChan := make(chan fileListingResult, 1)

	// Cache key includes filter mode
	cacheKey := e.workingDirectory + ":" + filterMode

	// If we can satisfy from cache, do so synchronously
	if files, ok := e.getCachedDirectoryListing(cacheKey); ok && useCache {
		LL.Debug("Using cached directory listing for %s (filter=%s)", e.workingDirectory, filterMode)
		result := e.processFileList(files)
		if len(result.paths) == 0 {
			contentUpdateChan <- []string{"No files found"}
		} else {
			contentUpdateChan <- result.paths
		}
		e.asyncEvents <- events.NewCustomEvent("overlay-update", nil)
		// End the async operation since we're serving from cache
		e.endAsyncOperation(operationID)
		// Return result immediately
		resultChan <- result
		return resultChan
	}

	// Start goroutine to fetch directory listing (and populate cache)
	go func() {
		defer e.endAsyncOperation(operationID)

		LL.Debug("Fetching directory listing for %s (filter=%s)", e.workingDirectory, filterMode)
		files, err := e.proxyManager.ListDirectoryRecursiveFiltered(e.workingDirectory, filterMode)

		if err != nil {
			LL.Debug("Directory listing error: %v", err)
			// Also show a banner so the user notices
			e.ShowError("Directory listing failed: %v", err)
			contentUpdateChan <- []string{fmt.Sprintf("Error: %v", err)}
			e.asyncEvents <- events.NewCustomEvent("overlay-update", nil)
			resultChan <- fileListingResult{paths: []string{}, filePaths: nil}
			return
		}

		LL.Debug("Got %d files from proxy", len(files))
		// Store in cache with filter mode as part of the key
		e.setCachedDirectoryListing(cacheKey, files)
		result := e.processFileList(files)

		if len(result.paths) == 0 {
			contentUpdateChan <- []string{"No files found"}
		} else {
			contentUpdateChan <- result.paths
		}
		e.asyncEvents <- events.NewCustomEvent("overlay-update", nil)

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

	// Start with supported files filter by default
	filterMode := "supported"

	// Send async request for directory listing and get result channel
	resultChan := e.sendDirectoryListingRequestFiltered(contentUpdateChan, true, filterMode)

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
	overlay.SetFilterMode(filterMode)
	overlayResultChan := e.renderingState.SetOverlay(overlay)

	// Wait for result in a goroutine; post a lambda to the main loop
	go func() {
		for {
			selectedPath := <-overlayResultChan
			if selectedPath == "" {
				return
			}

			// Handle tab key to switch filter mode
			if selectedPath == "__TAB__" {
				// Toggle filter mode
				if filterMode == "supported" {
					filterMode = "all"
				} else {
					filterMode = "supported"
				}
				overlay.SetFilterMode(filterMode)

				// Clear cache and request new listing with new filter
				if e.dirCache != nil {
					delete(e.dirCache, e.workingDirectory)
				}
				newResultChan := e.sendDirectoryListingRequestFiltered(contentUpdateChan, false, filterMode)

				// Update file paths in a goroutine
				go func() {
					result := <-newResultChan
					filePaths = result.filePaths
					close(newResultChan)
				}()
				continue
			}

			openInNewWin := false
			if strings.HasPrefix(selectedPath, "__SHIFT__\n") {
				openInNewWin = true
				selectedPath = strings.TrimPrefix(selectedPath, "__SHIFT__\n")
			}
			fullPath := e.resolveSelectedPath(selectedPath, filePaths)
			LL.Debug("Selected file: %s (full path: %s)", selectedPath, fullPath)
			e.Post(func(e *Editor) {
				if openInNewWin {
					// Shift+Enter: always open file in a newly created rightmost window
					e.createRightmostWindow(nil)
					e.openFile(fullPath)
					e.recomputeWindowsLayout()
					return
				}
				e.openFile(fullPath)
			})
			break // Exit the loop after opening a file
		}
		// Don't close contentUpdateChan - the sending goroutine might still be running
		// The channel will be garbage collected when no longer referenced
	}()
}

// startFuzzyFileListRefresh issues a fresh directory request bypassing cache
func (e *Editor) startFuzzyFileListRefresh(contentUpdateChan chan []string) chan fileListingResult {
	return e.startFuzzyFileListRefreshFiltered(contentUpdateChan, "supported")
}

// startFuzzyFileListRefreshFiltered issues a fresh directory request with filter mode bypassing cache
func (e *Editor) startFuzzyFileListRefreshFiltered(contentUpdateChan chan []string, filterMode string) chan fileListingResult {
	// Clear any cached entry for this directory and filter combination to force fetch
	if e.dirCache != nil {
		cacheKey := e.workingDirectory + ":" + filterMode
		delete(e.dirCache, cacheKey)
	}
	return e.sendDirectoryListingRequestFiltered(contentUpdateChan, false, filterMode)
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
	var lspError string

	if err != nil {
		// Surface the error to the user, but still fall back to creating a new buffer
		e.ShowError("Failed to open file: %v", err)
		// On any error, fall back to creating a new empty buffer for this path.
		// This mirrors typical editor behavior (open-to-create) and makes tests robust
		// even when the proxy isn't available.
		content = ""
		changes = &proxy.FileChanges{Added: []int{}, Deleted: []int{}, Modified: []int{}}
		branch = "main"
		lspError = ""
		LL.Info("Creating new file (fallback due to open error %v): %s", err, resolvedPath)
	} else {
		content = fileContent.Content
		changes = &fileContent.Changes
		branch = fileContent.Branch
		diagnostics = fileContent.Diagnostics
		lspError = fileContent.LSPError

		// Check for auto-save recovery if we have a file that actually exists
		// If the file doesn't exist (proxy returned empty content), skip auto-save recovery
		if e.autoSaveManager != nil && content != "" {
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

				// Handle user's choice asynchronously by updating the already-open buffer
				go func(origContent string) {
					res := <-resultChan
					finalContent := origContent
					if res == string(widgets.DialogResultYes) {
						if recoveredContent, err := e.autoSaveManager.GetAutoSaveContent(resolvedPath); err == nil {
							finalContent = recoveredContent
							LL.Info("Recovered content from auto-save for: %s", resolvedPath)
						} else {
							LL.Error("Failed to read auto-save content: %v", err)
						}
					} else if res == string(widgets.DialogResultNo) {
						e.autoSaveManager.CleanupAutoSave(resolvedPath)
					}

					// Update buffer content in place if it is still open
					if buf, exists := e.buffers[resolvedPath]; exists {
						// Replace content
						buf.Rope = buffer.NewRopeFromString(finalContent)
						// Reset iterators and cursor
						buf.ViewIterator = buffer.NewBufferIterator(buf)
						buf.CursorIterator = buffer.NewBufferIterator(buf)
						buf.CursorX = 0
						// Re-parse syntax and re-render
						e.updateSyntaxHighlighting()
						e.render()
					}
				}(content)
			}
		}
	}

	// Finish opening immediately (no recovery dialog shown)
	e.finishOpenFile(resolvedPath, content, changes, branch, diagnostics, lspError)
}

// finishOpenFile finalizes opening a file into a buffer and switching to it
func (e *Editor) finishOpenFile(resolvedPath string, content string, changes *proxy.FileChanges, branch string, diagnostics []proxy.Diagnostic, lspError string) {
	// Create a new buffer with the file content (empty for new files)
	newBuffer := buffer.NewBuffer(content, resolvedPath, resolvedPath, e.config.DefaultIndentType, e.config.DefaultIndentWidth)

	LL.Debug("Setting buffer changes to %+v", changes)

	// Store the git changes, branch, and diagnostics in the buffer
	newBuffer.GitChanges = changes
	newBuffer.GitBranch = branch
	newBuffer.Diagnostics = diagnostics

	// Add to buffers map and order
	e.buffers[resolvedPath] = newBuffer

	// If this is replacing the empty buffer AND it's still empty, remove it
	// We should keep the unnamed buffer if the user has typed anything into it
	if e.CurrentBuffer() == nil || !e.CurrentBuffer().HasFile() {
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

	// Ensure the active window owns this buffer so moves/removals work correctly
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
		if win := e.windows[e.activeWindow]; win != nil {
			win.addBufferToWindow(newBuffer)
			e.assertBufferInWindow(newBuffer, e.activeWindow)
		}
	}

	// Detect indentation for the new file
	newBuffer.Indentation = e.detectIndentation()

	// Request folding ranges from LSP worker thread
	e.requestFoldingRanges(resolvedPath)

	// Enable LSP support and request a non-blocking diagnostics refresh
	if newBuffer.HasFile() {
		newBuffer.HasLSP = true

		// Check if LSP startup failed
		if lspError != "" {
			newBuffer.LSPFailed = true
			e.ShowError("LSP server failed to start: %s", lspError)
		} else {
			e.requestDiagnosticsRefresh(resolvedPath)
		}
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

	var latestSeq int64                  // Monotonic sequence for requests; only latest displayed
	var symbols []proxy.ProjectSymbol    // Store the actual symbol objects
	var symbolMap = make(map[string]int) // Map display name to index

	// Wrap responses with originating request sequence
	type respWithSeq struct {
		resp proxyworker.ProxyResponse
		seq  int64
	}
	// Create a channel to receive LSP responses for this widget
	symbolResponseChan := make(chan respWithSeq, 10)

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
		for wrapped := range symbolResponseChan {
			response := wrapped.resp
			// Only accept the latest in-flight response
			if wrapped.seq != atomic.LoadInt64(&latestSeq) {
				continue
			}
			if response.Type == proxyworker.RequestProjectSymbols {
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

					// If working directory is remote, prefix symbol paths with host
					host, _, isRemote := parsePathComponents(e.workingDirectory)

					// Convert to display strings with icons; adjust paths when remote
					var displayNames []string
					newSymbolMap := make(map[string]int)
					adjusted := make([]proxy.ProjectSymbol, len(response.ProjectSymbols))

					for i, sym := range response.ProjectSymbols {
						adj := sym
						if isRemote && host != "" && !strings.Contains(sym.Path, ":") {
							adj.Path = host + ":" + sym.Path
						}
						// Get symbol icon based on kind
						iconInfo := utils.GetSymbolIconInfo(sym.Kind)
						// Format: icon symbol_name - file_path
						displayName := fmt.Sprintf("%s %s - %s", iconInfo.Icon, adj.RawName, adj.Path)
						displayNames = append(displayNames, displayName)
						newSymbolMap[displayName] = i
						adjusted[i] = adj
					}

					// Send the symbol data update
					symbolUpdateChan <- symbolUpdate{adjusted, newSymbolMap}

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
		// Bump request sequence and mark as latest
		seq := atomic.AddInt64(&latestSeq, 1)
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
		go func(query string, reqSeq int64) {
			defer e.endAsyncOperation(operationID)
			defer func() { e.connectionStatus = "" }() // Clear status when done

			projectSymbols, err := e.proxyManager.ListProjectSymbols(e.workingDirectory, query)

			// Snapshot buffer-related fields safely; buffer can be nil
			var filePath string
			var gen uint64
			var buf *buffer.Buffer
			if e.CurrentBuffer() != nil {
				buf = e.CurrentBuffer()
				filePath = e.CurrentBuffer().FilePath
				gen = e.CurrentBuffer().SignatureGeneration
			}

			response := proxyworker.ProxyResponse{
				Type:           proxyworker.RequestProjectSymbols,
				FilePath:       filePath,
				Buffer:         buf,
				Generation:     gen,
				ProjectSymbols: projectSymbols,
				Query:          query,
				Error:          err,
			}

			// Send response to the channel
			select {
			case symbolResponseChan <- respWithSeq{resp: response, seq: reqSeq}:
				LL.Trace("Sent project symbols response for query '%s'", query)
			default:
				LL.Debug("Symbol response channel full, dropping response")
			}
		}(newText, seq)
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

				// Send open-file-and-navigate event to main loop
				e.asyncEvents <- events.NewCustomEvent("open-file-and-navigate", map[string]interface{}{
					"file":      sym.Path,
					"line":      sym.Line,
					"character": sym.Character,
				})
			}
		}
	}()
}

// navigateToLocation moves cursor to specific line and character position
func (e *Editor) navigateToLocation(line, character int) {
	// Log requested navigation
	LL.Debug("navigateToLocation: request file=%s targetLine=%d targetChar=%d", e.CurrentBuffer().FilePath, line, character)

	targetLine := line // LSP line numbers are 0-based
	e.CurrentBuffer().CursorX = character

	// Create iterator and advance to target line
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(e.CurrentBuffer())

	// Move iterator to target line
	for i := 0; i < targetLine && !e.CurrentBuffer().CursorIterator.IsEOF(); i++ {
		e.CurrentBuffer().CursorIterator = e.CurrentBuffer().CursorIterator.Next()
	}

	// Ensure cursor X is within bounds
	currentLine := e.CurrentBuffer().CursorIterator.Line()
	if e.CurrentBuffer().CursorX > len(currentLine) {
		e.CurrentBuffer().CursorX = len(currentLine)
		if e.CurrentBuffer().CursorX > 0 && e.mode == commands.NormalMode {
			e.CurrentBuffer().CursorX-- // Don't go past last character in normal mode
		}
	}

	// Center the target for better visibility
	e.scrollToCenter()

	// Update window's cached state to match buffer
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
		win := e.windows[e.activeWindow]
		if win != nil && win.Buffer == e.CurrentBuffer() {
			win.ViewLine = e.CurrentBuffer().ViewIterator.LineNumber()
			win.CursorLine = e.CurrentBuffer().CursorIterator.LineNumber()
			win.CursorX = e.CurrentBuffer().CursorX
		}
	}

	// Log result state
	LL.Debug("navigateToLocation: result file=%s cursorLine=%d cursorX=%d viewTop=%d",
		e.CurrentBuffer().FilePath,
		e.CurrentBuffer().CursorIterator.LineNumber(),
		e.CurrentBuffer().CursorX,
		e.CurrentBuffer().ViewIterator.LineNumber(),
	)
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
	operationID := fmt.Sprintf("doc-symbols-%s", e.CurrentBuffer().FilePath)
	e.startAsyncOperation(operationID, "loading document symbols for: "+e.CurrentBuffer().FilePath)

	// Set connection status
	e.connectionStatus = "Loading document symbols"

	// Launch goroutine to fetch document symbols
	go func() {
		documentSymbols, err := e.proxyManager.GetDocumentSymbols(e.CurrentBuffer().FilePath)

		response := proxyworker.ProxyResponse{
			Type:            proxyworker.RequestDocumentSymbols,
			FilePath:        e.CurrentBuffer().FilePath,
			Buffer:          e.CurrentBuffer(),
			Generation:      e.CurrentBuffer().SignatureGeneration,
			DocumentSymbols: documentSymbols,
			Error:           err,
		}

		// Send response to the channel
		select {
		case symbolResponseChan <- response:
			LL.Trace("Sent document symbols response for %s", e.CurrentBuffer().FilePath)
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
					e.asyncEvents <- events.NewCustomEvent("overlay-update", nil)
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
					e.asyncEvents <- events.NewCustomEvent("overlay-update", nil)
				}
			}
		case <-time.After(5 * time.Second):
			contentUpdateChan <- []string{"Error: Timeout waiting for symbols"}
			e.asyncEvents <- events.NewCustomEvent("overlay-update", nil)
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
				LL.Debug("doc-symbol-select: file=%s symbol=%q kind=%d at %d:%d", e.CurrentBuffer().FilePath, sym.Name, sym.Kind, sym.Range.Start.Line, sym.Range.Start.Character)
				// Push current position to navigation history before jumping
				e.pushNavigationHistory()
				// Send navigation event to main loop instead of navigating directly
				e.asyncEvents <- events.NewCustomEvent("navigate-to-location", map[string]interface{}{
					"line":      sym.Range.Start.Line,
					"character": sym.Range.Start.Character,
				})
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
	// Capture the target buffer at invocation time; the current buffer may change later.
	buf := e.CurrentBuffer()
	if buf == nil {
		return fmt.Errorf("no active buffer")
	}

	var filePath string
	var saveAs bool

	// Determine the file path to use
	if len(args) > 0 {
		// Save as - use the provided filename
		filePath = e.resolvePath(args[0])
		saveAs = true
	} else if buf.HasFile() {
		// Save to current file (prefer real filesystem path)
		if buf.RealPath != "" {
			filePath = buf.RealPath
		} else {
			filePath = buf.FilePath
		}
		saveAs = false
	} else {
		// No filename and buffer has no file
		return fmt.Errorf("no file name")
	}

	// Remove trailing spaces from all changed lines before saving
	buf.RemoveTrailingSpacesFromChangedLines()

	// Get the content from the captured buffer
	content := buf.Rope.String()

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

	// Track this save operation
	e.pendingSaves.Add(1)

	// Perform the save operation asynchronously
	go func(target *buffer.Buffer) {
		defer e.pendingSaves.Done() // Always mark save as done
		defer e.endAsyncOperation(operationID)
		defer func() { e.connectionStatus = "" }() // Clear status when done

		// Write the file and get updated diagnostics and git changes
		diagnostics, gitChanges, err := e.proxyManager.WriteFile(filePath, content)
		if err != nil {
			LL.Error("Failed to write file %s: %v", filePath, err)
			e.ShowError("Failed to save file: %v", err)
			return
		}

		// Update buffer diagnostics with the returned diagnostics
		target.Diagnostics = diagnostics

		// Update buffer git changes with the returned git changes
		target.GitChanges = gitChanges
		// Propagate the new pointer to all buffers representing the same real file
		e.updateGitChangesForRealPath(filePath, gitChanges)

		// If this is a "save as" operation, update the buffer's file path
		if saveAs {
			// Remove old buffer from maps if it had a file
			if target.HasFile() {
				delete(e.buffers, target.FilePath)
				// Update buffer order
				for i, path := range e.bufferOrder {
					if path == target.FilePath {
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

			// Update buffer's identifiers
			target.FilePath = filePath
			target.RealPath = filePath

			// Add buffer with new path
			e.buffers[filePath] = target
		}

		// Mark buffer as saved
		target.MarkSaved()

		// Clear the changed lines tracking since file is now saved
		target.ChangedLines = make(map[int]bool)

		// Clean up auto-save since file was successfully saved
		if e.autoSaveManager != nil {
			e.autoSaveManager.CleanupAutoSave(filePath)
		}

		// Show success message
		LL.Info("Wrote %s", filePath)
		e.ShowInfo("File saved: %s", filePath)
	}(buf)

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

	// Check if it's a directory by getting its attributes
	attrs, err := e.proxyManager.GetFileAttributes(path)
	if err == nil && attrs.IsDirectory {
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
	if e.autoSaveManager == nil {
		return
	}

	buf, exists := e.buffers[filePath]
	if !exists || buf == nil || !buf.HasFile() {
		return
	}

	content := buf.Rope.String()
	dirtyTime, _ := e.autoSaveManager.GetPendingDirtyTime(filePath)

	if err := e.autoSaveManager.saveFile(filePath, content); err != nil {
		LL.Error("Failed to auto-save file %s: %v", filePath, err)
	}

	targetPath := buf.RealPath
	if targetPath == "" {
		targetPath = buf.FilePath
	}
	if targetPath == "" {
		return
	}

	startGeneration := buf.Generation
	e.pendingSaves.Add(1)

	go func(b *buffer.Buffer, originalPath, pendingPath, snapshot string, dirtyAt time.Time, generation uint64) {
		defer e.pendingSaves.Done()

		diagnostics, gitChanges, err := e.proxyManager.WriteFile(originalPath, snapshot)
		if err != nil {
			LL.Error("Auto-save failed to write original file %s: %v", originalPath, err)
			return
		}

		if generation != b.Generation {
			// Buffer changed again while the save was running; keep modified state
			return
		}

		b.Diagnostics = diagnostics
		b.GitChanges = gitChanges
		e.updateGitChangesForRealPath(originalPath, gitChanges)
		b.MarkSaved()
		b.ChangedLines = make(map[int]bool)

		if e.CurrentBuffer() == b {
			select {
			case e.asyncEvents <- events.NewCustomEvent("render", nil):
			default:
			}
		}

		if e.autoSaveManager != nil {
			e.autoSaveManager.MarkAutoSaved(pendingPath, dirtyAt)
		}
	}(buf, targetPath, filePath, content, dirtyTime, startGeneration)
}

// markBufferDirty marks a buffer as having unsaved changes for auto-save
func (e *Editor) markBufferDirty() {
	if e.autoSaveManager != nil && e.CurrentBuffer().HasFile() {
		e.autoSaveManager.MarkDirty(e.CurrentBuffer().FilePath)
	}
}

// reloadFile reloads the current file from disk
func (e *Editor) reloadFile(args []string) error {
	// Check if we have a file open
	if e.CurrentBuffer() == nil || !e.CurrentBuffer().HasFile() {
		e.ShowError("No file to reload")
		return nil
	}

	filePath := e.CurrentBuffer().FilePath

	// Check if there are unsaved changes
	if e.CurrentBuffer().IsModified {
		// Show confirmation dialog
		width, height := e.screen.Size()
		dialogWidth := 60
		dialogHeight := 8
		dialogX := (width - dialogWidth) / 2
		dialogY := (height - dialogHeight) / 2

		message := fmt.Sprintf("File has unsaved changes. Reload and lose changes?")
		dialog := widgets.NewYesNoDialog(dialogX, dialogY, dialogWidth, dialogHeight, "Reload File", message, e.config.Theme)

		overlay := NewYesNoDialogOverlay(dialog)
		resultChan := e.renderingState.SetOverlay(overlay)

		// Handle user's choice asynchronously
		go func() {
			res := <-resultChan
			if res == string(widgets.DialogResultYes) {
				// User confirmed, proceed with reload
				e.asyncEvents <- events.NewCustomEvent("reload-file-confirmed", map[string]interface{}{
					"path": filePath,
				})
			}
			// If No or Cancel, do nothing
		}()

		return nil
	}

	// No unsaved changes, reload immediately
	e.performReload(filePath)
	return nil
}

// performReload actually reloads the file content from disk
func (e *Editor) performReload(filePath string) {
	// Start async operation for spinner
	operationID := "reload-file"
	e.startAsyncOperation(operationID, "reloading file")

	go func() {
		defer e.endAsyncOperation(operationID)

		// Read file content using proxy
		fileContent, err := e.proxyManager.GetFile(filePath)
		if err != nil {
			e.ShowError("Failed to reload file: %v", err)
			return
		}

		// Send reload event to main thread
		e.asyncEvents <- events.NewCustomEvent("file-reloaded", map[string]interface{}{
			"path":        filePath,
			"content":     fileContent.Content,
			"changes":     fileContent.Changes,
			"branch":      fileContent.Branch,
			"diagnostics": fileContent.Diagnostics,
		})
	}()
}

// handleFileReloaded handles the file-reloaded event in the main event loop
func (e *Editor) handleFileReloaded(data map[string]interface{}) {
	path, _ := data["path"].(string)
	content, _ := data["content"].(string)
	changes, _ := data["changes"].(proxy.FileChanges)
	branch, _ := data["branch"].(string)
	diagnostics, _ := data["diagnostics"].([]proxy.Diagnostic)

	// Find the buffer
	buf, exists := e.buffers[path]
	if !exists {
		LL.Error("Buffer not found for reloaded file: %s", path)
		return
	}

	// Save cursor position
	cursorLine := buf.CursorIterator.LineNumber()
	cursorX := buf.CursorX

	// Create new buffer with reloaded content
	newBuf := buffer.NewBuffer(content, path, path, buf.Indentation.Type, buf.Indentation.Width)

	// Restore cursor position if possible
	maxLine := newBuf.LineCount - 1
	if cursorLine > maxLine {
		cursorLine = maxLine
	}
	newBuf.CursorIterator = buffer.NewBufferIteratorAtLine(newBuf, cursorLine)

	// Restore cursor X position
	lineLength := len(newBuf.CursorIterator.Line())
	if cursorX > lineLength {
		cursorX = lineLength
	}
	newBuf.CursorX = cursorX

	// Copy buffer settings
	newBuf.IsReadOnly = buf.IsReadOnly
	newBuf.IsInternal = buf.IsInternal

	// Clear modified flag (new buffer already has it false)
	newBuf.IsModified = false

	// Update git changes
	newBuf.GitChanges = &changes
	newBuf.GitBranch = branch

	// Update diagnostics
	newBuf.Diagnostics = diagnostics

	// Replace the old buffer with the new one
	e.buffers[path] = newBuf
	if e.CurrentBuffer() == buf {
		e.SetCurrentBuffer(newBuf)
	}

	// Update window references
	for _, win := range e.windows {
		if win.Buffer == buf {
			win.Buffer = newBuf
		}
		for i, winBuf := range win.Buffers {
			if winBuf == buf {
				win.Buffers[i] = newBuf
			}
		}
	}

	// Re-parse syntax highlighting
	e.updateSyntaxHighlighting()

	// If a diff view is open for this file, refresh the aligned diff buffers
	if path != "" {
		// Use the freshly reloaded content
		e.refreshDiffBuffersForRealPath(path, content)
	}

	// Show success message
	e.ShowInfo("File reloaded from disk")
}
