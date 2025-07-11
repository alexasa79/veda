package editor

// registerCommands registers all command handlers
func (e *Editor) registerCommands() {
	// Movement commands - these don't need full redraw unless they cause scrolling
	e.commandRegistry.Register("up", e.moveUp, false)
	e.commandRegistry.Register("down", e.moveDown, false)
	e.commandRegistry.Register("left", e.moveLeft, false)
	e.commandRegistry.Register("right", e.moveRight, false)
	e.commandRegistry.Register("word-forward", e.wordForward, false)
	e.commandRegistry.Register("word-backward", e.wordBackward, false)
	e.commandRegistry.Register("beginning-of-line", e.beginningOfLine, false)
	e.commandRegistry.Register("end-of-line", e.endOfLine, false)
	e.commandRegistry.Register("beginning-of-line-insert", e.beginningOfLineInsert, false)
	e.commandRegistry.Register("end-of-line-insert", e.endOfLineInsert, false)
	e.commandRegistry.Register("page-up", e.pageUp, true) // Page commands always scroll
	e.commandRegistry.Register("page-down", e.pageDown, true) // Page commands always scroll
	e.commandRegistry.Register("go-to-beginning", e.goToBeginning, true) // May scroll
	e.commandRegistry.Register("go-to-end", e.goToEnd, true) // May scroll

	// Insert mode movement commands - these don't need full redraw unless they cause scrolling
	e.commandRegistry.Register("up-insert", e.moveUpInsert, false)
	e.commandRegistry.Register("down-insert", e.moveDownInsert, false)
	e.commandRegistry.Register("left-insert", e.moveLeftInsert, false)
	e.commandRegistry.Register("right-insert", e.moveRightInsert, false)

	// Mode commands
	e.commandRegistry.Register("enter-command-mode", e.enterCommandModeCmd, true) // Status bar changes
	e.commandRegistry.Register("enter-insert-mode", e.enterInsertMode, true) // Status bar changes
	e.commandRegistry.Register("exit-insert-mode", e.exitInsertMode, true) // Status bar changes
	e.commandRegistry.Register("open-line-below", e.openLineBelow, true) // Modifies buffer
	e.commandRegistry.Register("open-line-above", e.openLineAbove, true) // Modifies buffer

	// Insert mode commands
	e.commandRegistry.Register("insert-newline", e.insertNewline, true) // Modifies buffer
	e.commandRegistry.Register("backspace", e.backspace, true) // Modifies buffer

	// File commands
	e.commandRegistry.Register("open-file", e.openFileCmd, true) // Opens dialog
	e.commandRegistry.Register("quit", e.quit, true)

	// LSP commands
	e.commandRegistry.Register("go-to-definition", e.goToDefinition, true) // May change file
	e.commandRegistry.Register("show-lsp-buffer-go", e.showLSPBufferGo, true) // Changes buffer
	
	// Navigation commands
	e.commandRegistry.Register("go-back", e.goBack, true) // May change file
	
	// Buffer commands
	e.commandRegistry.Register("show-buffers", e.showBuffers, true) // Opens dialog
}