package editor

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/events"
	"veda/internal/log"
	"veda/internal/lsp"
	"veda/internal/perf"
	"veda/internal/proxy"
	"veda/internal/syntax"
	"veda/internal/ui"
	"veda/internal/widgets"
)

var LL *log.Logger = log.NewLogger("EDITOR")

// Editor represents the state of the text editor.
type Editor struct {
	screen          Screen
	buffer          *buffer.Buffer          // Current buffer
	config          *config.Config
	mode            commands.Mode
	commandLine     *widgets.CommandLineWidget
	proxyClient     *proxy.Client
	commandRegistry *commands.CommandRegistry
	keyMapper       *commands.KeyMapper
	highlighter     *syntax.Highlighter
	perfTracker     *perf.TimeTracker

	// Buffer management
	buffers         map[string]*buffer.Buffer  // Map of file path to buffer
	bufferOrder     []string                   // Order of buffers for cycling

	// Navigation history
	navHistory      []NavigationEntry       // Stack of previous positions
	maxNavHistory   int                     // Maximum number of entries to keep

	// Context display
	contextCache    []int                   // Cached context line numbers
	contextLine     int                     // Line number for which context was cached
	showContext     bool                    // Whether to show context

	// Async notifications
	asyncEvents     chan events.Event       // Channel for async events

	// LSP communication
	lspRequestChan  chan lsp.LSPRequest     // Channel for LSP requests
	lspResponseChan chan lsp.LSPResponse    // Channel for LSP responses
	
	// Repeat last change
	lastChange      buffer.HistoryEntry     // Last text modification for repeat command
	
	// Pending operation for multi-key commands
	pendingOperator Operator                // Current operator waiting for text object
	pendingModifier TextObjectModifier      // Modifier for text object (a/i)
	pendingKeys     []string                // Accumulated keys for current operation
}

// NewWithScreen creates a new Editor with a custom screen implementation.
func NewWithScreen(screen Screen) (*Editor, error) {
	// Default style will be set from theme
	config := config.New()
	normalStyle := config.Theme.GetStyle(config.Theme.Normal)
	screen.SetStyle(normalStyle)

	// Initialize command line widget (will be positioned at bottom of screen)
	width, height := screen.Size()
	statusBarY := height - 1 // Status bar is always on the last line
	commandLine := widgets.NewCommandLineWidget(0, statusBarY, width)
	commandLine.SetStyle(config.Theme.GetStyle(config.Theme.CommandLine))

	// Start proxy client
	proxyClient, err := proxy.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to start proxy client: %v", err)
	}

	// Create initial empty buffer
	initialBuffer := buffer.NewBuffer("", "", config.DefaultIndentType, config.DefaultIndentWidth)

	// Create LSP channels
	lspRequestChan := make(chan lsp.LSPRequest, 10)
	lspResponseChan := make(chan lsp.LSPResponse, 10)

	// Start LSP worker thread
	lsp.StartLSPWorker(proxyClient, lspRequestChan, lspResponseChan)

	editor := &Editor{
		screen:          screen,
		buffer:          initialBuffer,
		config:          config,
		mode:            commands.NormalMode,
		commandLine:     commandLine,
		proxyClient:     proxyClient,
		commandRegistry: commands.NewCommandRegistry(),
		keyMapper:       commands.NewKeyMapper(),
		highlighter:     syntax.NewHighlighter(),
		perfTracker:     perf.New(true, 5*time.Second), // Report every 5 seconds
		buffers:         make(map[string]*buffer.Buffer),
		bufferOrder:     []string{},
		navHistory:      []NavigationEntry{},
		maxNavHistory:   50, // Keep last 50 navigation entries
		showContext:     true, // Show context by default
		contextCache:    nil,
		contextLine:     -1,
		asyncEvents:     make(chan events.Event, 100), // Buffered channel for async events
		lspRequestChan:  lspRequestChan,
		lspResponseChan: lspResponseChan,
	}

	// Add the initial empty buffer to the buffers map
	// Use empty string as the key for unnamed buffers
	editor.buffers[""] = initialBuffer
	editor.bufferOrder = append(editor.bufferOrder, "")

	// Register commands
	editor.registerCommands()

	// Set up default key bindings
	editor.setupDefaultBindings()

	return editor, nil
}

// New creates a new Editor with the default terminal screen.
func New() (*Editor, error) {
	screen, err := NewTerminalScreen()
	if err != nil {
		return nil, err
	}

	return NewWithScreen(screen)
}

