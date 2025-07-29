package editor

import (
	"strconv"
	"strings"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/events"
	"veda/internal/log"
)

// Mode management functions handle switching between different editor modes

// handleCommandMode handles key events in command mode
func (e *Editor) handleCommandMode(ev *events.KeyEvent) {
	switch ev.Key {
	case events.KeyEscape:
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

	// Store command arguments for handlers that need them
	e.lastCommandArgs = args

	// Try to execute from the command registry
	if e.commandRegistry.Has(cmd) {
		err := e.commandRegistry.Execute(cmd, args)
		if err != nil {
			LL.Debug("Command execution failed: %v", err)
		}
		return
	}
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
			e.buffer.SetIndentationType(config.IndentSpaces)
		} else if value == "false" || value == "0" || value == "no" {
			e.buffer.SetIndentationType(config.IndentTabs)
		}
	case "tabstop", "ts":
		// Set tab width
		if width, err := strconv.Atoi(value); err == nil && width > 0 && width <= 8 {
			e.buffer.SetIndentationWidth(width)
		}
	case "shiftwidth", "sw":
		// Set indent width (same as tabstop for now)
		if width, err := strconv.Atoi(value); err == nil && width > 0 && width <= 8 {
			e.buffer.SetIndentationWidth(width)
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
	}
}

// enterInsertMode switches the editor to insert mode
func (e *Editor) enterInsertMode(args []string) error {
	e.mode = commands.InsertMode
	// Start tracking insert mode changes
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.StartInsertSession("i")
	}
	// Visual selection is handled in the editor struct, not buffer
	return nil
}

// exitInsertMode switches the editor back to normal mode
func (e *Editor) exitInsertMode(args []string) error {
	// End the insert session
	if e.buffer != nil && e.buffer.History != nil {
		e.buffer.History.EndInsertSession()
	}

	e.mode = commands.NormalMode

	// Visual selection is handled in the editor struct, not buffer

	// Adjust cursor position if needed (vim behavior)
	line := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > 0 && e.buffer.CursorX >= len(line) {
		e.buffer.CursorX = len(line) - 1
		if e.buffer.CursorX < 0 {
			e.buffer.CursorX = 0
		}
	}

	// The setLastChange will be handled by the history system

	return nil
}

// enterVisualMode switches the editor to visual mode
func (e *Editor) enterVisualMode(args []string) error {
	e.mode = commands.VisualMode
	// Initialize visual selection at current position
	// Visual selection tracking would be handled in the editor
	// This functionality needs to be refactored to use editor fields
	// Exit insert mode if we're in it
	if e.buffer != nil && e.buffer.History != nil && false /* TODO: Check if in insert session */ {
		e.buffer.History.EndInsertSession()
	}
	return nil
}

// exitVisualMode switches back to normal mode and clears visual selection
func (e *Editor) exitVisualMode(args []string) error {
	e.mode = commands.NormalMode
	// Clear visual selection
	// Clear visual selection - this would be handled in the editor
	return nil
}

// enterCommandModeCmd is the command handler for entering command mode
func (e *Editor) enterCommandModeCmd(args []string) error {
	e.enterCommandMode()
	return nil
}