package editor

import (
	"strings"
	"testing"
)

func TestDiffAlignment_SameContent(t *testing.T) {
	original := "line1\nline2\nline3"
	modified := "line1\nline2\nline3"

	result := computeDiff(original, modified)

	// Both sides should have exactly the same number of lines
	if len(result.LeftLines) != len(result.RightLines) {
		t.Errorf("Line count mismatch: left=%d, right=%d", len(result.LeftLines), len(result.RightLines))
	}

	// Should have 3 lines each
	expectedLines := 3
	if len(result.LeftLines) != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, len(result.LeftLines))
	}

	// All lines should be context (unchanged)
	for i, line := range result.LeftLines {
		if line.LineType != DiffLineContext {
			t.Errorf("Line %d should be context, got %v", i, line.LineType)
		}
		if line.IsFillerLine {
			t.Errorf("Line %d should not be filler", i)
		}
	}

	// Content should match original
	leftContent := createBufferContent(result.LeftLines)
	rightContent := createBufferContent(result.RightLines)

	if leftContent != original {
		t.Errorf("Left content mismatch:\nExpected: %q\nGot: %q", original, leftContent)
	}
	if rightContent != modified {
		t.Errorf("Right content mismatch:\nExpected: %q\nGot: %q", modified, rightContent)
	}
}

func TestDiffAlignment_AddedLines(t *testing.T) {
	original := "line1\nline3"
	modified := "line1\nline2\nline3"

	result := computeDiff(original, modified)

	// Both sides should have exactly the same number of lines
	if len(result.LeftLines) != len(result.RightLines) {
		t.Errorf("Line count mismatch: left=%d, right=%d", len(result.LeftLines), len(result.RightLines))
	}

	// Should have 3 lines each (original 2 + 1 added)
	expectedLines := 3
	if len(result.LeftLines) != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, len(result.LeftLines))
	}

	// Check line types: left shows missing line as Deleted (filler), right shows Added (real)
	expectedLeft := []DiffLineType{DiffLineContext, DiffLineDeleted, DiffLineContext}
	expectedRight := []DiffLineType{DiffLineContext, DiffLineAdded, DiffLineContext}
	for i := range expectedLeft {
		if result.LeftLines[i].LineType != expectedLeft[i] {
			t.Errorf("Left line %d: expected type %v, got %v", i, expectedLeft[i], result.LeftLines[i].LineType)
		}
		if result.RightLines[i].LineType != expectedRight[i] {
			t.Errorf("Right line %d: expected type %v, got %v", i, expectedRight[i], result.RightLines[i].LineType)
		}
	}

	// Check that middle line is empty on left side (filler)
	if result.LeftLines[1].Content != "" {
		t.Errorf("Left line 1 should be empty filler, got %q", result.LeftLines[1].Content)
	}
	if !result.LeftLines[1].IsFillerLine {
		t.Error("Left line 1 should be marked as filler")
	}

	// Check that middle line has content on right side
	if result.RightLines[1].Content != "line2" {
		t.Errorf("Right line 1 should be 'line2', got %q", result.RightLines[1].Content)
	}
	if result.RightLines[1].IsFillerLine {
		t.Error("Right line 1 should not be marked as filler")
	}
}

func TestDiffAlignment_DeletedLines(t *testing.T) {
	original := "line1\nline2\nline3"
	modified := "line1\nline3"

	result := computeDiff(original, modified)

	// Both sides should have exactly the same number of lines
	if len(result.LeftLines) != len(result.RightLines) {
		t.Errorf("Line count mismatch: left=%d, right=%d", len(result.LeftLines), len(result.RightLines))
	}

	// Should have 3 lines each (modified 2 + 1 deleted)
	expectedLines := 3
	if len(result.LeftLines) != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, len(result.LeftLines))
	}

	// Check line types: left shows present (deleted in current) as Deleted (real), right shows Added (filler)
	expectedLeft := []DiffLineType{DiffLineContext, DiffLineDeleted, DiffLineContext}
	expectedRight := []DiffLineType{DiffLineContext, DiffLineAdded, DiffLineContext}
	for i := range expectedLeft {
		if result.LeftLines[i].LineType != expectedLeft[i] {
			t.Errorf("Left line %d: expected type %v, got %v", i, expectedLeft[i], result.LeftLines[i].LineType)
		}
		if result.RightLines[i].LineType != expectedRight[i] {
			t.Errorf("Right line %d: expected type %v, got %v", i, expectedRight[i], result.RightLines[i].LineType)
		}
	}

	// Check that middle line has content on left side
	if result.LeftLines[1].Content != "line2" {
		t.Errorf("Left line 1 should be 'line2', got %q", result.LeftLines[1].Content)
	}
	if result.LeftLines[1].IsFillerLine {
		t.Error("Left line 1 should not be marked as filler")
	}

	// Check that middle line is empty on right side (filler)
	if result.RightLines[1].Content != "" {
		t.Errorf("Right line 1 should be empty filler, got %q", result.RightLines[1].Content)
	}
	if !result.RightLines[1].IsFillerLine {
		t.Error("Right line 1 should be marked as filler")
	}
}