// Run starts the editor's main event loop.
func (e *Editor) Run() {
	// Initialize the screen
	if err := e.screen.Init(); err != nil {
		// TODO: Better error handling
		panic(err)
	}

	// Force a clear to ensure clean state
	e.screen.Clear()

	// Initial draw
	e.drawEditor()
	e.drawContext()
	e.drawSignature()
	e.drawStatusBar()
	e.updateCursor()
	e.screen.Show()

	// Get the screen event channel
	screenEvents := e.screen.EventChannel()

	// Start a timer that fires every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Event loop
	for {
		var ev events.Event

		LL.Debug("Rendering buffer %+v from top line %+v, cursor line %+v, cursor %d",
			e.buffer, e.buffer.ViewIterator, e.buffer.CursorIterator, e.buffer.CursorX)

		// Use select to handle multiple event sources
		select {
		case ev = <-screenEvents:
			// Got a screen event
		case <-ticker.C:
			// Timer fired - ev remains nil, will trigger redraw
		case ev = <-e.asyncEvents:
			// Got an async event from somewhere else
		case lspResponse := <-e.lspResponseChan:
			// Got an LSP response
			e.handleLSPResponse(lspResponse)
			// Continue to redraw
		}

		LL.Trace("Received event: %+v", ev)

		// Start performance tracking for this iteration
		t0 := e.perfTracker.Start()

		// Only handle event if we got one
		if ev != nil {
			switch ev := ev.(type) {
			case *events.KeyEvent:
				if e.mode == commands.CommandMode {
					// Command mode is special - it's a line editor
					e.handleCommandMode(ev)
					t0 = e.perfTracker.ReportSince("handleCommandMode", t0)
				} else {
					// Normal and Insert modes both use key mapping
					LL.Debug("Key event: Key=%d, Rune=%c, Modifiers=%d, Mode=%s", ev.Key, ev.Rune, ev.Modifiers, e.mode)

					// Check for escape to cancel pending operations
					if e.mode == commands.NormalMode && e.hasPendingOperation() && ev.Key == events.KeyEscape {
						e.cancelPendingOperation()
						t0 = e.perfTracker.ReportSince("cancelPendingOperation", t0)
					} else if e.mode == commands.NormalMode && e.hasPendingOperation() && ev.Key == events.KeyRune {
						// Handle as part of pending operation
						if err := e.handlePendingOperation(string(ev.Rune)); err != nil {
							LL.Debug("Pending operation error: %v", err)
						}
						t0 = e.perfTracker.ReportSince("handlePendingOperation", t0)
					} else if command, found := e.keyMapper.ProcessKey(ev, e.mode); found {
						LL.Debug("Found command: %s", command)
						err := e.commandRegistry.Execute(command)
						t0 = e.perfTracker.ReportSince("command:"+command, t0)

						if err != nil {
							LL.Debug("Command execution error: %v", err)
						}
					} else {
						LL.Debug("No command found for key")
						if e.mode == commands.InsertMode && ev.Key == events.KeyRune {
							// In insert mode, insert unmapped rune keys
							e.insertChar(ev.Rune)
							t0 = e.perfTracker.ReportSince("insertChar", t0)
						}
					}
				}
			case *events.ResizeEvent:
				// Handle screen resize events
				e.handleResize()
				t0 = e.perfTracker.ReportSince("handleResize", t0)
			default:
				LL.Debug("Unsupported event type: %+v", reflect.TypeOf(ev))
			}
		}

		// Check if scrolling occurred
		e.scroll()
		t0 = e.perfTracker.ReportSince("scroll", t0)

		LL.Trace("Redrawing the screen")

		// Always redraw the entire screen
		e.drawEditor()
		t0 = e.perfTracker.ReportSince("drawEditor", t0)

		// Draw context lines
		e.drawContext()
		t0 = e.perfTracker.ReportSince("drawContext", t0)

		// Always redraw signature and status bar to show updated cursor position
		e.drawSignature()
		e.drawStatusBar()
		t0 = e.perfTracker.ReportSince("drawStatusBar", t0)

		// Always update cursor position and show
		e.updateCursor()
		t0 = e.perfTracker.ReportSince("updateCursor", t0)
		e.screen.Show()
		e.perfTracker.ReportSince("screen.Show", t0)
	}
}

// updateCursor updates the cursor position and style
func (e *Editor) updateCursor() {
	// Set cursor style based on mode
	if e.mode == commands.InsertMode {
		// Use a bar cursor in insert mode
		e.screen.SetCursorStyle(ui.CursorStyleSteadyBar)
	} else {
		// Use a block cursor in normal and command modes
		e.screen.SetCursorStyle(ui.CursorStyleSteadyBlock)
	}

	// Show cursor at appropriate position
	if e.mode == commands.CommandMode {
		// Show cursor in command line
		cursorX, cursorY := e.commandLine.GetCursorPosition()
		e.screen.ShowCursor(cursorX, cursorY)
	} else {
		// Show cursor in editor
		// Use cached line number width from buffer
		lineNumberWidth := e.buffer.LineNumberWidth
		visualCursorX := e.getVisualCursorX()
		cursorY := e.buffer.CursorIterator.LineNumber() - e.buffer.ViewIterator.LineNumber() + e.getContentStartY()
		e.screen.ShowCursor(visualCursorX+lineNumberWidth, cursorY)
	}
}

func (e *Editor) scroll() {
	visibleHeight := e.getVisibleHeight()
	bufferLines := e.buffer.LineCount
	topLine := e.buffer.ViewIterator.LineNumber()

	// Adjust scroll offset for top margin only (to account for context lines)
	topScrollOffset := e.config.ScrollOffset - len(e.contextCache)
	if topScrollOffset < 0 {
		topScrollOffset = 0
	}

	// Scroll up when cursor is too close to top
	cursorLine := e.buffer.CursorIterator.LineNumber()
	if cursorLine < topLine+topScrollOffset {
		linesToMove := topLine - (cursorLine - topScrollOffset)
		if linesToMove > 0 {
			for i := 0; i < linesToMove; i++ {
				e.buffer.ViewIterator = e.buffer.ViewIterator.Previous()
			}
		}
	}

	// Scroll down when cursor is too close to bottom, but allow cursor to reach
	// the bottom of the screen when at the end of the buffer
	// Bottom scroll offset remains unchanged
	if cursorLine >= topLine+visibleHeight-e.config.ScrollOffset {
		// If we're near the end of the buffer, allow the last line to be at the bottom
		maxTopLine := bufferLines - visibleHeight
		if maxTopLine < 0 {
			maxTopLine = 0
		}

		desiredTopLine := cursorLine - visibleHeight + e.config.ScrollOffset
		linesToMove := desiredTopLine - topLine

		// Don't scroll past the maximum allowed top line
		if topLine + linesToMove > maxTopLine {
			linesToMove = maxTopLine - topLine
		}

		if linesToMove > 0 {
			for i := 0; i < linesToMove; i++ {
				e.buffer.ViewIterator = e.buffer.ViewIterator.Next()
			}
		}
	}
}

// scrollToShowAtTop positions the view so the cursor line appears at ScrollOffset from the top
func (e *Editor) scrollToShowAtTop() {
	// Create a new iterator at the beginning
	e.buffer.ViewIterator = buffer.NewBufferIterator(e.buffer)

	// Adjust scroll offset for top margin only (to account for context lines)
	topScrollOffset := e.config.ScrollOffset - len(e.contextCache)
	if topScrollOffset < 0 {
		topScrollOffset = 0
	}

	// Move down to position cursor at ScrollOffset from the top
	targetLine := e.buffer.CursorIterator.LineNumber() - topScrollOffset
	if targetLine < 0 {
		targetLine = 0
	}

	// Move iterator to target line
	for i := 0; i < targetLine && !e.buffer.ViewIterator.IsEOF(); i++ {
		e.buffer.ViewIterator = e.buffer.ViewIterator.Next()
	}
}

