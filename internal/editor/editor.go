package editor

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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

// AutoIndentInfo tracks auto-inserted indentation on a line
type AutoIndentInfo struct {
	LineNumber       int    // Line number where auto-indent was applied
	AutoIndentLength int    // Number of characters that were auto-inserted
	HasUserContent   bool   // Whether user has typed non-whitespace content on this line
	BufferGeneration uint64 // Buffer generation when this was created (for invalidation)
}

// Editor represents the state of the text editor.
type Editor struct {
	screen             Screen
	buffer             *buffer.Buffer // Current buffer
	config             *config.Config
	mode               commands.Mode
	temporaryNormal    bool // True when in temporary normal mode (Ctrl+O from insert)
	commandLine        *widgets.CommandLineWidget
	searchLine         *widgets.CommandLineWidget // For search mode (/)
	proxyManager       *proxy.ProxyManager
	commandRegistry    *commands.CommandRegistry
	keyMapper          *commands.KeyMapper
    bufferHighlighters map[*buffer.Buffer]*syntax.Highlighter
	perfTracker        *perf.TimeTracker

	// Buffer management
	buffers        map[string]*buffer.Buffer // Map of file path to buffer
	bufferOrder    []string                  // Order of buffers for cycling
	previousBuffer *buffer.Buffer            // Previous buffer for quick switching

	// Navigation history
	navHistory    []NavigationEntry // Stack of previous positions
	maxNavHistory int               // Maximum number of entries to keep

    // Context display (showContext remains global preference; cache is per-buffer)
    showContext  bool // Whether to show context

	// Async notifications
	asyncEvents chan events.Event // Channel for async events

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
	lastChange        buffer.HistoryEntry // Last text modification for repeat command
	lastCommand       *RepeatableCommand  // Last operator+text object command for repeat
	lastSimpleCommand string              // Name of last simple command (like indent-line) for repeat
	lastCommandCount  int                 // Count used with the last command for repeat

	// Pending operation for multi-key commands
	pendingOperator Operator           // Current operator waiting for text object
	pendingModifier TextObjectModifier // Modifier for text object (a/i)
	pendingKeys     []string           // Accumulated keys for current operation
	countPrefix     string             // Numeric prefix being built (e.g., "12" for 12j)
	repeatCount     int                // Parsed count for command repetition
	pendingCount    int                // Count for pending operations (dd, cc, etc.)

	// Search state
	searchPattern   string // Current search pattern
	searchDirection bool   // true = forward, false = backward
	searchMatches   []int  // Positions of all search matches
	currentMatch    int    // Index of current match in searchMatches (-1 if none)

	// Flash highlight state
	flashStartLine int   // Start line of flashed region
	flashStartChar int   // Start char of flashed region
	flashEndLine   int   // End line of flashed region
	flashEndChar   int   // End char of flashed region
	flashEndTime   int64 // When to stop flashing (unix millis)

	// Multiple cursors
	virtualCursors []VirtualCursor // Additional cursors for multi-cursor editing

	// Transient info banner (persists until next key press)
	transientInfoMessage string
	transientInfoActive  bool

	// Desired column for vertical cursor movement
	desiredColumn int // Desired column position when moving up/down (-1 means unset)

	// References tracking
	currentReferencesBuffer *ReferencesBuffer            // Current references buffer for g n/g p navigation
	referencesBuffers       map[string]*ReferencesBuffer // All references buffers by buffer name

	// Working directory
	workingDirectory     string // Current working directory (can include HOST: prefix)
	pendingDirectoryOpen bool   // Flag to show fuzzy list after editor starts
	pendingPath          string // Path to open after editor starts
	connectionStatus     string // Status message for ongoing connections

	// Ex-command support
	lastCommandArgs []string // Arguments from the last ex-command

	// Rendering state
	renderingState *RenderingState // Manages overlay/modal state

	// Registers support
	registers              map[rune]string // Named registers by single character
	registerPrefixActive   bool            // true after receiving '\'' awaiting register name/command
	pendingRegister        rune            // register name captured after '\''
	registerReadyForAction bool            // true when a register has been selected and awaiting an action key (y/p/P)
	registerTargetForYank  rune            // if non-zero, next yank stores into this register instead of clipboard

	// Async operation tracking
	asyncOperations map[string]bool // Track active async operations by ID
	spinnerIndex    int             // Current spinner animation frame
	lastSpinnerTime time.Time       // Last time spinner was updated
	spinnerRunning  bool            // True while spinner tick goroutine is active
	spinnerStop     chan struct{}   // Stop channel for spinner tick goroutine

	// Completion menu
	completionMenu *widgets.CompletionMenu // Completion menu widget

	// Key hints popup
	keyHintsPopup *widgets.KeyHintsPopup // Key hints popup widget

	// Whitespace rendering
	whitespaceStyle whitespaceStyle // Style and settings for whitespace visualization

	// Exit control
	shouldExit bool // Flag to signal the editor should exit

	// Error management
	errorManager *ErrorManager // Manages error display and logging

	// Local history
	localHistory *LocalHistory // Manages local file history

	// Auto-save manager
	autoSaveManager *AutoSaveManager // Manages auto-save functionality

	// Auto-indent cleanup tracking
	autoIndentLines map[string]AutoIndentInfo // Maps buffer path + line number to auto-indent info
	lastCursorLine  int                       // Last line cursor was on (for cleanup tracking)

	// Rewrap tracking
	lastRewrapLine  int // Line where last rewrap was performed
	lastRewrapChar  int // Character position where last rewrap was performed
	lastRewrapIndex int // Index into VerticalRulers for last rewrap

	// Bracketed paste state
	inPasteMode bool // True while receiving bracketed paste content
	// When true, the next line-wise paste should not auto-rebase indentation
	skipAutoIndentForNextPaste bool

	// Remote directory cache: maps normalized working directory to file list
	dirCache map[string][]proxy.FileInfo

	// Windows management
	windows      []*Window // When empty, single-window mode using e.buffer
	activeWindow int       // Index of active window in windows slice
	lastWindow   int       // Index of previously active window (-1 if none)

	// Window sizing preferences
	minActiveWindowWidth int // Minimum width (columns) for the active window

	// Terminal sessions per buffer (PTY via proxy)
	terminalSessions map[string]*TerminalSessionState
}

