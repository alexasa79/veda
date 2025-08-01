package editor

import (
	"fmt"
	"veda/internal/commands"
)

// registerKeyBinding registers a key binding with a command for a single mode
func (e *Editor) registerKeyBinding(mode commands.Mode, keys string, command string) {
	e.registerKeyBindingForModes([]commands.Mode{mode}, keys, command)
}

// registerKeyBindingForModes registers a key binding with a command for multiple modes
func (e *Editor) registerKeyBindingForModes(modes []commands.Mode, keys string, command string) {
	keySeq, err := commands.ParseKeySequence(keys)
	if err != nil {
		panic(fmt.Sprintf("Invalid key sequence: %s", keys))
	}

	e.keyMapper.AddBinding(commands.Binding{
		Keys:    keySeq,
		Command: command,
		Modes:   modes,
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
	e.registerKeyBinding(commands.NormalMode, "V", "enter-visual-line-mode")
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
	
	// Paragraph rewrapping in insert mode
	e.registerKeyBinding(commands.InsertMode, "alt+q", "rewrap-paragraph")

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
	e.registerKeyBinding(commands.NormalMode, "g e", "diagnostics")
	e.registerKeyBinding(commands.NormalMode, "g t", "show-project-changes")

	// Undo/Redo operations
	e.registerKeyBinding(commands.NormalMode, "u", "undo")
	e.registerKeyBinding(commands.NormalMode, "U", "redo")

	// Yank operations
	e.registerKeyBinding(commands.NormalMode, "y", "start-yank")

	// Paste operations
	e.registerKeyBinding(commands.NormalMode, "p", "paste-after")
	e.registerKeyBinding(commands.NormalMode, "P", "paste-before")

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
	e.registerKeyBinding(commands.NormalMode, "escape", "clear-search")
	e.registerKeyBinding(commands.NormalMode, "*", "search-word-under-cursor")

	// Visual modes (both character and line) - shared keybindings
	visualModes := []commands.Mode{commands.VisualMode, commands.VisualLineMode}

	// Exit visual mode
	e.registerKeyBindingForModes(visualModes, "escape", "exit-visual-mode")

	// Movement in visual modes - reuse same commands
	e.registerKeyBindingForModes(visualModes, "up", "up")
	e.registerKeyBindingForModes(visualModes, "down", "down")
	e.registerKeyBindingForModes(visualModes, "left", "left")
	e.registerKeyBindingForModes(visualModes, "right", "right")
	e.registerKeyBindingForModes(visualModes, "w", "word-forward")
	e.registerKeyBindingForModes(visualModes, "b", "word-backward")
	e.registerKeyBindingForModes(visualModes, "0", "beginning-of-line")
	e.registerKeyBindingForModes(visualModes, "$", "end-of-line")
	e.registerKeyBindingForModes(visualModes, "home", "beginning-of-line")
	e.registerKeyBindingForModes(visualModes, "end", "end-of-line")
	e.registerKeyBindingForModes(visualModes, "g g", "go-to-beginning")
	e.registerKeyBindingForModes(visualModes, "G", "go-to-end")
	e.registerKeyBindingForModes(visualModes, "pageup", "page-up")
	e.registerKeyBindingForModes(visualModes, "pagedown", "page-down")

	// Visual mode operations
	e.registerKeyBindingForModes(visualModes, "d", "delete-selection")
	e.registerKeyBindingForModes(visualModes, "x", "delete-selection")
	e.registerKeyBindingForModes(visualModes, "y", "yank-selection")
	e.registerKeyBindingForModes(visualModes, "c", "change-selection")
}
