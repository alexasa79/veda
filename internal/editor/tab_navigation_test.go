package editor

import (
	"testing"
)

func TestTabNavigation(t *testing.T) {
	t.Run("cursor preserves position when moving down through tab-indented lines", func(t *testing.T) {
		// Create content with tab indentation like in theme.go
		content := "\t// First comment line\n\t// Second comment line\n\t// Third comment line"
		te := NewTestEditorWithContent(80, 24, content)

		// Move to position after tab on first line
		te.buffer.CursorX = 1 // Right after the tab
		AssertCursorAt(t, te, 1, 0)

		// Move down - cursor should stay at position 1
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 1, 1) // Should still be at position 1, not jumping right

		// Move down again - cursor should stay at position 1
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 1, 2) // Should still be at position 1
	})

	t.Run("cursor preserves visual column through lines with different tab counts", func(t *testing.T) {
		// Lines with different tab indentation
		content := "\t// One tab\n\t\t// Two tabs\n\t// Back to one tab"
		te := NewTestEditorWithContent(80, 24, content)

		// Position after first tab on line 1
		te.buffer.CursorX = 1
		AssertCursorAt(t, te, 1, 0)

		// Get initial visual column
		_, initialVisualCol := te.GetCursorVisualPosition()
		t.Logf("Initial visual column: %d", initialVisualCol)

		// Move down to line with two tabs
		te.ExecuteCommand("down")
		_, visualCol2 := te.GetCursorVisualPosition()
		t.Logf("Visual column on line 2: %d", visualCol2)

		// Visual column should be preserved (same visual position)
		if visualCol2 != initialVisualCol {
			t.Errorf("Visual column not preserved on line 2. Expected %d, got %d", initialVisualCol, visualCol2)
		}

		// But buffer position might be different due to extra tab
		x2, _ := te.GetCursorPosition()
		t.Logf("Buffer position on line 2: %d", x2)

		// Move down to line with one tab again
		te.ExecuteCommand("down")
		_, visualCol3 := te.GetCursorVisualPosition()
		x3, _ := te.GetCursorPosition()

		// Visual column should still be preserved
		if visualCol3 != initialVisualCol {
			t.Errorf("Visual column not preserved on line 3. Expected %d, got %d", initialVisualCol, visualCol3)
		}

		// Buffer position should be back to 1
		if x3 != 1 {
			t.Errorf("Buffer position on line 3 should be 1, got %d", x3)
		}
	})

	t.Run("visual column to byte position conversion", func(t *testing.T) {
		// Test that we can correctly convert visual columns back to byte positions
		// Example line: "\t\tHello World" (2 tabs + text)

		testCases := []struct {
			visualCol       int
			expectedBytePos int
			description     string
		}{
			{0, 0, "Start of line"},
			{4, 1, "After first tab"},
			{8, 2, "After second tab"},
			{9, 3, "First char 'H'"},
			{10, 4, "Second char 'e'"},
		}

		for _, tc := range testCases {
			// This test documents the expected behavior
			// Currently there's no function to do this conversion
			t.Logf("%s: visual column %d should map to byte position %d",
				tc.description, tc.visualCol, tc.expectedBytePos)
		}
	})
}
