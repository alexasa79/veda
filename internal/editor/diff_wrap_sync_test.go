package editor

import (
	"strings"
	"testing"

	"veda/internal/buffer"
	"veda/internal/config"
)

// TestDiffViewLineWrapSync tests that long lines wrap correctly in diff view
// and both sides stay in sync. The bug is that lines on the right side don't
// wrap when they're too wide for the window, causing left and right to go
// out of sync.
func TestDiffViewLineWrapSync(t *testing.T) {
	// Create a test editor with 80 columns - in diff view each window gets ~40 columns
	te := NewTestEditorWithContent(80, 24, "")

	// Create content where the right side has a very long line that should wrap
	// The left side has a short line, right side has a line longer than 40 chars
	shortLine := "short line"
	longLine := "this is a very long line that should definitely wrap when the window is only about 40 characters wide"

	// In a real diff, we might have:
	// Left (original):  short line
	// Right (modified): very long line that needs wrapping
	originalContent := shortLine
	modifiedContent := longLine

	// Compute diff
	diffResult := computeDiff(originalContent, modifiedContent)

	// Create aligned buffer content
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)

	t.Logf("Left (HEAD) content: %q", alignedHeadContent)
	t.Logf("Right (current) content: %q", alignedCurrentContent)
	t.Logf("Left lines: %d, Right lines: %d", len(diffResult.LeftLines), len(diffResult.RightLines))

	// Create diff buffers
	headBufferName := "test.py@HEAD"
	currentBufferName := "test.py@current"

	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, currentBufferName, config.IndentTabs, 4)

	// Store buffers and diff line info
	te.buffers[headBufferName] = headBuffer
	te.buffers[currentBufferName] = currentBuffer
	te.diffLineInfo[headBuffer] = diffResult.LeftLines
	te.diffLineInfo[currentBuffer] = diffResult.RightLines

	// Set up windows for side-by-side diff view
	te.createSecondWindow()
	te.windows[0].setActiveBuffer(headBuffer)    // HEAD on left
	te.windows[1].setActiveBuffer(currentBuffer) // Current on right
	te.activeWindow = 1
	te.switchToBuffer(currentBuffer)
	// Recompute layout now that diff buffers are assigned - this triggers diff mode detection
	te.recomputeWindowsLayout()

	// Log window dimensions
	t.Logf("Window 0 (left): width=%d, height=%d", te.windows[0].Width, te.windows[0].Height)
	t.Logf("Window 1 (right): width=%d, height=%d", te.windows[1].Width, te.windows[1].Height)

	// The right window should have width around 40 (half of 80)
	rightWindowWidth := te.windows[1].Width
	if rightWindowWidth >= 80 {
		t.Fatalf("Right window width should be less than full screen width, got %d", rightWindowWidth)
	}

	// The long line is longer than the window width, so it should wrap
	if len(longLine) <= rightWindowWidth {
		t.Skipf("Long line (%d chars) fits in window width (%d), adjust test", len(longLine), rightWindowWidth)
	}

	t.Logf("Long line length: %d, right window width: %d", len(longLine), rightWindowWidth)

	// Calculate wrapped lines for the right (current) buffer
	// This is where the bug manifests - it uses screen width instead of window width
	wrappedLines := te.CalculateWrappedLines()

	t.Logf("Number of wrapped lines: %d", len(wrappedLines))
	for i, wl := range wrappedLines {
		t.Logf("  Wrapped line %d: logical=%d, visual=%d, start=%d, end=%d, text=%q",
			i, wl.LogicalLineNum, wl.VisualLineNum, wl.StartChar, wl.EndChar, wl.Text)
	}

	// The long line should be wrapped into multiple visual lines
	// Each wrapped segment should fit within the window width (minus gutter)
	gutterWidth := te.CurrentBuffer().LineNumberWidth
	availableWidth := rightWindowWidth - gutterWidth
	t.Logf("Available text width (window - gutter): %d - %d = %d", rightWindowWidth, gutterWidth, availableWidth)

	// Check that each wrapped line segment fits within the available window width
	for i, wl := range wrappedLines {
		if len(wl.Text) > availableWidth {
			t.Errorf("BUG REPRODUCED: Wrapped line %d has %d chars but window only has %d usable columns",
				i, len(wl.Text), availableWidth)
			t.Errorf("CalculateWrappedLines uses screen width instead of window width")
			t.Errorf("Line content: %q", wl.Text)
		}
	}

	// Now check that both sides have the same number of visual lines
	// Switch to left buffer and calculate its wrapped lines
	te.activeWindow = 0
	te.switchToBuffer(headBuffer)
	leftWrappedLines := te.CalculateWrappedLines()

	t.Logf("Left buffer wrapped lines: %d", len(leftWrappedLines))
	t.Logf("Right buffer wrapped lines: %d", len(wrappedLines))

	// In a properly synchronized diff view, both sides should have the same
	// number of visual lines (with padding/empty lines as needed)
	// This test documents the current behavior - if they're out of sync, that's a bug
	if len(leftWrappedLines) != len(wrappedLines) {
		t.Logf("Note: Left and right have different visual line counts (%d vs %d)",
			len(leftWrappedLines), len(wrappedLines))
	}
}

