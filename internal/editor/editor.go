package editor

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"veda/internal/buffer"
	"veda/internal/clipboard"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/events"
	"veda/internal/log"
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
	searchLine      *widgets.CommandLineWidget  // For search mode (/)
	proxyManager    *proxy.ProxyManager
	commandRegistry *commands.CommandRegistry
	keyMapper       *commands.KeyMapper
	highlighter     *syntax.Highlighter
	perfTracker     *perf.TimeTracker

	// Buffer management
	buffers         map[string]*buffer.Buffer  // Map of file path to buffer
	bufferOrder     []string                   // Order of buffers for cycling
	previousBuffer  *buffer.Buffer             // Previous buffer for quick switching

	// Navigation history
	navHistory      []NavigationEntry       // Stack of previous positions
	maxNavHistory   int                     // Maximum number of entries to keep

	// Context display
	contextCache    []int                   // Cached context line numbers
	contextLine     int                     // Line number for which context was cached
	showContext     bool                    // Whether to show context

	// Async notifications
	asyncEvents     chan events.Event       // Channel for async events

	// LSP communication - removed channels, using goroutines on demand

	// Repeat last change
	// We need two separate fields to handle vim's "." (repeat) command correctly:
	//
	// lastChange (buffer.HistoryEntry):
	//   - Stores the actual text modification that was made (e.g., inserted "hello", deleted "world")
	//   - Used for simple operations where repeat means "do the exact same text change"
	//   - Examples: inserting text, deleting a character, joining lines
	//   - When repeated, replays the exact same text modification at the cursor position
	//
	// lastCommand (*RepeatableCommand):
	//   - Stores the operator+text object combination (e.g., delete + word)
	//   - Used for complex operations that need to be re-evaluated at the new cursor position
	//   - Examples: dw (delete word), di{ (delete inside braces), daw (delete around word)
	//   - When repeated, finds the text object at the NEW cursor position and applies the operator
	//
	// Why we can't use just one field:
	// 1. Different repeat semantics:
	//    - "dw" on "apple banana" deletes "apple "
	//    - Move cursor to "banana" and press "."
	//    - Should delete "banana ", NOT try to delete "apple " again
	//
	// 2. Context requirements:
	//    - HistoryEntry only has access to the buffer
	//    - Operator commands need the full Editor context (for scrolling, syntax highlighting, etc.)
	//
	// 3. Lifecycle differences:
	//    - lastChange is set by ALL text modifications (for undo/redo)
	//    - lastCommand is ONLY set by operator+text object commands
	//    - Non-operator commands clear lastCommand but set lastChange
	//
	// The repeat command checks lastCommand first, then falls back to lastChange,
	// providing the correct vim-like behavior for all command types.
	lastChange      buffer.HistoryEntry     // Last text modification for repeat command
	lastCommand     *RepeatableCommand      // Last operator+text object command for repeat

	// Pending operation for multi-key commands
	pendingOperator Operator                // Current operator waiting for text object
	pendingModifier TextObjectModifier      // Modifier for text object (a/i)
	pendingKeys     []string                // Accumulated keys for current operation

	// Search state
	searchPattern   string                  // Current search pattern
	searchDirection bool                    // true = forward, false = backward
	
	// Visual mode state
	visualAnchorLine   int                  // Line number where visual selection started
	visualAnchorX      int                  // X position where visual selection started
	visualAnchorOffset int                  // Byte offset where visual selection started

	// Working directory
	workingDirectory string                 // Current working directory (can include HOST: prefix)
	pendingDirectoryOpen bool              // Flag to show fuzzy list after editor starts
	pendingPath      string                 // Path to open after editor starts
	connectionStatus string                 // Status message for ongoing connections

	// Ex-command support
	lastCommandArgs  []string               // Arguments from the last ex-command

	// Rendering state
	renderingState  *RenderingState        // Manages overlay/modal state

	// Async operation tracking
	asyncOperations map[string]bool        // Track active async operations by ID
	spinnerIndex    int                    // Current spinner animation frame
	lastSpinnerTime time.Time              // Last time spinner was updated
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

	// Initialize search line widget (also at bottom of screen, shown in search mode)
	searchLine := widgets.NewCommandLineWidget(0, statusBarY, width)
	searchLine.SetStyle(config.Theme.GetStyle(config.Theme.CommandLine))

	// Start proxy manager
	proxyManager, err := proxy.NewProxyManager()
	if err != nil {
		return nil, fmt.Errorf("failed to start proxy manager: %v", err)
	}

	// Create initial empty buffer
	initialBuffer := buffer.NewBuffer("", "", config.DefaultIndentType, config.DefaultIndentWidth)

	// LSP channels will be removed - using goroutines on demand instead

	editor := &Editor{
		screen:          screen,
		buffer:          initialBuffer,
		config:          config,
		mode:            commands.NormalMode,
		commandLine:     commandLine,
		renderingState:  NewRenderingState(),
		searchLine:      searchLine,
		proxyManager:    proxyManager,
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
		visualAnchorLine:   -1,  // -1 means no selection
		visualAnchorX:      -1,
		visualAnchorOffset: -1,
		// Removed lspRequestChan and proxyResponseChan - using goroutines on demand
		workingDirectory: ".", // Default to current directory
		asyncOperations: make(map[string]bool),
		lastSpinnerTime: time.Now(),
		connectionStatus: "",
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

// Run starts the editor's main event loop with unified rendering.
func (e *Editor) Run() {
	// Initialize the screen
	if err := e.screen.Init(); err != nil {
		panic(err)
	}

	// Force a clear to ensure clean state
	e.screen.Clear()

	// Initial draw
	e.render()

	// Check if we have a pending path to open
	if e.pendingPath != "" {
		LL.Debug("Editor.Run: Found pendingPath: %s", e.pendingPath)
		// Handle the pending path asynchronously
		go e.handlePendingPath()
	} else if e.pendingDirectoryOpen {
		// Legacy support for direct directory open
		e.pendingDirectoryOpen = false
		e.ShowFuzzyListWidget()
	}

	// Get the screen event channel
	screenEvents := e.screen.EventChannel()

	// Start a timer that fires every 100ms for smooth animations
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Event loop
	for {
		var ev events.Event

		// Use select to handle multiple event sources
		select {
		case ev = <-screenEvents:
			// Got a screen event
		case <-ticker.C:
			// Timer fired - ev remains nil, will trigger redraw
		case ev = <-e.asyncEvents:
			// Got an async event
		}

		// Handle overlay events first
		if e.renderingState.IsOverlayActive() && e.renderingState.GetOverlay() != nil {
			// Always check for content updates (even without events)
			e.renderingState.GetOverlay().CheckContentUpdates()

			if ev != nil {
				done, result := e.renderingState.GetOverlay().HandleEvent(ev)
				if done {
					e.renderingState.CompleteOverlay(result)
				}
			}
		} else {
			// Handle normal editor events
			if ev != nil {
				e.handleEvent(ev)
			}
		}

		// Check if scrolling occurred
		e.scroll()

		// Render the entire screen
		e.render()
	}
}

// render performs the complete screen rendering
func (e *Editor) render() {
	t0 := e.perfTracker.Start()

	// Render base editor
	e.drawEditor()
	t0 = e.perfTracker.ReportSince("drawEditor", t0)

	e.drawContext()
	t0 = e.perfTracker.ReportSince("drawContext", t0)

	e.drawSignature()
	e.drawStatusBar()
	t0 = e.perfTracker.ReportSince("drawStatusBar", t0)

	// Render overlay if active
	if e.renderingState.IsOverlayActive() && e.renderingState.GetOverlay() != nil {
		if widgetScreen, ok := e.screen.(widgets.Screen); ok {
			e.renderingState.GetOverlay().Draw(widgetScreen)
		}
	}
	t0 = e.perfTracker.ReportSince("drawOverlay", t0)

	// Update cursor position
	if e.renderingState.IsOverlayActive() && e.renderingState.GetOverlay() != nil {
		// Get cursor from overlay
		x, y := e.renderingState.GetOverlay().GetCursorPosition()
		e.screen.ShowCursor(x, y)
	} else {
		// Normal cursor update
		e.updateCursor()
	}
	t0 = e.perfTracker.ReportSince("updateCursor", t0)

	e.screen.Show()
	e.perfTracker.ReportSince("screen.Show", t0)
}

// handleEvent processes a single event in normal (non-overlay) mode
func (e *Editor) handleEvent(ev events.Event) {
	switch ev := ev.(type) {
	case *events.KeyEvent:
		if e.mode == commands.CommandMode {
			e.handleCommandMode(ev)
		} else if e.mode == commands.SearchMode {
			e.handleSearchMode(ev)
		} else {
			// Normal and Insert modes
			if e.mode == commands.NormalMode && e.hasPendingOperation() && ev.Key == events.KeyEscape {
				e.cancelPendingOperation()
			} else if e.mode == commands.NormalMode && e.hasPendingOperation() && ev.Key == events.KeyRune {
				if err := e.handlePendingOperation(string(ev.Rune)); err != nil {
					LL.Debug("Pending operation error: %v", err)
				}
			} else if command, found := e.keyMapper.ProcessKey(ev, e.mode); found {
				LL.Debug("Found command: %s", command)
				if err := e.commandRegistry.Execute(command, nil); err != nil {
					LL.Debug("Command execution error: %v", err)
				}
			} else {
				LL.Debug("No command found for key")
				if e.mode == commands.InsertMode && ev.Key == events.KeyRune {
					e.insertChar(ev.Rune)
				}
			}
		}
	case *events.ResizeEvent:
		e.handleResize()
	case *events.CustomEvent:
		LL.Debug("handleEvent: Received CustomEvent type=%s", ev.Type)
		switch ev.Type {
		case "show-fuzzy-list":
			e.ShowFuzzyListWidget()
		case "open-file":
			if path, ok := ev.Data.(string); ok {
				LL.Debug("handleEvent: Opening file: %s", path)
				e.openFile(path)
			} else {
				LL.Debug("handleEvent: open-file event data is not a string")
			}
		}
	default:
		LL.Debug("Unsupported event type: %+v", reflect.TypeOf(ev))
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
	} else if e.mode == commands.SearchMode {
		// Show cursor in search line
		cursorX, cursorY := e.searchLine.GetCursorPosition()
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

// scrollToCenter positions the view so the cursor line appears in the middle of the screen
func (e *Editor) scrollToCenter() {
	visibleHeight := e.getVisibleHeight()
	cursorLine := e.buffer.CursorIterator.LineNumber()

	// Calculate the desired view start line to center the cursor
	desiredViewLine := cursorLine - visibleHeight/2

	// Handle edge cases
	if desiredViewLine < 0 {
		desiredViewLine = 0
	} else {
		// Don't scroll past the end of the buffer
		maxViewLine := e.buffer.LineCount - visibleHeight
		if maxViewLine < 0 {
			maxViewLine = 0
		}
		if desiredViewLine > maxViewLine {
			desiredViewLine = maxViewLine
		}
	}

	// Create iterator at the desired view line
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, desiredViewLine)
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

	// Check if the command is a line number
	if lineNum, err := strconv.Atoi(command); err == nil {
		// Jump to line number (1-based, but 0 also goes to line 1)
		if lineNum <= 0 {
			e.goToLine(0) // Line 0 or negative goes to first line
		} else {
			e.goToLine(lineNum - 1) // Convert to 0-based
		}
		return
	}

	// Parse command and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}
	cmd := parts[0]
	args := parts[1:]

	// Store command arguments for handlers that need them
	e.lastCommandArgs = args

	// Try to execute from the command registry
	if e.commandRegistry.Has(cmd) {
		err := e.commandRegistry.Execute(cmd, args)
		if err != nil {
			LL.Debug("Command execution failed: %v", err)
		}
		return
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


// handleSearchMode processes key events in search mode
func (e *Editor) handleSearchMode(ev *events.KeyEvent) {
	switch ev.Key {
	case events.KeyEscape:
		// Exit search mode without searching
		e.exitSearchMode()
	case events.KeyEnter:
		// Execute search and stay in normal mode
		e.executeSearch()
		e.exitSearchMode()
	default:
		// Pass other keys to the search line widget
		e.searchLine.HandleKey(ev)

		// Check if the user deleted all text including the slash
		// If so, exit search mode automatically
		if e.searchLine.GetText() == "" {
			e.exitSearchMode()
		}
	}
}

// enterSearchMode switches to search mode and initializes the search line
func (e *Editor) enterSearchMode() {
	e.mode = commands.SearchMode
	e.searchLine.SetText("/")
	// Position cursor after the slash
	e.searchLine.SetCursor(1)
	e.searchDirection = true // Forward search by default
}

// exitSearchMode switches back to normal mode
func (e *Editor) exitSearchMode() {
	e.mode = commands.NormalMode
	e.searchLine.SetText("")
}

// executeSearch performs the search operation
func (e *Editor) executeSearch() {
	fullSearch := e.searchLine.GetText()

	// Remove the leading "/"
	pattern := fullSearch
	if len(pattern) > 1 {
		pattern = pattern[1:]
	} else {
		// Empty search, do nothing
		return
	}

	// Store the search pattern
	e.searchPattern = pattern

	// Perform the search
	e.findNext()
}

// findNext finds the next occurrence of the search pattern
func (e *Editor) findNext() {
	if e.searchPattern == "" {
		return
	}

	// Get current position
	startPos := e.getAbsolutePosition()

	// Convert buffer content to string for searching
	content := e.buffer.Rope.String()

	// Search forward from current position + 1
	searchStart := startPos + 1
	if searchStart >= len(content) {
		searchStart = 0 // Wrap around
	}

	// Find the pattern
	foundPos := strings.Index(content[searchStart:], e.searchPattern)

	if foundPos == -1 && searchStart > 0 {
		// Not found after cursor, try from beginning
		foundPos = strings.Index(content[:startPos], e.searchPattern)
		if foundPos >= 0 {
			// Found before cursor (wrapped around)
			e.moveCursorToPosition(foundPos)
			return
		}
	} else if foundPos >= 0 {
		// Found after cursor
		e.moveCursorToPosition(searchStart + foundPos)
		return
	}

	// Pattern not found
	LL.Debug("Pattern not found: %s", e.searchPattern)
}

// findPrevious finds the previous occurrence of the search pattern
func (e *Editor) findPrevious() {
	if e.searchPattern == "" {
		return
	}

	// Get current position
	startPos := e.getAbsolutePosition()

	// Convert buffer content to string for searching
	content := e.buffer.Rope.String()

	// Search backward from current position
	searchEnd := startPos
	if searchEnd <= 0 {
		searchEnd = len(content) // Wrap around
	}

	// Find the last occurrence before current position
	foundPos := -1
	searchContent := content[:searchEnd]

	// Find all occurrences and take the last one
	lastPos := -1
	searchStart := 0
	for {
		pos := strings.Index(searchContent[searchStart:], e.searchPattern)
		if pos == -1 {
			break
		}
		lastPos = searchStart + pos
		searchStart = lastPos + 1
	}

	if lastPos != -1 {
		foundPos = lastPos
	} else if searchEnd < len(content) {
		// Not found before cursor, try after cursor (wrap around)
		afterContent := content[startPos:]
		// Find the last occurrence after cursor
		lastPos = -1
		searchStart = 0
		for {
			pos := strings.Index(afterContent[searchStart:], e.searchPattern)
			if pos == -1 {
				break
			}
			lastPos = searchStart + pos
			searchStart = lastPos + 1
		}
		if lastPos != -1 {
			foundPos = startPos + lastPos
		}
	}

	if foundPos != -1 {
		e.moveCursorToPosition(foundPos)
	} else {
		// Pattern not found
		LL.Debug("Pattern not found: %s", e.searchPattern)
	}
}

// findNextCmd is the command handler for the 'n' key
func (e *Editor) findNextCmd(args []string) error {
	e.findNext()
	return nil
}

// findPreviousCmd is the command handler for the 'N' key
func (e *Editor) findPreviousCmd(args []string) error {
	e.findPrevious()
	return nil
}

// moveCursorToPosition moves the cursor to an absolute position in the buffer
func (e *Editor) moveCursorToPosition(pos int) {
	// Remember the current view line before moving cursor (if iterator is still valid)
	oldViewLine := 0
	if e.buffer.ViewIterator != nil && e.buffer.ViewIterator.Generation == e.buffer.Generation {
		oldViewLine = e.buffer.ViewIterator.LineNumber()
	}

	// Create an iterator at the beginning
	iter := buffer.NewBufferIterator(e.buffer)
	lineNum := 0
	lineStart := 0

	// Find the line containing the position
	for !iter.IsEOF() {
		lineEnd := iter.LineEnd()
		if pos <= lineEnd {
			// Position is on this line
			break
		}
		lineStart = lineEnd + 1 // +1 for newline
		iter = iter.Next()
		lineNum++
	}

	// Set cursor position
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.CursorX = pos - lineStart

	// Ensure cursor X is within bounds
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(line) {
		e.buffer.CursorX = len(line)
		if e.buffer.CursorX > 0 && e.mode == commands.NormalMode {
			e.buffer.CursorX--
		}
	}

	// Check if the cursor is already visible
	visibleHeight := e.getVisibleHeight()
	
	// Ensure ViewIterator is valid
	if e.buffer.ViewIterator == nil || e.buffer.ViewIterator.Generation != e.buffer.Generation {
		e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, oldViewLine)
	}
	
	viewLine := e.buffer.ViewIterator.LineNumber()
	cursorLine := e.buffer.CursorIterator.LineNumber()

	// Check if cursor is within visible range
	if cursorLine >= viewLine && cursorLine < viewLine + visibleHeight {
		// Cursor is already visible, no need to scroll
		return
	}

	// Cursor is not visible, center it on screen
	e.scrollToCenter()

	// Check if the view actually changed
	newViewLine := e.buffer.ViewIterator.LineNumber()
	if newViewLine == oldViewLine {
		// View didn't change, use default scrolling instead
		e.scrollToShowAtTop()
	}
}

// handleResize processes screen resize events
func (e *Editor) handleResize() {
	width, _ := e.screen.Size()

	// Update the command line widget's dimensions and position
	e.commandLine.SetWidth(width)
	e.commandLine.SetPosition(0, e.getStatusBarY())

	// Also update search line widget
	e.searchLine.SetWidth(width)
	e.searchLine.SetPosition(0, e.getStatusBarY())
}

// startAsyncOperation registers a new async operation
func (e *Editor) startAsyncOperation(id string) {
	e.asyncOperations[id] = true
}

// endAsyncOperation marks an async operation as complete
func (e *Editor) endAsyncOperation(id string) {
	delete(e.asyncOperations, id)
}

// hasAsyncOperations returns true if any async operations are in progress
func (e *Editor) hasAsyncOperations() bool {
	return len(e.asyncOperations) > 0
}

// getSpinnerChar returns the current spinner character and advances the animation
func (e *Editor) getSpinnerChar() string {
	// Unicode spinner characters: ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	// Update spinner every 100ms
	now := time.Now()
	if now.Sub(e.lastSpinnerTime) > 100*time.Millisecond {
		e.spinnerIndex = (e.spinnerIndex + 1) % len(spinnerChars)
		e.lastSpinnerTime = now
	}

	return spinnerChars[e.spinnerIndex]
}

// Close finalizes the screen and proxy client.
func (e *Editor) Close() {
	// Stop the performance tracker
	if e.perfTracker != nil {
		e.perfTracker.Stop()
	}

	e.screen.Fini()
	if e.proxyManager != nil {
		e.proxyManager.Close()
	}
}


// parsePathComponents is a helper to parse HOST:PATH format
func parsePathComponents(path string) (string, string, bool) {
	colonIdx := strings.Index(path, ":")
	if colonIdx > 0 {
		// Check if it's not a Windows drive letter
		if colonIdx != 1 || (colonIdx == 1 && len(path) > 2 && path[2] != '/' && path[2] != '\\') {
			host := path[:colonIdx]
			remotePath := path[colonIdx+1:]
			// Basic validation - host should not contain slashes
			if !strings.Contains(host, "/") && !strings.Contains(host, "\\") {
				return host, remotePath, true
			}
		}
	}
	return "", path, false
}

// resolvePath resolves a path relative to the working directory
func (e *Editor) resolvePath(path string) string {
	LL.Debug("resolvePath: input=%s, workingDirectory=%s", path, e.workingDirectory)

	// If path is already absolute or has HOST: prefix, return as is
	if strings.HasPrefix(path, "/") || strings.Contains(path, ":") {
		LL.Debug("resolvePath: path is absolute or has HOST:, returning as is")
		return path
	}

	// If path is relative, prepend working directory
	if e.workingDirectory == "." {
		return path
	}

	// Handle working directory with HOST: prefix
	if strings.Contains(e.workingDirectory, ":") {
		// For remote working directory, just append the path
		var result string
		if strings.HasSuffix(e.workingDirectory, "/") {
			result = e.workingDirectory + path
		} else {
			result = e.workingDirectory + "/" + path
		}
		LL.Debug("resolvePath: remote working directory, result=%s", result)
		return result
	}

	// For local working directory
	if strings.HasSuffix(e.workingDirectory, "/") {
		return e.workingDirectory + path
	}
	return e.workingDirectory + "/" + path
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

// actuallyQuit exits the editor completely
func (e *Editor) actuallyQuit(args []string) error {
	e.Close()
	os.Exit(0)
	return nil
}

// enterVisualMode enters visual mode and sets the anchor point
func (e *Editor) enterVisualMode(args []string) error {
	e.mode = commands.VisualMode
	// Set anchor to current cursor position
	e.visualAnchorLine = e.buffer.CursorIterator.LineNumber()
	e.visualAnchorX = e.buffer.CursorX
	// Calculate the byte offset of the anchor
	e.visualAnchorOffset = e.buffer.CursorIterator.Position() + e.buffer.CursorX
	return nil
}

// exitVisualMode exits visual mode and returns to normal mode
func (e *Editor) exitVisualMode(args []string) error {
	e.mode = commands.NormalMode
	// Clear visual selection state
	e.visualAnchorLine = -1
	e.visualAnchorX = -1
	e.visualAnchorOffset = -1
	return nil
}

// getSelectionRange returns the start and end byte offsets of the current selection
// Returns (start, end, hasSelection)
func (e *Editor) getSelectionRange() (int, int, bool) {
	if e.mode != commands.VisualMode || e.visualAnchorOffset < 0 {
		return 0, 0, false
	}
	
	// Calculate current cursor byte offset
	currentOffset := e.buffer.CursorIterator.Position() + e.buffer.CursorX
	
	// Return in order (start, end)
	if e.visualAnchorOffset <= currentOffset {
		return e.visualAnchorOffset, currentOffset + 1, true // +1 for inclusive selection
	} else {
		return currentOffset, e.visualAnchorOffset + 1, true // +1 for inclusive selection
	}
}

func (e *Editor) changeDirectory(args []string) error {
	if len(args) == 0 {
		// Show current directory
		LL.Info("Current directory: %s", e.workingDirectory)
	} else {
		// Change directory
		newPath := strings.Join(args, " ")
		// Resolve relative paths
		if !strings.HasPrefix(newPath, "/") && !strings.Contains(newPath, ":") {
			newPath = e.resolvePath(newPath)
		}

		// Try to list the directory to verify it exists
		_, err := e.proxyManager.ListDirectory(newPath)
		if err != nil {
			LL.Error("Failed to change directory to %s: %v", newPath, err)
			return err
		} else {
			e.workingDirectory = newPath
			LL.Info("Changed directory to: %s", e.workingDirectory)
		}
	}
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

	// Launch goroutine to fetch folding ranges
	go func() {
		ranges, err := e.proxyManager.GetFoldingRanges(filePath)
		
		// Update the buffer directly if it still exists
		if b, exists := e.buffers[filePath]; exists && b == buf {
			if err != nil {
				LL.Debug("LSP folding ranges error for %s: %v", filePath, err)
				buf.HasLSP = false
				buf.FoldingRanges = nil
			} else {
				LL.Trace("Updated folding ranges for %s: got %d ranges", filePath, len(ranges))
				buf.HasLSP = true
				buf.FoldingRanges = ranges
			}
			
			// If this is the current buffer, invalidate context cache to trigger redraw
			if buf == e.buffer {
				e.contextLine = -1 // Force context update
			}
		}
	}()
}


// Insert mode commands
func (e *Editor) enterInsertMode(args []string) error {
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession("i")
	}
	return nil
}

// appendAfterCursor enters insert mode after the current cursor position
func (e *Editor) appendAfterCursor(args []string) error {
	// Move cursor one position to the right if not at end of line
	if !e.buffer.CursorIterator.IsEOF() {
		line := e.buffer.CursorIterator.Line()
		if e.buffer.CursorX < len(line) {
			e.buffer.CursorX++
		}
	}
	
	// Enter insert mode
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession("a")
	}
	return nil
}

// appendAtEndOfLine enters insert mode at the end of the current line
func (e *Editor) appendAtEndOfLine(args []string) error {
	// Move cursor to end of line
	if !e.buffer.CursorIterator.IsEOF() {
		line := e.buffer.CursorIterator.Line()
		e.buffer.CursorX = len(line)
	}
	
	// Enter insert mode
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession("A")
	}
	return nil
}

// setLastChange sets the last change and clears any lastCommand
func (e *Editor) setLastChange(entry buffer.HistoryEntry) {
	e.lastChange = entry
	e.lastCommand = nil // Clear operator+text object command
}

func (e *Editor) exitInsertMode(args []string) error {
	e.mode = commands.NormalMode

	// Adjust cursor position for normal mode
	// In vim, when exiting insert mode, if cursor is not at position 0, move back one
	if e.buffer != nil && e.buffer.CursorX > 0 {
		e.buffer.CursorX--
	}

	// Finalize insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		if compositeEntry := e.buffer.History.EndInsertSession(); compositeEntry != nil {
			e.setLastChange(compositeEntry)
		}
	}
	
	// Update syntax highlighting after exiting insert mode
	e.updateSyntaxHighlighting()
	
	return nil
}

