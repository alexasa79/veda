package editor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/log"
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
	cursorX         int
	cursorY         int
	topLine         int
	mode            commands.Mode
	commandLine     *widgets.LineEditor
	proxyClient     *proxy.Client
	commandRegistry *commands.CommandRegistry
	keyMapper       *commands.KeyMapper
	highlighter     *syntax.Highlighter
	lineOffsets     []int  // Cache of byte offsets for each line
	offsetsDirty    bool   // Whether line offsets need recalculation

	// Buffer management
	buffers         map[string]*buffer.Buffer  // Map of file path to buffer
	bufferOrder     []string                   // Order of buffers for cycling

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
	initialBuffer := buffer.NewBuffer("", "", config.DefaultIndentType, config.DefaultIndentWidth)

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
		buffers:         make(map[string]*buffer.Buffer),
		bufferOrder:     []string{},
		navHistory:      []NavigationEntry{},
		maxNavHistory:   50, // Keep last 50 navigation entries
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
	e.drawStatusBar()
	e.updateCursor()
	e.screen.Show()

	// Event loop
	for {
		ev := e.screen.PollEvent()

		// Track if we need a full redraw
		needsRedraw := false
		oldTopLine := e.topLine

		switch ev := ev.(type) {
		case *ui.EventKey:
			if e.mode == commands.CommandMode {
				// Command mode is special - it's a line editor
				e.handleCommandMode(ev)
				needsRedraw = true // Command mode always needs redraw
			} else {
				// Normal and Insert modes both use key mapping
				if command, found := e.keyMapper.ProcessKey(ev, e.mode); found {
					err, cmdNeedsRedraw := e.commandRegistry.Execute(command)
					if err != nil {
						// TODO: Show error to user
					}
					needsRedraw = cmdNeedsRedraw
				} else if e.mode == commands.InsertMode && ev.Key == ui.KeyRune {
					// In insert mode, insert unmapped rune keys
					e.insertChar(ev.Rune)
					needsRedraw = true
				}
			}
		case *ui.EventResize:
			// Handle screen resize events
			e.handleResize()
			needsRedraw = true
		}

		// Check if scrolling occurred
		e.scroll()
		if oldTopLine != e.topLine {
			needsRedraw = true
		}

		// Redraw if needed
		if needsRedraw {
			e.drawEditor()
			e.drawStatusBar()
		}

		// Always update cursor position and show
		e.updateCursor()
		e.screen.Show()
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
		e.screen.ShowCursor(visualCursorX+lineNumberWidth, e.cursorY-e.topLine)
	}
}

func (e *Editor) scroll() {
	visibleHeight := e.getVisibleHeight()
	bufferLines := e.buffer.LineCount

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
	bufferLines := e.buffer.LineCount

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
		err, _ := e.commandRegistry.Execute(command)
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
	e.screen.Fini()
	if e.proxyClient != nil {
		e.proxyClient.Close()
	}
}