// TestDiffViewLongLineNoWrap specifically tests that a long line on the right
// doesn't wrap because CalculateWrappedLines uses screen width instead of window width.
func TestDiffViewLongLineNoWrap(t *testing.T) {
	// Use a narrow screen to make the issue obvious
	// 60 columns total, so each diff window gets ~30 columns
	te := NewTestEditorWithContent(60, 24, "")

	// Create a line that's 50 chars - fits full screen but not half-screen window
	longLine := strings.Repeat("x", 50)
	shortLine := "ok"

	originalContent := shortLine
	modifiedContent := longLine

	diffResult := computeDiff(originalContent, modifiedContent)
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)

	headBufferName := "test.py@HEAD"
	currentBufferName := "test.py@current"

	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, currentBufferName, config.IndentTabs, 4)

	te.buffers[headBufferName] = headBuffer
	te.buffers[currentBufferName] = currentBuffer
	te.diffLineInfo[headBuffer] = diffResult.LeftLines
	te.diffLineInfo[currentBuffer] = diffResult.RightLines

	te.createSecondWindow()
	te.windows[0].setActiveBuffer(headBuffer)
	te.windows[1].setActiveBuffer(currentBuffer)
	te.activeWindow = 1
	te.switchToBuffer(currentBuffer)
	// Recompute layout now that diff buffers are assigned - this triggers diff mode detection
	te.recomputeWindowsLayout()

	rightWindowWidth := te.windows[1].Width
	t.Logf("Screen width: 60, Right window width: %d, Long line length: %d", rightWindowWidth, len(longLine))

	// The 50-char line should NOT fit in a ~30-char window
	if len(longLine) <= rightWindowWidth {
		t.Skipf("Test setup issue: long line fits in window")
	}

	wrappedLines := te.CalculateWrappedLines()

	// BUG: CalculateWrappedLines uses screen width (60) instead of window width (~30)
	// So the 50-char line is NOT wrapped (50 < 60), but it should be (50 > 30)
	if len(wrappedLines) == 1 && len(longLine) > rightWindowWidth {
		t.Errorf("BUG: Line of %d chars not wrapped for window of %d chars wide",
			len(longLine), rightWindowWidth)
		t.Errorf("CalculateWrappedLines uses screen width instead of window width")
	}

	// Expected: line should wrap into 2 visual lines (50 chars / 30 = ~2 lines)
	expectedMinLines := (len(longLine) / rightWindowWidth) + 1
	if len(wrappedLines) < expectedMinLines {
		t.Errorf("Expected at least %d wrapped lines, got %d", expectedMinLines, len(wrappedLines))
	}
}
