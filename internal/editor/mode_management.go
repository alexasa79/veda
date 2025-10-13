package editor

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/events"
	"veda/internal/log"
	"veda/internal/widgets"
)

// Mode management functions handle switching between different editor modes

// handleCommandMode handles key events in command mode
func (e *Editor) handleCommandMode(ev *events.KeyEvent) {
	switch ev.Key {
	case events.KeyEscape:
		if e.dismissSignatureIfVisible() {
			return
		}
		// Exit command mode
		e.exitCommandMode()
	case events.KeyEnter:
		// Execute command and exit command mode
		e.executeCommand()
		e.exitCommandMode()
	default:
		// Pass other keys to the command line widget
		e.commandLine.HandleKey(ev)

		// Check if the user deleted all text including the colon
		// If so, exit command mode automatically
		if e.commandLine.GetText() == "" {
			e.exitCommandMode()
		}
	}
}

// enterCommandMode switches to command mode and initializes the command line
func (e *Editor) enterCommandMode() {
	// Clear virtual cursors when entering command mode
	e.clearVirtualCursors()
	// Hide key hints when switching modes
	e.keyHintsPopup.Hide()
	e.mode = commands.CommandMode
	e.commandLine.SetText(":")
	// Position cursor after the colon
	e.commandLine.SetCursor(1)

	// Set up command completion function
	e.commandLine.SetCompletionFunc(e.getCommandCompletions)
}

// exitCommandMode switches back to normal mode
func (e *Editor) exitCommandMode() {
	e.mode = commands.NormalMode
	e.commandLine.SetText("")
	// Hide key hints when exiting command mode
	e.keyHintsPopup.Hide()
}

// handleCursorMode handles key events in cursor mode
func (e *Editor) handleCursorMode(ev *events.KeyEvent) {
	switch ev.Key {
	case events.KeyEscape:
		if e.dismissSignatureIfVisible() {
			return
		}
		// Exit cursor mode
		e.exitCursorMode()
		return
	case events.KeyEnter:
		// Add virtual cursor at current position
		e.addVirtualCursor()
		return
	case events.KeyRune:
		if ev.Rune == 'm' {
			// Exit cursor mode with 'm'
			e.exitCursorMode()
			return
		}
	}

	// First, check for CursorMode-specific commands. Execute without exiting.
	if command, found := e.keyMapper.ProcessKey(ev, commands.CursorMode); found {
		// Execute cursor-mode command directly
		e.repeatCount = 1
		if err := e.commandRegistry.Execute(command, nil); err != nil {
			LL.Debug("Cursor mode command error: %v", err)
		}
		return
	}

	// Otherwise, check if it's a Normal mode movement command
	if command, found := e.keyMapper.ProcessKey(ev, commands.NormalMode); found {
		if isMovementCommand(command) {
			// Execute movement commands normally
			if err := e.commandRegistry.Execute(command, nil); err != nil {
				LL.Debug("Movement command error in cursor mode: %v", err)
			}
		} else {
			// Non-movement commands exit cursor mode first, then execute
			e.exitCursorMode()
			// Use executeWithRepeatCount to properly handle multiple cursors
			e.repeatCount = 1
			if err := e.executeWithRepeatCount(command); err != nil {
				LL.Debug("Command execution error: %v", err)
			}
		}
	}
}

// enterCursorMode switches to cursor mode
func (e *Editor) enterCursorMode() {
	// Start a composite history unit to group all virtual-cursor additions
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil && !e.CurrentBuffer().History.HasActiveComposite() {
		e.CurrentBuffer().History.StartComposite("Add virtual cursors")
		// Record pre-cursor snapshot so undo restores to this state
		primary := buffer.CursorPos{Line: e.CurrentBuffer().CursorIterator.LineNumber(), Char: e.CurrentBuffer().CursorX}
		v := make([]buffer.CursorPos, 0, len(e.virtualCursors))
		for _, vc := range e.virtualCursors {
			v = append(v, buffer.CursorPos{Line: vc.Y, Char: vc.X})
		}
		e.CurrentBuffer().History.SetActiveCompositeCursors(primary, v)
	}
	e.mode = commands.CursorMode
}

