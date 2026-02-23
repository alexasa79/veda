package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestDiffView_PageUpViewSync reproduces the scenario of identical HEAD/current
// in diff view, going to end and paging up repeatedly. It asserts that the
// visible view top (window.ViewLine) tracks the cursor movement instead of
// getting stuck on an old top line.
func TestDiffView_PageUpViewSync(t *testing.T) {
	// Build a large file to allow multiple page jumps
	var b strings.Builder
	for i := 1; i <= 5000; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	content := b.String()

	te := NewTestEditor(160, 40) // wider screen; layout will still split panes

	// Compute an identical diff (HEAD == current)
	diffResult := computeDiff(content, content)

	// Create diff buffers
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), "editor.go@HEAD", "editor.go@HEAD", config.IndentTabs, 4)
	head.IsReadOnly = true
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), "editor.go@current", "editor.go@current", config.IndentTabs, 4)

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr
	head.DiffHints = diffResult.LeftLines
	curr.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on a single window
	te.SetupDiffView(head, curr, "editor.go")

	// Go to end and render once
	if err := te.ExecuteCommand("go-to-end"); err != nil {
		t.Fatalf("go-to-end failed: %v", err)
	}
	te.scroll()
	te.syncDiffCursors()
	te.render()

	// Capture initial top lines from buffer ViewIterators
	prevLeftTop := head.ViewIterator.LineNumber()
	prevRightTop := curr.ViewIterator.LineNumber()

	// Page up several times; after each step ensure view lines do not stay
	// permanently stuck at the old value.
	for i := 0; i < 5; i++ {
		if err := te.ExecuteCommand("page-up"); err != nil {
			t.Fatalf("page-up failed at step %d: %v", i, err)
		}
		// Update cursor-dependent state and perform scrolling like the main loop
		te.updateCursorState()
		te.scroll()
		te.syncDiffCursors()
		te.render()

		leftTop := head.ViewIterator.LineNumber()
		rightTop := curr.ViewIterator.LineNumber()

		if leftTop == prevLeftTop && rightTop == prevRightTop {
			t.Fatalf("view top lines did not change after page-up step %d (left=%d right=%d)", i, leftTop, rightTop)
		}

		// Next iteration compares against the latest value
		prevLeftTop = leftTop
		prevRightTop = rightTop
	}
}
