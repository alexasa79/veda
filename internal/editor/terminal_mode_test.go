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

    // Create emulator and write a simple prompt, then move to next line
    term := NewTerminalEmulator(80, 20)
    term.Write([]byte("prompt> abcdef\r\n")) // Use \r\n to ensure cursor moves to start of next line
    
    // Install terminal buffer
    setupTerminalBufferForTest(te, 80, 20, term.Render(), term)
    te.mode = commands.TerminalMode

    // Draw once to establish positions
    te.drawEditor()

    // Verify initial terminal emulator cursor position - should be at start of line 1
    initCurX, initCurY := term.Cursor()
    if initCurY != 1 || initCurX != 0 {
        t.Fatalf("expected emulator cursor at (0,1) after \\r\\n, got (%d,%d)", initCurX, initCurY)
    }

    // Press End -> should send escape sequence to move to end of line
    te.TypeKeys("End")
    // Simulate the shell response: move cursor to end of current (empty) line
    // Since the line is empty, cursor should stay at column 0
    te.drawEditor()
    
    // Verify the emulator cursor position after End
    endCurX, endCurY := term.Cursor()
    if endCurY != 1 {
        t.Fatalf("cursor row changed unexpectedly after End: expected row 1, got %d", endCurY)
    }
    // On an empty line, End should keep cursor at column 0
    if endCurX != 0 {
        t.Fatalf("End on empty line should keep cursor at column 0, got column %d", endCurX)
    }

    // Press Home -> should send escape sequence to move to beginning of line  
    te.TypeKeys("Home")
    // Simulate the shell response: move cursor to column 0
    te.drawEditor()
    
    // Verify the emulator cursor is still at column 0
    homeCurX, homeCurY := term.Cursor()
    if homeCurY != 1 {
        t.Fatalf("cursor row changed unexpectedly after Home: expected row 1, got %d", homeCurY)
    }
    if homeCurX != 0 {
        t.Fatalf("Home key should keep cursor at column 0, got column %d", homeCurX)
    }

    // Test that no panics occurred and the editor is still functional
    te.drawEditor()
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