// enterCursorModeCmd is the command handler for entering cursor mode
func (e *Editor) enterCursorModeCmd(args []string) error {
	e.enterCursorMode()
	return nil
}

// exitCursorMode switches back to normal mode
func (e *Editor) exitCursorMode() {
	// If a composite for cursor mode is active, finalize it by storing the
	// post-operation cursor snapshot and pushing to history.
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil && e.CurrentBuffer().History.HasActiveComposite() {
		primary := buffer.CursorPos{Line: e.CurrentBuffer().CursorIterator.LineNumber(), Char: e.CurrentBuffer().CursorX}
		v := make([]buffer.CursorPos, 0, len(e.virtualCursors))
		for _, vc := range e.virtualCursors {
			v = append(v, buffer.CursorPos{Line: vc.Y, Char: vc.X})
		}
		e.CurrentBuffer().History.SetActiveCompositePostCursors(primary, v)
		e.CurrentBuffer().History.EndComposite()
	}
	e.mode = commands.NormalMode
}

// clearVirtualCursors removes all virtual cursors
func (e *Editor) clearVirtualCursors() {
	// Clearing cursors should be undoable as a single unit. Start a composite
	// that transitions from current cursor set to none.
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil && !e.CurrentBuffer().History.HasActiveComposite() {
		e.CurrentBuffer().History.StartComposite("Clear virtual cursors")
		// Pre-state: current cursors
		primary := buffer.CursorPos{Line: e.CurrentBuffer().CursorIterator.LineNumber(), Char: e.CurrentBuffer().CursorX}
		vlist := e.GetVirtualCursors()
		v := make([]buffer.CursorPos, 0, len(vlist))
		for _, vc := range vlist {
			v = append(v, buffer.CursorPos{Line: vc.Y, Char: vc.X})
		}
		e.CurrentBuffer().History.SetActiveCompositeCursors(primary, v)
	}
	// Apply the state change
	e.clearVirtualCursorsFor(e.CurrentBuffer())
	// Post-state: no virtual cursors
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil && e.CurrentBuffer().History.HasActiveComposite() {
		primary := buffer.CursorPos{Line: e.CurrentBuffer().CursorIterator.LineNumber(), Char: e.CurrentBuffer().CursorX}
		e.CurrentBuffer().History.SetActiveCompositePostCursors(primary, nil)
		e.CurrentBuffer().History.EndComposite()
	}
}

// addVirtualCursor adds a virtual cursor at the current position and moves primary cursor down
func (e *Editor) addVirtualCursor() {
	// Check if we already have a virtual cursor at this position
	currentX := e.CurrentBuffer().CursorX
	currentY := e.CurrentBuffer().CursorIterator.LineNumber()

	for _, existing := range e.GetVirtualCursors() {
		if existing.X == currentX && existing.Y == currentY {
			// Already have a cursor here, just move down
			if err := e.commandRegistry.Execute("down", nil); err != nil {
				LL.Debug("Failed to move cursor down: %v", err)
			}
			return
		}
	}

	// Add virtual cursor at current position
	cursor := VirtualCursor{
		X: currentX,
		Y: currentY,
	}
	// When adding cursors in cursor mode, treat them as one composite group.
	// Ensure a composite is open from enterCursorMode; if not, open one now.
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil && !e.CurrentBuffer().History.HasActiveComposite() {
		// if no active composite, start one capturing current state
		// (this covers programmatic calls/tests)
		e.CurrentBuffer().History.StartComposite("Add virtual cursors")
		primary := buffer.CursorPos{Line: e.CurrentBuffer().CursorIterator.LineNumber(), Char: e.CurrentBuffer().CursorX}
		vlist := e.GetVirtualCursors()
		v := make([]buffer.CursorPos, 0, len(vlist))
		for _, vc := range vlist {
			v = append(v, buffer.CursorPos{Line: vc.Y, Char: vc.X})
		}
		e.CurrentBuffer().History.SetActiveCompositeCursors(primary, v)
	}
	e.appendVirtualCursorFor(e.CurrentBuffer(), cursor)

	// Move primary cursor down one line
	if err := e.commandRegistry.Execute("down", nil); err != nil {
		LL.Debug("Failed to move cursor down after adding virtual cursor: %v", err)
	}
	// Update post-state snapshot continuously so undo can return to initial
	// state after the entire add sequence.
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil && e.CurrentBuffer().History.HasActiveComposite() {
		primary := buffer.CursorPos{Line: e.CurrentBuffer().CursorIterator.LineNumber(), Char: e.CurrentBuffer().CursorX}
		vlist := e.GetVirtualCursors()
		v := make([]buffer.CursorPos, 0, len(vlist))
		for _, vc := range vlist {
			v = append(v, buffer.CursorPos{Line: vc.Y, Char: vc.X})
		}
		e.CurrentBuffer().History.SetActiveCompositePostCursors(primary, v)
	}
}

