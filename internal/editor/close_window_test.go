package editor

import (
	"testing"
)

func TestCloseWindow(t *testing.T) {
	// Create test editor with multiple windows
	te := NewTestEditorWithContent(300, 40, "Hello, World!")
	
	// Create a second window by moving buffer right
	err := te.ExecuteCommand("move-buffer-right")
	if err != nil {
		t.Fatalf("Failed to create second window: %v", err)
	}
	
	// Verify we have 2 windows
	if len(te.windows) != 2 {
		t.Fatalf("Expected 2 windows, got %d", len(te.windows))
	}
	
	// Close the current window (should be window 1, the right window)
	err = te.ExecuteCommand("close-window")
	if err != nil {
		t.Fatalf("Failed to close window: %v", err)
	}
	
	// Verify we now have 1 window
	if len(te.windows) != 1 {
		t.Fatalf("Expected 1 window after close, got %d", len(te.windows))
	}
	
	// Verify active window is valid
	if te.activeWindow < 0 || te.activeWindow >= len(te.windows) {
		t.Fatalf("Active window index %d is invalid for %d windows", te.activeWindow, len(te.windows))
	}
}

func TestCloseWindowLastWindow(t *testing.T) {
	// Create test editor with only one window
	te := NewTestEditorWithContent(300, 40, "Hello, World!")
	
	// Verify we have 1 window
	if len(te.windows) != 1 {
		t.Fatalf("Expected 1 window, got %d", len(te.windows))
	}
	
	// Try to close the last window (should be ignored)
	err := te.ExecuteCommand("close-window")
	if err != nil {
		t.Fatalf("Failed to execute close-window command: %v", err)
	}
	
	// Verify we still have 1 window (shouldn't close the last window)
	if len(te.windows) != 1 {
		t.Fatalf("Expected 1 window after trying to close last window, got %d", len(te.windows))
	}
}

func TestCloseWindowMiddleWindow(t *testing.T) {
	// Create test editor with 3 windows
	te := NewTestEditorWithContent(300, 40, "Hello, World!")
	
	// Create second window
	err := te.ExecuteCommand("move-buffer-right")
	if err != nil {
		t.Fatalf("Failed to create second window: %v", err)
	}
	
	// Create third window
	err = te.ExecuteCommand("move-buffer-right")
	if err != nil {
		t.Fatalf("Failed to create third window: %v", err)
	}
	
	// Verify we have 3 windows
	if len(te.windows) != 3 {
		t.Fatalf("Expected 3 windows, got %d", len(te.windows))
	}
	
	// Switch to middle window (window 1, 0-indexed)
	err = te.ExecuteCommand("window-2")
	if err != nil {
		t.Fatalf("Failed to switch to window 2: %v", err)
	}
	
	// Verify active window is 1 (0-indexed, which is window 2 in 1-based)
	if te.activeWindow != 1 {
		t.Fatalf("Expected active window 1, got %d", te.activeWindow)
	}
	
	// Close the middle window
	err = te.ExecuteCommand("close-window")
	if err != nil {
		t.Fatalf("Failed to close middle window: %v", err)
	}
	
	// Verify we now have 2 windows
	if len(te.windows) != 2 {
		t.Fatalf("Expected 2 windows after closing middle window, got %d", len(te.windows))
	}
	
	// Verify active window is valid
	if te.activeWindow < 0 || te.activeWindow >= len(te.windows) {
		t.Fatalf("Active window index %d is invalid for %d windows", te.activeWindow, len(te.windows))
	}
}

func TestCloseWindowKeyBinding(t *testing.T) {
	// Create test editor with multiple windows
	te := NewTestEditorWithContent(300, 40, "Hello, World!")
	
	// Create a second window
	err := te.ExecuteCommand("move-buffer-right")
	if err != nil {
		t.Fatalf("Failed to create second window: %v", err)
	}
	
	// Verify we have 2 windows
	if len(te.windows) != 2 {
		t.Fatalf("Expected 2 windows, got %d", len(te.windows))
	}
	
	// Test the key binding "space w x"
	te.TypeKeys(" wx")
	
	// Verify we now have 1 window
	if len(te.windows) != 1 {
		t.Fatalf("Expected 1 window after using key binding, got %d", len(te.windows))
	}
}