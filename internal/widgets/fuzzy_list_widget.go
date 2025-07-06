package widgets

import (
	"strings"
	"vidya/internal/theme"
	
	"github.com/gdamore/tcell/v2"
)

// FuzzyListWidget represents a widget with two bordered sections:
// - Top section: multi-line text area
// - Bottom section: single-line text area using LineEditor
type FuzzyListWidget struct {
	x, y          int          // Position on screen
	width, height int          // Total dimensions
	topLines      []string     // Content of the top section (filtered)
	allLines      []string     // All content (unfiltered)
	bottomEditor  *LineEditor  // Line editor for the bottom section
	theme         *theme.Theme // Theme for styling
	topScroll     int          // Vertical scroll offset for top section
	selectedIndex int          // Currently selected line index
	title         string       // Title to display in the top border
}

// NewFuzzyListWidget creates a new fuzzy list widget
func NewFuzzyListWidget(x, y, width, height int, theme *theme.Theme) *FuzzyListWidget {
	// Calculate bottom editor position and width
	bottomY := y + height - 3 // Position for bottom editor (inside bottom border)
	bottomWidth := width - 2  // Width minus borders

	bottomEditor := NewLineEditor(x+1, bottomY, bottomWidth)

	return &FuzzyListWidget{
		x:             x,
		y:             y,
		width:         width,
		height:        height,
		topLines:      []string{},
		allLines:      []string{},
		bottomEditor:  bottomEditor,
		theme:         theme,
		topScroll:     0,
		selectedIndex: 0,
		title:         "",
	}
}

// SetPosition updates the position of the widget
func (flw *FuzzyListWidget) SetPosition(x, y int) {
	flw.x = x
	flw.y = y

	// Update bottom editor position
	bottomY := y + flw.height - 3
	flw.bottomEditor.SetPosition(x+1, bottomY)
}

// SetSize updates the dimensions of the widget
func (flw *FuzzyListWidget) SetSize(width, height int) {
	flw.width = width
	flw.height = height

	// Update bottom editor size and position
	bottomY := flw.y + height - 3
	bottomWidth := width - 2
	flw.bottomEditor.SetPosition(flw.x+1, bottomY)
	flw.bottomEditor.SetWidth(bottomWidth)
}

// SetWidth updates only the width of the widget
func (flw *FuzzyListWidget) SetWidth(width int) {
	flw.width = width

	// Update bottom editor width
	bottomWidth := width - 2
	flw.bottomEditor.SetWidth(bottomWidth)
}

// GetPosition returns the current position
func (flw *FuzzyListWidget) GetPosition() (int, int) {
	return flw.x, flw.y
}

// GetSize returns the current dimensions
func (flw *FuzzyListWidget) GetSize() (int, int) {
	return flw.width, flw.height
}

// SetTitle sets the title displayed in the top border
func (flw *FuzzyListWidget) SetTitle(title string) {
	flw.title = title
}


// SetContent sets the content of the top section
func (flw *FuzzyListWidget) SetContent(lines []string) {
	flw.allLines = make([]string, len(lines))
	copy(flw.allLines, lines)
	flw.topLines = make([]string, len(lines))
	copy(flw.topLines, lines)
	flw.topScroll = 0
	flw.selectedIndex = 0
}

// GetContent returns the content of the top section
func (flw *FuzzyListWidget) GetContent() []string {
	result := make([]string, len(flw.topLines))
	copy(result, flw.topLines)
	return result
}

// GetBottomEditor returns the line editor for the bottom section
func (flw *FuzzyListWidget) GetBottomEditor() *LineEditor {
	return flw.bottomEditor
}

// getTopSectionHeight calculates the height available for the top section content
func (flw *FuzzyListWidget) getTopSectionHeight() int {
	// Total height minus borders (2) minus bottom section (3: border + content + border)
	return flw.height - 5
}

// getContentWidth calculates the width available for content (excluding borders)
func (flw *FuzzyListWidget) getContentWidth() int {
	return flw.width - 2
}

// Draw renders the fuzzy list widget on the screen
func (flw *FuzzyListWidget) Draw(screen tcell.Screen) {
	if flw.width < 3 || flw.height < 5 {
		return // Widget too small to draw meaningfully
	}

	// Draw top section
	flw.drawTopSection(screen)

	// Draw bottom section
	flw.drawBottomSection(screen)
}

