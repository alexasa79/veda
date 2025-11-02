package editor

import (
	"testing"
	"time"
	"veda/internal/events"
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