// TerminalSessionState tracks a PTY-backed terminal attached to a buffer
type TerminalSessionState struct {
	SessionID  string
	WorkingDir string
	Cols       int
	Rows       int
	Closed     bool
	ExitCode   *int
	// Minimal emulator state for this session
	term *TerminalEmulator
	// Optional banner displayed above emulator content, not sent to PTY
	Banner string
	// Saved terminal cursor when switching to browse (Normal) mode
	SavedCurX int
	SavedCurY int
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

	// LSP channels will be removed - using goroutines on demand instead

	editor := &Editor{
		screen:               screen,
		buffer:               nil, // No initial buffer - will show "[empty]" until file is opened
		config:               config,
		mode:                 commands.NormalMode,
		commandLine:          commandLine,
		renderingState:       NewRenderingState(),
		searchLine:           searchLine,
		proxyManager:         proxyManager,
		commandRegistry:      commands.NewCommandRegistry(),
		keyMapper:            commands.NewKeyMapper(),
        bufferHighlighters:   make(map[*buffer.Buffer]*syntax.Highlighter),
		perfTracker:          perf.New(true, 30*time.Second), // Report every 30 seconds
		buffers:              make(map[string]*buffer.Buffer),
		bufferOrder:          []string{},
		navHistory:           []NavigationEntry{},
		maxNavHistory:        50,   // Keep last 50 navigation entries
        showContext:          true, // Show context by default
		asyncEvents:          make(chan events.Event, 100), // Buffered channel for async events
		flashStartLine:       -1,                           // -1 means no flash
		flashStartChar:       -1,
		flashEndLine:         -1,
		desiredColumn:        -1, // -1 means unset
		currentMatch:         -1, // -1 means no current match
		flashEndChar:         -1,
		flashEndTime:         0,
		transientInfoMessage: "",
		transientInfoActive:  false,
		referencesBuffers:    make(map[string]*ReferencesBuffer),
		// Removed lspRequestChan and proxyResponseChan - using goroutines on demand
		whitespaceStyle:      newWhitespaceStyle(config.Theme.GetStyle(config.Theme.Whitespace)),
		workingDirectory:     ".", // Default to current directory
		asyncOperations:      make(map[string]bool),
		spinnerStop:          make(chan struct{}),
		lastSpinnerTime:      time.Now(),
		connectionStatus:     "",
		errorManager:         NewErrorManager(),
		completionMenu:       widgets.NewCompletionMenu(config.Theme),
		keyHintsPopup:        widgets.NewKeyHintsPopup(config.Theme),
		lastRewrapLine:       -1, // -1 means no previous rewrap
		lastRewrapChar:       -1,
		lastRewrapIndex:      0,
		inPasteMode:          false,
		dirCache:             make(map[string][]proxy.FileInfo),
		windows:              nil,
		activeWindow:         0,
		lastWindow:           0,
		minActiveWindowWidth: 110,
		terminalSessions:     make(map[string]*TerminalSessionState),
	}

	// Initialize windows with a single empty window (always have a window)
	editor.windows = []*Window{newWindow(nil)} // nil buffer shows "[empty]"
	editor.activeWindow = 0
	editor.recomputeWindowsLayout()

	// Initialize local history
	localHistory, err := NewLocalHistory()
	if err != nil {
		LL.Error("Failed to initialize local history: %v", err)
		// Continue without local history - it's not critical
	} else {
		editor.localHistory = localHistory
	}

	// Initialize auto-save manager
	editor.autoSaveManager = NewAutoSaveManager()
	editor.autoSaveManager.SetSaveCallback(editor.performAutoSave)

	// Initialize auto-indent tracking
	editor.autoIndentLines = make(map[string]AutoIndentInfo)
	editor.lastCursorLine = -1

	// No initial buffer - buffers map starts empty
	// When no file arguments are provided, user can create new buffer with :new or by typing

	// LSP callbacks will be set up when buffers are created

	// Set up proxy status callback to update connection status
    proxyManager.SetStatusCallback(func(status string) {
        editor.connectionStatus = status
        // Wake main loop so the status/spinner updates without keypress
        editor.asyncEvents <- events.NewCustomEvent("spinner-tick", nil)
        // If status becomes empty (completion), force a final render wake
        if status == "" {
            editor.asyncEvents <- events.NewCustomEvent("render", nil)
        }
    })

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

    // No always-on ticker. We'll wake the loop via asyncEvents for spinner/overlay updates.

	// Event loop
	for !e.shouldExit {
		var ev events.Event

        // Use select to handle multiple event sources
        select {
        case ev = <-screenEvents:
            // Got a screen event
        case ev = <-e.asyncEvents:
            // Got an async event
        }

		// Handle overlay events first
		if e.renderingState.IsOverlayActive() && e.renderingState.GetOverlay() != nil {
			// Always check for content updates (even without events)
			e.renderingState.GetOverlay().CheckContentUpdates()

			if ev != nil {
				done, result := e.renderingState.GetOverlay().HandleEvent(ev)
				if result == "__REFRESH__" {
					// Refresh current fuzzy list content from remote (bypass cache)
					// We need to identify if the active overlay is the file list. If so, trigger refresh.
					// Here we simply re-issue the directory listing request with useCache=false.
					if overlay := e.renderingState.GetOverlay(); overlay != nil {
						// Only handle refresh for FuzzyListOverlay
						if fuzzyOverlay, ok := overlay.(*FuzzyListOverlay); ok {
							// Create a new update channel and swap it in by creating a new overlay
							contentUpdateChan := make(chan []string, 1)
							// Kick off fresh request
							_ = e.startFuzzyFileListRefresh(contentUpdateChan)
							// Replace overlay with a new one that uses the same widget and new channel
							// Keep same widget instance so selection and text persist
							newOverlay := NewFuzzyListOverlay(fuzzyOverlay.widget, fuzzyOverlay.onTextChanged, contentUpdateChan)
							e.renderingState.SetOverlay(newOverlay)
						}
					}
				}
				if done {
					e.renderingState.CompleteOverlay(result)
					// Clear screen to remove overlay artifacts
					e.screen.Clear()
				}
			}
		} else {
			// Handle normal editor events
			if ev != nil {
				// Measure end-to-end latency from event creation to handling start
				if e.perfTracker != nil {
					age := time.Since(ev.When())
					// Bucket event age to a histogram-like series
					// Keep metric names simple to avoid unbounded labels
					start := time.Now().Add(-age)
					if age < 2*time.Millisecond {
						e.perfTracker.ReportSince("event.age.<2ms", start)
					} else if age < 5*time.Millisecond {
						e.perfTracker.ReportSince("event.age.<5ms", start)
					} else if age < 10*time.Millisecond {
						e.perfTracker.ReportSince("event.age.<10ms", start)
					} else if age < 20*time.Millisecond {
						e.perfTracker.ReportSince("event.age.<20ms", start)
					} else {
						e.perfTracker.ReportSince("event.age.>=20ms", start)
					}
				}

				tHandle := e.perfTracker.Start()
				e.handleEvent(ev)
				e.perfTracker.ReportSince("handleEvent", tHandle)
			}
		}

		// Check if scrolling occurred
		if !e.inPasteMode {
			tScroll := e.perfTracker.Start()
			e.scroll()
			e.perfTracker.ReportSince("scroll", tScroll)
		}

		// Render the entire screen unless in paste mode
		if !e.inPasteMode {
			e.render()
		}
	}
}

