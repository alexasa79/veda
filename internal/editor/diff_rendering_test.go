package editor

import (
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
	te.diffLineInfo[headBuffer] = diffResult.LeftLines
	te.diffLineInfo[currentBuffer] = diffResult.RightLines

	// Set up windows for side-by-side diff view
	te.createSecondWindow()
	te.windows[0].Buffer = headBuffer    // HEAD on left
	te.windows[1].Buffer = currentBuffer // Current on right
	te.activeWindow = 1
	te.switchToBuffer(currentBuffer)
	// Recompute layout now that diff buffers are assigned - this triggers diff mode detection
	te.recomputeWindowsLayout()

	// Get expected diff colors (using the same brighter colors as in render.go)
	expectedAddedBg := ui.NewRGBColor(40, 100, 40)    // Even brighter green background
	expectedDeletedBg := ui.NewRGBColor(100, 40, 40)  // Even brighter red background
	expectedModifiedBg := ui.NewRGBColor(100, 80, 30) // Even brighter yellow/orange background

	t.Run("head_buffer_rendering", func(t *testing.T) {
		// Switch to head buffer and render
		te.activeWindow = 0
		te.switchToBuffer(headBuffer)

		// Verify diff info is set up correctly
		if diffLines, exists := te.diffLineInfo[headBuffer]; exists {
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

		te.drawEditor()

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
			case DiffLineContext:
				// Context lines should have normal background or current line highlight
				// We'll just check it's not a diff color
				if actualBg == expectedAddedBg || actualBg == expectedDeletedBg || actualBg == expectedModifiedBg {
					t.Errorf("Line %d (context): unexpected diff background color %v", lineIdx, actualBg)
				}
				t.Logf("✓ Line %d (context): background %v (content: %q)", lineIdx, actualBg, diffLine.Content)

			case DiffLineDeleted:
				if actualBg != expectedDeletedBg {
					t.Errorf("Line %d (deleted): expected background %v, got %v (content: %q)",
						lineIdx, expectedDeletedBg, actualBg, diffLine.Content)
				} else {
					t.Logf("✓ Line %d (deleted): correct background %v (content: %q)", lineIdx, actualBg, diffLine.Content)
				}

			case DiffLineModified:
				if actualBg != expectedModifiedBg {
					t.Errorf("Line %d (modified): expected background %v, got %v (content: %q)",
						lineIdx, expectedModifiedBg, actualBg, diffLine.Content)
				} else {
					t.Logf("✓ Line %d (modified): correct background %v (content: %q)", lineIdx, actualBg, diffLine.Content)
				}

			case DiffLineAdded:
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
		te.activeWindow = 1
		te.switchToBuffer(currentBuffer)

		// Verify diff info is set up correctly
		if diffLines, exists := te.diffLineInfo[currentBuffer]; exists {
			t.Logf("Current buffer has %d diff lines", len(diffLines))
			for i, line := range diffLines {
				t.Logf("  Line %d: type=%v, content=%q", i, line.LineType, line.Content)
			}
		} else {
			t.Error("Current buffer should have diff line info")
		}

		// Move cursor away from current line to avoid current line highlighting interference
		te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 3)

		te.drawEditor()

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
			case DiffLineContext:
				// Context lines should have normal background or current line highlight
				if actualBg == expectedAddedBg || actualBg == expectedDeletedBg || actualBg == expectedModifiedBg {
					t.Errorf("Line %d (context): unexpected diff background color %v", lineIdx, actualBg)
				}
				t.Logf("✓ Line %d (context): background %v (content: %q)", lineIdx, actualBg, diffLine.Content)

			case DiffLineAdded:
				if actualBg != expectedAddedBg {
					t.Errorf("Line %d (added): expected background %v, got %v (content: %q)",
						lineIdx, expectedAddedBg, actualBg, diffLine.Content)
				} else {
					t.Logf("✓ Line %d (added): correct background %v (content: %q)", lineIdx, actualBg, diffLine.Content)
				}

			case DiffLineModified:
				if actualBg != expectedModifiedBg {
					t.Errorf("Line %d (modified): expected background %v, got %v (content: %q)",
						lineIdx, expectedModifiedBg, actualBg, diffLine.Content)
				} else {
					t.Logf("✓ Line %d (modified): correct background %v (content: %q)", lineIdx, actualBg, diffLine.Content)
				}

			case DiffLineDeleted:
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
			case DiffLineContext:
				contextCount++
			case DiffLineAdded:
				addedCount++
			case DiffLineDeleted:
				deletedCount++
			case DiffLineModified:
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

	te.diffLineInfo[headBuffer] = diffResult.LeftLines
	te.diffLineInfo[currentBuffer] = diffResult.RightLines

	// Set up diff view
	te.createSecondWindow()
	te.windows[0].Buffer = headBuffer
	te.windows[1].Buffer = currentBuffer
	// Recompute layout now that diff buffers are assigned - this triggers diff mode detection
	te.recomputeWindowsLayout()

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
		te.activeWindow = 0
		te.switchToBuffer(headBuffer)
		te.render()

		te.activeWindow = 1
		te.switchToBuffer(currentBuffer)
		te.render()

		// Basic test: ensure rendering doesn't crash with complex diffs
		t.Log("✓ Complex diff rendering completed without errors")
	})
}
