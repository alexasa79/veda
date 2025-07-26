package editor

// registerCommands registers all command handlers
func (e *Editor) registerCommands() {
	// Movement commands
	e.commandRegistry.Register("up", e.moveUp)
	e.commandRegistry.Register("down", e.moveDown)
	e.commandRegistry.Register("left", e.moveLeft)
	e.commandRegistry.Register("right", e.moveRight)
	e.commandRegistry.Register("word-forward", e.wordForward)
	e.commandRegistry.Register("word-backward", e.wordBackward)
	e.commandRegistry.Register("beginning-of-line", e.beginningOfLine)
	e.commandRegistry.Register("end-of-line", e.endOfLine)
	e.commandRegistry.Register("beginning-of-line-insert", e.beginningOfLineInsert)
	e.commandRegistry.Register("end-of-line-insert", e.endOfLineInsert)
	e.commandRegistry.Register("page-up", e.pageUp)
	e.commandRegistry.Register("page-down", e.pageDown)
	e.commandRegistry.Register("go-to-beginning", e.goToBeginning)
	e.commandRegistry.Register("go-to-end", e.goToEnd)

	// Insert mode movement commands
	e.commandRegistry.Register("up-insert", e.moveUpInsert)
	e.commandRegistry.Register("down-insert", e.moveDownInsert)
	e.commandRegistry.Register("left-insert", e.moveLeftInsert)
	e.commandRegistry.Register("right-insert", e.moveRightInsert)

	// Mode commands
	e.commandRegistry.Register("enter-command-mode", e.enterCommandModeCmd)
	e.commandRegistry.Register("enter-insert-mode", e.enterInsertMode)
	e.commandRegistry.Register("exit-insert-mode", e.exitInsertMode)
	e.commandRegistry.Register("open-line-below", e.openLineBelow)
	e.commandRegistry.Register("open-line-above", e.openLineAbove)

	// Insert mode commands
	e.commandRegistry.Register("insert-newline", e.insertNewline)
	e.commandRegistry.Register("backspace", e.backspace)

	// Delete commands
	e.commandRegistry.Register("delete-current-character", e.deleteCurrentCharacter)

	// File commands
	e.commandRegistry.Register("open-file", e.openFileCmd)
	e.commandRegistry.Register("quit", e.quit)

	// LSP commands
	e.commandRegistry.Register("go-to-definition", e.goToDefinition)
	e.commandRegistry.Register("show-lsp-buffer-go", e.showLSPBufferGo)
	e.commandRegistry.Register("show-signature", e.showSignature)

	// Navigation commands
	e.commandRegistry.Register("go-back", e.goBack)

	// Buffer commands
	e.commandRegistry.Register("show-buffers", e.showBuffers)

	// Symbol commands
	e.commandRegistry.Register("list-project-symbols", e.listProjectSymbolsCmd)
	e.commandRegistry.Register("list-document-symbols", e.listDocumentSymbolsCmd)

	// Undo/Redo commands
	e.commandRegistry.Register("undo", e.undo)
	e.commandRegistry.Register("redo", e.redo)
	
	// Repeat command
	e.commandRegistry.Register("repeat-last-change", e.repeatLastChange)
}