// render performs the complete screen rendering
func (e *Editor) render() {
	t0 := e.perfTracker.Start()

	// Clear expired flash highlight
	e.clearFlashHighlight()

	// Unified rendering for any number of windows (including single-window)
	// Ensure there is at least one window to render into
	if len(e.windows) == 0 {
		// Create a default window covering the content area and attach current buffer
		w := &Window{Buffer: e.buffer}
		e.windows = []*Window{w}
		e.activeWindow = 0
		e.recomputeWindowsLayout()
	}

	// Persist active window's state from the live buffer before drawing windows,
	// so any cursor/scroll changes made by event handlers are not lost.
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
		if w := e.windows[e.activeWindow]; w != nil && e.buffer != nil && w.Buffer == e.buffer {
			w.ViewLine = e.buffer.ViewIterator.LineNumber()
			w.CursorLine = e.buffer.CursorIterator.LineNumber()
			w.CursorX = e.buffer.CursorX
		}
	}

	// Draw each window using a windowScreen. Inactive windows are dimmed.
	for i, win := range e.windows {
		// Per-window performance breakdown to analyze drawWindows cost
		winTimer := e.perfTracker.Start()
		if win == nil {
			continue
		}

		// Debug: print buffer and window-specific state before rendering this window
		if win.Buffer == nil {
			LL.Debug("render.window[%d]: buffer=<empty> viewTop=%d cursorLine=%d cursorX=%d", i, win.ViewLine, win.CursorLine, win.CursorX)
		} else {
			LL.Debug("render.window[%d]: buffer=%s viewTop=%d cursorLine=%d cursorX=%d", i, win.Buffer.DisplayName(), win.ViewLine, win.CursorLine, win.CursorX)
		}

		// Handle empty windows (Buffer == nil)
		if win.Buffer == nil {
			dim := i != e.activeWindow
			sub := newWindowScreen(e.screen, win.X, win.Y, win.Width, win.Height, dim)
			e.drawEmptyWindow(sub, i == e.activeWindow)
			continue
		}
		// Prepare sub-screen
		dim := i != e.activeWindow
		sub := newWindowScreen(e.screen, win.X, win.Y, win.Width, win.Height, dim)
		// Measure prepare phase
		winTimer = e.perfTracker.ReportSince("drawWindows.window.prepare", winTimer)

        // Temporarily swap screen and buffer per window
		realScreen := e.screen
		realBuffer := e.buffer
		e.screen = sub
		// Temporarily map editor-wide iterators/state from window's own view/cursor
		// Save buffer iterators/X
		savedView := win.Buffer.ViewIterator
		savedCursor := win.Buffer.CursorIterator
		savedX := win.Buffer.CursorX
		// Apply window state
		win.Buffer.ViewIterator = buffer.NewBufferIteratorAtLine(win.Buffer, max(0, win.ViewLine))
		win.Buffer.CursorIterator = buffer.NewBufferIteratorAtLine(win.Buffer, max(0, win.CursorLine))
		// Do not special-case EOF here; EOF is a valid state for empty buffers
		win.Buffer.CursorX = win.CursorX
		e.buffer = win.Buffer
        // Ensure per-buffer highlighter is initialized and parsed
        if e.buffer.Highlighter != nil {
            if e.buffer.IsInternal {
                // For internal buffers we often use markdown; ensure parsed
                if e.buffer.Highlighter.GetTree() == nil {
                    _ = e.buffer.Highlighter.SetLanguage("markdown")
                    e.buffer.Highlighter.ParseFull([]byte(e.buffer.Rope.String()))
                }
            } else if e.buffer.HasFile() {
                if e.buffer.Highlighter.GetTree() == nil {
                    _ = e.buffer.Highlighter.SetLanguageFromFile(e.buffer.FilePath)
                    e.buffer.Highlighter.ParseFull([]byte(e.buffer.Rope.String()))
                }
            }
        }
		// Draw editor content area then context (to match legacy order)
		e.drawEditor()
		// Measure drawEditor phase; split active vs inactive to keep label cardinality bounded
		if i == e.activeWindow {
			winTimer = e.perfTracker.ReportSince("drawWindows.window.drawEditor.active", winTimer)
		} else {
			winTimer = e.perfTracker.ReportSince("drawWindows.window.drawEditor.inactive", winTimer)
		}

		e.drawContext()
		winTimer = e.perfTracker.ReportSince("drawWindows.window.drawContext", winTimer)

		// Restore
		// Persist any scroll/cursor changes back into the window state
		win.ViewLine = win.Buffer.ViewIterator.LineNumber()
		win.CursorLine = win.Buffer.CursorIterator.LineNumber()
		win.CursorX = win.Buffer.CursorX
		// Restore buffer iterators/X
		// Only restore if the saved iterators are still valid for the current buffer generation.
		// If buffer content changed during rendering (e.g., async updates), keep the fresh iterators
		// we created above to avoid generation mismatches.
		if savedView != nil && savedView.Generation == win.Buffer.Generation {
			win.Buffer.ViewIterator = savedView
		}
		if savedCursor != nil && savedCursor.Generation == win.Buffer.Generation {
			win.Buffer.CursorIterator = savedCursor
		}
		win.Buffer.CursorX = savedX
		e.screen = realScreen
		e.buffer = realBuffer
		winTimer = e.perfTracker.ReportSince("drawWindows.window.restore", winTimer)
		// Report total time for this window
		if i == e.activeWindow {
			_ = e.perfTracker.ReportSince("drawWindows.window.total.active", winTimer)
		} else {
			_ = e.perfTracker.ReportSince("drawWindows.window.total.inactive", winTimer)
		}
	}
	t0 = e.perfTracker.ReportSince("drawWindows", t0)

	// Draw global UI once (signature, banners, status bar) for the active window/buffer.
	{
		win := e.windows[e.activeWindow]
		activeBuf := win.Buffer
		if activeBuf != nil {
			realBuf := e.buffer
			// Temporarily sync active window's state to buffer to display correct line/col
			w := win
			savedView := activeBuf.ViewIterator
			savedCursor := activeBuf.CursorIterator
			savedX := activeBuf.CursorX
			activeBuf.ViewIterator = buffer.NewBufferIteratorAtLine(activeBuf, max(0, w.ViewLine))
			activeBuf.CursorIterator = buffer.NewBufferIteratorAtLine(activeBuf, max(0, w.CursorLine))
			activeBuf.CursorX = w.CursorX
			e.buffer = activeBuf
			// Signature above status bar
			e.drawSignature()
			e.drawErrorLine()
			e.drawStatusBar()
			// restore
			activeBuf.ViewIterator = savedView
			activeBuf.CursorIterator = savedCursor
			activeBuf.CursorX = savedX
			e.buffer = realBuf
		} else {
			e.drawSignature()
			e.drawErrorLine()
			e.drawStatusBar()
		}
	}
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

	// Render key hints popup if visible
	if e.keyHintsPopup.IsVisible() {
		if widgetScreen, ok := e.screen.(widgets.Screen); ok {
			e.keyHintsPopup.Draw(widgetScreen)
		}
	}

	// Update cursor position
	if e.renderingState.IsOverlayActive() && e.renderingState.GetOverlay() != nil {
		// Get cursor from overlay
		x, y := e.renderingState.GetOverlay().GetCursorPosition()
		e.screen.ShowCursor(x, y)
	} else {
		// Normal cursor update
		// Command/Search modes are global; always place cursor on the real screen
		if e.mode == commands.CommandMode || e.mode == commands.SearchMode {
			e.updateCursor()
		} else {
			// Unified path: compute and show cursor relative to the active window region
			win := e.windows[e.activeWindow]
			if win.Buffer != nil {
				sub := newWindowScreen(e.screen, win.X, win.Y, win.Width, win.Height, false)
				realScreen := e.screen
				realBuffer := e.buffer
				e.screen = sub
				// Temporarily sync buffer to window state for correct cursor draw
				savedView := win.Buffer.ViewIterator
				savedCursor := win.Buffer.CursorIterator
				savedX := win.Buffer.CursorX
				win.Buffer.ViewIterator = buffer.NewBufferIteratorAtLine(win.Buffer, max(0, win.ViewLine))
				win.Buffer.CursorIterator = buffer.NewBufferIteratorAtLine(win.Buffer, max(0, win.CursorLine))
				win.Buffer.CursorX = win.CursorX
				e.buffer = win.Buffer
				e.updateCursor()
				// Restore
				win.Buffer.ViewIterator = savedView
				win.Buffer.CursorIterator = savedCursor
				win.Buffer.CursorX = savedX
				e.screen = realScreen
				e.buffer = realBuffer
			} else {
				e.updateCursor()
			}
		}
	}
	t0 = e.perfTracker.ReportSince("updateCursor", t0)

	e.screen.Show()
	e.perfTracker.ReportSince("screen.Show", t0)
}

