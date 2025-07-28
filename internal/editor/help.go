package editor

import (
	"strings"
	"veda/internal/buffer"
)

// showHelp creates a buffer with all available commands and their descriptions
func (e *Editor) showHelp(args []string) error {
	// Create help content
	var helpText strings.Builder
	
	helpText.WriteString("=== Veda Editor Help ===\n\n")
	helpText.WriteString("Veda is a terminal-based text editor with vim-like features.\n\n")
	
	// Mode information
	helpText.WriteString("== Modes ==\n")
	helpText.WriteString("Normal Mode  - Default mode for navigation and commands (ESC to enter)\n")
	helpText.WriteString("Insert Mode  - Mode for inserting text (i to enter)\n")
	helpText.WriteString("Command Mode - Mode for executing commands (: to enter)\n\n")
	
	// Key bindings by category
	helpText.WriteString("== Key Bindings ==\n\n")
	
	helpText.WriteString("= Movement (Normal Mode) =\n")
	helpText.WriteString("↑/↓/←/→      - Move cursor up/down/left/right\n")
	helpText.WriteString("w            - Move forward by word\n")
	helpText.WriteString("b            - Move backward by word\n")
	helpText.WriteString("0 or Home    - Move to beginning of line\n")
	helpText.WriteString("$ or End     - Move to end of line\n")
	helpText.WriteString("gg           - Go to beginning of file\n")
	helpText.WriteString("G            - Go to end of file\n")
	helpText.WriteString("PageUp       - Move up one page\n")
	helpText.WriteString("PageDown     - Move down one page\n\n")
	
	helpText.WriteString("= Editing =\n")
	helpText.WriteString("i            - Enter insert mode\n")
	helpText.WriteString("o            - Open new line below and enter insert mode\n")
	helpText.WriteString("O            - Open new line above and enter insert mode\n")
	helpText.WriteString("x            - Delete character under cursor\n")
	helpText.WriteString("d<motion>    - Delete with motion (e.g., dw = delete word)\n")
	helpText.WriteString("u            - Undo last change\n")
	helpText.WriteString("U            - Redo undone change\n")
	helpText.WriteString(".            - Repeat last change\n\n")
	
	helpText.WriteString("= Search =\n")
	helpText.WriteString("/            - Enter search mode\n")
	helpText.WriteString("n            - Find next occurrence\n")
	helpText.WriteString("N            - Find previous occurrence\n\n")
	
	helpText.WriteString("= File Operations =\n")
	helpText.WriteString("Space s f    - Open file browser\n")
	helpText.WriteString("Space s b    - Show open buffers\n\n")
	
	helpText.WriteString("= Code Navigation =\n")
	helpText.WriteString("gd           - Go to definition (requires LSP)\n")
	helpText.WriteString("g,           - Go back to previous location\n")
	helpText.WriteString("gs           - Show signature help\n")
	helpText.WriteString("Space s w    - List project symbols\n")
	helpText.WriteString("Space s d    - List document symbols\n\n")
	
	helpText.WriteString("= Insert Mode =\n")
	helpText.WriteString("ESC          - Exit insert mode\n")
	helpText.WriteString("Enter        - Insert newline\n")
	helpText.WriteString("Backspace    - Delete character before cursor\n")
	helpText.WriteString("↑/↓/←/→      - Move cursor\n")
	helpText.WriteString("Home/End     - Move to beginning/end of line\n\n")
	
	helpText.WriteString("== Command Mode Commands ==\n")
	helpText.WriteString(":q or :quit  - Quit the editor\n")
	helpText.WriteString(":cd <path>   - Change working directory\n")
	helpText.WriteString(":help        - Show this help\n\n")
	
	helpText.WriteString("== Configuration ==\n")
	helpText.WriteString(":expandtab=true/false     - Use spaces instead of tabs\n")
	helpText.WriteString(":tabstop=<n> or :ts=<n>  - Set tab width\n")
	helpText.WriteString(":log=<level>              - Set log level (info, debug, trace)\n\n")
	
	helpText.WriteString("== Tips ==\n")
	helpText.WriteString("- Press TAB in command mode for completion\n")
	helpText.WriteString("- Use HOST:PATH format to open remote files (e.g., server:/path/to/file)\n")
	helpText.WriteString("- The editor automatically detects file types and provides syntax highlighting\n")
	helpText.WriteString("- Auto-pairs are enabled for (), [], {}, \"\", '', ``\n")
	
	// Save current buffer state
	e.saveCurrentBufferState()
	
	// Check if help buffer already exists
	helpBufferName := "*Help*"
	if buf, exists := e.buffers[helpBufferName]; exists {
		// Update existing buffer's content
		buf.Rope = buffer.NewRopeFromString(helpText.String())
		buf.CursorX = 0
		buf.ViewIterator = buffer.NewBufferIterator(buf)
		buf.CursorIterator = buffer.NewBufferIterator(buf)
		e.switchToBuffer(buf)
	} else {
		// Create new buffer
		newBuffer := buffer.NewBuffer(helpText.String(), helpBufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		e.buffers[helpBufferName] = newBuffer
		e.bufferOrder = append(e.bufferOrder, helpBufferName)
		e.switchToBuffer(newBuffer)
	}
	
	// Position at the top of the help
	e.buffer.CursorIterator = buffer.NewBufferIterator(e.buffer)
	e.buffer.ViewIterator = buffer.NewBufferIterator(e.buffer)
	e.buffer.CursorX = 0
	
	return nil
}