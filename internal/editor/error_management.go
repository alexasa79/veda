package editor

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"veda/internal/buffer"
	"veda/internal/clipboard"
	"veda/internal/config"
)

// ErrorManager handles error display and tracking
type ErrorManager struct {
	mu              sync.Mutex
	currentError    string        // Current error to display
	currentSeverity string        // "error" or "info" (affects banner style)
	errorShowTime   time.Time     // When the current error was shown
	errorDuration   time.Duration // How long to show errors (default 2 seconds)
}

// NewErrorManager creates a new error manager
func NewErrorManager() *ErrorManager {
	return &ErrorManager{
		errorDuration: 2 * time.Second,
	}
}

// ShowError displays an error message
func (em *ErrorManager) ShowError(format string, args ...interface{}) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.currentError = fmt.Sprintf(format, args...)
	em.currentSeverity = "error"
	em.errorShowTime = time.Now()
}

// ShowInfo displays a low-severity informational message
func (em *ErrorManager) ShowInfo(format string, args ...interface{}) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.currentError = fmt.Sprintf(format, args...)
	em.currentSeverity = "info"
	em.errorShowTime = time.Now()
}

// GetCurrentBanner returns the current banner message and its severity
// if it should still be displayed. Returns ("", "") when none.
func (em *ErrorManager) GetCurrentBanner() (string, string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.currentError == "" {
		return "", ""
	}

	// Check if error has expired
	if time.Since(em.errorShowTime) > em.errorDuration {
		em.currentError = ""
		em.currentSeverity = ""
		return "", ""
	}

	return em.currentError, em.currentSeverity
}

// ClearError immediately clears the current error
func (em *ErrorManager) ClearError() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.currentError = ""
	em.currentSeverity = ""
}

// GetCurrentError retains the legacy API used by existing code/tests.
// It returns the current banner message (error or info) if visible, otherwise an empty string.
func (em *ErrorManager) GetCurrentError() string {
	msg, _ := em.GetCurrentBanner()
	return msg
}

// createMessagesBuffer creates the internal "Messages" buffer
func (e *Editor) createMessagesBuffer() *buffer.Buffer {
	// Create header content
	headerContent := "Message Log\n" + strings.Repeat("=", 80) + "\n\n"

	messagesBuffer := buffer.NewBuffer(headerContent, "*Messages*", e.config.DefaultIndentType, e.config.DefaultIndentWidth)
	messagesBuffer.IsReadOnly = true
	messagesBuffer.IsInternal = true

	return messagesBuffer
}

// getOrCreateMessagesBuffer gets or creates the messages buffer
func (e *Editor) getOrCreateMessagesBuffer() *buffer.Buffer {
	bufferName := "*Messages*"

	// Check if buffer already exists
	if buf, exists := e.buffers[bufferName]; exists {
		return buf
	}

	// Create new messages buffer
	messagesBuffer := e.createMessagesBuffer()
	e.buffers[bufferName] = messagesBuffer
	e.bufferOrder = append(e.bufferOrder, bufferName)

	return messagesBuffer
}

// ShowError displays an error to the user and logs it to the messages buffer
func (e *Editor) ShowError(format string, args ...interface{}) {
	errorMsg := fmt.Sprintf(format, args...)

	// Show in error line
	e.errorManager.ShowError("%s", errorMsg)

	// Log to messages buffer
	messagesBuffer := e.getOrCreateMessagesBuffer()

	// Format with timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, errorMsg)

	// Append to messages buffer; recreating iterators afterwards to avoid generation mismatch
	messagesBuffer.AppendLine(logEntry)

	// Ensure the buffer stays read-only
	messagesBuffer.IsReadOnly = true

	// Recreate iterators to match new generation (whether current or background)
	messagesBuffer.ViewIterator = buffer.NewBufferIterator(messagesBuffer)
	messagesBuffer.CursorIterator = buffer.NewBufferIterator(messagesBuffer)
}

// errorCommand implements the "error" command for testing
func (e *Editor) errorCommand(args []string) error {
	if len(args) == 0 {
		e.ShowError("Test error with no arguments")
		return nil
	}

	e.ShowError("%s", strings.Join(args, " "))
	return nil
}

// ShowTransientInfo displays a temporary informational banner that persists
// until the next key press (separate from ErrorManager time-based banners).
func (e *Editor) ShowTransientInfo(format string, args ...interface{}) {
	e.transientInfoMessage = fmt.Sprintf(format, args...)
	e.transientInfoActive = true
}

