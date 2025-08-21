package widgets

import (
	"strings"
	"veda/internal/events"
	"veda/internal/theme"
	"veda/internal/ui"
)

// DialogResult represents the result of a dialog
type DialogResult string

const (
	DialogResultYes    DialogResult = "yes"
	DialogResultNo     DialogResult = "no"
	DialogResultCancel DialogResult = "cancel"
)

// YesNoDialog represents a modal yes/no dialog widget
type YesNoDialog struct {
	x, y          int          // Position on screen
	width, height int          // Dimensions
	title         string       // Dialog title
	message       string       // Dialog message
	theme         *theme.Theme // Theme for styling
	selectedIndex int          // 0 = Yes, 1 = No
	closed        bool         // Whether dialog has been closed
	result        DialogResult // Result of the dialog
}

// NewYesNoDialog creates a new yes/no dialog
func NewYesNoDialog(x, y, width, height int, title, message string, theme *theme.Theme) *YesNoDialog {
	return &YesNoDialog{
		x:             x,
		y:             y,
		width:         width,
		height:        height,
		title:         title,
		message:       message,
		theme:         theme,
		selectedIndex: 0, // Default to "Yes"
		closed:        false,
		result:        DialogResultCancel,
	}
}

// HandleEvent processes keyboard events for the dialog
func (d *YesNoDialog) HandleEvent(event events.Event) bool {
	if d.closed {
		return false
	}

	switch ev := event.(type) {
	case *events.KeyEvent:
		switch ev.Key {
		case events.KeyTab, events.KeyRight:
			d.selectedIndex = (d.selectedIndex + 1) % 2
			return true
		case events.KeyBacktab, events.KeyLeft:
			d.selectedIndex = (d.selectedIndex - 1 + 2) % 2
			return true
		case events.KeyEnter:
			if d.selectedIndex == 0 {
				d.result = DialogResultYes
			} else {
				d.result = DialogResultNo
			}
			d.closed = true
			return true
		case events.KeyEscape:
			d.result = DialogResultCancel
			d.closed = true
			return true
		case events.KeyRune:
			switch ev.Rune {
			case 'y', 'Y':
				d.result = DialogResultYes
				d.closed = true
				return true
			case 'n', 'N':
				d.result = DialogResultNo
				d.closed = true
				return true
			case 'q':
				d.result = DialogResultCancel
				d.closed = true
				return true
			}
		}
	}
	return false
}

// Render draws the dialog on the screen
func (d *YesNoDialog) Render(screen Screen) {
	if d.closed {
		return
	}

	// Calculate actual dimensions based on content
	minWidth := len(d.title) + 4
	if len(d.message)+4 > minWidth {
		minWidth = len(d.message) + 4
	}
	if minWidth < 30 {
		minWidth = 30
	}

	actualWidth := d.width
	if actualWidth < minWidth {
		actualWidth = minWidth
	}

	actualHeight := d.height
	if actualHeight < 7 {
		actualHeight = 7
	}

	// Center the dialog if dimensions were adjusted
	startX := d.x - (actualWidth-d.width)/2
	startY := d.y - (actualHeight-d.height)/2

	// Define styles
	borderStyle := d.theme.GetStyle(d.theme.DialogBorder)
	backgroundStyle := d.theme.GetStyle(d.theme.DialogContent)
	titleStyle := d.theme.GetStyle(d.theme.FuzzyListSelection) // Use selection style for title
	buttonStyle := d.theme.GetStyle(d.theme.DialogContent)
	selectedButtonStyle := d.theme.GetStyle(d.theme.FuzzyListSelection)

	// Draw background and border
	for row := 0; row < actualHeight; row++ {
		for col := 0; col < actualWidth; col++ {
			x := startX + col
			y := startY + row

			var char rune
			var style ui.Style

			// Border characters
			if row == 0 || row == actualHeight-1 {
				if col == 0 {
					char = '┌'
					if row == actualHeight-1 {
						char = '└'
					}
				} else if col == actualWidth-1 {
					char = '┐'
					if row == actualHeight-1 {
						char = '┘'
					}
				} else {
					char = '─'
				}
				style = borderStyle
			} else {
				if col == 0 || col == actualWidth-1 {
					char = '│'
					style = borderStyle
				} else {
					char = ' '
					style = backgroundStyle
				}
			}

			screen.SetContent(x, y, char, nil, style)
		}
	}

	// Draw title
	if d.title != "" {
		titleX := startX + (actualWidth-len(d.title))/2
		titleY := startY + 1
		for i, r := range d.title {
			screen.SetContent(titleX+i, titleY, r, nil, titleStyle)
		}
	}

	// Draw message (word-wrapped if necessary)
	messageLines := d.wrapText(d.message, actualWidth-4)
	messageStartY := startY + 3
	for i, line := range messageLines {
		if messageStartY+i >= startY+actualHeight-3 {
			break // Don't overflow into button area
		}
		messageX := startX + 2
		for j, r := range line {
			if messageX+j < startX+actualWidth-2 {
				screen.SetContent(messageX+j, messageStartY+i, r, nil, backgroundStyle)
			}
		}
	}

	// Draw buttons
	buttonY := startY + actualHeight - 2
	yesText := "[Yes]"
	noText := "[No]"

	// Calculate button positions
	totalButtonWidth := len(yesText) + len(noText) + 4 // 4 spaces between buttons
	buttonStartX := startX + (actualWidth-totalButtonWidth)/2

	yesX := buttonStartX
	noX := buttonStartX + len(yesText) + 4

	// Draw Yes button
	yesStyle := buttonStyle
	if d.selectedIndex == 0 {
		yesStyle = selectedButtonStyle
	}
	for i, r := range yesText {
		screen.SetContent(yesX+i, buttonY, r, nil, yesStyle)
	}

	// Draw No button
	noStyle := buttonStyle
	if d.selectedIndex == 1 {
		noStyle = selectedButtonStyle
	}
	for i, r := range noText {
		screen.SetContent(noX+i, buttonY, r, nil, noStyle)
	}
}

// wrapText wraps text to fit within the specified width
func (d *YesNoDialog) wrapText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}

	words := strings.Fields(text)
	lines := make([]string, 0)
	currentLine := ""

	for _, word := range words {
		if len(currentLine) == 0 {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= maxWidth {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

// IsClosed returns whether the dialog has been closed
func (d *YesNoDialog) IsClosed() bool {
	return d.closed
}

// GetResult returns the result of the dialog
func (d *YesNoDialog) GetResult() DialogResult {
	return d.result
}

// SetMessage updates the dialog message
func (d *YesNoDialog) SetMessage(message string) {
	d.message = message
}

// SetTitle updates the dialog title
func (d *YesNoDialog) SetTitle(title string) {
	d.title = title
}
