package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestSearchIndicator(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		searchPattern   string
		expectedMatches int
		description     string
	}{
		{
			name:            "simple search",
			content:         "Hello world\nHello again\nGoodbye world",
			searchPattern:   "Hello",
			expectedMatches: 2,
			description:     "Should find 2 occurrences of Hello",
		},
		{
			name:            "multiple matches on same line",
			content:         "test test test\nno match here\ntest again",
			searchPattern:   "test",
			expectedMatches: 4,
			description:     "Should find 4 occurrences of test",
		},
		{
			name:            "no matches",
			content:         "Hello world\nGoodbye world",
			searchPattern:   "xyz",
			expectedMatches: 0,
			description:     "Should find 0 occurrences of xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor
			te := NewTestEditorWithContent(80, 24, tt.content)

			// Enter search mode and search for pattern
			te.enterSearchMode()
			te.searchLine.SetText("/" + tt.searchPattern)
			te.executeSearch()

			// Check number of matches
			if len(te.searchMatches) != tt.expectedMatches {
				t.Errorf("%s: Expected %d matches, got %d",
					tt.description, tt.expectedMatches, len(te.searchMatches))
			}

			// If matches exist, verify currentMatch is set
			if tt.expectedMatches > 0 && te.currentMatch != 0 {
				t.Errorf("%s: Expected currentMatch to be 0 after search, got %d",
					tt.description, te.currentMatch)
			}
		})
	}
}

func TestSearchNavigation(t *testing.T) {
	// Create test editor with multiple matches
	te := NewTestEditorWithContent(80, 24, "test one\ntest two\ntest three")

	// Search for "test"
	te.enterSearchMode()
	te.searchLine.SetText("/test")
	te.executeSearch()

	// Should have 3 matches
	if len(te.searchMatches) != 3 {
		t.Fatalf("Expected 3 matches, got %d", len(te.searchMatches))
	}

	// First match should be selected
	if te.currentMatch != 0 {
		t.Errorf("Expected currentMatch = 0, got %d", te.currentMatch)
	}

	// Navigate to next match
	te.findNext()
	if te.currentMatch != 1 {
		t.Errorf("After findNext, expected currentMatch = 1, got %d", te.currentMatch)
	}

	// Navigate to next match again
	te.findNext()
	if te.currentMatch != 2 {
		t.Errorf("After second findNext, expected currentMatch = 2, got %d", te.currentMatch)
	}

	// Navigate to next match (should wrap to first)
	te.findNext()
	if te.currentMatch != 0 {
		t.Errorf("After wrapping findNext, expected currentMatch = 0, got %d", te.currentMatch)
	}

	// Navigate to previous match (should wrap to last)
	te.findPrevious()
	if te.currentMatch != 2 {
		t.Errorf("After findPrevious from first, expected currentMatch = 2, got %d", te.currentMatch)
	}
}

func TestClearSearch(t *testing.T) {
	// Create test editor
	te := NewTestEditorWithContent(80, 24, "test test test")

	// Search for "test"
	te.enterSearchMode()
	te.searchLine.SetText("/test")
	te.executeSearch()

	// Verify search is active
	if te.searchPattern != "test" {
		t.Errorf("Expected searchPattern = 'test', got '%s'", te.searchPattern)
	}
	if len(te.searchMatches) == 0 {
		t.Errorf("Expected matches, got none")
	}

	// Clear search
	te.clearSearch()

	// Verify search is cleared
	if te.searchPattern != "" {
		t.Errorf("Expected searchPattern to be empty, got '%s'", te.searchPattern)
	}
	if len(te.searchMatches) != 0 {
		t.Errorf("Expected searchMatches to be empty, got %d matches", len(te.searchMatches))
	}
	if te.currentMatch != -1 {
		t.Errorf("Expected currentMatch = -1, got %d", te.currentMatch)
	}
}

func TestEscapeKeyClearsSearch(t *testing.T) {
	// Create test editor
	te := NewTestEditorWithContent(80, 24, "hello world hello")

	// Search for "hello"
	te.enterSearchMode()
	te.searchLine.SetText("/hello")
	te.executeSearch()
	te.exitSearchMode() // Return to normal mode

	// Verify search is active
	if te.searchPattern != "hello" {
		t.Errorf("Expected searchPattern = 'hello', got '%s'", te.searchPattern)
	}

	// Verify we're in normal mode
	if te.mode != commands.NormalMode {
		t.Errorf("Expected to be in normal mode, got %v", te.mode)
	}

	// Execute escape key command
	err := te.ExecuteCommand("clear-search")
	if err != nil {
		t.Fatalf("Failed to execute clear-search command: %v", err)
	}

	// Verify search is cleared
	if te.searchPattern != "" {
		t.Errorf("Expected searchPattern to be empty after escape, got '%s'", te.searchPattern)
	}
}

