package editor

import (
	"testing"
	"veda/internal/buffer"
)

// Ensure that closing the only buffer in a window leaves the window open and empty.
// If a window has only one buffer and it is closed, the window should be closed (when multiple windows exist).
func TestCloseBuffer_CloseWindowWhenLastBuffer_MultiWindow(t *testing.T) {
	te := NewTestEditor(80, 24)

	buf1 := buffer.NewBuffer("hello world", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("second", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	te.buffers[buf1.FilePath] = buf1
	te.buffers[buf2.FilePath] = buf2

	// Two windows: W0 has buf1 only; W1 has buf2 only
	w0 := newWindow(buf1)
	w1 := newWindow(buf2)
	te.windows = []*Window{w0, w1}
	te.activeWindow = 0
	te.SetCurrentBuffer(buf1)

	// Close the only buffer in W0 -> window 0 should be closed, leaving a single window (W1)
	if err := te.ExecuteCommand("q"); err != nil {
		t.Fatalf("close buffer failed: %v", err)
	}

	if len(te.windows) != 1 {
		t.Fatalf("expected windows to shrink to 1 after closing last buffer in W0, got %d", len(te.windows))
	}
	// Remaining window should be the one that had buf2
	if te.windows[0].Buffer != buf2 || len(te.windows[0].Buffers) != 1 {
		t.Fatalf("expected remaining window to contain buf2")
	}
	// buf1 should be removed globally
	if _, ok := te.buffers["file1.txt"]; ok {
		t.Fatalf("expected buf1 to be removed from global registry")
	}
}

// Closing one of multiple buffers in the same window removes it but leaves the other active.
func TestCloseBuffer_MultipleBuffersInWindow(t *testing.T) {
	te := NewTestEditor(80, 24)

	buf1 := buffer.NewBuffer("buf1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("buf2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	te.buffers[buf1.FilePath] = buf1
	te.buffers[buf2.FilePath] = buf2

	win := newWindow(buf1)
	win.addBufferToWindow(buf2) // buf2 becomes active
	te.windows = []*Window{win}
	te.activeWindow = 0
	te.SetCurrentBuffer(buf2)

	// Close the active buffer (buf2)
	if err := te.ExecuteCommand("q"); err != nil {
		t.Fatalf("close buffer failed: %v", err)
	}

	// Window remains open; buf1 should be active and only buffer in the list
	if len(te.windows) != 1 {
		t.Fatalf("expected 1 window to remain, got %d", len(te.windows))
	}
	if te.windows[0].Buffer != buf1 {
		t.Fatalf("expected buf1 to be active after closing buf2")
	}
	if len(te.windows[0].Buffers) != 1 || te.windows[0].Buffers[0] != buf1 {
		t.Fatalf("expected only buf1 to remain in window buffers")
	}
	// buf2 should be removed globally since it is not shown in any window
	if _, ok := te.buffers["file2.txt"]; ok {
		t.Fatalf("expected buf2 to be removed from global registry")
	}
}

// Closing a terminal buffer should remove it from windows and clean up terminal session
// when the buffer is no longer shown anywhere.
func TestCloseBuffer_TerminalBufferCleanup(t *testing.T) {
	te := NewTestEditor(80, 24)

	term := buffer.NewBuffer("", "*Terminal*", "", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	other := buffer.NewBuffer("", "file.txt", "file.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	te.buffers[term.FilePath] = term
	te.buffers[other.FilePath] = other
	w0 := newWindow(term)
	w1 := newWindow(other)
	te.windows = []*Window{w0, w1}
	te.activeWindow = 0
	te.SetCurrentBuffer(term)

	// Simulate an active terminal session tracked by the editor.
	te.terminalSessions[term.FilePath] = &TerminalSessionState{SessionID: "sid", WorkingDir: "."}

	if err := te.ExecuteCommand("q"); err != nil {
		t.Fatalf("close buffer failed: %v", err)
	}

	// Window with terminal should be closed (we started with 2 windows)
	if len(te.windows) != 1 {
		t.Fatalf("expected windows to shrink to 1 after closing last terminal buffer window, got %d", len(te.windows))
	}
	// Terminal session should be removed if buffer no longer shown
	if _, ok := te.terminalSessions[term.FilePath]; ok {
		t.Fatalf("expected terminal session to be removed for closed buffer")
	}
	// Buffer removed globally
	if _, ok := te.buffers[term.FilePath]; ok {
		t.Fatalf("expected terminal buffer to be removed from global registry")
	}
}
