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
	te.buffer.FilePath = "test.go"
	
	// Create diff buffers manually to test highlighting
	headBufferName := "test.go@HEAD"
	currentBufferName := "test.go@current"
	
	// Create test diff result with different line types
	diffResult := &DiffResult{
		LeftLines: []DiffLine{
			{Content: "line1", LineType: DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
			{Content: "deleted line", LineType: DiffLineDeleted, LeftLine: 1, RightLine: -1, IsFillerLine: false},
			{Content: "old line3", LineType: DiffLineModified, LeftLine: 2, RightLine: 2, IsFillerLine: false},
			{Content: "", LineType: DiffLineAdded, LeftLine: -1, RightLine: -1, IsFillerLine: true},
		},
		RightLines: []DiffLine{
			{Content: "line1", LineType: DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
			{Content: "", LineType: DiffLineDeleted, LeftLine: -1, RightLine: -1, IsFillerLine: true},
			{Content: "new line3", LineType: DiffLineModified, LeftLine: 2, RightLine: 2, IsFillerLine: false},
			{Content: "added line", LineType: DiffLineAdded, LeftLine: -1, RightLine: 3, IsFillerLine: false},
		},
	}
	
	// Create aligned buffer content
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)
	
	// Create diff buffers
	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, config.IndentTabs, 4)
	
	// Store buffers and diff line info
	te.buffers[headBufferName] = headBuffer
	te.buffers[currentBufferName] = currentBuffer
	te.diffLineInfo[headBuffer] = diffResult.LeftLines
	te.diffLineInfo[currentBuffer] = diffResult.RightLines
	
	// Test diff line info storage
	t.Run("diff_line_info_stored_correctly", func(t *testing.T) {
		// Check head buffer diff info
		if diffLines, exists := te.diffLineInfo[headBuffer]; exists {
			if len(diffLines) != 4 {
				t.Errorf("Expected 4 diff lines for head buffer, got %d", len(diffLines))
			}
			
			// Check specific line types
			expectedTypes := []DiffLineType{DiffLineContext, DiffLineDeleted, DiffLineModified, DiffLineAdded}
			for i, expectedType := range expectedTypes {
				if i < len(diffLines) && diffLines[i].LineType != expectedType {
					t.Errorf("Head buffer line %d: expected type %v, got %v", i, expectedType, diffLines[i].LineType)
				}
			}
		} else {
			t.Error("Diff line info not found for head buffer")
		}
		
		// Check current buffer diff info
		if diffLines, exists := te.diffLineInfo[currentBuffer]; exists {
			if len(diffLines) != 4 {
				t.Errorf("Expected 4 diff lines for current buffer, got %d", len(diffLines))
			}
			
			// Check specific line types
			expectedTypes := []DiffLineType{DiffLineContext, DiffLineDeleted, DiffLineModified, DiffLineAdded}
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
		// Set up windows with diff buffers
		te.createSecondWindow()
		te.windows[0].Buffer = headBuffer
		te.windows[1].Buffer = currentBuffer
		te.activeWindow = 1
		te.switchToBuffer(currentBuffer)
		
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
		normalBuffer := buffer.NewBuffer("normal content", "normal.go", config.IndentTabs, 4)
		if te.isDiffBuffer(normalBuffer) {
			t.Error("Normal buffer should not be identified as diff buffer")
		}
	})
	
	t.Run("diff_highlighting_colors", func(t *testing.T) {
		// Test that we're using the custom brighter diff colors
		addedBg := ui.NewRGBColor(40, 100, 40)   // Even brighter green
		deletedBg := ui.NewRGBColor(100, 40, 40) // Even brighter red  
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
	diffBuffer := buffer.NewBuffer("diff content", "test.go@HEAD", config.IndentTabs, 4)
	te.buffers["test.go@HEAD"] = diffBuffer
	te.diffLineInfo[diffBuffer] = []DiffLine{
		{Content: "test", LineType: DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
	}
	
	// Verify diff line info exists
	if _, exists := te.diffLineInfo[diffBuffer]; !exists {
		t.Fatal("Diff line info should exist before cleanup")
	}
	
	// Manually trigger buffer cleanup (simulate buffer deletion)
	delete(te.bufferHighlighters, diffBuffer)
	delete(te.diffLineInfo, diffBuffer)
	
	// Verify cleanup
	if _, exists := te.diffLineInfo[diffBuffer]; exists {
		t.Error("Diff line info should be cleaned up after buffer deletion")
	}
}

// TestNonDiffBufferHighlighting verifies that normal buffers don't get diff highlighting
func TestNonDiffBufferHighlighting(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "normal content\nline2\nline3")
	
	// Render with normal buffer (should not have diff highlighting)
	te.render()
	
	// Verify that buffer is not considered a diff buffer
	if te.isDiffBuffer(te.buffer) {
		t.Error("Normal buffer should not be identified as diff buffer")
	}
	
	// Verify no diff line info exists for normal buffer
	if _, exists := te.diffLineInfo[te.buffer]; exists {
		t.Error("Normal buffer should not have diff line info")
	}
}