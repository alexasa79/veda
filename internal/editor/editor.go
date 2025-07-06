package editor

import (
	"vidya/internal/buffer"
	"vidya/internal/config"
	"vidya/internal/proxy"
	"vidya/internal/utils"
	"vidya/internal/widgets"
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Editor represents the state of the text editor.
type Editor struct {
	screen      tcell.Screen
	buffer      *buffer.Buffer
	config      *config.Config
	cursorX     int
	cursorY     int
	topLine     int
	commandMode bool
	commandLine *widgets.LineEditor
	proxyClient *proxy.Client
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

	// Create a larger buffer for testing scrolling
	var longText string
	for i := range 100 {
		longText += fmt.Sprintf("This is line %d\n", i)
	}

	// Initialize command line widget (will be positioned at bottom of screen)
	width, height := screen.Size()
	commandLine := widgets.NewLineEditor(0, height-1, width)
	commandLine.SetStyle(config.Theme.GetStyle(config.Theme.CommandLine))

	// Start proxy client
	proxyClient, err := proxy.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to start proxy client: %v", err)
	}

	return &Editor{
		screen:      screen,
		buffer:      buffer.NewBufferFromString(longText),
		config:      config,
		cursorX:     0,
		cursorY:     0,
		topLine:     0,
		commandMode: false,
		commandLine: commandLine,
		proxyClient: proxyClient,
	}, nil
}

// Run starts the editor's main event loop.
func (e *Editor) Run() {
	// Event loop
	for {
		e.drawEditor()
		e.drawStatusBar()

		// Show cursor at appropriate position
		if e.commandMode {
			// Show cursor in command line
			cursorX, cursorY := e.commandLine.GetCursorPosition()
			e.screen.ShowCursor(cursorX, cursorY)
		} else {
			// Show cursor in editor
			e.screen.ShowCursor(e.cursorX, e.cursorY-e.topLine)
		}

		e.screen.Show()

		ev := e.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if e.commandMode {
				// Handle command mode events
				e.handleCommandMode(ev)
			} else {
				// Handle normal mode events
				e.handleNormalMode(ev)
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
	e.commandMode = true
	e.commandLine.SetText(":")
	// Position cursor after the colon
	e.commandLine.SetCursor(1)
}

// exitCommandMode switches back to normal mode
func (e *Editor) exitCommandMode() {
	e.commandMode = false
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
func (e *Editor) drawText(x, y int, style tcell.Style, text string) {
	for i, r := range text {
		e.screen.SetContent(x+i, y, r, nil, style)
	}
}

func (e *Editor) drawEditor() {
	width, height := e.screen.Size()
	for i := 0; i < height-2; i++ {
		for j := 0; j < width; j++ {
			e.screen.SetContent(j, i, ' ', nil, e.config.Theme.GetStyle(e.config.Theme.Normal))
		}
		line := e.buffer.Line(e.topLine + i)
		e.drawText(0, i, e.config.Theme.GetStyle(e.config.Theme.Normal), line)
	}
}

func (e *Editor) drawStatusBar() {
	width, height := e.screen.Size()
	statusBarStyle := e.config.Theme.GetStyle(e.config.Theme.StatusBar)

	// Clear the status bar lines
	for i := 0; i < width; i++ {
		e.screen.SetContent(i, height-2, ' ', nil, statusBarStyle)
		e.screen.SetContent(i, height-1, ' ', nil, statusBarStyle)
	}

	if e.commandMode {
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
	fuzzyListWidget.SetTitle("Files and Directories")

	// Get recursive directory listing from proxy
	files, err := e.proxyClient.ListDirectoryRecursive(".")
	if err != nil {
		fuzzyListWidget.SetContent([]string{"Error: " + err.Error()})
	} else {
		var lines []string
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
			
			icon := utils.GetFileIcon(file.Name, file.Type)
			line := fmt.Sprintf("%s %s", icon, file.Path)
			lines = append(lines, line)
		}
		fuzzyListWidget.SetContent(lines)
	}

	// Show the widget with a background render function
	fuzzyListWidget.Show(e.screen, func() {
		e.drawEditor()
		e.drawStatusBar()
	})
}
