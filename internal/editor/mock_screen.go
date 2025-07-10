package editor

import (
	"github.com/gdamore/tcell/v2"
)

// Cell represents a single cell on the screen
type Cell struct {
	Rune  rune
	Style tcell.Style
}

// MockScreen implements Screen interface for testing
type MockScreen struct {
	width         int
	height        int
	cells         map[int]map[int]Cell // [y][x]Cell
	cursorX       int
	cursorY       int
	cursorVisible bool
	cursorStyle   tcell.CursorStyle
	events        []tcell.Event
	eventIndex    int
	defaultStyle  tcell.Style
}

// NewMockScreen creates a new mock screen with the given dimensions
func NewMockScreen(width, height int) *MockScreen {
	return &MockScreen{
		width:        width,
		height:       height,
		cells:        make(map[int]map[int]Cell),
		events:       make([]tcell.Event, 0),
		defaultStyle: tcell.StyleDefault,
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

// Show is a no-op for mock screen
func (s *MockScreen) Show() {
	// No-op for testing
}

// Size returns the screen dimensions
func (s *MockScreen) Size() (width, height int) {
	return s.width, s.height
}

// SetStyle sets the default style
func (s *MockScreen) SetStyle(style tcell.Style) {
	s.defaultStyle = style
}

// SetContent sets the content at the given position
func (s *MockScreen) SetContent(x, y int, mainc rune, combc []rune, style tcell.Style) {
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
func (s *MockScreen) SetCursorStyle(style tcell.CursorStyle) {
	s.cursorStyle = style
}

// PollEvent returns the next event in the queue
func (s *MockScreen) PollEvent() tcell.Event {
	if s.eventIndex >= len(s.events) {
		// Return nil to indicate no more events
		return nil
	}

	event := s.events[s.eventIndex]
	s.eventIndex++
	return event
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
func (s *MockScreen) GetCursorStyle() tcell.CursorStyle {
	return s.cursorStyle
}

// GetCell returns the content at a specific position
func (s *MockScreen) GetCell(x, y int) (rune, tcell.Style) {
	if row, ok := s.cells[y]; ok {
		if cell, ok := row[x]; ok {
			return cell.Rune, cell.Style
		}
	}
	return ' ', s.defaultStyle
}

// GetLine returns the content of a specific line as a string
func (s *MockScreen) GetLine(y int) string {
	if row, ok := s.cells[y]; ok {
		line := make([]rune, s.width)
		for x := 0; x < s.width; x++ {
			if cell, ok := row[x]; ok {
				line[x] = cell.Rune
			} else {
				line[x] = ' '
			}
		}
		// Trim trailing spaces
		for i := len(line) - 1; i >= 0; i-- {
			if line[i] != ' ' {
				return string(line[:i+1])
			}
		}
		return ""
	}
	return ""
}

// AddEvent adds an event to the event queue for testing
func (s *MockScreen) AddEvent(ev tcell.Event) {
	s.events = append(s.events, ev)
}

// AddKeyEvent adds a key event to the event queue
func (s *MockScreen) AddKeyEvent(key tcell.Key, r rune, mod tcell.ModMask) {
	ev := tcell.NewEventKey(key, r, mod)
	s.AddEvent(ev)
}