package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/ui"
)

// TestDiffRendering verifies that diff lines are rendered with the correct styles
func TestDiffRendering(t *testing.T) {
	// Create test editor with synthetic diff content
	te := NewTestEditorWithContent(80, 24, "")

	// Create synthetic diff content with all change types
	originalContent := "unchanged line 1\ndeleted line\nunchanged line 3\noriginal line 4"
	modifiedContent := "unchanged line 1\nadded line\nunchanged line 3\nmodified line 4"

	// Create diff result with different line types
	diffResult := computeDiff(originalContent, modifiedContent)

	// Create aligned buffer content
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)

	// Create diff buffers
	headBufferName := "test.go@HEAD"
	currentBufferName := "test.go@current"

	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, currentBufferName, config.IndentTabs, 4)

	// Store buffers and diff line info
	te.buffers[headBufferName] = headBuffer
	te.buffers[currentBufferName] = currentBuffer
	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on a single window
	te.SetupDiffView(headBuffer, currentBuffer, "test.go")

	// Get expected diff colors (using the same brighter colors as in render.go)
	expectedAddedBg := ui.NewRGBColor(40, 100, 40)    // Even brighter green background
	expectedDeletedBg := ui.NewRGBColor(100, 40, 40)  // Even brighter red background
	expectedModifiedBg := ui.NewRGBColor(100, 80, 30) // Even brighter yellow/orange background

	t.Run("head_buffer_rendering", func(t *testing.T) {
		// Switch to head buffer and render
		te.switchToBuffer(headBuffer)

		// Verify diff info is set up correctly
		if diffLines := headBuffer.DiffHints; len(diffLines) > 0 {
			t.Logf("Head buffer has %d diff lines", len(diffLines))
			for i, line := range diffLines {
				t.Logf("  Line %d: type=%v, content=%q", i, line.LineType, line.Content)
			}
		} else {
			t.Error("Head buffer should have diff line info")
		}

		// Verify isDiffBuffer works
		if !te.isDiffBuffer(headBuffer) {
			t.Error("Head buffer should be identified as diff buffer")
		}

		// Move cursor away from current line to avoid current line highlighting interference
		// Move cursor to line 3 to test other lines without current line interference
		te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 3)

		te.renderPane()

		// Verify that the screen has been rendered
		width, height := te.screen.Size()
		if width == 0 || height == 0 {
			t.Fatal("Screen size should not be zero after rendering")
		}

		// Check line backgrounds in the rendered screen
		// Note: We need to account for gutter and window layout

		// Find the content area (skip gutter columns)
		contentStartX := te.CurrentBuffer().LineNumberWidth
		t.Logf("Using content start X: %d (line number width)", contentStartX)

		// Get diff lines for verification
		if len(diffResult.LeftLines) < 4 {
			t.Fatalf("Expected at least 4 diff lines, got %d", len(diffResult.LeftLines))
		}

		// Test each line type by checking rendered styles
		for lineIdx, diffLine := range diffResult.LeftLines {
			if lineIdx >= height-1 { // Skip if beyond screen height
				break
			}

			// Get the style of the first content character on this line
			mockScreen := te.screen.(*MockScreen)
			_, style := mockScreen.GetCell(contentStartX, lineIdx)
			_, actualBg, _ := style.Decompose()

			// Verify background color matches expected diff type
			switch diffLine.LineType {
			case buffer.DiffLineContext:
				// Context lines should have normal background or current line highlight
				// We'll just check it's not a diff color
				if actualBg == expectedAddedBg || actualBg == expectedDeletedBg || actualBg == expectedModifiedBg {
					t.Errorf("Line %d (context): unexpected diff background color %v", lineIdx, actualBg)
				}
				t.Logf("✓ Line %d (context): background %v (content: %q)", lineIdx, actualBg, diffLine.Content)

			case buffer.DiffLineDeleted:
				if actualBg != expectedDeletedBg {
					t.Errorf("Line %d (deleted): expected background %v, got %v (content: %q)",
						lineIdx, expectedDeletedBg, actualBg, diffLine.Content)
				} else {
					t.Logf("✓ Line %d (deleted): correct background %v (content: %q)", lineIdx, actualBg, diffLine.Content)
				}

			case buffer.DiffLineModified:
				if actualBg != expectedModifiedBg {
					t.Errorf("Line %d (modified): expected background %v, got %v (content: %q)",
						lineIdx, expectedModifiedBg, actualBg, diffLine.Content)
				} else {
					t.Logf("✓ Line %d (modified): correct background %v (content: %q)", lineIdx, actualBg, diffLine.Content)
				}

			case buffer.DiffLineAdded:
				// This should be a filler line in head buffer
				if !diffLine.IsFillerLine {
					t.Errorf("Line %d: expected filler line for added content in head buffer", lineIdx)
				}
				if actualBg != expectedAddedBg {
					t.Errorf("Line %d (added filler): expected background %v, got %v",
						lineIdx, expectedAddedBg, actualBg)
				} else {
					t.Logf("✓ Line %d (added filler): correct background %v", lineIdx, actualBg)
				}
			}
		}
	})

	t.Run("current_buffer_rendering", func(t *testing.T) {
		// Switch to current buffer and render
		te.switchToBuffer(currentBuffer)

		// Verify diff info is set up correctly
		if diffLines := currentBuffer.DiffHints; len(diffLines) > 0 {
			t.Logf("Current buffer has %d diff lines", len(diffLines))
			for i, line := range diffLines {
				t.Logf("  Line %d: type=%v, content=%q", i, line.LineType, line.Content)
			}
		} else {
			t.Error("Current buffer should have diff line info")
		}

		// Move cursor away from current line to avoid current line highlighting interference
		te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 3)

		te.renderPane()

		// Find the content area (skip gutter columns)
		contentStartX := te.CurrentBuffer().LineNumberWidth
		t.Logf("Using content start X: %d (line number width)", contentStartX)

		// Test each line type in current buffer
		for lineIdx, diffLine := range diffResult.RightLines {
			if lineIdx >= 24-1 { // Skip if beyond screen height
				break
			}

			// Get the style of the first content character on this line
			mockScreen := te.screen.(*MockScreen)
			_, style := mockScreen.GetCell(contentStartX, lineIdx)
			_, actualBg, _ := style.Decompose()

			// Verify background color matches expected diff type
			switch diffLine.LineType {
			case buffer.DiffLineContext:
				// Context lines should have normal background or current line highlight
				if actualBg == expectedAddedBg || actualBg == expectedDeletedBg || actualBg == expectedModifiedBg {
					t.Errorf("Line %d (context): unexpected diff background color %v", lineIdx, actualBg)
				}
				t.Logf("✓ Line %d (context): background %v (content: %q)", lineIdx, actualBg, diffLine.Content)

			case buffer.DiffLineAdded:
				if actualBg != expectedAddedBg {
					t.Errorf("Line %d (added): expected background %v, got %v (content: %q)",
						lineIdx, expectedAddedBg, actualBg, diffLine.Content)
				} else {
					t.Logf("✓ Line %d (added): correct background %v (content: %q)", lineIdx, actualBg, diffLine.Content)
				}

			case buffer.DiffLineModified:
				if actualBg != expectedModifiedBg {
					t.Errorf("Line %d (modified): expected background %v, got %v (content: %q)",
						lineIdx, expectedModifiedBg, actualBg, diffLine.Content)
				} else {
					t.Logf("✓ Line %d (modified): correct background %v (content: %q)", lineIdx, actualBg, diffLine.Content)
				}

			case buffer.DiffLineDeleted:
				// This should be a filler line in current buffer
				if !diffLine.IsFillerLine {
					t.Errorf("Line %d: expected filler line for deleted content in current buffer", lineIdx)
				}
				if actualBg != expectedDeletedBg {
					t.Errorf("Line %d (deleted filler): expected background %v, got %v",
						lineIdx, expectedDeletedBg, actualBg)
				} else {
					t.Logf("✓ Line %d (deleted filler): correct background %v", lineIdx, actualBg)
				}
			}
		}
	})

	t.Run("diff_line_structure_verification", func(t *testing.T) {
		t.Skip("Skipping test - asymmetric highlighting not yet implemented")
		// Verify the diff structure makes sense
		if len(diffResult.LeftLines) != len(diffResult.RightLines) {
			t.Errorf("Left and right diff lines should have equal length: left=%d, right=%d",
				len(diffResult.LeftLines), len(diffResult.RightLines))
		}

		// Log the diff structure for debugging
		t.Logf("Diff structure:")
		for i := 0; i < len(diffResult.LeftLines) && i < len(diffResult.RightLines); i++ {
			left := diffResult.LeftLines[i]
			right := diffResult.RightLines[i]
			t.Logf("  %d: LEFT[%v] %q | RIGHT[%v] %q",
				i, left.LineType, left.Content, right.LineType, right.Content)
		}

		// Count different line types
		var contextCount, addedCount, deletedCount, modifiedCount int
		for _, line := range diffResult.LeftLines {
			switch line.LineType {
			case buffer.DiffLineContext:
				contextCount++
			case buffer.DiffLineAdded:
				addedCount++
			case buffer.DiffLineDeleted:
				deletedCount++
			case buffer.DiffLineModified:
				modifiedCount++
			}
		}

		t.Logf("Line type counts: context=%d, added=%d, deleted=%d, modified=%d",
			contextCount, addedCount, deletedCount, modifiedCount)

		// Ensure we have the correct distribution (after fixing asymmetric highlighting)
		if contextCount == 0 {
			t.Error("Expected at least one context line")
		}
		// After fixing asymmetric highlighting, left side should only have deleted and context lines
		if deletedCount == 0 {
			t.Error("Expected at least one deleted line")
		}
		// Added and modified lines should not appear in left side with fixed asymmetric highlighting
		if addedCount > 0 {
			t.Error("Left side should not have added lines after fixing asymmetric highlighting")
		}
		if modifiedCount > 0 {
			t.Error("Left side should not have modified lines after fixing asymmetric highlighting")
		}
	})
}

