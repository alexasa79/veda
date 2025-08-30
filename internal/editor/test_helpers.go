package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/events"
)

// Constants for common keys
const (
	KeyEscape = events.KeyEscape
)

// TestEditor wraps an Editor with test-friendly methods
type TestEditor struct {
	*Editor
	mockScreen *MockScreen
}

// NewTestEditor creates a new editor configured for testing
func NewTestEditor(width, height int) *TestEditor {
	mockScreen := NewMockScreen(width, height)
	editor, err := NewWithScreen(mockScreen)
	if err != nil {
		panic(err) // In tests, we panic on setup errors
	}

	te := &TestEditor{
		Editor:     editor,
		mockScreen: mockScreen,
	}

	// Ensure tests start with a default unnamed buffer as before, even if
	// production does not create one by default.
	if te.CurrentBuffer() == nil {
		// Create unnamed buffer and attach to active window
		buf := buffer.NewBuffer("", "[No Name]", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
		te.buffers["[No Name]"] = buf
		te.bufferOrder = append(te.bufferOrder, "[No Name]")
		if len(te.windows) == 0 {
			te.windows = []*Window{newWindow(buf)}
			te.activeWindow = 0
		} else if te.activeWindow >= 0 && te.activeWindow < len(te.windows) {
			if te.windows[te.activeWindow] == nil {
				te.windows[te.activeWindow] = newWindow(buf)
			} else {
				te.windows[te.activeWindow].addBufferToWindow(buf)
			}
		}
		te.SetCurrentBuffer(buf)
	}

	return te
}

// NewTestEditorWithContent creates a test editor with initial content
func NewTestEditorWithContent(width, height int, content string) *TestEditor {
	te := NewTestEditor(width, height)
	te.SetCurrentBuffer(buffer.NewBuffer(content, "[Test]", te.config.DefaultIndentType, te.config.DefaultIndentWidth))
	// Update iterators to point to the new buffer
	te.CurrentBuffer().ViewIterator = buffer.NewBufferIterator(te.CurrentBuffer())
	te.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(te.CurrentBuffer())

	// Set up content change callback for search match updates
	te.CurrentBuffer().OnContentChange = func() {
		te.updateSearchMatchesOnContentChange()
	}

	// Add the buffer to the buffers map so buffer management functions work
	te.buffers[te.CurrentBuffer().FilePath] = te.CurrentBuffer()
	// Update buffer order if the file path isn't already there
	found := false
	for _, path := range te.bufferOrder {
		if path == te.CurrentBuffer().FilePath {
			found = true
			break
		}
	}
	if !found {
		te.bufferOrder = append(te.bufferOrder, te.CurrentBuffer().FilePath)
	}

	// Ensure the active window renders this new buffer in unified window mode
	// The editor initializes with a window bound to the initial empty buffer.
	// After swapping te.buffer, also attach it to the active window so render() paints it.
	if len(te.windows) == 0 {
		// Create a default window for this buffer and layout
		te.windows = []*Window{newWindow(te.CurrentBuffer())}
		te.activeWindow = 0
		te.recomputeWindowsLayout()
	} else {
		// Attach the test buffer to the currently active window
		aw := te.activeWindow
		if aw < 0 || aw >= len(te.windows) {
			aw = 0
			te.activeWindow = 0
		}
		if te.windows[aw] == nil {
			te.windows[aw] = newWindow(te.CurrentBuffer())
		} else {
			te.windows[aw].Buffer = te.CurrentBuffer()
		}
		// Sync per-window cursor/view state from the buffer so status/render are consistent
		te.windows[aw].ViewLine = te.CurrentBuffer().ViewIterator.LineNumber()
		te.windows[aw].CursorLine = te.CurrentBuffer().CursorIterator.LineNumber()
		te.windows[aw].CursorX = te.CurrentBuffer().CursorX
	}

	return te
}

// ExecuteCommand executes a command by name
func (te *TestEditor) ExecuteCommand(command string) error {
	// Use executeWithRepeatCount to properly handle multiple cursors
	te.repeatCount = 1
	return te.executeWithRepeatCount(command)
}

// InsertChar simulates inserting a character in insert mode
func (te *TestEditor) InsertChar(ch rune) error {
	return te.insertChar(ch)
}

// TypeKeys simulates typing a sequence of keys in normal mode
func (te *TestEditor) TypeKeys(keys string) {
	// Quick handling for named arrow keys passed as full strings
	switch keys {
	case "left":
		te.PressKey(events.KeyLeft, 0)
		return
	case "right":
		te.PressKey(events.KeyRight, 0)
		return
	case "up":
		te.PressKey(events.KeyUp, 0)
		return
	case "down":
		te.PressKey(events.KeyDown, 0)
		return
	}

	// If the input looks like a key sequence (e.g., "space v", "g d"),
	// parse it using the same parser the keybindings use so multi-key
	// sequences are recognized correctly.
	if fields := strings.Fields(keys); len(fields) > 1 {
		if seq, err := commands.ParseKeySequence(keys); err == nil {
			for _, k := range seq {
				ev := &events.KeyEvent{Key: k.Key, Rune: k.Rune, Modifiers: k.Mod}
				// Delegate to the main event handler logic
				te.handleEvent(ev)
			}
			return
		}
		// If parsing fails, fall through to raw typing below
	}

	// Raw typing: send each rune as a key event, mapping common control chars
	for _, ch := range keys {
		var ev *events.KeyEvent
		switch ch {
		case '\n':
			ev = &events.KeyEvent{Key: events.KeyEnter}
		case '\t':
			ev = &events.KeyEvent{Key: events.KeyTab}
		default:
			ev = &events.KeyEvent{Key: events.KeyRune, Rune: ch}
		}
		// Delegate to the main event handler logic
		te.handleEvent(ev)
	}
}

// PressKey simulates pressing a special key
func (te *TestEditor) PressKey(key events.Key, modifiers events.ModMask) {
	ev := &events.KeyEvent{
		Key:       key,
		Modifiers: modifiers,
	}
	// In terminal mode, route events through the main event loop to honor
	// terminal-specific handling (e.g., ESC exits terminal mode)
	if te.mode == commands.TerminalMode {
		te.handleEvent(ev)
		return
	}
	// Handle cursor mode specially
	if te.mode == commands.CursorMode {
		te.handleCursorMode(ev)
	} else if te.mode == commands.NormalMode && key == events.KeyEscape {
		// Route Escape through the main event handler so keymap binding runs
		te.handleEvent(ev)
		return
	} else if te.mode == commands.NormalMode && te.hasPendingOperation() {
		// Handle pending operations for special keys
		keyStr := ""
		switch key {
		case events.KeyLeft:
			keyStr = "left"
		case events.KeyRight:
			keyStr = "right"
		case events.KeyUp:
			keyStr = "up"
		case events.KeyDown:
			keyStr = "down"
		}

		if keyStr != "" {
			te.handlePendingOperation(keyStr)
		} else if command, found := te.keyMapper.ProcessKey(ev, te.mode); found {
			// Use executeWithRepeatCount to properly handle multiple cursors
			te.repeatCount = 1
			te.executeWithRepeatCount(command)
		}
	} else if command, found := te.keyMapper.ProcessKey(ev, te.mode); found {
		// Use executeWithRepeatCount to properly handle multiple cursors
		te.repeatCount = 1
		te.executeWithRepeatCount(command)
	}
}

// GetCursorPosition returns the current cursor position
func (te *TestEditor) GetCursorPosition() (x, y int) {
	return te.CurrentBuffer().CursorX, te.CurrentBuffer().CursorIterator.LineNumber()
}

// SetCursorPosition sets the cursor to a specific position
func (te *TestEditor) SetCursorPosition(x, y int) {
	te.CurrentBuffer().CursorX = x

	// Reset cursor iterator to beginning
	te.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(te.CurrentBuffer())

	// Move to target line
	for i := 0; i < y && !te.CurrentBuffer().CursorIterator.IsEOF(); i++ {
		te.CurrentBuffer().CursorIterator = te.CurrentBuffer().CursorIterator.Next()
	}

	// Ensure cursor X is within bounds
	if !te.CurrentBuffer().CursorIterator.IsEOF() {
		line := te.CurrentBuffer().CursorIterator.Line()
		if te.CurrentBuffer().CursorX > len(line) {
			te.CurrentBuffer().CursorX = len(line)
		}
		// Apply normal mode adjustment - in vim, cursor can't be past the last character
		if te.mode == commands.NormalMode && te.CurrentBuffer().CursorX == len(line) && len(line) > 0 {
			te.CurrentBuffer().CursorX--
		}
	}
}

// GetBufferContent returns the entire buffer content as a string
func (te *TestEditor) GetBufferContent() string {
	return te.CurrentBuffer().Rope.String()
}

// GetScreenContent returns what's visible on the screen
func (te *TestEditor) GetScreenContent() []string {
	// Force a full render using the editor's unified path so tests see exactly what users see
	te.Editor.render()

	lines := make([]string, 0)
	_, height := te.screen.Size()
	// Get all lines including status bar
	for y := 0; y < height; y++ {
		lines = append(lines, te.mockScreen.GetLine(y))
	}
	return lines
}

// GetScreenCursorPosition returns where the cursor is shown on screen
func (te *TestEditor) GetScreenCursorPosition() (x, y int) {
	return te.mockScreen.GetCursorPosition()
}

// GetVisualCursorPosition returns the visual cursor position considering line wrapping
func (te *TestEditor) GetVisualCursorPosition() (x, y int) {
	return te.GetCursorVisualPosition()
}

// AssertCursorAt checks if the cursor is at the expected position
func AssertCursorAt(t *testing.T, te *TestEditor, expectedX, expectedY int) {
	t.Helper()
	x, y := te.GetCursorPosition()
	if x != expectedX || y != expectedY {
		t.Errorf("Expected cursor at (%d, %d), but got (%d, %d)", expectedX, expectedY, x, y)
	}
}

// AssertBufferEquals checks if the buffer content matches expected
func AssertBufferEquals(t *testing.T, te *TestEditor, expected string) {
	t.Helper()
	actual := te.GetBufferContent()
	if actual != expected {
		t.Errorf("Expected buffer content:\n%q\nBut got:\n%q", expected, actual)
	}
}

// AssertScreenLineEquals checks if a specific screen line matches expected
func AssertScreenLineEquals(t *testing.T, te *TestEditor, lineNum int, expected string) {
	t.Helper()
	lines := te.GetScreenContent()
	if lineNum >= len(lines) {
		t.Errorf("Line %d does not exist (only %d lines on screen)", lineNum, len(lines))
		return
	}

	actual := lines[lineNum]
	if actual != expected {
		t.Errorf("Expected line %d to be:\n%q\nBut got:\n%q", lineNum, expected, actual)
	}
}
