package editor

import (
	"fmt"
	"testing"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestTerminalScrollInNormalMode(t *testing.T) {
	te := NewTestEditor(80, 12)

	cols, rows := 80, 8
	term := NewTerminalEmulator(cols, rows)
	totalLines := 40
	for i := 0; i < totalLines; i++ {
		line := fmt.Sprintf("line%02d\r\n", i)
		term.Write([]byte(line))
	}

	buf := setupTerminalBufferForTest(te, cols, rows, term.Render(), term)
	te.mode = commands.TerminalMode

	ts := te.terminalSessions[buf.FilePath]
	if ts == nil || ts.term == nil {
		t.Fatalf("missing terminal session state")
	}

	curX, curY := ts.term.Cursor()
	bannerOffset := 0
	if ts.Banner != "" {
		bannerOffset = 1
	}

	scrollbackLines := len(ts.term.scrollback)
	bufRow := scrollbackLines + curY + bannerOffset
	if bufRow < 0 {
		bufRow = 0
	}
	if bufRow >= buf.LineCount {
		bufRow = buf.LineCount - 1
	}

	visible := te.getVisibleHeight()
	if visible < 1 {
		visible = 1
	}
	termGridStart := scrollbackLines + bannerOffset
	top := termGridStart
	if bufRow >= termGridStart+visible {
		top = bufRow - (visible - 1)
	}
	if top < 0 {
		top = 0
	}
	if top >= buf.LineCount {
		top = buf.LineCount - 1
	}

	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, top)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, bufRow)
	line := buf.CursorIterator.Line()
	buf.CursorX = te.bytePositionFromVisualColumn(line, curX)

	if win := te.windows[te.activeWindow]; win != nil {
		win.ViewLine = top
		win.CursorLine = bufRow
		win.CursorX = buf.CursorX
	}

	te.render()

	if top == 0 {
		t.Fatalf("expected initial view to start after scrollback, got top=%d", top)
	}

	initialTop := buf.ViewIterator.LineNumber()

	te.PressKey(events.KeyEscape, 0)
	if te.mode != commands.NormalMode {
		t.Fatalf("expected to leave terminal mode after Escape, got mode=%v", te.mode)
	}

	te.render()

	te.PressKey(events.KeyUp, 0)
	te.scroll()
	te.render()

	finalTop := buf.ViewIterator.LineNumber()
	if finalTop >= initialTop {
		t.Fatalf("expected view to scroll upward; initial top=%d final=%d", initialTop, finalTop)
	}
}