// TestDiffRenderingWithComplexChanges tests rendering with more complex diff scenarios
func TestDiffRenderingWithComplexChanges(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")

	// Create more complex diff content
	originalContent := "line A\nline B\nline C\nline D\nline E"
	modifiedContent := "line A\nnew line\nline C modified\nline E"

	diffResult := computeDiff(originalContent, modifiedContent)

	// Create buffers
	headBuffer := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
	currentBuffer := buffer.NewBuffer(createBufferContent(diffResult.RightLines), "test.go@current", "test.go@current", config.IndentTabs, 4)

	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on a single window
	te.SetupDiffView(headBuffer, currentBuffer, "test.go")

	t.Run("complex_diff_structure", func(t *testing.T) {
		// Verify line count equality
		if len(diffResult.LeftLines) != len(diffResult.RightLines) {
			t.Errorf("Complex diff should have equal line counts: left=%d, right=%d",
				len(diffResult.LeftLines), len(diffResult.RightLines))
		}

		t.Logf("Complex diff structure:")
		for i := 0; i < len(diffResult.LeftLines); i++ {
			left := diffResult.LeftLines[i]
			right := diffResult.RightLines[i]
			t.Logf("  %d: L[%v,%v] %q | R[%v,%v] %q",
				i, left.LineType, left.IsFillerLine, left.Content,
				right.LineType, right.IsFillerLine, right.Content)
		}
	})

	t.Run("complex_diff_rendering", func(t *testing.T) {
		// Test rendering of both buffers
		te.switchToBuffer(headBuffer)
		te.renderPane()

		te.switchToBuffer(currentBuffer)
		te.renderPane()

		// Basic test: ensure rendering doesn't crash with complex diffs
		t.Log("✓ Complex diff rendering completed without errors")
	})
}

