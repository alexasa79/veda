package editor

import (
	"fmt"
	"strings"
	"time"
	"veda/internal/buffer"
	"veda/internal/clipboard"
	"veda/internal/commands"
	"veda/internal/events"
)

// registerCommands registers all command handlers
func (e *Editor) registerCommands() {
	// Movement commands - apply to all cursors
	e.commandRegistry.Register("up", e.moveUp, true, true)
	e.commandRegistry.Register("down", e.moveDown, true, true)
	e.commandRegistry.Register("left", e.moveLeft, true, true)
	e.commandRegistry.Register("right", e.moveRight, true, true)
	e.commandRegistry.Register("word-forward", e.wordForward, true, true)
	e.commandRegistry.Register("word-backward", e.wordBackward, true, true)
	e.commandRegistry.Register("word-forward-no-underscore", e.wordForwardNoUnderscore, true, true)   // W command
	e.commandRegistry.Register("word-backward-no-underscore", e.wordBackwardNoUnderscore, true, true) // B command
	e.commandRegistry.Register("find-character", e.findCharacter, true, true)
	e.commandRegistry.Register("till-character", e.tillCharacter, true, true)
	e.commandRegistry.Register("start-find-character", e.startFindCharacterOperation, true, true)
	e.commandRegistry.Register("start-till-character", e.startTillCharacterOperation, true, true)
	e.commandRegistry.Register("beginning-of-line", e.beginningOfLine, true, true)
	e.commandRegistry.Register("end-of-line", e.endOfLine, true, true)
	e.commandRegistry.Register("beginning-of-line-insert", e.beginningOfLineInsert, true, true)
	e.commandRegistry.Register("end-of-line-insert", e.endOfLineInsert, true, true)
	e.commandRegistry.Register("page-up", e.pageUp, true, true)
	e.commandRegistry.Register("page-down", e.pageDown, true, true)
	e.commandRegistry.Register("go-to-beginning", e.goToBeginning, true, true)
	e.commandRegistry.Register("go-to-end", e.goToEnd, true, true)

	// Insert mode movement commands - apply to all cursors
	e.commandRegistry.Register("up-insert", e.moveUpInsert, true, true)
	e.commandRegistry.Register("down-insert", e.moveDownInsert, true, true)
	e.commandRegistry.Register("left-insert", e.moveLeftInsert, true, true)
	e.commandRegistry.Register("right-insert", e.moveRightInsert, true, true)

	// Mode commands
	e.commandRegistry.Register("enter-command-mode", e.enterCommandModeCmd, false, false)
	e.commandRegistry.Register("enter-search-mode", e.enterSearchModeCmd, false, true)
	e.commandRegistry.Register("enter-search-mode-backward", e.enterSearchModeBackwardCmd, false, true)
	e.commandRegistry.Register("enter-insert-mode", e.enterInsertMode, false, true)
	e.commandRegistry.Register("append-after-cursor", e.appendAfterCursor, true, true)   // Apply to all cursors for positioning
	e.commandRegistry.Register("append-at-end-of-line", e.appendAtEndOfLine, true, true) // Apply to all cursors for positioning
	e.commandRegistry.Register("exit-insert-mode", e.exitInsertMode, true, true)         // Apply to all cursors for cursor adjustment
	e.commandRegistry.Register("enter-visual-mode", e.enterVisualMode, false, true)
	e.commandRegistry.Register("enter-visual-line-mode", e.enterVisualLineMode, false, true)
	e.commandRegistry.Register("exit-visual-mode", e.exitVisualMode, false, true)
	e.commandRegistry.Register("open-line-below", e.openLineBelow, false, true)
	e.commandRegistry.Register("open-line-above", e.openLineAbove, false, true)
	e.commandRegistry.Register("join-lines", e.joinLines, false, true)
	e.commandRegistry.Register("rewrap-paragraph", e.rewrapParagraphCmd, false, true)
	e.commandRegistry.Register("enter-cursor-mode", e.enterCursorModeCmd, false, true)
	e.commandRegistry.Register("add-virtual-cursor", e.addVirtualCursorCmd, false, true)

	// Insert mode commands - text modification applies to all cursors
	e.commandRegistry.Register("insert-newline", e.insertNewline, true, true)
	e.commandRegistry.Register("backspace", e.backspace, true, true)
	e.commandRegistry.Register("delete-character-insert", e.deleteCharacterInsert, true, true)
	e.commandRegistry.Register("delete-word-forward", e.deleteWordForward, true, true)
	e.commandRegistry.Register("delete-word-backward", e.deleteWordBackward, true, true)
	e.commandRegistry.Register("word-forward-insert", e.wordForwardInsert, true, true)   // Movement in insert mode
	e.commandRegistry.Register("word-backward-insert", e.wordBackwardInsert, true, true) // Movement in insert mode
	e.commandRegistry.Register("insert-tab", e.insertTab, true, true)

	// Delete commands - apply to all cursors
	e.commandRegistry.Register("delete-current-character", e.deleteCurrentCharacter, true, true)
	e.commandRegistry.Register("delete-to-end-of-line", e.deleteToEndOfLine, true, true)
	// Change to end of line (C)
	e.commandRegistry.Register("change-to-end-of-line", e.changeToEndOfLine, true, true)

	// Comment commands
	e.commandRegistry.Register("toggle-comment", e.toggleComment, true, true)

	// Line movement commands
	e.commandRegistry.Register("move-lines-up", e.moveLinesUp, false, true)
	e.commandRegistry.Register("move-lines-down", e.moveLinesDown, false, true)

	// Visual mode operations
	e.commandRegistry.Register("delete-selection", e.deleteSelection, false, true)
	e.commandRegistry.Register("yank-selection", e.yankSelection, false, true)
	e.commandRegistry.Register("change-selection", e.changeSelection, false, true)
	// Visual mode indentation
	e.commandRegistry.Register("indent-selection", e.indentSelection, false, true)
	e.commandRegistry.Register("unindent-selection", e.unindentSelection, false, true)

	// Shift+Arrow helpers: start visual selection and move
	e.commandRegistry.Register("start-visual-left", e.startVisualLeft, false, true)
	e.commandRegistry.Register("start-visual-right", e.startVisualRight, false, true)
	e.commandRegistry.Register("start-visual-up", e.startVisualUp, false, true)
	e.commandRegistry.Register("start-visual-down", e.startVisualDown, false, true)

	// File commands
	e.commandRegistry.Register("open-file", e.openFileCmd, false, false)
	e.commandRegistry.Register("reload-file", e.reloadFile, false, false)
	e.commandRegistry.Register("reload", e.reloadEditor, false, false)
	e.commandRegistry.Register("q", e.closeBuffer, false, true)      // Close current buffer
	e.commandRegistry.Register("q!", e.actuallyQuit, false, false)   // Force quit
	e.commandRegistry.Register("qa", e.actuallyQuit, false, false)   // Force quit (shortcut for :Q)
	e.commandRegistry.Register("Quit", e.actuallyQuit, false, false) // Actually quit the editor
	e.commandRegistry.Register("Q", e.actuallyQuit, false, false)    // Shortcut for Quit
	e.commandRegistry.Register("write", e.writeFile, false, true)
	e.commandRegistry.Register("w", e.writeFile, false, true)     // Shortcut for write
	e.commandRegistry.Register("wq", e.writeAndQuit, false, true) // Write and quit
	e.commandRegistry.Register("edit", e.editFile, false, false)
	e.commandRegistry.Register("e", e.editFile, false, false) // Shortcut for edit
	e.commandRegistry.Register("cd", e.changeDirectory, false, false)
	e.commandRegistry.Register("theme", e.themeCommand, false, false)

	// LSP commands
	e.commandRegistry.Register("go-to-definition", e.goToDefinition, false, true)
	e.commandRegistry.Register("go-to-definition-new-window", e.goToDefinitionNewWindow, false, true)

	// Git commands
	e.commandRegistry.Register("toggle-change-preview", e.toggleChangePreview, false, false)
	e.commandRegistry.Register("git-blame-line", e.gitBlameCurrentLine, false, false)
	e.commandRegistry.Register("git-show-commit", e.gitShowCommit, false, false)
	e.commandRegistry.Register("git-show-commit-new-window", e.gitShowCommitNewWindow, false, false)
	e.commandRegistry.Register("find-references", e.findReferences, false, true)
	e.commandRegistry.Register("show-lsp-buffer-go", e.showLSPBufferGo, false, true)
	e.commandRegistry.Register("show-signature", e.showSignature, false, true)
	e.commandRegistry.Register("jump-to-reference", e.jumpToReference, false, true)

	// Navigation commands
	e.commandRegistry.Register("go-back", e.goBack, false, true)
	e.commandRegistry.Register("next-item", e.nextContextualItem, false, true)
	e.commandRegistry.Register("previous-item", e.previousContextualItem, false, true)

	// Git staging operations
	e.commandRegistry.Register("toggle-stage-hunk", e.toggleStageHunk, false, true)
	e.commandRegistry.Register("toggle-stage-line", e.toggleStageLine, false, true)

	// Git drop (revert to HEAD) operations
	e.commandRegistry.Register("drop-change-hunk", e.dropChangeHunk, false, true)
	e.commandRegistry.Register("drop-change-line", e.dropChangeLine, false, true)

	// Buffer commands
	e.commandRegistry.Register("switch-to-previous-buffer", e.showBufferListWithPrevious, false, false)
	e.commandRegistry.Register("show-buffer-list-previous", e.showBufferListWithPrevious, false, false) // More descriptive alias
	e.commandRegistry.Register("new-buffer", e.newBuffer, false, false)                                 // Create a new empty buffer
	e.commandRegistry.Register("next-buffer", e.nextBufferInWindow, false, true)                        // Switch to next buffer in current window
	e.commandRegistry.Register("previous-buffer", e.previousBufferInWindow, false, true)                // Switch to previous buffer in current window

	// Symbol commands
	e.commandRegistry.Register("list-project-symbols", e.listProjectSymbolsCmd, false, false)
	e.commandRegistry.Register("list-document-symbols", e.listDocumentSymbolsCmd, false, true)

	// Undo/Redo commands
	e.commandRegistry.Register("undo", e.undo, false, true)
	e.commandRegistry.Register("redo", e.redo, false, true)

	// Yank commands
	e.commandRegistry.Register("yank-line", e.yankCurrentLine, false, true)
	e.commandRegistry.Register("start-yank", e.startYankOperation, false, true)
	// Yank to end of line (y$) - ensure character-wise yank without newline
	e.commandRegistry.Register("yank-to-end-of-line", e.yankToEndOfLine, false, true)

	// Paste commands
	e.commandRegistry.Register("paste-after", e.pasteAfter, false, true)
	e.commandRegistry.Register("paste-before", e.pasteBefore, false, true)

	// Repeat command
	e.commandRegistry.Register("repeat-last-change", e.repeatLastChange, false, true)

	// Delete commands
	e.commandRegistry.Register("start-delete", e.startDeleteOperation, false, true)

	// Change commands
	e.commandRegistry.Register("start-change", e.startChangeOperation, false, true)

	// Indentation commands
	e.commandRegistry.Register("indent-line", e.indentLine, true, true)                 // Apply to all cursors
	e.commandRegistry.Register("unindent-line", e.unindentLine, true, true)             // Apply to all cursors
	e.commandRegistry.Register("reindent-expression", e.reindentExpression, true, true) // Apply to all cursors

	// Multi-cursor alignment
	e.commandRegistry.Register("align-cursors", e.alignCursors, false, true) // Aligns all cursors to rightmost position

	// Search commands
	e.commandRegistry.Register("find-next", e.findNextCmd, true, true)         // Apply to all cursors for navigation
	e.commandRegistry.Register("find-previous", e.findPreviousCmd, true, true) // Apply to all cursors for navigation
	e.commandRegistry.Register("clear-search", e.clearSearchCmd, false, false)
	e.commandRegistry.Register("clear-state", e.clearStateCmd, false, false)
	e.commandRegistry.Register("search-word-under-cursor", e.searchWordUnderCursorCmd, false, true)

	// Help commands
	e.commandRegistry.Register("help", e.showHelp, false, false)
	e.commandRegistry.Register("help-keys", e.showHelpKeys, false, false)

	// Markdown commands
	e.commandRegistry.Register("open-markdown-link-new-window", e.openMarkdownLinkInNewWindow, false, true)

	// Diagnostics commands
	e.commandRegistry.Register("diagnostics", e.showDiagnostics, false, false)

	// Project changes commands
	e.commandRegistry.Register("show-project-changes", e.showProjectChanges, false, false)
	e.commandRegistry.Register("review-prs", e.reviewPRs, false, false)
	e.commandRegistry.Register("review-pr", e.reviewPR, false, false)

	// Diff commands
	e.commandRegistry.Register("diff-current-buffer", e.diffCurrentBuffer, false, true)

	// Completion commands
	e.commandRegistry.Register("show-completions", func([]string) error { return e.showCompletions() }, false, true)

	// Terminal commands
	e.commandRegistry.Register("term", e.openTerminalCmd, false, false)
	e.commandRegistry.Register("term-new-window", e.openTerminalInNewWindowCmd, false, false)

	// Error testing command
	e.commandRegistry.Register("error", e.errorCommand, false, false)

	// Messages command: append a line to *Messages*
	e.commandRegistry.Register("message", e.messageCommand, false, false)

	// Calculator commands
	e.commandRegistry.Register("calculate-append", e.calculateAppend, false, true)
	e.commandRegistry.Register("calculate-replace", e.calculateReplace, false, true)

	// Spell check command
	e.commandRegistry.Register("spellcheck", e.spellCheck, false, true)

	// Temporary normal mode
	e.commandRegistry.Register("enter-temporary-normal-mode", e.enterTemporaryNormalMode, false, false)

	// Window commands
	e.commandRegistry.Register("vsplit", e.vsplit, false, false)
	// Numeric window switches (1-based)
	e.commandRegistry.Register("window-1", func([]string) error { return e.switchToWindow(1) }, false, false)
	e.commandRegistry.Register("window-2", func([]string) error { return e.switchToWindow(2) }, false, false)
	e.commandRegistry.Register("window-3", func([]string) error { return e.switchToWindow(3) }, false, false)
	e.commandRegistry.Register("window-4", func([]string) error { return e.switchToWindow(4) }, false, false)
	e.commandRegistry.Register("window-5", func([]string) error { return e.switchToWindow(5) }, false, false)
	e.commandRegistry.Register("window-6", func([]string) error { return e.switchToWindow(6) }, false, false)
	e.commandRegistry.Register("window-7", func([]string) error { return e.switchToWindow(7) }, false, false)
	e.commandRegistry.Register("window-8", func([]string) error { return e.switchToWindow(8) }, false, false)
	e.commandRegistry.Register("window-9", func([]string) error { return e.switchToWindow(9) }, false, false)
	e.commandRegistry.Register("window-last", e.switchToLastWindow, false, false)
	e.commandRegistry.Register("window-next", e.switchToNextWindow, false, false)

	// Window buffer move commands
	e.commandRegistry.Register("move-buffer-left", e.moveBufferLeft, false, true)
	e.commandRegistry.Register("move-buffer-right", e.moveBufferRight, false, true)
	e.commandRegistry.Register("move-buffer-center", e.moveBufferCenter, false, true)

	// Window management commands
	e.commandRegistry.Register("close-window", e.closeCurrentWindow, false, false)
	e.commandRegistry.Register("toggle-window-fullscreen", e.toggleWindowFullscreen, false, false)
	e.commandRegistry.Register("toggle-zen-mode", e.toggleZenMode, false, false)

	// Info commands
	e.commandRegistry.Register("show-file-info", e.showFileInfoCommand, false, true)

	// Registers commands
	e.commandRegistry.Register("regcopy", e.regcopyCmd, false, false)
	e.commandRegistry.Register("reg", e.showRegistersCmd, false, false)

	// Debugging commands
	e.commandRegistry.Register("debug-dump", e.debugDumpCommand, false, true)

	// Case transform commands (apply to all cursors)
	e.commandRegistry.Register("to-upper", e.toUpperCmd, true, true)
	e.commandRegistry.Register("to-lower", e.toLowerCmd, true, true)

	// Cursor mode utilities
	e.commandRegistry.Register("remove-last-virtual-cursor", e.removeLastVirtualCursorCmd, false, true)

	// Screen commands
	e.commandRegistry.Register("redraw-screen", e.redrawScreen, false, false)
}

