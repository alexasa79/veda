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

func TestEmptyBufferReplacement(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Verify initial state: buffer has empty FilePath but is stored with "[No Name]" key
	if te.buffer.FilePath != "" {
		t.Errorf("Expected initial buffer FilePath to be empty string, got '%s'", te.buffer.FilePath)
	}

	if _, exists := te.buffers["[No Name]"]; !exists {
		t.Errorf("Expected '[No Name]' key in buffers map")
	}

	// Verify we have exactly one buffer initially
	if len(te.buffers) != 1 {
		t.Errorf("Expected 1 initial buffer, got %d", len(te.buffers))
	}

	// Create a new buffer with content (simulating opening a file)
	testContent := "Hello World!\nThis is test content."
	newBuffer := buffer.NewBuffer(testContent, "test.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	// Simulate the openFile logic by adding the new buffer
	te.buffers["test.txt"] = newBuffer

	// Execute the key replacement logic that we fixed
	if !te.buffer.HasFile() {
		if unnamedBuffer, exists := te.buffers["[No Name]"]; exists {
			// Only remove if the buffer is truly empty
			if unnamedBuffer.Rope.String() == "" {
				delete(te.buffers, "[No Name]")
				for i, path := range te.bufferOrder {
					if path == "[No Name]" {
						te.bufferOrder = append(te.bufferOrder[:i], te.bufferOrder[i+1:]...)
						break
					}
				}
			}
		}
	}

	te.bufferOrder = append(te.bufferOrder, "test.txt")
	te.switchToBuffer(newBuffer)

	// Verify the empty buffer was removed
	if _, exists := te.buffers["[No Name]"]; exists {
		t.Error("Empty buffer '[No Name]' should have been removed")
	}

	// Verify we now have exactly one buffer (the file we opened)
	if len(te.buffers) != 1 {
		t.Errorf("Expected 1 buffer after opening file, got %d", len(te.buffers))
	}

	// Verify current buffer is the opened file
	if te.buffer.FilePath != "test.txt" {
		t.Errorf("Expected current buffer to be 'test.txt', got '%s'", te.buffer.FilePath)
	}
}

func TestEmptyBufferPreservedWhenModified(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Add some content to the initial empty buffer to simulate user modifications
	te.buffer.Rope = buffer.NewRopeFromString("User typed this")

	// Create a new buffer (simulating opening a file)
	testContent := "Hello World!\nThis is test content."
	newBuffer := buffer.NewBuffer(testContent, "test.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	te.buffers["test.txt"] = newBuffer

	// Execute the replacement logic - should NOT remove modified buffer
	if !te.buffer.HasFile() {
		if unnamedBuffer, exists := te.buffers["[No Name]"]; exists {
			// Only remove if the buffer is truly empty
			if unnamedBuffer.Rope.String() == "" {
				delete(te.buffers, "[No Name]")
				for i, path := range te.bufferOrder {
					if path == "[No Name]" {
						te.bufferOrder = append(te.bufferOrder[:i], te.bufferOrder[i+1:]...)
						break
					}
				}
			}
		}
	}

	te.bufferOrder = append(te.bufferOrder, "test.txt")
	te.switchToBuffer(newBuffer)

	// Verify the modified empty buffer was NOT removed
	if _, exists := te.buffers["[No Name]"]; !exists {
		t.Error("Modified buffer '[No Name]' should have been preserved")
	}

	// Verify we now have exactly two buffers
	if len(te.buffers) != 2 {
		t.Errorf("Expected 2 buffers after opening file with modified unnamed buffer, got %d", len(te.buffers))
	}
}
