package widgets

import (
	"strings"
	"time"
	"veda/internal/events"
	"veda/internal/fuzzysearch"
	"veda/internal/theme"
	"veda/internal/ui"
	"veda/internal/utils"
)

// Screen is the interface for screen operations used by widgets
type Screen interface {
	SetContent(x, y int, mainc rune, combc []rune, style ui.Style)
	Size() (width, height int)
	Clear()
	Show()
	ShowCursor(x, y int)
	PollEvent() events.Event
	EventChannel() <-chan events.Event  // Returns the underlying event channel
}

// DecorationMode defines how items should be decorated in the list
type DecorationMode int

const (
	// DecorationNone shows items as-is without any decoration
	DecorationNone DecorationMode = iota
	// DecorationFileIcon adds file type icons based on file extension
	DecorationFileIcon
	// DecorationSymbol adds symbol icons (for future use with symbol lists)
	DecorationSymbol
)

// FuzzyListWidget represents a widget with two bordered sections:
// - Top section: multi-line text area
// - Bottom section: single-line text area using LineEditor
type FuzzyListWidget struct {
	x, y           int            // Position on screen
	width, height  int            // Total dimensions
	topLines       []string       // Content of the top section (filtered, decorated)
	allLines       []string       // All content (unfiltered, decorated)
	rawLines       []string       // Original undecorated lines
	topIndices     []int          // Indices mapping filtered lines to raw lines
	bottomEditor   *LineEditor    // Line editor for the bottom section
	theme          *theme.Theme   // Theme for styling
	topScroll      int            // Vertical scroll offset for top section
	selectedIndex  int            // Currently selected line index
	title          string         // Title to display in the top border
	decorationMode DecorationMode // How to decorate items
}

