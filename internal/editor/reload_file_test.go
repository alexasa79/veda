package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/proxy"
)

func TestReloadFileNoFile(t *testing.T) {
	// Test reload when no file is open
	te := NewTestEditor(80, 24)

	// Try to reload when no file is open
	err := te.reloadFile(nil)
	if err != nil {
		t.Errorf("reloadFile should return nil for no file, got: %v", err)
	}

	// Check error message was shown
	if te.errorManager.GetCurrentError() != "No file to reload" {
		t.Errorf("Expected 'No file to reload' error, got: %s", te.errorManager.GetCurrentError())
	}
}

func TestReloadFileNoUnsavedChanges(t *testing.T) {
	// Test reload with no unsaved changes
	te := NewTestEditor(80, 24)

	// Create a buffer with a file path
	buf := buffer.NewBuffer("Original content\nLine 2", "test.txt", config.IndentTabs, 4)
	te.buffers["test.txt"] = buf
	te.SetCurrentBuffer(buf)

	// Mark as not modified
	buf.IsModified = false

	// Call reload - should call performReload immediately
	err := te.reloadFile(nil)
	if err != nil {
		t.Errorf("reloadFile returned error: %v", err)
	}

	// Note: In a real test we'd need to mock the proxy manager
	// to verify performReload was called
}

func TestReloadFileWithUnsavedChanges(t *testing.T) {
	// Test reload with unsaved changes - should show dialog
	te := NewTestEditor(80, 24)

	// Create a buffer with unsaved changes
	buf := buffer.NewBuffer("Modified content", "test.txt", config.IndentTabs, 4)
	te.buffers["test.txt"] = buf
	te.SetCurrentBuffer(buf)

	// Mark as modified
	buf.IsModified = true

	// Call reload - should show confirmation dialog
	err := te.reloadFile(nil)
	if err != nil {
		t.Errorf("reloadFile returned error: %v", err)
	}

	// Check that overlay was set (dialog shown)
	if te.renderingState.overlay == nil {
		t.Error("Expected confirmation dialog to be shown")
	}
}

func TestReloadFilePreservesCursorPosition(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create buffer with multiple lines
	buf := buffer.NewBuffer("Line 1\nLine 2\nLine 3\nLine 4", "test.txt", config.IndentTabs, 4)
	te.buffers["test.txt"] = buf
	te.SetCurrentBuffer(buf)

	// Position cursor at line 2, column 3
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 2)
	buf.CursorX = 3

	// Simulate reload with same content structure
	data := map[string]interface{}{
		"path":        "test.txt",
		"content":     "Line A\nLine B\nLine C\nLine D",
		"changes":     proxy.FileChanges{},
		"branch":      "main",
		"diagnostics": []proxy.Diagnostic{},
	}

	te.handleFileReloaded(data)

	// Get the new buffer after reload
	newBuf := te.buffers["test.txt"]

	// Cursor should still be at line 2
	if newBuf.CursorIterator.LineNumber() != 2 {
		t.Errorf("Cursor line not preserved. Expected 2, got %d", newBuf.CursorIterator.LineNumber())
	}

	// Cursor X should still be 3
	if newBuf.CursorX != 3 {
		t.Errorf("Cursor X not preserved. Expected 3, got %d", newBuf.CursorX)
	}
}

func TestReloadFileAdjustsCursorForShorterFile(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create buffer with 5 lines
	buf := buffer.NewBuffer("Line 1\nLine 2\nLine 3\nLine 4\nLine 5", "test.txt", config.IndentTabs, 4)
	te.buffers["test.txt"] = buf
	te.SetCurrentBuffer(buf)

	// Position cursor at last line
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 4)
	buf.CursorX = 2

	// Reload with only 2 lines
	data := map[string]interface{}{
		"path":        "test.txt",
		"content":     "Line 1\nLine 2",
		"changes":     proxy.FileChanges{},
		"branch":      "main",
		"diagnostics": []proxy.Diagnostic{},
	}

	te.handleFileReloaded(data)

	// Get the new buffer after reload
	newBuf := te.buffers["test.txt"]

	// Cursor should be adjusted to last line (line 1, 0-indexed)
	if newBuf.CursorIterator.LineNumber() != 1 {
		t.Errorf("Cursor not adjusted to last line. Expected 1, got %d", newBuf.CursorIterator.LineNumber())
	}
}
