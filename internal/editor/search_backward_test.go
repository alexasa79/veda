package editor

import "testing"

func TestBackwardSearchInitialAndNavigation(t *testing.T) {
	content := "hello world\nand then hello again\nfinally hello there"
	te := NewTestEditorWithContent(80, 24, content)

	// Move cursor to middle of second line (after second "hello")
	te.ExecuteCommand("down") // line 1
	for i := 0; i < 15; i++ { // roughly after "hello again"
		te.ExecuteCommand("right")
	}

	// Enter backward search and execute '?hello'
	if err := te.ExecuteCommand("enter-search-mode-backward"); err != nil {
		t.Fatalf("failed to enter backward search mode: %v", err)
	}
	te.searchLine.SetText("?hello")
	te.executeSearch()

	// Should jump to the previous match (the one on line 0 or 1)
	// In this content, it should land on line 1 start (index 1)
	if te.currentMatch != 1 {
		t.Errorf("expected to jump to previous match index 1, got %d", te.currentMatch)
	}

	// Now 'n' should move forward regardless of initial direction
	te.findNext()
	if te.currentMatch != 2 {
		t.Errorf("after findNext, expected index 2, got %d", te.currentMatch)
	}

	// And 'N' should move backward
	te.findPrevious()
	if te.currentMatch != 1 {
		t.Errorf("after findPrevious, expected index 1, got %d", te.currentMatch)
	}
}

func TestBackwardSearchWrapsToLast(t *testing.T) {
	content := "hello one\nhello two\nhello three"
	te := NewTestEditorWithContent(80, 24, content)

	// Cursor at very start; backward search should wrap to last occurrence
	if err := te.ExecuteCommand("enter-search-mode-backward"); err != nil {
		t.Fatalf("failed to enter backward search mode: %v", err)
	}
	te.searchLine.SetText("?hello")
	te.executeSearch()

	if te.currentMatch != 2 { // last match index
		t.Errorf("expected wrap to last match (index 2), got %d", te.currentMatch)
	}
}
