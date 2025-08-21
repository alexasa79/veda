package editor

import (
	"fmt"
	"testing"
	"veda/internal/buffer"
)

func TestBufferManagement(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Initially should have no buffers - only created when needed
	if len(te.buffers) != 0 {
		t.Errorf("Expected 0 buffers, got %d", len(te.buffers))
	}

	// Buffer should be nil until needed
	if te.buffer != nil {
		t.Errorf("Expected initial buffer to be nil, but got: %v", te.buffer)
	}
}

func TestSaveCurrentBufferState(t *testing.T) {
	// Create editor with content so we can actually move the iterator
	content := ""
	for i := 1; i <= 20; i++ {
		if i > 1 {
			content += "\n"
		}
		content += fmt.Sprintf("Line %d", i)
	}
	te := NewTestEditorWithContent(80, 24, content)

	// Set some state
	te.SetCursorPosition(5, 10)
	// Move view down by 3 lines
	for i := 0; i < 3; i++ {
		te.buffer.ViewIterator = te.buffer.ViewIterator.Next()
	}

	// Save the state
	te.saveCurrentBufferState()

	// Check that it was saved
	buf := te.buffers[te.buffer.FilePath]
	if buf.CursorX != 5 {
		t.Errorf("Expected cursorX to be 5, got %d", buf.CursorX)
	}
	// Check view iterator was saved
	if buf.ViewIterator == nil {
		t.Errorf("Expected ViewIterator to be saved")
	} else if buf.ViewIterator.LineNumber() != 3 {
		t.Errorf("Expected view iterator line to be 3, got %d", buf.ViewIterator.LineNumber())
	}
}

func TestSwitchToBuffer(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create a new buffer
	newBuffer := buffer.NewBuffer("Hello, World!", "test.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	newBuffer.CursorX = 3
	// CursorIterator and ViewIterator will be created when switching

	// Add to buffers
	te.buffers["test.txt"] = newBuffer
	te.bufferOrder = append(te.bufferOrder, "test.txt")

	// Switch to it
	te.switchToBuffer(newBuffer)

	// Check that the editor state was updated
	if te.buffer.FilePath != "test.txt" {
		t.Errorf("Expected current file to be 'test.txt', got %s", te.buffer.FilePath)
	}
	x, _ := te.GetCursorPosition()
	if x != 3 {
		t.Errorf("Expected cursorX to be 3, got %d", x)
	}
	if te.buffer.Rope.String() != "Hello, World!" {
		t.Errorf("Expected buffer content to be 'Hello, World!', got %s", te.buffer.Rope.String())
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

	// Should have 2 buffers now (no initial empty buffer anymore)
	if len(te.buffers) != 2 {
		t.Errorf("Expected 2 buffers, got %d", len(te.buffers))
	}

	// Switch between buffers
	te.switchToBuffer(buffer1)
	if te.buffer.Rope.String() != "Buffer 1" {
		t.Errorf("Expected buffer content to be 'Buffer 1', got %s", te.buffer.Rope.String())
	}

	te.switchToBuffer(buffer2)
	if te.buffer.Rope.String() != "Buffer 2" {
		t.Errorf("Expected buffer content to be 'Buffer 2', got %s", te.buffer.Rope.String())
	}
}
