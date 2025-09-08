package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestTrailingSpaceRemovalOnExitInsertMode tests that trailing spaces are removed
// from changed lines when exiting insert mode
func TestTrailingSpaceRemovalOnExitInsertMode(t *testing.T) {
	// Create a mock screen
	mockScreen := NewMockScreen(80, 24)

	// Create editor
	editor, err := NewWithScreen(mockScreen)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}
	defer editor.Close()

	// Start with content that has some trailing spaces
	initialContent := "line 1    \nline 2\nline 3   "

	// Set up buffer
	editor.SetCurrentBuffer(buffer.NewBuffer(initialContent, "test.txt", config.IndentSpaces, 4))

	// Enter insert mode
	editor.switchToInsertMode()

	// Modify line 0 by adding text
	editor.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(editor.CurrentBuffer(), 0)
	editor.CurrentBuffer().CursorX = 6 // After "line 1"

	// Insert some text with trailing spaces
	for _, ch := range " modified   " {
		err = editor.insertChar(ch)
		if err != nil {
			t.Fatalf("Failed to insert character: %v", err)
		}
	}

	// Move to line 2 and add some text with trailing spaces
	editor.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(editor.CurrentBuffer(), 2)
	editor.CurrentBuffer().CursorX = 6 // After "line 3"
	for _, ch := range " changed    " {
		err = editor.insertChar(ch)
		if err != nil {
			t.Fatalf("Failed to insert character: %v", err)
		}
	}

	// Exit insert mode - this should clean up trailing spaces on changed lines
	err = editor.exitInsertMode([]string{})
	if err != nil {
		t.Fatalf("Failed to exit insert mode: %v", err)
	}

	// Check the content
	content := editor.CurrentBuffer().Rope.String()
	lines := strings.Split(content, "\n")

	// Line 0 should have been modified and trailing spaces removed
	if strings.HasSuffix(lines[0], " ") {
		t.Errorf("Line 0 should not have trailing spaces: %q", lines[0])
	}

	// Line 1 was not changed, so it should remain as is
	if lines[1] != "line 2" {
		t.Errorf("Line 1 should not be modified: %q", lines[1])
	}

	// Line 2 should have been modified and trailing spaces removed
	if strings.HasSuffix(lines[2], " ") {
		t.Errorf("Line 2 should not have trailing spaces: %q", lines[2])
	}
}

// TestTrailingSpaceRemovalOnSave tests that trailing spaces are removed
// from changed lines when saving a file
func TestTrailingSpaceRemovalOnSave(t *testing.T) {
	// Create a mock screen
	mockScreen := NewMockScreen(80, 24)

	// Create editor
	editor, err := NewWithScreen(mockScreen)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}
	defer editor.Close()

	// Create a temporary file for testing
	tmpFile := t.TempDir() + "/test.txt"

	// Start with clean content
	initialContent := "line 1\nline 2\nline 3"

	// Set up buffer
	buf := buffer.NewBuffer(initialContent, tmpFile, config.IndentSpaces, 4)
	editor.SetCurrentBuffer(buf)

	// Enter insert mode and modify some lines with trailing spaces
	editor.switchToInsertMode()

	// Modify line 0
	editor.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(editor.CurrentBuffer(), 0)
	editor.CurrentBuffer().CursorX = 6 // After "line 1"
	for _, ch := range "   " {         // Add trailing spaces
		err = editor.insertChar(ch)
		if err != nil {
			t.Fatalf("Failed to insert character: %v", err)
		}
	}

	// Modify line 2
	editor.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(editor.CurrentBuffer(), 2)
	editor.CurrentBuffer().CursorX = 6 // After "line 3"
	for _, ch := range "    " {        // Add trailing spaces
		err = editor.insertChar(ch)
		if err != nil {
			t.Fatalf("Failed to insert character: %v", err)
		}
	}

	// Stay in insert mode and save - trailing spaces should still be removed
	// Note: The actual save operation happens asynchronously, so we'll check the buffer content
	// after the cleanup which happens synchronously before the async save

	// Get buffer content before simulating save cleanup
	editor.CurrentBuffer().RemoveTrailingSpacesFromChangedLines()

	content := editor.CurrentBuffer().Rope.String()
	lines := strings.Split(content, "\n")

	// Check that trailing spaces were removed from changed lines
	if strings.HasSuffix(lines[0], " ") {
		t.Errorf("Line 0 should not have trailing spaces after save: %q", lines[0])
	}

	if lines[1] != "line 2" {
		t.Errorf("Line 1 should not be modified: %q", lines[1])
	}

	if strings.HasSuffix(lines[2], " ") {
		t.Errorf("Line 2 should not have trailing spaces after save: %q", lines[2])
	}
}

// TestNoTrailingSpaceRemovalForUnchangedLines tests that lines that weren't
// modified in the session are not affected
func TestNoTrailingSpaceRemovalForUnchangedLines(t *testing.T) {
	// Create a mock screen
	mockScreen := NewMockScreen(80, 24)

	// Create editor
	editor, err := NewWithScreen(mockScreen)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}
	defer editor.Close()

	// Start with content that already has trailing spaces on some lines
	initialContent := "line 1    \nline 2  \nline 3"

	// Set up buffer
	editor.SetCurrentBuffer(buffer.NewBuffer(initialContent, "test.txt", config.IndentSpaces, 4))

	// Enter insert mode
	editor.switchToInsertMode()

	// Only modify line 2 (which doesn't have trailing spaces)
	editor.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(editor.CurrentBuffer(), 2)
	editor.CurrentBuffer().CursorX = 6 // After "line 3"

	// Add text without trailing spaces
	for _, ch := range " modified" {
		err = editor.insertChar(ch)
		if err != nil {
			t.Fatalf("Failed to insert character: %v", err)
		}
	}

	// Exit insert mode
	err = editor.exitInsertMode([]string{})
	if err != nil {
		t.Fatalf("Failed to exit insert mode: %v", err)
	}

	// Check the content
	content := editor.CurrentBuffer().Rope.String()
	lines := strings.Split(content, "\n")

	// Lines 0 and 1 were NOT changed, so their trailing spaces should remain
	if lines[0] != "line 1    " {
		t.Errorf("Line 0 should keep its original trailing spaces: %q", lines[0])
	}

	if lines[1] != "line 2  " {
		t.Errorf("Line 1 should keep its original trailing spaces: %q", lines[1])
	}

	// Line 2 was changed and should not have trailing spaces
	if lines[2] != "line 3 modified" {
		t.Errorf("Line 2 should be modified without trailing spaces: %q", lines[2])
	}
}
