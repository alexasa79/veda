package editor

import (
	"fmt"
	"os"
	"strings"
	"vidya/internal/buffer"
	"vidya/internal/commands"
	"vidya/internal/config"
	"vidya/internal/proxy"
	"vidya/internal/syntax"
	"vidya/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

// Editor represents the state of the text editor.
type Editor struct {
	screen          tcell.Screen
	buffer          *buffer.Buffer
	config          *config.Config
	cursorX         int
	cursorY         int
	topLine         int
	mode            commands.Mode
	commandLine     *widgets.LineEditor
	proxyClient     *proxy.Client
	commandRegistry *commands.CommandRegistry
	keyMapper       *commands.KeyMapper
	highlighter     *syntax.Highlighter
	currentFile     string
	lineOffsets     []int  // Cache of byte offsets for each line
	offsetsDirty    bool   // Whether line offsets need recalculation
}

// New creates a new Editor.
func New() (*Editor, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}

	// Default style will be set from theme
	config := config.New()
	normalStyle := config.Theme.GetStyle(config.Theme.Normal)
	screen.SetStyle(normalStyle)

	// Initialize command line widget (will be positioned at bottom of screen)
	width, height := screen.Size()
	commandLine := widgets.NewLineEditor(0, height-1, width)
	commandLine.SetStyle(config.Theme.GetStyle(config.Theme.CommandLine))

	// Start proxy client
	proxyClient, err := proxy.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to start proxy client: %v", err)
	}

	editor := &Editor{
		screen:          screen,
		buffer:          buffer.NewBufferFromString(""), // Start with empty buffer
		config:          config,
		cursorX:         0,
		cursorY:         0,
		topLine:         0,
		mode:            commands.NormalMode,
		commandLine:     commandLine,
		proxyClient:     proxyClient,
		commandRegistry: commands.NewCommandRegistry(),
		keyMapper:       commands.NewKeyMapper(),
		highlighter:     syntax.NewHighlighter(),
		offsetsDirty:    true, // Need to calculate offsets for initial buffer
	}

	// Register commands
	editor.registerCommands()

	// Set up default key bindings
	editor.setupDefaultBindings()

	return editor, nil
}

// Run starts the editor's main event loop.
func (e *Editor) Run() {
	// Event loop
	for {
		e.drawEditor()
		e.drawStatusBar()

		// Show cursor at appropriate position
		if e.mode == commands.CommandMode {
			// Show cursor in command line
			cursorX, cursorY := e.commandLine.GetCursorPosition()
			e.screen.ShowCursor(cursorX, cursorY)
		} else {
			// Show cursor in editor
			// Calculate line number width based on visible lines
			_, height := e.screen.Size()
			visibleLines := height - 2
			lastVisibleLine := e.topLine + visibleLines
			lineNumberWidth := len(fmt.Sprintf("%d", lastVisibleLine)) + 1
			visualCursorX := e.getVisualCursorX()
			e.screen.ShowCursor(visualCursorX+lineNumberWidth, e.cursorY-e.topLine)
		}

		e.screen.Show()

		ev := e.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if e.mode == commands.CommandMode {
				// In command mode, handle keys directly (no key mapping)
				e.handleCommandMode(ev)
			} else {
				// In normal mode, use key mapper for commands
				if command, found := e.keyMapper.ProcessKey(ev, e.mode); found {
					if err := e.commandRegistry.Execute(command); err != nil {
						// TODO: Show error to user
					}
				}
			}
		case *tcell.EventResize:
			// Handle screen resize events
			e.handleResize()
		}
		e.scroll()
	}
}

func (e *Editor) scroll() {
	_, height := e.screen.Size()
	bufferLines := len(e.buffer.Lines())

	// Scroll up when cursor is too close to top
	if e.cursorY < e.topLine+e.config.ScrollOffset {
		e.topLine = e.cursorY - e.config.ScrollOffset
	}

	// Scroll down when cursor is too close to bottom, but allow cursor to reach
	// the bottom of the screen when at the end of the buffer
	if e.cursorY >= e.topLine+height-1-e.config.ScrollOffset {
		// If we're near the end of the buffer, allow the last line to be at the bottom
		maxTopLine := bufferLines - (height - 2)
		if maxTopLine < 0 {
			maxTopLine = 0
		}

		desiredTopLine := e.cursorY - height + 2 + e.config.ScrollOffset
		if desiredTopLine > maxTopLine {
			e.topLine = maxTopLine
		} else {
			e.topLine = desiredTopLine
		}
	}

	if e.topLine < 0 {
		e.topLine = 0
	}
}

