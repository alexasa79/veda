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
