package widgets

import (
	"strings"
	"veda/internal/theme"
)

// CompletionItem represents a single completion option
type CompletionItem struct {
	Label  string // Text to display in the menu
	Detail string // Additional detail (for future use)
	InsertText string // Text to insert when selected
}

// CompletionMenu is a floating menu widget for showing completion options
type CompletionMenu struct {
	// Position and dimensions
	x, y          int  // Position on screen (calculated from cursor)
	width, height int  // Dimensions of the menu
	maxHeight     int  // Maximum height before scrolling
	visible       bool // Whether the menu is currently visible
	
	// Content
	allItems      []CompletionItem // All completion items (unfiltered)
	items         []CompletionItem // Filtered completion items
	selectedIndex int              // Currently selected item
	scrollOffset  int              // Scroll offset for long lists
	filterText    string           // Current filter text
	
	// Styling
	theme         *theme.Theme
}

// NewCompletionMenu creates a new completion menu widget
func NewCompletionMenu(theme *theme.Theme) *CompletionMenu {
	return &CompletionMenu{
		maxHeight:     10, // Show max 10 items before scrolling
		visible:       false,
		selectedIndex: 0,
		scrollOffset:  0,
		theme:         theme,
	}
}

// SetItems sets the completion items to display
func (c *CompletionMenu) SetItems(items []CompletionItem) {
	c.allItems = items
	c.filterText = ""
	c.applyFilter()
}

// SetPosition sets the menu position (usually near cursor)
func (c *CompletionMenu) SetPosition(x, y int) {
	c.x = x
	c.y = y
}

// Show makes the menu visible
func (c *CompletionMenu) Show() {
	if len(c.items) > 0 {
		c.visible = true
	}
}

// Hide makes the menu invisible
func (c *CompletionMenu) Hide() {
	c.visible = false
}

// IsVisible returns whether the menu is currently visible
func (c *CompletionMenu) IsVisible() bool {
	return c.visible
}

// MoveUp moves selection up
func (c *CompletionMenu) MoveUp() {
	if c.selectedIndex > 0 {
		c.selectedIndex--
		// Adjust scroll if needed
		if c.selectedIndex < c.scrollOffset {
			c.scrollOffset = c.selectedIndex
		}
	}
}

// MoveDown moves selection down
func (c *CompletionMenu) MoveDown() {
	if c.selectedIndex < len(c.items)-1 {
		c.selectedIndex++
		// Adjust scroll if needed
		visibleEnd := c.scrollOffset + c.getVisibleItemCount() - 1
		if c.selectedIndex > visibleEnd {
			c.scrollOffset = c.selectedIndex - c.getVisibleItemCount() + 1
		}
	}
}

// GetSelectedItem returns the currently selected completion item
func (c *CompletionMenu) GetSelectedItem() *CompletionItem {
	if c.selectedIndex >= 0 && c.selectedIndex < len(c.items) {
		return &c.items[c.selectedIndex]
	}
	return nil
}

// Draw renders the completion menu to the screen
func (c *CompletionMenu) Draw(screen Screen) {
	if !c.visible || len(c.items) == 0 {
		return
	}
	
	// Get styles
	normalStyle := c.theme.GetStyle(c.theme.CompletionMenu)
	selectedStyle := c.theme.GetStyle(c.theme.CompletionMenuSelected)
	
	// Draw items without borders
	visibleItems := c.getVisibleItemCount()
	for i := 0; i < visibleItems; i++ {
		itemIndex := c.scrollOffset + i
		if itemIndex >= len(c.items) {
			break
		}
		
		item := c.items[itemIndex]
		lineY := c.y + i
		
		// Draw item content
		style := normalStyle
		if itemIndex == c.selectedIndex {
			style = selectedStyle
		}
		
		// Clear the line first
		for j := 0; j < c.width; j++ {
			screen.SetContent(c.x+j, lineY, ' ', nil, style)
		}
		
		// Draw the label with padding
		label := item.Label
		if len(label) > c.width-2 { // Leave space for padding
			label = label[:c.width-2]
		}
		for j, ch := range " " + label {
			if j < c.width-1 {
				screen.SetContent(c.x+j, lineY, ch, nil, style)
			}
		}
		
		// Draw scroll indicators if needed
		if (c.scrollOffset > 0 && i == 0) {
			// Up arrow on first line
			screen.SetContent(c.x+c.width-1, lineY, '▲', nil, style)
		}
		if (c.scrollOffset + visibleItems < len(c.items) && i == visibleItems-1) {
			// Down arrow on last line
			screen.SetContent(c.x+c.width-1, lineY, '▼', nil, style)
		}
	}
}

// updateDimensions calculates the menu dimensions based on items
func (c *CompletionMenu) updateDimensions() {
	if len(c.items) == 0 {
		c.width = 0
		c.height = 0
		return
	}
	
	// Find the longest item
	maxLen := 0
	for _, item := range c.items {
		if len(item.Label) > maxLen {
			maxLen = len(item.Label)
		}
	}
	
	// Width: padding (2) + content
	c.width = maxLen + 2
	if c.width < 20 { // Minimum width
		c.width = 20
	}
	if c.width > 60 { // Maximum width
		c.width = 60
	}
	
	// Height: just items (no borders)
	itemCount := len(c.items)
	if itemCount > c.maxHeight {
		itemCount = c.maxHeight
	}
	c.height = itemCount
}

// getVisibleItemCount returns how many items can be displayed
func (c *CompletionMenu) getVisibleItemCount() int {
	itemCount := len(c.items) - c.scrollOffset
	if itemCount > c.maxHeight {
		itemCount = c.maxHeight
	}
	return itemCount
}

// SetFilterText sets the filter text and updates the filtered items
func (c *CompletionMenu) SetFilterText(text string) {
	c.filterText = text
	c.applyFilter()
}

// applyFilter filters the items based on the current filter text
func (c *CompletionMenu) applyFilter() {
	if c.filterText == "" {
		// No filter, show all items
		c.items = make([]CompletionItem, len(c.allItems))
		copy(c.items, c.allItems)
	} else {
		// Filter items that contain the filter text (case-insensitive)
		c.items = []CompletionItem{}
		filterLower := strings.ToLower(c.filterText)
		for _, item := range c.allItems {
			if strings.Contains(strings.ToLower(item.Label), filterLower) {
				c.items = append(c.items, item)
			}
		}
	}
	
	// Reset selection and scroll
	c.selectedIndex = 0
	c.scrollOffset = 0
	c.updateDimensions()
	
	// Hide menu if no items match
	if len(c.items) == 0 {
		c.visible = false
	}
}