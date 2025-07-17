package editor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/log"
	"veda/internal/perf"
	"veda/internal/proxy"
	"veda/internal/syntax"
	"veda/internal/ui"
	"veda/internal/widgets"

	sitter "github.com/smacker/go-tree-sitter"
)

var LL *log.Logger = log.NewLogger("EDITOR")

// Editor represents the state of the text editor.
type Editor struct {
	screen          Screen
	buffer          *buffer.Buffer          // Current buffer
	config          *config.Config
	mode            commands.Mode
	commandLine     *widgets.LineEditor
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
	asyncEvents     chan ui.Event           // Channel for async events

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
	commandLine := widgets.NewLineEditor(0, statusBarY, width)
	commandLine.SetStyle(config.Theme.GetStyle(config.Theme.CommandLine))

	// Start proxy client
	proxyClient, err := proxy.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to start proxy client: %v", err)
	}

	// Create initial empty buffer
	initialBuffer := buffer.NewBuffer("", "", config.DefaultIndentType, config.DefaultIndentWidth)


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
		asyncEvents:     make(chan ui.Event, 100), // Buffered channel for async events
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
		var ev ui.Event
		
		// Use select to handle multiple event sources
		select {
		case ev = <-screenEvents:
			// Got a screen event
		case <-ticker.C:
			// Timer fired - ev remains nil, will trigger redraw
		case ev = <-e.asyncEvents:
			// Got an async event from somewhere else
		}

		// Start performance tracking for this iteration
		t0 := e.perfTracker.Start()

		// Only handle event if we got one
		if ev != nil {
			switch ev := ev.(type) {
			case *ui.EventKey:
				if e.mode == commands.CommandMode {
					// Command mode is special - it's a line editor
					e.handleCommandMode(ev)
					t0 = e.perfTracker.ReportSince("handleCommandMode", t0)
				} else {
					// Normal and Insert modes both use key mapping
					if command, found := e.keyMapper.ProcessKey(ev, e.mode); found {
						err := e.commandRegistry.Execute(command)
						t0 = e.perfTracker.ReportSince("command:"+command, t0)

						if err != nil {
							// TODO: Show error to user
						}
					} else if e.mode == commands.InsertMode && ev.Key == ui.KeyRune {
						// In insert mode, insert unmapped rune keys
						e.insertChar(ev.Rune)
						t0 = e.perfTracker.ReportSince("insertChar", t0)
					}
				}
			case *ui.EventResize:
				// Handle screen resize events
				e.handleResize()
				t0 = e.perfTracker.ReportSince("handleResize", t0)
			}
		}

		// Check if scrolling occurred
		e.scroll()
		t0 = e.perfTracker.ReportSince("scroll", t0)

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




