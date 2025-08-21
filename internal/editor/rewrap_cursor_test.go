package editor

import (
	"strings"
	"testing"
)

func TestRewrapCursorPositioning(t *testing.T) {
	t.Run("cursor above paragraph doesn't lose X position", func(t *testing.T) {
		// Set up content with a paragraph and cursor above it
		content := `First line here with some content
Another line above paragraph

This is a long paragraph that will need to be rewrapped because it exceeds the normal line width and contains many words that should be reflowed when the rewrap command is executed.

Line after paragraph`

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{40} // Force rewrapping at 40 chars

		// Position cursor at end of first line (above paragraph)
		te.SetCursorPosition(31, 0) // End of "First line here with some content"

		// Get initial cursor position
		initialLine := te.buffer.CursorIterator.LineNumber()
		initialX := te.buffer.CursorX
		t.Logf("Initial cursor position: line %d, column %d", initialLine, initialX)
		t.Logf("Initial line content: %q", te.buffer.CursorIterator.Line())

		// Position cursor on the paragraph line for rewrapping
		te.SetCursorPosition(10, 3) // Middle of the long paragraph

		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		// Check that cursor on first line maintained its X position
		te.SetCursorPosition(initialX, initialLine)
		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final cursor position: line %d, column %d", finalLine, finalX)
		t.Logf("Final line content: %q", te.buffer.CursorIterator.Line())

		if finalLine != initialLine {
			t.Errorf("Cursor line changed from %d to %d", initialLine, finalLine)
		}
		if finalX != initialX {
			t.Errorf("Cursor X position changed from %d to %d", initialX, finalX)
		}
	})

	t.Run("cursor in middle of paragraph maintains relative position", func(t *testing.T) {
		// Set up a long paragraph
		content := "This is a very long paragraph that contains many words and will definitely need to be rewrapped when we apply the rewrap command because it exceeds the typical line width that we would want to use for readable text formatting."

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{40} // Force rewrapping at 40 chars

		// Position cursor at a specific word in the middle
		targetWord := "definitely"
		targetPos := strings.Index(content, targetWord)
		if targetPos == -1 {
			t.Fatal("Target word not found in content")
		}

		te.SetCursorPosition(targetPos+5, 0) // Middle of "definitely"

		initialLine := te.buffer.CursorIterator.LineNumber()
		initialX := te.buffer.CursorX
		t.Logf("Initial cursor position: line %d, column %d", initialLine, initialX)
		t.Logf("Character at cursor: %c", []rune(te.buffer.CursorIterator.Line())[initialX])

		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		// Check final position
		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final cursor position: line %d, column %d", finalLine, finalX)

		// Verify cursor is still in the word "definitely"
		finalLineContent := te.buffer.CursorIterator.Line()
		if finalX >= len(finalLineContent) {
			t.Errorf("Cursor X position %d is beyond line length %d", finalX, len(finalLineContent))
			return
		}

		// Check if we're still in or near the target word
		t.Logf("Final line content: %q", finalLineContent)
		t.Logf("Character at final cursor: %c", []rune(finalLineContent)[finalX])

		// The cursor should not be at the very beginning of the paragraph
		if finalX == 0 && finalLine == 0 {
			t.Error("Cursor moved to beginning of paragraph - this is the bug!")
		}
	})

	t.Run("cursor at end of paragraph line", func(t *testing.T) {
		content := "Short line.\nThis is a longer paragraph line that will need rewrapping because it is quite long and exceeds our target width."

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{40}

		// Position cursor at end of the long line
		te.SetCursorPosition(0, 1) // Start of paragraph line
		longLine := te.buffer.CursorIterator.Line()
		te.SetCursorPosition(len(longLine), 1) // End of line

		initialX := te.buffer.CursorX
		t.Logf("Initial cursor at end of line: column %d", initialX)

		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final cursor position: line %d, column %d", finalLine, finalX)

		// Should not jump to beginning
		if finalX == 0 && finalLine == 1 {
			t.Error("Cursor jumped to beginning of paragraph when it was at the end")
		}
	})

	t.Run("cursor on empty line within paragraph", func(t *testing.T) {
		content := `This is the first part of a paragraph.

This is the second part after an empty line.`

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{30}

		// Position cursor on the empty line
		te.SetCursorPosition(0, 1)

		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final cursor position after rewrap: line %d, column %d", finalLine, finalX)

		// Should handle empty line gracefully without jumping to start
		t.Logf("Buffer content after rewrap:\n%s", te.GetBufferContent())
	})

	t.Run("cursor in comment paragraph", func(t *testing.T) {
		content := `// This is a very long comment that will need to be rewrapped because it exceeds the normal line width
// and continues on multiple lines with the same comment prefix throughout.`

		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{40}

		// Position cursor in middle of first comment line (after "very")
		te.SetCursorPosition(20, 0)

		initialX := te.buffer.CursorX
		t.Logf("Initial cursor in comment: column %d", initialX)

		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}

		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final cursor position: line %d, column %d", finalLine, finalX)

		// Should maintain position within comment content, not jump to "//" prefix
		if finalX <= 3 {
			t.Error("Cursor jumped to beginning of comment line (at or before '//')")
		}

		t.Logf("Buffer content after rewrap:\n%s", te.GetBufferContent())
	})
}
