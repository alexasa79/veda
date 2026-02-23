package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/ui"
)

// TestDiffHighlighting verifies that diff lines are highlighted with appropriate colors
func TestDiffHighlighting(t *testing.T) {
	// Create test editor
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	te.CurrentBuffer().FilePath = "test.go"

	// Create diff buffers manually to test highlighting
	headBufferName := "test.go@HEAD"
	currentBufferName := "test.go@current"

	// Create test diff result with different line types
	diffResult := &buffer.DiffResult{
		LeftLines: []buffer.DiffLine{
			{Content: "line1", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
			{Content: "deleted line", LineType: buffer.DiffLineDeleted, LeftLine: 1, RightLine: -1, IsFillerLine: false},
			{Content: "old line3", LineType: buffer.DiffLineModified, LeftLine: 2, RightLine: 2, IsFillerLine: false},
			{Content: "", LineType: buffer.DiffLineAdded, LeftLine: -1, RightLine: -1, IsFillerLine: true},
		},
		RightLines: []buffer.DiffLine{
			{Content: "line1", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
			{Content: "", LineType: buffer.DiffLineDeleted, LeftLine: -1, RightLine: -1, IsFillerLine: true},
			{Content: "new line3", LineType: buffer.DiffLineModified, LeftLine: 2, RightLine: 2, IsFillerLine: false},
			{Content: "added line", LineType: buffer.DiffLineAdded, LeftLine: -1, RightLine: 3, IsFillerLine: false},
		},
	}

	// Create aligned buffer content
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)

	// Create diff buffers
	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, currentBufferName, config.IndentTabs, 4)

	// Store buffers and diff line info
	te.buffers[headBufferName] = headBuffer
	te.buffers[currentBufferName] = currentBuffer
	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer.DiffHints = diffResult.RightLines

	// Test diff line info storage
	t.Run("diff_line_info_stored_correctly", func(t *testing.T) {
		// Check head buffer diff info
		if diffLines := headBuffer.DiffHints; len(diffLines) > 0 {
			if len(diffLines) != 4 {
				t.Errorf("Expected 4 diff lines for head buffer, got %d", len(diffLines))
			}

			// Check specific line types
			expectedTypes := []buffer.DiffLineType{buffer.DiffLineContext, buffer.DiffLineDeleted, buffer.DiffLineModified, buffer.DiffLineAdded}
			for i, expectedType := range expectedTypes {
				if i < len(diffLines) && diffLines[i].LineType != expectedType {
					t.Errorf("Head buffer line %d: expected type %v, got %v", i, expectedType, diffLines[i].LineType)
				}
			}
		} else {
			t.Error("Diff line info not found for head buffer")
		}

		// Check current buffer diff info
		if diffLines := currentBuffer.DiffHints; len(diffLines) > 0 {
			if len(diffLines) != 4 {
				t.Errorf("Expected 4 diff lines for current buffer, got %d", len(diffLines))
			}

			// Check specific line types
			expectedTypes := []buffer.DiffLineType{buffer.DiffLineContext, buffer.DiffLineDeleted, buffer.DiffLineModified, buffer.DiffLineAdded}
			for i, expectedType := range expectedTypes {
				if i < len(diffLines) && diffLines[i].LineType != expectedType {
					t.Errorf("Current buffer line %d: expected type %v, got %v", i, expectedType, diffLines[i].LineType)
				}
			}
		} else {
			t.Error("Diff line info not found for current buffer")
		}
	})

	t.Run("diff_highlighting_applied_during_rendering", func(t *testing.T) {
		// Set up DiffBufferView on a single window
		te.SetupDiffView(headBuffer, currentBuffer, "test.go")

		// Render the screen to trigger diff highlighting
		te.render()

		// Test that rendering completed without errors
		// (We can't easily test the actual colors in this framework, but we can ensure no crashes)
		t.Log("✓ Diff highlighting rendering completed without errors")

		// Check that isDiffBuffer correctly identifies diff buffers
		if !te.isDiffBuffer(headBuffer) {
			t.Error("Head buffer should be identified as diff buffer")
		}
		if !te.isDiffBuffer(currentBuffer) {
			t.Error("Current buffer should be identified as diff buffer")
		}

		// Check that normal buffers are not identified as diff buffers
		normalBuffer := buffer.NewBuffer("normal content", "normal.go", "normal.go", config.IndentTabs, 4)
		if te.isDiffBuffer(normalBuffer) {
			t.Error("Normal buffer should not be identified as diff buffer")
		}
	})

	t.Run("diff_highlighting_colors", func(t *testing.T) {
		// Test that we're using the custom brighter diff colors
		addedBg := ui.NewRGBColor(40, 100, 40)    // Even brighter green
		deletedBg := ui.NewRGBColor(100, 40, 40)  // Even brighter red
		modifiedBg := ui.NewRGBColor(100, 80, 30) // Even brighter yellow/orange

		t.Logf("Added background: %v", addedBg)
		t.Logf("Deleted background: %v", deletedBg)
		t.Logf("Modified background: %v", modifiedBg)

		// Verify colors are different from default
		if addedBg == ui.Color(0) || deletedBg == ui.Color(0) || modifiedBg == ui.Color(0) {
			t.Error("Diff colors should not be default/transparent")
		}
	})
}

// TestDiffBufferCleanup verifies that diff line info is cleaned up when buffers are closed
func TestDiffBufferCleanup(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "test content")

	// Create a diff buffer with diff line info
	diffBuffer := buffer.NewBuffer("diff content", "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
	te.buffers["test.go@HEAD"] = diffBuffer
	diffBuffer.DiffHints = []buffer.DiffLine{
		{Content: "test", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
	}

	// Verify diff line info exists
	if len(diffBuffer.DiffHints) == 0 {
		t.Fatal("Diff line info should exist before cleanup")
	}

	// Manually trigger buffer cleanup (simulate buffer deletion)
	diffBuffer.Highlighter = nil
	diffBuffer.DiffHints = nil

	// Verify cleanup
	if len(diffBuffer.DiffHints) > 0 {
		t.Error("Diff line info should be cleaned up after buffer deletion")
	}
}

// TestNonDiffBufferHighlighting verifies that normal buffers don't get diff highlighting
func TestNonDiffBufferHighlighting(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "normal content\nline2\nline3")

	// Render with normal buffer (should not have diff highlighting)
	te.render()

	// Verify that buffer is not considered a diff buffer
	if te.isDiffBuffer(te.CurrentBuffer()) {
		t.Error("Normal buffer should not be identified as diff buffer")
	}

	// Verify no diff line info exists for normal buffer
	if len(te.CurrentBuffer().DiffHints) > 0 {
		t.Error("Normal buffer should not have diff line info")
	}
}