// ScrollToShowAtBottom positions the view so the cursor line appears at the bottom of the screen
// This method calculates the view position so that the cursor is on the last visible line
func (e *Editor) ScrollToShowAtBottom() {
	visibleHeight := e.getVisibleHeight()
	cursorLine := e.buffer.CursorIterator.LineNumber()
	
	// Calculate the desired view start line
	// If cursor is below screen, we need to move view up by 1
	// View should start at: cursorLine - visibleHeight + 1 - 1
	desiredViewLine := cursorLine - visibleHeight + 2
	
	// Handle edge case where desired view would be negative
	if desiredViewLine < 0 {
		desiredViewLine = 0
	}
	
	// Create iterator at the desired view line
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, desiredViewLine)

	LL.Debug("After going to the end, the view iterator is at %+v, cursor iterator at %+v, visible height is %d", 
		e.buffer.ViewIterator, e.buffer.CursorIterator, visibleHeight)
}

func (e *Editor) handleCommandMode(ev *events.KeyEvent) {
	switch ev.Key {
	case events.KeyEscape:
		// Exit command mode
		e.exitCommandMode()
	case events.KeyEnter:
		// Execute command and exit command mode
		e.executeCommand()
		e.exitCommandMode()
	default:
		// Pass other keys to the command line widget
		e.commandLine.HandleKey(ev)

		// Check if the user deleted all text including the colon
		// If so, exit command mode automatically
		if e.commandLine.GetText() == "" {
			e.exitCommandMode()
		}
	}
}

// enterCommandMode switches to command mode and initializes the command line
func (e *Editor) enterCommandMode() {
	e.mode = commands.CommandMode
	e.commandLine.SetText(":")
	// Position cursor after the colon
	e.commandLine.SetCursor(1)
	
	// Set up command completion function
	e.commandLine.SetCompletionFunc(e.getCommandCompletions)
}

// exitCommandMode switches back to normal mode
func (e *Editor) exitCommandMode() {
	e.mode = commands.NormalMode
	e.commandLine.SetText("")
}

// executeCommand processes the command entered in the command line
func (e *Editor) executeCommand() {
	fullCommand := e.commandLine.GetText()
	
	// Add to history before executing
	e.commandLine.AddToHistory(fullCommand)
	
	// Remove the leading ":"
	command := fullCommand
	if len(command) > 1 {
		command = command[1:]
	}

	// Check if this is a variable assignment (VAR=VALUE)
	if strings.Contains(command, "=") && !strings.HasPrefix(command, " ") {
		e.handleVariableAssignment(command)
		return
	}

	// First try to execute from the command registry
	if e.commandRegistry.Has(command) {
		err := e.commandRegistry.Execute(command)
		if err != nil {
			LL.Debug("Command execution failed: %v", err)
		}
		return
	}

	// Handle basic commands that aren't in the registry
	switch command {
	case "q", "quit":
		e.Close()
		os.Exit(0)
	}
}

// handleVariableAssignment processes variable assignments in VAR=VALUE format
func (e *Editor) handleVariableAssignment(assignment string) {
	parts := strings.SplitN(assignment, "=", 2)
	if len(parts) != 2 {
		return
	}
	
	varName := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	
	switch varName {
	case "expandtab":
		// Boolean value
		if value == "true" || value == "1" || value == "yes" {
			e.buffer.SetIndentationType(config.IndentSpaces)
		} else if value == "false" || value == "0" || value == "no" {
			e.buffer.SetIndentationType(config.IndentTabs)
		}
	case "tabstop", "ts":
		// Set tab width
		if width, err := strconv.Atoi(value); err == nil && width > 0 && width <= 8 {
			e.buffer.SetIndentationWidth(width)
		}
	case "shiftwidth", "sw":
		// Set indent width (same as tabstop for now)
		if width, err := strconv.Atoi(value); err == nil && width > 0 && width <= 8 {
			e.buffer.SetIndentationWidth(width)
		}
	case "number":
		// Show line numbers (future feature)
		// TODO: implement line numbers
	case "relativenumber":
		// Show relative line numbers (future feature)
		// TODO: implement relative line numbers
	case "autoindent":
		// Auto indent (already implemented by default)
		// TODO: make this configurable
	case "smartindent":
		// Smart indent (already implemented by default)
		// TODO: make this configurable
	case "log":
		// Set log level
		if level, err := log.ParseLogLevel(value); err == nil {
			log.SetAllLogLevel(level)
			LL.Info("Log level set to %s", value)
		} else {
			LL.Warning("Invalid log level: %s (use: info, debug, trace)", value)
		}
	}
}


// handleResize processes screen resize events
func (e *Editor) handleResize() {
	width, _ := e.screen.Size()

	// Update the command line widget's dimensions and position
	e.commandLine.SetWidth(width)
	e.commandLine.SetPosition(0, e.getStatusBarY())
}

// Close finalizes the screen and proxy client.
func (e *Editor) Close() {
	// Stop the performance tracker
	if e.perfTracker != nil {
		e.perfTracker.Stop()
	}

	e.screen.Fini()
	if e.proxyClient != nil {
		e.proxyClient.Close()
	}
}

// visualWidth calculates the visual width of a string accounting for tabs
func (e *Editor) visualWidth(text string) int {
	width := 0
	for _, r := range text {
		if r == '\t' {
			width += 4
		} else {
			width++
		}
	}
	return width
}

// getVisualCursorX calculates the visual cursor position accounting for tabs
func (e *Editor) getVisualCursorX() int {
	if e.buffer.CursorIterator.IsEOF() {
		return 0
	}
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX >= len(line) {
		return e.visualWidth(line)
	}
	return e.visualWidth(line[:e.buffer.CursorX])
}


