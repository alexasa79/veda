package editor

import (
	"fmt"
	"testing"
)

func TestNavigationHistory(t *testing.T) {
	// Create editor with enough lines
	content := ""
	for i := 0; i < 20; i++ {
		if i > 0 {
			content += "\n"
		}
		content += fmt.Sprintf("Line %d", i)
	}
	te := NewTestEditorWithContent(80, 24, content)
	
	// Set file path and position
	te.buffer.FilePath = "test.go"
	te.SetCursorPosition(5, 10)
	
	// Push to navigation history
	te.pushNavigationHistory()
	
	// Check that history was recorded
	if len(te.navHistory) != 1 {
		t.Errorf("Expected 1 entry in navigation history, got %d", len(te.navHistory))
	}
	
	entry := te.navHistory[0]
	if entry.filePath != "test.go" {
		t.Errorf("Expected filePath to be 'test.go', got %s", entry.filePath)
	}
	if entry.cursorX != 5 {
		t.Errorf("Expected cursorX to be 5, got %d", entry.cursorX)
	}
	if entry.cursorLineNumber != 10 {
		t.Errorf("Expected cursorLineNumber to be 10, got %d", entry.cursorLineNumber)
	}
}

func TestNavigationHistoryMaxSize(t *testing.T) {
	// Create editor with enough lines
	content := ""
	for i := 0; i < 60; i++ {
		if i > 0 {
			content += "\n"
		}
		content += fmt.Sprintf("Line %d", i)
	}
	te := NewTestEditorWithContent(80, 24, content)
	te.buffer.FilePath = "test.go"
	
	// Push more entries than maxNavHistory
	for i := 0; i < 60; i++ {
		te.SetCursorPosition(0, i)
		te.pushNavigationHistory()
	}
	
	// Should only keep the last 50 entries
	if len(te.navHistory) != 50 {
		t.Errorf("Expected navigation history to be capped at 50 entries, got %d", len(te.navHistory))
	}
	
	// First entry should be from position 10 (0-59, keeping last 50)
	if te.navHistory[0].cursorLineNumber != 10 {
		t.Errorf("Expected first entry cursorLineNumber to be 10, got %d", te.navHistory[0].cursorLineNumber)
	}
}

func TestGoBack(t *testing.T) {
	// Create a test editor with enough content for the cursor positions
	content := ""
	for i := 0; i < 35; i++ {
		content += fmt.Sprintf("Line %d with some text to make it long enough\n", i)
	}
	te := NewTestEditorWithContent(80, 24, content)
	
	// Set up initial state with a buffer
	te.buffer.FilePath = "test1.go"
	te.buffers["test1.go"] = te.buffer
	
	// Push a few positions to history
	te.SetCursorPosition(5, 10)
	te.pushNavigationHistory()
	
	te.SetCursorPosition(15, 20)
	te.pushNavigationHistory()
	
	// Move to a different position
	te.SetCursorPosition(25, 30)
	
	// Go back once
	err := te.goBack()
	if err != nil {
		t.Errorf("goBack returned error: %v", err)
	}
	
	// Should be at the second position
	x, y := te.GetCursorPosition()
	if x != 15 || y != 20 {
		t.Errorf("Expected cursor at (15, 20), got (%d, %d)", x, y)
	}
	
	// History should have one entry left
	if len(te.navHistory) != 1 {
		t.Errorf("Expected 1 entry in navigation history, got %d", len(te.navHistory))
	}
	
	// Go back again
	err = te.goBack()
	if err != nil {
		t.Errorf("goBack returned error: %v", err)
	}
	
	// Should be at the first position
	x, y = te.GetCursorPosition()
	if x != 5 || y != 10 {
		t.Errorf("Expected cursor at (5, 10), got (%d, %d)", x, y)
	}
	
	// History should be empty
	if len(te.navHistory) != 0 {
		t.Errorf("Expected navigation history to be empty, got %d entries", len(te.navHistory))
	}
}

func TestGoBackEmptyHistory(t *testing.T) {
	te := NewTestEditor(80, 24)
	
	// Going back with empty history should not error
	err := te.goBack()
	if err != nil {
		t.Errorf("goBack with empty history returned error: %v", err)
	}
}