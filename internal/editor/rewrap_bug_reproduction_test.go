package editor

import (
	"testing"
)

func TestRewrapBugReproduction(t *testing.T) {
	t.Run("cursor offset calculation bug with comment prefix", func(t *testing.T) {
		// This tests the specific scenario from lines 455-464 in rewrap.go
		// where cursor offset calculation for Python comments might fail
		content := `# This is a Python comment that is quite long and needs to be rewrapped
# because it exceeds our target line width for readable formatting`
		
		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{30}
		
		// Position cursor right after "# " in the second line - this is tricky for offset calculation
		te.SetCursorPosition(2, 1) // Just after "# " on second line
		
		initialLine := te.buffer.CursorIterator.LineNumber()
		initialX := te.buffer.CursorX
		t.Logf("Initial: line %d, column %d", initialLine, initialX)
		
		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}
		
		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final: line %d, column %d", finalLine, finalX)
		t.Logf("Buffer:\n%s", te.GetBufferContent())
		
		// The bug would be if cursor jumps to line 0, column 0
		if finalLine == 0 && finalX == 0 {
			t.Error("BUG REPRODUCED: Cursor jumped to beginning of paragraph")
		}
		
		// Also check if it jumped to just after "# " on first line
		if finalLine == 0 && finalX == 2 {
			t.Error("BUG REPRODUCED: Cursor jumped to beginning of comment content")
		}
	})
	
	t.Run("cursorOffsetInParagraph goes negative", func(t *testing.T) {
		// Try to create a scenario where the cursor offset calculation fails
		// This could happen if the cursor is positioned in a way that confuses the algorithm
		content := `
		
This is a paragraph with weird indentation    and spacing that might cause offset calculation issues.`
		
		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{25}
		
		// Position cursor on one of the empty lines
		te.SetCursorPosition(4, 1) // Some random position on second line
		
		initialLine := te.buffer.CursorIterator.LineNumber()
		initialX := te.buffer.CursorX
		t.Logf("Initial: line %d, column %d", initialLine, initialX)
		t.Logf("Initial line content: %q", te.buffer.CursorIterator.Line())
		
		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}
		
		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final: line %d, column %d", finalLine, finalX)
		t.Logf("Final line content: %q", te.buffer.CursorIterator.Line())
		
		if finalLine == 0 && finalX == 0 {
			t.Error("BUG REPRODUCED: Cursor jumped to very beginning")
		}
	})
	
	t.Run("cursor above paragraph loses X position", func(t *testing.T) {
		// Test the specific scenario from lines 702-705 where cursor above paragraph
		// might lose its X position
		content := `Some text with cursor positioned here          <- cursor at specific position
		
This is a paragraph that needs rewrapping because it is quite long and exceeds our target width significantly.`
		
		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{40}
		
		// Position cursor at a specific X position on the first line
		te.SetCursorPosition(45, 0) // After "positioned here"
		
		initialLine := te.buffer.CursorIterator.LineNumber()
		initialX := te.buffer.CursorX
		t.Logf("Initial: line %d, column %d", initialLine, initialX)
		
		// Position cursor on the paragraph for rewrapping
		te.SetCursorPosition(20, 2) // Middle of paragraph
		
		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}
		
		// Now check if the first line cursor position was preserved
		te.SetCursorPosition(te.buffer.CursorX, 0) // Move back to first line
		
		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final: line %d, column %d", finalLine, finalX)
		
		// Check if cursor jumped to beginning of first line (this would be the bug)
		if finalLine == 0 && finalX == 0 && initialX > 0 {
			t.Error("BUG REPRODUCED: Cursor on line above paragraph lost its X position")
		}
	})
	
	t.Run("cursor position with remainingOffset negative fallback", func(t *testing.T) {
		// This targets the specific code path in lines 674-681 where remainingOffset < 0
		content := "a b c d e"
		
		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{5} // Very tight wrapping
		
		// Position cursor at the very end
		te.SetCursorPosition(len(content), 0)
		
		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}
		
		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final: line %d, column %d", finalLine, finalX)
		t.Logf("Buffer:\n%s", te.GetBufferContent())
		
		// The fallback code sets cursor to start of next line with just indentation + prefix
		// If this happens incorrectly, it might put cursor at beginning
		if finalLine == 0 && finalX == 0 {
			t.Error("BUG REPRODUCED: Negative remainingOffset caused cursor to jump to start")
		}
	})
	
	t.Run("empty paragraph text causes default positioning", func(t *testing.T) {
		// If paragraphText.String() ends up empty, cursor positioning might fail
		content := `
		
		
`
		
		te := NewTestEditorWithContent(80, 24, content)
		te.config.VerticalRulers = []int{40}
		
		// Position cursor on middle empty line
		te.SetCursorPosition(2, 1)
		
		// Execute rewrap
		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap failed: %v", err)
		}
		
		finalLine := te.buffer.CursorIterator.LineNumber()
		finalX := te.buffer.CursorX
		t.Logf("Final position with empty paragraph: line %d, column %d", finalLine, finalX)
		
		// This scenario might cause unpredictable cursor behavior
		t.Logf("Buffer after rewrapping empty paragraph:\n%q", te.GetBufferContent())
	})
}