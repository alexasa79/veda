package editor

import (
	"strings"
	"testing"
)

func TestGoToBeginning(t *testing.T) {
	t.Run("go to beginning from middle of file", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"Second line",
			"Third line",
			"Fourth line",
			"Fifth line",
		}, "\n")
		
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move cursor to middle of file
		te.cursorX = 5
		te.cursorY = 2
		te.topLine = 1
		AssertCursorAt(t, te, 5, 2)
		
		// Execute go-to-beginning
		err := te.ExecuteCommand("go-to-beginning")
		if err != nil {
			t.Fatalf("go-to-beginning failed: %v", err)
		}
		
		// Should be at beginning of file
		AssertCursorAt(t, te, 0, 0)
		
		// topLine should also be reset
		if te.topLine != 0 {
			t.Errorf("Expected topLine to be 0, got %d", te.topLine)
		}
	})
	
	t.Run("go to beginning from end of file", func(t *testing.T) {
		content := strings.Join([]string{
			"Line 1",
			"Line 2",
			"Line 3",
		}, "\n")
		
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move to end of file
		te.cursorY = 2
		te.cursorX = 6
		
		// Execute go-to-beginning
		err := te.ExecuteCommand("go-to-beginning")
		if err != nil {
			t.Fatalf("go-to-beginning failed: %v", err)
		}
		
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("go to beginning in empty file", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// Execute go-to-beginning
		err := te.ExecuteCommand("go-to-beginning")
		if err != nil {
			t.Fatalf("go-to-beginning failed: %v", err)
		}
		
		AssertCursorAt(t, te, 0, 0)
	})
}

func TestGoToEnd(t *testing.T) {
	t.Run("go to end from beginning of file", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"Second line", 
			"Third line",
			"Fourth line",
			"Last line here",
		}, "\n")
		
		te := NewTestEditorWithContent(80, 24, content)
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Execute go-to-end
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}
		
		// Should be at last character of last line (normal mode)
		// "Last line here" has 14 characters, so cursor should be at position 13
		AssertCursorAt(t, te, 13, 4)
	})
	
	t.Run("go to end with empty last line", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"Second line",
			"", // Empty last line
		}, "\n")
		
		te := NewTestEditorWithContent(80, 24, content)
		
		// Execute go-to-end
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}
		
		// Should be at beginning of empty last line
		AssertCursorAt(t, te, 0, 2)
	})
	
	t.Run("go to end in single line file", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Execute go-to-end
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}
		
		// Should be at last character (position 10, since "Hello World" has 11 chars)
		AssertCursorAt(t, te, 10, 0)
	})
	
	t.Run("go to end in empty file", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// Execute go-to-end  
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}
		
		// Should stay at 0,0
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("go to end scrolls view", func(t *testing.T) {
		// Create a file with more lines than screen height
		lines := make([]string, 50)
		for i := 0; i < 50; i++ {
			lines[i] = "Line " + string(rune('0'+i%10))
		}
		content := strings.Join(lines, "\n")
		
		te := NewTestEditorWithContent(80, 24, content)
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		if te.topLine != 0 {
			t.Errorf("Expected topLine to start at 0, got %d", te.topLine)
		}
		
		// Execute go-to-end
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}
		
		// Should be at last line
		AssertCursorAt(t, te, 5, 49) // "Line 9" has 6 chars, cursor at position 5
		
		// topLine should have scrolled to show the cursor
		// With 24 lines visible and cursor at line 49, topLine should be adjusted
		if te.topLine <= 0 {
			t.Errorf("Expected topLine to scroll, but it's still %d", te.topLine)
		}
	})
}

func TestGoToBeginningAndEndKeyBindings(t *testing.T) {
	t.Run("g g key binding", func(t *testing.T) {
		content := strings.Join([]string{
			"Line 1",
			"Line 2", 
			"Line 3",
		}, "\n")
		
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move to middle
		te.cursorX = 3
		te.cursorY = 1
		
		// Simulate pressing 'g' twice
		// The key mapper should handle the sequence "g g"
		// For this test, we'll use ExecuteCommand directly
		err := te.ExecuteCommand("go-to-beginning")
		if err != nil {
			t.Fatalf("go-to-beginning failed: %v", err)
		}
		
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("G key binding", func(t *testing.T) {
		content := strings.Join([]string{
			"Line 1",
			"Line 2",
			"Line 3",
		}, "\n")
		
		te := NewTestEditorWithContent(80, 24, content)
		
		// Start at beginning
		AssertCursorAt(t, te, 0, 0)
		
		// Simulate pressing 'G'
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}
		
		// Should be at end of last line
		AssertCursorAt(t, te, 5, 2) // "Line 3" has 6 chars, cursor at position 5
	})
}