func TestDiffAlignment_ModifiedLines(t *testing.T) {
	original := "line1\noriginal line\nline3"
	modified := "line1\nmodified line\nline3"

	result := computeDiff(original, modified)

	// Both sides should have exactly the same number of lines
	if len(result.LeftLines) != len(result.RightLines) {
		t.Errorf("Line count mismatch: left=%d, right=%d", len(result.LeftLines), len(result.RightLines))
	}

	// Should have 3 lines each
	expectedLines := 3
	if len(result.LeftLines) != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, len(result.LeftLines))
	}

	// Check line types
	expectedTypes := []DiffLineType{DiffLineContext, DiffLineModified, DiffLineContext}
	for i, expectedType := range expectedTypes {
		if result.LeftLines[i].LineType != expectedType {
			t.Errorf("Left line %d: expected type %v, got %v", i, expectedType, result.LeftLines[i].LineType)
		}
		if result.RightLines[i].LineType != expectedType {
			t.Errorf("Right line %d: expected type %v, got %v", i, expectedType, result.RightLines[i].LineType)
		}
	}

	// Check content
	if result.LeftLines[1].Content != "original line" {
		t.Errorf("Left line 1 should be 'original line', got %q", result.LeftLines[1].Content)
	}
	if result.RightLines[1].Content != "modified line" {
		t.Errorf("Right line 1 should be 'modified line', got %q", result.RightLines[1].Content)
	}

	// Neither should be filler
	if result.LeftLines[1].IsFillerLine || result.RightLines[1].IsFillerLine {
		t.Error("Modified lines should not be marked as filler")
	}
}

func TestDiffAlignment_ComplexChanges(t *testing.T) {
	original := "line1\nline2\nline4\nline5"
	modified := "line1\nmodified line2\nline3\nline4"

	result := computeDiff(original, modified)

	// Both sides should have exactly the same number of lines
	if len(result.LeftLines) != len(result.RightLines) {
		t.Errorf("Line count mismatch: left=%d, right=%d", len(result.LeftLines), len(result.RightLines))
	}

	// Should have 5 lines each (optimal alignment requires extra lines for proper matching)
	expectedLines := 5
	if len(result.LeftLines) != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, len(result.LeftLines))
	}

	// Log the actual result for debugging
	t.Logf("Left lines:")
	for i, line := range result.LeftLines {
		t.Logf("  %d: %q (type=%v, filler=%v)", i, line.Content, line.LineType, line.IsFillerLine)
	}
	t.Logf("Right lines:")
	for i, line := range result.RightLines {
		t.Logf("  %d: %q (type=%v, filler=%v)", i, line.Content, line.LineType, line.IsFillerLine)
	}
}