// openTerminalCmd opens a terminal buffer and switches to terminal mode.
// For now this is a minimal scaffold that creates a read-only internal buffer
// and switches mode to TERMINAL. Subsequent edits will wire PTY I/O.
func (e *Editor) openTerminalCmd(args []string) error {
	// Create an internal buffer for the terminal session
	bufName := "*Terminal*"
	// Ensure unique name if multiple sessions
	idx := 1
	name := bufName
	for {
		if _, exists := e.buffers[name]; !exists {
			break
		}
		idx++
		name = bufName + " " + fmt.Sprint(idx)
	}
	newBuf := buffer.NewBuffer("", name, "", e.config.DefaultIndentType, e.config.DefaultIndentWidth)
	newBuf.IsInternal = true
	newBuf.IsReadOnly = true
	// Register and switch
	e.buffers[name] = newBuf
	e.bufferOrder = append(e.bufferOrder, name)
	e.switchToBuffer(newBuf)
	// Start a PTY on the proxy host using the active window size
	cols, rows := e.getActiveWindowSizeForTerminal()
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	sessionID, pid, err := e.proxyManager.StartTerminal(e.workingDirectory, nil, nil, cols, rows)
	if err != nil {
		// Gracefully degrade in test or constrained environments: show error line, keep buffer
		newBuf.AppendLine("[terminal unavailable] " + err.Error())
		// Refresh iterators after programmatic modification
		if newBuf.ViewIterator != nil {
			newBuf.ViewIterator = buffer.NewBufferIteratorAtLine(newBuf, newBuf.ViewIterator.LineNum)
		} else {
			newBuf.ViewIterator = buffer.NewBufferIterator(newBuf)
		}
		if newBuf.CursorIterator != nil {
			newBuf.CursorIterator = buffer.NewBufferIteratorAtLine(newBuf, newBuf.CursorIterator.LineNum)
		} else {
			newBuf.CursorIterator = buffer.NewBufferIterator(newBuf)
		}
	} else {
		// Track session by buffer path
		e.terminalSessions[newBuf.FilePath] = &TerminalSessionState{SessionID: sessionID, WorkingDir: e.workingDirectory, Cols: cols, Rows: rows, term: NewTerminalEmulator(cols, rows)}
		// Store banner as metadata rather than writing to PTY or emulator grid
		if ts := e.terminalSessions[newBuf.FilePath]; ts != nil {
			ts.Banner = fmt.Sprintf("[terminal started] pid=%d (%dx%d)", pid, cols, rows)
		}
		// Kick an initial render so the user sees the buffer immediately (banner will be prepended in handler)
		e.asyncEvents <- events.NewCustomEvent("terminal-output", map[string]interface{}{
			"buffer": newBuf.FilePath,
			"data":   e.terminalSessions[newBuf.FilePath].term.Render(),
		})
		// Begin polling for output
		e.startTerminalPolling(newBuf.FilePath)
		// Avoid sending ANSI DSR queries which can echo visibly in some shells
	}
	// Enter terminal mode (keys will be forwarded in editor key loop)
	e.mode = commands.TerminalMode
	return nil
}

