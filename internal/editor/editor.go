package editor

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/proxy"
	"veda/internal/syntax"
	"veda/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

// Editor represents the state of the text editor.
type Editor struct {
	screen          Screen
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

// NewWithScreen creates a new Editor with a custom screen implementation.
func NewWithScreen(screen Screen) (*Editor, error) {
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

// New creates a new Editor with the default tcell screen.
func New() (*Editor, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	
	return NewWithScreen(NewTcellScreen(screen))
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
	isCurrentLine := lineIndex == e.cursorY
	
	// If current line, we'll need to override background colors
	var currentLineBg tcell.Color
	if isCurrentLine {
		currentLineStyle := e.config.Theme.GetStyle(e.config.Theme.CurrentLine)
		_, currentLineBg, _ = currentLineStyle.Decompose()
		normalStyle = normalStyle.Background(currentLineBg)
	}

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
			
			// Override background for current line
			if isCurrentLine {
				tokenStyle = tokenStyle.Background(currentLineBg)
			}
			
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
	width, height := e.screen.Size()
	lineNumberStyle := e.config.Theme.GetStyle(e.config.Theme.LineNumber)
	currentLineStyle := e.config.Theme.GetStyle(e.config.Theme.CurrentLine)
	totalLines := len(e.buffer.Lines())

	// Calculate line number width based on visible lines
	visibleLines := height - 2
	lastVisibleLine := e.topLine + visibleLines
	lineNumberWidth := len(fmt.Sprintf("%d", lastVisibleLine)) + 1 // +1 for padding

	// Clear the entire screen once
	e.screen.Clear()

	for i := 0; i < height-2; i++ {
		lineIndex := e.topLine + i
		isCurrentLine := lineIndex == e.cursorY
		
		// Fill current line background
		if isCurrentLine {
			for j := 0; j < width; j++ {
				e.screen.SetContent(j, i, ' ', nil, currentLineStyle)
			}
		}
		
		// Draw line number
		lineNum := lineIndex + 1 // Line numbers start at 1
		if lineNum <= totalLines {
			lineNumStr := fmt.Sprintf("%*d", lineNumberWidth-1, lineNum)
			e.drawText(0, i, lineNumberStyle, lineNumStr)

			// Draw separator after line number
			e.screen.SetContent(lineNumberWidth-1, i, ' ', nil, lineNumberStyle)

			// Draw the actual line content
			e.drawLine(lineIndex, lineNumberWidth, i)
		}
	}
}

func (e *Editor) drawStatusBar() {
	width, height := e.screen.Size()
	statusBarStyle := e.config.Theme.GetStyle(e.config.Theme.StatusBar)
	lineColStyle := e.config.Theme.GetStyle(e.config.Theme.LineColumn)

	// Fill status bar background
	for i := 0; i < width; i++ {
		e.screen.SetContent(i, height-2, ' ', nil, statusBarStyle)
		e.screen.SetContent(i, height-1, ' ', nil, statusBarStyle)
	}

	if e.mode == commands.CommandMode {
		// Draw the command line widget
		e.commandLine.Draw(e.screen)
		// Left side: mode
		e.drawText(0, height-2, statusBarStyle, "COMMAND")
	} else {
		// First line of status bar
		// Left side: mode and filename
		x := 0
		x += e.drawText(x, height-2, statusBarStyle, "NORMAL  ")
		
		filename := e.currentFile
		if filename == "" {
			filename = "[No Name]"
		}
		x += e.drawText(x, height-2, statusBarStyle, filename)
		
		// Right side: file size, lines, line:col
		lines := e.buffer.Lines()
		totalLines := len(lines)
		
		// Calculate file size in bytes
		fileSize := 0
		for _, line := range lines {
			fileSize += len(line) + 1 // +1 for newline
		}
		if totalLines > 0 {
			fileSize-- // Remove last newline
		}
		
		// Format right side info
		rightInfo := fmt.Sprintf("%d bytes  %d lines  ", fileSize, totalLines)
		// Use visual column position for display (accounting for tabs)
		visualCol := 1
		if e.cursorY < len(lines) {
			line := e.buffer.Line(e.cursorY)
			if e.cursorX <= len(line) {
				visualCol = e.visualWidth(line[:e.cursorX]) + 1
			}
		}
		lineColInfo := fmt.Sprintf("%d:%d", e.cursorY+1, visualCol)
		
		// Calculate position for right-aligned text
		rightWidth := len(rightInfo) + len(lineColInfo)
		rightX := width - rightWidth
		
		if rightX > x { // Only draw if there's space
			e.drawText(rightX, height-2, statusBarStyle, rightInfo)
			e.drawText(rightX+len(rightInfo), height-2, lineColStyle, lineColInfo)
		}
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
	// The fuzzy list widget needs the full tcell.Screen interface
	var tcellScreen tcell.Screen
	if provider, ok := e.screen.(TcellScreenProvider); ok {
		tcellScreen = provider.GetTcellScreen()
	} else {
		// For testing, we'll need to handle this differently
		// For now, just return empty selection
		return
	}
	
	selection := fuzzyListWidget.Show(tcellScreen, func() {
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
		// Adjust cursor X if the new line is shorter
		line := e.buffer.Line(e.cursorY)
		maxX := len(line)
		if maxX > 0 {
			maxX-- // Don't go past the last character
		}
		if e.cursorX > maxX {
			e.cursorX = maxX
		}
	}
	return nil
}

func (e *Editor) moveDown() error {
	if e.cursorY < len(e.buffer.Lines())-1 {
		e.cursorY++
		// Adjust cursor X if the new line is shorter
		line := e.buffer.Line(e.cursorY)
		maxX := len(line)
		if maxX > 0 {
			maxX-- // Don't go past the last character
		}
		if e.cursorX > maxX {
			e.cursorX = maxX
		}
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

// wordForward moves the cursor to the beginning of the next word
func (e *Editor) wordForward() error {
	lines := e.buffer.Lines()
	if len(lines) == 0 {
		return nil
	}
	
	// Start from current position
	y := e.cursorY
	x := e.cursorX + 1 // Start searching from next character
	
	// Track if we've seen a non-alphanumeric character
	seenNonAlnum := false
	
	// If we start on a non-alphanumeric character, we're ready to find the next word
	if e.cursorX < len(lines[e.cursorY]) {
		ch := rune(lines[e.cursorY][e.cursorX])
		seenNonAlnum = !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_'
	}
	
	// Loop through characters until we find a transition from non-alnum to alnum
	for y < len(lines) {
		line := lines[y]
		
		for x < len(line) {
			ch := rune(line[x])
			isAlnum := unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
			
			if seenNonAlnum && isAlnum {
				// Found transition from non-alnum to alnum
				e.cursorX = x
				e.cursorY = y
				return nil
			}
			
			if !isAlnum {
				seenNonAlnum = true
			}
			
			x++
		}
		
		// Move to next line
		y++
		x = 0
		seenNonAlnum = true // Line break counts as non-alnum
	}
	
	// No more words found, stay at current position
	return nil
}

// wordBackward moves the cursor to the beginning of the previous word
func (e *Editor) wordBackward() error {
	lines := e.buffer.Lines()
	if len(lines) == 0 {
		return nil
	}
	
	// If we're at the very beginning, can't go back
	if e.cursorY == 0 && e.cursorX == 0 {
		return nil
	}
	
	// Start from current position
	y := e.cursorY
	x := e.cursorX
	
	// Helper to check if character is word character
	isWordChar := func(ch rune) bool {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
	}
	
	// Move left one position
	x--
	if x < 0 {
		y--
		if y < 0 {
			return nil
		}
		x = len(lines[y]) - 1
		if x < 0 {
			x = 0
		}
	}
	
	// State: are we looking for a word start?
	lookingForWord := true
	
	// If we're on a word character, we need to skip to the beginning of this word
	if y >= 0 && x < len(lines[y]) && isWordChar(rune(lines[y][x])) {
		lookingForWord = false
	}
	
	// Search backwards
	for y >= 0 {
		line := lines[y]
		
		for x >= 0 {
			if x < len(line) {
				ch := rune(line[x])
				if lookingForWord {
					// Looking for a word character
					if isWordChar(ch) {
						// Found end of a word, now find its beginning
						lookingForWord = false
					}
				} else {
					// In a word, looking for its beginning
					if !isWordChar(ch) {
						// Found non-word character, word starts at x+1
						e.cursorX = x + 1
						e.cursorY = y
						return nil
					}
				}
			}
			
			x--
		}
		
		// If we were in a word and reached beginning of line
		if !lookingForWord {
			e.cursorX = 0
			e.cursorY = y
			return nil
		}
		
		// Move to previous line
		y--
		if y >= 0 {
			x = len(lines[y]) - 1
			lookingForWord = true // Line break is a word boundary
		}
	}
	
	// Reached beginning of buffer
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
	e.commandRegistry.Register("word-forward", e.wordForward)
	e.commandRegistry.Register("word-backward", e.wordBackward)
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
	e.registerKeyBinding(commands.NormalMode, "w", "word-forward")
	e.registerKeyBinding(commands.NormalMode, "b", "word-backward")

	// Command mode
	e.registerKeyBinding(commands.NormalMode, ":", "enter-command-mode")

	// File operations
	e.registerKeyBinding(commands.NormalMode, "space s f", "open-file")

	// Quit
	e.registerKeyBinding(commands.NormalMode, "escape", "quit")
}