func TestDiffAlignment_EmptyFiles(t *testing.T) {
	t.Run("both empty", func(t *testing.T) {
		result := computeDiff("", "")

		if len(result.LeftLines) != len(result.RightLines) {
			t.Errorf("Line count mismatch: left=%d, right=%d", len(result.LeftLines), len(result.RightLines))
		}

		// Empty files should result in 0 lines
		if len(result.LeftLines) != 0 {
			t.Errorf("Expected 0 lines for empty files, got %d", len(result.LeftLines))
		}
	})

	t.Run("original empty", func(t *testing.T) {
		result := computeDiff("", "line1\nline2")

		if len(result.LeftLines) != len(result.RightLines) {
			t.Errorf("Line count mismatch: left=%d, right=%d", len(result.LeftLines), len(result.RightLines))
		}

		expectedLines := 2
		if len(result.LeftLines) != expectedLines {
			t.Errorf("Expected %d lines, got %d", expectedLines, len(result.LeftLines))
		}

		// Original empty: left shows all Deleted fillers; right has Added real lines
		for i, line := range result.LeftLines {
			if line.LineType != DiffLineDeleted {
				t.Errorf("Left line %d should be deleted filler, got %v", i, line.LineType)
			}
			if !line.IsFillerLine {
				t.Errorf("Left line %d should be filler", i)
			}
		}
	})

	t.Run("modified empty", func(t *testing.T) {
		result := computeDiff("line1\nline2", "")

		if len(result.LeftLines) != len(result.RightLines) {
			t.Errorf("Line count mismatch: left=%d, right=%d", len(result.LeftLines), len(result.RightLines))
		}

		expectedLines := 2
		if len(result.LeftLines) != expectedLines {
			t.Errorf("Expected %d lines, got %d", expectedLines, len(result.LeftLines))
		}

		// All right-side lines should be Added filler (missing in modified)
		for i, line := range result.RightLines {
			if line.LineType != DiffLineAdded {
				t.Errorf("Right line %d should be added filler, got %v", i, line.LineType)
			}
			if !line.IsFillerLine {
				t.Errorf("Right line %d should be filler", i)
			}
		}
		// All left-side lines should be Deleted real
		for i, line := range result.LeftLines {
			if line.LineType != DiffLineDeleted {
				t.Errorf("Left line %d should be deleted real, got %v", i, line.LineType)
			}
			if line.IsFillerLine {
				t.Errorf("Left line %d should not be filler", i)
			}
		}
	})
}

func TestDiffAlignment_LineNumberTracking(t *testing.T) {
	original := "line1\nline2\nline4"
	modified := "line1\nline3\nline4"

	result := computeDiff(original, modified)

	// Check that original line numbers are preserved
	for i, line := range result.LeftLines {
		if !line.IsFillerLine {
			if line.LeftLine < 0 {
				t.Errorf("Left line %d should have valid LeftLine number, got %d", i, line.LeftLine)
			}
		} else {
			if line.LeftLine != -1 {
				t.Errorf("Filler line %d should have LeftLine=-1, got %d", i, line.LeftLine)
			}
		}
	}

	// Check that modified line numbers are preserved
	for i, line := range result.RightLines {
		if !line.IsFillerLine {
			if line.RightLine < 0 {
				t.Errorf("Right line %d should have valid RightLine number, got %d", i, line.RightLine)
			}
		} else {
			if line.RightLine != -1 {
				t.Errorf("Filler line %d should have RightLine=-1, got %d", i, line.RightLine)
			}
		}
	}
}

func TestDiffAlignment_BufferContentReconstruction(t *testing.T) {
	testCases := []struct {
		name     string
		original string
		modified string
	}{
		{"simple", "line1\nline2", "line1\nline3"},
		{"additions", "line1", "line1\nline2\nline3"},
		{"deletions", "line1\nline2\nline3", "line1"},
		{"mixed", "a\nb\nd\ne", "a\nc\nd"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := computeDiff(tc.original, tc.modified)

			// Reconstruct content from aligned lines
			leftContent := createBufferContent(result.LeftLines)
			rightContent := createBufferContent(result.RightLines)

			// Count non-empty lines in each reconstruction
			leftNonEmpty := countNonEmptyLines(leftContent)
			rightNonEmpty := countNonEmptyLines(rightContent)
			originalLines := countNonEmptyLines(tc.original)
			modifiedLines := countNonEmptyLines(tc.modified)

			if leftNonEmpty != originalLines {
				t.Errorf("Left reconstruction should preserve %d non-empty lines, got %d", originalLines, leftNonEmpty)
			}
			if rightNonEmpty != modifiedLines {
				t.Errorf("Right reconstruction should preserve %d non-empty lines, got %d", modifiedLines, rightNonEmpty)
			}

			// Total line counts should be equal
			leftTotal := len(strings.Split(leftContent, "\n"))
			rightTotal := len(strings.Split(rightContent, "\n"))
			if leftTotal != rightTotal {
				t.Errorf("Reconstructed content line count mismatch: left=%d, right=%d", leftTotal, rightTotal)
			}
		})
	}
}

func countNonEmptyLines(content string) int {
	if content == "" {
		return 0
	}
	lines := strings.Split(content, "\n")
	count := 0
	for _, line := range lines {
		if line != "" {
			count++
		}
	}
	return count
}
