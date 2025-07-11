package terminal

import (
	"fmt"
	"strings"
	"veda/internal/ui"
)

// ScreenAdapter adapts our Terminal to match the editor's Screen interface
// This allows gradual migration from tcell to our string-based rendering
type ScreenAdapter struct {
	term *Terminal

	// Cell buffer for compatibility with SetContent
	cells           [][]cell
	defaultStyle    ui.Style
	needsFullRedraw bool // Track if we need a full redraw
}

type cell struct {
	ch    rune
	style ui.Style
}

// NewScreenAdapter creates a new adapter around our terminal
func NewScreenAdapter(term *Terminal) *ScreenAdapter {
	width, height := term.Size()
	cells := make([][]cell, height)
	for i := range cells {
		cells[i] = make([]cell, width)
		for j := range cells[i] {
			cells[i][j] = cell{ch: ' ', style: ui.StyleDefault}
		}
	}

	return &ScreenAdapter{
		term:         term,
		cells:        cells,
		defaultStyle: ui.StyleDefault,
	}
}

// Init initializes the screen
func (s *ScreenAdapter) Init() error {
	if err := s.term.Init(); err != nil {
		return err
	}

	// Resize cells to match terminal size after init
	width, height := s.term.Size()
	s.resizeCells(width, height)

	return nil
}

// Fini finalizes the screen
func (s *ScreenAdapter) Fini() {
	s.term.Close()
}

// Clear clears the screen
func (s *ScreenAdapter) Clear() {
	s.term.Clear()

	// Clear cell buffer
	width, height := s.term.Size()
	for y := 0; y < height && y < len(s.cells); y++ {
		for x := 0; x < width && x < len(s.cells[y]); x++ {
			s.cells[y][x] = cell{ch: ' ', style: ui.StyleDefault}
		}
	}
}

// Show updates the screen
func (s *ScreenAdapter) Show() {
	// Convert cell buffer to lines with ANSI escape sequences
	width, height := s.term.Size()

	for y := 0; y < height && y < len(s.cells); y++ {
		line := s.renderLine(y, width)
		s.term.SetLine(y, line)
	}

	s.term.Show()
}

// Size returns the screen dimensions
func (s *ScreenAdapter) Size() (width, height int) {
	return s.term.Size()
}

// SetStyle sets the default style
func (s *ScreenAdapter) SetStyle(style ui.Style) {
	s.defaultStyle = style
}

// SetContent sets the content at a specific position
func (s *ScreenAdapter) SetContent(x, y int, mainc rune, combc []rune, style ui.Style) {
	width, height := s.term.Size()
	if x < 0 || y < 0 || x >= width || y >= height {
		return
	}

	// Ensure cells array is properly sized
	if y >= len(s.cells) || x >= len(s.cells[y]) {
		return
	}

	s.cells[y][x] = cell{ch: mainc, style: style}
}

// ShowCursor shows the cursor at the specified position
func (s *ScreenAdapter) ShowCursor(x, y int) {
	s.term.SetCursor(x, y)
	s.term.SetCursorVisible(true)
}

// SetCursorStyle sets the cursor style
func (s *ScreenAdapter) SetCursorStyle(style ui.CursorStyle) {
	switch style {
	case ui.CursorStyleSteadyBar:
		s.term.SetCursorStyle(CursorStyleBar)
	case ui.CursorStyleSteadyUnderline:
		s.term.SetCursorStyle(CursorStyleUnderline)
	default:
		s.term.SetCursorStyle(CursorStyleBlock)
	}
}

// PollEvent polls for events
func (s *ScreenAdapter) PollEvent() ui.Event {
	ev := s.term.PollEvent()

	switch e := ev.(type) {
	case *EventKey:
		// Convert our key event to tcell key event
		return s.convertKeyEvent(e)
	case *EventResize:
		// Resize cell buffer
		s.resizeCells(e.Width, e.Height)
		return ui.NewEventResize(e.Width, e.Height)
	case *EventMouse:
		// Convert mouse event
		return s.convertMouseEvent(e)
	}

	return nil
}

// Helper methods

// renderLine converts a line of cells to a string with ANSI escape sequences
func (s *ScreenAdapter) renderLine(y int, width int) string {
	if y >= len(s.cells) {
		return ""
	}

	var result strings.Builder
	lastStyle := ui.StyleDefault
	styleApplied := false

	for x := 0; x < width && x < len(s.cells[y]); x++ {
		cell := s.cells[y][x]

		// If style changed, apply new style
		if cell.style != lastStyle || !styleApplied {
			// Reset previous style
			if styleApplied {
				result.WriteString("\033[0m")
			}

			// Apply new style
			fg, bg, attrs := cell.style.Decompose()
			styles := s.buildStyleSequence(fg, bg, attrs)
			for _, style := range styles {
				result.WriteString(style)
			}

			lastStyle = cell.style
			styleApplied = true
		}

		// Add character
		if cell.ch == 0 || cell.ch == '\x00' {
			result.WriteRune(' ')
		} else {
			result.WriteRune(cell.ch)
		}
	}

	// Reset at end of line
	if styleApplied {
		result.WriteString("\033[0m")
	}

	return result.String()
}