// getLineTokens returns syntax tokens for a line using iterator position
func (e *Editor) getLineTokens(iter *buffer.BufferIterator) []syntax.Token {
	// Pass the iterator positions directly to the highlighter
	// This allows future caching based on byte positions rather than line numbers
	return e.highlighter.GetTokensForLine(iter.Position(), iter.LineEnd())
}

// updateCursorState updates any state that depends on cursor position
func (e *Editor) updateCursorState() {
	// Check if we need to update signature help
	e.checkSignatureUpdate()
}

func (e *Editor) quit() error {
	e.Close()
	os.Exit(0)
	return nil
}

// parseSignatureHelpResponse parses the raw signature help response and stores it in the buffer
func (e *Editor) parseSignatureHelpResponse(buf *buffer.Buffer, rawResponse interface{}) {
	if rawResponse == nil {
		LL.Debug("No signature help available")
		buf.CurrentSignature = nil
		return
	}

	LL.Debug("Parsing signature help response: %+v", rawResponse)

	// Convert the result to SignatureHelp structure
	if sigHelp, ok := rawResponse.(map[string]interface{}); ok {
		if signatures, ok := sigHelp["signatures"].([]interface{}); ok && len(signatures) > 0 {
			// Take the first signature
			if sig, ok := signatures[0].(map[string]interface{}); ok {
				help := &buffer.SignatureHelp{}

				// Extract label
				if label, ok := sig["label"].(string); ok {
					help.Label = label
				}

				// Extract documentation
				if doc, ok := sig["documentation"].(string); ok {
					help.Documentation = doc
				}

				// Extract active parameter
				if activeParam, ok := sigHelp["activeParameter"].(float64); ok {
					help.ActiveParameter = int(activeParam)
				}

				// Extract parameters
				if params, ok := sig["parameters"].([]interface{}); ok {
					for _, p := range params {
						if param, ok := p.(map[string]interface{}); ok {
							paramInfo := buffer.ParameterInfo{}
							
							// Handle label - can be string or [start, end] array
							if label, ok := param["label"].(string); ok {
								paramInfo.Label = label
							} else if labelArray, ok := param["label"].([]interface{}); ok && len(labelArray) == 2 {
								// Label is a range [start, end]
								if start, ok := labelArray[0].(float64); ok {
									paramInfo.LabelStart = int(start)
								}
								if end, ok := labelArray[1].(float64); ok {
									paramInfo.LabelEnd = int(end)
								}
							}
							
							if doc, ok := param["documentation"].(string); ok {
								paramInfo.Documentation = doc
							}
							help.Parameters = append(help.Parameters, paramInfo)
						}
					}
				}

				// Store the signature
				buf.CurrentSignature = help
				LL.Info("Stored async signature: %s", help.Label)
			}
		} else {
			// No signatures available - we're outside a function call
			LL.Debug("No signatures in response - clearing signature help")
			buf.CurrentSignature = nil
		}
	} else {
		LL.Debug("Unexpected signature help response format: %T", rawResponse)
		buf.CurrentSignature = nil
	}
}

// handleLSPResponse processes responses from the LSP worker thread
func (e *Editor) handleLSPResponse(response lsp.LSPResponse) {
	LL.Trace("Received LSP response: type=%d, filePath=%s", response.Type, response.FilePath)

	switch response.Type {
	case lsp.RequestFoldingRanges:
		// Find the buffer that requested this
		if buf, exists := e.buffers[response.FilePath]; exists {
			if response.Error != nil {
				LL.Debug("LSP folding ranges error for %s: %v", response.FilePath, response.Error)
				buf.HasLSP = false
				buf.FoldingRanges = nil
			} else {
				LL.Trace("Updated folding ranges for %s: got %d ranges", response.FilePath, len(response.FoldingRanges))
				buf.HasLSP = true
				buf.FoldingRanges = response.FoldingRanges
			}

			// If this is the current buffer, invalidate context cache to trigger redraw
			if buf == e.buffer {
				e.contextLine = -1 // Force context update
			}
		}

	case lsp.RequestSignatureHelp:
		// Find the buffer that requested this
		if buf, exists := e.buffers[response.FilePath]; exists {
			// Only apply response if generation matches current state
			if response.Generation == buf.SignatureGeneration {
				if response.Error != nil {
					LL.Debug("LSP signature help error for %s: %v", response.FilePath, response.Error)
					buf.CurrentSignature = nil
				} else {
					// Parse and store the signature help
					e.parseSignatureHelpResponse(buf, response.SignatureHelp)
				}
			} else {
				// Stale response - ignore it
				LL.Debug("Ignoring stale signature help response: gen=%d, current=%d",
					response.Generation, buf.SignatureGeneration)
			}
		}

	case lsp.RequestProjectSymbols:
		// Project symbols responses are handled by specific widgets via ResponseTo channel
		// This case is here to prevent "unknown response type" warnings
		LL.Trace("Received project symbols response for query '%s', forwarded to widget", response.Query)

	case lsp.RequestDocumentSymbols:
		// Document symbols responses are handled by specific widgets via ResponseTo channel
		// This case is here to prevent "unknown response type" warnings
		LL.Trace("Received document symbols response for file '%s', forwarded to widget", response.FilePath)
	}
}

// requestFoldingRanges sends a request to the LSP worker for folding ranges
func (e *Editor) requestFoldingRanges(filePath string) {
	if filePath == "" {
		return
	}

	// Find the buffer for this file path
	buf, exists := e.buffers[filePath]
	if !exists {
		return
	}

	request := lsp.LSPRequest{
		Type:     lsp.RequestFoldingRanges,
		FilePath: filePath,
		Buffer:   buf,
	}

	// Send request to LSP worker (non-blocking)
	select {
	case e.lspRequestChan <- request:
		LL.Trace("Sent folding ranges request for %s", filePath)
	default:
		LL.Debug("LSP request channel full, dropping folding ranges request for %s", filePath)
	}
}


// Insert mode commands
func (e *Editor) enterInsertMode() error {
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession()
	}
	return nil
}

