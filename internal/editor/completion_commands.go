package editor

import (
	"strings"
	"veda/internal/events"
	"veda/internal/widgets"
)

// showCompletions shows the completion menu with LSP completions
func (e *Editor) showCompletions() error {
	// Check if we have a file open
	if !e.buffer.HasFile() {
		LL.Debug("No file open, cannot get completions")
		return nil
	}
	
	// Get the current cursor position
	line := e.buffer.CursorIterator.LineNumber()
	character := e.buffer.CursorX
	
	// Get the current word prefix to use as initial filter
	prefix := e.getWordBeforeCursor()
	
	// Track as async operation to show spinner
	operationID := "completions"
	e.startAsyncOperation(operationID, "fetching code completions for: "+prefix)
	
	// Launch goroutine to fetch completions
	go func() {
		defer e.endAsyncOperation(operationID)
		
		// Get completions from LSP
		result, err := e.proxyManager.GetCompletions(e.buffer.FilePath, line, character)
		if err != nil {
			LL.Debug("Failed to get completions: %v", err)
			return // Don't show error to user, just no completions
		}
		
		// Send result back to main event loop
		e.asyncEvents <- events.NewCustomEvent("show-completions", map[string]interface{}{
			"result": result,
			"prefix": prefix,
		})
	}()
	
	return nil
}

// handleShowCompletions handles the async completion result
func (e *Editor) handleShowCompletions(data map[string]interface{}) {
	result, ok := data["result"]
	if !ok {
		LL.Debug("No result in completion data")
		return
	}
	
	prefix, _ := data["prefix"].(string)
	
	// Convert LSP completion result to our format
	completionItems := e.convertLSPCompletions(result)
	if len(completionItems) == 0 {
		LL.Debug("No completions available")
		return
	}
	
	// Set the items
	e.completionMenu.SetItems(completionItems)
	
	// Set initial filter based on word before cursor
	e.completionMenu.SetFilterText(prefix)
	
	// Calculate menu position (below cursor)
	menuX, menuY := e.getCursorScreenPosition()
	menuY++ // Position below cursor
	
	// Adjust position if too close to bottom
	screenWidth, screenHeight := e.screen.Size()
	menuHeight := len(completionItems) // Calculate based on items we set
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

// convertLSPCompletions converts LSP completion response to widget CompletionItems
func (e *Editor) convertLSPCompletions(result interface{}) []widgets.CompletionItem {
	var items []widgets.CompletionItem
	
	// LSP can return either a CompletionList or array of CompletionItems
	switch v := result.(type) {
	case map[string]interface{}:
		// This is a CompletionList with isIncomplete flag and items array
		if itemsRaw, ok := v["items"].([]interface{}); ok {
			items = e.convertCompletionItemArray(itemsRaw)
		}
	case []interface{}:
		// This is directly an array of CompletionItems
		items = e.convertCompletionItemArray(v)
	}
	
	return items
}

// convertCompletionItemArray converts an array of LSP CompletionItems
func (e *Editor) convertCompletionItemArray(itemsRaw []interface{}) []widgets.CompletionItem {
	var items []widgets.CompletionItem
	
	for _, itemRaw := range itemsRaw {
		if item, ok := itemRaw.(map[string]interface{}); ok {
			// Extract label (required)
			label, ok := item["label"].(string)
			if !ok || label == "" {
				continue
			}
			
			// Extract insertText or use label
			insertText := label
			if text, ok := item["insertText"].(string); ok && text != "" {
				insertText = text
			}
			
			// Extract detail for additional info
			detail := ""
			if d, ok := item["detail"].(string); ok {
				detail = d
			}
			
			// Create our completion item
			items = append(items, widgets.CompletionItem{
				Label:      label,
				Detail:     detail,
				InsertText: insertText,
			})
		}
	}
	
	return items
}