// openTerminalInNewWindowCmd opens a terminal in a new right-hand window and focuses it.
func (e *Editor) openTerminalInNewWindowCmd(args []string) error {
	// Always create a new window to the far right and focus it
	e.createRightmostWindow(nil)
	// Now open terminal in the new active window
	return e.openTerminalCmd(nil)
}

// startTerminalPolling starts a goroutine to poll terminal output via proxy and append to buffer
func (e *Editor) startTerminalPolling(bufferPath string) {
	ts := e.terminalSessions[bufferPath]
	if ts == nil {
		return
	}
	e.terminalPollWG.Add(1)
	go func() {
		defer e.terminalPollWG.Done()
		for {
			select {
			case <-e.terminalPollStop:
				return
			default:
			}
			data, closed, exitCode, err := e.proxyManager.ReadTerminal(ts.WorkingDir, ts.SessionID, 65536)
			if err != nil {
				return
			}
			if len(data) > 0 {
				// Feed emulator, then render the grid as text
				if ts.term != nil {
					response := ts.term.WriteWithResponse(data)
					if len(response) > 0 {
						_ = e.proxyManager.WriteTerminal(ts.WorkingDir, ts.SessionID, response)
					}
					rendered := ts.term.Render()
					e.postTerminalEvent(events.NewCustomEvent("terminal-output", map[string]interface{}{
						"buffer": bufferPath,
						"data":   rendered,
					}))
				} else {
					e.postTerminalEvent(events.NewCustomEvent("terminal-output", map[string]interface{}{
						"buffer": bufferPath,
						"data":   string(data),
					}))
				}
			}
			if closed {
				ts.Closed = true
				ts.ExitCode = exitCode
				// Notify user with an exit banner appended to the current snapshot
				var exitMsg string
				if exitCode != nil {
					exitMsg = fmt.Sprintf("[terminal exited] status=%d", *exitCode)
				} else {
					exitMsg = "[terminal exited]"
				}
				var rendered string
				if ts.term != nil {
					rendered = ts.term.Render()
				}
				// Place the exit banner just below the last non-empty line,
				// not after the full terminal height worth of blank lines.
				if rendered != "" {
					lines := strings.Split(rendered, "\n")
					// Trim trailing empty lines
					i := len(lines) - 1
					for i >= 0 && lines[i] == "" {
						i--
					}
					trimmed := strings.Join(lines[:i+1], "\n")
					if trimmed == "" {
						rendered = exitMsg
					} else {
						rendered = trimmed + "\n" + exitMsg
					}
				} else {
					rendered = exitMsg
				}
				e.postTerminalEvent(events.NewCustomEvent("terminal-output", map[string]interface{}{
					"buffer": bufferPath,
					"data":   rendered,
				}))
				// Also switch the editor out of TerminalMode so user returns to Normal
				e.postTerminalEvent(events.NewCustomEvent("terminal-closed", map[string]interface{}{
					"buffer": bufferPath,
				}))
				return
			}
			// Poll interval
			time.Sleep(25 * time.Millisecond)
		}
	}()
}

