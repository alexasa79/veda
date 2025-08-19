package editor

import (
	"testing"
)

func TestEmptyWindowStatusBarCrash(t *testing.T) {
	// Create test editor with some content
	te := NewTestEditorWithContent(300, 40, "Hello, World!")
	
	// Move buffer to the right (this creates a new window and leaves the first window empty)
	err := te.ExecuteCommand("move-buffer-right")
	if err != nil {
		t.Fatalf("Failed to execute moveBufferRight: %v", err)
	}
	
	// Switch to the first window (which should now be empty)
	err = te.ExecuteCommand("window-1")
	if err != nil {
		t.Fatalf("Failed to switch to window 1: %v", err)
	}
	
	// Force a complete render cycle by calling render directly
	// This should trigger the crash in drawStatusBar -> drawText
	te.render()
	
	t.Log("Test completed without crash")
}