package terminal

import (
	"fmt"
	"hash/fnv"
	"strings"
	"veda/internal/events"
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

	// Line rendering cache
	renderedLines []string // Cache of rendered ANSI strings
	lineHashes    []uint64 // Hash of cell content for change detection
	hasShown      bool
	kittyImages   []KittyImagePlacement
}

// shouldLogANSI returns true if we want to emit TRACE logs for this style
// We narrowly scope to bright blue (94) to minimize log volume while debugging
func (s *ScreenAdapter) shouldLogANSI(style ui.Style) bool {
	fg, _, _ := style.Decompose()
	return fg == ui.ColorBrightBlue
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

	sa := &ScreenAdapter{
		term:          term,
		cells:         cells,
		defaultStyle:  ui.StyleDefault,
		renderedLines: make([]string, height),
		lineHashes:    make([]uint64, height),
	}
	return sa
}

// Init initializes the screen
func (s *ScreenAdapter) Init() error {
	if err := s.term.Init(); err != nil {
		return err
	}

	// Resize cells to match terminal size after init
	width, height := s.term.Size()
	LL.Debug("Terminal size is %d:%d", width, height)
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

	// Clear cell buffer and cache
	width, height := s.term.Size()

	// Check if terminal size has changed and resize cell buffer if needed
	if len(s.cells) != height || (len(s.cells) > 0 && len(s.cells[0]) != width) {
		s.resizeCells(width, height)
	}

	for y := 0; y < height && y < len(s.cells); y++ {
		for x := 0; x < width && x < len(s.cells[y]); x++ {
			s.cells[y][x] = cell{ch: ' ', style: ui.StyleDefault}
		}
		// Clear cache for this line
		if y < len(s.lineHashes) {
			s.lineHashes[y] = 0
			s.renderedLines[y] = ""
		}
	}
	s.hasShown = false
}

// Show updates the screen
func (s *ScreenAdapter) Show() {
	// Convert cell buffer to lines with ANSI escape sequences
	width, height := s.term.Size()

	// Check if terminal size has changed and resize cell buffer if needed
	if len(s.cells) != height || (len(s.cells) > 0 && len(s.cells[0]) != width) {
		LL.Debug("ScreenAdapter.Show: Terminal size changed to %dx%d, resizing cell buffer", width, height)
		s.resizeCells(width, height)
	}

	currentHashes := make([]uint64, height)
	for y := 0; y < height && y < len(s.cells); y++ {
		currentHashes[y] = s.hashLine(y, width)
	}

	// The editor reserves the last row for its status bar. If the visible
	// content is the previous frame shifted by one row, let the terminal move
	// those cells locally instead of retransmitting every row.
	if s.hasShown && height > 2 {
		if op, ok := detectScrollOperation(currentHashes, s.lineHashes, height-2); ok {
			s.term.QueueScrollRegion(op.top, op.bottom, op.lines)
		}
	}

	for y := 0; y < height && y < len(s.cells); y++ {
		currentHash := currentHashes[y]

		// Check if we need to re-render this line
		var line string
		if y < len(s.lineHashes) && currentHash == s.lineHashes[y] && y < len(s.renderedLines) {
			// Use cached rendered line
			line = s.renderedLines[y]
		} else {
			// Re-render the line
			line = s.renderLine(y, width)
			// Update cache
			if y < len(s.renderedLines) {
				s.renderedLines[y] = line
				s.lineHashes[y] = currentHash
			}
		}

		s.term.SetLine(y, line)
		// Only log lines that include bright-blue SGR to reduce volume
		if strings.Contains(line, "\x1b[94m") {
			LL.Trace("ANSI.Show y=%d line=%q", y, line)
		}
	}

	s.term.SetKittyImagePlacements(s.kittyImages)
	s.term.Show()
	s.hasShown = true
}

func (s *ScreenAdapter) BeginKittyImageFrame() {
	s.kittyImages = nil
}

func (s *ScreenAdapter) CellPixelSize() (width, height int) {
	return s.term.CellPixelSize()
}

func (s *ScreenAdapter) DrawKittyImage(key, path string, x, y, columns, rows int) {
	if key == "" || path == "" || columns <= 0 || rows <= 0 {
		return
	}
	s.kittyImages = append(s.kittyImages, KittyImagePlacement{
		Key: key, Path: path, X: x, Y: y, Columns: columns, Rows: rows,
	})
}

const minimumScrollMatches = 3

