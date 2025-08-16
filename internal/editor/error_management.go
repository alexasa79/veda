package editor

import (
    "fmt"
    "strings"
    "sync"
    "time"
    "veda/internal/buffer"
    "veda/internal/config"
)

// ErrorManager handles error display and tracking
type ErrorManager struct {
	mu              sync.Mutex
	currentError    string      // Current error to display
    currentSeverity string      // "error" or "info" (affects banner style)
	errorShowTime   time.Time   // When the current error was shown
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

// ShowInfo displays a temporary informational banner above the status bar.
// Unlike ShowError, this does not log to the *Messages* buffer.
func (e *Editor) ShowInfo(format string, args ...interface{}) {
    e.errorManager.ShowInfo(format, args...)
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
    e.ShowInfo("%s — %d lines, %d bytes, %s", path, lines, bytes, indent)
    return nil
}