func (e *Editor) exitInsertMode() error {
	e.mode = commands.NormalMode
	
	// Adjust cursor position for normal mode
	// In vim, when exiting insert mode, if cursor is not at position 0, move back one
	if e.buffer != nil && e.buffer.CursorX > 0 {
		e.buffer.CursorX--
	}
	
	// Finalize insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		if compositeEntry := e.buffer.History.EndInsertSession(); compositeEntry != nil {
			e.lastChange = compositeEntry
		}
	}
	return nil
}

// Undo/Redo commands
func (e *Editor) undo() error {
	if e.buffer == nil || e.buffer.History == nil {
		return nil
	}
	
	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	
	// Perform undo
	if err := e.buffer.History.Undo(e.buffer); err != nil {
		return err
	}
	
	// Ensure iterators are valid after undo
	// If cursor line is beyond buffer, move to last line
	maxLine := e.buffer.LineCount - 1
	if lineNum > maxLine {
		lineNum = maxLine
	}
	if viewLineNum > maxLine {
		viewLineNum = maxLine
	}
	
	// Recreate iterators
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	
	// Ensure cursor X is within bounds
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(line) {
		e.buffer.CursorX = len(line)
		if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
			e.buffer.CursorX--
		}
	}
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	return nil
}

func (e *Editor) redo() error {
	if e.buffer == nil || e.buffer.History == nil {
		return nil
	}
	
	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	
	// Perform redo
	if err := e.buffer.History.Redo(e.buffer); err != nil {
		return err
	}
	
	// Ensure iterators are valid after redo
	// If cursor line is beyond buffer, move to last line
	maxLine := e.buffer.LineCount - 1
	if lineNum > maxLine {
		lineNum = maxLine
	}
	if viewLineNum > maxLine {
		viewLineNum = maxLine
	}
	
	// Recreate iterators
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	
	// Ensure cursor X is within bounds
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(line) {
		e.buffer.CursorX = len(line)
		if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
			e.buffer.CursorX--
		}
	}
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	return nil
}

// repeatLastChange repeats the last text modification at the current cursor position
func (e *Editor) repeatLastChange() error {
	if e.lastChange == nil {
		return nil // Nothing to repeat
	}
	
	if e.buffer == nil || e.buffer.History == nil {
		return nil
	}
	
	// Get current cursor position
	lineNum := e.buffer.CursorIterator.LineNumber()
	cursorX := e.buffer.CursorX
	
	// Create a new entry based on the type of the last change
	var newEntry buffer.HistoryEntry
	
	switch original := e.lastChange.(type) {
	case *buffer.InsertTextEntry:
		// For insert, we need to insert after the current cursor position in normal mode
		// This mimics pressing 'i' which inserts before the cursor
		newEntry = buffer.NewInsertTextEntry(lineNum, cursorX, original.Text)
		
	case *buffer.DeleteTextEntry:
		// Delete happens at the current cursor position
		newEntry = buffer.NewDeleteTextEntry(lineNum, cursorX, original.Text)
		
	case *buffer.CompositeEntry:
		// For composite entries (like insert mode sessions), replay all operations
		// starting from the current cursor position
		composite := buffer.NewCompositeEntry("Repeat: " + original.Description())
		
		// Calculate the base position offset
		var baseLineOffset, baseCharOffset int
		var firstLine int
		if len(original.Entries) > 0 {
			// Get the first entry's position as reference
			switch first := original.Entries[0].(type) {
			case *buffer.InsertTextEntry:
				firstLine = first.Line
				baseLineOffset = lineNum - first.Line
				baseCharOffset = cursorX - first.Character
			case *buffer.DeleteTextEntry:
				firstLine = first.Line
				baseLineOffset = lineNum - first.Line
				baseCharOffset = cursorX - first.Character
			}
		}
		
		// Translate each sub-entry
		for _, subEntry := range original.Entries {
			switch se := subEntry.(type) {
			case *buffer.InsertTextEntry:
				newLine := se.Line + baseLineOffset
				newChar := se.Character
				// Only apply character offset to the first entry on the same line
				if se.Line == firstLine {
					newChar = se.Character + baseCharOffset
				}
				composite.Add(buffer.NewInsertTextEntry(newLine, newChar, se.Text))
				
			case *buffer.DeleteTextEntry:
				newLine := se.Line + baseLineOffset
				newChar := se.Character
				// Only apply character offset to the first entry on the same line
				if se.Line == firstLine {
					newChar = se.Character + baseCharOffset
				}
				composite.Add(buffer.NewDeleteTextEntry(newLine, newChar, se.Text))
			}
		}
		newEntry = composite
		
	default:
		// Unknown entry type, cannot repeat
		return nil
	}
	
	// Save the original lastChange before executing
	originalLastChange := e.lastChange
	
	// Remember the view position before we make changes
	viewLine := e.buffer.ViewIterator.LineNumber()
	
	// Execute the new entry (this will also update lastChange)
	if _, err := e.buffer.History.Execute(e.buffer, newEntry); err != nil {
		return err
	}
	
	// Restore the original lastChange to maintain it for future repeats
	e.lastChange = originalLastChange
	
	// Recreate iterators to ensure they're valid after buffer changes
	// This is especially important for ViewIterator which can become invalid
	
	// Recreate cursor iterator
	cursorLine := e.buffer.CursorIterator.LineNumber()
	if cursorLine >= e.buffer.LineCount {
		cursorLine = e.buffer.LineCount - 1
		if cursorLine < 0 {
			cursorLine = 0
		}
	}
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, cursorLine)
	
	// Recreate view iterator
	if viewLine >= e.buffer.LineCount {
		viewLine = e.buffer.LineCount - 1
		if viewLine < 0 {
			viewLine = 0
		}
	}
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLine)
	
	// Ensure cursor X is within bounds
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(line) {
		e.buffer.CursorX = len(line)
		if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
			e.buffer.CursorX--
		}
	}
	
	// Update cursor state
	e.updateCursorState()
	
	return nil
}

