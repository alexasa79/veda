package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestVisualLineMode(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		startLine         int
		endLine           int
		expectedSelection string
		description       string
	}{
		{
			name: "single line selection",
			content: `Line 1
Line 2
Line 3`,
			startLine:         1,
			endLine:           1,
			expectedSelection: "Line 2\n",
			description:       "Should select entire second line including newline",
		},
		{
			name: "multi-line selection forward",
			content: `First line
Second line
Third line
Fourth line`,
			startLine:         1,
			endLine:           2,
			expectedSelection: "Second line\nThird line\n",
			description:       "Should select lines 2 and 3 with newlines",
		},
		{
			name: "multi-line selection backward",
			content: `First line
Second line
Third line
Fourth line`,
			startLine:         2,
			endLine:           1,
			expectedSelection: "Second line\nThird line\n",
			description:       "Should select lines 2 and 3 when selecting backward",
		},
		{
			name: "last line selection",
			content: `Line 1
Line 2
Last line`,
			startLine:         2,
			endLine:           2,
			expectedSelection: "Last line",
			description:       "Should select last line without trailing newline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor
			te := NewTestEditorWithContent(80, 24, tt.content)

			// Move to start line
			for i := 0; i < tt.startLine; i++ {
				te.ExecuteCommand("down")
			}

			// Enter visual line mode
			err := te.ExecuteCommand("enter-visual-line-mode")
			if err != nil {
				t.Fatalf("Failed to enter visual line mode: %v", err)
			}

			// Check mode
			if te.mode != commands.VisualLineMode {
				t.Errorf("Expected mode to be VISUAL LINE, got %v", te.mode)
			}

			// Move to end line
			if tt.endLine > tt.startLine {
				for i := 0; i < tt.endLine-tt.startLine; i++ {
					te.ExecuteCommand("down")
				}
			} else if tt.endLine < tt.startLine {
				for i := 0; i < tt.startLine-tt.endLine; i++ {
					te.ExecuteCommand("up")
				}
			}

			// Get selection
			start, end, hasSelection := te.getSelectionRange()
			if !hasSelection {
				t.Errorf("%s: Expected selection but got none", tt.description)
			}

			// Extract selected text
			content := te.CurrentBuffer().Rope.String()
			if end > len(content) {
				end = len(content)
			}
			selectedText := content[start:end]

			// Check selection
			if selectedText != tt.expectedSelection {
				t.Errorf("%s: Selection mismatch\nExpected: %q\nGot: %q",
					tt.description, tt.expectedSelection, selectedText)
			}
		})
	}
}

func TestVisualLineModeOperations(t *testing.T) {
	// Test deleting lines in visual line mode
	t.Run("delete lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, `Line 1
Line 2
Line 3
Line 4`)

		// Select lines 2 and 3
		te.ExecuteCommand("down")
		te.ExecuteCommand("enter-visual-line-mode")
		te.ExecuteCommand("down")

		// Delete selection
		err := te.ExecuteCommand("delete-selection")
		if err != nil {
			t.Fatalf("Failed to delete selection: %v", err)
		}

		// Check result
		expected := `Line 1
Line 4`
		if te.CurrentBuffer().Rope.String() != expected {
			t.Errorf("After delete, expected:\n%s\nGot:\n%s",
				expected, te.CurrentBuffer().Rope.String())
		}

		// Should be back in normal mode
		if te.mode != commands.NormalMode {
			t.Errorf("Expected to be in normal mode after delete, got %v", te.mode)
		}
	})

	// Test yanking lines in visual line mode
	t.Run("yank lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, `Line 1
Line 2
Line 3`)

		// Select line 2
		te.ExecuteCommand("down")
		te.ExecuteCommand("enter-visual-line-mode")

		// Yank selection
		err := te.ExecuteCommand("yank-selection")
		if err != nil {
			t.Fatalf("Failed to yank selection: %v", err)
		}

		// Buffer should be unchanged
		expected := `Line 1
Line 2
Line 3`
		if te.CurrentBuffer().Rope.String() != expected {
			t.Errorf("Buffer changed after yank, expected:\n%s\nGot:\n%s",
				expected, te.CurrentBuffer().Rope.String())
		}

		// Should be back in normal mode
		if te.mode != commands.NormalMode {
			t.Errorf("Expected to be in normal mode after yank, got %v", te.mode)
		}
	})
}

func TestVisualLineModeExit(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "Test content")

	// Enter visual line mode
	te.ExecuteCommand("enter-visual-line-mode")

	// Exit with escape
	err := te.ExecuteCommand("exit-visual-mode")
	if err != nil {
		t.Fatalf("Failed to exit visual mode: %v", err)
	}

	// Should be back in normal mode
	if te.mode != commands.NormalMode {
		t.Errorf("Expected to be in normal mode, got %v", te.mode)
	}

	// Visual anchor should be cleared
	if te.CurrentBuffer().VisualAnchorLine != -1 {
		t.Errorf("Expected visual anchor line to be -1, got %d", te.CurrentBuffer().VisualAnchorLine)
	}
}
