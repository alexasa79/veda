package editor

import (
	"fmt"
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

// ReferencesBuffer tracks references found by LSP find-references
type ReferencesBuffer struct {
	References   []proxy.Location // LSP locations
	CurrentIndex int              // Current reference index for g n/g p (-1 if none)
	SourceFile   string           // File where find-references was called
	SourceLine   int              // Line where find-references was called
	SourceChar   int              // Character where find-references was called
	BufferName   string           // Name of the buffer displaying these references
	SymbolName   string           // Name of the symbol being referenced
}

// VirtualCursor represents a secondary cursor position for multi-cursor editing
type VirtualCursor struct {
	X int // Column position in the buffer
	Y int // Line number in the buffer
}

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
	countPrefix     string                  // Numeric prefix being built (e.g., "12" for 12j)
	repeatCount     int                     // Parsed count for command repetition
	pendingCount    int                     // Count for pending operations (dd, cc, etc.)

	// Search state
	searchPattern   string                  // Current search pattern
	searchDirection bool                    // true = forward, false = backward
	searchMatches   []int                   // Positions of all search matches
	currentMatch    int                     // Index of current match in searchMatches (-1 if none)

	// Flash highlight state
	flashStartLine     int                  // Start line of flashed region
	flashStartChar     int                  // Start char of flashed region
	flashEndLine       int                  // End line of flashed region
	flashEndChar       int                  // End char of flashed region
	flashEndTime       int64                // When to stop flashing (unix millis)

	// Multiple cursors
	virtualCursors     []VirtualCursor      // Additional cursors for multi-cursor editing

	// References tracking
	currentReferencesBuffer *ReferencesBuffer            // Current references buffer for g n/g p navigation
	referencesBuffers       map[string]*ReferencesBuffer // All references buffers by buffer name

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

	// Completion menu
	completionMenu  *widgets.CompletionMenu // Completion menu widget

	// Whitespace rendering
	whitespaceStyle whitespaceStyle        // Style and settings for whitespace visualization

	// Exit control
	shouldExit      bool                   // Flag to signal the editor should exit

	// Error management
	errorManager    *ErrorManager          // Manages error display and logging
	
	// Local history
	localHistory    *LocalHistory          // Manages local file history
	
	// Rewrap tracking
	lastRewrapLine  int                    // Line where last rewrap was performed
	lastRewrapChar  int                    // Character position where last rewrap was performed
	lastRewrapIndex int                    // Index into VerticalRulers for last rewrap
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
		flashStartLine:     -1,  // -1 means no flash
		flashStartChar:     -1,
		flashEndLine:       -1,
		currentMatch:       -1,  // -1 means no current match
		flashEndChar:       -1,
		flashEndTime:       0,
		referencesBuffers:  make(map[string]*ReferencesBuffer),
		// Removed lspRequestChan and proxyResponseChan - using goroutines on demand
		whitespaceStyle:    newWhitespaceStyle(config.Theme.GetStyle(config.Theme.Whitespace)),
		workingDirectory: ".", // Default to current directory
		asyncOperations: make(map[string]bool),
		lastSpinnerTime: time.Now(),
		connectionStatus: "",
		errorManager:    NewErrorManager(),
		completionMenu: widgets.NewCompletionMenu(config.Theme),
		lastRewrapLine:  -1,  // -1 means no previous rewrap
		lastRewrapChar:  -1,
		lastRewrapIndex: 0,
	}

	// Initialize local history
	localHistory, err := NewLocalHistory()
	if err != nil {
		LL.Error("Failed to initialize local history: %v", err)
		// Continue without local history - it's not critical
	} else {
		editor.localHistory = localHistory
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
	for !e.shouldExit {
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

	// Clear expired flash highlight
	e.clearFlashHighlight()

	// Render base editor
	e.drawEditor()
	t0 = e.perfTracker.ReportSince("drawEditor", t0)

	e.drawContext()
	t0 = e.perfTracker.ReportSince("drawContext", t0)

	e.drawSignature()
	e.drawErrorLine()
	e.drawStatusBar()
	t0 = e.perfTracker.ReportSince("drawStatusBar", t0)

	// Render overlay if active
	if e.renderingState.IsOverlayActive() && e.renderingState.GetOverlay() != nil {
		if widgetScreen, ok := e.screen.(widgets.Screen); ok {
			e.renderingState.GetOverlay().Draw(widgetScreen)
		}
	}
	t0 = e.perfTracker.ReportSince("drawOverlay", t0)

	// Render completion menu if visible
	if e.completionMenu.IsVisible() {
		if widgetScreen, ok := e.screen.(widgets.Screen); ok {
			e.completionMenu.Draw(widgetScreen)
		}
	}

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
		// Handle completion menu navigation if visible
		if e.mode == commands.InsertMode && e.completionMenu.IsVisible() {
			handled := e.handleCompletionMenuKey(ev)
			if handled {
				return
			}
		}

		if e.mode == commands.CommandMode {
			e.handleCommandMode(ev)
		} else if e.mode == commands.SearchMode {
			e.handleSearchMode(ev)
		} else if e.mode == commands.CursorMode {
			e.handleCursorMode(ev)
		} else {
			// Normal mode: Handle numeric prefix
			if e.mode == commands.NormalMode && ev.Key == events.KeyRune && ev.Rune >= '0' && ev.Rune <= '9' {
				// Special case: '0' is a command if no prefix is being built
				if ev.Rune == '0' && e.countPrefix == "" {
					// Let it be processed as a normal command (beginning-of-line)
				} else {
					// Build the count prefix
					e.countPrefix += string(ev.Rune)
					return // Don't process further
				}
			}

			// Normal and Insert modes
			if e.mode == commands.NormalMode && ev.Key == events.KeyEscape {
				if e.hasPendingOperation() {
					e.cancelPendingOperation()
					e.countPrefix = "" // Clear count prefix on escape
					e.repeatCount = 0
				} else if len(e.virtualCursors) > 0 {
					// Clear virtual cursors on escape in normal mode
					e.clearVirtualCursors()
				}
			} else if e.mode == commands.NormalMode && e.hasPendingOperation() {
				// Handle pending operations for both regular keys and special keys
				keyStr := ""
				if ev.Key == events.KeyRune {
					keyStr = string(ev.Rune)
				} else {
					// Convert special keys to their string names
					switch ev.Key {
					case events.KeyLeft:
						keyStr = "left"
					case events.KeyRight:
						keyStr = "right"
					case events.KeyUp:
						keyStr = "up"
					case events.KeyDown:
						keyStr = "down"
					default:
						// For other special keys, let them be processed normally
						if command, found := e.keyMapper.ProcessKey(ev, e.mode); found {
							LL.Debug("Key event: Key=%v, Rune=%q, Modifiers=%v", ev.Key, ev.Rune, ev.Modifiers)
							LL.Debug("Found command: %s", command)
							// Execute the command (this might cancel the pending operation)
							if e.countPrefix != "" {
								if count, err := strconv.Atoi(e.countPrefix); err == nil && count > 0 {
									e.repeatCount = count
								} else {
									e.repeatCount = 1
								}
								e.countPrefix = ""
							} else {
								e.repeatCount = 1
							}
							if err := e.executeWithRepeatCount(command); err != nil {
								LL.Debug("Command execution error: %v", err)
							}
						}
						return
					}
				}
				
				if keyStr != "" {
					if err := e.handlePendingOperation(keyStr); err != nil {
						LL.Debug("Pending operation error: %v", err)
					}
				}
			} else if command, found := e.keyMapper.ProcessKey(ev, e.mode); found {
				LL.Debug("Key event: Key=%v, Rune=%q, Modifiers=%v", ev.Key, ev.Rune, ev.Modifiers)
				LL.Debug("Found command: %s", command)

				// Parse the count prefix if we have one
				if e.countPrefix != "" {
					if count, err := strconv.Atoi(e.countPrefix); err == nil && count > 0 {
						e.repeatCount = count
					} else {
						e.repeatCount = 1
					}
					e.countPrefix = "" // Clear the prefix
				} else {
					e.repeatCount = 1
				}

				// Execute the command with repetition
				if err := e.executeWithRepeatCount(command); err != nil {
					LL.Debug("Command execution error: %v", err)
				}

				// Clear repeat count after execution
				e.repeatCount = 0
			} else {
				LL.Debug("No command found for key %+v", ev)
				if e.mode == commands.InsertMode && ev.Key == events.KeyRune {
					// Handle insert char for multiple cursors
					if len(e.virtualCursors) > 0 {
						e.insertCharForAllCursors(ev.Rune)
					} else {
						e.insertChar(ev.Rune)
					}
				}
				// Clear count prefix if no command was found
				if e.mode == commands.NormalMode {
					e.countPrefix = ""
					e.repeatCount = 0
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
		case "show-completions":
			if data, ok := ev.Data.(map[string]interface{}); ok {
				e.handleShowCompletions(data)
			}
		}
	default:
		LL.Debug("Unsupported event type: %+v", reflect.TypeOf(ev))
	}
}

// executeWithRepeatCount executes a command with the current repeat count
func (e *Editor) executeWithRepeatCount(command string) error {
	count := e.repeatCount
	if count <= 0 {
		count = 1
	}

	// Commands that need special handling for count
	switch command {
	case "yank-line":
		// For yy, count means "yank N lines" not "yank same line N times"
		return e.yankMultipleLines(count)
	case "start-delete", "start-change", "start-yank":
		// For dd/cc/yy, count means "delete/change/yank N lines" not "repeat operation N times"
		// Store the count for the pending operation
		e.pendingCount = count
		return e.executeCommandForAllCursors(command, nil)
	default:
		// Check if we should apply to all cursors
		applyToAll := false
		if e.mode != commands.CursorMode && len(e.virtualCursors) > 0 {
			// Get command info from registry
			if cmdInfo, exists := e.commandRegistry.GetCommand(command); exists {
				applyToAll = cmdInfo.ApplyToAllCursors
			}
		}

		if applyToAll {
			// Execute for all cursors
			for i := 0; i < count; i++ {
				if err := e.executeCommandForAllCursors(command, nil); err != nil {
					return err
				}
			}
		} else {
			// Execute normally (single cursor)
			for i := 0; i < count; i++ {
				if err := e.commandRegistry.Execute(command, nil); err != nil {
					return err
				}
			}
		}
		return nil
	}
}


// executeCommandForAllCursors executes a command for all cursors
func (e *Editor) executeCommandForAllCursors(command string, args []string) error {
	// Save view position and primary cursor position
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	primaryLineNum := e.buffer.CursorIterator.LineNumber()
	primaryCursorX := e.buffer.CursorX

	// First execute for virtual cursors (in reverse order to maintain positions)
	for i := len(e.virtualCursors) - 1; i >= 0; i-- {
		cursor := e.virtualCursors[i]

		// Temporarily set primary cursor to virtual cursor position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, cursor.Y)
		e.buffer.CursorX = cursor.X

		// Execute command
		if err := e.commandRegistry.Execute(command, args); err != nil {
			// Continue with other cursors even if one fails
			LL.Debug("Command failed for virtual cursor %d: %v", i, err)
		}

		// Update virtual cursor position after command
		e.virtualCursors[i].Y = e.buffer.CursorIterator.LineNumber()
		e.virtualCursors[i].X = e.buffer.CursorX
	}

	// Restore primary cursor position
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primaryLineNum)
	e.buffer.CursorX = primaryCursorX

	// Execute for primary cursor
	if err := e.commandRegistry.Execute(command, args); err != nil {
		return err
	}

	// Restore view iterator
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	return nil
}

// insertCharForAllCursors inserts a character at all cursor positions
func (e *Editor) insertCharForAllCursors(ch rune) error {
	// Save view position and primary cursor position
	viewLineNum := e.buffer.ViewIterator.LineNumber()
	primaryLineNum := e.buffer.CursorIterator.LineNumber()
	primaryCursorX := e.buffer.CursorX

	// First insert at virtual cursors (in reverse order to maintain positions)
	for i := len(e.virtualCursors) - 1; i >= 0; i-- {
		cursor := e.virtualCursors[i]

		// Temporarily set primary cursor to virtual cursor position
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, cursor.Y)
		e.buffer.CursorX = cursor.X

		// Insert character
		if err := e.insertChar(ch); err != nil {
			// Continue with other cursors even if one fails
			LL.Debug("Insert char failed for virtual cursor %d: %v", i, err)
		}

		// Update virtual cursor position after insert
		e.virtualCursors[i].Y = e.buffer.CursorIterator.LineNumber()
		e.virtualCursors[i].X = e.buffer.CursorX
	}

	// Restore primary cursor position
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primaryLineNum)
	e.buffer.CursorX = primaryCursorX

	// Insert at primary cursor
	if err := e.insertChar(ch); err != nil {
		return err
	}

	// Restore view iterator
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	return nil
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
		// Get visual position with line wrapping
		visualLine, visualCol := e.GetCursorVisualPosition()
		cursorY := visualLine + e.getContentStartY()
		e.screen.ShowCursor(visualCol, cursorY)
	}
}

