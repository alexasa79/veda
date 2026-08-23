package terminal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"
	"veda/internal/events"
	"veda/internal/log"

	"golang.org/x/sys/unix"
)

var LL = log.NewLogger("TERMINAL")

// Terminal provides low-level terminal control with ANSI escape sequences
// and double buffering for flicker-free rendering.
type Terminal struct {
	mu sync.Mutex

	width           int
	height          int
	cellPixelWidth  int
	cellPixelHeight int

	// Double buffering
	frontBuffer        []string // What's currently displayed on screen
	backBuffer         []string // What we're building for next frame
	pendingScroll      *scrollOperation
	pendingKittyImages []KittyImagePlacement
	visibleKittyImages map[string]KittyImagePlacement
	kittyImageIDs      map[string]uint32
	kittyUsedImageIDs  map[uint32]string
	kittyTransmitted   map[string]bool

	// Cursor state
	cursorX       int
	cursorY       int
	cursorVisible bool
	cursorStyle   CursorStyle

	// Terminal state
	oldTermios *unix.Termios
	isRawMode  bool
	tty        *os.File // Direct terminal device

	// Input handling
	inputChan chan events.Event
	stopChan  chan struct{}

	// OSC 52 clipboard support
	osc52ResponseCh chan string // Channel to receive OSC 52 responses
	osc52Waiting    bool        // True when we're waiting for an OSC 52 response
	osc52Mu         sync.Mutex  // Protects osc52Waiting
}

