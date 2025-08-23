package editor

import (
	"testing"
	"veda/internal/buffer"
)

// Helper function to get buffer keys for debugging
func getBufferKeys(buffers map[string]*buffer.Buffer) []string {
	keys := make([]string, 0, len(buffers))
	for k := range buffers {
		keys = append(keys, k)
	}
	return keys
}

func TestFileOpening(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Verify initial state: one initial buffer (the default "[No Name]" buffer)
	if len(te.buffers) != 1 {
		t.Errorf("Expected 1 initial buffer, got %d", len(te.buffers))
	}

	if te.CurrentBuffer() == nil {
		t.Errorf("Expected initial buffer to exist, got nil")
	}

	// Create a new buffer with content (simulating opening a file)
	testContent := "Hello World!\nThis is test content."
	newBuffer := buffer.NewBuffer(testContent, "test.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	// Simulate the openFile logic by adding the new buffer
	te.buffers["test.txt"] = newBuffer
	te.bufferOrder = append(te.bufferOrder, "test.txt")
	te.switchToBuffer(newBuffer)

	// Verify we now have two buffers (the initial buffer and the file we opened)
	if len(te.buffers) != 2 {
		t.Errorf("Expected 2 buffers after opening file, got %d", len(te.buffers))
	}

	// Verify current buffer is the opened file
	if te.CurrentBuffer().FilePath != "test.txt" {
		t.Errorf("Expected current buffer to be 'test.txt', got '%s'", te.CurrentBuffer().FilePath)
	}
}

func TestMultipleBuffersManagement(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Start with one initial buffer (the default "[No Name]" buffer)
	if len(te.buffers) != 1 {
		t.Errorf("Expected 1 initial buffer, got %d", len(te.buffers))
	}

	// Replace the content of the existing "[No Name]" buffer
	if unnamedBuffer, exists := te.buffers["[No Name]"]; exists {
		unnamedBuffer.Insert(0, "User typed this")
		te.switchToBuffer(unnamedBuffer)
	} else {
		t.Fatal("Expected initial [No Name] buffer to exist")
	}

	// Now open a file
	testContent := "Hello World!\nThis is test content."
	newBuffer := buffer.NewBuffer(testContent, "test.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	te.buffers["test.txt"] = newBuffer
	te.bufferOrder = append(te.bufferOrder, "test.txt")
	te.switchToBuffer(newBuffer)

	// Verify both buffers exist
	if _, exists := te.buffers["[No Name]"]; !exists {
		t.Error("Unnamed buffer should still exist")
	}

	if _, exists := te.buffers["test.txt"]; !exists {
		t.Error("File buffer should exist")
	}

	// Verify we now have exactly two buffers
	if len(te.buffers) != 2 {
		t.Errorf("Expected 2 buffers, got %d", len(te.buffers))
	}
}