// openLineBelow opens a new line below the current line and enters insert mode
func (e *Editor) openLineBelow() error {
	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()

	// Get the current line and move to end
	currentLine := ""
	if !e.buffer.CursorIterator.IsEOF() {
		currentLine = e.buffer.CursorIterator.Line()
		e.buffer.CursorX = len(currentLine)
	}

	// Calculate auto-indent for the new line
	autoIndent := e.buffer.CalculateAutoIndent(currentLine, e.buffer.CursorX)

	// Create history entry for the newline and indentation
	textToInsert := "\n" + autoIndent.IndentStr
	entry := buffer.NewInsertTextEntry(lineNum, e.buffer.CursorX, textToInsert)
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		// Fallback if history is not initialized
		pos := e.getAbsolutePosition()
		e.buffer.Insert(pos, textToInsert)
	}

	// Recreate iterators - cursor moves to next line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum + 1)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Move cursor to end of indentation
	e.buffer.CursorX = len(autoIndent.IndentStr)

	// Enter insert mode
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession()
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// openLineAbove opens a new line above the current line and enters insert mode
func (e *Editor) openLineAbove() error {
	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()

	// Get the previous line for auto-indent calculation (if we're not on the first line)
	previousLine := ""
	if lineNum > 0 {
		// Temporarily move to previous line to get its content
		prevIter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum - 1)
		if !prevIter.IsEOF() {
			previousLine = prevIter.Line()
		}
	}

	// Calculate auto-indent based on the previous line
	// Use the length of the previous line as cursor position for calculation
	autoIndent := e.buffer.CalculateAutoIndent(previousLine, len(previousLine))

	// Move to beginning of current line
	e.buffer.CursorX = 0

	// Create history entry for the newline
	entry := buffer.NewInsertTextEntry(lineNum, 0, "\n")
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		// Fallback if history is not initialized
		pos := e.getAbsolutePosition()
		e.buffer.Insert(pos, "\n")
	}

	// The cursor stays at the same line number (which is now the new empty line)
	// because we inserted before the current position
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Insert indentation at the beginning of the new line
	if autoIndent.IndentStr != "" {
		// Create history entry for the indentation
		indentEntry := buffer.NewInsertTextEntry(lineNum, 0, autoIndent.IndentStr)
		
		// Execute through history
		if e.buffer.History != nil {
			if executed, err := e.buffer.History.Execute(e.buffer, indentEntry); err != nil {
				return err
			} else if executed != nil {
				e.lastChange = executed
			}
		} else {
			// Fallback if history is not initialized
			pos := e.getAbsolutePosition()
			e.buffer.Insert(pos, autoIndent.IndentStr)
		}

		// Recreate iterators after insertion
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	}

	// Move cursor to end of indentation
	e.buffer.CursorX = len(autoIndent.IndentStr)

	// Enter insert mode
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession()
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

func (e *Editor) insertChar(ch rune) error {
	// Check for auto-pair characters
	pairedChar, isPaired := e.getPairedChar(ch)

	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	cursorX := e.buffer.CursorX

	// Create history entry
	textToInsert := string(ch)
	if isPaired {
		textToInsert = string(ch) + string(pairedChar)
	}
	
	entry := buffer.NewInsertTextEntry(lineNum, cursorX, textToInsert)
	
	// Execute through history (this handles both direct execution and insert mode tracking)
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.buffer); err != nil {
			return err
		}
	}

	// Recreate iterators at the same line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Move cursor appropriately
	if isPaired {
		// Move cursor between the pair (only forward by 1)
		e.buffer.CursorX = cursorX + 1
	} else {
		// Move cursor forward
		e.buffer.CursorX = cursorX + 1
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// getPairedChar returns the closing character for auto-pairing and whether the character should be paired
func (e *Editor) getPairedChar(ch rune) (rune, bool) {
	switch ch {
	case '(':
		return ')', true
	case '[':
		return ']', true
	case '{':
		return '}', true
	case '"':
		return '"', true
	case '\'':
		return '\'', true
	case '`':
		return '`', true
	default:
		return 0, false
	}
}

func (e *Editor) insertNewline() error {
	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	cursorX := e.buffer.CursorX

	// Get the current line for auto-indent calculation
	currentLine := ""
	if !e.buffer.CursorIterator.IsEOF() {
		currentLine = e.buffer.CursorIterator.Line()
	}

	// Calculate auto-indent for the new line
	autoIndent := e.buffer.CalculateAutoIndent(currentLine, e.buffer.CursorX)

	// Create history entry for newline + indentation
	textToInsert := "\n" + autoIndent.IndentStr
	entry := buffer.NewInsertTextEntry(lineNum, cursorX, textToInsert)
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.buffer); err != nil {
			return err
		}
	}

	// Recreate iterators after buffer modification
	// Cursor moves to next line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum + 1)
	// View iterator stays at its current line
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Move cursor to end of indentation
	e.buffer.CursorX = len(autoIndent.IndentStr)

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

func (e *Editor) backspace() error {
	// If at beginning of buffer, nothing to delete
	if e.buffer.CursorX == 0 && e.buffer.CursorIterator.LineNumber() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	cursorX := e.buffer.CursorX
	pos := e.getAbsolutePosition()

	if cursorX == 0 {
		// At beginning of line, join with previous line
		if pos > 0 && lineNum > 0 {
			// Store the length of the previous line before joining
			prevIter := e.buffer.CursorIterator.Previous()
			prevLineLen := 0
			if !prevIter.Equals(e.buffer.CursorIterator) {
				prevLineLen = len(prevIter.Line())
			}

			// We're deleting a newline character
			deletedText := "\n"
			entry := buffer.NewDeleteTextEntry(lineNum-1, prevLineLen, deletedText)
			
			// Execute through history
			if e.buffer.History != nil {
				if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
					return err
				} else if executed != nil {
					e.lastChange = executed
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.buffer); err != nil {
					return err
				}
			}

			// Recreate iterators - cursor moves to previous line
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum-1)
			e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

			// Position cursor at end of previous line
			e.buffer.CursorX = prevLineLen

			// Update syntax highlighting
			e.updateSyntaxHighlighting()
		}
	} else {
		// Delete character before cursor
		if pos > 0 {
			// Get the character being deleted
			deletedChar := e.buffer.Rope.ByteAt(pos - 1)
			deletedText := string(deletedChar)
			
			entry := buffer.NewDeleteTextEntry(lineNum, cursorX-1, deletedText)
			
			// Execute through history
			if e.buffer.History != nil {
				if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
					return err
				} else if executed != nil {
					e.lastChange = executed
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.buffer); err != nil {
					return err
				}
			}

			// Recreate iterators at the same line
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
			e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

			// Move cursor back one position
			e.buffer.CursorX = cursorX - 1

			// Update syntax highlighting
			e.updateSyntaxHighlighting()
		}
	}

	return nil
}