// drawTopSection draws the top multi-line section with border
func (flw *FuzzyListWidget) drawTopSection(screen tcell.Screen) {
	topHeight := flw.getTopSectionHeight()
	contentWidth := flw.getContentWidth()

	// Draw top border with title (using rounded corners)
	flw.drawTopBorderWithTitle(screen)

	// Draw content and side borders
	for i := 0; i < topHeight; i++ {
		row := flw.y + 1 + i

		// Left border
		screen.SetContent(flw.x, row, '│', nil, flw.theme.GetStyle(flw.theme.FuzzyListBorder))

		// Draw content if available
		lineIndex := flw.topScroll + i
		
		// Determine style for this line
		style := flw.theme.GetStyle(flw.theme.FuzzyListContent)
		if lineIndex == flw.selectedIndex {
			style = flw.theme.GetStyle(flw.theme.FuzzyListSelection)
		}
		
		// Clear content area with appropriate style
		for j := 1; j < flw.width-1; j++ {
			screen.SetContent(flw.x+j, row, ' ', nil, style)
		}
		
		if lineIndex < len(flw.topLines) {
			line := flw.topLines[lineIndex]
			if len(line) > contentWidth {
				line = line[:contentWidth]
			}
			for j, r := range line {
				if j < contentWidth {
					screen.SetContent(flw.x+1+j, row, r, nil, style)
				}
			}
		}

		// Right border
		screen.SetContent(flw.x+flw.width-1, row, '│', nil, flw.theme.GetStyle(flw.theme.FuzzyListBorder))
	}

	// Draw separator between top and bottom sections
	separatorY := flw.y + 1 + topHeight
	flw.drawHorizontalBorder(screen, separatorY, '├', '─', '┤')
}

// drawBottomSection draws the bottom single-line section with border
func (flw *FuzzyListWidget) drawBottomSection(screen tcell.Screen) {
	topHeight := flw.getTopSectionHeight()
	bottomY := flw.y + 2 + topHeight

	// Left and right borders for the bottom section
	borderStyle := flw.theme.GetStyle(flw.theme.FuzzyListBorder)
	screen.SetContent(flw.x, bottomY, '│', nil, borderStyle)
	screen.SetContent(flw.x+flw.width-1, bottomY, '│', nil, borderStyle)

	// Draw the line editor (it will handle its own content area)
	flw.bottomEditor.Draw(screen)

	// Draw bottom border with rounded corners
	flw.drawHorizontalBorder(screen, bottomY+1, '╰', '─', '╯')
}

// drawHorizontalBorder draws a horizontal border line
func (flw *FuzzyListWidget) drawHorizontalBorder(screen tcell.Screen, y int, left, middle, right rune) {
	borderStyle := flw.theme.GetStyle(flw.theme.FuzzyListBorder)
	screen.SetContent(flw.x, y, left, nil, borderStyle)
	for i := 1; i < flw.width-1; i++ {
		screen.SetContent(flw.x+i, y, middle, nil, borderStyle)
	}
	screen.SetContent(flw.x+flw.width-1, y, right, nil, borderStyle)
}

// drawTopBorderWithTitle draws the top border with an optional title
func (flw *FuzzyListWidget) drawTopBorderWithTitle(screen tcell.Screen) {
	borderStyle := flw.theme.GetStyle(flw.theme.FuzzyListBorder)
	y := flw.y
	
	// Draw left corner (rounded)
	screen.SetContent(flw.x, y, '╭', nil, borderStyle)
	
	// Calculate title position and padding
	titleLen := len(flw.title)
	borderWidth := flw.width - 2 // excluding corners
	
	if titleLen > 0 && titleLen < borderWidth-4 { // Need at least 2 chars padding on each side
		// Center the title
		leftPadding := (borderWidth - titleLen - 2) / 2 // -2 for the brackets
		
		// Draw left border
		for i := 1; i <= leftPadding; i++ {
			screen.SetContent(flw.x+i, y, '─', nil, borderStyle)
		}
		
		// Draw title with brackets
		screen.SetContent(flw.x+leftPadding+1, y, '┤', nil, borderStyle)
		for i, r := range flw.title {
			screen.SetContent(flw.x+leftPadding+2+i, y, r, nil, borderStyle)
		}
		screen.SetContent(flw.x+leftPadding+2+titleLen, y, '├', nil, borderStyle)
		
		// Draw right border
		for i := flw.x + leftPadding + titleLen + 3; i < flw.x+flw.width-1; i++ {
			screen.SetContent(i, y, '─', nil, borderStyle)
		}
	} else {
		// No title or title too long, just draw border
		for i := 1; i < flw.width-1; i++ {
			screen.SetContent(flw.x+i, y, '─', nil, borderStyle)
		}
	}
	
	// Draw right corner (rounded)
	screen.SetContent(flw.x+flw.width-1, y, '╮', nil, borderStyle)
}