// TestDiffView_LeftAndRightShowDifferentContent verifies that the left pane
// shows the previous (HEAD) content and the right pane shows the current
// (modified) content after a full render.
func TestDiffView_LeftAndRightShowDifferentContent(t *testing.T) {
	originalContent := "line one\nline two\nline three"
	modifiedContent := "line one\nchanged line two\nline three\nextra line"

	te := NewTestEditorWithContent(80, 24, modifiedContent)
	te.CurrentBuffer().FilePath = "test.go"

	diffResult := computeDiff(originalContent, modifiedContent)
	alignedHead := createBufferContent(diffResult.LeftLines)
	alignedCurrent := createBufferContent(diffResult.RightLines)

	headBuf := buffer.NewBuffer(alignedHead, "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
	headBuf.IsReadOnly = true
	headBuf.DiffHints = diffResult.LeftLines

	currBuf := buffer.NewBuffer(alignedCurrent, "test.go@current", "test.go@current", config.IndentTabs, 4)
	currBuf.DiffHints = diffResult.RightLines

	te.buffers[headBuf.FilePath] = headBuf
	te.buffers[currBuf.FilePath] = currBuf

	te.SetupDiffView(headBuf, currBuf, "test.go")

	// Full render so DiffBufferView.Render draws both panes
	te.render()

	mockScreen := te.screen.(*MockScreen)
	totalW, _ := mockScreen.Size()
	leftW := totalW / 2

	// Helper: extract visible text from a row within [xStart, xEnd)
	extractText := func(y, xStart, xEnd int) string {
		runes := make([]rune, 0, xEnd-xStart)
		for x := xStart; x < xEnd; x++ {
			ch, _ := mockScreen.GetCell(x, y)
			runes = append(runes, ch)
		}
		return strings.TrimRight(string(runes), " ")
	}

	// The diff has aligned rows. Find rows where left and right content
	// should differ. Line index 1 should have "line two" on the left and
	// "changed line two" on the right (both modified).
	// The gutter (line numbers) takes a few columns; scan past them.

	// Determine gutter width for each pane. Line numbers are right-aligned
	// digits followed by a space. Scan the first non-space rune after digits.
	gutterEnd := func(y, xStart, xEnd int) int {
		for x := xStart; x < xEnd; x++ {
			ch, _ := mockScreen.GetCell(x, y)
			if ch != ' ' && (ch < '0' || ch > '9') {
				return x
			}
		}
		return xStart
	}

	leftGutter := gutterEnd(0, 0, leftW)
	rightGutter := gutterEnd(0, leftW, totalW)

	t.Logf("Left gutter ends at x=%d, right gutter ends at x=%d", leftGutter, rightGutter)

	// Log all visible rows for debugging
	for row := 0; row < 6; row++ {
		leftText := extractText(row, leftGutter, leftW)
		rightText := extractText(row, rightGutter, totalW)
		t.Logf("Row %d: left=%q  right=%q", row, leftText, rightText)
	}

	// Verify that the left and right buffers contain their respective content.
	// At least one row must differ between left and right (the modified line).
	foundDifference := false
	for row := 0; row < len(diffResult.LeftLines) && row < 20; row++ {
		leftText := extractText(row, leftGutter, leftW)
		rightText := extractText(row, rightGutter, totalW)
		if leftText != rightText {
			foundDifference = true
			break
		}
	}
	if !foundDifference {
		t.Errorf("Left and right panes show identical content — expected them to differ")
	}

	// Verify specific content: the modified line should show different text
	// on each side. Check that "line two" appears on the left and
	// "changed line two" appears on the right.
	foundOriginal := false
	foundModified := false
	for row := 0; row < len(diffResult.LeftLines) && row < 20; row++ {
		leftText := extractText(row, leftGutter, leftW)
		rightText := extractText(row, rightGutter, totalW)
		if strings.Contains(leftText, "line two") && !strings.Contains(leftText, "changed") {
			foundOriginal = true
		}
		if strings.Contains(rightText, "changed line two") {
			foundModified = true
		}
	}
	if !foundOriginal {
		t.Errorf("Left pane should contain original 'line two'")
	}
	if !foundModified {
		t.Errorf("Right pane should contain modified 'changed line two'")
	}
}

// TestDiffView_CursorPositionFollowsActivePane verifies that the screen cursor
// appears in the right pane by default (ActivePane=1) and moves to the left
// pane after switching panes.
func TestDiffView_CursorPositionFollowsActivePane(t *testing.T) {
	originalContent := "line one\nline two\nline three"
	modifiedContent := "line one\nchanged two\nline three"

	te := NewTestEditorWithContent(80, 24, modifiedContent)
	te.CurrentBuffer().FilePath = "test.go"

	diffResult := computeDiff(originalContent, modifiedContent)
	alignedHead := createBufferContent(diffResult.LeftLines)
	alignedCurrent := createBufferContent(diffResult.RightLines)

	headBuf := buffer.NewBuffer(alignedHead, "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
	headBuf.IsReadOnly = true
	headBuf.DiffHints = diffResult.LeftLines

	currBuf := buffer.NewBuffer(alignedCurrent, "test.go@current", "test.go@current", config.IndentTabs, 4)
	currBuf.DiffHints = diffResult.RightLines

	te.buffers[headBuf.FilePath] = headBuf
	te.buffers[currBuf.FilePath] = currBuf

	dv := te.SetupDiffView(headBuf, currBuf, "test.go")

	totalW, _ := te.screen.Size()
	leftW := totalW / 2

	// Default: ActivePane == 1 (right pane). Cursor should be in the right half.
	te.render()
	curX, _ := te.GetScreenCursorPosition()
	if curX < leftW {
		t.Errorf("Default active pane is right; expected cursor X >= %d, got %d", leftW, curX)
	}

	// Switch pane to the left (ActivePane becomes 0).
	dv.SwitchPane()
	te.switchToBuffer(dv.ActiveBuffer())
	te.render()
	curX, _ = te.GetScreenCursorPosition()
	if curX >= leftW {
		t.Errorf("After switching to left pane; expected cursor X < %d, got %d", leftW, curX)
	}

	// Switch back to the right pane.
	dv.SwitchPane()
	te.switchToBuffer(dv.ActiveBuffer())
	te.render()
	curX, _ = te.GetScreenCursorPosition()
	if curX < leftW {
		t.Errorf("After switching back to right pane; expected cursor X >= %d, got %d", leftW, curX)
	}
}
