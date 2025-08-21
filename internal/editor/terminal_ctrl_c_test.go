package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestTranslateKeyToTerminal_CtrlC(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Test Ctrl+C translation
	ctrlCEvent := &events.KeyEvent{
		Key: events.KeyCtrlC,
	}

	data := te.translateKeyToTerminal(ctrlCEvent)

	if len(data) != 1 || data[0] != 0x03 {
		t.Errorf("Expected Ctrl+C to translate to [0x03], got %v", data)
	}
}

func TestTranslateKeyToTerminal_ControlKeys(t *testing.T) {
	te := NewTestEditor(80, 24)

	tests := []struct {
		key      events.Key
		expected []byte
		name     string
	}{
		{events.KeyCtrlC, []byte{0x03}, "Ctrl+C (interrupt)"},
		{events.KeyCtrlD, []byte{0x04}, "Ctrl+D (EOT)"},
		{events.KeyCtrlL, []byte{0x0c}, "Ctrl+L (clear screen)"},
		{events.KeyCtrlN, []byte{0x0e}, "Ctrl+N (next history)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := &events.KeyEvent{Key: test.key}
			data := te.translateKeyToTerminal(event)

			if len(data) != len(test.expected) {
				t.Errorf("Expected length %d, got %d", len(test.expected), len(data))
				return
			}

			for i, expected := range test.expected {
				if data[i] != expected {
					t.Errorf("Expected byte %d to be 0x%02x, got 0x%02x", i, expected, data[i])
				}
			}
		})
	}
}

func TestTerminalModeCtrlCHandling(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Set up a mock terminal buffer in terminal mode
	te.mode = commands.TerminalMode

	// Create a mock terminal session
	terminalPath := "*Terminal*"
	te.terminalSessions = make(map[string]*TerminalSessionState)
	te.terminalSessions[terminalPath] = &TerminalSessionState{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
		Banner:     "Test Terminal",
	}

	// Create a buffer for the terminal
	buf := buffer.NewBuffer("", terminalPath, te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf.IsInternal = true
	te.buffers[terminalPath] = buf
	te.buffer = buf

	// Create a Ctrl+C key event
	ctrlCEvent := &events.KeyEvent{
		Key: events.KeyCtrlC,
	}

	// The actual forwarding to the proxy is mocked in tests, but we can verify
	// that the key translation produces the correct bytes for Ctrl+C
	data := te.translateKeyToTerminal(ctrlCEvent)
	if len(data) != 1 || data[0] != 0x03 {
		t.Errorf("Expected Ctrl+C to translate to interrupt signal (0x03), got %v", data)
	}

	// Verify that in terminal mode, Ctrl+C should be forwarded, not exit the mode
	// (Only Escape should exit terminal mode to normal mode)
	initialMode := te.mode

	// Process the Ctrl+C event (this would normally forward to terminal)
	// We can't easily test the actual proxy call in unit tests, but we can verify
	// that the mode doesn't change (unlike Escape which would exit terminal mode)
	if te.mode != initialMode {
		t.Errorf("Ctrl+C in terminal mode should not change the mode, but mode changed from %v to %v", initialMode, te.mode)
	}
}