type scrollOperation struct {
	top    int
	bottom int
	lines  int // Positive scrolls up; negative scrolls down.
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
	cellPixelWidth, cellPixelHeight := getTerminalCellSize()

	LL.Debug("Terminal.New: Got terminal size: width=%d, height=%d", width, height)

	// Open the terminal device directly
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/tty: %w", err)
	}

	t := &Terminal{
		width:           width,
		height:          height, // Use actual height
		cellPixelWidth:  cellPixelWidth,
		cellPixelHeight: cellPixelHeight,
		frontBuffer:     make([]string, height),
		backBuffer:      make([]string, height),
		cursorVisible:   true,
		cursorStyle:     CursorStyleBlock,
		inputChan:       make(chan events.Event, 100),
		stopChan:        make(chan struct{}),
		tty:             tty,
		osc52ResponseCh: make(chan string, 1),
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
	// Also enable bracketed paste mode (CSI ? 2004 h)
	fmt.Fprint(t.tty, "\033[?1049h\033[2J\033[H\033[?25l\033[?2004h")

	// Update terminal size after switching to alternate screen
	if width, height, err := getTerminalSize(); err == nil {
		LL.Debug("Terminal.Init: Got terminal size after alternate screen: width=%d, height=%d", width, height)
		t.width = width
		t.height = height // Use actual height
		t.cellPixelWidth, t.cellPixelHeight = getTerminalCellSize()
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

	// Disable bracketed paste mode (CSI ? 2004 l), show cursor and switch back to main screen buffer
	fmt.Fprint(t.tty, "\033[?2004l\033[?25h\033[?1049l")

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

func (t *Terminal) CellPixelSize() (width, height int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cellPixelWidth, t.cellPixelHeight
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

// QueueScrollRegion schedules a terminal-side scroll for the next Show call.
// top and bottom are zero-based, inclusive rows. Positive lines scroll up;
// negative lines scroll down.
func (t *Terminal) QueueScrollRegion(top, bottom, lines int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if lines == 0 || top < 0 || bottom >= t.height || top >= bottom {
		return
	}
	count := lines
	if count < 0 {
		count = -count
	}
	if count > bottom-top {
		return
	}

	t.pendingScroll = &scrollOperation{top: top, bottom: bottom, lines: lines}
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

	if t.pendingScroll != nil {
		op := *t.pendingScroll
		fmt.Fprintf(&output, "\033[%d;%dr", op.top+1, op.bottom+1)
		if op.lines > 0 {
			fmt.Fprintf(&output, "\033[%dS", op.lines)
		} else {
			fmt.Fprintf(&output, "\033[%dT", -op.lines)
		}
		output.WriteString("\033[r")
		applyScrollToFrontBuffer(t.frontBuffer, op)
		t.pendingScroll = nil
		hasChanges = true
	}

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

	if imageCommands := t.updateKittyImages(); imageCommands != "" {
		output.WriteString(imageCommands)
		hasChanges = true
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

func applyScrollToFrontBuffer(front []string, op scrollOperation) {
	if op.lines == 0 || op.top < 0 || op.bottom >= len(front) || op.top >= op.bottom {
		return
	}

	count := op.lines
	if count < 0 {
		count = -count
	}
	if count > op.bottom-op.top {
		return
	}

	if op.lines > 0 {
		copy(front[op.top:op.bottom-count+1], front[op.top+count:op.bottom+1])
		for row := op.bottom - count + 1; row <= op.bottom; row++ {
			front[row] = "\x00"
		}
		return
	}

	copy(front[op.top+count:op.bottom+1], front[op.top:op.bottom-count+1])
	for row := op.top; row < op.top+count; row++ {
		front[row] = "\x00"
	}
}

// PollEvent returns the next input event (blocking)
func (t *Terminal) PollEvent() events.Event {
	return <-t.inputChan
}

// EventChannel returns the underlying event channel
func (t *Terminal) EventChannel() <-chan events.Event {
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

const (
	defaultCellPixelWidth  = 8
	defaultCellPixelHeight = 16
)

func getTerminalCellSize() (width, height int) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return defaultCellPixelWidth, defaultCellPixelHeight
	}
	width = int(ws.Xpixel) / int(ws.Col)
	height = int(ws.Ypixel) / int(ws.Row)
	if width < 1 || height < 1 {
		return defaultCellPixelWidth, defaultCellPixelHeight
	}
	return width, height
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
	t.cellPixelWidth, t.cellPixelHeight = getTerminalCellSize()

	if width != t.width || height != t.height {
		t.width = width
		t.height = height

		// Resize buffers. Invalidate frontBuffer with sentinels to force
		// a full redraw — the terminal emulator has already cleared/reflowed
		// its display, so frontBuffer no longer reflects what's on screen.
		newFront := make([]string, height)
		newBack := make([]string, height)
		for i := range newFront {
			newFront[i] = "\x00"
		}

		t.frontBuffer = newFront
		t.backBuffer = newBack

		// Send resize event
		t.inputChan <- events.NewResizeEvent(width, height)
	}
}

// OSC 52 clipboard support

// CopyToClipboard copies text to the system clipboard using OSC 52
func (t *Terminal) CopyToClipboard(text string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	seq := fmt.Sprintf("\033]52;c;%s\007", encoded)
	_, err := t.tty.Write([]byte(seq))
	return err
}

// QueryClipboard queries the system clipboard using OSC 52 and returns the content.
// This sends the query sequence and waits for the terminal to respond.
// timeout specifies how long to wait for the response.
func (t *Terminal) QueryClipboard(timeout time.Duration) (string, error) {
	LL.Debug("QueryClipboard: starting, timeout=%v", timeout)

	t.osc52Mu.Lock()
	t.osc52Waiting = true
	t.osc52Mu.Unlock()

	defer func() {
		t.osc52Mu.Lock()
		t.osc52Waiting = false
		t.osc52Mu.Unlock()
		LL.Debug("QueryClipboard: cleared waiting flag")
	}()

	// Clear any pending response
	select {
	case <-t.osc52ResponseCh:
		LL.Debug("QueryClipboard: cleared pending response")
	default:
	}

	// Send the OSC 52 query
	LL.Debug("QueryClipboard: sending query")
	_, err := t.tty.Write([]byte("\033]52;c;?\007"))
	if err != nil {
		return "", fmt.Errorf("failed to send OSC 52 query: %w", err)
	}

	// Wait for response with timeout
	select {
	case response := <-t.osc52ResponseCh:
		LL.Debug("QueryClipboard: received response, len=%d", len(response))
		return response, nil
	case <-time.After(timeout):
		LL.Debug("QueryClipboard: timeout")
		return "", fmt.Errorf("timeout waiting for OSC 52 response")
	}
}

// handleOSC52Response processes an OSC 52 response and sends it to the response channel.
// This is called from the input parser when an OSC 52 response is detected.
func (t *Terminal) handleOSC52Response(payload string) {
	t.osc52Mu.Lock()
	waiting := t.osc52Waiting
	t.osc52Mu.Unlock()

	if waiting {
		// Decode the base64 payload
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			LL.Error("Failed to decode OSC 52 response: %v", err)
			return
		}
		select {
		case t.osc52ResponseCh <- string(decoded):
		default:
			// Channel full, discard
		}
	}
}

// IsWaitingForOSC52 returns true if we're waiting for an OSC 52 response.
// This can be used by the input parser to route OSC 52 responses correctly.
func (t *Terminal) IsWaitingForOSC52() bool {
	t.osc52Mu.Lock()
	defer t.osc52Mu.Unlock()
	return t.osc52Waiting
}
