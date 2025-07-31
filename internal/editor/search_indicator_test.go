package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestSearchIndicator(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		searchPattern  string
		expectedMatches int
		description    string
	}{
		{
			name:           "simple search",
			content:        "Hello world\nHello again\nGoodbye world",
			searchPattern:  "Hello",
			expectedMatches: 2,
			description:    "Should find 2 occurrences of Hello",
		},
		{
			name:           "multiple matches on same line",
			content:        "test test test\nno match here\ntest again",
			searchPattern:  "test",
			expectedMatches: 4,
			description:    "Should find 4 occurrences of test",
		},
		{
			name:           "no matches",
			content:        "Hello world\nGoodbye world",
			searchPattern:  "xyz",
			expectedMatches: 0,
			description:    "Should find 0 occurrences of xyz",
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