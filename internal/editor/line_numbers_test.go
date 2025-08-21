package editor

import (
	"strings"
	"testing"
)

func TestLineNumbersRendering(t *testing.T) {
	t.Run("line numbers are rendered", func(t *testing.T) {
		// Create editor with multiple lines
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
		te := NewTestEditorWithContent(80, 24, content)

		// Force a render
		te.drawEditor()
		te.drawStatusBar()

		// Check if line numbers are present
		// Line numbers should appear at the beginning of each line
		foundLineNumbers := false
		for y := 0; y < 5; y++ {
			line := te.mockScreen.GetLine(y)
			// Only check first 10 chars if line is long enough
			checkLength := 10
			if len(line) < checkLength {
				checkLength = len(line)
			}
			trimmedLine := strings.TrimSpace(line[:checkLength])

			// Check if this line starts with a number
			if len(trimmedLine) > 0 && trimmedLine[0] >= '1' && trimmedLine[0] <= '5' {
				foundLineNumbers = true
				t.Logf("Found line number at row %d: %s", y, trimmedLine)
			}
		}

		if !foundLineNumbers {
			t.Error("Expected to find line numbers, but none were rendered")

			// Debug output
			t.Log("First 10 characters of each line:")
			for y := 0; y < 5; y++ {
				line := te.mockScreen.GetLine(y)
				if len(line) > 10 {
					line = line[:10]
				}
				t.Logf("Row %d: [%s]", y, line)
			}
		}

		// Also check LineNumberWidth
		if te.buffer.LineNumberWidth == 0 {
			t.Errorf("Expected LineNumberWidth > 0, got %d", te.buffer.LineNumberWidth)
		}
		t.Logf("LineNumberWidth: %d", te.buffer.LineNumberWidth)

		// Check buffer line count
		t.Logf("Buffer LineCount: %d", te.buffer.LineCount)

		// Debug: Check what's at position 0,0
		ch, _ := te.mockScreen.GetCell(0, 0)
		t.Logf("Character at (0,0): '%c' (rune value: %d)", ch, ch)
	})
}
