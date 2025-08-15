package editor

import (
	"veda/internal/commands"
	"veda/internal/widgets"
)

// getCommandDescriptions returns a map of command names to user-friendly descriptions
func (e *Editor) getCommandDescriptions() map[string]string {
	return map[string]string{
		// Movement commands
		"word-forward":               "next word",
		"word-backward":              "previous word", 
		"word-forward-no-underscore": "next word (underscore boundary)",
		"word-backward-no-underscore":"previous word (underscore boundary)",
		"beginning-of-line":          "beginning of line",
		"end-of-line":                "end of line",
		"up":                         "move up",
		"down":                       "move down",
		"left":                       "move left",
		"right":                      "move right",
		"page-up":                    "page up",
		"page-down":                  "page down",
		"go-to-beginning":            "beginning of file",
		"go-to-end":                  "end of file",
		
		// Mode commands
		"enter-command-mode":         "command mode",
		"enter-search-mode":          "search mode",
		"enter-insert-mode":          "insert mode",
		"append-after-cursor":        "append after cursor",
		"append-at-end-of-line":      "append at end of line",
		"enter-visual-mode":          "visual mode",
		"enter-visual-line-mode":     "visual line mode",
		"open-line-below":            "open line below",
		"open-line-above":            "open line above",
		"join-lines":                 "join lines",
		"enter-cursor-mode":          "cursor mode",
		
		// Edit commands
		"delete-current-character":   "delete character",
		"delete-to-end-of-line":      "delete to end of line",
		"change-to-end-of-line":      "change to end of line",
		"start-delete":               "delete (with motion)",
		"start-change":               "change (with motion)",
		"start-yank":                 "yank (copy)",
		"paste-after":                "paste after",
		"paste-before":               "paste before",
		
		// Undo/Redo
		"undo":                       "undo",
		"redo":                       "redo",
		"repeat-last-change":         "repeat last change",
		
		// LSP commands
		"go-to-definition":           "go to definition",
		"find-references":            "find references",
		"show-signature":             "show signature",
		"diagnostics":                "show diagnostics",
		
		// File/Buffer commands
		"open-file":                  "open file",
		"show-buffers":               "show buffers",
		"switch-to-previous-buffer":  "previous buffer",
		"new-buffer":                 "new buffer",
		"show-project-changes":       "project changes",
		
		// Search commands
		"find-next":                  "find next",
		"find-previous":              "find previous",
		"clear-search":               "clear search",
		"search-word-under-cursor":   "search word under cursor",
		
		// Symbol commands
		"list-project-symbols":       "project symbols",
		"list-document-symbols":      "document symbols",
		
		// Navigation
		"go-back":                    "go back",
		"next-reference":             "next reference",
		"previous-reference":         "previous reference",
		"jump-to-reference":          "jump to reference",
		
		// Indentation
		"indent-line":                "indent line",
		"unindent-line":              "unindent line",
		
		// Character commands
		"find-character":             "find character",
		"till-character":             "till character",
		"start-find-character":       "find character",
		"start-till-character":       "till character",
		
		// Calculation
		"calculate-append":           "calculate and append",
		"calculate-replace":          "calculate and replace",
		
		// Rewrap
		"rewrap-paragraph":           "rewrap paragraph",
	}
}

// showKeyHints displays the key hints popup for the current pending key sequence
func (e *Editor) showKeyHints() {
	// Only show hints in normal mode
	if e.mode != commands.NormalMode {
		e.keyHintsPopup.Hide()
		return
	}
	
	// Get pending keys from key mapper
	pendingKeys := e.keyMapper.GetPendingKeys()
	if len(pendingKeys) == 0 {
		e.keyHintsPopup.Hide()
		return
	}
	
	// Get available key hints
	commandDescs := e.getCommandDescriptions()
	hints := widgets.GetAvailableKeyHints(e.keyMapper, pendingKeys, e.mode, commandDescs)
	
	if len(hints) == 0 {
		e.keyHintsPopup.Hide()
		return
	}
	
	// Position popup in top-right corner
	screenWidth, _ := e.screen.Size()
	popupX := screenWidth - 35 // Leave some margin from the right edge
	if popupX < 0 {
		popupX = 0
	}
	popupY := 1 // Just below the top edge
	
	e.keyHintsPopup.SetPosition(popupX, popupY)
	e.keyHintsPopup.SetHints(hints)
	e.keyHintsPopup.Show()
}