// changeToEndOfLine changes from the cursor to the end of the current line
// and enters insert mode (equivalent to Vim's 'C'). It uses the ChangeOperator
// with an EndOfLineTextObject and supports multiple cursors.
func (e *Editor) changeToEndOfLine(args []string) error {
	op := ChangeOperator{}
	textObj := EndOfLineTextObject{}
	return e.executeOperatorWithTextObjectForAllCursors(op, textObj)
}

// yankToEndOfLine yanks from cursor to end of line (character-wise, no trailing newline)
func (e *Editor) yankToEndOfLine(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}
	// Compute the range once and reuse the YankOperator's extraction logic
	lineNum := e.CurrentBuffer().CursorIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX
	iter := e.CurrentBuffer().CursorIterator
	if iter.IsEOF() {
		return nil
	}
	line := iter.Line()
	if cursorX > len(line) {
		cursorX = len(line)
	}
	startLine, startChar, endLine, endChar := lineNum, cursorX, lineNum, len(line)

	// Extract text explicitly to avoid newline semantics of full-line yank
	var textToYank string
	if startLine == endLine {
		if startChar < len(line) && endChar <= len(line) {
			textToYank = line[startChar:endChar]
		}
	} else {
		textToYank = e.getTextInRange(startLine, startChar, endLine, endChar)
		// Do not add newline here; this is character-wise
	}
	if textToYank == "" {
		return nil
	}

	// Copy to clipboard
	if err := clipboard.Copy(textToYank); err != nil {
		LL.Error("Failed to copy to clipboard: %v", err)
		return err
	}

	// Flash highlight to indicate yanked region
	e.setFlashHighlight(startLine, startChar, endLine, endChar)
	return nil
}