func (e *Editor) handleCommandMode(ev *ui.EventKey) {
	switch ev.Key {
	case ui.KeyEscape:
		// Exit command mode
		e.exitCommandMode()
	case ui.KeyEnter:
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
}

// exitCommandMode switches back to normal mode
func (e *Editor) exitCommandMode() {
	e.mode = commands.NormalMode
	e.commandLine.SetText("")
}

// executeCommand processes the command entered in the command line
func (e *Editor) executeCommand() {
	command := e.commandLine.GetText()
	// Remove the leading ":"
	if len(command) > 1 {
		command = command[1:]
	}

	// First try to execute from the command registry
	if e.commandRegistry.Has(command) {
		err := e.commandRegistry.Execute(command)
		if err != nil {
			LL.Debug("Command execution failed: %v", err)
		}
		return
	}

	// Handle set commands
	if strings.HasPrefix(command, "set ") {
		e.handleSetCommand(command[4:])
		return
	}

	// Handle basic commands that aren't in the registry
	switch command {
	case "q", "quit":
		e.Close()
		os.Exit(0)
	}
}

// handleSetCommand processes :set commands
func (e *Editor) handleSetCommand(args string) {
	// Handle formats like "tabstop=4" and "tabstop 4"
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return
	}

	// Check if first part contains '='
	if strings.Contains(parts[0], "=") {
		eqParts := strings.SplitN(parts[0], "=", 2)
		if len(eqParts) == 2 {
			parts = []string{eqParts[0], eqParts[1]}
		}
	}

	switch parts[0] {
	case "expandtab":
		// Use spaces instead of tabs
		e.buffer.SetIndentationType(config.IndentSpaces)
	case "noexpandtab":
		// Use tabs instead of spaces
		e.buffer.SetIndentationType(config.IndentTabs)
	case "tabstop", "ts":
		// Set tab width
		if len(parts) >= 2 {
			if width, err := strconv.Atoi(parts[1]); err == nil && width > 0 && width <= 8 {
				e.buffer.SetIndentationWidth(width)
			}
		}
	case "shiftwidth", "sw":
		// Set indent width (same as tabstop for now)
		if len(parts) >= 2 {
			if width, err := strconv.Atoi(parts[1]); err == nil && width > 0 && width <= 8 {
				e.buffer.SetIndentationWidth(width)
			}
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
	// Update argument list state if we have syntax highlighting
	if e.highlighter != nil && e.highlighter.GetTree() != nil {
		e.updateArgumentListState()
	}
}

func (e *Editor) quit() error {
	e.Close()
	os.Exit(0)
	return nil
}


// Insert mode commands
func (e *Editor) enterInsertMode() error {
	e.mode = commands.InsertMode
	return nil
}

func (e *Editor) exitInsertMode() error {
	e.mode = commands.NormalMode
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

	// Get absolute position at end of line
	pos := e.getAbsolutePosition()

	// Insert newline and indentation (this will invalidate iterators)
	e.buffer.Insert(pos, "\n"+autoIndent.IndentStr)

	// Recreate iterators - cursor moves to next line
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum + 1)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Move cursor to end of indentation
	e.buffer.CursorX = len(autoIndent.IndentStr)

	// Enter insert mode
	e.mode = commands.InsertMode

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

	// Get absolute position
	pos := e.getAbsolutePosition()

	// Insert newline before current position (this will invalidate iterators)
	// Move the cursor up and add indentation
	e.buffer.Insert(pos, "\n")

	// The cursor stays at the same line number (which is now the new empty line)
	// because we inserted before the current position
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
	e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

	// Insert indentation at the beginning of the new line
	if autoIndent.IndentStr != "" {
		pos = e.getAbsolutePosition()
		e.buffer.Insert(pos, autoIndent.IndentStr)

		// Recreate iterators after insertion
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)
	}

	// Move cursor to end of indentation
	e.buffer.CursorX = len(autoIndent.IndentStr)

	// Enter insert mode
	e.mode = commands.InsertMode

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

	// Calculate the absolute position in the buffer
	pos := e.getAbsolutePosition()

	if isPaired {
		// Insert both the opening and closing characters
		e.buffer.Insert(pos, string(ch) + string(pairedChar))

		// Recreate iterators at the same line
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

		// Move cursor between the pair (only forward by 1)
		e.buffer.CursorX = cursorX + 1
	} else {
		// Normal character insert
		e.buffer.Insert(pos, string(ch))

		// Recreate iterators at the same line
		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

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

	// Get the current line for auto-indent calculation
	currentLine := ""
	if !e.buffer.CursorIterator.IsEOF() {
		currentLine = e.buffer.CursorIterator.Line()
	}

	// Calculate auto-indent for the new line
	autoIndent := e.buffer.CalculateAutoIndent(currentLine, e.buffer.CursorX)

	// Get the absolute position
	pos := e.getAbsolutePosition()

	// Insert newline and indentation
	e.buffer.Insert(pos, "\n"+autoIndent.IndentStr)

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
	pos := e.getAbsolutePosition()

	if e.buffer.CursorX == 0 {
		// At beginning of line, join with previous line
		if pos > 0 {
			// Store the length of the previous line before joining
			prevIter := e.buffer.CursorIterator.Previous()
			prevLineLen := 0
			if !prevIter.Equals(e.buffer.CursorIterator) {
				prevLineLen = len(prevIter.Line())
			}

			// Delete the newline character before current position
			e.buffer.Delete(pos-1, pos)

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
			// Delete one character before cursor position
			e.buffer.Delete(pos-1, pos)

			// Recreate iterators at the same line
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
			e.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(e.buffer, viewLineNum)

			// Move cursor back one position
			e.buffer.CursorX--

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

	// Get current line to check bounds
	line := e.buffer.CursorIterator.Line()

	// If cursor is past the end of the line, nothing to delete
	if e.buffer.CursorX >= len(line) {
		return nil
	}

	// Remember current position
	lineNum := e.buffer.CursorIterator.LineNumber()
	viewLineNum := e.buffer.ViewIterator.LineNumber()

	// Get absolute position of cursor
	pos := e.getAbsolutePosition()

	// Delete one character at cursor position
	e.buffer.Delete(pos, pos+1)

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

	// Get folding ranges from LSP
	ranges, err := e.proxyClient.GetFoldingRanges(filePath)
	if err != nil || len(ranges) == 0 {
		e.contextCache = nil
		// Mark LSP as not available if we got an error
		if err != nil {
			e.buffer.HasLSP = false
		}
		return
	}

	// If we got here, LSP is working
	e.buffer.HasLSP = true

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
		content := e.buffer.String()
		// Force a full reparse by not reusing the old tree
		e.highlighter.ParseFull([]byte(content))
		// Update argument list state after parsing
		e.updateArgumentListState()
	}
}

// findNodeAtPosition finds the most specific node containing the given position
func (e *Editor) findNodeAtPosition(node *sitter.Node, pos int) *sitter.Node {
	// Check if position is within this node
	if int(node.StartByte()) <= pos && pos <= int(node.EndByte()) {
		// Check children first for more specific match
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if result := e.findNodeAtPosition(child, pos); result != nil {
				return result
			}
		}
		return node
	}
	return nil
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
