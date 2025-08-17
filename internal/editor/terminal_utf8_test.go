package editor

import (
    "testing"
    "veda/internal/buffer"
    "veda/internal/commands"
    "veda/internal/ui"
)

// Builds an internal terminal buffer attached to an emulator and installs it in the editor
func setupTerminalBufferForTest(te *TestEditor, cols, rows int, content string, term *TerminalEmulator) *buffer.Buffer {
    // Create internal buffer and attach to editor
    b := buffer.NewBuffer("", "*terminal*", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
    b.IsInternal = true
    b.IsReadOnly = true

    // Add to editor buffers and select it
    te.buffers[b.FilePath] = b
    te.buffer = b
    te.bufferOrder = []string{b.FilePath}

    // Attach terminal session state with emulator
    if te.terminalSessions == nil {
        te.terminalSessions = make(map[string]*TerminalSessionState)
    }
    te.terminalSessions[b.FilePath] = &TerminalSessionState{SessionID: "test", Cols: cols, Rows: rows, term: term}

    // Replace buffer content with emulator render snapshot
    wasRO := b.IsReadOnly
    b.IsReadOnly = false
    if b.Length() > 0 { b.Delete(0, b.Length()) }
    b.Insert(0, content)
    b.IsReadOnly = wasRO

    // Reset iterators and cursor to top-left
    b.ViewIterator = buffer.NewBufferIteratorAtLine(b, 0)
    b.CursorIterator = buffer.NewBufferIteratorAtLine(b, 0)
    b.CursorX = 0

    return b
}

func TestTerminalUTF8_NoMojibake(t *testing.T) {
    te := NewTestEditor(80, 10)

    // Create emulator and write some UTF-8 including emoji and return symbol
    term := NewTerminalEmulator(80, 10)
    term.Write([]byte("🐟 hi ⏎ — € ✓\n"))
    // Also write a colored rune to ensure styles applied per cell do not corrupt UTF-8
    term.Write([]byte("\x1b[34m✓\x1b[0m\n"))

    // Install terminal buffer
    setupTerminalBufferForTest(te, 80, 10, term.Render(), term)
    te.mode = commands.TerminalMode

    // Render
    te.drawEditor()

    // Verify that content area begins with our UTF-8 sequence (no mojibake)
    startX := te.buffer.LineNumberWidth
    want := []rune("🐟 hi ⏎ — € ✓")
    gotRunes := make([]rune, 0, len(want))
    for i := 0; i < len(want); i++ {
        r, _ := te.mockScreen.GetCell(startX+i, 0)
        gotRunes = append(gotRunes, r)
    }
    if string(gotRunes) != string(want) {
        // Also dump the whole visible line for debugging
        line := te.mockScreen.GetLine(0)
        t.Fatalf("mojibake detected; got %q, want %q (full line: %q)", string(gotRunes), string(want), line)
    }
}

func TestTerminalColorsPersistInNormalMode(t *testing.T) {
    te := NewTestEditor(40, 6)

    // Create emulator and write colored text: blue X then reset
    term := NewTerminalEmulator(40, 6)
    term.Write([]byte("\x1b[34mX\x1b[0m\n"))

    setupTerminalBufferForTest(te, 40, 6, term.Render(), term)

    // Draw in terminal mode first
    te.mode = commands.TerminalMode
    te.drawEditor()

    // Find the style at column 0, row 0
    r0, style0 := te.mockScreen.GetCell(te.buffer.LineNumberWidth+0, 0)
    if r0 != 'X' {
        t.Fatalf("expected 'X' at content start, got %q", r0)
    }
    fg0, _, _ := style0.Decompose()
    if fg0 == ui.ColorDefault {
        t.Fatalf("expected colored 'X' in terminal mode, got default fg")
    }

    // Switch to normal mode and draw again; terminal/internal buffers should still use terminal styling
    te.mode = commands.NormalMode
    te.drawEditor()

    r1, style1 := te.mockScreen.GetCell(te.buffer.LineNumberWidth+0, 0)
    if r1 != 'X' {
        t.Fatalf("after normal mode, expected 'X', got %q", r1)
    }
    fg1, _, _ := style1.Decompose()
    if fg1 == ui.ColorDefault {
        t.Fatalf("colors disappeared in normal mode; want colored fg, got default")
    }
}