// removeLastVirtualCursor removes the most recently added virtual cursor and adjusts
// the active composite cursor snapshot if present. It also moves the primary cursor
// up one line to mirror the add behavior.
func (e *Editor) removeLastVirtualCursor() {
	if e.CurrentBuffer() == nil {
		return
	}
	vcs := e.GetVirtualCursors()
	if len(vcs) == 0 {
		return
	}
	// Remove the last-added virtual cursor
	newList := make([]VirtualCursor, len(vcs)-1)
	copy(newList, vcs[:len(vcs)-1])
	e.setVirtualCursorsFor(e.CurrentBuffer(), newList)

	// Move primary cursor up one line (if possible) to mirror the add operation
	_ = e.commandRegistry.Execute("up", nil)

	// Update post-state snapshot in the active composite so a later undo still restores
	if e.CurrentBuffer().History != nil && e.CurrentBuffer().History.HasActiveComposite() {
		primary := buffer.CursorPos{Line: e.CurrentBuffer().CursorIterator.LineNumber(), Char: e.CurrentBuffer().CursorX}
		v := make([]buffer.CursorPos, 0, len(newList))
		for _, vc := range newList {
			v = append(v, buffer.CursorPos{Line: vc.Y, Char: vc.X})
		}
		e.CurrentBuffer().History.SetActiveCompositePostCursors(primary, v)
	}
}

// removeLastVirtualCursorCmd is the command handler for removing the last virtual cursor
func (e *Editor) removeLastVirtualCursorCmd(args []string) error {
	// Only respond in cursor mode
	if e.mode != commands.CursorMode {
		return nil
	}
	e.removeLastVirtualCursor()
	return nil
}

// addVirtualCursorCmd is the command handler for adding a virtual cursor
func (e *Editor) addVirtualCursorCmd(args []string) error {
	// Only allow adding virtual cursors in cursor mode
	if e.mode != commands.CursorMode {
		return nil
	}
	e.addVirtualCursor()
	return nil
}

// isMovementCommand checks if a command is a movement command
func isMovementCommand(command string) bool {
	movementCommands := []string{
		"up", "down", "left", "right",
		"word-forward", "word-backward",
		"beginning-of-line", "end-of-line",
		"page-up", "page-down",
		"goto-line-start", "goto-line-end",
		"next-word", "previous-word",
		"find-char", "find-char-backward",
	}

	for _, cmd := range movementCommands {
		if command == cmd {
			return true
		}
	}
	return false
}

