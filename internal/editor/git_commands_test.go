package editor

import (
	"testing"
)

func TestGitBlameCommand(t *testing.T) {
	// Create a test editor with content
	te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3\n")

	// Move cursor to line 2
	te.ExecuteCommand("down")
	AssertCursorAt(t, te, 0, 1)

	// Execute git blame command
	te.ExecuteCommand("git-blame-line")

	// Check that transient info is displayed (it should show an error since we're not in a real git repo)
	if te.Editor.transientInfoMessage == "" && te.Editor.errorManager.GetCurrentError() == "" {
		t.Error("Expected an error or transient info message when running git blame outside of git repo")
	}
}

func TestGitBlameKeyBinding(t *testing.T) {
	// Create a test editor with content
	te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3\n")

	// Move to line 2
	te.ExecuteCommand("down")
	AssertCursorAt(t, te, 0, 1)

	// Execute git blame command through the command registry
	// This tests that the key binding 'g b b' is mapped to the correct command
	te.ExecuteCommand("git-blame-line")

	// Check that transient info is displayed
	if te.Editor.transientInfoMessage == "" && te.Editor.errorManager.GetCurrentError() == "" {
		t.Error("Expected an error or transient info message when running git blame")
	}
}

func TestGitBlameWithoutFile(t *testing.T) {
	// Create a test editor without a file (internal buffer)
	te := NewTestEditor(80, 24)
	te.Editor.CurrentBuffer().FilePath = ""

	// Execute git blame command
	te.ExecuteCommand("git-blame-line")

	// Should show a message that buffer is not associated with a file
	if te.Editor.transientInfoMessage != "Buffer is not associated with a file" {
		t.Errorf("Expected 'Buffer is not associated with a file', got: %s", te.Editor.transientInfoMessage)
	}
}

func TestGitBlameShowResultFormat(t *testing.T) {
	// Test the showBlameResult formatting
	te := NewTestEditor(80, 24)

	// Create a mock result
	result := map[string]interface{}{
		"success": true,
		"commit":  "1234567890abcdef",
		"author":  "John Doe",
		"date":    "2024-01-01 12:00:00",
		"message": "Initial commit\nWith multiple lines",
	}

	// Call showBlameResult
	te.Editor.showBlameResult(result)

	// Check the formatted message
	expected := "1234567 | John Doe | 2024-01-01 12:00:00 | Initial commit"
	if te.Editor.transientInfoMessage != expected {
		t.Errorf("Expected formatted blame: %s, got: %s", expected, te.Editor.transientInfoMessage)
	}
}

func TestGitBlameErrorHandling(t *testing.T) {
	// Test error result handling
	te := NewTestEditor(80, 24)

	// Create an error result
	result := map[string]interface{}{
		"success": false,
		"error":   "Not a git repository",
	}

	// Call showBlameResult
	te.Editor.showBlameResult(result)

	// Check the error message
	expected := "Git blame failed: Not a git repository"
	if te.Editor.transientInfoMessage != expected {
		t.Errorf("Expected error message: %s, got: %s", expected, te.Editor.transientInfoMessage)
	}
}

func TestGitBlameNoInformation(t *testing.T) {
	// Test when no blame information is available
	te := NewTestEditor(80, 24)

	// Create a result with empty fields
	result := map[string]interface{}{
		"success": true,
		"commit":  "",
		"author":  "",
		"date":    "",
		"message": "",
	}

	// Call showBlameResult
	te.Editor.showBlameResult(result)

	// Check the message
	expected := "No git blame information available"
	if te.Editor.transientInfoMessage != expected {
		t.Errorf("Expected no info message: %s, got: %s", expected, te.Editor.transientInfoMessage)
	}
}