package terminal

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"golang.org/x/sys/unix"
)

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

	// Input handling
	inputChan chan Event
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

	t := &Terminal{
		width:         width,
		height:        height,
		frontBuffer:   make([]string, height),
		backBuffer:    make([]string, height),
		cursorVisible: true,
		cursorStyle:   CursorStyleBlock,
		inputChan:     make(chan Event, 100),
		stopChan:      make(chan struct{}),
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
	fmt.Print("\033[?1049h\033[2J\033[H\033[?25l")

	// Update terminal size after switching to alternate screen
	if width, height, err := getTerminalSize(); err == nil {
		t.width = width
		t.height = height
		// Reinitialize buffers with correct size
		t.frontBuffer = make([]string, height)
		t.backBuffer = make([]string, height)
		for i := range t.backBuffer {
			t.backBuffer[i] = ""
			// Initialize front buffer with a sentinel value to force initial render
			t.frontBuffer[i] = "\x00"
		}
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
	fmt.Print("\033[?25h\033[?1049l")

	// Clear screen
	fmt.Print("\033[2J\033[H")

	// Exit raw mode
	if t.isRawMode {
		t.exitRawMode()
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

// Show renders the entire screen
func (t *Terminal) Show() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	var output bytes.Buffer
	
	// Clear screen and move to home position
	output.WriteString("\033[2J\033[H")
	
	// Render all lines with newlines
	for y, line := range t.backBuffer {
		if y > 0 {
			output.WriteString("\n")
		}
		output.WriteString(line)
	}
	
	// Single write to stdout
	os.Stdout.Write(output.Bytes())
	os.Stdout.Sync() // Force flush
}

// PollEvent returns the next input event (blocking)
func (t *Terminal) PollEvent() Event {
	return <-t.inputChan
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