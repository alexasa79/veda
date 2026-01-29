package editor

import (
	"fmt"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestDiffView_CursorSynchronization(t *testing.T) {
	// Create test content with multiple lines
	previousContent := "line1\nline2\nline3\nline4\nline5"
	currentContent := "line1\nmodified line2\nline3\nline4\nline5\nextra line6"

	te := NewTestEditorWithContent(120, 30, currentContent)
	te.CurrentBuffer().FilePath = "test.go"

	// Mock the git functionality by creating the head buffer manually
	headBufferName := "test.go@HEAD"
	currentBufferName := "test.go@current"

	// Manually set up the diff buffers like StartDiffViewForCurrentBuffer would
	te.buffers[headBufferName] = buffer.NewBuffer(previousContent, headBufferName, headBufferName, config.IndentTabs, 4)
	te.buffers[headBufferName].IsReadOnly = true
	te.buffers[currentBufferName] = buffer.NewBuffer(currentContent, currentBufferName, currentBufferName, config.IndentTabs, 4)

	// Set up two windows manually
	te.createSecondWindow()
	if len(te.windows) < 2 {
		t.Fatalf("Expected at least 2 windows, got %d", len(te.windows))
	}

	// Set up windows: HEAD on left (window 0), current on right (window 1)
	te.windows[0].Buffer = te.buffers[headBufferName]
	te.windows[1].Buffer = te.buffers[currentBufferName]
	te.activeWindow = 1 // Start on right side (current)
	te.switchToBuffer(te.buffers[currentBufferName])
	// Recompute layout now that diff buffers are assigned - this triggers diff mode detection
	te.recomputeWindowsLayout()

	t.Run("pressing down moves cursors in both buffers", func(t *testing.T) {
		// Get initial cursor positions
		rightBuffer := te.buffers[currentBufferName]
		leftBuffer := te.buffers[headBufferName]

		initialRightLine := rightBuffer.CursorIterator.LineNumber()
		initialLeftLine := leftBuffer.CursorIterator.LineNumber()

		t.Logf("Initial positions - Right: line=%d, Left: line=%d", initialRightLine, initialLeftLine)

		// Press down
		err := te.ExecuteCommand("down")
		if err != nil {
			t.Fatalf("Failed to execute down command: %v", err)
		}

		// Check final positions
		finalRightLine := rightBuffer.CursorIterator.LineNumber()
		finalLeftLine := leftBuffer.CursorIterator.LineNumber()

		t.Logf("Final positions - Right: line=%d, Left: line=%d", finalRightLine, finalLeftLine)

		// Both cursors should have moved down by 1
		if finalRightLine != initialRightLine+1 {
			t.Errorf("Right buffer cursor should move down by 1: initial=%d, final=%d", initialRightLine, finalRightLine)
		}

		if finalLeftLine != initialLeftLine+1 {
			t.Errorf("Left buffer cursor should move down by 1: initial=%d, final=%d", initialLeftLine, finalLeftLine)
		}

		// Both cursors should be at the same line number
		if finalRightLine != finalLeftLine {
			t.Errorf("Cursors should be synchronized: right=%d, left=%d", finalRightLine, finalLeftLine)
		}
	})

	t.Run("pressing down multiple times keeps cursors synchronized", func(t *testing.T) {
		// Press down 3 more times
		for i := 0; i < 3; i++ {
			err := te.ExecuteCommand("down")
			if err != nil {
				t.Fatalf("Failed to execute down command (iteration %d): %v", i, err)
			}
		}

		rightBuffer := te.buffers[currentBufferName]
		leftBuffer := te.buffers[headBufferName]

		rightLine := rightBuffer.CursorIterator.LineNumber()
		leftLine := leftBuffer.CursorIterator.LineNumber()

		t.Logf("After 3 more downs - Right: line=%d, Left: line=%d", rightLine, leftLine)

		// Cursors should still be synchronized
		if rightLine != leftLine {
			t.Errorf("Cursors should remain synchronized after multiple moves: right=%d, left=%d", rightLine, leftLine)
		}
	})

	t.Run("horizontal movement does not sync X position (only Y and ViewIterator sync)", func(t *testing.T) {
		// Press right a few times
		for i := 0; i < 3; i++ {
			err := te.ExecuteCommand("right")
			if err != nil {
				t.Fatalf("Failed to execute right command (iteration %d): %v", i, err)
			}
		}

		rightBuffer := te.buffers[currentBufferName]
		leftBuffer := te.buffers[headBufferName]

		rightX := rightBuffer.CursorX
		leftX := leftBuffer.CursorX
		rightY := rightBuffer.CursorIterator.LineNumber()
		leftY := leftBuffer.CursorIterator.LineNumber()

		t.Logf("After 3 rights - Right: col=%d line=%d, Left: col=%d line=%d", rightX, rightY, leftX, leftY)

		// Y positions should still be synchronized
		if rightY != leftY {
			t.Errorf("Y cursor positions should be synchronized: right=%d, left=%d", rightY, leftY)
		}

		// X positions are deliberately NOT synchronized (content may differ)
		// This is expected behavior as confirmed by user requirements
	})

	t.Run("scrolling past screen height synchronizes both panes", func(t *testing.T) {
		// Create content with many lines (more than screen height)
		manyLinesContent := ""
		for i := 1; i <= 50; i++ {
			if i > 1 {
				manyLinesContent += "\n"
			}
			manyLinesContent += fmt.Sprintf("line %d content here", i)
		}

		// Set up a smaller screen to force scrolling
		te := NewTestEditorWithContent(80, 10, manyLinesContent) // Only 10 lines tall
		te.CurrentBuffer().FilePath = "test.go"

		headBufferName := "test.go@HEAD"
		currentBufferName := "test.go@current"

		// Create diff-aligned content (both sides same content so equal line counts)
		diffResult := computeDiff(manyLinesContent, manyLinesContent)
		alignedHeadContent := createBufferContent(diffResult.LeftLines)
		alignedCurrentContent := createBufferContent(diffResult.RightLines)

		// Set up diff buffers with aligned content
		te.buffers[headBufferName] = buffer.NewBuffer(alignedHeadContent, headBufferName, headBufferName, config.IndentTabs, 4)
		te.buffers[headBufferName].IsReadOnly = true
		te.buffers[currentBufferName] = buffer.NewBuffer(alignedCurrentContent, currentBufferName, currentBufferName, config.IndentTabs, 4)

		// Set up two windows
		te.createSecondWindow()
		te.windows[0].Buffer = te.buffers[headBufferName]
		te.windows[1].Buffer = te.buffers[currentBufferName]
		te.activeWindow = 1 // Start on right side
		te.switchToBuffer(te.buffers[currentBufferName])

		// Get initial view positions
		rightBuffer := te.buffers[currentBufferName]
		leftBuffer := te.buffers[headBufferName]

		initialRightViewLine := rightBuffer.ViewIterator.LineNumber()
		initialLeftViewLine := leftBuffer.ViewIterator.LineNumber()

		t.Logf("Initial view positions - Right: %d, Left: %d", initialRightViewLine, initialLeftViewLine)

		// Debug: Check buffer line count and content
		t.Logf("Buffer line count: %d", rightBuffer.LineCount)
		t.Logf("Buffer content length: %d", len(manyLinesContent))
		debugLen := 100
		if len(manyLinesContent) < debugLen {
			debugLen = len(manyLinesContent)
		}
		t.Logf("First few lines of content: %q", manyLinesContent[:debugLen])

		// Move down many times to force scrolling (more than screen height)
		// Use direct cursor movement instead of visual movement commands
		for i := 0; i < 15; i++ {
			beforeLine := rightBuffer.CursorIterator.LineNumber()
			beforeViewLine := rightBuffer.ViewIterator.LineNumber()

			// Move cursor down one logical line directly
			if beforeLine < rightBuffer.LineCount-1 {
				rightBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(rightBuffer, beforeLine+1)
			}

			// Manually trigger scroll (since test environment might not call it)
			te.scroll()

			// Sync diff cursors after scrolling (mimicking what happens in the main event loop)
			if te.isDiffBuffer(te.CurrentBuffer()) {
				te.syncDiffCursors()
			}

			afterLine := rightBuffer.CursorIterator.LineNumber()
			afterViewLine := rightBuffer.ViewIterator.LineNumber()

			t.Logf("Iteration %d: cursor %d->%d, view %d->%d", i, beforeLine, afterLine, beforeViewLine, afterViewLine)

			// If cursor didn't move, we've hit a boundary
			if afterLine == beforeLine {
				t.Logf("Cursor stopped moving at iteration %d, line %d", i, afterLine)
				break
			}
		}

		// Get final cursor and view positions
		finalRightCursorLine := rightBuffer.CursorIterator.LineNumber()
		finalLeftCursorLine := leftBuffer.CursorIterator.LineNumber()
		finalRightViewLine := rightBuffer.ViewIterator.LineNumber()
		finalLeftViewLine := leftBuffer.ViewIterator.LineNumber()

		t.Logf("Final cursor positions - Right: %d, Left: %d", finalRightCursorLine, finalLeftCursorLine)
		t.Logf("Final view positions - Right: %d, Left: %d", finalRightViewLine, finalLeftViewLine)

		// Cursors should be synchronized
		if finalRightCursorLine != finalLeftCursorLine {
			t.Errorf("Cursors should be synchronized after scrolling: right=%d, left=%d",
				finalRightCursorLine, finalLeftCursorLine)
		}

		// Cursor should have moved down by 15
		if finalRightCursorLine != 15 {
			t.Errorf("Expected cursor to be at line 15 after 15 down movements, got %d", finalRightCursorLine)
		}

		// View positions should also be synchronized for proper scrolling
		if finalRightViewLine != finalLeftViewLine {
			t.Errorf("View positions should be synchronized: right=%d, left=%d",
				finalRightViewLine, finalLeftViewLine)
		}

		// Since we moved past screen height (10), view should have scrolled
		if finalRightViewLine <= initialRightViewLine {
			t.Errorf("Expected view to scroll down, initial=%d, final=%d",
				initialRightViewLine, finalRightViewLine)
		}
	})
}
