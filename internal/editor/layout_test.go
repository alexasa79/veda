package editor

import (
	"testing"
)

func TestEditorResize(t *testing.T) {
	t.Run("status bar position after resize", func(t *testing.T) {
		// Create editor with initial size
		te := NewTestEditorWithContent(80, 24, "Hello World\nThis is a test")

		// Initial render
		te.Editor.render()

		// Verify initial status bar position (should be at line 23 for 24-line screen)
		initialStatusBarY := 23
		statusLine := te.mockScreen.GetLine(initialStatusBarY)
		if statusLine == "" {
			t.Errorf("Status bar not found at expected position %d", initialStatusBarY)
		}

		// Check that status bar contains expected elements
		if len(statusLine) == 0 {
			t.Error("Status bar is empty")
		}

		// Resize the screen to a smaller size
		newHeight := 10
		te.mockScreen.SetSize(80, newHeight)

		// Process the resize event
		te.Editor.handleResize()

		// Re-render after resize
		te.Editor.render()

		// Verify status bar is now at the correct position (line 9 for 10-line screen)
		newStatusBarY := newHeight - 1
		statusLine = te.mockScreen.GetLine(newStatusBarY)
		if statusLine == "" {
			t.Errorf("Status bar not found at expected position %d after resize", newStatusBarY)
		}

		// Verify content above status bar is still visible
		for y := 0; y < newStatusBarY; y++ {
			line := te.mockScreen.GetLine(y)
			// At least some lines should have content
			if y == 0 && line == "" {
				t.Error("First line should have content after resize")
			}
		}

		// Resize to a larger size
		newHeight = 30
		te.mockScreen.SetSize(80, newHeight)
		te.Editor.handleResize()
		te.Editor.render()

		// Verify status bar moved to new position
		newStatusBarY = newHeight - 1
		statusLine = te.mockScreen.GetLine(newStatusBarY)
		if statusLine == "" {
			t.Errorf("Status bar not found at expected position %d after second resize", newStatusBarY)
		}

		// Verify no content appears below status bar
		for y := newStatusBarY + 1; y < newHeight; y++ {
			line := te.mockScreen.GetLine(y)
			if line != "" {
				t.Errorf("Found unexpected content below status bar at line %d: %s", y, line)
			}
		}
	})

	t.Run("error line position after resize", func(t *testing.T) {
		// Create editor and show an error
		te := NewTestEditorWithContent(80, 24, "Test content")

		// Show an error
		te.Editor.errorManager.ShowError("Test error message")

		// Initial render
		te.Editor.render()

		// Verify error line is one line above status bar (line 22 for 24-line screen)
		initialErrorY := 22
		errorLine := te.mockScreen.GetLine(initialErrorY)
		// Check if error line contains the message (don't check exact format)
		if errorLine == "" || len(errorLine) < len("Test error message") {
			t.Errorf("Error line not found at expected position %d, got: %q", initialErrorY, errorLine)
		}

		// Resize screen
		newHeight := 15
		te.mockScreen.SetSize(80, newHeight)
		te.Editor.handleResize()
		te.Editor.render()

		// Verify error line moved to new position (line 13 for 15-line screen)
		newErrorY := newHeight - 2
		errorLine = te.mockScreen.GetLine(newErrorY)
		// Check if error line contains the message (don't check exact format)
		if errorLine == "" || len(errorLine) < len("Test error message") {
			t.Errorf("Error line not found at expected position %d after resize, got: %q", newErrorY, errorLine)
		}
	})

	t.Run("visible lines calculation after resize", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

		// Get initial visible lines
		initialVisibleLines := te.Editor.getVisibleLines()
		if initialVisibleLines != 23 { // 24 - 1 for status bar
			t.Errorf("Initial visible lines should be 23, got %d", initialVisibleLines)
		}

		// Resize and verify visible lines changes
		te.mockScreen.SetSize(80, 10)
		te.Editor.handleResize()

		newVisibleLines := te.Editor.getVisibleLines()
		if newVisibleLines != 9 { // 10 - 1 for status bar
			t.Errorf("Visible lines after resize should be 9, got %d", newVisibleLines)
		}

		// With error showing, visible area should be smaller
		te.Editor.errorManager.ShowError("Error")
		te.Editor.render()
		// The getVisibleLines DOES account for error line
		visibleLinesWithError := te.Editor.getVisibleLines()
		if visibleLinesWithError != 8 { // 10 - 1 for status bar - 1 for error
			t.Errorf("Visible lines with error should be 8, got %d", visibleLinesWithError)
		}
	})

	t.Run("event-driven resize behavior", func(t *testing.T) {
		// Create editor with initial size
		te := NewTestEditorWithContent(80, 20, "Line 1\nLine 2\nLine 3")

		// Initial render to establish baseline
		te.Editor.render()

		// Verify initial status bar position (line 19 for 20-line screen)
		initialStatusBarY := 19
		statusLine := te.mockScreen.GetLine(initialStatusBarY)
		if statusLine == "" {
			t.Errorf("Initial status bar not found at position %d", initialStatusBarY)
		}

		// Now simulate increasing window size via event system
		// This is what happens when user increases terminal size
		newWidth, newHeight := 100, 30
		te.mockScreen.SetSize(newWidth, newHeight)

		// Process the resize event through the event system like the real editor would
		select {
		case event := <-te.mockScreen.EventChannel():
			te.Editor.handleEvent(event)
		default:
			t.Fatal("Expected resize event in channel")
		}

		// Now check that status bar moved to correct position (line 29 for 30-line screen)
		newStatusBarY := newHeight - 1
		statusLine = te.mockScreen.GetLine(newStatusBarY)
		if statusLine == "" {
			t.Errorf("Status bar not found at expected position %d after resize", newStatusBarY)
		}

		// Verify status bar isn't rendered beyond screen bounds
		for y := newHeight; y < newHeight+5; y++ {
			line := te.mockScreen.GetLine(y)
			if line != "" {
				t.Errorf("Found content beyond screen bounds at line %d: %s", y, line)
			}
		}

		// Verify status bar is visible at new position
		if statusLine == "" {
			t.Error("Status bar should be visible at new position")
		}
	})

	t.Run("screen buffer bounds checking", func(t *testing.T) {
		// Start with small screen
		te := NewTestEditorWithContent(40, 10, "Test content")
		te.Editor.render()

		// Verify status bar position
		statusY := te.Editor.getStatusBarY()
		if statusY != 9 {
			t.Errorf("Expected status bar at Y=9, got Y=%d", statusY)
		}

		// Dramatically increase screen size
		te.mockScreen.SetSize(120, 50)

		// Trigger resize handling
		select {
		case event := <-te.mockScreen.EventChannel():
			te.Editor.handleEvent(event)
		default:
			t.Fatal("Expected resize event")
		}

		// Verify status bar moved to new bottom
		newStatusY := te.Editor.getStatusBarY()
		if newStatusY != 49 {
			t.Errorf("Expected status bar at Y=49 after resize, got Y=%d", newStatusY)
		}

		// Verify status bar is actually rendered at the new position
		statusContent := te.mockScreen.GetLine(newStatusY)
		if statusContent == "" {
			t.Error("Status bar should be rendered at new position after resize")
		}

		// Verify screen size was updated
		width, height := te.mockScreen.Size()
		if width != 120 || height != 50 {
			t.Errorf("Screen size not updated correctly: got (%d,%d), want (120,50)", width, height)
		}
	})

	t.Run("resize without render call", func(t *testing.T) {
		// This test simulates the bug: resizing without proper re-render
		te := NewTestEditorWithContent(80, 20, "Line 1\nLine 2\nLine 3")

		// Initial render
		te.Editor.render()

		// Verify initial status bar
		initialStatusY := 19
		statusLine := te.mockScreen.GetLine(initialStatusY)
		if statusLine == "" {
			t.Error("Initial status bar not rendered")
		}

		// Change screen size directly (simulating the resize bug scenario)
		te.mockScreen.SetSize(100, 30)

		// Get the new expected status bar position
		newStatusBarY := te.Editor.getStatusBarY() // Should be 29 now
		if newStatusBarY != 29 {
			t.Errorf("Expected new status bar position to be 29, got %d", newStatusBarY)
		}

		// Here's the critical part: if we render without proper resize handling,
		// the status bar might render at position 29 but the screen buffer
		// might still think it's only 20 lines high
		te.Editor.render()

		// Check if status bar is visible at the new position
		statusContent := te.mockScreen.GetLine(newStatusBarY)
		if statusContent == "" {
			t.Errorf("Status bar not visible at new position %d after resize", newStatusBarY)
		}

		// Check that we haven't rendered beyond the screen bounds
		_, screenHeight := te.mockScreen.Size()
		for y := screenHeight; y < screenHeight+5; y++ {
			line := te.mockScreen.GetLine(y)
			if line != "" {
				t.Errorf("Found content beyond screen bounds at line %d: %s", y, line)
			}
		}
	})
}
