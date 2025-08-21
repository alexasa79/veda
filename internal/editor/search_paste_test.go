package editor

import (
	"testing"
	"veda/internal/events"
)

// TestPasteIntoSearchDoesNotModifyBuffer reproduces the bug where pasted text while in
// search mode is inserted into the buffer instead of the search box. This test asserts
// the desired behavior (paste should go into the search line), which currently fails.
func TestPasteIntoSearchDoesNotModifyBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello world\nsecond line")

	// Place cursor at a known position to detect unintended buffer edits
	te.SetCursorPosition(0, 0)
	original := te.GetBufferContent()

	// Enter search mode ('/')
	if err := te.ExecuteCommand("enter-search-mode"); err != nil {
		t.Fatalf("failed to enter search mode: %v", err)
	}

	// Simulate bracketed paste of the word "world" while in search mode
	te.handleEvent(events.NewPasteEvent(true))
	for _, r := range []rune("world") {
		te.handleEvent(events.NewKeyEvent(events.KeyRune, r, events.ModNone))
	}
	te.handleEvent(events.NewPasteEvent(false))

	// Desired behavior: buffer untouched; search line contains "/world"
	if te.GetBufferContent() != original {
		t.Fatalf("buffer was modified by paste during search mode; got %q, want %q", te.GetBufferContent(), original)
	}
	if te.searchLine.GetText() != "/world" {
		t.Fatalf("search line did not receive pasted text; got %q, want %q", te.searchLine.GetText(), "/world")
	}
}
