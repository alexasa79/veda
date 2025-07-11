package editor

import (
	"veda/internal/terminal"
	"veda/internal/ui"
)

// TerminalScreen implements the Screen interface using our terminal package
type TerminalScreen struct {
	adapter *terminal.ScreenAdapter
}

// NewTerminalScreen creates a new terminal-based screen
func NewTerminalScreen() (*TerminalScreen, error) {
	term, err := terminal.New()
	if err != nil {
		return nil, err
	}
	
	adapter := terminal.NewScreenAdapter(term)
	return &TerminalScreen{adapter: adapter}, nil
}

// Init initializes the screen
func (s *TerminalScreen) Init() error {
	return s.adapter.Init()
}

// Fini finalizes the screen
func (s *TerminalScreen) Fini() {
	s.adapter.Fini()
}

// Clear clears the screen
func (s *TerminalScreen) Clear() {
	s.adapter.Clear()
}

// Show updates the screen
func (s *TerminalScreen) Show() {
	s.adapter.Show()
}

// Size returns the screen dimensions
func (s *TerminalScreen) Size() (width, height int) {
	return s.adapter.Size()
}

// SetStyle sets the default style
func (s *TerminalScreen) SetStyle(style ui.Style) {
	s.adapter.SetStyle(style)
}

// SetContent sets the content at a specific position
func (s *TerminalScreen) SetContent(x, y int, mainc rune, combc []rune, style ui.Style) {
	s.adapter.SetContent(x, y, mainc, combc, style)
}

// ShowCursor shows the cursor at the specified position
func (s *TerminalScreen) ShowCursor(x, y int) {
	s.adapter.ShowCursor(x, y)
}

// SetCursorStyle sets the cursor style
func (s *TerminalScreen) SetCursorStyle(style ui.CursorStyle) {
	s.adapter.SetCursorStyle(style)
}

// PollEvent polls for events
func (s *TerminalScreen) PollEvent() ui.Event {
	return s.adapter.PollEvent()
}