// detectScrollOperation finds the longest run that moved by one row between
// frames. lastRow is inclusive; rows below it, such as the status bar, are not
// considered scrollable.
func detectScrollOperation(current, previous []uint64, lastRow int) (scrollOperation, bool) {
	if len(current) != len(previous) || len(current) == 0 {
		return scrollOperation{}, false
	}
	if lastRow >= len(current) {
		lastRow = len(current) - 1
	}
	if lastRow < minimumScrollMatches {
		return scrollOperation{}, false
	}

	type candidate struct {
		start   int
		end     int
		matches int
		changed int
		lines   int
	}

	findLongest := func(lines int) candidate {
		best := candidate{lines: lines}
		start := 0
		end := lastRow - 1
		if lines < 0 {
			start = 1
			end = lastRow
		}

		var run candidate
		run.lines = lines
		for row := start; row <= end; row++ {
			oldRow := row + lines
			if current[row] == previous[oldRow] {
				if run.matches == 0 {
					run.start = row
				}
				run.end = row
				run.matches++
				if current[row] != previous[row] {
					run.changed++
				}
				continue
			}
			if run.matches > best.matches || (run.matches == best.matches && run.changed > best.changed) {
				best = run
			}
			run = candidate{lines: lines}
		}
		if run.matches > best.matches || (run.matches == best.matches && run.changed > best.changed) {
			best = run
		}
		return best
	}

	up := findLongest(1)
	down := findLongest(-1)
	best := up
	if down.matches > best.matches || (down.matches == best.matches && down.changed > best.changed) {
		best = down
	}
	if best.matches < minimumScrollMatches || best.changed == 0 {
		return scrollOperation{}, false
	}

	if best.lines > 0 {
		return scrollOperation{top: best.start, bottom: best.end + 1, lines: 1}, true
	}
	return scrollOperation{top: best.start - 1, bottom: best.end, lines: -1}, true
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

	// Check if terminal size has changed and resize cell buffer if needed
	if len(s.cells) != height || (len(s.cells) > 0 && len(s.cells[0]) != width) {
		s.resizeCells(width, height)
	}

	// Ensure cells array is properly sized
	if y >= len(s.cells) || x >= len(s.cells[y]) {
		return
	}

	s.cells[y][x] = cell{ch: mainc, style: style}
	if s.shouldLogANSI(style) {
		fg, bg, attrs := style.Decompose()
		LL.Trace("ANSI.SetContent x=%d y=%d ch=%q fg=%v bg=%v attrs=%v", x, y, mainc, fg, bg, attrs)
	}
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
func (s *ScreenAdapter) PollEvent() events.Event {
	return s.term.PollEvent()
}

// EventChannel returns the underlying event channel
func (s *ScreenAdapter) EventChannel() <-chan events.Event {
	return s.term.EventChannel()
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

	// Always render the full width of the terminal
	for x := 0; x < width; x++ {
		var cell cell
		if x < len(s.cells[y]) {
			cell = s.cells[y][x]
		} else {
			// Beyond the cell buffer, use default style with space
			cell.ch = ' '
			cell.style = s.defaultStyle
		}

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

	// Palette helpers: emit raw SGR when available, otherwise 8/16 palette or truecolor
	toPalette := func(c ui.Color, isFg bool) string {
		// Default
		if c == ui.ColorDefault {
			return ""
		}
		// Raw SGR code stored
		if (int32(c) & (1 << 26)) != 0 {
			code := int32(c) & 0xff
			if isFg {
				return fmt.Sprintf("\033[%dm", code)
			}
			return fmt.Sprintf("\033[%dm", code) // bg codes already in 40–47/100–107
		}
		// 256-color index
		if (int32(c) & (1 << 25)) != 0 {
			idx := int32(c) & 0xff
			if isFg {
				return fmt.Sprintf("\033[38;5;%dm", idx)
			}
			return fmt.Sprintf("\033[48;5;%dm", idx)
		}
		// RGB colors (created via NewRGBColor) have the RGB tag bit set; use truecolor
		if (int32(c) & (1 << 24)) != 0 {
			r, g, b := c.RGB()
			if isFg {
				return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
			}
			return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
		}
		// Named palette 0..15
		val := int32(c)
		if val >= 0 && val <= 15 {
			if isFg {
				if val < 8 {
					return fmt.Sprintf("\033[%dm", 30+val)
				}
				return fmt.Sprintf("\033[%dm", 90+(val-8))
			}
			if val < 8 {
				return fmt.Sprintf("\033[%dm", 40+val)
			}
			return fmt.Sprintf("\033[%dm", 100+(val-8))
		}
		// Fallback to truecolor
		r, g, b := c.RGB()
		if isFg {
			return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
		}
		return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
	}

	var fgSeq, bgSeq string
	if fgSeq = toPalette(fg, true); fgSeq != "" {
		styles = append(styles, fgSeq)
	}
	if bgSeq = toPalette(bg, false); bgSeq != "" {
		styles = append(styles, bgSeq)
	}

	// Add attributes (let terminal handle bold as palette-bright if applicable)
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

	if fg == ui.ColorBrightBlue || strings.Contains(strings.Join(styles, ""), "\x1b[94m") {
		LL.Trace("ANSI.Style fg=%v bg=%v attrs=%v -> %q", fg, bg, attrs, strings.Join(styles, ""))
	}
	return styles
}

// convertKeyEvent converts our key event to editor format
func (s *ScreenAdapter) convertKeyEvent(e *events.KeyEvent) ui.Event {
	// Map our keys to editor keys
	editorKey := s.mapKeyToEditor(e.Key)
	editorMod := s.mapModToEditor(e.Modifiers)

	return ui.NewEventKey(editorKey, e.Rune, editorMod)
}

// mapKeyToEditor maps our Key type to ui.Key
func (s *ScreenAdapter) mapKeyToEditor(k events.Key) ui.Key {
	switch k {
	case events.KeyRune:
		return ui.KeyRune
	case events.KeyBackspace, events.KeyBackspace2:
		return ui.KeyBackspace2
	case events.KeyTab:
		return ui.KeyTab
	case events.KeyEnter:
		return ui.KeyEnter
	case events.KeyEscape:
		return ui.KeyEscape
	case events.KeyCtrlSpace:
		return ui.KeyCtrlSpace
	case events.KeyDelete:
		return ui.KeyDelete
	case events.KeyInsert:
		return ui.KeyInsert
	case events.KeyUp:
		return ui.KeyUp
	case events.KeyDown:
		return ui.KeyDown
	case events.KeyLeft:
		return ui.KeyLeft
	case events.KeyRight:
		return ui.KeyRight
	case events.KeyHome:
		return ui.KeyHome
	case events.KeyEnd:
		return ui.KeyEnd
	case events.KeyPgUp:
		return ui.KeyPgUp
	case events.KeyPgDn:
		return ui.KeyPgDn
	case events.KeyF1:
		return ui.KeyF1
	case events.KeyF2:
		return ui.KeyF2
	case events.KeyF3:
		return ui.KeyF3
	case events.KeyF4:
		return ui.KeyF4
	case events.KeyF5:
		return ui.KeyF5
	case events.KeyF6:
		return ui.KeyF6
	case events.KeyF7:
		return ui.KeyF7
	case events.KeyF8:
		return ui.KeyF8
	case events.KeyF9:
		return ui.KeyF9
	case events.KeyF10:
		return ui.KeyF10
	case events.KeyF11:
		return ui.KeyF11
	case events.KeyF12:
		return ui.KeyF12
	default:
		return ui.KeyRune
	}
}

// mapModToEditor maps our ModMask to ui.ModMask
func (s *ScreenAdapter) mapModToEditor(m events.ModMask) ui.ModMask {
	var editorMod ui.ModMask
	if m&events.ModShift != 0 {
		editorMod |= ui.ModShift
	}
	if m&events.ModCtrl != 0 {
		editorMod |= ui.ModCtrl
	}
	if m&events.ModAlt != 0 {
		editorMod |= ui.ModAlt
	}
	if m&events.ModMeta != 0 {
		editorMod |= ui.ModMeta
	}
	return editorMod
}

// convertMouseEvent converts our mouse event to editor format
func (s *ScreenAdapter) convertMouseEvent(_ *events.MouseEvent) ui.Event {
	// For now, we don't have mouse events in editor types
	// Return nil to ignore mouse events
	return nil
}

// resizeCells resizes the cell buffer
func (s *ScreenAdapter) resizeCells(width, height int) {
	newCells := make([][]cell, height)
	newRenderedLines := make([]string, height)
	newLineHashes := make([]uint64, height)

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
		// Copy cached rendered lines if available
		if y < len(s.renderedLines) {
			newRenderedLines[y] = s.renderedLines[y]
			newLineHashes[y] = s.lineHashes[y]
		}
	}
	s.cells = newCells
	s.renderedLines = newRenderedLines
	s.lineHashes = newLineHashes
	s.hasShown = false
}

// hashLine computes a hash of the cell content for a line
func (s *ScreenAdapter) hashLine(y int, width int) uint64 {
	if y >= len(s.cells) {
		return 0
	}

	h := fnv.New64a()
	// Hash the full width to detect changes in terminal width or default style
	for x := 0; x < width; x++ {
		var cell cell
		if x < len(s.cells[y]) {
			cell = s.cells[y][x]
		} else {
			// Beyond the cell buffer, use default style with space
			cell.ch = ' '
			cell.style = s.defaultStyle
		}
		// Hash character
		h.Write([]byte{byte(cell.ch >> 24), byte(cell.ch >> 16), byte(cell.ch >> 8), byte(cell.ch)})
		// Hash style components
		fg, bg, attrs := cell.style.Decompose()
		h.Write([]byte{byte(fg >> 24), byte(fg >> 16), byte(fg >> 8), byte(fg)})
		h.Write([]byte{byte(bg >> 24), byte(bg >> 16), byte(bg >> 8), byte(bg)})
		h.Write([]byte{byte(attrs >> 8), byte(attrs)})
	}
	return h.Sum64()
}
