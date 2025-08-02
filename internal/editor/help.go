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
	helpText.WriteString("Visual Mode  - Mode for selecting text (v to enter)\n")
	helpText.WriteString("Command Mode - Mode for executing commands (: to enter)\n")
	helpText.WriteString("Cursor Mode  - Mode for adding multiple cursors (m to enter)\n\n")
	
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
	helpText.WriteString("d<motion>    - Delete with motion (e.g., dw = delete word, dd = delete line)\n")
	helpText.WriteString("c<motion>    - Change with motion (e.g., cw = change word, cc = change line)\n")
	helpText.WriteString("y<motion>    - Yank with motion (e.g., yw = yank word, yy = yank line)\n")
	helpText.WriteString("p            - Paste after cursor/line\n")
	helpText.WriteString("P            - Paste before cursor/line\n")
	helpText.WriteString("u            - Undo last change\n")
	helpText.WriteString("U            - Redo undone change\n")
	helpText.WriteString(".            - Repeat last change\n")
	helpText.WriteString("v            - Enter visual mode for character selection\n")
	helpText.WriteString("V            - Enter visual line mode for line selection\n")
	helpText.WriteString("J            - Join current line with next line\n")
	helpText.WriteString("Alt+Q        - Rewrap current paragraph\n")
	helpText.WriteString("m            - Enter cursor mode for multiple cursors\n\n")
	
	helpText.WriteString("= Cursor Mode =\n")
	helpText.WriteString("Enter        - Add virtual cursor at current position and move down\n")
	helpText.WriteString("m or ESC     - Exit cursor mode\n")
	helpText.WriteString("Movement keys work normally (only primary cursor moves)\n")
	helpText.WriteString("Any editing command exits cursor mode\n\n")
	
	helpText.WriteString("= Search =\n")
	helpText.WriteString("/            - Enter search mode\n")
	helpText.WriteString("*            - Search for word under cursor\n")
	helpText.WriteString("n            - Find next occurrence\n")
	helpText.WriteString("N            - Find previous occurrence\n")
	helpText.WriteString("ESC          - Clear current search\n\n")
	
	helpText.WriteString("= File Operations =\n")
	helpText.WriteString("Space s f    - Open file browser\n")
	helpText.WriteString("Space s b    - Show open buffers\n")
	helpText.WriteString("Space Space  - Show buffers (cursor on previous)\n\n")
	
	helpText.WriteString("= Code Navigation =\n")
	helpText.WriteString("gd           - Go to definition (requires LSP)\n")
	helpText.WriteString("g,           - Go back to previous location\n")
	helpText.WriteString("gs           - Show signature help\n")
	helpText.WriteString("gr           - Find all references to symbol under cursor\n")
	helpText.WriteString("ge           - Show project diagnostics/errors\n")
	helpText.WriteString("gt           - Show git changes in project\n")
	helpText.WriteString("gn           - Go to next reference (when references are found)\n")
	helpText.WriteString("gp           - Go to previous reference (when references are found)\n")
	helpText.WriteString("Space s w    - List project symbols\n")
	helpText.WriteString("Space s d    - List document symbols\n\n")
	
	helpText.WriteString("= References Buffer =\n")
	helpText.WriteString("Enter        - Jump to reference under cursor\n")
	helpText.WriteString("gn/gp        - Navigate through all references\n\n")
	
	helpText.WriteString("= Visual Mode =\n")
	helpText.WriteString("v            - Enter character visual mode\n")
	helpText.WriteString("V            - Enter line visual mode\n")
	helpText.WriteString("ESC          - Exit visual mode\n")
	helpText.WriteString("d or x       - Delete selected text\n")
	helpText.WriteString("y            - Yank (copy) selected text to clipboard\n")
	helpText.WriteString("c            - Change selected text (delete and enter insert mode)\n")
	helpText.WriteString("All movement keys work in visual mode\n\n")
	
	helpText.WriteString("= Insert Mode =\n")
	helpText.WriteString("ESC          - Exit insert mode\n")
	helpText.WriteString("Enter        - Insert newline\n")
	helpText.WriteString("Tab          - Insert indentation (spaces or tab based on settings)\n")
	helpText.WriteString("Backspace    - Delete character before cursor\n")
	helpText.WriteString("Delete       - Delete character under cursor\n")
	helpText.WriteString("↑/↓/←/→      - Move cursor\n")
	helpText.WriteString("Home/End     - Move to beginning/end of line\n")
	helpText.WriteString("Alt+←/→      - Move cursor by word\n")
	helpText.WriteString("Alt+Backspace - Delete word before cursor\n")
	helpText.WriteString("Alt+Delete   - Delete word after cursor\n")
	helpText.WriteString("Ctrl+Space   - Show completion menu\n\n")
	
	helpText.WriteString("= Completion Menu =\n")
	helpText.WriteString("↑/↓          - Navigate through completion items\n")
	helpText.WriteString("Enter        - Accept selected completion\n")
	helpText.WriteString("ESC          - Close completion menu\n")
	helpText.WriteString("Any other key - Close menu and process key normally\n\n")
	
	helpText.WriteString("== Command Mode Commands ==\n")
	helpText.WriteString(":q           - Close current buffer (quits if last buffer)\n")
	helpText.WriteString(":q!          - Force quit (same as :Q)\n")
	helpText.WriteString(":Q or :Quit  - Quit the editor\n")
	helpText.WriteString(":w or :write - Write current file\n")
	helpText.WriteString(":wq          - Write file and close buffer\n")
	helpText.WriteString(":e <path>    - Edit file or directory\n")
	helpText.WriteString(":cd <path>   - Change working directory\n")
	helpText.WriteString(":help        - Show this help\n")
	helpText.WriteString(":help-keys   - Show all key bindings\n")
	helpText.WriteString(":diagnostics - Show project diagnostics/errors\n")
	helpText.WriteString(":show-project-changes - Show all git changes in project\n\n")
	
	helpText.WriteString("== Configuration ==\n")
	helpText.WriteString(":expandtab=true/false     - Use spaces instead of tabs\n")
	helpText.WriteString(":tabstop=<n> or :ts=<n>  - Set tab width\n")
	helpText.WriteString(":log=<level>              - Set log level (info, debug, trace)\n")
	helpText.WriteString(":rulers=<n>,<n>,...       - Set vertical ruler positions (e.g. rulers=78,80,100,120)\n\n")
	
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