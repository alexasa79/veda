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
	e.registerKeyBinding(commands.NormalMode, "o", "open-line-below")
	e.registerKeyBinding(commands.NormalMode, "O", "open-line-above")

	// Delete commands
	e.registerKeyBinding(commands.NormalMode, "x", "delete-current-character")

	// Insert mode keys
	e.registerKeyBinding(commands.InsertMode, "enter", "insert-newline")
	e.registerKeyBinding(commands.InsertMode, "backspace", "backspace")
	e.registerKeyBinding(commands.InsertMode, "escape", "exit-insert-mode")

	// Cursor movement in insert mode
	e.registerKeyBinding(commands.InsertMode, "up", "up-insert")
	e.registerKeyBinding(commands.InsertMode, "down", "down-insert")
	e.registerKeyBinding(commands.InsertMode, "left", "left-insert")
	e.registerKeyBinding(commands.InsertMode, "right", "right-insert")
	e.registerKeyBinding(commands.InsertMode, "home", "beginning-of-line-insert")
	e.registerKeyBinding(commands.InsertMode, "end", "end-of-line-insert")

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

	// Undo/Redo operations
	e.registerKeyBinding(commands.NormalMode, "u", "undo")
	e.registerKeyBinding(commands.NormalMode, "U", "redo")
	
	// Repeat last change
	e.registerKeyBinding(commands.NormalMode, ".", "repeat-last-change")
	
	// Delete operations
	e.registerKeyBinding(commands.NormalMode, "d", "start-delete")
	e.registerKeyBinding(commands.NormalMode, "ctrl+shift+space", "show-signature")
	e.registerKeyBinding(commands.InsertMode, "ctrl+shift+space", "show-signature")

	// Navigation operations
	e.registerKeyBinding(commands.NormalMode, "g ,", "go-back")
	e.registerKeyBinding(commands.NormalMode, "g g", "go-to-beginning")
	e.registerKeyBinding(commands.NormalMode, "G", "go-to-end")
	
	// Search navigation
	e.registerKeyBinding(commands.NormalMode, "n", "find-next")
	e.registerKeyBinding(commands.NormalMode, "N", "find-previous")
}
