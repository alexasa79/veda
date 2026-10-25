package editor

import (
	"testing"
	"time"
	"veda/internal/buffer"
	"veda/internal/events"
	"veda/internal/theme"
)

func TestCurrentWordHighlightAppearsAfterDelay(t *testing.T) {
	te := NewTestEditorWithContent(80, 10, "alpha beta alpha")
	te.cursorWordHighlightDelay = time.Millisecond

	// Place cursor on the first word so it becomes the tracked word.
	te.SetCursorPosition(1, 0)

	// Initial render schedules the deferred highlight.
	te.render()

	// Wait for the highlight timer to enqueue its render request.
	select {
	case ev := <-te.asyncEvents:
		if custom, ok := ev.(*events.CustomEvent); ok && custom.Type != "render" {
			t.Fatalf("unexpected custom event type %q", custom.Type)
		}
		te.handleEvent(ev)
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("did not receive highlight trigger event")
	}

	lineNumberWidth := te.CurrentBuffer().LineNumberWidth
	screenX := lineNumberWidth // first column of the word
	expectedStyle := te.config.Theme.GetStyle(te.config.Theme.CurrentWord)
	_, expectedBg, _ := expectedStyle.Decompose()

	_, cellStyle := te.mockScreen.GetCell(screenX, 0)
	_, actualBg, _ := cellStyle.Decompose()

	if actualBg != expectedBg {
		t.Fatalf("expected current word highlight at column %d, got background %v (expected %v)", screenX, actualBg, expectedBg)
	}
}

func TestCurrentWordHighlightWithMultiplePanes(t *testing.T) {
	for _, name := range []string{"left_window", "right_window", "diff"} {
		t.Run(name, func(t *testing.T) {
			te := NewTestEditorWithContent(100, 10, "alpha beta alpha")
			te.config.Theme = theme.NewVSCodeTheme()
			te.cursorWordHighlightDelay = time.Hour
			first := te.CurrentBuffer()
			second := buffer.NewBuffer("alpha beta alpha", "", "", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
			te.windows = []*Window{newWindow(first), newWindow(second)}
			if name == "right_window" {
				te.activeWindow = 1
			} else if name == "diff" {
				te.windows = []*Window{newWindow(second)}
				te.windows[0].View = &DiffBufferView{LeftBuffer: first, RightBuffer: second, ActivePane: 1}
			}
			te.recomputeWindowsLayout()
			te.SetCursorPosition(1, 0)
			te.setVirtualCursorsFor(te.CurrentBuffer(), []VirtualCursor{{X: 7, Y: 0}})
			te.render()
			t.Cleanup(func() { te.cursorWordHighlightTimer.Stop() })

			// Let the active cursor's delay expire without waiting for a timer.
			te.cursorWordLastMove = time.Now().Add(-2 * te.cursorWordHighlightDelay)
			te.render()

			win := te.windows[te.activeWindow]
			xOffset := win.X
			if win.View != nil {
				paneOffset, _ := win.View.CursorRegion(win.Width)
				xOffset += paneOffset
			}
			expectedBg := te.config.Theme.GetStyle(te.config.Theme.CurrentWord).Bg
			for _, col := range []int{0, 6, 11} {
				x := xOffset + win.Buffer.LineNumberWidth + col
				_, style := te.mockScreen.GetCell(x, win.Y)
				if style.Bg != expectedBg {
					t.Errorf("word at column %d has background %v, want %v", col, style.Bg, expectedBg)
				}
			}

			// Moving the active cursor must restart the delay.
			te.SetCursorPosition(7, 0)
			te.render()
			x := xOffset + win.Buffer.LineNumberWidth + 6
			_, style := te.mockScreen.GetCell(x, win.Y)
			if style.Bg == expectedBg {
				t.Fatal("new cursor word highlighted before the delay expired")
			}
		})
	}
}
