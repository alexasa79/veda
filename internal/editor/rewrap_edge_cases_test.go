package editor

import (
	"strings"
	"testing"
)

// Helper function to get character at cursor position
func (te *TestEditor) getCharAtCursor() rune {
	line := te.buffer.CursorIterator.Line()
	if te.buffer.CursorX >= len([]rune(line)) {
		return 0 // Beyond end of line
	}
	runes := []rune(line)
	if te.buffer.CursorX < len(runes) {
		return runes[te.buffer.CursorX]
	}
	return 0
}

func TestRewrapEdgeCases(t *testing.T) {
	t.Run("cursor calculation overflow causes jump to start", func(t *testing.T) {
		// This test tries to reproduce cursor offset calculation errors
		content := "a b c d e f g h i j k l m n o p q r s t u v w x y z"

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{10} // Very short lines to force many breaks

		// Position cursor near the end
		te.SetCursorPosition(45, 0) // Near "v"

		initialChar := te.getCharAtCursor()
		t.Logf("Initial character at cursor: %c", initialChar)

		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		finalChar := te.getCharAtCursor()

		t.Logf("Final position: line %d, column %d", finalLine, finalX)
		t.Logf("Final character at cursor: %c", finalChar)
		t.Logf("Buffer after rewrap:\n%s", te.GetBufferContent())

		// Check if cursor jumped to beginning
		if finalLine == 0 && finalX == 0 {
			t.Error("BUG REPRODUCED: Cursor jumped to beginning of paragraph")
		}

		// Ideally, we should still be at or near the same character
		if initialChar != ' ' && finalChar != initialChar {
			t.Logf("Warning: Cursor moved from character '%c' to '%c'", initialChar, finalChar)
		}
	})

	t.Run("negative cursor offset causes jump", func(t *testing.T) {
		// Try to reproduce negative remainingOffset scenario
		content := `
This is a paragraph with some empty lines and weird spacing that might confuse the cursor offset calculation when we try to rewrap it.`

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{25}

		// Position cursor on the empty line at the start
		te.SetCursorPosition(0, 0)

		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX

		t.Logf("Final position after rewrap empty line: line %d, column %d", finalLine, finalX)
		t.Logf("Buffer content:\n%s", te.GetBufferContent())
	})

	t.Run("cursor beyond paragraph end", func(t *testing.T) {
		content := "Short paragraph."

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{50} // Wide enough to not need rewrapping normally

		// Position cursor at end of line
		te.SetCursorPosition(len(content), 0)

		initialX := te.buffer.CursorX
		t.Logf("Initial cursor at position: %d", initialX)

		// Execute rewrap (even though it shouldn't change anything)
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX

		t.Logf("Final position: line %d, column %d", finalLine, finalX)

		// For a short paragraph, cursor should stay roughly in place
		if finalX == 0 && initialX > 0 {
			t.Error("Cursor unexpectedly moved to beginning")
		}
	})

	t.Run("cursor in unicode text", func(t *testing.T) {
		content := "This párágráph cõntáins ūnïcōdé chárāctërs thāt might cáūsé íssūës with bÿtē ōffsét cálcūlātiōns whën rëwrāppïng."

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{30}

		// Position cursor on a unicode character
		unicodePos := strings.Index(content, "ūnïcōdé")
		te.SetCursorPosition(unicodePos+3, 0) // Middle of "ūnïcōdé"

		initialChar := te.getCharAtCursor()
		t.Logf("Initial unicode character at cursor: %c", initialChar)

		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		finalChar := te.getCharAtCursor()

		t.Logf("Final position: line %d, column %d", finalLine, finalX)
		t.Logf("Final character: %c", finalChar)
		t.Logf("Buffer after rewrap:\n%s", te.GetBufferContent())

		// Check for cursor jump
		if finalLine == 0 && finalX == 0 {
			t.Error("BUG: Cursor jumped to beginning when handling unicode text")
		}
	})

	t.Run("multiple consecutive rewraps", func(t *testing.T) {
		// Sometimes the bug might only appear after multiple rewraps
		content := "This is a medium length paragraph that will be rewrapped multiple times to test if cursor position degrades over multiple operations."

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{30, 40, 50} // Multiple rulers for cycling

		// Position cursor in middle
		te.SetCursorPosition(30, 0)
		initialChar := te.getCharAtCursor()
		t.Logf("Initial character: %c", initialChar)

		// Execute multiple rewraps
		for i := 0; i < 5; i++ {
			err := te.ExecuteCommand("rewrap-paragraph")
			if err != nil {
				t.Fatalf("Rewrap %d failed: %v", i+1, err)
			}

			currentLine := te.buffer.CursorIterator.LineNumber()
			currentX := te.buffer.CursorX
			currentChar := te.getCharAtCursor()

			t.Logf("After rewrap %d: line %d, column %d, char '%c'", i+1, currentLine, currentX, currentChar)

			if currentLine == 0 && currentX == 0 {
				t.Errorf("BUG: Cursor jumped to beginning after rewrap %d", i+1)
				break
			}
		}

		t.Logf("Final buffer:\n%s", te.GetBufferContent())
	})
}