// ScrollTop scrolls the top section by the specified number of lines
func (flw *FuzzyListWidget) ScrollTop(lines int) {
	flw.topScroll += lines
	if flw.topScroll < 0 {
		flw.topScroll = 0
	}

	maxScroll := len(flw.topLines) - flw.getTopSectionHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if flw.topScroll > maxScroll {
		flw.topScroll = maxScroll
	}
}

// ClearTop clears the content of the top section
func (flw *FuzzyListWidget) ClearTop() {
	flw.topLines = []string{}
	flw.allLines = []string{}
	flw.topScroll = 0
	flw.selectedIndex = 0
}

// ClearBottom clears the content of the bottom section
func (flw *FuzzyListWidget) ClearBottom() {
	flw.bottomEditor.SetText("")
}

// Clear clears both sections
func (flw *FuzzyListWidget) Clear() {
	flw.ClearTop()
	flw.ClearBottom()
}

// MoveSelectionUp moves the selection up by one line
func (flw *FuzzyListWidget) MoveSelectionUp() {
	if flw.selectedIndex > 0 {
		flw.selectedIndex--
		// Scroll up if selection moves above visible area
		if flw.selectedIndex < flw.topScroll {
			flw.topScroll = flw.selectedIndex
		}
	}
}

// MoveSelectionDown moves the selection down by one line
func (flw *FuzzyListWidget) MoveSelectionDown() {
	if flw.selectedIndex < len(flw.topLines)-1 {
		flw.selectedIndex++
		// Scroll down if selection moves below visible area
		topHeight := flw.getTopSectionHeight()
		if flw.selectedIndex >= flw.topScroll+topHeight {
			flw.topScroll = flw.selectedIndex - topHeight + 1
		}
	}
}

// GetSelectedIndex returns the currently selected index
func (flw *FuzzyListWidget) GetSelectedIndex() int {
	return flw.selectedIndex
}

// SetSelectedIndex sets the selected index
func (flw *FuzzyListWidget) SetSelectedIndex(index int) {
	if index >= 0 && index < len(flw.topLines) {
		flw.selectedIndex = index
	}
}

// fuzzyMatch checks if a line matches the search criteria
// The criteria is split by spaces and all words must be found in the line
func (flw *FuzzyListWidget) fuzzyMatch(line string, criteria string) bool {
	if criteria == "" {
		return true
	}
	
	// Convert to lowercase for case-insensitive matching
	lineLower := strings.ToLower(line)
	criteriaLower := strings.ToLower(criteria)
	
	// Split criteria into words
	words := strings.Fields(criteriaLower)
	
	// Check if all words are present in the line
	for _, word := range words {
		if !strings.Contains(lineLower, word) {
			return false
		}
	}
	
	return true
}

// applyFilter filters the content based on the search text
func (flw *FuzzyListWidget) applyFilter() {
	searchText := flw.bottomEditor.GetText()
	
	// If search is empty, show all lines
	if searchText == "" {
		flw.topLines = make([]string, len(flw.allLines))
		copy(flw.topLines, flw.allLines)
	} else {
		// Filter lines based on search criteria
		flw.topLines = []string{}
		for _, line := range flw.allLines {
			if flw.fuzzyMatch(line, searchText) {
				flw.topLines = append(flw.topLines, line)
			}
		}
	}
	
	// Reset scroll and selection
	flw.topScroll = 0
	flw.selectedIndex = 0
}

// Show displays the fuzzy list widget as a modal dialog and handles its own event loop
func (flw *FuzzyListWidget) Show(screen tcell.Screen, backgroundRender func()) {
	// Event loop
	for {
		// Render background (editor and status bar)
		backgroundRender()
		
		// Draw widget on top
		flw.Draw(screen)
		
		// Show cursor in the search box
		cursorX, cursorY := flw.bottomEditor.GetCursorPosition()
		screen.ShowCursor(cursorX, cursorY)
		
		// Update screen
		screen.Show()
		
		// Handle events
		ev := screen.PollEvent()
		if ev, ok := ev.(*tcell.EventKey); ok {
			switch ev.Key() {
			case tcell.KeyUp:
				flw.MoveSelectionUp()
			case tcell.KeyDown:
				flw.MoveSelectionDown()
			case tcell.KeyEscape:
				return
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'A':
					return
				default:
					// Pass the key to the line editor
					oldText := flw.bottomEditor.GetText()
					flw.bottomEditor.HandleKey(ev)
					newText := flw.bottomEditor.GetText()
					// If text changed, apply filter
					if oldText != newText {
						flw.applyFilter()
					}
				}
			default:
				// Handle other keys for the line editor (like backspace)
				oldText := flw.bottomEditor.GetText()
				flw.bottomEditor.HandleKey(ev)
				newText := flw.bottomEditor.GetText()
				// If text changed, apply filter
				if oldText != newText {
					flw.applyFilter()
				}
			}
		}
	}
}
