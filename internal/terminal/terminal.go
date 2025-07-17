package terminal

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"veda/internal/log"
	"veda/internal/ui"

	"golang.org/x/sys/unix"
)

var LL = log.NewLogger("TERMINAL")

// Terminal provides low-level terminal control with ANSI escape sequences
// and double buffering for flicker-free rendering.
type Terminal struct {
	mu sync.Mutex

	width  int
	height int

	// Double buffering
	frontBuffer []string  // What's currently displayed on screen
	backBuffer  []string  // What we're building for next frame

	// Cursor state
	cursorX       int
	cursorY       int
	cursorVisible bool
	cursorStyle   CursorStyle

	// Terminal state
	oldTermios *unix.Termios
	isRawMode  bool
	tty        *os.File  // Direct terminal device

	// Input handling
	inputChan chan ui.Event
	stopChan  chan struct{}
}

// CursorStyle represents the cursor appearance
type CursorStyle int

const (
	CursorStyleBlock CursorStyle = iota
	CursorStyleBar
	CursorStyleUnderline
)

// New creates a new Terminal instance
func New() (*Terminal, error) {
	// Get terminal size
	width, height, err := getTerminalSize()
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal size: %w", err)
	}

	LL.Debug("Terminal.New: Got terminal size: width=%d, height=%d", width, height)

	// Open the terminal device directly
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/tty: %w", err)
	}

	t := &Terminal{
		width:         width,
		height:        height,  // Use actual height
		frontBuffer:   make([]string, height),
		backBuffer:    make([]string, height),
		cursorVisible: true,
		cursorStyle:   CursorStyleBlock,
		inputChan:     make(chan ui.Event, 100),
		stopChan:      make(chan struct{}),
		tty:           tty,
	}

	// Initialize buffers with empty lines
	for i := range t.backBuffer {
		t.backBuffer[i] = ""
		// Initialize front buffer with a sentinel value to force initial render
		t.frontBuffer[i] = "\x00"
	}

	return t, nil
}

// Init initializes the terminal (enters raw mode, clears screen, etc.)
func (t *Terminal) Init() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Enter raw mode
	if err := t.enterRawMode(); err != nil {
		return fmt.Errorf("failed to enter raw mode: %w", err)
	}

	// Switch to alternate screen buffer, clear screen and hide cursor initially
	fmt.Fprint(t.tty, "\033[?1049h\033[2J\033[H\033[?25l")

	// Update terminal size after switching to alternate screen
	if width, height, err := getTerminalSize(); err == nil {
		LL.Debug("Terminal.Init: Got terminal size after alternate screen: width=%d, height=%d", width, height)
		t.width = width
		t.height = height  // Use actual height
		// Reinitialize buffers with correct size
		t.frontBuffer = make([]string, t.height)
		t.backBuffer = make([]string, t.height)
		for i := range t.backBuffer {
			t.backBuffer[i] = ""
			// Initialize front buffer with a sentinel value to force initial render
			t.frontBuffer[i] = "\x00"
		}
	} else {
		LL.Error("Terminal.Init: Failed to get terminal size: %v", err)
	}

	// Start input handler
	go t.inputHandler()

	// Handle terminal resize
	go t.resizeHandler()

	return nil
}

// Close restores the terminal to its original state
func (t *Terminal) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Stop input handler
	close(t.stopChan)

	// Show cursor and switch back to main screen buffer
	fmt.Fprint(t.tty, "\033[?25h\033[?1049l")

	// Clear screen
	fmt.Fprint(t.tty, "\033[2J\033[H")

	// Exit raw mode
	if t.isRawMode {
		t.exitRawMode()
	}

	// Close tty file
	if t.tty != nil {
		t.tty.Close()
	}
}

// Size returns the current terminal dimensions
func (t *Terminal) Size() (width, height int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.width, t.height
}

// Clear clears the back buffer
func (t *Terminal) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.backBuffer {
		t.backBuffer[i] = ""
	}
}

// SetLine sets a pre-rendered line in the back buffer
func (t *Terminal) SetLine(y int, line string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if y >= 0 && y < t.height {
		t.backBuffer[y] = line
	}
}

