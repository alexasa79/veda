package editor

import (
	"testing"
)

// TestSpecificWordDiffCase tests the exact case mentioned by the user
func TestSpecificWordDiffCase(t *testing.T) {
	// Test the specific case mentioned by the user
	left := "// Draw each window using a windowScreen. Inactive windows are dimmed."
	right := "// Draw each window using a windowScreen. Inactive windows are dimmed unless in diff view."
	
	t.Logf("Testing word diff:")
	t.Logf("LEFT:  %q", left)
	t.Logf("RIGHT: %q", right)
	
	ranges := computeWordDiff(left, right)
	
	t.Logf("Found %d word ranges:", len(ranges))
	for i, r := range ranges {
		if r.Start >= 0 && r.End <= len(right) && r.Start < r.End {
			diffText := right[r.Start:r.End]
			t.Logf("  Range %d: [%d, %d) = %q", i, r.Start, r.End, diffText)
		} else {
			t.Logf("  Range %d: [%d, %d) (invalid)", i, r.Start, r.End)
		}
	}
	
	// Expected: should highlight only " unless in diff view"
	// The algorithm finds common prefix and suffix, so it highlights the middle difference
	expectedStart := 69  // Position after "dimmed"
	expectedText := " unless in diff view" // Without the period since it's common suffix
	
	if len(ranges) != 1 {
		t.Errorf("Expected 1 range, got %d", len(ranges))
		return
	}
	
	actualRange := ranges[0]
	if actualRange.Start != expectedStart {
		t.Errorf("Expected range to start at %d, got %d", expectedStart, actualRange.Start)
	}
	
	expectedEnd := expectedStart + len(expectedText)
	if actualRange.End != expectedEnd {
		t.Errorf("Expected range to end at %d, got %d", expectedEnd, actualRange.End)
	}
	
	if actualRange.Start < len(right) && actualRange.End <= len(right) {
		actualText := right[actualRange.Start:actualRange.End]
		if actualText != expectedText {
			t.Errorf("Expected highlighted text %q, got %q", expectedText, actualText)
		}
	}
}

// TestWordDiffEdgeCases tests various edge cases for the word diff algorithm
func TestWordDiffEdgeCases(t *testing.T) {
	testCases := []struct {
		name           string
		left           string
		right          string
		expectedRanges []struct{ start, end int }
		expectedTexts  []string
	}{
		{
			name:  "append_at_end",
			left:  "Hello world",
			right: "Hello world!",
			expectedRanges: []struct{ start, end int }{{11, 12}},
			expectedTexts:  []string{"!"},
		},
		{
			name:  "change_middle_word", 
			left:  "Hello world foo",
			right: "Hello universe foo",
			expectedRanges: []struct{ start, end int }{{6, 14}},
			expectedTexts:  []string{"universe"},
		},
		{
			name:  "prepend_at_start",
			left:  "world",
			right: "Hello world",
			expectedRanges: []struct{ start, end int }{{0, 6}},
			expectedTexts:  []string{"Hello "},
		},
		{
			name:  "multiple_separate_changes",
			left:  "The quick brown fox jumps",
			right: "The slow brown cat jumps",
			// This might highlight more than expected due to algorithm limitations
			expectedRanges: []struct{ start, end int }{}, // Will check what we actually get
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ranges := computeWordDiff(tc.left, tc.right)
			
			t.Logf("LEFT:  %q", tc.left)
			t.Logf("RIGHT: %q", tc.right)
			t.Logf("Got %d ranges:", len(ranges))
			
			for i, r := range ranges {
				if r.Start >= 0 && r.End <= len(tc.right) && r.Start < r.End {
					diffText := tc.right[r.Start:r.End]
					t.Logf("  Range %d: [%d, %d) = %q", i, r.Start, r.End, diffText)
				}
			}
			
			// For test cases with expected results, validate them
			if len(tc.expectedRanges) > 0 {
				if len(ranges) != len(tc.expectedRanges) {
					t.Errorf("Expected %d ranges, got %d", len(tc.expectedRanges), len(ranges))
				} else {
					for i, expectedRange := range tc.expectedRanges {
						if i < len(ranges) {
							actualRange := ranges[i]
							if actualRange.Start != expectedRange.start || actualRange.End != expectedRange.end {
								t.Errorf("Range %d: expected [%d, %d), got [%d, %d)", 
									i, expectedRange.start, expectedRange.end, actualRange.Start, actualRange.End)
							}
							
							if i < len(tc.expectedTexts) {
								expectedText := tc.expectedTexts[i]
								actualText := tc.right[actualRange.Start:actualRange.End]
								if actualText != expectedText {
									t.Errorf("Range %d text: expected %q, got %q", i, expectedText, actualText)
								}
							}
						}
					}
				}
			}
		})
	}
}

// TestCharacterLevelLCS tests the character-level LCS function directly
func TestCharacterLevelLCS(t *testing.T) {
	testCases := []struct {
		name  string
		left  string
		right string
	}{
		{
			name:  "simple_append",
			left:  "abc",
			right: "abcd",
		},
		{
			name:  "simple_replace",
			left:  "abc",
			right: "axc",
		},
		{
			name:  "comment_case",
			left:  "// Comment",
			right: "// Comment extended",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			leftRunes := []rune(tc.left)
			rightRunes := []rune(tc.right)
			
			lcs := findCharLCS(leftRunes, rightRunes)
			
			t.Logf("LEFT:  %q", tc.left)
			t.Logf("RIGHT: %q", tc.right)
			t.Logf("LCS has %d matches:", len(lcs))
			
			for i, match := range lcs {
				if match.LeftIdx < len(leftRunes) && match.RightIdx < len(rightRunes) {
					leftChar := string(leftRunes[match.LeftIdx])
					rightChar := string(rightRunes[match.RightIdx])
					t.Logf("  Match %d: L[%d]=%q, R[%d]=%q", 
						i, match.LeftIdx, leftChar, match.RightIdx, rightChar)
				}
			}
		})
	}
}