// handleNormalMode processes events when in normal editing mode
func (e *Editor) handleNormalMode(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.Close()
		os.Exit(0)
	case tcell.KeyUp:
		if e.cursorY > 0 {
			e.cursorY--
		}
	case tcell.KeyDown:
		if e.cursorY < len(e.buffer.Lines())-1 {
			e.cursorY++
		}
	case tcell.KeyLeft:
		if e.cursorX > 0 {
			e.cursorX--
		}
	case tcell.KeyRight:
		line := e.buffer.Line(e.cursorY)
		// Only allow cursor to move to the last character, not past it
		maxX := len(line)
		if maxX > 0 {
			maxX-- // Don't go past the last character
		}
		if e.cursorX < maxX {
			e.cursorX++
		}
	case tcell.KeyPgUp:
		_, height := e.screen.Size()
		visibleLines := height - 2 // Subtract status bar lines

		// Move cursor up by visible height
		e.cursorY -= visibleLines
		if e.cursorY < 0 {
			e.cursorY = 0
		}

		// Adjust cursor X if needed
		line := e.buffer.Line(e.cursorY)
		if e.cursorX > len(line) {
			e.cursorX = len(line)
			if e.cursorX > 0 {
				e.cursorX--
			}
		}
	case tcell.KeyPgDn:
		_, height := e.screen.Size()
		visibleLines := height - 2 // Subtract status bar lines

		// Move cursor down by visible height
		e.cursorY += visibleLines
		maxY := len(e.buffer.Lines()) - 1
		if e.cursorY > maxY {
			e.cursorY = maxY
		}

		// Adjust cursor X if needed
		line := e.buffer.Line(e.cursorY)
		if e.cursorX > len(line) {
			e.cursorX = len(line)
			if e.cursorX > 0 {
				e.cursorX--
			}
		}
	case tcell.KeyRune:
		// Handle colon to enter command mode
		if ev.Rune() == ':' {
			e.enterCommandMode()
		} else if ev.Rune() == 'A' {
			e.showFuzzyListWidget()
		}
	}
}

// handleCommandMode processes events when in command mode
func (e *Editor) handleCommandMode(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		// Exit command mode
		e.exitCommandMode()
	case tcell.KeyEnter:
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

	// Handle basic commands
	switch command {
	case "q", "quit":
		e.Close()
		os.Exit(0)
	}
}

// handleResize processes screen resize events
func (e *Editor) handleResize() {
	width, height := e.screen.Size()

	// Update the command line widget's dimensions and position
	e.commandLine.SetWidth(width)
	e.commandLine.SetPosition(0, height-1)
}

// Close finalizes the screen and proxy client.
func (e *Editor) Close() {
	e.screen.Fini()
	if e.proxyClient != nil {
		e.proxyClient.Close()
	}
}

// drawText is a helper function to draw text on the screen.
// Returns the number of screen columns used.
func (e *Editor) drawText(x, y int, style tcell.Style, text string) int {
	screenX := x
	for _, r := range text {
		if r == '\t' {
			// Render tab as 4 spaces
			for i := 0; i < 4; i++ {
				e.screen.SetContent(screenX, y, ' ', nil, style)
				screenX++
			}
		} else {
			e.screen.SetContent(screenX, y, r, nil, style)
			screenX++
		}
	}
	return screenX - x
}

