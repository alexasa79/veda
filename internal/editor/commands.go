package editor

// registerCommands registers all command handlers
func (e *Editor) registerCommands() {
	// Movement commands - apply to all cursors
	e.commandRegistry.Register("up", e.moveUp, true)
	e.commandRegistry.Register("down", e.moveDown, true)
	e.commandRegistry.Register("left", e.moveLeft, true)
	e.commandRegistry.Register("right", e.moveRight, true)
	e.commandRegistry.Register("word-forward", e.wordForward, true)
	e.commandRegistry.Register("word-backward", e.wordBackward, true)
	e.commandRegistry.Register("beginning-of-line", e.beginningOfLine, true)
	e.commandRegistry.Register("end-of-line", e.endOfLine, true)
	e.commandRegistry.Register("beginning-of-line-insert", e.beginningOfLineInsert, true)
	e.commandRegistry.Register("end-of-line-insert", e.endOfLineInsert, true)
	e.commandRegistry.Register("page-up", e.pageUp, true)
	e.commandRegistry.Register("page-down", e.pageDown, true)
	e.commandRegistry.Register("go-to-beginning", e.goToBeginning, true)
	e.commandRegistry.Register("go-to-end", e.goToEnd, true)

	// Insert mode movement commands - apply to all cursors
	e.commandRegistry.Register("up-insert", e.moveUpInsert, true)
	e.commandRegistry.Register("down-insert", e.moveDownInsert, true)
	e.commandRegistry.Register("left-insert", e.moveLeftInsert, true)
	e.commandRegistry.Register("right-insert", e.moveRightInsert, true)

	// Mode commands
	e.commandRegistry.Register("enter-command-mode", e.enterCommandModeCmd, false)
	e.commandRegistry.Register("enter-search-mode", e.enterSearchModeCmd, false)
	e.commandRegistry.Register("enter-insert-mode", e.enterInsertMode, false)
	e.commandRegistry.Register("append-after-cursor", e.appendAfterCursor, true) // Apply to all cursors for positioning
	e.commandRegistry.Register("append-at-end-of-line", e.appendAtEndOfLine, true) // Apply to all cursors for positioning
	e.commandRegistry.Register("exit-insert-mode", e.exitInsertMode, true) // Apply to all cursors for cursor adjustment
	e.commandRegistry.Register("enter-visual-mode", e.enterVisualMode, false)
	e.commandRegistry.Register("enter-visual-line-mode", e.enterVisualLineMode, false)
	e.commandRegistry.Register("exit-visual-mode", e.exitVisualMode, false)
	e.commandRegistry.Register("open-line-below", e.openLineBelow, false)
	e.commandRegistry.Register("open-line-above", e.openLineAbove, false)
	e.commandRegistry.Register("join-lines", e.joinLines, false)
	e.commandRegistry.Register("rewrap-paragraph", e.rewrapParagraphCmd, false)
	e.commandRegistry.Register("enter-cursor-mode", e.enterCursorModeCmd, false)
	e.commandRegistry.Register("add-virtual-cursor", e.addVirtualCursorCmd, false)

	// Insert mode commands - text modification applies to all cursors
	e.commandRegistry.Register("insert-newline", e.insertNewline, true)
	e.commandRegistry.Register("backspace", e.backspace, true)
	e.commandRegistry.Register("delete-character-insert", e.deleteCharacterInsert, true)
	e.commandRegistry.Register("delete-word-forward", e.deleteWordForward, true)
	e.commandRegistry.Register("delete-word-backward", e.deleteWordBackward, true)
	e.commandRegistry.Register("word-forward-insert", e.wordForwardInsert, true) // Movement in insert mode
	e.commandRegistry.Register("word-backward-insert", e.wordBackwardInsert, true) // Movement in insert mode
	e.commandRegistry.Register("insert-tab", e.insertTab, true)

	// Delete commands - apply to all cursors
	e.commandRegistry.Register("delete-current-character", e.deleteCurrentCharacter, true)
	e.commandRegistry.Register("delete-to-end-of-line", e.deleteToEndOfLine, true)

	// Visual mode operations
	e.commandRegistry.Register("delete-selection", e.deleteSelection, false)
	e.commandRegistry.Register("yank-selection", e.yankSelection, false)
	e.commandRegistry.Register("change-selection", e.changeSelection, false)

	// File commands
	e.commandRegistry.Register("open-file", e.openFileCmd, false)
	e.commandRegistry.Register("q", e.closeBuffer, false)  // Close current buffer
	e.commandRegistry.Register("qa", e.actuallyQuit, false)  // Force quit (shortcut for :Q)
	e.commandRegistry.Register("Quit", e.actuallyQuit, false)  // Actually quit the editor
	e.commandRegistry.Register("Q", e.actuallyQuit, false)  // Shortcut for Quit
	e.commandRegistry.Register("write", e.writeFile, false)
	e.commandRegistry.Register("w", e.writeFile, false)  // Shortcut for write
	e.commandRegistry.Register("wq", e.writeAndQuit, false)  // Write and quit
	e.commandRegistry.Register("edit", e.editFile, false)
	e.commandRegistry.Register("e", e.editFile, false)  // Shortcut for edit
	e.commandRegistry.Register("cd", e.changeDirectory, false)

	// LSP commands
	e.commandRegistry.Register("go-to-definition", e.goToDefinition, false)
	e.commandRegistry.Register("find-references", e.findReferences, false)
	e.commandRegistry.Register("show-lsp-buffer-go", e.showLSPBufferGo, false)
	e.commandRegistry.Register("show-signature", e.showSignature, false)
	e.commandRegistry.Register("jump-to-reference", e.jumpToReference, false)

	// Navigation commands
	e.commandRegistry.Register("go-back", e.goBack, false)
	e.commandRegistry.Register("next-reference", e.nextReference, false)
	e.commandRegistry.Register("previous-reference", e.previousReference, false)

	// Buffer commands
	e.commandRegistry.Register("show-buffers", e.showBuffers, false)
	e.commandRegistry.Register("switch-to-previous-buffer", e.showBufferListWithPrevious, false)
	e.commandRegistry.Register("show-buffer-list-previous", e.showBufferListWithPrevious, false) // More descriptive alias

	// Symbol commands
	e.commandRegistry.Register("list-project-symbols", e.listProjectSymbolsCmd, false)
	e.commandRegistry.Register("list-document-symbols", e.listDocumentSymbolsCmd, false)

	// Undo/Redo commands
	e.commandRegistry.Register("undo", e.undo, false)
	e.commandRegistry.Register("redo", e.redo, false)

	// Yank commands
	e.commandRegistry.Register("yank-line", e.yankCurrentLine, false)
	e.commandRegistry.Register("start-yank", e.startYankOperation, false)

	// Paste commands
	e.commandRegistry.Register("paste-after", e.pasteAfter, false)
	e.commandRegistry.Register("paste-before", e.pasteBefore, false)

	// Repeat command
	e.commandRegistry.Register("repeat-last-change", e.repeatLastChange, false)

	// Delete commands
	e.commandRegistry.Register("start-delete", e.startDeleteOperation, false)

	// Change commands
	e.commandRegistry.Register("start-change", e.startChangeOperation, false)

	// Search commands
	e.commandRegistry.Register("find-next", e.findNextCmd, true) // Apply to all cursors for navigation
	e.commandRegistry.Register("find-previous", e.findPreviousCmd, true) // Apply to all cursors for navigation
	e.commandRegistry.Register("clear-search", e.clearSearchCmd, false)
	e.commandRegistry.Register("search-word-under-cursor", e.searchWordUnderCursorCmd, false)

	// Help commands
	e.commandRegistry.Register("help", e.showHelp, false)
	e.commandRegistry.Register("help-keys", e.showHelpKeys, false)
	
	// Diagnostics commands
	e.commandRegistry.Register("diagnostics", e.showDiagnostics, false)
	
	// Project changes commands
	e.commandRegistry.Register("show-project-changes", e.showProjectChanges, false)

	// Completion commands
	e.commandRegistry.Register("show-completions", func([]string) error { return e.showCompletions() }, false)
	
	// Error testing command
	e.commandRegistry.Register("error", e.errorCommand, false)
}