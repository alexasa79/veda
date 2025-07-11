package editor

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/log"
	"veda/internal/proxy"
	"veda/internal/syntax"
	"veda/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

var LL *log.Logger = log.NewLogger("EDITOR")

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

	// Buffer management
	buffers         map[string]*BufferInfo  // Map of file path to buffer info
	bufferOrder     []string                // Order of buffers for cycling

	// Navigation history
	navHistory      []NavigationEntry       // Stack of previous positions
	maxNavHistory   int                     // Maximum number of entries to keep
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
	initialBuffer := buffer.NewBufferFromString("")

	editor := &Editor{
		screen:          screen,
		buffer:          initialBuffer,
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
		buffers:         make(map[string]*BufferInfo),
		bufferOrder:     []string{},
		navHistory:      []NavigationEntry{},
		maxNavHistory:   50, // Keep last 50 navigation entries
	}

	// Add the initial empty buffer
	emptyBufferInfo := &BufferInfo{
		buffer:       initialBuffer,
		filePath:     "[No Name]",
		cursorX:      0,
		cursorY:      0,
		topLine:      0,
		lineOffsets:  []int{},
		offsetsDirty: true,
	}
	editor.buffers["[No Name]"] = emptyBufferInfo
	editor.bufferOrder = append(editor.bufferOrder, "[No Name]")
	editor.currentFile = "[No Name]"

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

		// Set cursor style based on mode
		if e.mode == commands.InsertMode {
			// Use a bar cursor in insert mode
			e.screen.SetCursorStyle(tcell.CursorStyleSteadyBar)
		} else {
			// Use a block cursor in normal and command modes
			e.screen.SetCursorStyle(tcell.CursorStyleSteadyBlock)
		}

		// Show cursor at appropriate position
		if e.mode == commands.CommandMode {
			// Show cursor in command line
			cursorX, cursorY := e.commandLine.GetCursorPosition()
			e.screen.ShowCursor(cursorX, cursorY)
		} else {
			// Show cursor in editor
			// Calculate line number width based on visible lines
			visibleLines := e.getVisibleLines()
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
				// Command mode is special - it's a line editor
				e.handleCommandMode(ev)
			} else {
				// Normal and Insert modes both use key mapping
				if command, found := e.keyMapper.ProcessKey(ev, e.mode); found {
					if err := e.commandRegistry.Execute(command); err != nil {
						// TODO: Show error to user
					}
				} else if e.mode == commands.InsertMode && ev.Key() == tcell.KeyRune {
					// In insert mode, insert unmapped rune keys
					e.insertChar(ev.Rune())
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
	visibleHeight := e.getVisibleHeight()
	bufferLines := len(e.buffer.Lines())

	// Scroll up when cursor is too close to top
	if e.cursorY < e.topLine+e.config.ScrollOffset {
		e.topLine = e.cursorY - e.config.ScrollOffset
	}

	// Scroll down when cursor is too close to bottom, but allow cursor to reach
	// the bottom of the screen when at the end of the buffer
	if e.cursorY >= e.topLine+visibleHeight-e.config.ScrollOffset {
		// If we're near the end of the buffer, allow the last line to be at the bottom
		maxTopLine := bufferLines - visibleHeight
		if maxTopLine < 0 {
			maxTopLine = 0
		}

		desiredTopLine := e.cursorY - visibleHeight + e.config.ScrollOffset
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

// scrollToShowAtTop positions the view so the cursor line appears at ScrollOffset from the top
func (e *Editor) scrollToShowAtTop() {
	visibleHeight := e.getVisibleHeight()
	bufferLines := len(e.buffer.Lines())
	
	// Position cursor at ScrollOffset from the top
	e.topLine = e.cursorY - e.config.ScrollOffset
	
	// Make sure we don't scroll past the beginning
	if e.topLine < 0 {
		e.topLine = 0
	}
	
	// Make sure we don't scroll past the end
	maxTopLine := bufferLines - visibleHeight
	if maxTopLine < 0 {
		maxTopLine = 0
	}
	if e.topLine > maxTopLine {
		e.topLine = maxTopLine
	}
}



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

	// First try to execute from the command registry
	if e.commandRegistry.Has(command) {
		if err := e.commandRegistry.Execute(command); err != nil {
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

// handleResize processes screen resize events
func (e *Editor) handleResize() {
	width, _ := e.screen.Size()

	// Update the command line widget's dimensions and position
	e.commandLine.SetWidth(width)
	e.commandLine.SetPosition(0, e.getStatusBarY())
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
		if token.Start > lastEnd && token.Start <= len(line) {
			endPos := token.Start
			if endPos > len(line) {
				endPos = len(line)
			}
			prefix := line[lastEnd:endPos]
			width := e.drawText(x, screenY, normalStyle, prefix)
			x += width
		}

		// Draw the token with its specific style
		if token.Start < len(line) && token.End <= len(line) {
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
	width, _ := e.screen.Size()
	lineNumberStyle := e.config.Theme.GetStyle(e.config.Theme.LineNumber)
	currentLineStyle := e.config.Theme.GetStyle(e.config.Theme.CurrentLine)
	totalLines := len(e.buffer.Lines())

	// Calculate line number width based on visible lines
	visibleLines := e.getVisibleLines()
	lastVisibleLine := e.topLine + visibleLines
	lineNumberWidth := len(fmt.Sprintf("%d", lastVisibleLine)) + 1 // +1 for padding

	// Clear the entire screen once
	e.screen.Clear()

	visibleHeight := e.getVisibleHeight()
	for i := 0; i < visibleHeight; i++ {
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
	width, _ := e.screen.Size()
	statusBarStyle := e.config.Theme.GetStyle(e.config.Theme.StatusBar)
	lineColStyle := e.config.Theme.GetStyle(e.config.Theme.LineColumn)
	statusBarY := e.getStatusBarY()

	// Fill status bar background
	for i := 0; i < width; i++ {
		e.screen.SetContent(i, statusBarY, ' ', nil, statusBarStyle)
	}

	if e.mode == commands.CommandMode {
		// Draw the command line widget
		e.commandLine.Draw(e.screen)
	} else {
		// Status bar content
		// Left side: mode and filename
		x := 0
		modeText := "NORMAL  "
		if e.mode == commands.InsertMode {
			modeText = "INSERT  "
		}
		x += e.drawText(x, statusBarY, statusBarStyle, modeText)

		filename := e.currentFile
		if filename == "" {
			filename = "[No Name]"
		}
		x += e.drawText(x, statusBarY, statusBarStyle, filename)

		// Right side: line:col
		lines := e.buffer.Lines()

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
		rightX := width - len(lineColInfo)

		if rightX > x { // Only draw if there's space
			e.drawText(rightX, statusBarY, lineColStyle, lineColInfo)
		}
	}
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

// Insert mode movement commands - allow cursor to go past last character
func (e *Editor) moveUpInsert() error {
	if e.cursorY > 0 {
		e.cursorY--
		// Adjust cursor X if the new line is shorter
		line := e.buffer.Line(e.cursorY)
		if e.cursorX > len(line) {
			e.cursorX = len(line)
		}
	}
	return nil
}

func (e *Editor) moveDownInsert() error {
	if e.cursorY < len(e.buffer.Lines())-1 {
		e.cursorY++
		// Adjust cursor X if the new line is shorter
		line := e.buffer.Line(e.cursorY)
		if e.cursorX > len(line) {
			e.cursorX = len(line)
		}
	}
	return nil
}

func (e *Editor) moveLeftInsert() error {
	if e.cursorX > 0 {
		e.cursorX--
	}
	return nil
}

func (e *Editor) moveRightInsert() error {
	line := e.buffer.Line(e.cursorY)
	// In insert mode, allow cursor to go past the last character
	if e.cursorX < len(line) {
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

// beginningOfLine moves cursor to the beginning of the current line
func (e *Editor) beginningOfLine() error {
	e.cursorX = 0
	return nil
}

// endOfLine moves cursor to the end of the current line
func (e *Editor) endOfLine() error {
	lines := e.buffer.Lines()
	if e.cursorY < len(lines) {
		lineLen := len(lines[e.cursorY])
		if lineLen > 0 {
			e.cursorX = lineLen - 1
		} else {
			e.cursorX = 0
		}
	}
	return nil
}

// beginningOfLineInsert is the same as beginningOfLine but for insert mode
func (e *Editor) beginningOfLineInsert() error {
	return e.beginningOfLine()
}

// endOfLineInsert moves cursor past the end of line for insert mode
func (e *Editor) endOfLineInsert() error {
	lines := e.buffer.Lines()
	if e.cursorY < len(lines) {
		e.cursorX = len(lines[e.cursorY])
	}
	return nil
}

func (e *Editor) pageUp() error {
	visibleLines := e.getVisibleLines()

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
	visibleLines := e.getVisibleLines()

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

// Insert mode commands
func (e *Editor) enterInsertMode() error {
	e.mode = commands.InsertMode
	return nil
}

func (e *Editor) exitInsertMode() error {
	e.mode = commands.NormalMode
	return nil
}

func (e *Editor) insertChar(ch rune) error {
	// Calculate the absolute position in the buffer
	pos := e.getAbsolutePosition()

	// Insert the character
	e.buffer.Insert(pos, string(ch))

	// Move cursor forward
	e.cursorX++

	// Mark offsets as dirty
	e.offsetsDirty = true

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

func (e *Editor) insertNewline() error {
	// Get the absolute position
	pos := e.getAbsolutePosition()

	// Insert newline
	e.buffer.Insert(pos, "\n")

	// Move cursor to beginning of next line
	e.cursorY++
	e.cursorX = 0

	// Mark offsets as dirty
	e.offsetsDirty = true

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

func (e *Editor) backspace() error {
	// If at beginning of buffer, nothing to delete
	if e.cursorX == 0 && e.cursorY == 0 {
		return nil
	}

	pos := e.getAbsolutePosition()

	if e.cursorX == 0 {
		// At beginning of line, join with previous line
		if pos > 0 {
			// Store the length of the previous line before joining
			lines := e.buffer.Lines()
			prevLineLen := 0
			if e.cursorY > 0 && e.cursorY-1 < len(lines) {
				prevLineLen = len(lines[e.cursorY-1])
			}

			// For now, we'll implement a simple delete by recreating the buffer
			// TODO: Add proper Delete method to buffer
			content := e.buffer.String()
			if pos-1 < len(content) {
				newContent := content[:pos-1] + content[pos:]
				e.buffer = buffer.NewBufferFromString(newContent)
				e.cursorY--
				e.cursorX = prevLineLen
				// Update syntax highlighting
				e.updateSyntaxHighlighting()
			}
		}
	} else {
		// Delete character before cursor
		if pos > 0 {
			// For now, we'll implement a simple delete by recreating the buffer
			// TODO: Add proper Delete method to buffer
			content := e.buffer.String()
			if pos-1 < len(content) {
				newContent := content[:pos-1] + content[pos:]
				e.buffer = buffer.NewBufferFromString(newContent)
				e.cursorX--
				// Update syntax highlighting
				e.updateSyntaxHighlighting()
			}
		}
	}

	// Mark offsets as dirty
	e.offsetsDirty = true

	return nil
}

// Layout calculation methods
// These methods centralize the calculation of screen areas, making it easy to change
// the layout (e.g., status bar height) without modifying code throughout the editor.

// getVisibleHeight returns the height of the visible editing area (excluding status bar)
func (e *Editor) getVisibleHeight() int {
	_, height := e.screen.Size()
	return height - 1 // Reserve 1 line for status bar
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
	lines := e.buffer.Lines()
	pos := 0

	// Add lengths of all previous lines
	for i := 0; i < e.cursorY && i < len(lines); i++ {
		pos += len(lines[i]) + 1 // +1 for newline
	}

	// Add current X position
	if e.cursorY < len(lines) {
		if e.cursorX > len(lines[e.cursorY]) {
			pos += len(lines[e.cursorY])
		} else {
			pos += e.cursorX
		}
	}

	return pos
}

// updateSyntaxHighlighting re-parses the buffer for syntax highlighting
func (e *Editor) updateSyntaxHighlighting() {
	if e.highlighter != nil && e.currentFile != "" {
		content := e.buffer.String()
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
		// Update existing buffer
		bufferInfo.buffer = buffer.NewBufferFromString(content.String())
		bufferInfo.cursorX = 0
		bufferInfo.cursorY = 0
		bufferInfo.topLine = 0
		bufferInfo.offsetsDirty = true
		e.switchToBuffer(bufferInfo)
	} else {
		// Create new buffer
		newBuffer := buffer.NewBufferFromString(content.String())
		bufferInfo := &BufferInfo{
			buffer:       newBuffer,
			filePath:     lspBufferName,
			cursorX:      0,
			cursorY:      0,
			topLine:      0,
			lineOffsets:  []int{},
			offsetsDirty: true,
		}
		e.buffers[lspBufferName] = bufferInfo
		e.bufferOrder = append(e.bufferOrder, lspBufferName)
		e.switchToBuffer(bufferInfo)
	}

	return nil
}

func (e *Editor) goToDefinition() error {
	// Only works if we have a file open
	if e.currentFile == "" {
		// TODO: Show error to user
		return fmt.Errorf("no file open")
	}

	// Get cursor position (LSP uses 0-based line numbers)
	line := e.cursorY
	character := e.cursorX

	LL.Trace("goToDefinition: file=%s, line=%d, character=%d", e.currentFile, line, character)

	// Get definition location from LSP
	locations, err := e.proxyClient.GoToDefinition(e.currentFile, line, character)
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
	if filePath != e.currentFile {
		e.openFile(filePath)
	}

	// Jump to the location
	e.cursorY = location.Range.Start.Line
	e.cursorX = location.Range.Start.Character

	// Ensure cursor is within bounds
	lines := e.buffer.Lines()
	if e.cursorY >= len(lines) {
		e.cursorY = len(lines) - 1
	}
	if e.cursorY >= 0 && e.cursorY < len(lines) {
		line := lines[e.cursorY]
		if e.cursorX > len(line) {
			e.cursorX = len(line)
			if e.cursorX > 0 {
				e.cursorX-- // Don't go past last character in normal mode
			}
		}
	}

	// Position the definition at ScrollOffset from the top of the screen
	e.scrollToShowAtTop()

	return nil
}

