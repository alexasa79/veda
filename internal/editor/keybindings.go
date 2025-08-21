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
	// Vim-style movement keys
	e.registerKeyBinding(commands.NormalMode, "h", "left")
	e.registerKeyBinding(commands.NormalMode, "j", "down")
	e.registerKeyBinding(commands.NormalMode, "k", "up")
	e.registerKeyBinding(commands.NormalMode, "l", "right")
	e.registerKeyBinding(commands.NormalMode, "w", "word-forward")
	e.registerKeyBinding(commands.NormalMode, "b", "word-backward")
	e.registerKeyBinding(commands.NormalMode, "W", "word-forward-no-underscore")
	e.registerKeyBinding(commands.NormalMode, "B", "word-backward-no-underscore")
	e.registerKeyBinding(commands.NormalMode, "f", "start-find-character")
	e.registerKeyBinding(commands.NormalMode, "t", "start-till-character")
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
	e.registerKeyBinding(commands.NormalMode, "m", "enter-cursor-mode")

	// Delete commands
	e.registerKeyBinding(commands.NormalMode, "x", "delete-current-character")
	e.registerKeyBinding(commands.NormalMode, "D", "delete-to-end-of-line")
	// Change-to-end-of-line (Vim 'C')
	e.registerKeyBinding(commands.NormalMode, "C", "change-to-end-of-line")

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
	e.registerKeyBinding(commands.NormalMode, "s f", "open-file")
	// Open terminal
	e.registerKeyBinding(commands.NormalMode, "space t", "term")
	// Open terminal in a new window (right split)
	e.registerKeyBinding(commands.NormalMode, "space T", "term-new-window")

	// Buffer operations
	e.registerKeyBinding(commands.NormalMode, "s b", "show-buffers")
	e.registerKeyBinding(commands.NormalMode, "space space", "switch-to-previous-buffer")
	e.registerKeyBinding(commands.NormalMode, "space n", "new-buffer")
	e.registerKeyBinding(commands.NormalMode, "space x", "q")

	// Window operations
	e.registerKeyBinding(commands.NormalMode, "space v", "vsplit")
	// Window selection: space 1..9 -> window 1..9
	e.registerKeyBinding(commands.NormalMode, "space 1", "window-1")
	e.registerKeyBinding(commands.NormalMode, "space 2", "window-2")
	e.registerKeyBinding(commands.NormalMode, "space 3", "window-3")
	e.registerKeyBinding(commands.NormalMode, "space 4", "window-4")
	e.registerKeyBinding(commands.NormalMode, "space 5", "window-5")
	e.registerKeyBinding(commands.NormalMode, "space 6", "window-6")
	e.registerKeyBinding(commands.NormalMode, "space 7", "window-7")
	e.registerKeyBinding(commands.NormalMode, "space 8", "window-8")
	e.registerKeyBinding(commands.NormalMode, "space 9", "window-9")
	e.registerKeyBinding(commands.NormalMode, "space o", "window-last")
	e.registerKeyBinding(commands.NormalMode, "space tab", "window-next")

	// Move buffer between windows
	e.registerKeyBinding(commands.NormalMode, "space w left", "move-buffer-left")
	e.registerKeyBinding(commands.NormalMode, "space w right", "move-buffer-right")
	e.registerKeyBinding(commands.NormalMode, "space w c", "move-buffer-center")
	e.registerKeyBinding(commands.NormalMode, "space w x", "close-window")

	// Symbol operations
	e.registerKeyBinding(commands.NormalMode, "s w", "list-project-symbols")
	e.registerKeyBinding(commands.NormalMode, "s d", "list-document-symbols")

	// LSP operations
	e.registerKeyBinding(commands.NormalMode, "g d", "go-to-definition")
	e.registerKeyBinding(commands.NormalMode, "g space d", "go-to-definition-new-window")
	e.registerKeyBinding(commands.NormalMode, "s s", "show-signature")
	e.registerKeyBinding(commands.NormalMode, "g r", "find-references")
	e.registerKeyBinding(commands.NormalMode, "g e", "diagnostics")
	e.registerKeyBinding(commands.NormalMode, "s c", "show-project-changes")

	// Undo/Redo operations
	e.registerKeyBinding(commands.NormalMode, "u", "undo")
	e.registerKeyBinding(commands.NormalMode, "U", "redo")

	// Yank operations
	e.registerKeyBinding(commands.NormalMode, "y", "start-yank")
	// Provide a direct binding for y$ for ergonomics (optional; operators already support $)
	// This binds the two-key sequence y + $ to a dedicated yank-to-EOL command.
	// It's redundant but harmless and can improve keymap lookup speed in tests.
	e.registerKeyBinding(commands.NormalMode, "y $", "yank-to-end-of-line")

	// Paste operations
	e.registerKeyBinding(commands.NormalMode, "p", "paste-after")
	e.registerKeyBinding(commands.NormalMode, "P", "paste-before")

	// Repeat last change
	e.registerKeyBinding(commands.NormalMode, ".", "repeat-last-change")

	// Delete operations
	e.registerKeyBinding(commands.NormalMode, "d", "start-delete")

	// Change operations
	e.registerKeyBinding(commands.NormalMode, "c", "start-change")

	// Indentation operations
	e.registerKeyBinding(commands.NormalMode, "> >", "indent-line")
	e.registerKeyBinding(commands.NormalMode, "< <", "unindent-line")
	// Reindent current line (vim '==')
	e.registerKeyBinding(commands.NormalMode, "= =", "reindent-line")
	// Reindent expression (vim '=' over a motion/obj) – map '= =' again to expression reindent cycling
	e.registerKeyBinding(commands.NormalMode, "=", "reindent-expression")

	// Navigation operations
	e.registerKeyBinding(commands.NormalMode, "g ,", "go-back")
	e.registerKeyBinding(commands.NormalMode, "g g", "go-to-beginning")
	e.registerKeyBinding(commands.NormalMode, "g n", "next-reference")
	e.registerKeyBinding(commands.NormalMode, "g p", "previous-reference")
	e.registerKeyBinding(commands.NormalMode, "G", "go-to-end")

	// Special handling for references buffers
	e.registerKeyBinding(commands.NormalMode, "enter", "jump-to-reference")

	// Calculator operations
	e.registerKeyBinding(commands.NormalMode, "- -", "calculate-append")
	e.registerKeyBinding(commands.NormalMode, "- =", "calculate-replace")

	// Info
	e.registerKeyBinding(commands.NormalMode, "ctrl+g", "show-file-info")

	// Temporary normal mode from insert mode (like vim's Ctrl+O)
	e.registerKeyBinding(commands.InsertMode, "ctrl+o", "enter-temporary-normal-mode")

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
	e.registerKeyBindingForModes(visualModes, "W", "word-forward-no-underscore")
	e.registerKeyBindingForModes(visualModes, "B", "word-backward-no-underscore")
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
	// Visual mode indentation
	e.registerKeyBindingForModes(visualModes, "> >", "indent-selection")
	e.registerKeyBindingForModes(visualModes, "< <", "unindent-selection")

	// Shift+Arrow: enter visual mode and extend selection
	shiftArrowModes := []commands.Mode{commands.NormalMode, commands.InsertMode, commands.VisualMode, commands.VisualLineMode}
	e.registerKeyBindingForModes(shiftArrowModes, "shift+left", "start-visual-left")
	e.registerKeyBindingForModes(shiftArrowModes, "shift+right", "start-visual-right")
	e.registerKeyBindingForModes(shiftArrowModes, "shift+up", "start-visual-up")
	e.registerKeyBindingForModes(shiftArrowModes, "shift+down", "start-visual-down")
}
