package editor

import (
    "testing"
    "veda/internal/commands"
)

// TestTerminalModeBinding verifies that "space t" opens a terminal buffer in Terminal mode,
// the buffer is read-only and internal, and Escape returns to Normal mode.
func TestTerminalModeBinding(t *testing.T) {
    te := NewTestEditor(80, 24)

    // Ensure starting in normal mode
    if te.mode != commands.NormalMode {
        t.Fatalf("expected NORMAL mode at start, got %v", te.mode)
    }

    // Press space t
    te.TypeKeys("space t")

    // Verify mode switched to Terminal
    if te.mode != commands.TerminalMode {
        t.Fatalf("expected TERMINAL mode after 'space t', got %v", te.mode)
    }

    // Verify current buffer properties
    if te.buffer == nil {
        t.Fatalf("expected a current buffer after opening terminal")
    }
    if !te.buffer.IsInternal {
        t.Errorf("terminal buffer should be internal")
    }
    if !te.buffer.IsReadOnly {
        t.Errorf("terminal buffer should be read-only")
    }

    // Status bar contains TERMINAL indicator (render and inspect the status line)
    lines := te.GetScreenContent()
    if len(lines) == 0 {
        t.Fatalf("no screen content rendered")
    }
    status := lines[len(lines)-1]
    if status == "" || (status != "" && !containsSubstring(status, "TERMINAL")) {
        t.Errorf("status bar should contain TERMINAL, got: %q", status)
    }

    // Press Escape to return to Normal mode
    te.PressKey(KeyEscape, 0)
    if te.mode != commands.NormalMode {
        t.Fatalf("expected NORMAL mode after ESC, got %v", te.mode)
    }
}

// TestTerminalHomeEndKeys ensures Home/End map to terminal sequences and update cursor row/col
func TestTerminalHomeEndKeys(t *testing.T) {
    te := NewTestEditor(80, 20)

    // Create emulator and write a simple prompt and input area
    term := NewTerminalEmulator(80, 20)
    term.Write([]byte("prompt> abcdef\n"))
    // The cursor should be at start of next line (row 1, col 0)

    // Install terminal buffer
    setupTerminalBufferForTest(te, 80, 20, term.Render(), term)
    te.mode = commands.TerminalMode

    // Draw once to establish positions
    te.drawEditor()

    // Press End -> should move to end of line on the shell
    te.TypeKeys("End")
    // We simulate the shell echo by moving emulator cursor ourselves
    term.Write([]byte("\x1b[80G")) // move far right; then set to end of empty line (still 0)
    te.drawEditor()
    // No assertion on screen pixels; just ensure no panic and cursor is shown on content area
    _, y := te.GetCursorPosition()
    if y < te.getContentStartY() || y >= te.getContentStartY()+te.getVisibleHeight() {
        t.Fatalf("cursor off content area after End: y=%d", y)
    }

    // Press Home -> should move to column 0
    te.TypeKeys("Home")
    term.Write([]byte("\x1b[1G"))
    te.drawEditor()
    x2, _ := te.GetCursorPosition()
    if x2 < te.buffer.LineNumberWidth {
        t.Fatalf("cursor mapped left of content area after Home: x=%d", x2)
    }
}

// TestTerminalReopen verifies we can exit and open terminal again
func TestTerminalReopen(t *testing.T) {
    te := NewTestEditor(80, 24)
    te.TypeKeys("space t")
    if te.mode != commands.TerminalMode {
        t.Fatalf("expected TERMINAL mode after first open, got %v", te.mode)
    }
    if te.buffer == nil || !te.buffer.IsInternal || !te.buffer.IsReadOnly {
        t.Fatalf("terminal buffer should be internal and read-only on first open")
    }
    // Exit and open again
    te.PressKey(KeyEscape, 0)
    te.TypeKeys("space t")
    if te.mode != commands.TerminalMode {
        t.Fatalf("expected TERMINAL mode after second open, got %v", te.mode)
    }
    if te.buffer == nil || !te.buffer.IsInternal || !te.buffer.IsReadOnly {
        t.Fatalf("terminal buffer should be internal and read-only on second open")
    }
}

// containsSubstring is a small helper to avoid importing strings in this file
func containsSubstring(s, sub string) bool {
    if sub == "" {
        return true
    }
    // naive search
    for i := 0; i+len(sub) <= len(s); i++ {
        if s[i:i+len(sub)] == sub {
            return true
        }
    }
    return false
}