// deleteCurrentCharacter deletes the character at the current cursor position
func (e *Editor) deleteCurrentCharacter() error {
	// If at EOF or empty buffer, nothing to delete
	if e.buffer.CursorIterator.IsEOF() || e.buffer.Rope.Length() == 0 {
		return nil
	}

	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()

	// Get current line to check bounds
	line := e.buffer.CursorIterator.Line()

	// If cursor is past the end of the line, check if we can delete newline
	if e.buffer.CursorX >= len(line) {
		// Check if there's a newline character at the end of this line
		pos := e.getAbsolutePosition()
		if pos < e.buffer.Rope.Length() && e.buffer.Rope.ByteAt(pos) == '\n' {
			// We can delete the newline
			deletedText := "\n"
			entry := buffer.NewDeleteTextEntry(lineNum, e.buffer.CursorX, deletedText)
			
			// Execute through history
			if e.buffer.History != nil {
				if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
					return err
				} else if executed != nil {
					e.lastChange = executed
				}
			} else {
				// Fallback if history is not initialized
				if err := entry.Execute(e.buffer); err != nil {
					return err
				}
			}
			
			// Recreate iterators at the same position
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
			e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
			
			// Update syntax highlighting
			e.updateSyntaxHighlighting()
			return nil
		}
		return nil
	}

	// Get cursor X position
	cursorX := e.buffer.CursorX

	// Get the character being deleted
	pos := e.getAbsolutePosition()
	deletedChar := e.buffer.Rope.ByteAt(pos)
	deletedText := string(deletedChar)
	
	// Create history entry
	entry := buffer.NewDeleteTextEntry(lineNum, cursorX, deletedText)
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, entry); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		// Fallback if history is not initialized
		if err := entry.Execute(e.buffer); err != nil {
			return err
		}
	}

	// Recreate iterators at the same line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Cursor position stays the same, but we need to ensure it's not past the end of the line
	newLine := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(newLine) {
		if len(newLine) > 0 {
			e.buffer.CursorX = len(newLine) - 1 // Stay on last character in normal mode
		} else {
			e.buffer.CursorX = 0
		}
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// Layout calculation methods
// These methods centralize the calculation of screen areas, making it easy to change
// the layout (e.g., status bar height) without modifying code throughout the editor.

// getVisibleHeight returns the height of the visible editing area (excluding status bar and context)
func (e *Editor) getVisibleHeight() int {
	_, height := e.screen.Size()
	reserved := 1 // Status bar
	reserved += len(e.contextCache) // Context lines
	// Reserve space for signature if shown
	if e.buffer.CurrentSignature != nil {
		reserved += e.getSignatureLines()
	}
	return height - reserved
}

// getContentStartY returns the Y position where content starts
func (e *Editor) getContentStartY() int {
	return len(e.contextCache) // Start after context lines
}

// shouldShowContext determines if context should be shown
func (e *Editor) shouldShowContext() bool {
	if !e.showContext {
		return false
	}

	// Always show context if we have it
	// The scroll offset will handle limiting the number of lines
	return true
}

// updateContext updates the cached context for the current cursor position
func (e *Editor) updateContext() {
	// Get current cursor line
	cursorIter := e.buffer.CursorIterator
	currentLine := cursorIter.LineNumber()

	// Only update if line changed
	if currentLine == e.contextLine && len(e.contextCache) > 0 {
		return
	}

	// Get file path
	filePath := e.buffer.FilePath
	if filePath == "" {
		e.contextCache = nil
		return
	}

	// Use cached folding ranges from buffer instead of calling LSP directly
	ranges := e.buffer.FoldingRanges
	if len(ranges) == 0 {
		e.contextCache = nil
		return
	}

	// Calculate visible range
	viewStartLine := e.buffer.ViewIterator.LineNumber()
	viewEndLine := viewStartLine + e.getVisibleHeight() - 1

	// Get context line numbers, limited by scroll offset
	e.contextCache = getContextLineNumbers(ranges, currentLine, viewStartLine, viewEndLine, e.config.ScrollOffset)
	e.contextLine = currentLine
}

// getStatusBarY returns the Y position of the status bar
func (e *Editor) getStatusBarY() int {
	_, height := e.screen.Size()
	return height - 1
}


// getVisibleLines returns the number of lines that can be displayed in the editor
func (e *Editor) getVisibleLines() int {
	return e.getVisibleHeight()
}

// getAbsolutePosition returns the absolute position in the buffer for the current cursor
func (e *Editor) getAbsolutePosition() int {
	// Get the position at the start of the current line
	pos := e.buffer.CursorIterator.Position()

	// Add current X position
	// Need to get current line length to validate cursorX
	if !e.buffer.CursorIterator.IsEOF() {
		line := e.buffer.CursorIterator.Line()
		if e.buffer.CursorX > len(line) {
			pos += len(line)
		} else {
			pos += e.buffer.CursorX
		}
	}

	return pos
}

// updateSyntaxHighlighting re-parses the buffer for syntax highlighting
func (e *Editor) updateSyntaxHighlighting() {
	if e.highlighter != nil && e.buffer.HasFile() {
		content := e.buffer.Rope.String()
		// Force a full reparse by not reusing the old tree
		e.highlighter.ParseFull([]byte(content))
	}
}


func (e *Editor) enterCommandModeCmd() error {
	e.enterCommandMode()
	return nil
}


// goToDefinition uses LSP to navigate to the definition of the symbol under cursor
func (e *Editor) showLSPBufferGo() error {
	// Save current buffer state first
	e.saveCurrentBufferState()

	// Get LSP messages for Go
	messages, err := e.proxyClient.GetLSPMessages("go")
	if err != nil {
		LL.Debug("Failed to get LSP messages: %v", err)
		return err
	}

	// Format messages as a string buffer
	var content strings.Builder
	content.WriteString("=== LSP Messages for Go ===\n\n")

	for _, msg := range messages {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			// Format the message nicely
			if method, ok := msgMap["method"].(string); ok {
				content.WriteString(fmt.Sprintf("Method: %s\n", method))
			}
			if id, ok := msgMap["id"]; ok {
				content.WriteString(fmt.Sprintf("ID: %v\n", id))
			}
			if params, ok := msgMap["params"]; ok {
				content.WriteString(fmt.Sprintf("Params: %v\n", params))
			}
			if result, ok := msgMap["result"]; ok {
				content.WriteString(fmt.Sprintf("Result: %v\n", result))
			}
			if error, ok := msgMap["error"]; ok {
				content.WriteString(fmt.Sprintf("Error: %v\n", error))
			}
			content.WriteString("\n---\n\n")
		}
	}

	// Check if LSP buffer already exists
	lspBufferName := "*LSP Messages - Go*"
	if bufferInfo, exists := e.buffers[lspBufferName]; exists {
		// Update existing buffer's content
		bufferInfo.Rope = buffer.NewRopeFromString(content.String())
		bufferInfo.CursorX = 0
		bufferInfo.ViewIterator = buffer.NewBufferIterator(bufferInfo)
		bufferInfo.CursorIterator = buffer.NewBufferIterator(bufferInfo)
		e.switchToBuffer(bufferInfo)
	} else {
		// Create new buffer
		newBuffer := buffer.NewBuffer(content.String(), lspBufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		e.buffers[lspBufferName] = newBuffer
		e.bufferOrder = append(e.bufferOrder, lspBufferName)
		e.switchToBuffer(newBuffer)
	}

	return nil
}

func (e *Editor) goToDefinition() error {
	// Only works if we have a file open
	if !e.buffer.HasFile() {
		// TODO: Show error to user
		return fmt.Errorf("no file open")
	}

	// Get cursor position (LSP uses 0-based line numbers)
	line := e.buffer.CursorIterator.LineNumber()
	character := e.buffer.CursorX

	LL.Trace("goToDefinition: file=%s, line=%d, character=%d", e.buffer.FilePath, line, character)

	// Get definition location from LSP
	locations, err := e.proxyClient.GoToDefinition(e.buffer.FilePath, line, character)
	if err != nil {
		// TODO: Show error to user
		LL.Debug("goToDefinition failed: %v", err)
		return err
	}

	LL.Trace("goToDefinition: got %d locations", len(locations))

	// If no locations found, nothing to do
	if len(locations) == 0 {
		// TODO: Show message to user
		return nil
	}

	// Use the first location
	location := locations[0]

	// Parse the file URI
	if !strings.HasPrefix(location.URI, "file://") {
		return fmt.Errorf("unsupported URI scheme: %s", location.URI)
	}

	filePath := strings.TrimPrefix(location.URI, "file://")

	// Push current position to navigation history before jumping
	e.pushNavigationHistory()

	// If it's a different file, open it
	if filePath != e.buffer.FilePath {
		e.openFile(filePath)
	}

	// Jump to the location
	targetLine := location.Range.Start.Line
	e.buffer.CursorX = location.Range.Start.Character

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
		if e.buffer.CursorX > 0 {
			e.buffer.CursorX-- // Don't go past last character in normal mode
		}
	}

	// Position the definition at ScrollOffset from the top of the screen
	e.scrollToShowAtTop()

	return nil
}

