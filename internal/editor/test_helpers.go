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
	
	return &TestEditor{
		Editor:     editor,
		mockScreen: mockScreen,
	}
}

// NewTestEditorWithContent creates a test editor with initial content
func NewTestEditorWithContent(width, height int, content string) *TestEditor {
	te := NewTestEditor(width, height)
	te.buffer = buffer.NewBuffer(content, "[Test]", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	// Update iterators to point to the new buffer
	te.buffer.ViewIterator = buffer.NewBufferIterator(te.buffer)
	te.buffer.CursorIterator = buffer.NewBufferIterator(te.buffer)
	
	// Set up content change callback for search match updates
	te.buffer.OnContentChange = func() {
		te.updateSearchMatchesOnContentChange()
	}
	
	// Add the buffer to the buffers map so buffer management functions work
	te.buffers[te.buffer.FilePath] = te.buffer
	// Update buffer order if the file path isn't already there
	found := false
	for _, path := range te.bufferOrder {
		if path == te.buffer.FilePath {
			found = true
			break
		}
	}
	if !found {
		te.bufferOrder = append(te.bufferOrder, te.buffer.FilePath)
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
    // Quick handling for some named special keys passed as full strings
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
    case "Home", "home":
        te.PressKey(events.KeyHome, 0)
        return
    case "End", "end":
        te.PressKey(events.KeyEnd, 0)
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
        if te.hasPendingOperation() {
            te.cancelPendingOperation()
        } else if len(te.virtualCursors) > 0 {
            // Clear virtual cursors on escape in normal mode
            te.clearVirtualCursors()
        }
        // In normal mode, ESC should hide key hints popup
        te.keyHintsPopup.Hide()
        // Also clear any pending key sequence to match real event path
        te.keyMapper.ResetPending()
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
    // In terminal mode, compute the on-screen cursor position similarly to updateCursor
    if te.mode == commands.TerminalMode && te.buffer != nil && te.buffer.IsInternal {
        if ts := te.terminalSessions[te.buffer.FilePath]; ts != nil && ts.term != nil {
            curX, curY := ts.term.Cursor()
            bannerOffset := 0
            if ts.Banner != "" { bannerOffset = 1 }
            globalRow := bannerOffset + len(ts.term.scrollback) + curY
            viewTop := te.buffer.ViewIterator.LineNumber()
            relRow := globalRow - viewTop
            contentTop := te.getContentStartY()
            visible := te.getVisibleHeight()
            if relRow < 0 { relRow = 0 }
            if relRow >= visible { relRow = visible - 1 }
            screenY := contentTop + relRow
            screenX := te.buffer.LineNumberWidth + curX
            return screenX, screenY
        }
        // Fallback to screen-reported position
        return te.mockScreen.GetCursorPosition()
    }
    return te.buffer.CursorX, te.buffer.CursorIterator.LineNumber()
}

// SetCursorPosition sets the cursor to a specific position
func (te *TestEditor) SetCursorPosition(x, y int) {
	te.buffer.CursorX = x
	
	// Reset cursor iterator to beginning
	te.buffer.CursorIterator = buffer.NewBufferIterator(te.buffer)
	
	// Move to target line
	for i := 0; i < y && !te.buffer.CursorIterator.IsEOF(); i++ {
		te.buffer.CursorIterator = te.buffer.CursorIterator.Next()
	}
	
	// Ensure cursor X is within bounds
	if !te.buffer.CursorIterator.IsEOF() {
		line := te.buffer.CursorIterator.Line()
		if te.buffer.CursorX > len(line) {
			te.buffer.CursorX = len(line)
		}
		// Apply normal mode adjustment - in vim, cursor can't be past the last character
		if te.mode == commands.NormalMode && te.buffer.CursorX == len(line) && len(line) > 0 {
			te.buffer.CursorX--
		}
	}
}

// GetBufferContent returns the entire buffer content as a string
func (te *TestEditor) GetBufferContent() string {
	return te.buffer.Rope.String()
}

// GetScreenContent returns what's visible on the screen
func (te *TestEditor) GetScreenContent() []string {
	// Force a render
	te.drawEditor()
	te.drawErrorLine()
	te.drawStatusBar()
	
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