// Undo/Redo commands
func (e *Editor) undo(args []string) error {
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

func (e *Editor) redo(args []string) error {
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

// yankCurrentLine yanks (copies) the current line to the system clipboard
func (e *Editor) yankCurrentLine(args []string) error {
	// Get current line
	line := e.buffer.CursorIterator.Line()
	
	// Add newline if we're not at the last line
	lineNum := e.buffer.CursorIterator.LineNumber()
	textToYank := line
	if lineNum < e.buffer.LineCount - 1 {
		textToYank += "\n"
	}
	
	// Copy to system clipboard
	if err := clipboard.Copy(textToYank); err != nil {
		LL.Error("Failed to copy to clipboard: %v", err)
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	
	LL.Debug("Yanked line %d (%d characters) to clipboard", lineNum + 1, len(textToYank))
	
	return nil
}

// repeatLastChange repeats the last text modification at the current cursor position
func (e *Editor) repeatLastChange(args []string) error {
	// First check if we have a repeatable command (operator + text object)
	if e.lastCommand != nil {
		// Repeat the operator+text object combination
		return e.executeOperatorWithTextObject(e.lastCommand.Operator, e.lastCommand.TextObject)
	}

	// Otherwise fall back to repeating the last change
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

	case *buffer.InsertModeEntry:
		// For insert mode entries, check if we need to move cursor first
		if original.InitCommand == "A" {
			// Move to end of line first
			line := e.buffer.CursorIterator.Line()
			cursorX = len(line)
		}
		
		// Then replay the composite entry
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
func (e *Editor) openLineBelow(args []string) error {
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
			e.setLastChange(executed)
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
		e.buffer.History.StartInsertSession("o")
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// openLineAbove opens a new line above the current line and enters insert mode
func (e *Editor) openLineAbove(args []string) error {
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
			e.setLastChange(executed)
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
				e.setLastChange(executed)
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
		e.buffer.History.StartInsertSession("O")
	}

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// joinLines joins the current line with the next line
func (e *Editor) joinLines(args []string) error {
	// Get current line info
	lineNum := e.buffer.CursorIterator.LineNumber()
	
	// Can't join if we're on the last line
	if lineNum >= e.buffer.LineCount - 1 {
		return nil
	}
	
	// Get current line and next line
	currentLine := e.buffer.CursorIterator.Line()
	nextIter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum + 1)
	if nextIter.IsEOF() {
		return nil
	}
	nextLine := nextIter.Line()
	
	// Determine what to delete and what to insert
	// We need to delete the newline and any leading whitespace on the next line
	leadingSpaces := 0
	for i, ch := range nextLine {
		if ch != ' ' && ch != '\t' {
			leadingSpaces = i
			break
		}
		if i == len(nextLine) - 1 {
			// Next line is all whitespace
			leadingSpaces = len(nextLine)
		}
	}
	
	// Determine what text to insert between the lines
	insertText := ""
	if len(currentLine) > 0 && currentLine[len(currentLine)-1] != ' ' && 
	   leadingSpaces < len(nextLine) && nextLine[leadingSpaces] != ' ' {
		// Add a space if current line doesn't end with space and next line content doesn't start with space
		insertText = " "
	}
	
	// Create a composite entry for the join operation
	compositeEntry := buffer.NewCompositeEntry("Join lines")
	
	// First, insert the space if needed
	if insertText != "" {
		insertEntry := buffer.NewInsertTextEntry(lineNum, len(currentLine), insertText)
		compositeEntry.Add(insertEntry)
	}
	
	// Then delete the newline and leading whitespace
	deleteEntry := buffer.NewDeleteTextEntry(lineNum, len(currentLine) + len(insertText), "\n" + nextLine[:leadingSpaces])
	compositeEntry.Add(deleteEntry)
	
	// Execute through history
	if e.buffer.History != nil {
		if executed, err := e.buffer.History.Execute(e.buffer, compositeEntry); err != nil {
			return err
		} else if executed != nil {
			e.setLastChange(executed)
		}
	}
	
	// Remember where we want the cursor to be (at the join point)
	cursorX := len(currentLine)
	if insertText != "" {
		cursorX++ // Account for the added space
	}
	
	// Recreate iterators
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, e.buffer.ViewIterator.LineNumber())
	
	// Set cursor position to the join point
	e.buffer.CursorX = cursorX
	
	// Update syntax highlighting
	e.updateSyntaxHighlighting()
	
	return nil
}

func (e *Editor) insertChar(ch rune) error {
	// Check if we should skip over a closing character
	if e.shouldSkipClosingChar(ch) {
		// Just move cursor forward without inserting
		e.buffer.CursorX++
		return nil
	}

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
			e.setLastChange(executed)
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

// shouldSkipClosingChar checks if we should skip over a closing character
// This happens when the character at the cursor position matches what we're trying to type
// and it's a closing character from auto-pairing
func (e *Editor) shouldSkipClosingChar(ch rune) bool {
	// Check if this is a closing character
	isClosing := false
	switch ch {
	case ')', ']', '}', '"', '\'', '`':
		isClosing = true
	}
	
	if !isClosing {
		return false
	}
	
	// Check if cursor is at a position where we have this character
	if e.buffer.CursorIterator.IsEOF() {
		return false
	}
	
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX >= len(line) {
		return false
	}
	
	// Check if the character at cursor position matches what we're typing
	return rune(line[e.buffer.CursorX]) == ch
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

func (e *Editor) insertNewline(args []string) error {
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
			e.setLastChange(executed)
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

func (e *Editor) backspace(args []string) error {
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
					e.setLastChange(executed)
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
					e.setLastChange(executed)
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
func (e *Editor) deleteCurrentCharacter(args []string) error {
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
					e.setLastChange(executed)
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
			e.setLastChange(executed)
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

	// Calculate visible range
	viewStartLine := e.buffer.ViewIterator.LineNumber()
	viewEndLine := viewStartLine + e.getVisibleHeight() - 1

	// Try to use tree-sitter for context if available
	if e.highlighter != nil && e.highlighter.GetTree() != nil {
		// Use tree-sitter based context detection
		e.contextCache = getTreeSitterContext(e.buffer, e.highlighter, currentLine, viewStartLine, viewEndLine, e.config.ScrollOffset)
	} else {
		// No tree-sitter available, no context
		e.contextCache = nil
		return
	}

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


func (e *Editor) enterCommandModeCmd(args []string) error {
	e.enterCommandMode()
	return nil
}

func (e *Editor) enterSearchModeCmd(args []string) error {
	e.enterSearchMode()
	return nil
}


// goToDefinition uses LSP to navigate to the definition of the symbol under cursor
func (e *Editor) showLSPBufferGo(args []string) error {
	// Save current buffer state first
	e.saveCurrentBufferState()

	// Get LSP messages for Go
	messages, err := e.proxyManager.GetLSPMessages("go")
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

func (e *Editor) goToDefinition(args []string) error {
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
	locations, err := e.proxyManager.GoToDefinition(e.buffer.FilePath, line, character)
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

	// If we're working with a remote file, preserve the host prefix
	if strings.Contains(e.buffer.FilePath, ":") {
		// Extract host from current buffer path
		parts := strings.SplitN(e.buffer.FilePath, ":", 2)
		if len(parts) == 2 {
			host := parts[0]
			// Add host prefix to the file path from LSP
			filePath = host + ":" + filePath
		}
	}

	LL.Debug("goToDefinition: location URI=%s, filePath=%s, current buffer=%s", location.URI, filePath, e.buffer.FilePath)

	// Push current position to navigation history before jumping
	e.pushNavigationHistory()

	// If it's a different file, open it
	if filePath != e.buffer.FilePath {
		LL.Debug("goToDefinition: opening different file: %s", filePath)
		e.openFile(filePath)
	} else {
		LL.Debug("goToDefinition: staying in same file")
	}

	// Jump to the location
	targetLine := location.Range.Start.Line
	e.buffer.CursorX = location.Range.Start.Character

	// Create iterator at the target line directly
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, targetLine)

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
// handlePendingPath processes the pending path asynchronously after UI is shown
func (e *Editor) handlePendingPath() {
	path := e.pendingPath
	e.pendingPath = "" // Clear it
	LL.Debug("handlePendingPath: Processing path: %s", path)

	// Parse the path to check if it's remote
	host, remotePath, isRemote := parsePathComponents(path)
	LL.Debug("handlePendingPath: host=%s, remotePath=%s, isRemote=%v", host, remotePath, isRemote)

	// Start async operation
	operationID := "path-resolution"
	e.startAsyncOperation(operationID)
	defer e.endAsyncOperation(operationID)

	// Update connection status
	if isRemote {
		e.connectionStatus = fmt.Sprintf("Connecting to %s...", host)
	} else {
		e.connectionStatus = "Loading..."
	}

	// Try to list the path to determine if it's a directory
	_, err := e.proxyManager.ListDirectoryRecursive(path)
	LL.Debug("handlePendingPath: ListDirectoryRecursive returned err=%v", err)

	// Clear connection status
	e.connectionStatus = ""

	if err == nil {
		// It's a directory
		e.workingDirectory = path
		LL.Debug("handlePendingPath: Opening directory: %s", e.workingDirectory)

		// Send async event to show fuzzy list
		LL.Debug("handlePendingPath: Sending show-fuzzy-list event")
		e.asyncEvents <- events.NewCustomEvent("show-fuzzy-list", nil)
	} else {
		// It's a file or doesn't exist - extract directory and open file
		LL.Debug("handlePendingPath: Path is a file, will open it")
		if isRemote {
			// For remote paths, extract directory
			lastSlash := strings.LastIndex(remotePath, "/")
			if lastSlash > 0 {
				dirPath := remotePath[:lastSlash]
				e.workingDirectory = host + ":" + dirPath
			} else {
				e.workingDirectory = host + ":~/"
			}
		} else {
			// For local paths, keep current directory
			// We don't want to change the working directory just because
			// we're opening a file in a subdirectory
			e.workingDirectory = "."
		}
		LL.Debug("Set working directory to: %s", e.workingDirectory)

		// Send async event to open the file
		LL.Debug("handlePendingPath: Sending open-file event for path: %s", path)
		e.asyncEvents <- events.NewCustomEvent("open-file", path)
	}
}

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
