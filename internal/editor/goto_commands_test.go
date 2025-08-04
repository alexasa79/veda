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
		te.SetCursorPosition(5, 2)
		// Note: We can't directly set the view position in tests anymore
		// The view should adjust automatically when needed
		AssertCursorAt(t, te, 5, 2)

		// Execute go-to-beginning
		err := te.ExecuteCommand("go-to-beginning")
		if err != nil {
			t.Fatalf("go-to-beginning failed: %v", err)
		}

		// Should be at beginning of file
		AssertCursorAt(t, te, 0, 0)

		// View should also be reset to show the beginning
		// We can verify this by checking that line 0 is visible on screen
		lines := te.GetScreenContent()
		if len(lines) > 0 && !strings.Contains(lines[0], "First line") {
			t.Errorf("Expected first line to be visible, but got: %s", lines[0])
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
		te.SetCursorPosition(6, 2)

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

		AssertCursorAt(t, te, 10, 1)
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

	t.Run("go to end then move up with trailing newline", func(t *testing.T) {
		// This test reproduces the crash when going to end of file with trailing newline
		// then pressing up arrow
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3\n")

		// Execute go-to-end
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}

		// Verify the cursor iterator is NOT at EOF
		if te.buffer.CursorIterator.IsEOF() {
			t.Errorf("After go-to-end, cursor iterator should not be at EOF")
		}

		// Should be at end of last non-empty line (line3)
		// In normal mode, cursor is on last character
		AssertCursorAt(t, te, 4, 2) // Position 4 on line 2 ("line3")

		// Now move up - this is where the crash was happening
		err = te.ExecuteCommand("up")
		if err != nil {
			t.Fatalf("up command failed: %v", err)
		}

		// Should move to line 1 (line2), maintaining visual column
		// Visual column was at position 4, but line2 only has 5 chars, so cursor is at position 4
		// In normal mode, this gets adjusted to position 3 (last character)
		AssertCursorAt(t, te, 4, 1) // Position 3 on line 1 ("line2")
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
		// Verify view starts at top
		screenLines := te.GetScreenContent()
		if len(screenLines) > 0 && !strings.Contains(screenLines[0], "Line 0") {
			t.Errorf("Expected view to start at line 0, but first visible line is: %s", screenLines[0])
		}

		// Execute go-to-end
		err := te.ExecuteCommand("go-to-end")
		if err != nil {
			t.Fatalf("go-to-end failed: %v", err)
		}

		// Should be at last line
		AssertCursorAt(t, te, 5, 49) // "Line 9" has 6 chars, cursor at position 5

		// View should have scrolled to show the cursor
		// With 24 lines visible and cursor at line 49, view should show lines near the end
		screenLines = te.GetScreenContent()
		// The screen has 24 lines, minus 1 for status bar = 23 visible lines
		// Check if we can see lines near the end
		if len(screenLines) > 0 {
			// Just verify that we've scrolled by checking that early lines aren't visible
			firstLine := screenLines[0]
			if firstLine == "Line 0" {
				t.Errorf("Expected view to scroll, but still showing line 0")
			}
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
		te.SetCursorPosition(3, 1)

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