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
		if te.CurrentBuffer().LineNumberWidth == 0 {
			t.Errorf("Expected LineNumberWidth > 0, got %d", te.CurrentBuffer().LineNumberWidth)
		}
		t.Logf("LineNumberWidth: %d", te.CurrentBuffer().LineNumberWidth)

		// Check buffer line count
		t.Logf("Buffer LineCount: %d", te.CurrentBuffer().LineCount)

		// Debug: Check what's at position 0,0
		ch, _ := te.mockScreen.GetCell(0, 0)
		t.Logf("Character at (0,0): '%c' (rune value: %d)", ch, ch)
	})
}

func extractRenderedLineNumber(te *TestEditor, row int) string {
	digitsWidth := te.CurrentBuffer().LineNumberWidth - 3
	if digitsWidth < 1 {
		digitsWidth = 1
	}
	var builder strings.Builder
	for x := 0; x < digitsWidth; x++ {
		ch, _ := te.mockScreen.GetCell(x, row)
		if ch != ' ' {
			builder.WriteRune(ch)
		}
	}
	return strings.TrimSpace(builder.String())
}

func TestRelativeLineNumbersAroundCursor(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "a\nb\nc\nd")
	te.drawEditor()

	if got := extractRenderedLineNumber(te, 0); got != "1" {
		t.Fatalf("cursor line should show absolute number 1, got %q", got)
	}
	if got := extractRenderedLineNumber(te, 1); got != "1" {
		t.Fatalf("line below cursor should show relative number 1, got %q", got)
	}
	if got := extractRenderedLineNumber(te, 2); got != "2" {
		t.Fatalf("two lines below cursor should show relative number 2, got %q", got)
	}
}

func TestTerminalBuffersKeepAbsoluteLineNumbers(t *testing.T) {
	te := NewTestEditor(80, 10)
	term := NewTerminalEmulator(80, 10)
	term.Write([]byte("first\nsecond\nthird\n"))
	setupTerminalBufferForTest(te, 80, 10, term.Render(), term)
	te.drawEditor()

	if got := extractRenderedLineNumber(te, 0); got != "1" {
		t.Fatalf("terminal cursor line should show absolute number 1, got %q", got)
	}
	if got := extractRenderedLineNumber(te, 1); got != "2" {
		t.Fatalf("terminal line 2 should show absolute number 2, got %q", got)
	}
}
