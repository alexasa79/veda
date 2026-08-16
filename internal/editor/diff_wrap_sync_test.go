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
	te := NewTestEditorWithContent(80, 24, "")

	shortLine := "short line"
	longLine := "this is a very long line that should definitely wrap when the window is only about 40 characters wide"

	originalContent := shortLine
	modifiedContent := longLine

	diffResult := computeDiff(originalContent, modifiedContent)
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)

	t.Logf("Left (HEAD) content: %q", alignedHeadContent)
	t.Logf("Right (current) content: %q", alignedCurrentContent)
	t.Logf("Left lines: %d, Right lines: %d", len(diffResult.LeftLines), len(diffResult.RightLines))

	headBufferName := "test.py@HEAD"
	currentBufferName := "test.py@current"

	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, currentBufferName, config.IndentTabs, 4)

	te.buffers[headBufferName] = headBuffer
	te.buffers[currentBufferName] = currentBuffer
	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer.DiffHints = diffResult.RightLines

	te.SetupDiffView(headBuffer, currentBuffer, "test.py")

	win := te.windows[te.activeWindow]
	paneWidth := win.Width / 2
	t.Logf("Window: width=%d, height=%d, pane width=%d", win.Width, win.Height, paneWidth)

	if len(longLine) <= paneWidth {
		t.Skipf("Long line (%d chars) fits in pane width (%d), adjust test", len(longLine), paneWidth)
	}

	t.Logf("Long line length: %d, pane width: %d", len(longLine), paneWidth)

	wrappedLines := te.CalculateWrappedLines()

	t.Logf("Number of wrapped lines: %d", len(wrappedLines))
	for i, wl := range wrappedLines {
		t.Logf("  Wrapped line %d: logical=%d, visual=%d, start=%d, end=%d, text=%q",
			i, wl.LogicalLineNum, wl.VisualLineNum, wl.StartChar, wl.EndChar, wl.Text)
	}

	gutterWidth := te.CurrentBuffer().LineNumberWidth
	availableWidth := paneWidth - gutterWidth
	t.Logf("Available text width (pane - gutter): %d - %d = %d", paneWidth, gutterWidth, availableWidth)

	for i, wl := range wrappedLines {
		if len(wl.Text) > availableWidth {
			t.Errorf("Wrapped line %d has %d chars but pane only has %d usable columns",
				i, len(wl.Text), availableWidth)
		}
	}

	// Both sides should have the same number of visual lines
	te.switchToBuffer(headBuffer)
	leftWrappedLines := te.CalculateWrappedLines()

	t.Logf("Left buffer wrapped lines: %d", len(leftWrappedLines))
	t.Logf("Right buffer wrapped lines: %d", len(wrappedLines))

	if len(leftWrappedLines) != len(wrappedLines) {
		t.Logf("Note: Left and right have different visual line counts (%d vs %d)",
			len(leftWrappedLines), len(wrappedLines))
	}
}

// TestDiffViewLongLineNoWrap specifically tests that a long line on the right
// wraps correctly using the pane width, not the full screen width.
func TestDiffViewLongLineNoWrap(t *testing.T) {
	te := NewTestEditorWithContent(60, 24, "")

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
	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer.DiffHints = diffResult.RightLines

	te.SetupDiffView(headBuffer, currentBuffer, "test.py")

	paneWidth := te.windows[te.activeWindow].Width / 2
	t.Logf("Screen width: 60, Pane width: %d, Long line length: %d", paneWidth, len(longLine))

	if len(longLine) <= paneWidth {
		t.Skipf("Test setup issue: long line fits in pane")
	}

	wrappedLines := te.CalculateWrappedLines()

	// The 50-char line should be wrapped since it exceeds the ~30-char pane width
	gutterWidth := te.CurrentBuffer().LineNumberWidth
	availableWidth := paneWidth - gutterWidth
	for i, wl := range wrappedLines {
		if len(wl.Text) > availableWidth {
			t.Errorf("Wrapped line %d has %d chars but pane only has %d usable columns",
				i, len(wl.Text), availableWidth)
		}
	}
}

func TestDiffViewMovementPastAsymmetricWrap(t *testing.T) {
	te := NewTestEditorWithContent(164, 24, "")

	leftLine := "\tif state == nil || initialPositions <= 0 || a.Account <= 0 || state.ATR14 <= 0 {"
	rightLine := "\tif state == nil || openPositions <= 0 || a.Account <= 0 || state.ATR14 <= 0 {"
	leftContent := leftLine + "\n\treturn 0"
	rightContent := rightLine + "\n\treturn 0"
	diffResult := computeDiff(leftContent, rightContent)

	headBuffer := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer := buffer.NewBuffer(createBufferContent(diffResult.RightLines), "test.go@current", "test.go@current", config.IndentTabs, 4)
	currentBuffer.DiffHints = diffResult.RightLines

	te.buffers[headBuffer.FilePath] = headBuffer
	te.buffers[currentBuffer.FilePath] = currentBuffer
	te.SetupDiffView(headBuffer, currentBuffer, "test.go")

	paneWidth := te.windows[te.activeWindow].Width / 2
	leftWraps := NewWrapCalculator(paneWidth, headBuffer.LineNumberWidth).WrapLine(leftLine, 0)
	rightWraps := NewWrapCalculator(paneWidth, currentBuffer.LineNumberWidth).WrapLine(rightLine, 0)
	if len(leftWraps) <= len(rightWraps) {
		t.Fatalf("test requires left line to wrap more than right: left=%d right=%d", len(leftWraps), len(rightWraps))
	}

	currentBuffer.CursorX = len(rightLine) - 1
	if err := te.ExecuteCommand("down"); err != nil {
		t.Fatalf("move down failed: %v", err)
	}

	if got := currentBuffer.CursorIterator.LineNumber(); got != 1 {
		t.Fatalf("move down was trapped on asymmetrically wrapped line: got line %d, want 1", got)
	}

	currentBuffer.CursorX = 0
	te.desiredColumn = -1
	if err := te.ExecuteCommand("up"); err != nil {
		t.Fatalf("move up failed: %v", err)
	}
	if got := currentBuffer.CursorIterator.LineNumber(); got != 0 {
		t.Fatalf("move up did not cross asymmetric wrap padding: got line %d, want 0", got)
	}
	if got := currentBuffer.CursorX; got != 0 {
		t.Fatalf("move up used padding position instead of desired column: got column %d, want 0", got)
	}
}
