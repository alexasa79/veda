package editor

import (
	"fmt"
	"testing"
)

func TestNavigationHistory(t *testing.T) {
	te := NewTestEditor(80, 24)
	
	// Set initial position
	te.cursorX = 5
	te.cursorY = 10
	te.currentFile = "test.go"
	
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
	if entry.cursorY != 10 {
		t.Errorf("Expected cursorY to be 10, got %d", entry.cursorY)
	}
}

func TestNavigationHistoryMaxSize(t *testing.T) {
	te := NewTestEditor(80, 24)
	te.currentFile = "test.go"
	
	// Push more entries than maxNavHistory
	for i := 0; i < 60; i++ {
		te.cursorY = i
		te.pushNavigationHistory()
	}
	
	// Should only keep the last 50 entries
	if len(te.navHistory) != 50 {
		t.Errorf("Expected navigation history to be capped at 50 entries, got %d", len(te.navHistory))
	}
	
	// First entry should be from position 10 (0-59, keeping last 50)
	if te.navHistory[0].cursorY != 10 {
		t.Errorf("Expected first entry cursorY to be 10, got %d", te.navHistory[0].cursorY)
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
	te.currentFile = "test1.go"
	te.buffers["test1.go"] = &BufferInfo{
		buffer:   te.buffer,
		filePath: "test1.go",
	}
	
	// Push a few positions to history
	te.cursorX = 5
	te.cursorY = 10
	te.pushNavigationHistory()
	
	te.cursorX = 15
	te.cursorY = 20
	te.pushNavigationHistory()
	
	// Move to a different position
	te.cursorX = 25
	te.cursorY = 30
	
	// Go back once
	err := te.goBack()
	if err != nil {
		t.Errorf("goBack returned error: %v", err)
	}
	
	// Should be at the second position
	if te.cursorX != 15 || te.cursorY != 20 {
		t.Errorf("Expected cursor at (15, 20), got (%d, %d)", te.cursorX, te.cursorY)
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
	if te.cursorX != 5 || te.cursorY != 10 {
		t.Errorf("Expected cursor at (5, 10), got (%d, %d)", te.cursorX, te.cursorY)
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