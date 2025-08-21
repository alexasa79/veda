package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestLSPNotificationCallback(t *testing.T) {
	tests := []struct {
		name           string
		operation      func(b *Buffer)
		expectCallback bool
	}{
		{
			name: "Insert triggers callback",
			operation: func(b *Buffer) {
				b.Insert(0, "Hello")
			},
			expectCallback: true,
		},
		{
			name: "Delete triggers callback",
			operation: func(b *Buffer) {
				b.Insert(0, "Hello")
				b.Delete(0, 5)
			},
			expectCallback: true,
		},
		{
			name: "AppendLine triggers callback through Insert",
			operation: func(b *Buffer) {
				b.AppendLine("test line")
			},
			expectCallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new buffer
			b := NewBuffer("", "test.go", config.IndentSpaces, 4)
			b.HasLSP = true // Enable LSP for this buffer

			// Set up callback to track if it was called
			callbackCalled := false
			b.OnContentChange = func() {
				callbackCalled = true
			}

			// Perform the operation
			tt.operation(b)

			// Check if callback was called as expected
			if tt.expectCallback && !callbackCalled {
				t.Errorf("Expected LSP notification callback to be called, but it wasn't")
			} else if !tt.expectCallback && callbackCalled {
				t.Errorf("Expected LSP notification callback not to be called, but it was")
			}
		})
	}
}

func TestLSPNotificationCallbackNotSetByDefault(t *testing.T) {
	// Create a buffer without setting the callback
	b := NewBuffer("", "test.go", config.IndentSpaces, 4)

	// This should not panic even though callback is nil
	b.Insert(0, "Hello")
	b.Delete(0, 5)

	// If we get here without panicking, the test passes
}
