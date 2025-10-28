package widgets

import (
	"sort"
	"strings"
	"veda/internal/events"
	"veda/internal/ui"
)

// CommandLineWidget wraps LineEditor with command-specific features like history and tab completion
type CommandLineWidget struct {
	editor           *LineEditor
	history          []string
	historyIndex     int
	tempText         string
	completionFunc   func(string) []string
	completions      []string
	completionIndex  int
	completionPrefix string
	lastWasTab       bool
}

// NewCommandLineWidget creates a new command line widget
func NewCommandLineWidget(x, y, width int) *CommandLineWidget {
	return &CommandLineWidget{
		editor:       NewLineEditor(x, y, width),
		history:      make([]string, 0),
		historyIndex: -1,
		tempText:     "",
	}
}

// SetStyle sets the display style for the command line
func (cl *CommandLineWidget) SetStyle(style ui.Style) {
	cl.editor.SetStyle(style)
}

// SetText sets the text content of the command line
func (cl *CommandLineWidget) SetText(text string) {
	cl.editor.SetText(text)
	cl.historyIndex = -1
	cl.tempText = ""
	// Reset completion state when text is manually set
	cl.completions = nil
	cl.completionIndex = 0
	cl.completionPrefix = ""
	cl.lastWasTab = false
}

// GetText returns the current text content
func (cl *CommandLineWidget) GetText() string {
	return cl.editor.GetText()
}

// SetCursor sets the cursor position within the text
func (cl *CommandLineWidget) SetCursor(pos int) {
	cl.editor.SetCursor(pos)
}

// GetCursor returns the cursor position within the text.
func (cl *CommandLineWidget) GetCursor() int {
	return cl.editor.GetCursor()
}

// Draw renders the command line on the screen
func (cl *CommandLineWidget) Draw(screen ScreenWriter) {
	cl.editor.Draw(screen)
}

// GetCursorPosition returns the screen coordinates where the cursor should be displayed
func (cl *CommandLineWidget) GetCursorPosition() (int, int) {
	return cl.editor.GetCursorPosition()
}

// SetPosition updates the position of the command line
func (cl *CommandLineWidget) SetPosition(x, y int) {
	cl.editor.SetPosition(x, y)
}

// SetWidth updates the width of the command line
func (cl *CommandLineWidget) SetWidth(width int) {
	cl.editor.SetWidth(width)
}

// AddToHistory adds a command to the history
func (cl *CommandLineWidget) AddToHistory(command string) {
	// Remove leading colon if present
	if strings.HasPrefix(command, ":") {
		command = command[1:]
	}

	// Don't add empty commands or duplicates of the last command
	if command == "" || (len(cl.history) > 0 && cl.history[len(cl.history)-1] == command) {
		return
	}

	cl.history = append(cl.history, command)
}

// SetCompletionFunc sets the function to call for tab completion
func (cl *CommandLineWidget) SetCompletionFunc(f func(string) []string) {
	cl.completionFunc = f
}

// HandleKey processes keyboard input for the command line
func (cl *CommandLineWidget) HandleKey(ev *events.KeyEvent) bool {
	// Track if this is a Tab key
	isTab := ev.Key == events.KeyTab

	// If the last key wasn't Tab, reset completion state
	if !isTab && cl.lastWasTab {
		cl.completions = nil
		cl.completionIndex = 0
		cl.completionPrefix = ""
		cl.lastWasTab = false
	}

	switch ev.Key {
	case events.KeyUp:
		return cl.navigateHistoryUp()
	case events.KeyDown:
		return cl.navigateHistoryDown()
	case events.KeyTab:
		cl.lastWasTab = true
		return cl.handleTabCompletion()
	default:
		// For any other key, reset history navigation
		if cl.historyIndex != -1 {
			cl.historyIndex = -1
			cl.tempText = ""
		}
		return cl.editor.HandleKey(ev)
	}
}

// navigateHistoryUp moves up in the command history
func (cl *CommandLineWidget) navigateHistoryUp() bool {
	if len(cl.history) == 0 {
		return true
	}

	// If we're at the current input, save it
	if cl.historyIndex == -1 {
		text := cl.editor.GetText()
		// Remove leading colon
		if strings.HasPrefix(text, ":") {
			text = text[1:]
		}
		cl.tempText = text
		cl.historyIndex = len(cl.history) - 1
	} else if cl.historyIndex > 0 {
		cl.historyIndex--
	}

	// Set the historical command with colon prefix
	cl.editor.SetText(":" + cl.history[cl.historyIndex])
	cl.editor.SetCursor(len(cl.editor.GetText()))
	return true
}

// navigateHistoryDown moves down in the command history
func (cl *CommandLineWidget) navigateHistoryDown() bool {
	if cl.historyIndex == -1 {
		return true
	}

	if cl.historyIndex < len(cl.history)-1 {
		cl.historyIndex++
		cl.editor.SetText(":" + cl.history[cl.historyIndex])
	} else {
		// Return to the temporary text
		cl.historyIndex = -1
		cl.editor.SetText(":" + cl.tempText)
	}

	cl.editor.SetCursor(len(cl.editor.GetText()))
	return true
}

// handleTabCompletion performs tab completion
func (cl *CommandLineWidget) handleTabCompletion() bool {
	if cl.completionFunc == nil {
		return false
	}

	text := cl.editor.GetText()
	// Remove leading colon
	if strings.HasPrefix(text, ":") {
		text = text[1:]
	}

	// Check if we need to get new completions
	// We need new completions if:
	// 1. We don't have any completions yet
	// 2. The text is different from what we started completing AND we're not cycling
	needNewCompletions := cl.completions == nil

	// If we're not cycling yet, check if the text changed
	if cl.completionIndex == 0 && text != cl.completionPrefix {
		needNewCompletions = true
	}

	if needNewCompletions {
		cl.completions = cl.completionFunc(text)
		cl.completionIndex = 0
		cl.completionPrefix = text

		if len(cl.completions) == 0 {
			return true
		}

		// Sort completions for predictable cycling order
		sort.Strings(cl.completions)
	}

	// Single completion - just use it
	if len(cl.completions) == 1 {
		cl.editor.SetText(":" + cl.completions[0])
		cl.editor.SetCursor(len(cl.editor.GetText()))
		return true
	}

	// Multiple completions - cycle through them
	if len(cl.completions) > 1 {
		// Use modulo to cycle through completions
		idx := cl.completionIndex % len(cl.completions)
		cl.editor.SetText(":" + cl.completions[idx])
		cl.editor.SetCursor(len(cl.editor.GetText()))

		// Move to next index
		cl.completionIndex++
	}

	return true
}

// findCommonPrefix finds the longest common prefix among strings
func findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		// Shorten prefix until it matches
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if len(prefix) == 0 {
			break
		}
	}

	return prefix
}
