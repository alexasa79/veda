package editor

import (
	"strings"
	"veda/internal/events"
	"veda/internal/widgets"
)

// showCompletions shows the completion menu with dummy items for testing
func (e *Editor) showCompletions() error {
	// Get the current word prefix to use as initial filter
	prefix := e.getWordBeforeCursor()
	
	// For now, create dummy completion items
	dummyItems := []widgets.CompletionItem{
		{Label: "append", InsertText: "append"},
		{Label: "array", InsertText: "array"},
		{Label: "buffer", InsertText: "buffer"},
		{Label: "create", InsertText: "create"},
		{Label: "delete", InsertText: "delete"},
		{Label: "editor", InsertText: "editor"},
		{Label: "function", InsertText: "function"},
		{Label: "generate", InsertText: "generate"},
		{Label: "handle", InsertText: "handle"},
		{Label: "implement", InsertText: "implement"},
		{Label: "insert", InsertText: "insert"},
		{Label: "keyboard", InsertText: "keyboard"},
		{Label: "length", InsertText: "length"},
		{Label: "method", InsertText: "method"},
		{Label: "navigate", InsertText: "navigate"},
		{Label: "operator", InsertText: "operator"},
		{Label: "process", InsertText: "process"},
		{Label: "query", InsertText: "query"},
		{Label: "render", InsertText: "render"},
		{Label: "string", InsertText: "string"},
		{Label: "terminal", InsertText: "terminal"},
		{Label: "update", InsertText: "update"},
		{Label: "variable", InsertText: "variable"},
		{Label: "widget", InsertText: "widget"},
		{Label: "window", InsertText: "window"},
	}
	
	// Set the items
	e.completionMenu.SetItems(dummyItems)
	
	// Set initial filter based on word before cursor
	e.completionMenu.SetFilterText(prefix)
	
	// Calculate menu position (below cursor)
	menuX, menuY := e.getCursorScreenPosition()
	menuY++ // Position below cursor
	
	// Adjust position if too close to bottom
	screenWidth, screenHeight := e.screen.Size()
	menuHeight := len(dummyItems) // Calculate based on items we set
	if menuHeight > 10 { // Max height is 10 items
		menuHeight = 10
	}
	
	// If menu would go off bottom, position above cursor
	if menuY + menuHeight > screenHeight - 1 {
		menuY = menuY - menuHeight - 1 // Position above cursor
	}
	
	// Ensure menu doesn't go off right edge
	menuWidth := 30 // Approximate width
	if menuX + menuWidth > screenWidth {
		menuX = screenWidth - menuWidth
	}
	
	// Set position and show
	e.completionMenu.SetPosition(menuX, menuY)
	e.completionMenu.Show()
	
	return nil
}

// getCursorScreenPosition returns the screen position of the cursor
func (e *Editor) getCursorScreenPosition() (int, int) {
	// Calculate X position (accounting for line numbers)
	lineNumberWidth := e.buffer.LineNumberWidth
	x := lineNumberWidth + e.buffer.CursorX
	
	// Calculate Y position (accounting for context and scroll)
	y := e.getContentStartY()
	cursorLine := e.buffer.CursorIterator.LineNumber()
	viewLine := e.buffer.ViewIterator.LineNumber()
	y += cursorLine - viewLine
	
	return x, y
}

// acceptCompletion accepts the currently selected completion
func (e *Editor) acceptCompletion() error {
	if !e.completionMenu.IsVisible() {
		return nil
	}
	
	item := e.completionMenu.GetSelectedItem()
	if item == nil {
		e.completionMenu.Hide()
		return nil
	}
	
	// Get the prefix that was already typed
	prefix := e.getWordBeforeCursor()
	
	// Calculate what text needs to be inserted
	// If the item starts with the prefix, only insert the remaining part
	textToInsert := item.InsertText
	if strings.HasPrefix(strings.ToLower(item.InsertText), strings.ToLower(prefix)) {
		textToInsert = item.InsertText[len(prefix):]
	}
	
	// Insert the remaining completion text
	for _, ch := range textToInsert {
		if err := e.insertChar(ch); err != nil {
			return err
		}
	}
	
	// Hide the menu
	e.completionMenu.Hide()
	
	return nil
}

// hideCompletionMenu hides the completion menu
func (e *Editor) hideCompletionMenu() {
	e.completionMenu.Hide()
}

// getWordBeforeCursor returns the word before the cursor position
func (e *Editor) getWordBeforeCursor() string {
	if e.buffer == nil || e.buffer.CursorIterator.IsEOF() {
		return ""
	}
	
	line := e.buffer.CursorIterator.Line()
	cursorX := e.buffer.CursorX
	
	// Find start of word
	start := cursorX
	for start > 0 && start <= len(line) {
		if start-1 < len(line) && isWordChar(rune(line[start-1])) {
			start--
		} else {
			break
		}
	}
	
	// Extract word
	if start < cursorX && cursorX <= len(line) {
		return line[start:cursorX]
	}
	
	return ""
}

// handleCompletionMenuKey handles key events when the completion menu is visible
func (e *Editor) handleCompletionMenuKey(ev *events.KeyEvent) bool {
	switch ev.Key {
	case events.KeyEscape:
		e.hideCompletionMenu()
		return true
	case events.KeyEnter:
		e.acceptCompletion()
		return true
	case events.KeyUp:
		e.completionMenu.MoveUp()
		return true
	case events.KeyDown:
		e.completionMenu.MoveDown()
		return true
	case events.KeyBackspace, events.KeyBackspace2:
		// Let backspace be processed normally, then update filter
		return false
	case events.KeyRune:
		// Let the character be inserted, then update filter
		return false
	default:
		// For other special keys (arrows, etc), hide the menu
		e.hideCompletionMenu()
		return false
	}
}

// updateCompletionFilter updates the completion menu filter based on current word
func (e *Editor) updateCompletionFilter() {
	if !e.completionMenu.IsVisible() {
		return
	}
	
	prefix := e.getWordBeforeCursor()
	e.completionMenu.SetFilterText(prefix)
}