// getCommandCompletions returns possible completions for the given command prefix
func (e *Editor) getCommandCompletions(prefix string) []string {
	// Check if this is a variable assignment
	if strings.Contains(prefix, "=") {
		// Complete variable values
		parts := strings.SplitN(prefix, "=", 2)
		if len(parts) == 2 {
			varName := parts[0]
			
			switch varName {
			case "expandtab":
				values := []string{"true", "false", "yes", "no", "1", "0"}
				var completions []string
				for _, v := range values {
					completion := varName + "=" + v
					if strings.HasPrefix(completion, prefix) {
						completions = append(completions, completion)
					}
				}
				return completions
			case "tabstop", "ts", "shiftwidth", "sw":
				// Suggest common values
				values := []string{"2", "4", "8"}
				var completions []string
				for _, v := range values {
					completion := varName + "=" + v
					if strings.HasPrefix(completion, prefix) {
						completions = append(completions, completion)
					}
				}
				return completions
			case "log":
				// Suggest log levels
				values := []string{"info", "debug", "trace"}
				var completions []string
				for _, v := range values {
					completion := varName + "=" + v
					if strings.HasPrefix(completion, prefix) {
						completions = append(completions, completion)
					}
				}
				return completions
			}
		}
	}
	
	// Variable names for assignment
	variables := []string{
		"expandtab", "tabstop", "ts", "shiftwidth", "sw",
		"number", "relativenumber", "autoindent", "smartindent",
		"log",
	}
	
	// Add registered commands
	var commands []string
	for cmd := range e.commandRegistry.GetAll() {
		commands = append(commands, cmd)
	}
	
	// Add built-in commands
	builtinCommands := []string{"q", "quit"}
	commands = append(commands, builtinCommands...)
	
	// Combine all possible completions
	allCompletions := append(commands, variables...)
	
	// Filter based on prefix
	var completions []string
	for _, item := range allCompletions {
		if strings.HasPrefix(item, prefix) {
			completions = append(completions, item)
		}
	}
	
	return completions
}