// handleEvent processes a single event in normal (non-overlay) mode
func (e *Editor) handleEvent(ev events.Event) {
	switch ev := ev.(type) {
	case *events.PasteEvent:
		if ev.Start {
			e.inPasteMode = true
		} else {
			// End paste: exit paste mode and perform a single flush
			e.inPasteMode = false
			// Re-parse syntax highlighting once
			e.updateSyntaxHighlighting()
			// Notify LSP once
			e.notifyLSPFileChange()
			// Render once
			e.render()
		}
		return
	case *events.KeyEvent:
		// Terminal mode: forward all keys to terminal session; Esc switches to normal browse
		if e.mode == commands.TerminalMode {
			if ev.Key == events.KeyEscape {
				// Leave terminal input mode to normal browsing
				if e.buffer != nil {
					if ts := e.terminalSessions[e.buffer.FilePath]; ts != nil && ts.term != nil {
						ts.SavedCurX, ts.SavedCurY = ts.term.Cursor()
						// Align buffer cursor to emulator cursor location for browsing
						bannerOffset := 0
						if ts.Banner != "" {
							bannerOffset = 1
						}
						// Global row includes scrollback
						bufRow := len(ts.term.scrollback) + ts.SavedCurY + bannerOffset
						if bufRow < 0 {
							bufRow = 0
						}
						if bufRow >= e.buffer.LineCount {
							bufRow = e.buffer.LineCount - 1
						}
						it := buffer.NewBufferIteratorAtLine(e.buffer, bufRow)
						line := it.Line()
						bytePos := e.bytePositionFromVisualColumn(line, ts.SavedCurX)
						if bytePos < 0 {
							bytePos = 0
						}
						if bytePos > len(line) {
							bytePos = len(line)
						}
						e.buffer.CursorIterator = it
						e.buffer.CursorX = bytePos
					}
				}
				e.mode = commands.NormalMode
				return
			}
			// Forward key to active terminal session
			if e.buffer != nil {
				if ts := e.terminalSessions[e.buffer.FilePath]; ts != nil && ts.SessionID != "" {
					data := e.translateKeyToTerminal(ev)
					if len(data) > 0 {
						_ = e.proxyManager.WriteTerminal(ts.WorkingDir, ts.SessionID, data)
					}
				}
			}
			return
		}
		// Fast-path for paste mode: route runes/enter to the active input target
		if e.inPasteMode {
			if ev.Key == events.KeyRune {
				// If we're in a modal input (search/command), send characters there
				switch e.mode {
				case commands.SearchMode:
					e.searchLine.HandleKey(ev)
					return
				case commands.CommandMode:
					e.commandLine.HandleKey(ev)
					return
				default:
					// Editor text input
					if len(e.virtualCursors) > 0 {
						e.insertCharForAllCursors(ev.Rune)
					} else {
						e.insertChar(ev.Rune)
					}
					return
				}
			}
			if ev.Key == events.KeyEnter {
				// Delegate enter to current mode handler to preserve semantics
				switch e.mode {
				case commands.SearchMode:
					e.handleSearchMode(ev)
					return
				case commands.CommandMode:
					e.handleCommandMode(ev)
					return
				default:
					_ = e.insertNewline(nil)
					return
				}
			}
			// Allow Escape to cancel paste early or other control keys pass through
		}
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
			// Normal and Visual modes: Handle numeric prefix for count operations
			// Do not consume digits if a key sequence is pending (enables bindings like "space 1")
			if (e.mode == commands.NormalMode || e.mode == commands.VisualMode || e.mode == commands.VisualLineMode) && ev.Key == events.KeyRune && ev.Rune >= '0' && ev.Rune <= '9' && !e.keyMapper.HasPending() && !e.registerPrefixActive {
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
				// Always hide key hints popup when escape is pressed
				e.keyHintsPopup.Hide()

				// Clear any pending keys in the key mapper
				e.keyMapper.ResetPending()

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
					// When register prefix is active, handle register selection/yank target
					if e.registerPrefixActive {
						r := ev.Rune
						// First rune after '\'' is the register name
						e.pendingRegister = r
						e.registerReadyForAction = true
						// Do not forward this key to pending operator
						return
					}
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
						tMap := e.perfTracker.Start()
						command, found := e.keyMapper.ProcessKey(ev, e.mode)
						e.perfTracker.ReportSince("keymap.ProcessKey", tMap)
						if found {
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
							tExec := e.perfTracker.Start()
							if err := e.executeWithRepeatCount(command); err != nil {
								LL.Debug("Command execution error: %v", err)
							}
							e.perfTracker.ReportSince("executeWithRepeatCount", tExec)
						}
						return
					}
				}

				if keyStr != "" {
					if err := e.handlePendingOperation(keyStr); err != nil {
						LL.Debug("Pending operation error: %v", err)
					}
				}
			} else {
				tMap := e.perfTracker.Start()
				// If register prefix just typed a name, capture it BEFORE key mapping to avoid conflicts with bindings
				if e.mode == commands.NormalMode && ev.Key == events.KeyRune && e.registerPrefixActive && !e.registerReadyForAction {
					e.pendingRegister = ev.Rune
					e.registerReadyForAction = true
					return
				}

				// If a register has been selected ("'<reg>") and user presses an action key,
				// handle it before key mapping: 'y' -> target register for yank; 'p'/'P' -> paste from register
				if e.mode == commands.NormalMode && ev.Key == events.KeyRune && e.registerReadyForAction && e.pendingRegister != 0 {
					switch ev.Rune {
					case 'y':
						e.registerTargetForYank = e.pendingRegister
						e.registerPrefixActive = false
						e.registerReadyForAction = false
						e.pendingRegister = 0
					case 'p':
						// Paste-after from the specified register
						_ = e.commandRegistry.Execute("paste-after", nil)
						return
					case 'P':
						// Paste-before from the specified register
						_ = e.commandRegistry.Execute("paste-before", nil)
						return
					}
				}

				command, found := e.keyMapper.ProcessKey(ev, e.mode)
				e.perfTracker.ReportSince("keymap.ProcessKey", tMap)
				if found {
					LL.Debug("Key event: Key=%v, Rune=%q, Modifiers=%v", ev.Key, ev.Rune, ev.Modifiers)
					LL.Debug("Found command: %s", command)

					// Hide key hints popup since a command was found
					e.keyHintsPopup.Hide()

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
					tExec := e.perfTracker.Start()
					if err := e.executeWithRepeatCount(command); err != nil {
						LL.Debug("Command execution error: %v", err)
					}
					e.perfTracker.ReportSince("executeWithRepeatCount", tExec)

					// Clear repeat count after execution
					e.repeatCount = 0
				} else {
					// Handle key hints for normal mode only
					if e.mode == commands.NormalMode {
						// Check if we have pending keys and should show key hints
						if e.keyMapper.HasPending() {
							e.showKeyHints()
						} else {
							// Hide key hints if no pending keys (invalid sequence or no sequence)
							e.keyHintsPopup.Hide()
						}
					} else {
						// Hide key hints in non-normal modes
						e.keyHintsPopup.Hide()
					}
					LL.Debug("No command found for key %+v", ev)
					if e.mode == commands.InsertMode && ev.Key == events.KeyRune {
						// Handle insert char for multiple cursors
						if len(e.virtualCursors) > 0 {
							e.insertCharForAllCursors(ev.Rune)
						} else {
							e.insertChar(ev.Rune)
						}
					}
					// In normal mode, handle register prefix ''
					if e.mode == commands.NormalMode && ev.Key == events.KeyRune {
						if ev.Rune == '\'' {
							e.registerPrefixActive = true
							e.pendingRegister = 0
							e.registerReadyForAction = false
							return
						}
						// If register name was just typed without an operator, capture it
						// and wait for the next action key (y/p/P).
						if e.registerPrefixActive && !e.registerReadyForAction {
							e.pendingRegister = ev.Rune
							e.registerReadyForAction = true
							return
						}
					}
					// Clear count prefix if no command was found
					if e.mode == commands.NormalMode {
						e.countPrefix = ""
						e.repeatCount = 0

						// If we're in temporary normal mode and no keys are pending,
						// return to insert mode
						if e.temporaryNormal && !e.keyMapper.HasPending() {
							e.temporaryNormal = false
							e.switchToInsertMode()
						}
					}
				}
			}
		}
	case *events.ResizeEvent:
		e.handleResize()
	case *events.CustomEvent:
		LL.Debug("handleEvent: Received CustomEvent type=%s", ev.Type)
		switch ev.Type {
		case "terminal-output":
			// Data is map[string]interface{}{ "buffer": string, "data": string }
			if m, ok := ev.Data.(map[string]interface{}); ok {
				bufPath, _ := m["buffer"].(string)
				text, _ := m["data"].(string)
				if bufPath != "" {
					if b, ok := e.buffers[bufPath]; ok {
						// For terminal buffers with an emulator, replace screen content instead of appending
						wasRO := b.IsReadOnly
						if b.IsInternal && b.IsReadOnly {
							b.IsReadOnly = false
						}
						// Compose banner + emulator render if banner present
						textToShow := text
						if ts := e.terminalSessions[bufPath]; ts != nil && ts.Banner != "" {
							if textToShow == "" {
								textToShow = ts.Banner
							} else {
								textToShow = ts.Banner + "\n" + textToShow
							}
						}
						// Debug: log any occurrences of U+23CE (RETURN SYMBOL)
						if strings.ContainsRune(textToShow, '\u23CE') {
							LL.Debug("terminal.rx contains RETURN SYMBOL U+23CE")
						}
						// Replace entire buffer content with the composed snapshot
						if b.Length() > 0 {
							b.Delete(0, b.Length())
						}
						if textToShow != "" {
							b.Insert(0, textToShow)
						}
						if wasRO {
							b.IsReadOnly = true
						}
						// For terminal snapshots, align view to show the prompt row
						if e.buffer == b {
							if ts := e.terminalSessions[bufPath]; ts != nil && ts.term != nil {
								curX, curY := ts.term.Cursor()
								// Account for banner occupying the first buffer line
								bannerOffset := 0
								if ts.Banner != "" {
									bannerOffset = 1
								}
								// Place buffer view so that emulator grid bottom aligns with buffer bottom (scrollback included)
								// Cursor row within full content is scrollback + curY
								bufRow := len(ts.term.scrollback) + curY + bannerOffset
								// If session was closed, move cursor to last non-empty line (exit banner line)
								if ts.Closed {
									bufRow = b.LineCount - 1
									// Walk back over any trailing empty lines to land on the banner/content
									for bufRow > 0 {
										it := buffer.NewBufferIteratorAtLine(b, bufRow)
										line := it.Line()
										if len(line) > 0 {
											break
										}
										bufRow--
									}
									curX = 0
								}
								if bufRow < 0 {
									bufRow = 0
								}
								if bufRow >= b.LineCount {
									bufRow = b.LineCount - 1
								}
								// Ensure iterators are valid after content replacement
								// Auto-follow: keep cursor visible near bottom
								visible := e.getVisibleHeight()
								if visible < 1 {
									visible = 1
								}
								top := bufRow - (visible - 1)
								if top < 0 {
									top = 0
								}
								if top >= b.LineCount {
									top = b.LineCount - 1
								}
								b.ViewIterator = buffer.NewBufferIteratorAtLine(b, top)
								b.CursorIterator = buffer.NewBufferIteratorAtLine(b, bufRow)
								// Map terminal column to byte offset within the buffer line (UTF-8 aware)
								line := b.CursorIterator.Line()
								bytePos := e.bytePositionFromVisualColumn(line, curX)
								if bytePos < 0 {
									bytePos = 0
								}
								if bytePos > len(line) {
									bytePos = len(line)
								}
								b.CursorX = bytePos
							} else {
								b.ViewIterator = buffer.NewBufferIteratorAtLine(b, 0)
								b.CursorIterator = buffer.NewBufferIteratorAtLine(b, 0)
							}
						}
						e.render()
					}
				}
			}
		case "terminal-closed":
			// Leave terminal mode when the session closes
			if e.mode == commands.TerminalMode {
				e.mode = commands.NormalMode
			}
			e.render()
			return
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

// sanitizeTerminalText converts common terminal control sequences into a text-only form
// suitable for our MVP terminal buffer rendering. It:
// - Converts CRLF to LF
// - Converts lone CR to LF
// - Strips simple ANSI escape sequences (CSI ... letter)
// - Leaves printable characters, tabs, and newlines intact
func sanitizeTerminalText(input string) string {
	// Fast path: if no ESC or CR or BEL, return as-is
	if !strings.ContainsAny(input, "\x1b\r\a") {
		return input
	}
	// Normalize newlines
	s := strings.ReplaceAll(input, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	var b strings.Builder
	b.Grow(len(s))

	i := 0
	for i < len(s) {
		c := s[i]
		// Drop all control chars except tab/newline
		if c < 0x20 || c == 0x7f {
			if c == '\n' || c == '\t' {
				b.WriteByte(c)
			}
			i++
			continue
		}

		if c == 0x1b { // ESC
			// Look ahead to classify sequence
			if i+1 >= len(s) {
				// Trailing ESC, drop
				i++
				continue
			}
			next := s[i+1]

			switch next {
			case '[': // CSI: ESC [ ... final 0x40..0x7E
				i += 2
				for i < len(s) {
					fc := s[i]
					i++
					if fc >= 0x40 && fc <= 0x7e {
						break
					}
				}
				continue
			case ']': // OSC: ESC ] ... BEL (0x07) or ST (ESC \\)
				i += 2
				for i < len(s) {
					if s[i] == 0x07 { // BEL terminator
						i++
						break
					}
					if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' { // ST terminator
						i += 2
						break
					}
					i++
				}
				continue
			case 'P', 'X', '^', '_': // DCS/SOS/PM/APC: ESC <type> ... ST (ESC \\)
				i += 2
				for i < len(s) {
					if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
				continue
			case '(', ')', '*', '+', '-', '.', '/':
				// Character set designation: ESC ( X or ESC ) X etc -> skip selector and one byte
				i += 3
				if i > len(s) {
					i = len(s)
				}
				continue
			default:
				// Other single-char escapes (e.g., ESC c, ESC 7/8, etc.). Skip ESC and the next byte.
				i += 2
				continue
			}
		}

		// Printable
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// translateKeyToTerminal converts a key event to terminal bytes (UTF-8 / ANSI)
// This function handles the translation of keyboard input into the appropriate
// byte sequences that terminals expect, including control characters like Ctrl+C
func (e *Editor) translateKeyToTerminal(ev *events.KeyEvent) []byte {
	// Basic mapping: runes, Enter, Backspace, Tab, arrows
	switch ev.Key {
	case events.KeyRune:
		if ev.Rune == 0 {
			return nil
		}
		buf := make([]byte, 4)
		n := utf8.EncodeRune(buf, ev.Rune)
		return buf[:n]
	case events.KeyEnter:
		LL.Trace("terminal.tx key=Enter -> CR (0x0d)")
		return []byte{'\r'}
	case events.KeyBackspace, events.KeyBackspace2:
		return []byte{'\x7f'}
	case events.KeyTab:
		return []byte{'\t'}
	// Common control keys to forward raw to the shell
	case events.KeyCtrlC:
		return []byte{0x03} // ETX (interrupt signal)
	case events.KeyCtrlD:
		return []byte{0x04} // EOT
	case events.KeyCtrlL:
		return []byte{0x0c} // FF (clear-screen in many shells)
	case events.KeyCtrlN:
		return []byte{0x0e} // SO (next-history/forward in readline/fish bindings)
	case events.KeyHome:
		return []byte{0x1b, '[', 'H'}
	case events.KeyEnd:
		return []byte{0x1b, '[', 'F'}
	case events.KeyUp:
		return []byte{'\x1b', '[', 'A'}
	case events.KeyDown:
		return []byte{'\x1b', '[', 'B'}
	case events.KeyRight:
		return []byte{'\x1b', '[', 'C'}
	case events.KeyLeft:
		return []byte{'\x1b', '[', 'D'}
	default:
		return nil
	}
}

// executeWithRepeatCount executes a command with the current repeat count
func (e *Editor) executeWithRepeatCount(command string) error {
	t0 := e.perfTracker.Start()
	count := e.repeatCount
	if count <= 0 {
		count = 1
	}

	// Store whether we're in temporary normal mode before executing
	wasTemporaryNormal := e.temporaryNormal

	// Commands that need special handling for count
	switch command {
	case "yank-line":
		// For yy, count means "yank N lines" not "yank same line N times"
		e.lastCommandCount = count
		return e.yankMultipleLines(count)
	case "indent-selection":
		// In visual modes, apply indent by count levels
		if e.mode == commands.VisualMode || e.mode == commands.VisualLineMode {
			return e.indentSelectionWithCount(count)
		}
		fallthrough
	case "start-delete", "start-change", "start-yank":
		// For dd/cc/yy, count means "delete/change/yank N lines" not "repeat operation N times"
		// Store the count for the pending operation
		e.pendingCount = count
		e.lastCommandCount = count
		return e.executeCommandForAllCursors(command, nil)
	case "unindent-selection":
		// In visual modes, apply unindent by count levels
		if e.mode == commands.VisualMode || e.mode == commands.VisualLineMode {
			return e.unindentSelectionWithCount(count)
		}
		// Otherwise fall through to normal repetition handling
		fallthrough
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

		// After executing a command in temporary normal mode, return to insert mode
		// unless the command itself changes the mode
		if wasTemporaryNormal && e.mode == commands.NormalMode {
			LL.Debug("executeWithRepeatCount: Returning to insert mode after temporary normal command")
			e.temporaryNormal = false
			e.switchToInsertMode()
		}

		// Store the count for repeat command only for text-modifying commands
		// Movement commands should not overwrite the repeat count
		if e.isTextModifyingCommand(command) {
			e.lastCommandCount = count
		}

		e.perfTracker.ReportSince("command.executeWithRepeatCount", t0)
		return nil
	}
}

// isTextModifyingCommand returns true if the command modifies text content
func (e *Editor) isTextModifyingCommand(command string) bool {
	// Commands that modify text and should store count for repeat
	textModifyingCommands := map[string]bool{
		"delete-current-character": true,
		"delete-to-end-of-line":    true,
		"delete-character-insert":  true,
		"delete-word-forward":      true,
		"delete-word-backward":     true,
		"indent-line":              true,
		"unindent-line":            true,
		"reindent-line":            true,
		"yank-line":                true,
		"insert-tab":               true,
		"enter-insert-mode":        true,
		"append-after-cursor":      true,
		"append-at-end-of-line":    true,
		"start-delete":             true,
		"start-change":             true,
		"start-yank":               true,
	}

	return textModifyingCommands[command]
}

// executeCommandForAllCursors executes a command for all cursors
func (e *Editor) executeCommandForAllCursors(command string, args []string) error {
	// Check if we have a buffer to work with
	if e.buffer == nil {
		return nil // No buffer, no cursors to execute command on
	}

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
	if e.mode == commands.InsertMode || e.mode == commands.TerminalMode {
		// Use a bar cursor in insert and terminal modes
		e.screen.SetCursorStyle(ui.CursorStyleSteadyBar)
	} else {
		// Use a block cursor in other modes
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
	} else if e.buffer == nil {
		// Empty window - hide the cursor or show it at top-left
		e.screen.ShowCursor(0, e.getContentStartY())
	} else {
		// Terminal/internal buffers: position cursor directly from emulator state
		if e.buffer.IsInternal {
			if ts := e.terminalSessions[e.buffer.FilePath]; ts != nil && ts.term != nil {
				if e.mode == commands.TerminalMode {
					// In terminal mode, show at terminal cursor, accounting for scrollback and view top
					curX, curY := ts.term.Cursor()
					bannerOffset := 0
					if ts.Banner != "" {
						bannerOffset = 1
					}
					globalRow := bannerOffset + len(ts.term.scrollback) + curY
					viewTop := e.buffer.ViewIterator.LineNumber()
					relRow := globalRow - viewTop
					// Clamp to visible content area
					contentTop := e.getContentStartY()
					visible := e.getVisibleHeight()
					if relRow < 0 {
						relRow = 0
					}
					if relRow >= visible {
						relRow = visible - 1
					}
					screenY := contentTop + relRow
					if ts.Closed {
						// If closed, prefer to place cursor on last non-empty visible line
						row := e.buffer.LineCount - 1
						for row > 0 {
							it := buffer.NewBufferIteratorAtLine(e.buffer, row)
							line := it.Line()
							if len(line) > 0 {
								break
							}
							row--
						}
						relRow = row - viewTop
						if relRow < 0 {
							relRow = 0
						}
						if relRow >= visible {
							relRow = visible - 1
						}
						screenY = contentTop + relRow
						curX = 0
					}
					screenX := e.buffer.LineNumberWidth + curX
					e.screen.ShowCursor(screenX, screenY)
					return
				}
				// In normal mode on terminal buffer: position cursor using regular buffer logic
			}
		}
		// Show cursor in editor (non-terminal)
		visualLine, visualCol := e.GetCursorVisualPosition()
		cursorY := visualLine + e.getContentStartY()
		LL.Trace("updateCursor: logical(line=%d,x=%d) viewTop=%d -> visual(line=%d,col=%d) cursorY=%d",
			e.buffer.CursorIterator.LineNumber(), e.buffer.CursorX, e.buffer.ViewIterator.LineNumber(), visualLine, visualCol, cursorY)
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

	// Stop the auto-save manager
	if e.autoSaveManager != nil {
		e.autoSaveManager.Stop()
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

// bytePositionFromVisualColumn converts a visual column (cell) to a byte index in the line.
// Accounts for tabs and wide runes (emoji/CJK). If visualCol is beyond the line, returns len(line).
func (e *Editor) bytePositionFromVisualColumn(line string, visualCol int) int {
	if visualCol <= 0 {
		return 0
	}
	current := 0
	for i, r := range line {
		var w int
		if r == '\t' {
			w = 4
		} else {
			w = runeDisplayWidth(r)
			if w < 1 {
				w = 1
			}
		}
		if current+w > visualCol {
			// Desired column falls within this rune's cell span; place cursor at this rune
			return i
		}
		current += w
	}
	// If we matched exactly the end of line width, place at EOL
	return len(line)
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
    if e.buffer == nil || e.buffer.Highlighter == nil {
        return nil
    }
    return e.buffer.Highlighter.GetTokensForLine(iter.Position(), iter.LineEnd())
}

// updateCursorState updates any state that depends on cursor position
func (e *Editor) updateCursorState() {
	// Check if we need to update signature help
	e.checkSignatureUpdate()
}

// actuallyQuit exits the editor completely
func (e *Editor) actuallyQuit(args []string) error {
	// Clean up auto-save files for all open buffers since user is quitting without saving
	if e.autoSaveManager != nil {
		for filePath := range e.buffers {
			if filePath != "" { // Skip unnamed buffers
				e.autoSaveManager.CleanupAutoSave(filePath)
			}
		}
	}

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
		if endLine < e.buffer.LineCount-1 {
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
                // Force context update for active buffer
                buf.ContextLine = -1
            }
		}
	}()
}

// requestDiagnosticsRefresh polls the proxy for diagnostics for a file and updates the buffer
// This is used to bridge the gap between didOpen/didChange and when publishDiagnostics arrive
func (e *Editor) requestDiagnosticsRefresh(filePath string) {
	if filePath == "" {
		return
	}

	// If the current buffer has LSP, send a didChange; rely on async diagnostics publish
	buf, exists := e.buffers[filePath]
	if !exists || !buf.HasFile() {
		return
	}

	content := buf.Rope.String()
	go func(target *buffer.Buffer) {
		LL.Debug("requestDiagnosticsRefresh: starting for %s", filePath)
		if err := e.proxyManager.NotifyFileChange(filePath, content); err != nil {
			LL.Debug("requestDiagnosticsRefresh: notify failed for %s: %v", filePath, err)
			return
		}
		LL.Debug("requestDiagnosticsRefresh: didChange nudge sent for %s", filePath)
		// No polling here; diagnostics will arrive via proxy's publish cache on next GetFile
	}(buf)
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
	e.switchToInsertMode()
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
	e.switchToInsertMode()
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession("A")
	}
	return nil
}

// setLastChange sets the last change and clears any lastCommand
func (e *Editor) setLastChange(entry buffer.HistoryEntry) {
	e.lastChange = entry

	// Mark buffer as dirty for auto-save whenever content changes
	e.markBufferDirty()

	// Update search matches after any text modification to keep positions current
	if e.searchPattern != "" {
		e.updateSearchMatches()
	}

	// Only clear lastCommand if it's not a change operator
	// Change operators need both lastCommand and lastChange for proper repeat
	if e.lastCommand != nil && e.lastCommand.Operator.GetType() == "change" {
		// Keep lastCommand for change operators
		return
	}
	e.lastCommand = nil // Clear operator+text object command for other cases
	// Clear simple command unless it was just set (indent/unindent/reindent set it right before calling this)
	if e.lastSimpleCommand != "indent-line" && e.lastSimpleCommand != "unindent-line" && e.lastSimpleCommand != "reindent-line" {
		e.lastSimpleCommand = ""
	}
}

// Undo/Redo commands
func (e *Editor) undo(args []string) error {
	if e.buffer == nil || e.buffer.History == nil {
		return nil
	}

	// Remember current position; also attempt to restore multi-cursor snapshot
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

	// If the redo stack top is a composite with a cursor snapshot, restore it
	if e.buffer.History != nil {
		if primary, virtual, ok := e.buffer.History.GetTopRedoCompositeCursors(); ok {
			// Restore primary cursor
			if primary.Line >= 0 && primary.Line < e.buffer.LineCount {
				e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primary.Line)
				e.buffer.CursorX = primary.Char
			}
			// Restore virtual cursors
			restored := make([]VirtualCursor, 0, len(virtual))
			for _, c := range virtual {
				if c.Line >= 0 && c.Line < e.buffer.LineCount {
					iter := buffer.NewBufferIteratorAtLine(e.buffer, c.Line)
					line := iter.Line()
					x := c.Char
					if x < 0 {
						x = 0
					}
					if x > len(line) {
						x = len(line)
					}
					restored = append(restored, VirtualCursor{X: x, Y: c.Line})
				}
			}
			e.virtualCursors = restored
		}
	}

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

	// If the undo stack top is a composite with a post-cursor snapshot, restore it
	if e.buffer.History != nil {
		if primary, virtual, ok := e.buffer.History.GetTopUndoCompositePostCursors(); ok {
			if primary.Line >= 0 && primary.Line < e.buffer.LineCount {
				e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, primary.Line)
				e.buffer.CursorX = primary.Char
			}
			restored := make([]VirtualCursor, 0, len(virtual))
			for _, c := range virtual {
				if c.Line >= 0 && c.Line < e.buffer.LineCount {
					iter := buffer.NewBufferIteratorAtLine(e.buffer, c.Line)
					line := iter.Line()
					x := c.Char
					if x < 0 {
						x = 0
					}
					if x > len(line) {
						x = len(line)
					}
					restored = append(restored, VirtualCursor{X: x, Y: c.Line})
				}
			}
			e.virtualCursors = restored
		}
	}

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

	for i := 0; i <= endLine-startLine; i++ {
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
	// Get the count to use for repetition
	count := e.lastCommandCount
	if count <= 0 {
		count = 1
	}

	// First check if we have a simple command to repeat (like indent-line)
	if e.lastSimpleCommand != "" {
		// Execute the simple command with the original count
		for i := 0; i < count; i++ {
			if err := e.commandRegistry.Execute(e.lastSimpleCommand, []string{}); err != nil {
				return err
			}
		}
		return nil
	}

	// Then check if we have a repeatable command (operator + text object)
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

		// For non-change operators, repeat the operator+text object with the original count
		for i := 0; i < count; i++ {
			if err := e.executeOperatorWithTextObject(e.lastCommand.Operator, e.lastCommand.TextObject); err != nil {
				return err
			}
		}
		return nil
	}

	// Otherwise fall back to repeating the last change
	if e.lastChange == nil {
		return nil // Nothing to repeat
	}

	if e.buffer == nil || e.buffer.History == nil {
		return nil
	}

	// Repeat the change for the original count
	for repeatIdx := 0; repeatIdx < count; repeatIdx++ {
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
	}

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

// notifyLSPFileChange sends a didChange notification to the LSP server if applicable
func (e *Editor) notifyLSPFileChange() {
	// Only send notification if buffer has LSP enabled and has a file path
	if !e.buffer.HasLSP || e.buffer.FilePath == "" || e.buffer.IsInternal {
		return
	}

	// Get the full content of the buffer
	content := e.buffer.Rope.String()

	// Send the notification asynchronously to avoid blocking
	go func() {
		if err := e.proxyManager.NotifyFileChange(e.buffer.FilePath, content); err != nil {
			LL.Debug("Failed to notify LSP of file change: %v", err)
		} else {
			LL.Debug("notifyLSPFileChange: didChange sent for %s", e.buffer.FilePath)
		}
	}()
}

func (e *Editor) goToDefinition(args []string) error {
	// Only works if we have a file open
	if e.buffer == nil || !e.buffer.HasFile() {
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

// goToDefinitionNewWindow opens the definition in a new right-hand window
func (e *Editor) goToDefinitionNewWindow(args []string) error {
	// Use the same lookup as goToDefinition, but open in a new window if target differs
	if !e.buffer.HasFile() {
		e.errorManager.ShowError("No file open")
		return fmt.Errorf("no file open")
	}
	line := e.buffer.CursorIterator.LineNumber()
	character := e.buffer.CursorX
	locations, err := e.proxyManager.GoToDefinition(e.buffer.FilePath, line, character)
	if err != nil || len(locations) == 0 {
		if err == nil {
			e.errorManager.ShowError("No definition found")
		} else {
			e.errorManager.ShowError("LSP error: %v", err)
		}
		return err
	}
	loc := locations[0]
	if !strings.HasPrefix(loc.URI, "file://") {
		return fmt.Errorf("unsupported URI scheme: %s", loc.URI)
	}
	targetPath := strings.TrimPrefix(loc.URI, "file://")
	if strings.Contains(e.buffer.FilePath, ":") {
		parts := strings.SplitN(e.buffer.FilePath, ":", 2)
		if len(parts) == 2 {
			targetPath = parts[0] + ":" + targetPath
		}
	}

	// Create a new window to the right and open the file there
	insertAt := e.activeWindow + 1
	if insertAt >= len(e.windows) {
		e.windows = append(e.windows, newWindow(e.buffer))
	} else {
		e.windows = append(e.windows[:insertAt+1], e.windows[insertAt:]...)
		e.windows[insertAt] = newWindow(e.buffer)
	}
	e.activateWindow(insertAt)

	// Open file and jump to location
	e.openFile(targetPath)
	e.buffer.CursorX = loc.Range.Start.Character
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, loc.Range.Start.Line)
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
	e.startAsyncOperation(operationID, "resolving file path: "+path)
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

// cleanupAllAutoIndentLines cleans up all tracked auto-indented lines for the current buffer
// This should be called when exiting insert mode to clean up all empty auto-indented lines
func (e *Editor) cleanupAllAutoIndentLines() {
	if e.buffer == nil {
		return
	}

	currentBufferPath := e.buffer.FilePath
	keysToCleanup := make([]string, 0)

	// Collect all keys for the current buffer
	for key, info := range e.autoIndentLines {
		// Check if this key belongs to the current buffer
		if keyBufferPath := e.extractBufferPathFromKey(key); keyBufferPath == currentBufferPath {
			// Only cleanup lines that don't have user content
			if !info.HasUserContent {
				keysToCleanup = append(keysToCleanup, key)
			}
		}
	}

	// Clean up all the identified lines
	for _, key := range keysToCleanup {
		if info, exists := e.autoIndentLines[key]; exists {
			e.cleanupAutoIndentOnLine(info.LineNumber)
		}
	}
}

// extractBufferPathFromKey extracts the buffer path from an auto-indent key
func (e *Editor) extractBufferPathFromKey(key string) string {
	// Key format is: bufferPath + ":" + lineNumber
	// Find the last colon to separate buffer path from line number
	lastColon := strings.LastIndex(key, ":")
	if lastColon == -1 {
		return key // Fallback if format is unexpected
	}
	return key[:lastColon]
}

// makeAutoIndentKey creates a unique key for tracking auto-indent on a specific line
func (e *Editor) makeAutoIndentKey(bufferPath string, lineNumber int) string {
	return fmt.Sprintf("%s:%d", bufferPath, lineNumber)
}

// trackAutoIndent records that auto-indent was applied to a line
func (e *Editor) trackAutoIndent(lineNumber int, autoIndentLength int) {
	key := e.makeAutoIndentKey(e.buffer.FilePath, lineNumber)
	e.autoIndentLines[key] = AutoIndentInfo{
		LineNumber:       lineNumber,
		AutoIndentLength: autoIndentLength,
		HasUserContent:   false,
		BufferGeneration: e.buffer.Generation, // Track buffer generation for invalidation
	}
}

// markLineHasUserContent marks that the user has typed non-whitespace content on a line
func (e *Editor) markLineHasUserContent(lineNumber int) {
	key := e.makeAutoIndentKey(e.buffer.FilePath, lineNumber)
	if info, exists := e.autoIndentLines[key]; exists {
		info.HasUserContent = true
		e.autoIndentLines[key] = info
	}
}

// cleanupAutoIndentOnLine removes trailing auto-indent whitespace from a line if appropriate
func (e *Editor) cleanupAutoIndentOnLine(lineNumber int) {
	key := e.makeAutoIndentKey(e.buffer.FilePath, lineNumber)
	info, exists := e.autoIndentLines[key]
	if !exists || info.HasUserContent {
		return // Nothing to clean up
	}

	// Note: Do not invalidate cleanup based on buffer generation differences.
	// Even if there have been many edits during the insert session, we can
	// safely inspect the current line content and remove auto-inserted
	// indentation when the line is whitespace-only. This ensures trailing
	// spaces on empty auto-indented lines are cleaned up reliably on exit
	// from insert mode.

	// Get the line content
	iter := buffer.NewBufferIteratorAtLine(e.buffer, lineNumber)
	if iter.IsEOF() {
		delete(e.autoIndentLines, key)
		return
	}

	line := iter.Line()

	// Check if line has only whitespace up to AutoIndentLength
	if len(line) <= info.AutoIndentLength {
		// Check if the line contains only whitespace
		hasOnlyWhitespace := true
		for _, char := range line {
			if char != ' ' && char != '\t' {
				hasOnlyWhitespace = false
				break
			}
		}

		if hasOnlyWhitespace {
			// Save current positions before buffer modification
			// Use safe access to avoid iterator generation mismatch
			var cursorLine, viewLine int
			var cursorX int

			// Safely get cursor line, defaulting to 0 if iterator is invalid
			if e.buffer.CursorIterator != nil {
				if e.buffer.CursorIterator.Generation == e.buffer.Generation {
					cursorLine = e.buffer.CursorIterator.LineNumber()
				} else {
					cursorLine = 0 // Default fallback
				}
			}

			// Safely get view line, defaulting to 0 if iterator is invalid
			if e.buffer.ViewIterator != nil {
				if e.buffer.ViewIterator.Generation == e.buffer.Generation {
					viewLine = e.buffer.ViewIterator.LineNumber()
				} else {
					viewLine = 0 // Default fallback
				}
			}

			cursorX = e.buffer.CursorX

			// Remove the auto-indented whitespace
			deleteEntry := buffer.NewDeleteTextEntry(lineNumber, 0, line)
			if e.buffer.History != nil {
				if executed, err := e.buffer.History.Execute(e.buffer, deleteEntry); err == nil && executed != nil {
					e.setLastChange(executed)
				}
			} else {
				deleteEntry.Execute(e.buffer)
			}

			// Refresh iterators after buffer modification
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, cursorLine)
			e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLine)
			e.buffer.CursorX = cursorX
		}
	}

	// Remove from tracking
	delete(e.autoIndentLines, key)
}

// cleanupAutoIndentOnCursorMove should be called when cursor moves to a different line
func (e *Editor) cleanupAutoIndentOnCursorMove(oldLineNumber int) {
	if oldLineNumber >= 0 && oldLineNumber != e.buffer.CursorIterator.LineNumber() {
		e.cleanupAutoIndentOnLine(oldLineNumber)
	}
}

// checkAndCleanupAutoIndent should be called after any cursor movement to handle auto-indent cleanup
func (e *Editor) checkAndCleanupAutoIndent() {
	currentLine := e.buffer.CursorIterator.LineNumber()
	if e.lastCursorLine != -1 && e.lastCursorLine != currentLine {
		e.cleanupAutoIndentOnLine(e.lastCursorLine)
	}
	e.lastCursorLine = currentLine
}
