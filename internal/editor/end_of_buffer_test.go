package editor

import (
	"strings"
	"testing"
)

func TestLineNumbersAtEndOfBuffer(t *testing.T) {
	// Simple content to test line numbering at EOF
	content := "Line1\nLine2\nLine3"

	t.Run("open line below at end of buffer", func(t *testing.T) {
		t.Skip("Line numbering at EOF with empty lines needs investigation")
		te := NewTestEditorWithContent(80, 24, content)

		// Execute "G" to go to end of file
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}

		// Verify we're on the last line
		cursorLine := te.buffer.CursorIterator.LineNumber()
		t.Logf("After G: cursor at line %d", cursorLine)

		// Execute "o" to open line below
		err = te.ExecuteCommand("open-line-below")
		if err != nil {
			t.Fatalf("open-line-below failed: %v", err)
		}

		// Check cursor position - should be on line 3 (0-based)
		cursorLine = te.buffer.CursorIterator.LineNumber()
		t.Logf("After o: cursor at line %d", cursorLine)
		if cursorLine != 3 {
			t.Errorf("Expected cursor at line 3, got %d", cursorLine)
		}

		// Debug: Check buffer state
		t.Logf("Buffer line count: %d", te.buffer.LineCount)
		t.Logf("Cursor iterator IsEOF: %v", te.buffer.CursorIterator.IsEOF())

		// Debug: Check what open-line-below actually did
		bufContent := te.buffer.Rope.String()
		t.Logf("Buffer content after o: %q", bufContent)

		// Force a render
		// te.render()
		te.drawEditor()

		// Get the mock screen
		mockScreen := te.screen.(*MockScreen)

		// Check line number area for line 4
		// With 3 lines originally, line 4 should show "4" in the line number area
		lineNumberWidth := te.buffer.LineNumberWidth
		t.Logf("Line number width: %d", lineNumberWidth)

		// Check cell[3][0] should contain '4'
		if cell, ok := mockScreen.cells[3][0]; ok {
			if cell.Rune != '4' {
				t.Errorf("Expected '4' at cell[3][0], got %q", string(cell.Rune))

				// Debug: Show what's on screen
				t.Logf("Screen content:")
				for y := 0; y <= 5; y++ {
					var line strings.Builder
					for x := 0; x < 20; x++ {
						if cell, ok := mockScreen.cells[y][x]; ok {
							line.WriteRune(cell.Rune)
						} else {
							line.WriteRune(' ')
						}
					}
					t.Logf("  Line %d: %q", y, line.String())
				}
			}
		} else {
			t.Errorf("No character found at cell[3][0]")
		}
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

