package editor

import (
	"strings"
	"testing"
	"veda/internal/events"
)

func TestTerminalOutputResizesEmulatorWhenGutterWidthChanges(t *testing.T) {
	te := NewTestEditor(20, 12)
	buf := setupTerminalBufferForTest(te, 1, 1, "", NewTerminalEmulator(1, 1))

	initialCols, rows := te.getActiveWindowSizeForTerminal()
	ts := te.terminalSessions[buf.FilePath]
	ts.SessionID = ""
	ts.Cols = initialCols
	ts.Rows = rows
	ts.term = NewTerminalEmulator(initialCols, rows)
	initialGutter := buf.LineNumberWidth

	// A terminal snapshot with ten lines grows the line-number gutter from
	// one digit to two. The PTY-facing emulator must lose the same column that
	// the visible terminal content area loses.
	te.handleEvent(events.NewCustomEvent("terminal-output", map[string]interface{}{
		"buffer": buf.FilePath,
		"data":   strings.Repeat("x\n", 10),
	}))

	if buf.LineNumberWidth != initialGutter+1 {
		t.Fatalf("gutter width = %d, want %d", buf.LineNumberWidth, initialGutter+1)
	}
	wantCols, wantRows := te.getActiveWindowSizeForTerminal()
	gotCols, gotRows := ts.term.Dimensions()
	if gotCols != wantCols || gotRows != wantRows {
		t.Fatalf("emulator size = %dx%d, visible terminal area = %dx%d", gotCols, gotRows, wantCols, wantRows)
	}
}
