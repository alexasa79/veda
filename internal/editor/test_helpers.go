package editor

import (
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
	return te.commandRegistry.Execute(command, nil)
}

// InsertChar simulates inserting a character in insert mode
func (te *TestEditor) InsertChar(ch rune) error {
	return te.insertChar(ch)
}

// TypeKeys simulates typing a sequence of keys in normal mode
func (te *TestEditor) TypeKeys(keys string) {
	for _, ch := range keys {
		ev := &events.KeyEvent{
			Key:  events.KeyRune,
			Rune: ch,
		}
		// Simulate the key handling logic
		if te.mode == commands.NormalMode && te.hasPendingOperation() {
			te.handlePendingOperation(string(ch))
		} else if command, found := te.keyMapper.ProcessKey(ev, te.mode); found {
			te.commandRegistry.Execute(command, nil)
		}
	}
}

// PressKey simulates pressing a special key
func (te *TestEditor) PressKey(key events.Key, modifiers events.ModMask) {
	ev := &events.KeyEvent{
		Key:       key,
		Modifiers: modifiers,
	}
	// Handle escape for pending operations
	if te.mode == commands.NormalMode && te.hasPendingOperation() && key == events.KeyEscape {
		te.cancelPendingOperation()
	} else if command, found := te.keyMapper.ProcessKey(ev, te.mode); found {
		te.commandRegistry.Execute(command, nil)
	}
}

// GetCursorPosition returns the current cursor position
func (te *TestEditor) GetCursorPosition() (x, y int) {
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