func TestSearchMatchPositionsAfterTextDeletion(t *testing.T) {
	// Test that verifies search match positions are automatically updated when buffer content changes.
	// This test uses a realistic multi-line scenario with line deletions to ensure search navigation
	// works correctly after text modifications.

	content := `hello world this is line 1
some random text here
another line with words
middle content to delete
more stuff in the middle
yet another middle line
final text before target
goodbye and farewell
hello world this is last line
end of document`

	// Create editor with multi-line content containing matches on first and last lines
	te := NewTestEditorWithContent(80, 24, content)

	// Search for "hello" - should find 2 matches: line 1 and line 9
	te.enterSearchMode()
	te.searchLine.SetText("/hello")
	te.executeSearch()
	te.exitSearchMode()

	// Verify we found 2 matches
	if len(te.searchMatches) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(te.searchMatches))
	}

	// Record original match positions and line numbers
	firstMatchPos := te.searchMatches[0]
	secondMatchPos := te.searchMatches[1]

	t.Logf("Original first match at position: %d", firstMatchPos)
	t.Logf("Original second match at position: %d", secondMatchPos)

	// Verify we're at the first match (should be on line 0)
	if te.currentMatch != 0 {
		t.Errorf("Expected to be at first match (0), got %d", te.currentMatch)
	}
	AssertCursorAt(t, te, 0, 0) // Should be at start of first "hello"

	// Navigate to the second match to establish context
	err := te.ExecuteCommand("find-next")
	if err != nil {
		t.Fatalf("find-next command failed: %v", err)
	}

	// Should now be at second match (line 8, position 0 in that line)
	cursorCol, cursorRow := te.GetCursorPosition()
	t.Logf("After find-next, cursor at: (%d, %d)", cursorCol, cursorRow)

	if te.currentMatch != 1 {
		t.Errorf("Expected to be at second match (1), got %d", te.currentMatch)
	}

	// Now delete several lines in the middle (lines 3-6: "middle content..." through "yet another middle line")
	// Move cursor to start of line 3
	te.SetCursorPosition(0, 3)
	AssertCursorAt(t, te, 0, 3)

	// Use dd command to delete 4 lines
	for i := 0; i < 4; i++ {
		// Type 'dd' to delete current line
		te.TypeKeys("dd")
		// After each deletion, we should still be at line 3 (or wherever the cursor lands)
		// since deleting a line moves remaining lines up
	}

	t.Logf("Buffer after deletion:\n%s", te.GetBufferContent())

	// Check what happens to search matches after deletion
	t.Logf("Search matches after deletion: %v", te.searchMatches)
	t.Logf("Current match index: %d", te.currentMatch)

	// FIXED: Search match positions ARE now updated after line deletions!
	// The fix automatically recalculates match positions when buffer content changes.

	// Verify we still have 2 matches
	if len(te.searchMatches) != 2 {
		t.Errorf("Expected 2 matches after deletion, got %d", len(te.searchMatches))
	}

	// Check if the match positions make sense with the new buffer content
	newContent := te.GetBufferContent()
	if len(te.searchMatches) >= 2 {
		firstMatchPos := te.searchMatches[0]
		secondMatchPos := te.searchMatches[1]

		// First match should still be at position 0
		if firstMatchPos != 0 {
			t.Errorf("First match position changed unexpectedly. Expected 0, got %d", firstMatchPos)
		}

		// The second match position should now be updated correctly
		t.Logf("Second match is at position %d in buffer of length %d", secondMatchPos, len(newContent))

		// Verify the position is within buffer bounds
		if secondMatchPos >= len(newContent) {
			t.Errorf("Match position error: Second match at position %d is beyond buffer length %d",
				secondMatchPos, len(newContent))
		}

		// Verify that the match position actually points to the search pattern
		if secondMatchPos+len("hello") <= len(newContent) {
			actualText := newContent[secondMatchPos : secondMatchPos+len("hello")]
			if actualText != "hello" {
				t.Errorf("Match position error: Expected 'hello' at position %d, got '%s'",
					secondMatchPos, actualText)
			}
		}
	}

	// The key test: what happens when we press 'n' to go to next match?
	// After deletion, the current match was reset to 0, so 'n' should go to match 1
	beforeNextCol, beforeNextRow := te.GetCursorPosition()
	t.Logf("Cursor before 'n' command: (%d, %d)", beforeNextCol, beforeNextRow)

	err = te.ExecuteCommand("find-next")
	if err != nil {
		t.Errorf("find-next command failed after deletion: %v", err)
	}

	afterNextCol, afterNextRow := te.GetCursorPosition()
	t.Logf("Cursor after 'n' command: (%d, %d)", afterNextCol, afterNextRow)
	t.Logf("Current match after 'n' command: %d", te.currentMatch)

	// FIXED BEHAVIOR: Should jump to the actual second "hello" which is now on line 4
	// The second "hello" should now be on line 4 (0-indexed), at the beginning of the line
	expectedRow := 4
	expectedCol := 0

	if afterNextRow != expectedRow || afterNextCol != expectedCol {
		t.Errorf("Navigation works correctly: Expected cursor at (%d, %d) for second match, got (%d, %d)",
			expectedCol, expectedRow, afterNextCol, afterNextRow)
	}

	// Verify we're now on the second match
	if te.currentMatch != 1 {
		t.Errorf("Expected to be on match 1 after find-next, got %d", te.currentMatch)
	}
}