// executeCommand processes the command entered in the command line
func (e *Editor) executeCommand() {
	fullCommand := e.commandLine.GetText()

	// Add to history before executing
	e.commandLine.AddToHistory(fullCommand)

	// Remove the leading ":"
	command := fullCommand
	if len(command) > 1 {
		command = command[1:]
	}

	// Check if this is a variable assignment (VAR=VALUE)
	if strings.Contains(command, "=") && !strings.HasPrefix(command, " ") {
		e.handleVariableAssignment(command)
		return
	}

	// Check if the command is a line number
	if lineNum, err := strconv.Atoi(command); err == nil {
		// Jump to line number (1-based, but 0 also goes to line 1)
		if lineNum <= 0 {
			e.goToLine(0) // Line 0 or negative goes to first line
		} else {
			e.goToLine(lineNum - 1) // Convert to 0-based
		}
		return
	}

	// Parse command and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}
	cmd := parts[0]
	args := parts[1:]

	// Check for force flag (! suffix)
	force := false
	if strings.HasSuffix(cmd, "!") {
		force = true
		cmd = strings.TrimSuffix(cmd, "!")
	}

	// Store command arguments for handlers that need them
	e.lastCommandArgs = args

	// Handle special commands that support warnings
	switch cmd {
	case "q", "quit":
		err := e.quitWithWarning(args, force)
		if err != nil {
			e.ShowError("Command failed: %v", err)
		}
		return
	case "qa", "qall":
		err := e.quitAllWithWarning(args, force)
		if err != nil {
			e.ShowError("Command failed: %v", err)
		}
		return
	}

	// Try to execute from the command registry
	if e.commandRegistry.Has(cmd) {
		// If this command is marked to apply to all cursors and we have
		// active virtual cursors (and are not in CursorMode), route through
		// the multi-cursor executor so behavior matches key-initiated commands.
		if info, ok := e.commandRegistry.GetCommand(cmd); ok {
			if e.mode != commands.CursorMode && len(e.virtualCursors) > 0 && info.ApplyToAllCursors {
				if err := e.executeCommandForAllCursors(cmd, args); err != nil {
					e.ShowError("Command failed: %v", err)
				}
				return
			}
		}
		// Otherwise execute normally
		if err := e.commandRegistry.Execute(cmd, args); err != nil {
			e.ShowError("Command failed: %v", err)
		}
		return
	}

	// Command not found
	e.ShowError("Unknown command: %s", cmd)
}

// handleVariableAssignment processes variable assignments in VAR=VALUE format
func (e *Editor) handleVariableAssignment(assignment string) {
	parts := strings.SplitN(assignment, "=", 2)
	if len(parts) != 2 {
		return
	}

	varName := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch varName {
	case "expandtab":
		// Boolean value
		if value == "true" || value == "1" || value == "yes" {
			e.CurrentBuffer().SetIndentationType(config.IndentSpaces)
		} else if value == "false" || value == "0" || value == "no" {
			e.CurrentBuffer().SetIndentationType(config.IndentTabs)
		}
	case "tabstop", "ts":
		// Set tab width
		if width, err := strconv.Atoi(value); err == nil && width > 0 && width <= 8 {
			e.CurrentBuffer().SetIndentationWidth(width)
		}
	case "shiftwidth", "sw":
		// Set indent width (same as tabstop for now)
		if width, err := strconv.Atoi(value); err == nil && width > 0 && width <= 8 {
			e.CurrentBuffer().SetIndentationWidth(width)
		}
	case "number":
		// Show line numbers (future feature)
		// TODO: implement line numbers
	case "relativenumber":
		// Show relative line numbers (future feature)
		// TODO: implement relative line numbers
	case "autoindent":
		// Auto indent (already implemented by default)
		// TODO: make this configurable
	case "smartindent":
		// Smart indent (already implemented by default)
		// TODO: make this configurable
	case "log":
		// Set log level
		if level, err := log.ParseLogLevel(value); err == nil {
			log.SetAllLogLevel(level)
			LL.Info("Log level set to %s", value)
		} else {
			LL.Warning("Invalid log level: %s (use: info, debug, trace)", value)
		}
	case "rulers":
		// Set vertical ruler positions (comma-separated list)
		rulers := []int{}
		if value != "" {
			parts := strings.Split(value, ",")
			for _, part := range parts {
				if pos, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && pos > 0 {
					rulers = append(rulers, pos)
				}
			}
		}
		e.config.VerticalRulers = rulers
		LL.Info("Vertical rulers set to: %v", rulers)
	}
}

