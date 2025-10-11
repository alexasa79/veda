package editor

import (
	"testing"
	"time"
)

func TestAsyncFileSaveTracking(t *testing.T) {
	t.Run("save operation correctly tracks async state and status", func(t *testing.T) {
		// Create test editor with content
		te := NewTestEditorWithContent(80, 24, "Hello, World!\nThis is a test file.")

		// Set a file path for the buffer within project artifacts to avoid using /tmp
		fixture := testFixturePath("test_file.txt")
		te.CurrentBuffer().FilePath = fixture
		te.buffers[fixture] = te.CurrentBuffer()

		// Check that no async operations are running initially
		if te.hasAsyncOperations() {
			t.Error("Expected no async operations initially")
		}

		// Check that connection status is empty initially
		if te.connectionStatus != "" {
			t.Error("Expected connection status to be empty initially")
		}

		// Execute save command (this will start async operation)
		// Note: This will fail at proxy level, but we're just testing the async tracking
		err := te.writeFile([]string{})
		if err != nil {
			t.Fatalf("writeFile returned error: %v", err)
		}

		// Check that async operation was started immediately
		if !te.hasAsyncOperations() {
			t.Error("Expected async operation to be running after writeFile call")
		}

		// Check that connection status is set immediately
		if te.connectionStatus == "" {
			t.Error("Expected connection status to be set during save")
		}

		// Verify the connection status contains "Saving"
		if te.connectionStatus != "Saving file" {
			t.Errorf("Expected connection status to be 'Saving file', got '%s'", te.connectionStatus)
		}

		// Wait for async operation to complete (it should fail and cleanup)
		timeout := time.After(100 * time.Millisecond)
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				t.Error("Async operation did not complete within timeout")
				return
			case <-ticker.C:
				if !te.hasAsyncOperations() {
					// Async operation completed
					if te.connectionStatus != "" {
						t.Error("Expected connection status to be cleared after save completion")
					}
					return
				}
			}
		}
	})
}
