package editor

import (
	"fmt"
	"testing"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestTerminalReenterInsertRestoresCursorToPrompt(t *testing.T) {
	te := NewTestEditor(80, 12)

	cols, rows := 80, 6
	term := NewTerminalEmulator(cols, rows)
	for i := 0; i < 200; i++ {
		term.Write([]byte(fmt.Sprintf("line%04d\r\n", i)))
	}

	buf := setupTerminalBufferForTest(te, cols, rows, term.Render(), term)
	te.mode = commands.TerminalMode
	te.render()

	te.PressKey(events.KeyEscape, 0)
	if te.mode != commands.NormalMode {
		t.Fatalf("expected to leave terminal mode; mode=%v", te.mode)
	}

	// Simulate user scrolling to the top
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, 0)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 0)
	buf.CursorX = 0
	if win := te.windows[te.activeWindow]; win != nil {
		win.ViewLine = 0
		win.CursorLine = 0
		win.CursorX = 0
	}
	te.render()

	te.TypeKeys("i")

	if te.mode != commands.TerminalMode {
		t.Fatalf("expected to return to terminal mode after 'i'; mode=%v", te.mode)
	}

	ts := te.terminalSessions[buf.FilePath]
	if ts == nil {
		t.Fatalf("missing terminal session state")
	}
	bannerOffset := 0
	if ts.Banner != "" {
		bannerOffset = 1
	}
	expectedBufRow := len(ts.term.scrollback) + ts.SavedCurY + bannerOffset
	if expectedBufRow >= buf.LineCount {
		expectedBufRow = buf.LineCount - 1
	}

	if got := buf.CursorIterator.LineNumber(); got != expectedBufRow {
		t.Fatalf("expected cursor to return to prompt at line %d, got %d", expectedBufRow, got)
	}
}
