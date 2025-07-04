package widgets

import (
	"github.com/gdamore/tcell/v2"
)

// DualPaneWidget represents a widget with two bordered sections:
// - Top section: multi-line text area
// - Bottom section: single-line text area using LineEditor
type DualPaneWidget struct {
	x, y          int         // Position on screen
	width, height int         // Total dimensions
	topLines      []string    // Content of the top section
	bottomEditor  *LineEditor // Line editor for the bottom section
	topStyle      tcell.Style // Style for the top section
	borderStyle   tcell.Style // Style for the borders
	topScroll     int         // Vertical scroll offset for top section
}

// NewDualPaneWidget creates a new dual pane widget
func NewDualPaneWidget(x, y, width, height int) *DualPaneWidget {
	// Calculate bottom editor position and width
	bottomY := y + height - 3 // Position for bottom editor (inside bottom border)
	bottomWidth := width - 2  // Width minus borders

	bottomEditor := NewLineEditor(x+1, bottomY, bottomWidth)

	return &DualPaneWidget{
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		topLines:     []string{},
		bottomEditor: bottomEditor,
		topStyle:     tcell.StyleDefault,
		borderStyle:  tcell.StyleDefault,
		topScroll:    0,
	}
}

// SetPosition updates the position of the widget
func (dpw *DualPaneWidget) SetPosition(x, y int) {
	dpw.x = x
	dpw.y = y

	// Update bottom editor position
	bottomY := y + dpw.height - 3
	dpw.bottomEditor.SetPosition(x+1, bottomY)
}

// SetSize updates the dimensions of the widget
func (dpw *DualPaneWidget) SetSize(width, height int) {
	dpw.width = width
	dpw.height = height

	// Update bottom editor size and position
	bottomY := dpw.y + height - 3
	bottomWidth := width - 2
	dpw.bottomEditor.SetPosition(dpw.x+1, bottomY)
	dpw.bottomEditor.SetWidth(bottomWidth)
}

// SetWidth updates only the width of the widget
func (dpw *DualPaneWidget) SetWidth(width int) {
	dpw.width = width

	// Update bottom editor width
	bottomWidth := width - 2
	dpw.bottomEditor.SetWidth(bottomWidth)
}

// GetPosition returns the current position
func (dpw *DualPaneWidget) GetPosition() (int, int) {
	return dpw.x, dpw.y
}

// GetSize returns the current dimensions
func (dpw *DualPaneWidget) GetSize() (int, int) {
	return dpw.width, dpw.height
}

// SetTopStyle sets the style for the top section
func (dpw *DualPaneWidget) SetTopStyle(style tcell.Style) {
	dpw.topStyle = style
}

// SetBottomStyle sets the style for the bottom section (line editor)
func (dpw *DualPaneWidget) SetBottomStyle(style tcell.Style) {
	dpw.bottomEditor.SetStyle(style)
}

// SetBorderStyle sets the style for the borders
func (dpw *DualPaneWidget) SetBorderStyle(style tcell.Style) {
	dpw.borderStyle = style
}

// SetTopContent sets the content of the top section
func (dpw *DualPaneWidget) SetTopContent(lines []string) {
	dpw.topLines = make([]string, len(lines))
	copy(dpw.topLines, lines)
}

// AddTopLine adds a line to the top section
func (dpw *DualPaneWidget) AddTopLine(line string) {
	dpw.topLines = append(dpw.topLines, line)
}

// GetTopContent returns the content of the top section
func (dpw *DualPaneWidget) GetTopContent() []string {
	result := make([]string, len(dpw.topLines))
	copy(result, dpw.topLines)
	return result
}

// GetBottomEditor returns the line editor for the bottom section
func (dpw *DualPaneWidget) GetBottomEditor() *LineEditor {
	return dpw.bottomEditor
}

// getTopSectionHeight calculates the height available for the top section content
func (dpw *DualPaneWidget) getTopSectionHeight() int {
	// Total height minus borders (2) minus bottom section (3: border + content + border)
	return dpw.height - 5
}