// switchToInsertMode is a helper function to handle the common logic for entering insert mode
func (e *Editor) switchToInsertMode() {
	// Hide key hints when switching to insert mode
	e.keyHintsPopup.Hide()
	// Special case: terminal buffers use TerminalMode instead of InsertMode
	if e.CurrentBuffer() != nil && e.CurrentBuffer().IsInternal {
		if ts, ok := e.terminalSessions[e.CurrentBuffer().FilePath]; ok && ts != nil && ts.term != nil {
			// Restore saved terminal cursor if any
			if ts.SavedCurX >= 0 || ts.SavedCurY >= 0 {
				cx, cy := ts.SavedCurX, ts.SavedCurY
				if cx < 0 {
					cx = 0
				}
				if cy < 0 {
					cy = 0
				}
				ts.term.curX, ts.term.curY = cx, cy
			}
			// Re-align the view to follow the terminal cursor location
			bannerOffset := 0
			if ts.Banner != "" {
				bannerOffset = 1
			}
			bufRow := len(ts.term.scrollback) + ts.SavedCurY + bannerOffset
			if bufRow < 0 {
				bufRow = 0
			}
			if bufRow >= e.CurrentBuffer().LineCount {
				bufRow = e.CurrentBuffer().LineCount - 1
			}
			// Auto-follow so the cursor row is visible near bottom
			visible := e.getVisibleHeight()
			if visible < 1 {
				visible = 1
			}
			top := bufRow - (visible - 1)
			if top < 0 {
				top = 0
			}
			if top >= e.CurrentBuffer().LineCount {
				top = e.CurrentBuffer().LineCount - 1
			}
			e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), top)
		}
		e.mode = commands.TerminalMode
		return
	}
	e.mode = commands.InsertMode

	// Check if we're between parentheses and automatically show signature help
	if e.CurrentBuffer() != nil && e.isLikelyInsideCallOnCurrentLine() {
		// Increment generation and request signature help
		e.CurrentBuffer().SignatureGeneration++
		e.requestSignatureHelpAsync()
	}
}

// enterInsertMode switches the editor to insert mode
func (e *Editor) enterInsertMode(args []string) error {
	e.switchToInsertMode()
	// If we switched to TerminalMode for terminal buffers, do not start an insert session
	if e.mode == commands.TerminalMode {
		return nil
	}
	// Start tracking insert mode changes
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.StartInsertSession("i")
	}

	// Visual selection is handled in the editor struct, not buffer
	return nil
}

// enterTemporaryNormalMode enters normal mode temporarily from insert mode (Ctrl+O behavior)
func (e *Editor) enterTemporaryNormalMode(args []string) error {
	if e.mode != commands.InsertMode {
		LL.Debug("enterTemporaryNormalMode: Not in insert mode, current mode: %v", e.mode)
		return nil
	}

	LL.Debug("enterTemporaryNormalMode: Switching from insert to temporary normal mode")

	// Switch to normal mode but mark it as temporary
	e.mode = commands.NormalMode
	e.temporaryNormal = true

	// Don't end the insert session yet - we'll resume it
	return nil
}

// exitInsertMode switches the editor back to normal mode
func (e *Editor) exitInsertMode(args []string) error {
	// Remove trailing spaces from all lines that were changed in this session
	e.removeTrailingSpacesFromChangedLines()

	// Clear temporary normal flag if set
	e.temporaryNormal = false

	// Hide signature help when exiting insert mode
	e.clearSignature(e.CurrentBuffer())

	// End the insert session
	if e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil {
		entry := e.CurrentBuffer().History.EndInsertSession()
		if entry != nil {
			e.setLastChange(entry)
			// Clear simple command when we have a new insert session
			e.lastSimpleCommand = ""
		}
		// If a generic composite was open (e.g., multi-cursor change), close it now
		e.CurrentBuffer().History.EndComposite()
	}

	e.mode = commands.NormalMode

	// Visual selection is handled in the editor struct, not buffer

	// Do not shift cursor left on exiting insert mode for terminal buffers
	if !(e.CurrentBuffer() != nil && e.CurrentBuffer().IsInternal) {
		// Adjust cursor position if needed (vim behavior)
		// When exiting insert mode, vim moves cursor back one position
		// unless we're at the beginning of the line
		if e.CurrentBuffer().CursorX > 0 {
			e.CurrentBuffer().CursorX--
		}
	}

	// Update syntax highlighting after all insert mode changes are complete
	e.updateSyntaxHighlighting()

	// The setLastChange will be handled by the history system

	return nil
}

