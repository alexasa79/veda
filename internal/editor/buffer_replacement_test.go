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

	// Verify initial state: no buffers, current buffer is nil
	if len(te.buffers) != 0 {
		t.Errorf("Expected 0 initial buffers, got %d", len(te.buffers))
	}

	if te.buffer != nil {
		t.Errorf("Expected initial buffer to be nil, got %v", te.buffer)
	}

	// Create a new buffer with content (simulating opening a file)
	testContent := "Hello World!\nThis is test content."
	newBuffer := buffer.NewBuffer(testContent, "test.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	// Simulate the openFile logic by adding the new buffer
	te.buffers["test.txt"] = newBuffer
	te.bufferOrder = append(te.bufferOrder, "test.txt")
	te.switchToBuffer(newBuffer)

	// Verify we now have exactly one buffer (the file we opened)
	if len(te.buffers) != 1 {
		t.Errorf("Expected 1 buffer after opening file, got %d", len(te.buffers))
	}

	// Verify current buffer is the opened file
	if te.buffer.FilePath != "test.txt" {
		t.Errorf("Expected current buffer to be 'test.txt', got '%s'", te.buffer.FilePath)
	}
}

func TestMultipleBuffersManagement(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Start with no buffers
	if len(te.buffers) != 0 {
		t.Errorf("Expected 0 initial buffers, got %d", len(te.buffers))
	}

	// Create an unnamed buffer manually (user creates via :new command)
	unnamedBuffer := buffer.NewBuffer("User typed this", "", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	te.buffers["[No Name]"] = unnamedBuffer
	te.bufferOrder = append(te.bufferOrder, "[No Name]")
	te.switchToBuffer(unnamedBuffer)

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
