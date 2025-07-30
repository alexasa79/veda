package editor

import (
	"fmt"
	"veda/internal/commands"
)

// registerKeyBinding registers a key binding with a command
func (e *Editor) registerKeyBinding(mode commands.Mode, keys string, command string) {
	keySeq, err := commands.ParseKeySequence(keys)
	if err != nil {
		panic(fmt.Sprintf("Invalid key sequence: %s", keys))
	}

	e.keyMapper.AddBinding(commands.Binding{
		Keys:    keySeq,
		Command: command,
		Modes:   []commands.Mode{mode},
	})
}

// setupDefaultBindings sets up the default key bindings
func (e *Editor) setupDefaultBindings() {
	// Movement keys
	e.registerKeyBinding(commands.NormalMode, "up", "up")
	e.registerKeyBinding(commands.NormalMode, "down", "down")
	e.registerKeyBinding(commands.NormalMode, "left", "left")
	e.registerKeyBinding(commands.NormalMode, "right", "right")
	e.registerKeyBinding(commands.NormalMode, "pageup", "page-up")
	e.registerKeyBinding(commands.NormalMode, "pagedown", "page-down")
	e.registerKeyBinding(commands.NormalMode, "w", "word-forward")
	e.registerKeyBinding(commands.NormalMode, "b", "word-backward")
	e.registerKeyBinding(commands.NormalMode, "0", "beginning-of-line")
	e.registerKeyBinding(commands.NormalMode, "$", "end-of-line")
	e.registerKeyBinding(commands.NormalMode, "home", "beginning-of-line")
	e.registerKeyBinding(commands.NormalMode, "end", "end-of-line")

	// Mode switching
	e.registerKeyBinding(commands.NormalMode, ":", "enter-command-mode")
	e.registerKeyBinding(commands.NormalMode, "/", "enter-search-mode")
	e.registerKeyBinding(commands.NormalMode, "i", "enter-insert-mode")
	e.registerKeyBinding(commands.NormalMode, "a", "append-after-cursor")
	e.registerKeyBinding(commands.NormalMode, "A", "append-at-end-of-line")
	e.registerKeyBinding(commands.NormalMode, "v", "enter-visual-mode")
	e.registerKeyBinding(commands.NormalMode, "o", "open-line-below")
	e.registerKeyBinding(commands.NormalMode, "O", "open-line-above")
	e.registerKeyBinding(commands.NormalMode, "J", "join-lines")
	e.registerKeyBinding(commands.NormalMode, "alt+q", "rewrap-paragraph")

	// Delete commands
	e.registerKeyBinding(commands.NormalMode, "x", "delete-current-character")

	// Insert mode keys
	e.registerKeyBinding(commands.InsertMode, "enter", "insert-newline")
	e.registerKeyBinding(commands.InsertMode, "backspace", "backspace")
	e.registerKeyBinding(commands.InsertMode, "escape", "exit-insert-mode")
	e.registerKeyBinding(commands.InsertMode, "delete", "delete-character-insert")
	e.registerKeyBinding(commands.InsertMode, "alt+delete", "delete-word-forward")
	e.registerKeyBinding(commands.InsertMode, "alt+backspace", "delete-word-backward")
	e.registerKeyBinding(commands.InsertMode, "tab", "insert-tab")

	// Cursor movement in insert mode
	e.registerKeyBinding(commands.InsertMode, "up", "up-insert")
	e.registerKeyBinding(commands.InsertMode, "down", "down-insert")
	e.registerKeyBinding(commands.InsertMode, "left", "left-insert")
	e.registerKeyBinding(commands.InsertMode, "right", "right-insert")
	e.registerKeyBinding(commands.InsertMode, "home", "beginning-of-line-insert")
	e.registerKeyBinding(commands.InsertMode, "end", "end-of-line-insert")
	e.registerKeyBinding(commands.InsertMode, "alt+left", "word-backward-insert")
	e.registerKeyBinding(commands.InsertMode, "alt+right", "word-forward-insert")
	
	// Completion
	e.registerKeyBinding(commands.InsertMode, "ctrl+space", "show-completions")

	// File operations
	e.registerKeyBinding(commands.NormalMode, "space s f", "open-file")

	// Buffer operations
	e.registerKeyBinding(commands.NormalMode, "space s b", "show-buffers")
	e.registerKeyBinding(commands.NormalMode, "space space", "switch-to-previous-buffer")

	// Symbol operations
	e.registerKeyBinding(commands.NormalMode, "space s w", "list-project-symbols")
	e.registerKeyBinding(commands.NormalMode, "space s d", "list-document-symbols")

	// LSP operations
	e.registerKeyBinding(commands.NormalMode, "g d", "go-to-definition")
	e.registerKeyBinding(commands.NormalMode, "g s", "show-signature")
	e.registerKeyBinding(commands.NormalMode, "g r", "find-references")

	// Undo/Redo operations
	e.registerKeyBinding(commands.NormalMode, "u", "undo")
	e.registerKeyBinding(commands.NormalMode, "U", "redo")

	// Yank operations
	e.registerKeyBinding(commands.NormalMode, "y", "start-yank")

	// Repeat last change
	e.registerKeyBinding(commands.NormalMode, ".", "repeat-last-change")

	// Delete operations
	e.registerKeyBinding(commands.NormalMode, "d", "start-delete")
	
	// Change operations
	e.registerKeyBinding(commands.NormalMode, "c", "start-change")

	// Navigation operations
	e.registerKeyBinding(commands.NormalMode, "g ,", "go-back")
	e.registerKeyBinding(commands.NormalMode, "g g", "go-to-beginning")
	e.registerKeyBinding(commands.NormalMode, "g n", "next-reference")
	e.registerKeyBinding(commands.NormalMode, "g p", "previous-reference")
	e.registerKeyBinding(commands.NormalMode, "G", "go-to-end")
	
	// Special handling for references buffers
	e.registerKeyBinding(commands.NormalMode, "enter", "jump-to-reference")

	// Search navigation
	e.registerKeyBinding(commands.NormalMode, "n", "find-next")
	e.registerKeyBinding(commands.NormalMode, "N", "find-previous")

	// Visual mode keys
	e.registerKeyBinding(commands.VisualMode, "escape", "exit-visual-mode")
	// Movement in visual mode - reuse same commands
	e.registerKeyBinding(commands.VisualMode, "up", "up")
	e.registerKeyBinding(commands.VisualMode, "down", "down")
	e.registerKeyBinding(commands.VisualMode, "left", "left")
	e.registerKeyBinding(commands.VisualMode, "right", "right")
	e.registerKeyBinding(commands.VisualMode, "w", "word-forward")
	e.registerKeyBinding(commands.VisualMode, "b", "word-backward")
	e.registerKeyBinding(commands.VisualMode, "0", "beginning-of-line")
	e.registerKeyBinding(commands.VisualMode, "$", "end-of-line")
	e.registerKeyBinding(commands.VisualMode, "home", "beginning-of-line")
	e.registerKeyBinding(commands.VisualMode, "end", "end-of-line")
	e.registerKeyBinding(commands.VisualMode, "g g", "go-to-beginning")
	e.registerKeyBinding(commands.VisualMode, "G", "go-to-end")
	e.registerKeyBinding(commands.VisualMode, "pageup", "page-up")
	e.registerKeyBinding(commands.VisualMode, "pagedown", "page-down")

	// Visual mode operations
	e.registerKeyBinding(commands.VisualMode, "d", "delete-selection")
	e.registerKeyBinding(commands.VisualMode, "x", "delete-selection")
	e.registerKeyBinding(commands.VisualMode, "y", "yank-selection")
	e.registerKeyBinding(commands.VisualMode, "c", "change-selection")
}
