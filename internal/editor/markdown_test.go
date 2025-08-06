package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestMarkdownLinkInsertMode(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		insertText     string
		expectedBuffer string
		description    string
	}{
		{
			name:           "simple_link",
			initialContent: "",
			insertText:     "[link]",
			expectedBuffer: "[link]",
			description:    "typing simple link should not duplicate",
		},
		{
			name:           "link_with_url",
			initialContent: "",
			insertText:     "[link](url)",
			expectedBuffer: "[link](url)",
			description:    "typing link with URL should not duplicate",
		},
		{
			name:           "link_in_sentence",
			initialContent: "Click ",
			insertText:     "[here]",
			expectedBuffer: "Click [here]",
			description:    "adding link to existing text should not duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor with markdown file
			te := NewTestEditorWithContent(80, 24, tt.initialContent)
			
			// Simulate opening a markdown file by setting up the highlighter
			if te.highlighter != nil {
				err := te.highlighter.SetLanguageFromFile("test.md")
				if err != nil {
					t.Fatalf("Failed to set markdown language: %v", err)
				}
			}
			
			// Enter insert mode
			te.ExecuteCommand("enter-insert-mode")
			
			// Move to end of line
			te.ExecuteCommand("end-of-line-insert")
			
			// Type the text character by character
			for i, ch := range tt.insertText {
				t.Logf("Inserting character %d: %c", i, ch)
				t.Logf("Buffer before: %q", te.buffer.Rope.String())
				t.Logf("Cursor position: (%d, %d)", te.buffer.CursorX, te.buffer.CursorIterator.LineNumber())
				
				te.insertChar(ch)
				
				t.Logf("Buffer after: %q", te.buffer.Rope.String())
				t.Logf("Cursor position: (%d, %d)", te.buffer.CursorX, te.buffer.CursorIterator.LineNumber())
				t.Logf("---")
			}
			
			// Check buffer before exiting insert mode
			bufferBeforeExit := te.buffer.Rope.String()
			if bufferBeforeExit != tt.expectedBuffer {
				t.Errorf("%s: Buffer before exit insert mode is wrong\nExpected: %q\nGot: %q", 
					tt.description, tt.expectedBuffer, bufferBeforeExit)
			}
			
			// Exit insert mode
			te.ExecuteCommand("exit-insert-mode")
			
			// Check buffer after exiting insert mode
			bufferAfterExit := te.buffer.Rope.String()
			if bufferAfterExit != tt.expectedBuffer {
				t.Errorf("%s: Buffer after exit insert mode changed!\nExpected: %q\nGot: %q", 
					tt.description, tt.expectedBuffer, bufferAfterExit)
			}
			
			// Verify we're in normal mode
			if te.mode != commands.NormalMode {
				t.Errorf("Expected to be in normal mode")
			}
		})
	}
}

func TestMarkdownLinkEscapeIssue(t *testing.T) {
	// This test verifies that typing "[link" and pressing Escape
	// leaves the buffer as "[link" (no auto-pairing or duplication)
	
	te := NewTestEditorWithContent(80, 24, "")
	
	// Simulate opening a markdown file by setting file path and highlighter
	te.buffer.FilePath = "test.md"
	if te.highlighter != nil {
		err := te.highlighter.SetLanguageFromFile("test.md")
		if err != nil {
			t.Fatalf("Failed to set markdown language: %v", err)
		}
	}
	
	// Enter insert mode
	te.ExecuteCommand("enter-insert-mode")
	
	// Type exactly what the user types: [, l, i, n, k
	chars := []rune{'[', 'l', 'i', 'n', 'k'}
	for i, ch := range chars {
		t.Logf("Step %d: Inserting '%c'", i+1, ch)
		t.Logf("  Buffer before: %q", te.buffer.Rope.String())
		t.Logf("  Cursor at: (%d, %d)", te.buffer.CursorX, te.buffer.CursorIterator.LineNumber())
		
		te.insertChar(ch)
		
		t.Logf("  Buffer after: %q", te.buffer.Rope.String())
		t.Logf("  Cursor at: (%d, %d)", te.buffer.CursorX, te.buffer.CursorIterator.LineNumber())
	}
	
	// Buffer should be "[link" at this point (auto-pairing disabled)
	expectedBeforeEsc := "[link"
	actualBeforeEsc := te.buffer.Rope.String()
	if actualBeforeEsc != expectedBeforeEsc {
		t.Errorf("Buffer before ESC is wrong\nExpected: %q\nGot: %q", expectedBeforeEsc, actualBeforeEsc)
	}
	
	// Now press Escape to exit insert mode
	t.Logf("Pressing ESC to exit insert mode")
	t.Logf("  Buffer before ESC: %q", te.buffer.Rope.String())
	
	te.ExecuteCommand("exit-insert-mode")
	
	t.Logf("  Buffer after ESC: %q", te.buffer.Rope.String())
	t.Logf("  Cursor at: (%d, %d)", te.buffer.CursorX, te.buffer.CursorIterator.LineNumber())
	
	// Buffer should still be "[link", not duplicated
	expectedAfterEsc := "[link"
	actualAfterEsc := te.buffer.Rope.String()
	if actualAfterEsc != expectedAfterEsc {
		t.Errorf("Buffer changed after ESC!\nExpected: %q\nGot: %q", expectedAfterEsc, actualAfterEsc)
	}
}