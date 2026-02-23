package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestDiffViewWindowWidth(t *testing.T) {
	// Create a terminal (80 columns)
	width := 80
	height := 24
	te := NewTestEditor(width, height)

	// Create diff content
	leftContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	rightContent := "Line 1 modified\nLine 2\nNew Line 3\nLine 4\nLine 5 modified"

	// Compute diff
	diffResult := computeDiff(leftContent, rightContent)

	// Create aligned buffers
	headBuffer := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), "test.txt@HEAD", "test.txt@HEAD", config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	headBuffer.DiffHints = diffResult.LeftLines

	currentBuffer := buffer.NewBuffer(createBufferContent(diffResult.RightLines), "test.txt@current", "test.txt@current", config.IndentTabs, 4)
	currentBuffer.DiffHints = diffResult.RightLines

	te.buffers["test.txt@HEAD"] = headBuffer
	te.buffers["test.txt@current"] = currentBuffer

	// Set up a DiffBufferView on a single window
	diffView := &DiffBufferView{
		LeftBuffer:  headBuffer,
		RightBuffer: currentBuffer,
		ActivePane:  1,
		RealPath:    "test.txt",
	}
	te.windows[0].View = diffView
	te.windows[0].setActiveBuffer(currentBuffer)
	te.windows[0].addBufferToWindow(headBuffer)
	te.switchToBuffer(currentBuffer)

	te.recomputeWindowsLayout()

	// Test 1: DiffBufferView renders both panes within a single window
	if len(te.windows) != 1 {
		t.Errorf("Expected 1 window with DiffBufferView, got %d", len(te.windows))
	}

	// Test 2: The single window should have the full screen width
	win := te.windows[0]
	if win.Width != width {
		t.Errorf("Diff window should have full screen width %d, got %d", width, win.Width)
	}

	// Test 3: DiffBufferView.Render splits the screen in half
	// Create a mock sub-screen and render
	sub := newWindowScreen(te.screen, win.X, win.Y, win.Width, win.Height, false)
	diffView.Render(te.Editor, win, sub)

	// The left pane gets width/2, right pane gets width-width/2
	expectedLeftW := width / 2
	expectedRightW := width - expectedLeftW
	if expectedLeftW < 1 || expectedRightW < 1 {
		t.Errorf("Pane widths too small: left=%d, right=%d", expectedLeftW, expectedRightW)
	}
	t.Logf("Pane widths: left=%d, right=%d (total=%d)", expectedLeftW, expectedRightW, width)

	// Test 4: Adding a second window reduces the diff window width proportionally
	thirdBuffer := buffer.NewBuffer("Third buffer content", "third.txt", "third.txt", config.IndentTabs, 4)
	te.buffers["third.txt"] = thirdBuffer
	newWin := newWindow(thirdBuffer)
	te.windows = append(te.windows, newWin)
	te.recomputeWindowsLayout()

	// Both windows should share screen width
	totalW := 0
	for _, w := range te.windows {
		totalW += w.Width
	}
	if totalW != width {
		t.Errorf("Total window widths should equal screen width: got %d, want %d", totalW, width)
	}
}