// getContentWidth calculates the width available for content (excluding borders)
func (dpw *DualPaneWidget) getContentWidth() int {
	return dpw.width - 2
}

// Draw renders the dual pane widget on the screen
func (dpw *DualPaneWidget) Draw(screen tcell.Screen) {
	if dpw.width < 3 || dpw.height < 5 {
		return // Widget too small to draw meaningfully
	}

	// Draw top section
	dpw.drawTopSection(screen)

	// Draw bottom section
	dpw.drawBottomSection(screen)
}

// drawTopSection draws the top multi-line section with border
func (dpw *DualPaneWidget) drawTopSection(screen tcell.Screen) {
	topHeight := dpw.getTopSectionHeight()
	contentWidth := dpw.getContentWidth()

	// Draw top border
	dpw.drawHorizontalBorder(screen, dpw.y, '┌', '─', '┐')

	// Draw content and side borders
	for i := 0; i < topHeight; i++ {
		row := dpw.y + 1 + i

		// Left border
		screen.SetContent(dpw.x, row, '│', nil, dpw.borderStyle)

		// Clear content area
		for j := 1; j < dpw.width-1; j++ {
			screen.SetContent(dpw.x+j, row, ' ', nil, dpw.topStyle)
		}

		// Draw content if available
		lineIndex := dpw.topScroll + i
		if lineIndex < len(dpw.topLines) {
			line := dpw.topLines[lineIndex]
			if len(line) > contentWidth {
				line = line[:contentWidth]
			}
			for j, r := range line {
				if j < contentWidth {
					screen.SetContent(dpw.x+1+j, row, r, nil, dpw.topStyle)
				}
			}
		}

		// Right border
		screen.SetContent(dpw.x+dpw.width-1, row, '│', nil, dpw.borderStyle)
	}

	// Draw separator between top and bottom sections
	separatorY := dpw.y + 1 + topHeight
	dpw.drawHorizontalBorder(screen, separatorY, '├', '─', '┤')
}

// drawBottomSection draws the bottom single-line section with border
func (dpw *DualPaneWidget) drawBottomSection(screen tcell.Screen) {
	topHeight := dpw.getTopSectionHeight()
	bottomY := dpw.y + 2 + topHeight

	// Left and right borders for the bottom section
	screen.SetContent(dpw.x, bottomY, '│', nil, dpw.borderStyle)
	screen.SetContent(dpw.x+dpw.width-1, bottomY, '│', nil, dpw.borderStyle)

	// Draw the line editor (it will handle its own content area)
	dpw.bottomEditor.Draw(screen)

	// Draw bottom border
	dpw.drawHorizontalBorder(screen, bottomY+1, '└', '─', '┘')
}

// drawHorizontalBorder draws a horizontal border line
func (dpw *DualPaneWidget) drawHorizontalBorder(screen tcell.Screen, y int, left, middle, right rune) {
	screen.SetContent(dpw.x, y, left, nil, dpw.borderStyle)
	for i := 1; i < dpw.width-1; i++ {
		screen.SetContent(dpw.x+i, y, middle, nil, dpw.borderStyle)
	}
	screen.SetContent(dpw.x+dpw.width-1, y, right, nil, dpw.borderStyle)
}

// ScrollTop scrolls the top section by the specified number of lines
func (dpw *DualPaneWidget) ScrollTop(lines int) {
	dpw.topScroll += lines
	if dpw.topScroll < 0 {
		dpw.topScroll = 0
	}

	maxScroll := len(dpw.topLines) - dpw.getTopSectionHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if dpw.topScroll > maxScroll {
		dpw.topScroll = maxScroll
	}
}

// ClearTop clears the content of the top section
func (dpw *DualPaneWidget) ClearTop() {
	dpw.topLines = []string{}
	dpw.topScroll = 0
}

// ClearBottom clears the content of the bottom section
func (dpw *DualPaneWidget) ClearBottom() {
	dpw.bottomEditor.SetText("")
}

// Clear clears both sections
func (dpw *DualPaneWidget) Clear() {
	dpw.ClearTop()
	dpw.ClearBottom()
}
