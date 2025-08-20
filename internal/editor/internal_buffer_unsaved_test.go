package editor

import (
	"testing"
)

func TestInternalBuffersExcludedFromUnsavedChecks(t *testing.T) {
	t.Run("internal Messages buffer should not trigger unsaved warnings", func(t *testing.T) {
		te := NewTestEditor(80, 24)
		
		// Create the Messages buffer by logging an error
		te.ShowError("Test error message")
		
		// Verify the Messages buffer was created and is marked as modified
		messagesBuffer, exists := te.buffers["*Messages*"]
		if !exists {
			t.Fatal("Messages buffer should be created")
		}
		
		if !messagesBuffer.IsInternal {
			t.Error("Messages buffer should be marked as internal")
		}
		
		if !messagesBuffer.IsModified {
			t.Error("Messages buffer should be modified after adding content")
		}
		
		// Now check that unsaved functions don't include internal buffers
		if te.hasUnsavedChanges() {
			t.Error("hasUnsavedChanges() should return false when only internal buffers are modified")
		}
		
		unsavedBuffers := te.getUnsavedBuffers()
		if len(unsavedBuffers) > 0 {
			t.Errorf("getUnsavedBuffers() should return empty slice, got: %v", unsavedBuffers)
		}
		
		// Test quitWithWarning doesn't trigger for internal buffer
		// Switch to the Messages buffer
		te.switchToBuffer(messagesBuffer)
		
		// Try to quit - should not show unsaved warning
		err := te.quitWithWarning([]string{}, false)
		if err != nil {
			t.Errorf("quitWithWarning should not show warning for internal buffer, got error: %v", err)
		}
	})
	
	t.Run("regular modified buffer should still trigger unsaved warnings", func(t *testing.T) {
		te := NewTestEditor(80, 24)
		
        // Create a real file buffer (use a non-internal dummy path)
        te.openFile("test_file.txt")
		
		// Modify the buffer
		te.InsertChar('x')
		
		// Verify the buffer is marked as modified
		if !te.buffer.IsModified {
			t.Error("Buffer should be modified")
		}
		
		if te.buffer.IsInternal {
			t.Error("Regular file buffer should not be internal")
		}
		
		// Check that unsaved functions properly detect this
		if !te.hasUnsavedChanges() {
			t.Error("hasUnsavedChanges() should return true for modified file buffer")
		}
		
		unsavedBuffers := te.getUnsavedBuffers()
		if len(unsavedBuffers) == 0 {
			t.Error("getUnsavedBuffers() should return the modified buffer")
		}
	})
	
	t.Run("mixed internal and regular buffers", func(t *testing.T) {
		te := NewTestEditor(80, 24)
		
		// Create Messages buffer
		te.ShowError("Test error")
		
        // Create a regular file buffer but don't modify it
        te.openFile("test_file2.txt")
		
		// Both buffers exist, but only Messages is modified
		messagesBuffer := te.buffers["*Messages*"]
		if !messagesBuffer.IsModified || !messagesBuffer.IsInternal {
			t.Error("Messages buffer should be modified and internal")
		}
		
		if te.buffer.IsModified {
			t.Error("File buffer should not be modified")
		}
		
		// Should not trigger unsaved warnings
		if te.hasUnsavedChanges() {
			t.Error("Should not have unsaved changes when only internal buffers are modified")
		}
		
		// Now modify the file buffer
		te.InsertChar('y')
		
		// Now should trigger unsaved warnings
		if !te.hasUnsavedChanges() {
			t.Error("Should have unsaved changes when file buffer is modified")
		}
		
		unsavedBuffers := te.getUnsavedBuffers()
		if len(unsavedBuffers) != 1 {
			t.Errorf("Should have exactly 1 unsaved buffer, got: %d", len(unsavedBuffers))
		}
	})
}