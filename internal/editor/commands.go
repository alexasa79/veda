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
	e.commandRegistry.Register("enter-search-mode", e.enterSearchModeCmd)
	e.commandRegistry.Register("enter-insert-mode", e.enterInsertMode)
	e.commandRegistry.Register("exit-insert-mode", e.exitInsertMode)
	e.commandRegistry.Register("enter-visual-mode", e.enterVisualMode)
	e.commandRegistry.Register("exit-visual-mode", e.exitVisualMode)
	e.commandRegistry.Register("open-line-below", e.openLineBelow)
	e.commandRegistry.Register("open-line-above", e.openLineAbove)
	e.commandRegistry.Register("join-lines", e.joinLines)

	// Insert mode commands
	e.commandRegistry.Register("insert-newline", e.insertNewline)
	e.commandRegistry.Register("backspace", e.backspace)

	// Delete commands
	e.commandRegistry.Register("delete-current-character", e.deleteCurrentCharacter)
	
	// Visual mode operations
	e.commandRegistry.Register("delete-selection", e.deleteSelection)
	e.commandRegistry.Register("yank-selection", e.yankSelection)
	e.commandRegistry.Register("change-selection", e.changeSelection)

	// File commands
	e.commandRegistry.Register("open-file", e.openFileCmd)
	e.commandRegistry.Register("q", e.closeBuffer)  // Close current buffer
	e.commandRegistry.Register("q!", e.actuallyQuit)  // Force quit (shortcut for :Q)
	e.commandRegistry.Register("Quit", e.actuallyQuit)  // Actually quit the editor
	e.commandRegistry.Register("Q", e.actuallyQuit)  // Shortcut for Quit
	e.commandRegistry.Register("write", e.writeFile)
	e.commandRegistry.Register("w", e.writeFile)  // Shortcut for write
	e.commandRegistry.Register("edit", e.editFile)
	e.commandRegistry.Register("e", e.editFile)  // Shortcut for edit
	e.commandRegistry.Register("cd", e.changeDirectory)

	// LSP commands
	e.commandRegistry.Register("go-to-definition", e.goToDefinition)
	e.commandRegistry.Register("show-lsp-buffer-go", e.showLSPBufferGo)
	e.commandRegistry.Register("show-signature", e.showSignature)

	// Navigation commands
	e.commandRegistry.Register("go-back", e.goBack)

	// Buffer commands
	e.commandRegistry.Register("show-buffers", e.showBuffers)
	e.commandRegistry.Register("switch-to-previous-buffer", e.switchToPreviousBuffer)

	// Symbol commands
	e.commandRegistry.Register("list-project-symbols", e.listProjectSymbolsCmd)
	e.commandRegistry.Register("list-document-symbols", e.listDocumentSymbolsCmd)

	// Undo/Redo commands
	e.commandRegistry.Register("undo", e.undo)
	e.commandRegistry.Register("redo", e.redo)
	
	// Yank commands
	e.commandRegistry.Register("yank-line", e.yankCurrentLine)
	
	// Repeat command
	e.commandRegistry.Register("repeat-last-change", e.repeatLastChange)
	
	// Delete commands
	e.commandRegistry.Register("start-delete", e.startDeleteOperation)
	
	// Search commands
	e.commandRegistry.Register("find-next", e.findNextCmd)
	e.commandRegistry.Register("find-previous", e.findPreviousCmd)
	
	// Help commands
	e.commandRegistry.Register("help", e.showHelp)
	e.commandRegistry.Register("help-keys", e.showHelpKeys)
}