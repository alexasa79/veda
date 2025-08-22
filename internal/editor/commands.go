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
	e.commandRegistry.Register("up", e.moveUp, true)
	e.commandRegistry.Register("down", e.moveDown, true)
	e.commandRegistry.Register("left", e.moveLeft, true)
	e.commandRegistry.Register("right", e.moveRight, true)
	e.commandRegistry.Register("word-forward", e.wordForward, true)
	e.commandRegistry.Register("word-backward", e.wordBackward, true)
	e.commandRegistry.Register("word-forward-no-underscore", e.wordForwardNoUnderscore, true)   // W command
	e.commandRegistry.Register("word-backward-no-underscore", e.wordBackwardNoUnderscore, true) // B command
	e.commandRegistry.Register("find-character", e.findCharacter, true)
	e.commandRegistry.Register("till-character", e.tillCharacter, true)
	e.commandRegistry.Register("start-find-character", e.startFindCharacterOperation, true)
	e.commandRegistry.Register("start-till-character", e.startTillCharacterOperation, true)
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
	e.commandRegistry.Register("append-after-cursor", e.appendAfterCursor, true)   // Apply to all cursors for positioning
	e.commandRegistry.Register("append-at-end-of-line", e.appendAtEndOfLine, true) // Apply to all cursors for positioning
	e.commandRegistry.Register("exit-insert-mode", e.exitInsertMode, true)         // Apply to all cursors for cursor adjustment
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
	e.commandRegistry.Register("word-forward-insert", e.wordForwardInsert, true)   // Movement in insert mode
	e.commandRegistry.Register("word-backward-insert", e.wordBackwardInsert, true) // Movement in insert mode
	e.commandRegistry.Register("insert-tab", e.insertTab, true)

	// Delete commands - apply to all cursors
	e.commandRegistry.Register("delete-current-character", e.deleteCurrentCharacter, true)
	e.commandRegistry.Register("delete-to-end-of-line", e.deleteToEndOfLine, true)
	// Change to end of line (C)
	e.commandRegistry.Register("change-to-end-of-line", e.changeToEndOfLine, true)

	// Visual mode operations
	e.commandRegistry.Register("delete-selection", e.deleteSelection, false)
	e.commandRegistry.Register("yank-selection", e.yankSelection, false)
	e.commandRegistry.Register("change-selection", e.changeSelection, false)
	// Visual mode indentation
	e.commandRegistry.Register("indent-selection", e.indentSelection, false)
	e.commandRegistry.Register("unindent-selection", e.unindentSelection, false)

	// Shift+Arrow helpers: start visual selection and move
	e.commandRegistry.Register("start-visual-left", e.startVisualLeft, false)
	e.commandRegistry.Register("start-visual-right", e.startVisualRight, false)
	e.commandRegistry.Register("start-visual-up", e.startVisualUp, false)
	e.commandRegistry.Register("start-visual-down", e.startVisualDown, false)

	// File commands
	e.commandRegistry.Register("open-file", e.openFileCmd, false)
	e.commandRegistry.Register("q", e.closeBuffer, false)     // Close current buffer
	e.commandRegistry.Register("q!", e.actuallyQuit, false)   // Force quit
	e.commandRegistry.Register("qa", e.actuallyQuit, false)   // Force quit (shortcut for :Q)
	e.commandRegistry.Register("Quit", e.actuallyQuit, false) // Actually quit the editor
	e.commandRegistry.Register("Q", e.actuallyQuit, false)    // Shortcut for Quit
	e.commandRegistry.Register("write", e.writeFile, false)
	e.commandRegistry.Register("w", e.writeFile, false)     // Shortcut for write
	e.commandRegistry.Register("wq", e.writeAndQuit, false) // Write and quit
	e.commandRegistry.Register("edit", e.editFile, false)
	e.commandRegistry.Register("e", e.editFile, false) // Shortcut for edit
	e.commandRegistry.Register("cd", e.changeDirectory, false)

	// LSP commands
	e.commandRegistry.Register("go-to-definition", e.goToDefinition, false)
	e.commandRegistry.Register("go-to-definition-new-window", e.goToDefinitionNewWindow, false)
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
	e.commandRegistry.Register("new-buffer", e.newBuffer, false)                                 // Create a new empty buffer
	e.commandRegistry.Register("next-buffer", e.nextBufferInWindow, false)                       // Switch to next buffer in current window
	e.commandRegistry.Register("previous-buffer", e.previousBufferInWindow, false)               // Switch to previous buffer in current window

	// Symbol commands
	e.commandRegistry.Register("list-project-symbols", e.listProjectSymbolsCmd, false)
	e.commandRegistry.Register("list-document-symbols", e.listDocumentSymbolsCmd, false)

	// Undo/Redo commands
	e.commandRegistry.Register("undo", e.undo, false)
	e.commandRegistry.Register("redo", e.redo, false)

	// Yank commands
	e.commandRegistry.Register("yank-line", e.yankCurrentLine, false)
	e.commandRegistry.Register("start-yank", e.startYankOperation, false)
	// Yank to end of line (y$) - ensure character-wise yank without newline
	e.commandRegistry.Register("yank-to-end-of-line", e.yankToEndOfLine, false)

	// Paste commands
	e.commandRegistry.Register("paste-after", e.pasteAfter, false)
	e.commandRegistry.Register("paste-before", e.pasteBefore, false)

	// Repeat command
	e.commandRegistry.Register("repeat-last-change", e.repeatLastChange, false)

	// Delete commands
	e.commandRegistry.Register("start-delete", e.startDeleteOperation, false)

	// Change commands
	e.commandRegistry.Register("start-change", e.startChangeOperation, false)

	// Indentation commands
	e.commandRegistry.Register("indent-line", e.indentLine, true)                 // Apply to all cursors
	e.commandRegistry.Register("unindent-line", e.unindentLine, true)             // Apply to all cursors
	e.commandRegistry.Register("reindent-line", e.reindentLine, true)             // Apply to all cursors
	e.commandRegistry.Register("reindent-expression", e.reindentExpression, true) // Apply to all cursors

	// Search commands
	e.commandRegistry.Register("find-next", e.findNextCmd, true)         // Apply to all cursors for navigation
	e.commandRegistry.Register("find-previous", e.findPreviousCmd, true) // Apply to all cursors for navigation
	e.commandRegistry.Register("clear-search", e.clearSearchCmd, false)
	e.commandRegistry.Register("search-word-under-cursor", e.searchWordUnderCursorCmd, false)

	// Help commands
	e.commandRegistry.Register("help", e.showHelp, false)
	e.commandRegistry.Register("help-keys", e.showHelpKeys, false)

	// Markdown commands
	e.commandRegistry.Register("open-markdown-link-new-window", e.openMarkdownLinkInNewWindow, false)

	// Diagnostics commands
	e.commandRegistry.Register("diagnostics", e.showDiagnostics, false)

	// Project changes commands
	e.commandRegistry.Register("show-project-changes", e.showProjectChanges, false)

	// Completion commands
	e.commandRegistry.Register("show-completions", func([]string) error { return e.showCompletions() }, false)

	// Terminal commands
	e.commandRegistry.Register("term", e.openTerminalCmd, false)
	e.commandRegistry.Register("term-new-window", e.openTerminalInNewWindowCmd, false)

	// Error testing command
	e.commandRegistry.Register("error", e.errorCommand, false)

	// Messages command: append a line to *Messages*
	e.commandRegistry.Register("message", e.messageCommand, false)

	// Calculator commands
	e.commandRegistry.Register("calculate-append", e.calculateAppend, false)
	e.commandRegistry.Register("calculate-replace", e.calculateReplace, false)

	// Temporary normal mode
	e.commandRegistry.Register("enter-temporary-normal-mode", e.enterTemporaryNormalMode, false)

	// Window commands
	e.commandRegistry.Register("vsplit", e.vsplit, false)
	// Numeric window switches (1-based)
	e.commandRegistry.Register("window-1", func([]string) error { return e.switchToWindow(1) }, false)
	e.commandRegistry.Register("window-2", func([]string) error { return e.switchToWindow(2) }, false)
	e.commandRegistry.Register("window-3", func([]string) error { return e.switchToWindow(3) }, false)
	e.commandRegistry.Register("window-4", func([]string) error { return e.switchToWindow(4) }, false)
	e.commandRegistry.Register("window-5", func([]string) error { return e.switchToWindow(5) }, false)
	e.commandRegistry.Register("window-6", func([]string) error { return e.switchToWindow(6) }, false)
	e.commandRegistry.Register("window-7", func([]string) error { return e.switchToWindow(7) }, false)
	e.commandRegistry.Register("window-8", func([]string) error { return e.switchToWindow(8) }, false)
	e.commandRegistry.Register("window-9", func([]string) error { return e.switchToWindow(9) }, false)
	e.commandRegistry.Register("window-last", e.switchToLastWindow, false)
	e.commandRegistry.Register("window-next", e.switchToNextWindow, false)

	// Window buffer move commands
	e.commandRegistry.Register("move-buffer-left", e.moveBufferLeft, false)
	e.commandRegistry.Register("move-buffer-right", e.moveBufferRight, false)
	e.commandRegistry.Register("move-buffer-center", e.moveBufferCenter, false)

	// Window management commands
	e.commandRegistry.Register("close-window", e.closeCurrentWindow, false)

	// Info commands
	e.commandRegistry.Register("show-file-info", e.showFileInfoCommand, false)

	// Registers commands
	e.commandRegistry.Register("regcopy", e.regcopyCmd, false)
	e.commandRegistry.Register("reg", e.showRegistersCmd, false)
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
	newBuf := buffer.NewBuffer("", name, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
	newBuf.IsInternal = true
	newBuf.IsReadOnly = true
	// Register and switch
	e.buffers[name] = newBuf
	e.bufferOrder = append(e.bufferOrder, name)
	e.switchToBuffer(newBuf)
	// Start a PTY on the proxy host
	cols, rows := e.getVisibleWidth(), e.getVisibleHeight()
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
		// Force an early ReadTerminal to populate emulator promptly (no synthetic input)
		go func(bufPath, wd, sid string) {
			data, _, _, err := e.proxyManager.ReadTerminal(wd, sid, 65536)
			if err == nil && len(data) > 0 {
				if ts := e.terminalSessions[bufPath]; ts != nil && ts.term != nil {
					ts.term.Write(data)
					e.asyncEvents <- events.NewCustomEvent("terminal-output", map[string]interface{}{
						"buffer": bufPath,
						"data":   ts.term.Render(),
					})
				}
			}
		}(newBuf.FilePath, e.workingDirectory, sessionID)
		// Avoid sending ANSI DSR queries which can echo visibly in some shells
	}
	// Enter terminal mode (keys will be forwarded in editor key loop)
	e.mode = commands.TerminalMode
	return nil
}

