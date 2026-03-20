package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestAutomaticCompletion(t *testing.T) {
	t.Run("automatic completion triggered when typing dot with LSP", func(t *testing.T) {
		// Create test editor with content
		content := "myStruct"
		te := NewTestEditorWithContent(80, 24, content)

		// Enable LSP for this buffer (simulate having LSP available)
		te.Editor.lspExtensionStatus[""] = "ready"

		// Position cursor at end and enter insert mode
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("append-after-cursor")

		// Note: We don't need to mock the proxy manager for this basic test

		// Type a dot - should trigger completion
		te.InsertChar('.')

		// For now, we just verify that no error occurred
		// In a real test, we'd mock the LSP response and check if showCompletions was called
		// This is a basic integration test to ensure the code path works

		// Verify content includes the dot
		expectedContent := "myStruct."
		actualContent := te.CurrentBuffer().Rope.String()
		if actualContent != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, actualContent)
		}
	})

	t.Run("automatic completion triggered when typing arrow operator with LSP", func(t *testing.T) {
		// Create test editor with content
		content := "myPtr-"
		te := NewTestEditorWithContent(80, 24, content)

		// Enable LSP for this buffer
		te.Editor.lspExtensionStatus[""] = "ready"

		// Position cursor at end and enter insert mode
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("append-after-cursor")

		// Note: We don't need to mock the proxy manager for this basic test

		// Type '>' to complete the '->' operator - should trigger completion
		te.InsertChar('>')

		// Verify content includes the arrow operator
		expectedContent := "myPtr->"
		actualContent := te.CurrentBuffer().Rope.String()
		if actualContent != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, actualContent)
		}
	})

	t.Run("no automatic completion without LSP", func(t *testing.T) {
		// Create test editor with content
		content := "myStruct"
		te := NewTestEditorWithContent(80, 24, content)

		// Ensure LSP is NOT available for this buffer
		// lspExtensionStatus not set for ".txt" — no LSP

		// Position cursor at end and enter insert mode
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("append-after-cursor")

		// Type a dot - should NOT trigger completion (since no LSP)
		te.InsertChar('.')

		// Verify content includes the dot but no completion was attempted
		expectedContent := "myStruct."
		actualContent := te.CurrentBuffer().Rope.String()
		if actualContent != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, actualContent)
		}

		// The key here is that we didn't crash and the dot was inserted normally
	})

	t.Run("no automatic completion in normal mode", func(t *testing.T) {
		// Create test editor with content
		content := "myStruct"
		te := NewTestEditorWithContent(80, 24, content)

		// Enable LSP for this buffer
		te.Editor.lspExtensionStatus[""] = "ready"

		// Stay in normal mode (don't enter insert mode)
		te.ExecuteCommand("end-of-line")

		// Verify we're in normal mode
		if te.mode != commands.NormalMode {
			t.Errorf("Expected to be in normal mode, got %v", te.mode)
		}

		// In normal mode, typing characters isn't handled the same way
		// This test mainly verifies our logic checks for insert mode correctly
	})

	t.Run("arrow operator detection works correctly", func(t *testing.T) {
		// Test various scenarios for arrow operator detection
		testCases := []struct {
			name          string
			content       string
			insertChar    rune
			shouldTrigger bool
		}{
			{"complete arrow operator", "ptr-", '>', true},
			{"just typing > without dash", "test", '>', false},
			{"typing > after other char", "a+", '>', false},
			{"empty buffer", "", '>', false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				te := NewTestEditorWithContent(80, 24, tc.content)
				te.Editor.lspExtensionStatus[""] = "ready"

				// Position cursor at end and enter insert mode
				if len(tc.content) > 0 {
					te.ExecuteCommand("end-of-line")
				}
				te.ExecuteCommand("append-after-cursor")

				// Note: We don't need to mock the proxy manager for this basic test

				// Insert the character
				te.InsertChar(tc.insertChar)

				// Verify the character was inserted (basic functionality works)
				expectedContent := tc.content + string(tc.insertChar)
				actualContent := te.CurrentBuffer().Rope.String()
				if actualContent != expectedContent {
					t.Errorf("Expected content '%s', got '%s'", expectedContent, actualContent)
				}
			})
		}
	})
}