func (e *Editor) removeTrailingSpacesFromChangedLines() {
	buf := e.CurrentBuffer()
	if buf == nil {
		return
	}
	if len(buf.ChangedLines) == 0 {
		return
	}

	// Preserve cursor/view state so trimming does not shift the user.
	var (
		savedCursorLine = -1
		savedCursorX    = 0
		savedViewLine   = -1
	)
	if buf.CursorIterator != nil && buf.CursorIterator.Generation == buf.Generation {
		savedCursorLine = buf.CursorIterator.LineNumber()
		savedCursorX = buf.CursorX
	}
	if buf.ViewIterator != nil && buf.ViewIterator.Generation == buf.Generation {
		savedViewLine = buf.ViewIterator.LineNumber()
	}

	lines := make([]int, 0, len(buf.ChangedLines))
	for line := range buf.ChangedLines {
		lines = append(lines, line)
	}
	sort.Ints(lines)

	for i := len(lines) - 1; i >= 0; i-- {
		lineNum := lines[i]
		iter := buffer.NewBufferIteratorAtLine(buf, lineNum)
		if iter.IsEOF() {
			continue
		}
		lineText := iter.Line()
		trimIdx := len(lineText)
		for trimIdx > 0 && (lineText[trimIdx-1] == ' ' || lineText[trimIdx-1] == '\t') {
			trimIdx--
		}
		if trimIdx == len(lineText) {
			continue
		}
		trimText := lineText[trimIdx:]
		entry := buffer.NewDeleteTextEntry(lineNum, trimIdx, trimText)
		if buf.History != nil {
			if _, err := buf.History.Execute(buf, entry); err != nil {
				// Fallback: execute directly if history execution fails
				_ = entry.Execute(buf)
			}
		} else {
			_ = entry.Execute(buf)
		}
	}

	// Restore cursor/view after trimming.
	if savedCursorLine >= 0 {
		if savedCursorLine >= buf.LineCount {
			savedCursorLine = buf.LineCount - 1
		}
		if savedCursorLine < 0 {
			savedCursorLine = 0
		}
		buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, savedCursorLine)
		if savedCursorX > len(buf.CursorIterator.Line()) {
			savedCursorX = len(buf.CursorIterator.Line())
		}
		buf.CursorX = savedCursorX
	}
	if savedViewLine >= 0 {
		if savedViewLine >= buf.LineCount {
			savedViewLine = buf.LineCount - 1
		}
		if savedViewLine < 0 {
			savedViewLine = 0
		}
		buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, savedViewLine)
	}
}

// enterVisualMode switches the editor to visual mode
func (e *Editor) enterVisualMode(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}

	e.mode = commands.VisualMode

	// Set visual anchor at current cursor position
	e.CurrentBuffer().VisualAnchorLine = e.CurrentBuffer().CursorIterator.LineNumber()
	e.CurrentBuffer().VisualAnchorX = e.CurrentBuffer().CursorX
	e.CurrentBuffer().VisualAnchorOffset = e.CurrentBuffer().CursorIterator.Position() + e.CurrentBuffer().CursorX

	// Exit insert mode if we're in it
	if e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.EndInsertSession()
	}

	LL.Debug("Entered visual mode: anchor at line %d, x %d, offset %d",
		e.CurrentBuffer().VisualAnchorLine, e.CurrentBuffer().VisualAnchorX, e.CurrentBuffer().VisualAnchorOffset)

	return nil
}

// enterVisualLineMode switches the editor to visual line mode
func (e *Editor) enterVisualLineMode(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}

	e.mode = commands.VisualLineMode

	// Set visual anchor at current cursor line
	e.CurrentBuffer().VisualAnchorLine = e.CurrentBuffer().CursorIterator.LineNumber()
	e.CurrentBuffer().VisualAnchorX = 0 // Line mode always starts from beginning of line
	e.CurrentBuffer().VisualAnchorOffset = e.CurrentBuffer().CursorIterator.Position()

	// Exit insert mode if we're in it
	if e.CurrentBuffer().History != nil {
		e.CurrentBuffer().History.EndInsertSession()
	}

	LL.Debug("Entered visual line mode: anchor at line %d", e.CurrentBuffer().VisualAnchorLine)

	return nil
}

