package editor

import (
	"testing"
	"veda/internal/buffer"
)

func TestBufferManagement(t *testing.T) {
	te := NewTestEditor(80, 24)
	
	// Initially should have one empty buffer
	if len(te.buffers) != 1 {
		t.Errorf("Expected 1 buffer, got %d", len(te.buffers))
	}
	
	if te.buffer.HasFile() {
		t.Errorf("Expected initial buffer to have no file, but it has: %s", te.buffer.FilePath)
	}
}

func TestSaveCurrentBufferState(t *testing.T) {
	te := NewTestEditor(80, 24)
	
	// Set some state
	te.cursorX = 5
	te.cursorY = 10
	te.topLine = 3
	
	// Save the state
	te.saveCurrentBufferState()
	
	// Check that it was saved
	buf := te.buffers[te.buffer.FilePath]
	if buf.CursorX != 5 {
		t.Errorf("Expected cursorX to be 5, got %d", buf.CursorX)
	}
	if buf.CursorY != 10 {
		t.Errorf("Expected cursorY to be 10, got %d", buf.CursorY)
	}
	if buf.TopLine != 3 {
		t.Errorf("Expected topLine to be 3, got %d", buf.TopLine)
	}
}

func TestSwitchToBuffer(t *testing.T) {
	te := NewTestEditor(80, 24)
	
	// Create a new buffer
	newBuffer := buffer.NewBuffer("Hello, World!", "test.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	newBuffer.CursorX = 3
	newBuffer.CursorY = 0
	newBuffer.TopLine = 0
	newBuffer.LineOffsets = []int{}
	newBuffer.OffsetsDirty = true
	
	// Add to buffers
	te.buffers["test.txt"] = newBuffer
	te.bufferOrder = append(te.bufferOrder, "test.txt")
	
	// Switch to it
	te.switchToBuffer(newBuffer)
	
	// Check that the editor state was updated
	if te.buffer.FilePath != "test.txt" {
		t.Errorf("Expected current file to be 'test.txt', got %s", te.buffer.FilePath)
	}
	if te.cursorX != 3 {
		t.Errorf("Expected cursorX to be 3, got %d", te.cursorX)
	}
	if te.buffer.String() != "Hello, World!" {
		t.Errorf("Expected buffer content to be 'Hello, World!', got %s", te.buffer.String())
	}
}

func TestMultipleBuffers(t *testing.T) {
	te := NewTestEditor(80, 24)
	
	// Create multiple buffers
	buffer1 := buffer.NewBuffer("Buffer 1", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buffer2 := buffer.NewBuffer("Buffer 2", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	
	te.buffers["file1.txt"] = buffer1
	te.buffers["file2.txt"] = buffer2
	te.bufferOrder = append(te.bufferOrder, "file1.txt", "file2.txt")
	
	// Should have 3 buffers now (including the initial empty one)
	if len(te.buffers) != 3 {
		t.Errorf("Expected 3 buffers, got %d", len(te.buffers))
	}
	
	// Switch between buffers
	te.switchToBuffer(buffer1)
	if te.buffer.String() != "Buffer 1" {
		t.Errorf("Expected buffer content to be 'Buffer 1', got %s", te.buffer.String())
	}
	
	te.switchToBuffer(buffer2)
	if te.buffer.String() != "Buffer 2" {
		t.Errorf("Expected buffer content to be 'Buffer 2', got %s", te.buffer.String())
	}
}