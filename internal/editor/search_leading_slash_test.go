package editor

import (
	"testing"
	"veda/internal/events"
)

// TestSearchDeleteLeadingSlash reproduces the bug where deleting the leading
// slash in the search prompt is allowed after moving the cursor to the start.
func TestSearchDeleteLeadingSlash(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello\nworld")

	if err := te.ExecuteCommand("enter-search-mode"); err != nil {
		t.Fatalf("failed to enter search mode: %v", err)
	}

	for _, r := range []rune("query") {
		te.handleEvent(events.NewKeyEvent(events.KeyRune, r, events.ModNone))
	}

	for i := 0; i < len("query")+1; i++ {
		te.handleEvent(events.NewKeyEvent(events.KeyLeft, 0, events.ModNone))
	}

	te.handleEvent(events.NewKeyEvent(events.KeyDelete, 0, events.ModNone))

	if got := te.searchLine.GetText(); got != "/query" {
		t.Fatalf("expected search line to keep leading slash; got %q", got)
	}

	te.handleEvent(events.NewKeyEvent(events.KeyEnd, 0, events.ModNone))
	te.handleEvent(events.NewKeyEvent(events.KeyBackspace, 0, events.ModAlt))

	if got := te.searchLine.GetText(); got != "/" {
		t.Fatalf("expected alt+backspace to keep leading slash and remove word; got %q", got)
	}

	for _, r := range []rune("again") {
		te.handleEvent(events.NewKeyEvent(events.KeyRune, r, events.ModNone))
	}

	te.handleEvent(events.NewKeyEvent(events.KeyHome, 0, events.ModNone))
	te.handleEvent(events.NewKeyEvent(events.KeyRight, 0, events.ModNone))
	te.handleEvent(events.NewKeyEvent(events.KeyBackspace, 0, events.ModAlt))

	if got := te.searchLine.GetText(); got != "/again" {
		t.Fatalf("expected alt+backspace near prefix to keep slash; got %q", got)
	}
}
