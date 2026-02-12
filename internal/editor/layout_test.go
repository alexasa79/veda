package editor

import (
	"fmt"
	"strings"
	"testing"
)

// simulateResize changes the mock screen size and processes the resize event
// through the same code path as a real terminal resize (SIGWINCH → event → handleEvent → scroll → render).
func simulateResize(t *testing.T, te *TestEditor, width, height int) {
	t.Helper()
	te.mockScreen.SetSize(width, height)
	select {
	case ev := <-te.mockScreen.EventChannel():
		te.Editor.handleEvent(ev)
	default:
		t.Fatal("Expected resize event in channel after SetSize")
	}
	te.Editor.scroll()
	te.Editor.render()
}

func TestEditorResizeContentPreservation(t *testing.T) {
	t.Run("content preserved after shrink and grow", func(t *testing.T) {
		// Build 50 lines of content
		var lines []string
		for i := 1; i <= 50; i++ {
			lines = append(lines, fmt.Sprintf("Line %d content here", i))
		}
		content := strings.Join(lines, "\n")

		// 51 rows = 50 content rows + 1 status bar, so all 50 lines fit
		te := NewTestEditorWithContent(80, 51, content)

		// Initial render via the same path the main loop uses
		te.Editor.scroll()
		te.Editor.render()

		// Verify line 45 is rendered on screen row 44 (0-indexed)
		row44 := te.mockScreen.GetLine(44)
		if !strings.Contains(row44, "Line 45 content here") {
			t.Fatalf("Before resize: expected row 44 to contain 'Line 45 content here', got %q", row44)
		}

		// Shrink to 41 rows (40 content + 1 status bar)
		simulateResize(t, te, 80, 41)

		// Grow back to original size
		simulateResize(t, te, 80, 51)

		// Line 45 must be visible again at row 44
		row44Restored := te.mockScreen.GetLine(44)
		if !strings.Contains(row44Restored, "Line 45 content here") {
			t.Errorf("After grow back: expected row 44 to contain 'Line 45 content here', got %q", row44Restored)
		}

		// Spot-check several other lines survived the round-trip
		for _, tc := range []struct {
			row      int
			contains string
		}{
			{0, "Line 1 content here"},
			{9, "Line 10 content here"},
			{24, "Line 25 content here"},
			{39, "Line 40 content here"},
			{49, "Line 50 content here"},
		} {
			got := te.mockScreen.GetLine(tc.row)
			if !strings.Contains(got, tc.contains) {
				t.Errorf("After grow back: expected row %d to contain %q, got %q", tc.row, tc.contains, got)
			}
		}
	})
}

func TestEditorResize(t *testing.T) {
	t.Run("status bar position after resize", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World\nThis is a test")
		te.Editor.render()

		// Verify initial status bar position (line 23 for 24-line screen)
		statusLine := te.mockScreen.GetLine(23)
		if statusLine == "" {
			t.Errorf("Status bar not found at expected position 23")
		}

		// Shrink to 10 rows
		simulateResize(t, te, 80, 10)

		statusLine = te.mockScreen.GetLine(9)
		if statusLine == "" {
			t.Errorf("Status bar not found at position 9 after shrink")
		}

		// First line should still have content
		line0 := te.mockScreen.GetLine(0)
		if line0 == "" {
			t.Error("First line should have content after resize")
		}

		// Grow to 30 rows
		simulateResize(t, te, 80, 30)

		statusLine = te.mockScreen.GetLine(29)
		if statusLine == "" {
			t.Errorf("Status bar not found at position 29 after grow")
		}

		// No content below status bar
		for y := 30; y < 35; y++ {
			line := te.mockScreen.GetLine(y)
			if line != "" {
				t.Errorf("Found unexpected content below status bar at line %d: %s", y, line)
			}
		}
	})

	t.Run("error line position after resize", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Test content")
		te.Editor.errorManager.ShowError("Test error message")
		te.Editor.render()

		// Error line is one above status bar (line 22 for 24-line screen)
		errorLine := te.mockScreen.GetLine(22)
		if !strings.Contains(errorLine, "Test error message") {
			t.Errorf("Error line not found at position 22, got: %q", errorLine)
		}

		// Resize screen
		simulateResize(t, te, 80, 15)

		// Error line should now be at position 13 (15-2)
		errorLine = te.mockScreen.GetLine(13)
		if !strings.Contains(errorLine, "Test error message") {
			t.Errorf("Error line not found at position 13 after resize, got: %q", errorLine)
		}
	})

	t.Run("visible lines calculation after resize", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

		initialVisibleLines := te.Editor.getVisibleLines()
		if initialVisibleLines != 23 {
			t.Errorf("Initial visible lines should be 23, got %d", initialVisibleLines)
		}

		simulateResize(t, te, 80, 10)

		newVisibleLines := te.Editor.getVisibleLines()
		if newVisibleLines != 9 {
			t.Errorf("Visible lines after resize should be 9, got %d", newVisibleLines)
		}

		// With error showing, visible area should be smaller
		te.Editor.errorManager.ShowError("Error")
		te.Editor.render()
		visibleLinesWithError := te.Editor.getVisibleLines()
		if visibleLinesWithError != 8 {
			t.Errorf("Visible lines with error should be 8, got %d", visibleLinesWithError)
		}
	})

	t.Run("event-driven resize behavior", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 20, "Line 1\nLine 2\nLine 3")
		te.Editor.render()

		statusLine := te.mockScreen.GetLine(19)
		if statusLine == "" {
			t.Errorf("Initial status bar not found at position 19")
		}

		simulateResize(t, te, 100, 30)

		statusLine = te.mockScreen.GetLine(29)
		if statusLine == "" {
			t.Errorf("Status bar not found at position 29 after resize")
		}

		// No content beyond screen bounds
		for y := 30; y < 35; y++ {
			line := te.mockScreen.GetLine(y)
			if line != "" {
				t.Errorf("Found content beyond screen bounds at line %d: %s", y, line)
			}
		}
	})

	t.Run("screen buffer bounds checking", func(t *testing.T) {
		te := NewTestEditorWithContent(40, 10, "Test content")
		te.Editor.render()

		statusY := te.Editor.getStatusBarY()
		if statusY != 9 {
			t.Errorf("Expected status bar at Y=9, got Y=%d", statusY)
		}

		simulateResize(t, te, 120, 50)

		newStatusY := te.Editor.getStatusBarY()
		if newStatusY != 49 {
			t.Errorf("Expected status bar at Y=49 after resize, got Y=%d", newStatusY)
		}

		statusContent := te.mockScreen.GetLine(newStatusY)
		if statusContent == "" {
			t.Error("Status bar should be rendered at new position after resize")
		}

		width, height := te.mockScreen.Size()
		if width != 120 || height != 50 {
			t.Errorf("Screen size not updated correctly: got (%d,%d), want (120,50)", width, height)
		}
	})
}