// NewFuzzyListWidget creates a new fuzzy list widget
func NewFuzzyListWidget(x, y, width, height int, theme *theme.Theme) *FuzzyListWidget {
	// Calculate bottom editor position and width
	bottomY := y + height - 3 // Position for bottom editor (inside bottom border)
	bottomWidth := width - 2  // Width minus borders

	bottomEditor := NewLineEditor(x+1, bottomY, bottomWidth)

	return &FuzzyListWidget{
		x:              x,
		y:              y,
		width:          width,
		height:         height,
		topLines:       []string{},
		allLines:       []string{},
		rawLines:       []string{},
		topIndices:     []int{},
		bottomEditor:   bottomEditor,
		theme:          theme,
		topScroll:      0,
		selectedIndex:  0,
		title:          "",
		decorationMode: DecorationNone,
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

// SetDecorationMode sets how items should be decorated
func (flw *FuzzyListWidget) SetDecorationMode(mode DecorationMode) {
	flw.decorationMode = mode
	// Re-apply decorations if we have content
	if len(flw.rawLines) > 0 {
		flw.applyDecorations()
	}
}

// applyDecorations applies the current decoration mode to raw lines
func (flw *FuzzyListWidget) applyDecorations() {
	flw.allLines = make([]string, len(flw.rawLines))

	for i, line := range flw.rawLines {
		decorated := line

		switch flw.decorationMode {
		case DecorationFileIcon:
			// Extract filename from path for icon determination
			parts := strings.Split(line, "/")
			filename := parts[len(parts)-1]
			// Assume files for now, could be enhanced to detect directories
			icon := utils.GetFileIcon(filename, "file")
			decorated = icon + " " + line

		case DecorationSymbol:
			// TODO: Add symbol decoration logic
			decorated = "◉ " + line

		case DecorationNone:
			// Keep as-is
		}

		flw.allLines[i] = decorated
	}

	// Re-apply filter with new decorations
	flw.applyFilter()
}

// SetContent sets the content of the top section
func (flw *FuzzyListWidget) SetContent(lines []string) {
	flw.rawLines = make([]string, len(lines))
	copy(flw.rawLines, lines)
	flw.applyDecorations()
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
func (flw *FuzzyListWidget) Draw(screen Screen) {
	if flw.width < 3 || flw.height < 5 {
		return // Widget too small to draw meaningfully
	}

	// Draw top section
	flw.drawTopSection(screen)

	// Draw bottom section
	flw.drawBottomSection(screen)
}

// drawTopSection draws the top multi-line section with border
func (flw *FuzzyListWidget) drawTopSection(screen Screen) {
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
			// Convert to runes to handle multi-byte characters correctly
			runes := []rune(line)
			if len(runes) > contentWidth {
				runes = runes[:contentWidth]
			}
			for j, r := range runes {
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
func (flw *FuzzyListWidget) drawBottomSection(screen Screen) {
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
func (flw *FuzzyListWidget) drawHorizontalBorder(screen Screen, y int, left, middle, right rune) {
	borderStyle := flw.theme.GetStyle(flw.theme.FuzzyListBorder)
	screen.SetContent(flw.x, y, left, nil, borderStyle)
	for i := 1; i < flw.width-1; i++ {
		screen.SetContent(flw.x+i, y, middle, nil, borderStyle)
	}
	screen.SetContent(flw.x+flw.width-1, y, right, nil, borderStyle)
}

// drawTopBorderWithTitle draws the top border with an optional title
func (flw *FuzzyListWidget) drawTopBorderWithTitle(screen Screen) {
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
	flw.rawLines = []string{}
	flw.topIndices = []int{}
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

// MoveSelectionPageUp moves the selection up by one page
func (flw *FuzzyListWidget) MoveSelectionPageUp() {
	pageSize := flw.getTopSectionHeight()
	if pageSize <= 0 {
		return
	}
	
	// Move selection up by one page
	flw.selectedIndex -= pageSize
	if flw.selectedIndex < 0 {
		flw.selectedIndex = 0
	}
	
	// Adjust scroll to keep selection visible
	if flw.selectedIndex < flw.topScroll {
		flw.topScroll = flw.selectedIndex
	}
}

// MoveSelectionPageDown moves the selection down by one page
func (flw *FuzzyListWidget) MoveSelectionPageDown() {
	pageSize := flw.getTopSectionHeight()
	if pageSize <= 0 {
		return
	}
	
	// Move selection down by one page
	flw.selectedIndex += pageSize
	maxIndex := len(flw.topLines) - 1
	if flw.selectedIndex > maxIndex {
		flw.selectedIndex = maxIndex
	}
	if flw.selectedIndex < 0 {
		flw.selectedIndex = 0
	}
	
	// Adjust scroll to keep selection visible
	topHeight := flw.getTopSectionHeight()
	if flw.selectedIndex >= flw.topScroll+topHeight {
		flw.topScroll = flw.selectedIndex - topHeight + 1
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

// GetSelectedRawLine returns the raw (undecorated) line for the current selection
func (flw *FuzzyListWidget) GetSelectedRawLine() string {
	if flw.selectedIndex >= 0 && flw.selectedIndex < len(flw.topIndices) {
		rawIndex := flw.topIndices[flw.selectedIndex]
		if rawIndex >= 0 && rawIndex < len(flw.rawLines) {
			return flw.rawLines[rawIndex]
		}
	}
	return ""
}

// applyFilter filters the content based on the search text
func (flw *FuzzyListWidget) applyFilter() {
	searchText := flw.bottomEditor.GetText()

	// Clear the filtered lists
	flw.topLines = []string{}
	flw.topIndices = []int{}

	// Filter lines and track indices
	for i, line := range flw.allLines {
		if fuzzysearch.Match(line, searchText) {
			flw.topLines = append(flw.topLines, line)
			flw.topIndices = append(flw.topIndices, i)
		}
	}

	// Reset scroll and selection
	flw.topScroll = 0
	flw.selectedIndex = 0
}

// Show displays the fuzzy list widget as a modal dialog and handles its own event loop
// Returns the selected item, or empty string if cancelled
// If onTextChanged is provided, it will be called when the search text changes
// If contentUpdateChan is provided, the widget will use async event handling
func (flw *FuzzyListWidget) Show(screen Screen, backgroundRender func()) string {
	return flw.ShowWithCallbacks(screen, backgroundRender, nil, nil)
}

// ShowWithCallbacks displays the fuzzy list widget with optional callbacks for async operations
// The onTextChanged callback is called whenever the search text changes (with the new text)
// The contentUpdateChan can be used to send new content to display asynchronously
// Returns the selected item (raw line), or empty string if cancelled
func (flw *FuzzyListWidget) ShowWithCallbacks(screen Screen, backgroundRender func(), 
	onTextChanged func(string), contentUpdateChan <-chan []string) string {
	
	// Determine if we should use async event handling
	useAsync := contentUpdateChan != nil
	
	var screenEvents <-chan events.Event
	var ticker *time.Ticker
	
	if useAsync {
		// Get the screen event channel for non-blocking event handling
		screenEvents = screen.EventChannel()
		
		// Create a ticker for periodic redraws
		ticker = time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
	}
	
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

		var ev events.Event
		
		if useAsync {
			// Use select for non-blocking event handling
			select {
			case newContent := <-contentUpdateChan:
				// Update content asynchronously
				flw.SetContent(newContent)
				continue // Redraw with new content
				
			case ev = <-screenEvents:
				// Got a keyboard event
				
			case <-ticker.C:
				// Periodic redraw to handle any visual updates
				continue
			}
		} else {
			// Use blocking event handling for simple case
			ev = screen.PollEvent()
		}
		
		// Handle keyboard events
		if keyEv, ok := ev.(*events.KeyEvent); ok {
			switch keyEv.Key {
			case events.KeyUp:
				flw.MoveSelectionUp()
			case events.KeyDown:
				flw.MoveSelectionDown()
			case events.KeyPgUp:
				flw.MoveSelectionPageUp()
			case events.KeyPgDn:
				flw.MoveSelectionPageDown()
			case events.KeyEnter:
				// Return the raw (undecorated) selected item
				if flw.selectedIndex >= 0 && flw.selectedIndex < len(flw.topIndices) {
					rawIndex := flw.topIndices[flw.selectedIndex]
					return flw.rawLines[rawIndex]
				}
				return ""
			case events.KeyEscape:
				return ""
			case events.KeyRune:
				switch keyEv.Rune {
				case 'A':
					return ""
				default:
					// Pass the key to the line editor
					oldText := flw.bottomEditor.GetText()
					flw.bottomEditor.HandleKey(keyEv)
					newText := flw.bottomEditor.GetText()
					// If text changed, notify callback and apply filter
					if oldText != newText {
						if onTextChanged != nil {
							onTextChanged(newText)
						}
						flw.applyFilter()
					}
				}
			default:
				// Handle other keys for the line editor (like backspace)
				oldText := flw.bottomEditor.GetText()
				flw.bottomEditor.HandleKey(keyEv)
				newText := flw.bottomEditor.GetText()
				// If text changed, notify callback and apply filter
				if oldText != newText {
					if onTextChanged != nil {
						onTextChanged(newText)
					}
					flw.applyFilter()
				}
			}
		}
	}
}
