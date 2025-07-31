package editor

import (
	"veda/internal/events"
	"veda/internal/widgets"
)

// FuzzyListOverlay wraps a FuzzyListWidget for use in the overlay rendering system
type FuzzyListOverlay struct {
	widget            *widgets.FuzzyListWidget
	onTextChanged     func(string)
	contentUpdateChan <-chan []string
}

// NewFuzzyListOverlay creates a new fuzzy list overlay
func NewFuzzyListOverlay(widget *widgets.FuzzyListWidget, onTextChanged func(string), contentUpdateChan <-chan []string) *FuzzyListOverlay {
	return &FuzzyListOverlay{
		widget:            widget,
		onTextChanged:     onTextChanged,
		contentUpdateChan: contentUpdateChan,
	}
}

// Draw renders the widget
func (flo *FuzzyListOverlay) Draw(screen widgets.Screen) {
	flo.widget.Draw(screen)
}

// CheckContentUpdates checks for any pending content updates
func (flo *FuzzyListOverlay) CheckContentUpdates() {
	if flo.contentUpdateChan != nil {
		select {
		case newContent := <-flo.contentUpdateChan:
			flo.widget.SetContent(newContent)
		default:
			// No new content, continue
		}
	}
}

// HandleEvent processes keyboard events for the fuzzy list
// Returns (done, result) where done indicates if the overlay should close
func (flo *FuzzyListOverlay) HandleEvent(event interface{}) (bool, string) {
	// Handle resize events
	if resizeEv, ok := event.(*events.ResizeEvent); ok {
		// Recalculate widget dimensions based on new screen size
		width := resizeEv.Width
		height := resizeEv.Height
		
		// Use same proportions as when created (80% width, 70% height)
		fuzzyListWidth := width * 4 / 5
		fuzzyListHeight := height * 7 / 10
		fuzzyListX := (width - fuzzyListWidth) / 2
		fuzzyListY := (height - fuzzyListHeight) / 2
		
		// Update the widget position and size
		flo.widget.SetPosition(fuzzyListX, fuzzyListY)
		flo.widget.SetSize(fuzzyListWidth, fuzzyListHeight)
		
		// Don't close the overlay, just update dimensions
		return false, ""
	}
	
	// Handle keyboard events
	if keyEv, ok := event.(*events.KeyEvent); ok {
		switch keyEv.Key {
		case events.KeyUp:
			flo.widget.MoveSelectionUp()
		case events.KeyDown:
			flo.widget.MoveSelectionDown()
		case events.KeyPgUp:
			flo.widget.MoveSelectionPageUp()
		case events.KeyPgDn:
			flo.widget.MoveSelectionPageDown()
		case events.KeyEnter:
			// Return the selected item
			selectedIndex := flo.widget.GetSelectedIndex()
			if selectedIndex >= 0 {
				// Get the raw (undecorated) selected item
				content := flo.widget.GetContent()
				if selectedIndex < len(content) {
					// The widget tracks the raw line internally
					return true, flo.widget.GetSelectedRawLine()
				}
			}
			return true, ""
		case events.KeyEscape:
			return true, ""
		case events.KeyRune:
			switch keyEv.Rune {
			case 'A':
				return true, ""
			default:
				// Pass the key to the line editor
				oldText := flo.widget.GetBottomEditor().GetText()
				flo.widget.GetBottomEditor().HandleKey(keyEv)
				newText := flo.widget.GetBottomEditor().GetText()
				// If text changed, re-apply filter and notify callback
				if oldText != newText {
					flo.widget.ApplyFilter()
					if flo.onTextChanged != nil {
						flo.onTextChanged(newText)
					}
				}
			}
		default:
			// Handle other keys for the line editor
			oldText := flo.widget.GetBottomEditor().GetText()
			flo.widget.GetBottomEditor().HandleKey(keyEv)
			newText := flo.widget.GetBottomEditor().GetText()
			// If text changed, re-apply filter and notify callback
			if oldText != newText {
				flo.widget.ApplyFilter()
				if flo.onTextChanged != nil {
					flo.onTextChanged(newText)
				}
			}
		}
	}

	return false, ""
}

// GetCursorPosition returns the cursor position in the search box
func (flo *FuzzyListOverlay) GetCursorPosition() (int, int) {
	return flo.widget.GetBottomEditor().GetCursorPosition()
}