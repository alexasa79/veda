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
	
	if te.currentFile != "[No Name]" {
		t.Errorf("Expected current file to be '[No Name]', got %s", te.currentFile)
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
	bufferInfo := te.buffers[te.currentFile]
	if bufferInfo.cursorX != 5 {
		t.Errorf("Expected cursorX to be 5, got %d", bufferInfo.cursorX)
	}
	if bufferInfo.cursorY != 10 {
		t.Errorf("Expected cursorY to be 10, got %d", bufferInfo.cursorY)
	}
	if bufferInfo.topLine != 3 {
		t.Errorf("Expected topLine to be 3, got %d", bufferInfo.topLine)
	}
}

func TestSwitchToBuffer(t *testing.T) {
	te := NewTestEditor(80, 24)
	
	// Create a new buffer info
	newBuffer := buffer.NewBufferFromString("Hello, World!")
	bufferInfo := &BufferInfo{
		buffer:       newBuffer,
		filePath:     "test.txt",
		cursorX:      3,
		cursorY:      0,
		topLine:      0,
		lineOffsets:  []int{},
		offsetsDirty: true,
	}
	
	// Add to buffers
	te.buffers["test.txt"] = bufferInfo
	te.bufferOrder = append(te.bufferOrder, "test.txt")
	
	// Switch to it
	te.switchToBuffer(bufferInfo)
	
	// Check that the editor state was updated
	if te.currentFile != "test.txt" {
		t.Errorf("Expected current file to be 'test.txt', got %s", te.currentFile)
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
	buffer1 := &BufferInfo{
		buffer:   buffer.NewBufferFromString("Buffer 1"),
		filePath: "file1.txt",
	}
	buffer2 := &BufferInfo{
		buffer:   buffer.NewBufferFromString("Buffer 2"),
		filePath: "file2.txt",
	}
	
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