// SetCursor positions the cursor
func (t *Terminal) SetCursor(x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursorX = x
	t.cursorY = y
}

// SetCursorVisible sets cursor visibility
func (t *Terminal) SetCursorVisible(visible bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursorVisible = visible
}

// SetCursorStyle sets the cursor style
func (t *Terminal) SetCursorStyle(style CursorStyle) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursorStyle = style
}

// Show renders the screen with minimal updates to prevent flickering
func (t *Terminal) Show() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Don't render if we don't have a valid size
	if t.width <= 0 || t.height <= 0 {
		LL.Error("Terminal.Show: Invalid dimensions, skipping render")
		return
	}

	var output bytes.Buffer
	hasChanges := false

	// Hide cursor during screen update
	output.WriteString("\033[?25l")

	// Differential rendering - only update changed lines
	for y := 0; y < t.height && y < len(t.backBuffer); y++ {
		if t.backBuffer[y] != t.frontBuffer[y] {
			// Position cursor at beginning of line
			fmt.Fprintf(&output, "\033[%d;1H", y+1)
			output.WriteString(t.backBuffer[y])
			// Clear to end of line in case the new line is shorter
			output.WriteString("\033[K")
			t.frontBuffer[y] = t.backBuffer[y]
			hasChanges = true
		}
	}

	// Always update cursor position and visibility
	if t.cursorVisible {
		// Set cursor style
		switch t.cursorStyle {
		case CursorStyleBar:
			output.WriteString("\033[6 q") // Steady bar
		case CursorStyleUnderline:
			output.WriteString("\033[4 q") // Steady underline
		default:
			output.WriteString("\033[2 q") // Steady block
		}

		// Position cursor and make it visible
		fmt.Fprintf(&output, "\033[%d;%dH\033[?25h", t.cursorY+1, t.cursorX+1)
	}

	// Only write if there were changes or cursor needs update
	if hasChanges || t.cursorVisible {
		t.tty.Write(output.Bytes())
		t.tty.Sync() // Force flush
	}
}

// PollEvent returns the next input event (blocking)
func (t *Terminal) PollEvent() ui.Event {
	return <-t.inputChan
}

// EventChannel returns the underlying event channel
func (t *Terminal) EventChannel() <-chan ui.Event {
	return t.inputChan
}

// enterRawMode puts the terminal in raw mode
func (t *Terminal) enterRawMode() error {
	// Get current terminal settings
	oldState, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), tcGet)
	if err != nil {
		return err
	}
	t.oldTermios = oldState

	// Make a copy and modify it for raw mode
	newState := *oldState

	// Turn off echo, canonical mode, signals, and flow control
	newState.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN
	newState.Iflag &^= unix.IXON | unix.ICRNL | unix.BRKINT | unix.INPCK | unix.ISTRIP
	newState.Oflag &^= unix.OPOST
	newState.Cflag |= unix.CS8

	// Set minimum read of 1 byte with no timeout
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0

	// Apply the new settings
	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), tcSet, &newState); err != nil {
		return err
	}

	t.isRawMode = true
	return nil
}

// exitRawMode restores the terminal to its original state
func (t *Terminal) exitRawMode() {
	if t.oldTermios != nil {
		unix.IoctlSetTermios(int(os.Stdin.Fd()), tcSet, t.oldTermios)
		t.isRawMode = false
	}
}

// getTerminalSize returns the current terminal dimensions
func getTerminalSize() (width, height int, err error) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}


// handleResize updates the terminal dimensions
func (t *Terminal) handleResize() {
	width, height, err := getTerminalSize()
	if err != nil {
		return
	}

	LL.Debug("Got resize signal to %d:%d", width, height)

	t.mu.Lock()
	defer t.mu.Unlock()

	if width != t.width || height != t.height {
		t.width = width
		t.height = height

		// Resize buffers
		newFront := make([]string, height)
		newBack := make([]string, height)

		// Copy existing content
		for i := 0; i < height && i < len(t.frontBuffer); i++ {
			newFront[i] = t.frontBuffer[i]
			newBack[i] = t.backBuffer[i]
		}

		t.frontBuffer = newFront
		t.backBuffer = newBack

		// Send resize event
		t.inputChan <- &EventResize{Width: width, Height: height}
	}
}