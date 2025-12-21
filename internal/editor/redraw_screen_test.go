package editor

import (
	"testing"
)

func TestRedrawScreenCommand(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "Hello World\nLine 2\nLine 3")

	// Render once
	te.render()

	// Get initial screen content
	initialContent := te.GetScreenContent()
	if len(initialContent) == 0 {
		t.Error("Expected screen to have content after initial render")
	}

	// Call redraw-screen command
	err := te.ExecuteCommand("redraw-screen")
	if err != nil {
		t.Errorf("redraw-screen command failed: %v", err)
	}

	// Screen should still have content after redraw
	afterContent := te.GetScreenContent()
	if len(afterContent) == 0 {
		t.Error("Expected screen to have content after redraw")
	}
}

func TestRedrawScreenClearsAndRedraws(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

	// Render once
	te.render()

	// Call redraw-screen command
	err := te.ExecuteCommand("redraw-screen")
	if err != nil {
		t.Errorf("redraw-screen command failed: %v", err)
	}

	// Check that content is still visible
	content := te.GetScreenContent()
	found := false
	for _, line := range content {
		if len(line) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected screen to have visible content after redraw")
	}
}
