package editor

import (
	"fmt"
	"strings"
	"testing"
)

// TestDiffLineCountEquality focuses on ensuring both diff buffers always have exactly the same line count
// This is critical for proper cursor synchronization and scrolling behavior
func TestDiffLineCountEquality(t *testing.T) {
	testCases := []struct {
		name     string
		original string
		modified string
		desc     string
	}{
		{
			name:     "identical_content",
			original: "line1\nline2\nline3",
			modified: "line1\nline2\nline3",
			desc:     "Identical content should have equal line counts",
		},
		{
			name:     "single_line_addition",
			original: "line1\nline3",
			modified: "line1\nline2\nline3",
			desc:     "Single line addition should have equal line counts with filler",
		},
		{
			name:     "single_line_deletion",
			original: "line1\nline2\nline3",
			modified: "line1\nline3",
			desc:     "Single line deletion should have equal line counts with filler",
		},
		{
			name:     "single_line_modification",
			original: "line1\noriginal\nline3",
			modified: "line1\nmodified\nline3",
			desc:     "Single line modification should have equal line counts",
		},
		{
			name:     "multiple_additions",
			original: "line1\nline4",
			modified: "line1\nline2\nline3\nline4",
			desc:     "Multiple additions should have equal line counts with fillers",
		},
		{
			name:     "multiple_deletions",
			original: "line1\nline2\nline3\nline4",
			modified: "line1\nline4",
			desc:     "Multiple deletions should have equal line counts with fillers",
		},
		{
			name:     "mixed_changes",
			original: "line1\nold2\nline3\nold4\nline5",
			modified: "line1\nnew2\nline3\nnew4\nline5",
			desc:     "Mixed modifications should have equal line counts",
		},
		{
			name:     "additions_at_beginning",
			original: "line3\nline4",
			modified: "line1\nline2\nline3\nline4",
			desc:     "Additions at beginning should have equal line counts",
		},
		{
			name:     "deletions_at_beginning",
			original: "line1\nline2\nline3\nline4",
			modified: "line3\nline4",
			desc:     "Deletions at beginning should have equal line counts",
		},
		{
			name:     "additions_at_end",
			original: "line1\nline2",
			modified: "line1\nline2\nline3\nline4",
			desc:     "Additions at end should have equal line counts",
		},
		{
			name:     "deletions_at_end",
			original: "line1\nline2\nline3\nline4",
			modified: "line1\nline2",
			desc:     "Deletions at end should have equal line counts",
		},
		{
			name:     "complex_reorganization",
			original: "A\nB\nC\nD\nE",
			modified: "A\nC\nX\nD\nY\nE",
			desc:     "Complex reorganization should have equal line counts",
		},
		{
			name:     "empty_to_content",
			original: "",
			modified: "line1\nline2\nline3",
			desc:     "Empty to content should have equal line counts",
		},
		{
			name:     "content_to_empty",
			original: "line1\nline2\nline3",
			modified: "",
			desc:     "Content to empty should have equal line counts",
		},
		{
			name:     "single_line_to_single_line",
			original: "original",
			modified: "modified",
			desc:     "Single line to single line should have equal line counts",
		},
		{
			name:     "single_line_to_multiple",
			original: "original",
			modified: "line1\nline2\nline3",
			desc:     "Single line to multiple should have equal line counts",
		},
		{
			name:     "multiple_to_single_line",
			original: "line1\nline2\nline3",
			modified: "single",
			desc:     "Multiple to single line should have equal line counts",
		},
		{
			name:     "whitespace_only_changes",
			original: "line1\n\nline3",
			modified: "line1\n  \nline3",
			desc:     "Whitespace-only changes should have equal line counts",
		},
		{
			name:     "very_large_diff",
			original: generateLargeContent(100, "original"),
			modified: generateLargeContent(150, "modified"),
			desc:     "Very large diffs should have equal line counts",
		},
		{
			name:     "alternating_pattern",
			original: "keep1\nremove1\nkeep2\nremove2\nkeep3",
			modified: "keep1\nadd1\nkeep2\nadd2\nkeep3",
			desc:     "Alternating add/remove pattern should have equal line counts",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the core diff algorithm
			result := computeDiff(tc.original, tc.modified)
			
			// Critical check: line counts must be equal
			if len(result.LeftLines) != len(result.RightLines) {
				t.Errorf("CRITICAL: Line count mismatch - Left: %d, Right: %d",
					len(result.LeftLines), len(result.RightLines))
				t.Errorf("Test case: %s", tc.desc)
				t.Errorf("Original: %q", tc.original)
				t.Errorf("Modified: %q", tc.modified)
				
				// Debug output to understand the mismatch
				t.Logf("Left lines (%d):", len(result.LeftLines))
				for i, line := range result.LeftLines {
					t.Logf("  %d: %q (type=%v, filler=%v)", i, line.Content, line.LineType, line.IsFillerLine)
				}
				t.Logf("Right lines (%d):", len(result.RightLines))
				for i, line := range result.RightLines {
					t.Logf("  %d: %q (type=%v, filler=%v)", i, line.Content, line.LineType, line.IsFillerLine)
				}
				
				return // Don't continue with other checks if this fails
			}
			
			// Verify that reconstructed content preserves non-empty lines correctly
			leftContent := createBufferContent(result.LeftLines)
			rightContent := createBufferContent(result.RightLines)
			
			leftNonEmpty := countNonEmptyLines(leftContent)
			rightNonEmpty := countNonEmptyLines(rightContent)
			originalNonEmpty := countNonEmptyLines(tc.original)
			modifiedNonEmpty := countNonEmptyLines(tc.modified)
			
			// Verify content reconstruction preserves original line counts
			if leftNonEmpty != originalNonEmpty {
				t.Errorf("Left reconstruction lost content: expected %d non-empty lines, got %d",
					originalNonEmpty, leftNonEmpty)
			}
			if rightNonEmpty != modifiedNonEmpty {
				t.Errorf("Right reconstruction lost content: expected %d non-empty lines, got %d",
					modifiedNonEmpty, rightNonEmpty)
			}
			
			// Verify that every line has the correct filler status
			for i := 0; i < len(result.LeftLines); i++ {
				leftLine := result.LeftLines[i]
				rightLine := result.RightLines[i]
				
				// If one side is a filler, the other should have content (except for context lines)
				if leftLine.IsFillerLine && rightLine.IsFillerLine {
					t.Errorf("Line %d: Both sides cannot be fillers", i)
				}
				
				// Filler lines should be empty
				if leftLine.IsFillerLine && leftLine.Content != "" {
					t.Errorf("Line %d: Left filler line should be empty, got %q", i, leftLine.Content)
				}
				if rightLine.IsFillerLine && rightLine.Content != "" {
					t.Errorf("Line %d: Right filler line should be empty, got %q", i, rightLine.Content)
				}
			}
			
			t.Logf("✓ %s: Equal line counts (%d each)", tc.desc, len(result.LeftLines))
		})
	}
}