// drawText is a helper function to draw text on the screen.
// Returns the number of screen columns used.
func (e *Editor) drawText(x, y int, style ui.Style, text string) int {
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
	if lineIndex < 0 || lineIndex >= e.buffer.LineCount {
		return
	}

	line := e.buffer.Line(lineIndex)
	normalStyle := e.config.Theme.GetStyle(e.config.Theme.Normal)
	isCurrentLine := lineIndex == e.cursorY

	// If current line, we'll need to override background colors and pad the line
	var currentLineBg ui.Color
	if isCurrentLine {
		currentLineStyle := e.config.Theme.GetStyle(e.config.Theme.CurrentLine)
		_, currentLineBg, _ = currentLineStyle.Decompose()
		normalStyle = normalStyle.Background(currentLineBg)

		// Calculate how many spaces we need to fill the screen width
		width, _ := e.screen.Size()
		lineVisualWidth := e.visualWidth(line)
		remainingWidth := width - startX - lineVisualWidth
		if remainingWidth > 0 {
			// Append spaces to fill the width
			line = line + strings.Repeat(" ", remainingWidth)
		}
	}

	// If no highlighter is set up, just draw normally
	if e.highlighter == nil || !e.buffer.HasFile() {
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
	if e.cursorY >= e.buffer.LineCount {
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
	lineNumberStyle := e.config.Theme.GetStyle(e.config.Theme.LineNumber)
	totalLines := e.buffer.LineCount

	// Use cached line number width from buffer
	lineNumberWidth := e.buffer.LineNumberWidth

	// Clear the entire screen once
	e.screen.Clear()

	visibleHeight := e.getVisibleHeight()
	for i := 0; i < visibleHeight; i++ {
		lineIndex := e.topLine + i

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
	statusBarY := e.getStatusBarY()

	// Debug: ensure statusBarY is valid
	if statusBarY < 0 || statusBarY >= height {
		return
	}

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

		filename := e.buffer.DisplayName()
		x += e.drawText(x, statusBarY, statusBarStyle, filename)

		// Right side: indentation and line:col
		lines := e.buffer.Lines()

		// Use visual column position for display (accounting for tabs)
		visualCol := 1
		if e.cursorY < len(lines) {
			line := e.buffer.Line(e.cursorY)
			if e.cursorX <= len(line) {
				visualCol = e.visualWidth(line[:e.cursorX]) + 1
			}
		}

		// Combine indentation info and line:col
		var indentInfo string
		if e.buffer.Indentation.Type == config.IndentTabs {
			indentInfo = "Tabs"
		} else {
			indentInfo = fmt.Sprintf("Spaces:%d", e.buffer.Indentation.Width)
		}

		// Get current line indentation info
		lineIndent := e.getCurrentLineIndentInfo()
		var lineIndentDisplay string
		if lineIndent.Level > 0 {
			lineIndentDisplay = fmt.Sprintf("L%d", lineIndent.Level)
			if lineIndent.IsContinuation {
				lineIndentDisplay += "+C"
			}
		} else {
			lineIndentDisplay = "L0"
		}

		lineColInfo := fmt.Sprintf("%d:%d", e.cursorY+1, visualCol)
		rightInfo := fmt.Sprintf("%s  %s  %s", indentInfo, lineIndentDisplay, lineColInfo)

		// Calculate position for right-aligned text
		rightX := width - len(rightInfo)

		if rightX > x { // Only draw if there's space
			// Draw indentation info
			rightX += e.drawText(rightX, statusBarY, statusBarStyle, indentInfo)
			// Add spacing
			rightX += e.drawText(rightX, statusBarY, statusBarStyle, "  ")
			// Draw line indentation level
			rightX += e.drawText(rightX, statusBarY, statusBarStyle, lineIndentDisplay)
			// Add spacing
			rightX += e.drawText(rightX, statusBarY, statusBarStyle, "  ")
			// Draw line:col with different style
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
	if e.cursorY < e.buffer.LineCount-1 {
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
	if e.cursorY < e.buffer.LineCount-1 {
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
	maxY := e.buffer.LineCount - 1
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

// goToBeginning moves the cursor to the beginning of the file
func (e *Editor) goToBeginning() error {
	e.cursorY = 0
	e.cursorX = 0
	e.topLine = 0
	return nil
}

// goToEnd moves the cursor to the end of the file
func (e *Editor) goToEnd() error {
	lines := e.buffer.Lines()
	if len(lines) == 0 {
		e.cursorY = 0
		e.cursorX = 0
		return nil
	}

	// Move to last line
	e.cursorY = len(lines) - 1

	// Move to end of last line (but stay on last character in normal mode)
	lastLine := lines[e.cursorY]
	e.cursorX = len(lastLine)
	if e.cursorX > 0 && e.mode == commands.NormalMode {
		e.cursorX--
	}

	// Scroll to show the cursor
	e.scroll()

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

// openLineBelow opens a new line below the current line and enters insert mode
func (e *Editor) openLineBelow() error {
	// Move to end of current line
	lines := e.buffer.Lines()
	if e.cursorY < len(lines) {
		e.cursorX = len(lines[e.cursorY])
	}

	// Get absolute position at end of line
	pos := e.getAbsolutePosition()

	// Insert newline
	e.buffer.Insert(pos, "\n")

	// Move cursor to beginning of new line
	e.cursorY++
	e.cursorX = 0

	// Enter insert mode
	e.mode = commands.InsertMode

	// Mark buffer as modified (also updates line count)
	e.buffer.MarkModified()

	// Mark offsets as dirty
	e.offsetsDirty = true

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

// openLineAbove opens a new line above the current line and enters insert mode
func (e *Editor) openLineAbove() error {
	// Move to beginning of current line
	e.cursorX = 0

	// Get absolute position
	pos := e.getAbsolutePosition()

	// Insert newline before current position
	e.buffer.Insert(pos, "\n")

	// Move cursor to the newly created line above
	// (cursor stays at same Y position, which is now the new empty line)
	e.cursorX = 0

	// Enter insert mode
	e.mode = commands.InsertMode

	// Mark buffer as modified (also updates line count)
	e.buffer.MarkModified()

	// Mark offsets as dirty
	e.offsetsDirty = true

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	return nil
}

func (e *Editor) insertChar(ch rune) error {
	// Calculate the absolute position in the buffer
	pos := e.getAbsolutePosition()

	// Insert the character
	e.buffer.Insert(pos, string(ch))

	// Move cursor forward
	e.cursorX++

	// Mark buffer as modified (also updates line count)
	e.buffer.MarkModified()

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

	// Mark buffer as modified (also updates line count)
	e.buffer.MarkModified()

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
				e.buffer.Rope = buffer.NewRopeFromString(newContent)
				e.cursorY--
				e.cursorX = prevLineLen
				// Mark buffer as modified (also updates line count)
				e.buffer.MarkModified()
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
				e.buffer.Rope = buffer.NewRopeFromString(newContent)
				e.cursorX--
				// Mark buffer as modified (also updates line count)
				e.buffer.MarkModified()
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
	if e.highlighter != nil && e.buffer.HasFile() {
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
		// Update existing buffer's content
		bufferInfo.Rope = buffer.NewRopeFromString(content.String())
		bufferInfo.CursorX = 0
		bufferInfo.CursorY = 0
		bufferInfo.TopLine = 0
		bufferInfo.OffsetsDirty = true
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
	line := e.cursorY
	character := e.cursorX

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

