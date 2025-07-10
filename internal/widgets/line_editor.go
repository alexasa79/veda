package widgets

import (
	"github.com/gdamore/tcell/v2"
)

// LineEditor represents a single-line text input widget
type LineEditor struct {
	x, y   int    // Position on screen
	width  int    // Width of the editor
	text   string // Current text content
	cursor int    // Cursor position within the text
	style  tcell.Style
}

// NewLineEditor creates a new single-line editor widget
func NewLineEditor(x, y, width int) *LineEditor {
	return &LineEditor{
		x:     x,
		y:     y,
		width: width,
		text:  "",
		cursor: 0,
		style: tcell.StyleDefault,
	}
}

// SetStyle sets the display style for the line editor
func (le *LineEditor) SetStyle(style tcell.Style) {
	le.style = style
}

// SetText sets the text content of the line editor
func (le *LineEditor) SetText(text string) {
	le.text = text
	if le.cursor > len(le.text) {
		le.cursor = len(le.text)
	}
}

// GetText returns the current text content
func (le *LineEditor) GetText() string {
	return le.text
}

// getScrollOffset calculates the horizontal scroll offset for long text
func (le *LineEditor) getScrollOffset() int {
	// If text fits within width, no scrolling needed
	if len(le.text) < le.width {
		return 0
	}

	// Keep cursor visible with some margin
	margin := 5
	if le.width < 10 {
		margin = le.width / 4
	}

	// If cursor is near the beginning, show from the start
	if le.cursor < margin {
		return 0
	}

	// Calculate the desired scroll offset to keep cursor at margin from left edge
	desiredOffset := le.cursor - margin

	// If the cursor is at or past the end of text, we need to ensure
	// there's space to show the cursor after the last character
	maxOffset := len(le.text) - le.width + 1
	if maxOffset < 0 {
		maxOffset = 0
	}

	// Use the smaller of desired offset and max offset
	if desiredOffset > maxOffset {
		return maxOffset
	}

	return desiredOffset
}

// ScreenWriter is a minimal interface for drawing to screen
type ScreenWriter interface {
	SetContent(x, y int, mainc rune, combc []rune, style tcell.Style)
}

// Draw renders the line editor on the screen
func (le *LineEditor) Draw(screen ScreenWriter) {
	// Clear the line editor area
	for i := 0; i < le.width; i++ {
		screen.SetContent(le.x+i, le.y, ' ', nil, le.style)
	}

	// Draw the text content
	displayText := le.text
	scrollOffset := le.getScrollOffset()

	if len(displayText) > le.width {
		end := scrollOffset + le.width
		if end > len(displayText) {
			end = len(displayText)
		}
		displayText = displayText[scrollOffset:end]
	}

	for i, r := range displayText {
		if i < le.width {
			screen.SetContent(le.x+i, le.y, r, nil, le.style)
		}
	}
}

// GetCursorPosition returns the screen coordinates where the cursor should be displayed
func (le *LineEditor) GetCursorPosition() (int, int) {
	scrollOffset := le.getScrollOffset()
	cursorX := le.x + le.cursor - scrollOffset

	// Ensure cursor stays within the widget bounds
	if cursorX >= le.x+le.width {
		cursorX = le.x + le.width - 1
	}
	if cursorX < le.x {
		cursorX = le.x
	}

	return cursorX, le.y
}

// HandleKey processes keyboard input for the line editor
func (le *LineEditor) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyLeft:
		if le.cursor > 0 {
			le.cursor--
		}
		return true
	case tcell.KeyRight:
		if le.cursor < len(le.text) {
			le.cursor++
		}
		return true
	case tcell.KeyHome:
		le.cursor = 0
		return true
	case tcell.KeyEnd:
		le.cursor = len(le.text)
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if le.cursor > 0 {
			le.text = le.text[:le.cursor-1] + le.text[le.cursor:]
			le.cursor--
		}
		return true
	case tcell.KeyDelete:
		if le.cursor < len(le.text) {
			le.text = le.text[:le.cursor] + le.text[le.cursor+1:]
		}
		return true
	case tcell.KeyRune:
		// Insert the typed character
		char := ev.Rune()
		le.text = le.text[:le.cursor] + string(char) + le.text[le.cursor:]
		le.cursor++
		return true
	}
	return false
}

// SetPosition updates the position of the line editor
func (le *LineEditor) SetPosition(x, y int) {
	le.x = x
	le.y = y
}

// SetWidth updates the width of the line editor
func (le *LineEditor) SetWidth(width int) {
	le.width = width
}

// GetPosition returns the current position of the line editor
func (le *LineEditor) GetPosition() (int, int) {
	return le.x, le.y
}

// GetWidth returns the current width of the line editor
func (le *LineEditor) GetWidth() int {
	return le.width
}

// SetCursor sets the cursor position within the text
func (le *LineEditor) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(le.text) {
		pos = len(le.text)
	}
	le.cursor = pos
}