// buildStyleSequence creates ANSI escape sequences for editor style
func (s *ScreenAdapter) buildStyleSequence(fg, bg ui.Color, attrs ui.AttrMask) []string {
	var styles []string

	// Use true color (24-bit) mode for accurate colors
	if fg != ui.ColorDefault {
		r, g, b := fg.RGB()
		styles = append(styles, fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b))
	}
	if bg != ui.ColorDefault {
		r, g, b := bg.RGB()
		styles = append(styles, fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b))
	}

	// Add attributes
	if attrs&ui.AttrBold != 0 {
		styles = append(styles, Bold)
	}
	if attrs&ui.AttrBlink != 0 {
		styles = append(styles, Blink)
	}
	if attrs&ui.AttrReverse != 0 {
		styles = append(styles, Reverse)
	}
	if attrs&ui.AttrUnderline != 0 {
		styles = append(styles, Underline)
	}
	if attrs&ui.AttrDim != 0 {
		styles = append(styles, Dim)
	}
	if attrs&ui.AttrItalic != 0 {
		styles = append(styles, Italic)
	}
	if attrs&ui.AttrStrikeThrough != 0 {
		styles = append(styles, Strikethrough)
	}

	return styles
}


// convertKeyEvent converts our key event to editor format
func (s *ScreenAdapter) convertKeyEvent(e *EventKey) ui.Event {
	// Map our keys to editor keys
	editorKey := s.mapKeyToEditor(e.Key)
	editorMod := s.mapModToEditor(e.Mod)

	return ui.NewEventKey(editorKey, e.Rune, editorMod)
}

// mapKeyToEditor maps our Key type to ui.Key
func (s *ScreenAdapter) mapKeyToEditor(k Key) ui.Key {
	switch k {
	case KeyRune:
		return ui.KeyRune
	case KeyBackspace, KeyBackspace2:
		return ui.KeyBackspace2
	case KeyTab:
		return ui.KeyTab
	case KeyEnter:
		return ui.KeyEnter
	case KeyEscape:
		return ui.KeyEscape
	case KeyDelete:
		return ui.KeyDelete
	case KeyInsert:
		return ui.KeyInsert
	case KeyUp:
		return ui.KeyUp
	case KeyDown:
		return ui.KeyDown
	case KeyLeft:
		return ui.KeyLeft
	case KeyRight:
		return ui.KeyRight
	case KeyHome:
		return ui.KeyHome
	case KeyEnd:
		return ui.KeyEnd
	case KeyPgUp:
		return ui.KeyPgUp
	case KeyPgDn:
		return ui.KeyPgDn
	case KeyF1:
		return ui.KeyF1
	case KeyF2:
		return ui.KeyF2
	case KeyF3:
		return ui.KeyF3
	case KeyF4:
		return ui.KeyF4
	case KeyF5:
		return ui.KeyF5
	case KeyF6:
		return ui.KeyF6
	case KeyF7:
		return ui.KeyF7
	case KeyF8:
		return ui.KeyF8
	case KeyF9:
		return ui.KeyF9
	case KeyF10:
		return ui.KeyF10
	case KeyF11:
		return ui.KeyF11
	case KeyF12:
		return ui.KeyF12
	default:
		return ui.KeyRune
	}
}

// mapModToEditor maps our ModMask to ui.ModMask
func (s *ScreenAdapter) mapModToEditor(m ModMask) ui.ModMask {
	var editorMod ui.ModMask
	if m&ModShift != 0 {
		editorMod |= ui.ModShift
	}
	if m&ModCtrl != 0 {
		editorMod |= ui.ModCtrl
	}
	if m&ModAlt != 0 {
		editorMod |= ui.ModAlt
	}
	if m&ModMeta != 0 {
		editorMod |= ui.ModMeta
	}
	return editorMod
}

// convertMouseEvent converts our mouse event to editor format
func (s *ScreenAdapter) convertMouseEvent(e *EventMouse) ui.Event {
	// For now, we don't have mouse events in editor types
	// Return nil to ignore mouse events
	return nil
}

// resizeCells resizes the cell buffer
func (s *ScreenAdapter) resizeCells(width, height int) {
	newCells := make([][]cell, height)
	for y := range newCells {
		newCells[y] = make([]cell, width)
		for x := range newCells[y] {
			// Initialize with default values
			newCells[y][x] = cell{ch: ' ', style: ui.StyleDefault}
			// Copy existing content if available
			if y < len(s.cells) && x < len(s.cells[y]) {
				newCells[y][x] = s.cells[y][x]
			}
		}
	}
	s.cells = newCells
}