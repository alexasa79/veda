package editor

import (
	"veda/internal/events"
	"veda/internal/ui"
)

// Cell represents a single cell on the screen
type Cell struct {
	Rune  rune
	Style ui.Style
}

// MockScreen implements Screen interface for testing.
// It models the real Terminal's double-buffering: cells is the back buffer
// (written by SetContent), displayed is what GetLine reads from, and
// frontLines tracks what Show() last flushed (for differential rendering).
type MockScreen struct {
	width         int
	height        int
	cells         map[int]map[int]Cell // [y][x]Cell — back buffer
	displayed     map[int]string       // what's "on screen" — GetLine reads this
	frontLines    map[int]string       // what Show thinks is on screen — for diffing
	cursorX       int
	cursorY       int
	cursorVisible bool
	cursorStyle   ui.CursorStyle
	events        []events.Event
	eventIndex    int
	defaultStyle  ui.Style
	eventChan     chan events.Event
}

// NewMockScreen creates a new mock screen with the given dimensions
func NewMockScreen(width, height int) *MockScreen {
	return &MockScreen{
		width:        width,
		height:       height,
		cells:        make(map[int]map[int]Cell),
		displayed:    make(map[int]string),
		frontLines:   make(map[int]string),
		events:       make([]events.Event, 0),
		defaultStyle: ui.StyleDefault,
		eventChan:    make(chan events.Event, 100),
	}
}

// Init initializes the mock screen
func (s *MockScreen) Init() error {
	return nil
}

// Fini finalizes the mock screen
func (s *MockScreen) Fini() {
	// No-op for testing
}

// Clear clears the screen
func (s *MockScreen) Clear() {
	s.cells = make(map[int]map[int]Cell)
}

// Show performs differential rendering like the real Terminal:
// only lines whose content changed since the last Show are flushed to
// the displayed buffer. This models the frontBuffer/backBuffer diffing
// in internal/terminal/terminal.go so tests can catch stale-buffer bugs.
func (s *MockScreen) Show() {
	for y := 0; y < s.height; y++ {
		line := s.buildLine(y)
		if line != s.frontLines[y] {
			s.displayed[y] = line
			s.frontLines[y] = line
		}
	}
}

// Size returns the screen dimensions
func (s *MockScreen) Size() (width, height int) {
	return s.width, s.height
}

// SetStyle sets the default style
func (s *MockScreen) SetStyle(style ui.Style) {
	s.defaultStyle = style
}

// SetContent sets the content at the given position
func (s *MockScreen) SetContent(x, y int, mainc rune, combc []rune, style ui.Style) {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return
	}

	if s.cells[y] == nil {
		s.cells[y] = make(map[int]Cell)
	}

	s.cells[y][x] = Cell{
		Rune:  mainc,
		Style: style,
	}
}

// ShowCursor shows the cursor at the given position
func (s *MockScreen) ShowCursor(x, y int) {
	s.cursorX = x
	s.cursorY = y
	s.cursorVisible = true
}

// SetCursorStyle sets the cursor style
func (s *MockScreen) SetCursorStyle(style ui.CursorStyle) {
	s.cursorStyle = style
}

// PollEvent returns the next event in the queue
func (s *MockScreen) PollEvent() events.Event {
	if s.eventIndex >= len(s.events) {
		// Return nil to indicate no more events
		return nil
	}

	event := s.events[s.eventIndex]
	s.eventIndex++
	return event
}

// EventChannel returns the underlying event channel
func (s *MockScreen) EventChannel() <-chan events.Event {
	return s.eventChan
}

// Testing-specific methods

// GetCursorPosition returns the current cursor position
func (s *MockScreen) GetCursorPosition() (x, y int) {
	return s.cursorX, s.cursorY
}

// IsCursorVisible returns whether the cursor is visible
func (s *MockScreen) IsCursorVisible() bool {
	return s.cursorVisible
}

// GetCursorStyle returns the current cursor style
func (s *MockScreen) GetCursorStyle() ui.CursorStyle {
	return s.cursorStyle
}

// GetCell returns the content at a specific position
func (s *MockScreen) GetCell(x, y int) (rune, ui.Style) {
	if row, ok := s.cells[y]; ok {
		if cell, ok := row[x]; ok {
			return cell.Rune, cell.Style
		}
	}
	return ' ', s.defaultStyle
}

// buildLine builds the trimmed string for row y from the cells back buffer.
func (s *MockScreen) buildLine(y int) string {
	row, ok := s.cells[y]
	if !ok {
		return ""
	}
	line := make([]rune, s.width)
	for x := 0; x < s.width; x++ {
		if cell, ok := row[x]; ok {
			line[x] = cell.Rune
		} else {
			line[x] = ' '
		}
	}
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] != ' ' {
			return string(line[:i+1])
		}
	}
	return ""
}

// GetLine returns what is currently "displayed" on screen for row y.
// This reads from the displayed buffer, which is only updated by Show()
// through differential rendering — matching the real Terminal behavior.
func (s *MockScreen) GetLine(y int) string {
	return s.displayed[y]
}

// AddEvent adds an event to the event queue for testing
func (s *MockScreen) AddEvent(ev events.Event) {
	s.events = append(s.events, ev)
}

// AddKeyEvent adds a key event to the event queue
func (s *MockScreen) AddKeyEvent(key events.Key, r rune, mod events.ModMask) {
	ev := &events.KeyEvent{
		Key:       key,
		Rune:      r,
		Modifiers: mod,
	}
	s.AddEvent(ev)
}

// GetContent returns the entire screen content as a map
func (s *MockScreen) GetContent() map[int]map[int]Cell {
	return s.cells
}

// SetSize changes the screen dimensions and sends a resize event.
// It clears the displayed buffer (the real terminal emulator clears its
// screen on resize) but keeps frontLines as-is (modeling Terminal.handleResize
// which copies the old frontBuffer without invalidation).
func (s *MockScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
	// Terminal emulator clears the display on resize
	s.displayed = make(map[int]string)
	// Invalidate frontLines so Show() re-flushes every line
	s.frontLines = make(map[int]string)
	// Send a resize event through the event channel
	resizeEvent := events.NewResizeEvent(width, height)
	select {
	case s.eventChan <- resizeEvent:
	default:
		// If channel is full, ignore
	}
}