// exitVisualMode switches back to normal mode and clears visual selection
func (e *Editor) exitVisualMode(args []string) error {
	e.mode = commands.NormalMode

	// Clear visual selection
	if e.CurrentBuffer() != nil {
		e.CurrentBuffer().VisualAnchorLine = -1
		e.CurrentBuffer().VisualAnchorX = -1
		e.CurrentBuffer().VisualAnchorOffset = -1
	}

	LL.Debug("Exited visual mode")

	return nil
}

// enterCommandModeCmd is the command handler for entering command mode
func (e *Editor) enterCommandModeCmd(args []string) error {
	e.enterCommandMode()
	return nil
}

// hasUnsavedChanges returns true if any buffer has unsaved changes
func (e *Editor) hasUnsavedChanges() bool {
	for _, buf := range e.buffers {
		if buf.IsModified && buf.HasFile() && !buf.IsInternal {
			return true
		}
	}
	return false
}

// getUnsavedBuffers returns a list of buffer names that have unsaved changes
func (e *Editor) getUnsavedBuffers() []string {
	var unsaved []string
	for _, buf := range e.buffers {
		if buf.IsModified && buf.HasFile() && !buf.IsInternal {
			unsaved = append(unsaved, buf.FileName())
		}
	}
	return unsaved
}

// quitWithWarning handles :q and :quit commands with unsaved change warnings
func (e *Editor) quitWithWarning(args []string, force bool) error {
	// Ignore quit command in empty windows
	if e.CurrentBuffer() == nil {
		return nil
	}

	// Check if current buffer has unsaved changes
	if !force && e.CurrentBuffer().IsModified && e.CurrentBuffer().HasFile() && !e.CurrentBuffer().IsInternal {
		return e.showUnsavedBufferDialog(e.CurrentBuffer().FileName(), "quit")
	}

	// Proceed with normal quit
	return e.closeBuffer(args)
}

// quitAllWithWarning handles :qa and :qall commands with unsaved change warnings
func (e *Editor) quitAllWithWarning(args []string, force bool) error {
	// Check for unsaved changes in any buffer
	if !force && e.hasUnsavedChanges() {
		unsavedBuffers := e.getUnsavedBuffers()
		if len(unsavedBuffers) == 1 {
			return e.showUnsavedBufferDialog(unsavedBuffers[0], "quit all")
		} else {
			return e.showUnsavedBuffersDialog(unsavedBuffers)
		}
	}

	// Proceed with normal quit all
	return e.actuallyQuit(args)
}

// showUnsavedBufferDialog shows a dialog for a single unsaved buffer
func (e *Editor) showUnsavedBufferDialog(fileName, action string) error {
	width, height := e.screen.Size()
	dialogWidth := 60
	dialogHeight := 8
	x := (width - dialogWidth) / 2
	y := (height - dialogHeight) / 2

	title := "Unsaved Changes"
	message := fmt.Sprintf("Buffer '%s' has unsaved changes. %s without saving?", fileName, action)

	dialog := widgets.NewYesNoDialog(x, y, dialogWidth, dialogHeight, title, message, e.config.Theme)
	overlay := NewUnsavedBufferDialogOverlay(dialog, fileName, action, false, e)
	e.renderingState.SetOverlay(overlay)

	return nil // Dialog will be handled by the overlay system
}

// showUnsavedBuffersDialog shows a dialog for multiple unsaved buffers
func (e *Editor) showUnsavedBuffersDialog(fileNames []string) error {
	width, height := e.screen.Size()
	dialogWidth := 70
	dialogHeight := 10
	x := (width - dialogWidth) / 2
	y := (height - dialogHeight) / 2

	title := "Multiple Unsaved Changes"
	fileList := strings.Join(fileNames, ", ")
	message := fmt.Sprintf("The following buffers have unsaved changes: %s. Quit without saving?", fileList)

	dialog := widgets.NewYesNoDialog(x, y, dialogWidth, dialogHeight, title, message, e.config.Theme)
	overlay := NewUnsavedBufferDialogOverlay(dialog, "", "quit all", true, e)
	e.renderingState.SetOverlay(overlay)

	return nil // Dialog will be handled by the overlay system
}