// openTerminalInNewWindowCmd opens a terminal in a new right-hand window and focuses it.
func (e *Editor) openTerminalInNewWindowCmd(args []string) error {
	// Use existing vsplit logic to create a new right-side window
	if err := e.vsplit(nil); err != nil {
		return err
	}
	// Now open terminal in the new active window
	return e.openTerminalCmd(nil)
}

// startTerminalPolling starts a goroutine to poll terminal output via proxy and append to buffer
func (e *Editor) startTerminalPolling(bufferPath string) {
	ts := e.terminalSessions[bufferPath]
	if ts == nil {
		return
	}
	go func() {
		for {
			data, closed, exitCode, err := e.proxyManager.ReadTerminal(ts.WorkingDir, ts.SessionID, 65536)
			if err != nil {
				return
			}
			if len(data) > 0 {
				// Feed emulator, then render the grid as text
				if ts.term != nil {
					ts.term.Write(data)
					rendered := ts.term.Render()
					e.asyncEvents <- events.NewCustomEvent("terminal-output", map[string]interface{}{
						"buffer": bufferPath,
						"data":   rendered,
					})
				} else {
					e.asyncEvents <- events.NewCustomEvent("terminal-output", map[string]interface{}{
						"buffer": bufferPath,
						"data":   string(data),
					})
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
				e.asyncEvents <- events.NewCustomEvent("terminal-output", map[string]interface{}{
					"buffer": bufferPath,
					"data":   rendered,
				})
				// Also switch the editor out of TerminalMode so user returns to Normal
				e.asyncEvents <- events.NewCustomEvent("terminal-closed", map[string]interface{}{
					"buffer": bufferPath,
				})
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
	if e.buffer == nil {
		return nil
	}
	// Compute the range once and reuse the YankOperator's extraction logic
	lineNum := e.buffer.CursorIterator.LineNumber()
	cursorX := e.buffer.CursorX
	iter := e.buffer.CursorIterator
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
