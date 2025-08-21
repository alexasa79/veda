package editor

import (
	"testing"
)

func TestSearchStartsFromCursorPosition(t *testing.T) {
	t.Run("search starts from cursor, not from beginning", func(t *testing.T) {
		// Content with multiple matches: "hello" appears at positions 0, 20, and 40
		content := "hello world\nand then hello again\nfinally hello there"
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to position 25 (middle of second line, after "hello again")
		te.ExecuteCommand("down") // Move to line 1
		for i := 0; i < 15; i++ { // Move to position after "hello again"
			te.ExecuteCommand("right")
		}

		// Search for "hello" - should find the third occurrence (not the first)
		te.enterSearchMode()
		te.searchLine.SetText("/hello")
		te.executeSearch()

		// Should be at the third match (currentMatch = 2), not the first (currentMatch = 0)
		if te.currentMatch != 2 {
			t.Errorf("Expected to jump to third match (index 2), got index %d", te.currentMatch)
		}

		// Cursor should be at beginning of "hello there" (line 2, char 8)
		AssertCursorAt(t, te, 8, 2)
	})

	t.Run("search wraps to beginning when no matches after cursor", func(t *testing.T) {
		content := "hello world\nsome text here\nno matches after this"
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to end of content
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		te.ExecuteCommand("end")

		// Search for "hello" - should wrap to beginning
		te.enterSearchMode()
		te.searchLine.SetText("/hello")
		te.executeSearch()

		// Should be at the first (and only) match
		if te.currentMatch != 0 {
			t.Errorf("Expected to wrap to first match (index 0), got index %d", te.currentMatch)
		}

		// Cursor should be at beginning of first line
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("search finds match at cursor position", func(t *testing.T) {
		content := "hello world\nhello again"
		te := NewTestEditorWithContent(80, 24, content)

		// Cursor is already at position 0 (start of first "hello")
		AssertCursorAt(t, te, 0, 0)

		// Search for "hello" - should find the match at cursor position
		te.enterSearchMode()
		te.searchLine.SetText("/hello")
		te.executeSearch()

		// Should stay at current position (first match)
		if te.currentMatch != 0 {
			t.Errorf("Expected to stay at current match (index 0), got index %d", te.currentMatch)
		}

		// Cursor should remain at beginning
		AssertCursorAt(t, te, 0, 0)
	})
}
