package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

// Test that virtual cursors are visible when appending after the end of line
func TestVirtualCursorVisibleWhenAppendingAfterEndOfLine(t *testing.T) {
	t.Run("virtual cursor visible with 'a' command after end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello\nWorld")

		// Move to end of first line
		te.TypeKeys("$")

		// Enter cursor mode
		te.TypeKeys("m")

		// Add virtual cursor at end of "Hello" (5,0)
		te.PressKey(events.KeyEnter, 0)

		// Move down and add another virtual cursor at end of "World" (5,1)
		te.TypeKeys("$")
		te.PressKey(events.KeyEnter, 0)

		// Exit cursor mode
		te.TypeKeys("m")

		// Use 'a' command to append after the end of line
		// This should position cursors at (6,0) and (6,1) - after the last characters
		te.TypeKeys("a")

		// Verify we're in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode, got %v", te.mode)
		}

		// Verify virtual cursors exist
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors, got %d", len(virtualCursors))
		}

		// Force a render to update the screen
		te.renderPane()

		// Get the virtual cursor style from theme
		expectedStyle := te.config.Theme.GetStyle(te.config.Theme.VirtualCursor)

		// Check that virtual cursors are rendered with correct style
		// First virtual cursor should be at position (6,0) on screen
		// Since line numbers take up space, the actual screen position will be offset
		lineNumberWidth := te.CurrentBuffer().LineNumberWidth

		// Debug: Print virtual cursor positions and primary cursor position
		for i, vc := range virtualCursors {
			t.Logf("Virtual cursor %d: X=%d, Y=%d", i, vc.X, vc.Y)
		}
		primaryX, primaryY := te.GetCursorPosition()
		t.Logf("Primary cursor: X=%d, Y=%d", primaryX, primaryY)

		// Debug: Print line number width and screen content
		t.Logf("Line number width: %d", lineNumberWidth)
		t.Logf("Current mode: %v", te.mode)
		screenContent := te.GetScreenContent()
		t.Logf("Line 0 content: %q", screenContent[0])
		if len(screenContent) > 1 {
			t.Logf("Line 1 content: %q", screenContent[1])
		}

		// Check what the current line style is
		currentLineStyle := te.config.Theme.GetStyle(te.config.Theme.CurrentLine)
		_, currentLineBg, _ := currentLineStyle.Decompose()
		t.Logf("Current line background: %v", currentLineBg)

		// Check what normal background is
		normalStyle := te.config.Theme.GetStyle(te.config.Theme.Normal)
		_, normalBg, _ := normalStyle.Decompose()
		t.Logf("Normal background: %v", normalBg)

		// Get the virtual cursor style from theme
		_, expectedBg, _ := expectedStyle.Decompose()

		// Check virtual cursor at line 0, using the first virtual cursor's position
		if len(virtualCursors) > 0 {
			vc := virtualCursors[0]

			// Check all positions on line 0 to see if any has the virtual cursor background
			found := false
			width, _ := te.screen.Size()
			for x := lineNumberWidth; x < width; x++ {
				_, style := te.mockScreen.GetCell(x, vc.Y)
				_, actualBg, _ := style.Decompose()

				if actualBg == expectedBg {
					found = true
					t.Logf("Found virtual cursor style at screen position (%d, %d)", x, vc.Y)
					break
				}
			}

			if !found {
				// Debug: show what we actually have on the line
				t.Logf("No virtual cursor style found on line %d. Styles found:", vc.Y)
				maxX := width
				if lineNumberWidth+20 < width {
					maxX = lineNumberWidth + 20
				}
				for x := lineNumberWidth; x < maxX; x++ {
					char, style := te.mockScreen.GetCell(x, vc.Y)
					_, actualBg, _ := style.Decompose()
					t.Logf("  Position (%d, %d): %q (bg: %v)", x, vc.Y, char, actualBg)
				}
				t.Errorf("First virtual cursor not rendered with correct style anywhere on the line")
			}
		}

		// Check second virtual cursor at line 1
		if len(virtualCursors) > 1 {
			vc2 := virtualCursors[1]

			// Check all positions on line 1 to see if any has the virtual cursor background
			found2 := false
			width, _ := te.screen.Size()
			for x := lineNumberWidth; x < width; x++ {
				_, style := te.mockScreen.GetCell(x, vc2.Y)
				_, actualBg, _ := style.Decompose()

				if actualBg == expectedBg {
					found2 = true
					t.Logf("Found virtual cursor style at screen position (%d, %d)", x, vc2.Y)
					break
				}
			}

			if !found2 {
				t.Errorf("Second virtual cursor not rendered with correct style anywhere on line %d", vc2.Y)
			}
		}
	})

	t.Run("virtual cursor visible with 'A' command at end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello\nWorld")

		// Enter cursor mode at beginning
		te.TypeKeys("m")

		// Add virtual cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)

		// Move down and add another virtual cursor at (0,1)
		te.PressKey(events.KeyEnter, 0)

		// Exit cursor mode
		te.TypeKeys("m")

		// Use 'A' command to append at the end of each line
		// This should position cursors at (5,0) and (5,1) - after the last characters
		te.TypeKeys("A")

		// Verify we're in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode, got %v", te.mode)
		}

		// Verify virtual cursors exist
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors, got %d", len(virtualCursors))
		}

		// Force a render to update the screen
		te.renderPane()

		// Get the virtual cursor style from theme
		expectedStyle := te.config.Theme.GetStyle(te.config.Theme.VirtualCursor)

		// Check that virtual cursors are rendered with correct style
		lineNumberWidth := te.CurrentBuffer().LineNumberWidth

		// Check virtual cursor at line 0, position after "Hello" (which would be column 5 in line, plus line number width)
		screenX := lineNumberWidth + 5 // 5 is length of "Hello"
		screenY := 0

		_, style := te.mockScreen.GetCell(screenX, screenY)

		// Compare background colors - virtual cursors should have the VirtualCursor background
		_, expectedBg, _ := expectedStyle.Decompose()
		_, actualBg, _ := style.Decompose()

		if actualBg != expectedBg {
			t.Errorf("First virtual cursor not rendered with correct style. Expected background %v, got %v", expectedBg, actualBg)
		}

		// Check second virtual cursor at line 1, position after "World"
		screenX2 := lineNumberWidth + 5 // 5 is length of "World"
		screenY2 := 1

		_, style2 := te.mockScreen.GetCell(screenX2, screenY2)
		_, actualBg2, _ := style2.Decompose()

		if actualBg2 != expectedBg {
			t.Errorf("Second virtual cursor not rendered with correct style. Expected background %v, got %v", expectedBg, actualBg2)
		}
	})
}