// drawLine draws a single line with syntax highlighting
func (e *Editor) drawLine(lineIndex int, startX int, screenY int) {
	if lineIndex < 0 || lineIndex >= len(e.buffer.Lines()) {
		return
	}

	line := e.buffer.Line(lineIndex)
	normalStyle := e.config.Theme.GetStyle(e.config.Theme.Normal)

	// If no highlighter is set up, just draw normally
	if e.highlighter == nil || e.currentFile == "" {
		e.drawText(startX, screenY, normalStyle, line)
		return
	}

	// Get syntax tokens for this line
	tokens := e.getLineTokens(lineIndex)

	if len(tokens) == 0 {
		// No tokens, draw with normal style
		e.drawText(startX, screenY, normalStyle, line)
		return
	}

	// Draw with syntax highlighting
	x := startX
	lastEnd := 0

	for _, token := range tokens {
		// Draw any text before this token with normal style
		if token.Start > lastEnd {
			prefix := line[lastEnd:token.Start]
			width := e.drawText(x, screenY, normalStyle, prefix)
			x += width
		}

		// Draw the token with its specific style
		if token.End <= len(line) {
			tokenText := line[token.Start:token.End]
			tokenStyle := e.config.Theme.GetStyle(syntax.GetStyleForToken(token.TokenType, e.config.Theme))
			width := e.drawText(x, screenY, tokenStyle, tokenText)
			x += width
			lastEnd = token.End
		}
	}

	// Draw any remaining text with normal style
	if lastEnd < len(line) {
		suffix := line[lastEnd:]
		e.drawText(x, screenY, normalStyle, suffix)
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
	if e.cursorY >= len(e.buffer.Lines()) {
		return 0
	}
	line := e.buffer.Line(e.cursorY)
	if e.cursorX >= len(line) {
		return e.visualWidth(line)
	}
	return e.visualWidth(line[:e.cursorX])
}

// updateLineOffsets recalculates the byte offset cache for all lines
func (e *Editor) updateLineOffsets() {
	lines := e.buffer.Lines()
	e.lineOffsets = make([]int, len(lines)+1)
	
	offset := 0
	for i, line := range lines {
		e.lineOffsets[i] = offset
		offset += len(line) + 1 // +1 for newline
	}
	e.lineOffsets[len(lines)] = offset
	e.offsetsDirty = false
}

// getLineTokens returns syntax tokens for a specific line
func (e *Editor) getLineTokens(lineIndex int) []syntax.Token {
	// Update cache if needed
	if e.offsetsDirty || len(e.lineOffsets) == 0 {
		e.updateLineOffsets()
	}
	
	// Use cached offsets
	if lineIndex >= 0 && lineIndex < len(e.lineOffsets)-1 {
		lineStart := e.lineOffsets[lineIndex]
		lineEnd := e.lineOffsets[lineIndex+1] - 1 // -1 to exclude newline
		return e.highlighter.GetTokensForLine(lineIndex, lineStart, lineEnd)
	}
	
	return nil
}

func (e *Editor) drawEditor() {
	_, height := e.screen.Size()
	lineNumberStyle := e.config.Theme.GetStyle(e.config.Theme.LineNumber)
	totalLines := len(e.buffer.Lines())

	// Calculate line number width based on visible lines
	visibleLines := height - 2
	lastVisibleLine := e.topLine + visibleLines
	lineNumberWidth := len(fmt.Sprintf("%d", lastVisibleLine)) + 1 // +1 for padding

	// Clear the entire screen once
	e.screen.Clear()

	for i := 0; i < height-2; i++ {
		// Draw line number
		lineNum := e.topLine + i + 1 // Line numbers start at 1
		if lineNum <= totalLines {
			lineNumStr := fmt.Sprintf("%*d", lineNumberWidth-1, lineNum)
			e.drawText(0, i, lineNumberStyle, lineNumStr)

			// Draw separator after line number
			e.screen.SetContent(lineNumberWidth-1, i, ' ', nil, lineNumberStyle)

			// Draw the actual line content
			e.drawLine(e.topLine + i, lineNumberWidth, i)
		}
	}
}

func (e *Editor) drawStatusBar() {
	width, height := e.screen.Size()
	statusBarStyle := e.config.Theme.GetStyle(e.config.Theme.StatusBar)

	// Fill status bar background
	for i := 0; i < width; i++ {
		e.screen.SetContent(i, height-2, ' ', nil, statusBarStyle)
		e.screen.SetContent(i, height-1, ' ', nil, statusBarStyle)
	}

	if e.mode == commands.CommandMode {
		// Draw the command line widget
		e.commandLine.Draw(e.screen)
		e.drawText(0, height-2, statusBarStyle, "Command mode - Press ESC to cancel, Enter to execute")
	} else {
		e.drawText(0, height-2, statusBarStyle, "Press : for command mode, ESC to quit.")
		e.drawText(0, height-1, statusBarStyle, fmt.Sprintf("Length: %d", e.buffer.Length()))
	}
}

// showFuzzyListWidget creates and displays the fuzzy list widget with directory listing
func (e *Editor) showFuzzyListWidget() {
	width, height := e.screen.Size()

	// Create fuzzy list widget (centered overlay)
	fuzzyListWidth := width * 3 / 5  // 60% of screen width
	fuzzyListHeight := height * 3 / 5 // 60% of screen height
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
	selection := fuzzyListWidget.Show(e.screen, func() {
		e.drawEditor()
		e.drawStatusBar()
	})

	// If a file was selected, open it
	if selection != "" {
		e.openFile(selection)
	}
}

// openFile loads a file's content into the editor buffer
func (e *Editor) openFile(filePath string) {
	// Read file content using proxy
	content, err := e.proxyClient.GetFile(filePath)
	if err != nil {
		// TODO: Show error to user
		return
	}

	// Create a new buffer with the file content
	e.buffer = buffer.NewBufferFromString(content)

	// Store the current file path
	e.currentFile = filePath

	// Set up syntax highlighting
	if e.highlighter != nil {
		e.highlighter.SetLanguageFromFile(filePath)
		e.highlighter.Parse([]byte(content))
	}

	// Reset cursor and scroll position
	e.cursorX = 0
	e.cursorY = 0
	e.topLine = 0
	
	// Mark line offsets as needing recalculation
	e.offsetsDirty = true
}

// Movement commands
func (e *Editor) moveUp() error {
	if e.cursorY > 0 {
		e.cursorY--
	}
	return nil
}

func (e *Editor) moveDown() error {
	if e.cursorY < len(e.buffer.Lines())-1 {
		e.cursorY++
	}
	return nil
}

func (e *Editor) moveLeft() error {
	if e.cursorX > 0 {
		e.cursorX--
	}
	return nil
}

func (e *Editor) moveRight() error {
	line := e.buffer.Line(e.cursorY)
	maxX := len(line)
	if maxX > 0 {
		maxX-- // Don't go past the last character
	}
	if e.cursorX < maxX {
		e.cursorX++
	}
	return nil
}

func (e *Editor) pageUp() error {
	_, height := e.screen.Size()
	visibleLines := height - 2 // Subtract status bar lines

	e.cursorY -= visibleLines
	if e.cursorY < 0 {
		e.cursorY = 0
	}

	// Adjust cursor X if needed
	line := e.buffer.Line(e.cursorY)
	if e.cursorX > len(line) {
		e.cursorX = len(line)
		if e.cursorX > 0 {
			e.cursorX--
		}
	}
	return nil
}

func (e *Editor) pageDown() error {
	_, height := e.screen.Size()
	visibleLines := height - 2 // Subtract status bar lines

	e.cursorY += visibleLines
	maxY := len(e.buffer.Lines()) - 1
	if e.cursorY > maxY {
		e.cursorY = maxY
	}

	// Adjust cursor X if needed
	line := e.buffer.Line(e.cursorY)
	if e.cursorX > len(line) {
		e.cursorX = len(line)
		if e.cursorX > 0 {
			e.cursorX--
		}
	}
	return nil
}

func (e *Editor) quit() error {
	e.Close()
	os.Exit(0)
	return nil
}

func (e *Editor) enterCommandModeCmd() error {
	e.enterCommandMode()
	return nil
}

func (e *Editor) openFileCmd() error {
	e.showFuzzyListWidget()
	return nil
}

// registerKeyBinding registers a key binding with a command
func (e *Editor) registerKeyBinding(mode commands.Mode, keys string, command string) {
	keySeq, err := commands.ParseKeySequence(keys)
	if err != nil {
		panic(fmt.Sprintf("Invalid key sequence: %s", keys))
	}

	e.keyMapper.AddBinding(commands.Binding{
		Keys:    keySeq,
		Command: command,
		Modes:   []commands.Mode{mode},
	})
}

// registerCommands registers all command handlers
func (e *Editor) registerCommands() {
	// Movement commands
	e.commandRegistry.Register("up", e.moveUp)
	e.commandRegistry.Register("down", e.moveDown)
	e.commandRegistry.Register("left", e.moveLeft)
	e.commandRegistry.Register("right", e.moveRight)
	e.commandRegistry.Register("page-up", e.pageUp)
	e.commandRegistry.Register("page-down", e.pageDown)

	// Mode commands
	e.commandRegistry.Register("enter-command-mode", e.enterCommandModeCmd)

	// File commands
	e.commandRegistry.Register("open-file", e.openFileCmd)
	e.commandRegistry.Register("quit", e.quit)
}

// setupDefaultBindings sets up the default key bindings
func (e *Editor) setupDefaultBindings() {
	// Movement keys
	e.registerKeyBinding(commands.NormalMode, "up", "up")
	e.registerKeyBinding(commands.NormalMode, "down", "down")
	e.registerKeyBinding(commands.NormalMode, "left", "left")
	e.registerKeyBinding(commands.NormalMode, "right", "right")
	e.registerKeyBinding(commands.NormalMode, "pageup", "page-up")
	e.registerKeyBinding(commands.NormalMode, "pagedown", "page-down")

	// Command mode
	e.registerKeyBinding(commands.NormalMode, ":", "enter-command-mode")

	// File operations
	e.registerKeyBinding(commands.NormalMode, "space s f", "open-file")

	// Quit
	e.registerKeyBinding(commands.NormalMode, "escape", "quit")
}