// showFileInfoCommand shows information about the current file/buffer
func (e *Editor) showFileInfoCommand(args []string) error {
	if e.buffer == nil {
		return nil
	}
	// Gather basic info
	display := e.buffer.DisplayName()
	path := e.buffer.FilePath
	if path == "" {
		path = display
	}
	// Normalize to full path for local files; keep HOST:PATH for remotes
	if host, remotePath, isRemote := parsePathComponents(path); isRemote {
		// Ensure we display HOST:REMOTE_PATH
		if host != "" && remotePath != "" {
			path = host + ":" + remotePath
		}
	} else {
		// Local: make absolute if not already
		if !strings.HasPrefix(path, "/") {
			if absPath, err := filepath.Abs(path); err == nil {
				path = absPath
			}
		}
	}
	lines := e.buffer.LineCount
	bytes := 0
	if e.buffer.Rope != nil {
		bytes = e.buffer.Rope.Length()
	}
	indent := "Spaces"
	if e.buffer.Indentation.Type == config.IndentTabs {
		indent = "Tabs"
	} else {
		indent = fmt.Sprintf("Spaces:%d", e.buffer.Indentation.Width)
	}
	e.ShowTransientInfo("%s — %d lines, %d bytes, %s", path, lines, bytes, indent)
	return nil
}

// regcopy: copy contents between registers. Treat '0' as system clipboard.
// Usage: :regcopy <src> <dst>
func (e *Editor) regcopyCmd(args []string) error {
	if len(args) < 2 {
		e.ShowTransientInfo("Usage: :regcopy <src> <dst>")
		return nil
	}
	srcName := []rune(args[0])
	dstName := []rune(args[1])
	if len(srcName) != 1 || len(dstName) != 1 {
		e.ShowTransientInfo("Registers are single characters")
		return nil
	}
	if e.registers == nil {
		e.registers = make(map[rune]string)
	}
	// Resolve source content
	var content string
	if srcName[0] == '0' {
		// System clipboard as source
		if c, err := clipboard.Paste(); err == nil {
			content = c
		} else {
			e.ShowTransientInfo("Clipboard read failed")
			return nil
		}
	} else {
		content = e.registers[srcName[0]]
	}
	// Write to destination
	if dstName[0] == '0' {
		// System clipboard as destination
		if err := clipboard.Copy(content); err != nil {
			e.ShowTransientInfo("Clipboard write failed")
			return nil
		}
	} else {
		e.registers[dstName[0]] = content
	}
	e.ShowTransientInfo("Registers copied")
	return nil
}

// reg: show non-empty registers in an internal read-only buffer
func (e *Editor) showRegistersCmd(args []string) error {
	if e.registers == nil {
		e.registers = make(map[rune]string)
	}
	var b strings.Builder
	b.WriteString("# Registers\n\n")
	// Include system clipboard as '0'
	if clip, err := clipboard.Paste(); err == nil && clip != "" {
		b.WriteString("['0'] System Clipboard\n\n")
		// Show a preview with escaped newlines for readability
		preview := clip
		if len(preview) > 500 {
			preview = preview[:500] + "…"
		}
		preview = strings.ReplaceAll(preview, "\n", "\\n")
		b.WriteString("    ")
		b.WriteString(preview)
		b.WriteString("\n\n")
	}
	// Show named registers
	count := 0
	for r, v := range e.registers {
		if v == "" {
			continue
		}
		count++
		b.WriteString(fmt.Sprintf("['%c']\n\n", r))
		preview := v
		if len(preview) > 500 {
			preview = preview[:500] + "…"
		}
		preview = strings.ReplaceAll(preview, "\n", "\\n")
		b.WriteString("    ")
		b.WriteString(preview)
		b.WriteString("\n\n")
	}
	if count == 0 && !strings.Contains(b.String(), "System Clipboard") {
		b.WriteString("(no non-empty registers)\n")
	}

	// Save current buffer state and open internal buffer
	e.saveCurrentBufferState()
	bufferName := "*Registers*"
	content := b.String()
	if buf, exists := e.buffers[bufferName]; exists {
		buf.Rope = buffer.NewRopeFromString(content)
		buf.CursorX = 0
		buf.ViewIterator = buffer.NewBufferIterator(buf)
		buf.CursorIterator = buffer.NewBufferIterator(buf)
		buf.IsReadOnly = true
		buf.IsInternal = true
		e.switchToBuffer(buf)
	} else {
		newBuffer := buffer.NewBuffer(content, bufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		newBuffer.IsReadOnly = true
		newBuffer.IsInternal = true
		e.buffers[bufferName] = newBuffer
		e.bufferOrder = append(e.bufferOrder, bufferName)
		e.switchToBuffer(newBuffer)
	}
	e.clearSearch()
	return nil
}