// TestDiffLineCountInvariant tests that line count equality is maintained across various edge cases
func TestDiffLineCountInvariant(t *testing.T) {
	// Test property: For any two strings A and B, computeDiff(A, B) should always
	// produce LeftLines and RightLines with exactly the same length
	
	testPairs := []struct {
		a, b string
	}{
		{"", ""},
		{"", "a"},
		{"a", ""},
		{"a", "a"},
		{"a", "b"},
		{"a\n", "a"},
		{"a", "a\n"},
		{"a\nb", "a\nb"},
		{"a\nb", "a\nc"},
		{"a\nb", "a\n\nb"},
		{"a\n\nb", "a\nb"},
		{"\n", ""},
		{"", "\n"},
		{"\n\n", "\n"},
		{"\n", "\n\n"},
		{"a\n\n", "a\n"},
		{"a\n", "a\n\n"},
	}
	
	for i, pair := range testPairs {
		t.Run(fmt.Sprintf("pair_%d", i), func(t *testing.T) {
			result := computeDiff(pair.a, pair.b)
			
			if len(result.LeftLines) != len(result.RightLines) {
				t.Errorf("Invariant violated: Left=%d, Right=%d",
					len(result.LeftLines), len(result.RightLines))
				t.Errorf("Input A: %q", pair.a)
				t.Errorf("Input B: %q", pair.b)
			}
		})
	}
}

// TestDiffWithRealFileContent tests diff algorithm with realistic file content
func TestDiffWithRealFileContent(t *testing.T) {
	// Simulate realistic file changes
	originalCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	modifiedCode := `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Println("Hello,", os.Args[1])
	} else {
		fmt.Println("Hello, World!")
	}
}
`

	result := computeDiff(originalCode, modifiedCode)
	
	// Critical check
	if len(result.LeftLines) != len(result.RightLines) {
		t.Errorf("CRITICAL: Real file diff failed line count equality")
		t.Errorf("Left: %d, Right: %d", len(result.LeftLines), len(result.RightLines))
		
		// Debug output
		t.Logf("Original code:\n%s", originalCode)
		t.Logf("Modified code:\n%s", modifiedCode)
		
		t.Logf("Left lines (%d):", len(result.LeftLines))
		for i, line := range result.LeftLines {
			t.Logf("  %d: %q (type=%v, filler=%v)", i, line.Content, line.LineType, line.IsFillerLine)
		}
		t.Logf("Right lines (%d):", len(result.RightLines))
		for i, line := range result.RightLines {
			t.Logf("  %d: %q (type=%v, filler=%v)", i, line.Content, line.LineType, line.IsFillerLine)
		}
	}
}

// TestDiffStressTest performs stress testing with various random inputs
func TestDiffStressTest(t *testing.T) {
	// Test with various combinations of line counts
	lineCounts := []int{0, 1, 2, 5, 10, 25, 50}
	
	for _, leftCount := range lineCounts {
		for _, rightCount := range lineCounts {
			t.Run(fmt.Sprintf("lines_%d_vs_%d", leftCount, rightCount), func(t *testing.T) {
				left := generateLargeContent(leftCount, "left")
				right := generateLargeContent(rightCount, "right")
				
				result := computeDiff(left, right)
				
				if len(result.LeftLines) != len(result.RightLines) {
					t.Errorf("Stress test failed: %d lines vs %d lines", leftCount, rightCount)
					t.Errorf("Result: Left=%d, Right=%d", len(result.LeftLines), len(result.RightLines))
				}
			})
		}
	}
}

// Helper function to generate large content for testing
func generateLargeContent(lineCount int, prefix string) string {
	if lineCount == 0 {
		return ""
	}
	
	lines := make([]string, lineCount)
	for i := 0; i < lineCount; i++ {
		lines[i] = fmt.Sprintf("%s line %d content here", prefix, i+1)
	}
	return strings.Join(lines, "\n")
}