// enterCommandMode switches to command mode and initializes the command line
// exitCommandMode switches back to normal mode
// executeCommand processes the command entered in the command line
// handleVariableAssignment processes variable assignments in VAR=VALUE format

// GetVirtualCursors returns the current virtual cursors
func (e *Editor) GetVirtualCursors() []VirtualCursor {
	return e.virtualCursors
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
	e.shouldExit = true
	return nil
}

// enterVisualMode enters visual mode and sets the anchor point
// exitVisualMode exits visual mode and returns to normal mode
// getSelectionRange returns the start and end byte offsets of the current selection
// Returns (start, end, hasSelection)
func (e *Editor) getSelectionRange() (int, int, bool) {
	if (e.mode != commands.VisualMode && e.mode != commands.VisualLineMode) || e.buffer == nil || e.buffer.VisualAnchorOffset < 0 {
		return 0, 0, false
	}

	if e.mode == commands.VisualLineMode {
		// For visual line mode, select entire lines
		anchorLine := e.buffer.VisualAnchorLine
		currentLine := e.buffer.CursorIterator.LineNumber()

		// Determine start and end lines
		startLine := anchorLine
		endLine := currentLine
		if currentLine < anchorLine {
			startLine = currentLine
			endLine = anchorLine
		}

		// Get byte offset for start of first line
		startIter := buffer.NewBufferIteratorAtLine(e.buffer, startLine)
		startOffset := startIter.Position()

		// Get byte offset for end of last line (including newline)
		endIter := buffer.NewBufferIteratorAtLine(e.buffer, endLine)
		endOffset := endIter.LineEnd()
		if endLine < e.buffer.LineCount - 1 {
			endOffset++ // Include the newline character
		}

		return startOffset, endOffset, true
	}

	// Character-wise visual mode
	currentOffset := e.buffer.CursorIterator.Position() + e.buffer.CursorX

	// Return in order (start, end)
	if e.buffer.VisualAnchorOffset <= currentOffset {
		return e.buffer.VisualAnchorOffset, currentOffset + 1, true // +1 for inclusive selection
	} else {
		return currentOffset, e.buffer.VisualAnchorOffset + 1, true // +1 for inclusive selection
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
	// Only clear lastCommand if it's not a change operator
	// Change operators need both lastCommand and lastChange for proper repeat
	if e.lastCommand != nil && e.lastCommand.Operator.GetType() == "change" {
		// Keep lastCommand for change operators
		return
	}
	e.lastCommand = nil // Clear operator+text object command for other cases
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
	// If we have virtual cursors, use multi-cursor yank
	if len(e.virtualCursors) > 0 {
		return e.yankForAllCursors(e.yankLineForCursor)
	}
	// Default to yanking just the current line
	return e.yankMultipleLines(1)
}

// yankMultipleLines yanks the specified number of lines starting from the current cursor position
func (e *Editor) yankMultipleLines(count int) error {
	if count <= 0 {
		count = 1
	}
	
	// If we have virtual cursors and count is 1, use multi-cursor yank
	if len(e.virtualCursors) > 0 && count == 1 {
		return e.yankForAllCursors(e.yankLineForCursor)
	}

	// Get starting line
	startLine := e.buffer.CursorIterator.LineNumber()
	endLine := startLine + count - 1

	// Ensure we don't go past end of buffer
	if endLine >= e.buffer.LineCount {
		endLine = e.buffer.LineCount - 1
	}

	// Build the text to yank
	var textToYank strings.Builder
	iter := e.buffer.CursorIterator

	for i := 0; i <= endLine - startLine; i++ {
		if i > 0 {
			iter = iter.Next()
			if iter.IsEOF() {
				break
			}
		}
		line := iter.Line()
		textToYank.WriteString(line)
		textToYank.WriteString("\n")
	}

	// Copy to system clipboard
	yankedText := textToYank.String()
	if err := clipboard.Copy(yankedText); err != nil {
		LL.Error("Failed to copy to clipboard: %v", err)
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	actualCount := endLine - startLine + 1
	LL.Debug("Yanked %d lines (%d characters) to clipboard", actualCount, len(yankedText))

	// Flash the yanked lines
	// Get end line length
	endIter := buffer.NewBufferIteratorAtLine(e.buffer, endLine)
	endLineLen := 0
	if !endIter.IsEOF() {
		endLineLen = len(endIter.Line())
	}
	e.setFlashHighlight(startLine, 0, endLine, endLineLen)

	return nil
}

// repeatLastChange repeats the last text modification at the current cursor position
func (e *Editor) repeatLastChange(args []string) error {
	// First check if we have a repeatable command (operator + text object)
	if e.lastCommand != nil {
		// Check if the last command was a change operator
		if e.lastCommand.Operator.GetType() == "change" && e.lastChange != nil {
			// For change operators, we need to:
			// 1. Execute the operator (which deletes text and enters insert mode)
			// 2. Exit insert mode immediately
			// 3. Then replay the insert session

			// Save the insert change before executing the operator
			savedChange := e.lastChange

			// Execute the operator (delete + enter insert mode)
			if err := e.executeOperatorWithTextObject(e.lastCommand.Operator, e.lastCommand.TextObject); err != nil {
				return err
			}

			// Exit insert mode without recording anything
			if e.mode == commands.InsertMode {
				e.mode = commands.NormalMode
				if e.buffer != nil && e.buffer.History != nil {
					e.buffer.History.EndInsertSession()
				}
			}

			// Restore the saved change for next repeat
			e.lastChange = savedChange

			// Now replay the insert session
			if insertEntry, ok := savedChange.(*buffer.InsertModeEntry); ok && len(insertEntry.Entries) > 0 {
				// Save line numbers before iterators become invalid
				savedCursorLine := e.buffer.CursorIterator.LineNumber()
				savedViewLine := e.buffer.ViewIterator.LineNumber()

				// Get current cursor position
				lineNum := savedCursorLine
				cursorX := e.buffer.CursorX

				// Simply replay all the insert operations at the current position
				for _, entry := range insertEntry.Entries {
					switch origEntry := entry.(type) {
					case *buffer.InsertTextEntry:
						// Create new entry at current position
						newEntry := buffer.NewInsertTextEntry(lineNum, cursorX, origEntry.Text)
						if executed, err := e.buffer.History.Execute(e.buffer, newEntry); err != nil {
							return err
						} else if executed != nil {
							// Update cursor position based on the insert
							e.buffer.CursorX = cursorX + len(origEntry.Text)
							cursorX = e.buffer.CursorX
						}
					}
				}

				// Recreate iterators after all inserts are complete
				e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
				if savedViewLine >= e.buffer.LineCount {
					savedViewLine = e.buffer.LineCount - 1
				}
				e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, savedViewLine)
			}

			return nil
		}

		// For non-change operators, just repeat the operator+text object
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
				// For 'a' command, we need to adjust by 1 since 'a' moves cursor right before inserting
				if original.InitCommand == "a" {
					baseCharOffset += 1
				}
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









// goToDefinition uses LSP to navigate to the definition of the symbol under cursor
func (e *Editor) showLSPBufferGo(args []string) error {
	// Save current buffer state first
	e.saveCurrentBufferState()

	// Get LSP messages for Go
	messages, err := e.proxyManager.GetLSPMessages(e.workingDirectory, "go")
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
		e.errorManager.ShowError("No file open")
		return fmt.Errorf("no file open")
	}

	// Get cursor position (LSP uses 0-based line numbers)
	line := e.buffer.CursorIterator.LineNumber()
	character := e.buffer.CursorX

	LL.Trace("goToDefinition: file=%s, line=%d, character=%d", e.buffer.FilePath, line, character)

	// Get definition location from LSP
	locations, err := e.proxyManager.GoToDefinition(e.buffer.FilePath, line, character)
	if err != nil {
		e.errorManager.ShowError("LSP error: %v", err)
		LL.Debug("goToDefinition failed: %v", err)
		return err
	}

	LL.Trace("goToDefinition: got %d locations", len(locations))

	// If no locations found, nothing to do
	if len(locations) == 0 {
		e.errorManager.ShowError("No definition found")
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

	// Position the definition at the center of the screen
	e.scrollToCenter()

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

// setFlashHighlight sets a temporary highlight for a text range
func (e *Editor) setFlashHighlight(startLine, startChar, endLine, endChar int) {
	e.flashStartLine = startLine
	e.flashStartChar = startChar
	e.flashEndLine = endLine
	e.flashEndChar = endChar
	e.flashEndTime = time.Now().UnixMilli() + 500 // Flash for 500ms
}

// clearFlashHighlight clears the flash highlight if it has expired
func (e *Editor) clearFlashHighlight() {
	if e.flashEndTime > 0 && time.Now().UnixMilli() > e.flashEndTime {
		e.flashStartLine = -1
		e.flashStartChar = -1
		e.flashEndLine = -1
		e.flashEndChar = -1
		e.flashEndTime = 0
	}
}

// isInFlashRange checks if a position is within the flash highlight range
func (e *Editor) isInFlashRange(line, charPos int) bool {
	if e.flashStartLine == -1 || e.flashEndTime == 0 {
		return false
	}

	// Check if flash has expired
	if time.Now().UnixMilli() > e.flashEndTime {
		return false
	}

	// Single line flash
	if e.flashStartLine == e.flashEndLine {
		return line == e.flashStartLine && charPos >= e.flashStartChar && charPos < e.flashEndChar
	}

	// Multi-line flash
	if line == e.flashStartLine {
		return charPos >= e.flashStartChar
	} else if line == e.flashEndLine {
		return charPos < e.flashEndChar
	} else if line > e.flashStartLine && line < e.flashEndLine {
		return true
	}

	return false
}

// hasFlashHighlight returns true if there is an active flash highlight
func (e *Editor) hasFlashHighlight() bool {
	return e.flashStartLine != -1 && e.flashEndTime > 0 && time.Now().UnixMilli() <= e.flashEndTime
}

