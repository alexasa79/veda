package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestSearchResultTextObject(t *testing.T) {
	t.Run("delete search result basic", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world hello universe hello")
		
		// Search for "hello"
		te.searchPattern = "hello"
		te.updateSearchMatches()
		
		// Should find 3 matches at positions 0, 12, and 27
		if len(te.searchMatches) != 3 {
			t.Fatalf("Expected 3 matches, got %d", len(te.searchMatches))
		}
		
		// Position cursor at first "hello" (position 0)
		AssertCursorAt(t, te, 0, 0)
		
		// Execute "ds" to delete the search result
		te.TypeKeys("ds")
		
		// Should have deleted first "hello"
		AssertBufferEquals(t, te, " world hello universe hello")
		AssertCursorAt(t, te, 0, 0) // Cursor should be at start after deletion
	})
	
	t.Run("delete search result when cursor on match", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "foo hello bar hello baz")
		
		// Search for "hello"
		te.searchPattern = "hello"
		te.updateSearchMatches()
		
		// Move cursor to middle of first "hello" (position 5)
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 5, 0)
		
		// Execute "ds" to delete the search result
		te.TypeKeys("ds")
		
		// Should have deleted first "hello"
		AssertBufferEquals(t, te, "foo  bar hello baz")
		AssertCursorAt(t, te, 4, 0) // Cursor should be after "foo "
	})
	
	t.Run("delete search result finds closest match", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world hello universe")
		
		// Search for "hello"
		te.searchPattern = "hello"
		te.updateSearchMatches()
		
		// Move cursor to position 8 (between the two "hello"s)
		for i := 0; i < 8; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 8, 0)
		
		// Execute "ds" to delete the search result
		te.TypeKeys("ds")
		
		// Should have deleted the second "hello" (closest after cursor)
		AssertBufferEquals(t, te, "hello world  universe")
		// Cursor should be at start of where the deletion occurred
		// The second "hello" started at position 12, so cursor should be there
		AssertCursorAt(t, te, 12, 0)
	})
	
	t.Run("change search result", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world hello universe")
		
		// Search for "hello"
		te.searchPattern = "hello"
		te.updateSearchMatches()
		
		// Position cursor at first "hello"
		AssertCursorAt(t, te, 0, 0)
		
		// Execute "cs" to change the search result
		te.TypeKeys("cs")
		
		// Should enter insert mode after deleting "hello"
		if te.mode != commands.InsertMode {
			t.Fatalf("Expected insert mode, got %v", te.mode)
		}
		AssertBufferEquals(t, te, " world hello universe")
		
		// Type replacement text
		te.InsertChar('h')
		te.InsertChar('i')
		te.ExecuteCommand("exit-insert-mode")
		
		// Should have replaced "hello" with "hi"
		AssertBufferEquals(t, te, "hi world hello universe")
	})
	
	t.Run("no search pattern active", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world hello universe")
		
		// No search pattern set
		te.searchPattern = ""
		te.searchMatches = []int{}
		
		// Execute "ds" - should do nothing
		te.TypeKeys("ds")
		
		// Buffer should be unchanged
		AssertBufferEquals(t, te, "hello world hello universe")
	})
	
	t.Run("search result across lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello\nworld\nhello\nuniverse")
		
		// Search for "hello"
		te.searchPattern = "hello"
		te.updateSearchMatches()
		
		// Should find 2 matches
		if len(te.searchMatches) != 2 {
			t.Fatalf("Expected 2 matches, got %d", len(te.searchMatches))
		}
		
		// Move to line 2 (second "hello")
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 2)
		
		// Execute "ds" to delete the search result
		te.TypeKeys("ds")
		
		// Should have deleted second "hello" - since it's the only content on the line,
		// the entire line is removed
		AssertBufferEquals(t, te, "hello\nworld\nuniverse")
	})
	
	t.Run("search result with multiline match", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world hello test hello end")
		
		// Search for "hello"
		te.searchPattern = "hello"
		te.updateSearchMatches()
		
		// Move cursor to middle "hello" (around position 12)
		for i := 0; i < 14; i++ {
			te.ExecuteCommand("right")
		}
		
		// Execute "ds" to delete the search result
		te.TypeKeys("ds")
		
		// Should have deleted middle "hello"
		AssertBufferEquals(t, te, "hello world  test hello end")
	})
	
	t.Run("change search result with modifier", func(t *testing.T) {
		// Test to ensure our search text object works with change operations
		te := NewTestEditorWithContent(80, 24, "test hello world")
		
		// Search for "hello"
		te.searchPattern = "hello"
		te.updateSearchMatches()
		
		// Move cursor to "hello"
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		
		// Execute "cs" to change search result
		te.TypeKeys("cs")
		
		// Should be in insert mode with "hello" deleted
		AssertBufferEquals(t, te, "test  world")
		if te.mode != commands.InsertMode {
			t.Fatalf("Expected insert mode after cs, got %v", te.mode)
		}
		
		// Insert replacement
		te.InsertChar('h')
		te.InsertChar('i')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "test hi world")
	})
}