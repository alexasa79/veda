package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/ui"
)

// TestWordLevelDiffHighlighting verifies that only changed portions of modified lines are highlighted
func TestWordLevelDiffHighlighting(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")

	// Create test content with word-level changes
	originalContent := "Hello world, this is a test\nSecond line unchanged\nAnother test line"
	modifiedContent := "Hello universe, this is a demo\nSecond line unchanged\nAnother example line"

	// Create diff result
	diffResult := computeDiff(originalContent, modifiedContent)

	// Create aligned buffer content
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)

	// Create diff buffers
	headBufferName := "test.go@HEAD"
	currentBufferName := "test.go@current"

	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, config.IndentTabs, 4)

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

	// Expected modified background color
	expectedModifiedBg := ui.NewRGBColor(100, 80, 30) // Even brighter yellow/orange background

	t.Run("word_level_highlighting_in_current_buffer", func(t *testing.T) {
		// Switch to current buffer and render
		te.activeWindow = 1
		te.switchToBuffer(currentBuffer)

		// Move cursor away from current line to avoid interference
		te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 2)

		te.drawEditor()

		// Find the content area (skip gutter columns)
		contentStartX := te.CurrentBuffer().LineNumberWidth
		t.Logf("Using content start X: %d (line number width)", contentStartX)

		// Check first line - should have word-level highlighting
		// "Hello universe, this is a demo" vs "Hello world, this is a test"
		// Expected highlights: "universe" and "demo" should be highlighted
		for lineIdx, diffLine := range diffResult.RightLines {
			if lineIdx >= 24-1 { // Skip if beyond screen height
				break
			}

			if diffLine.LineType == DiffLineModified {
				t.Logf("Checking modified line %d: %q with %d word ranges", lineIdx, diffLine.Content, len(diffLine.WordRanges))

				// Log the word ranges for debugging
				for i, wordRange := range diffLine.WordRanges {
					t.Logf("  Word range %d: [%d, %d) = %q", i, wordRange.Start, wordRange.End, 
						diffLine.Content[wordRange.Start:wordRange.End])
				}

				// Check some characters in the line to see if highlighting is applied correctly
				// We'll check a few key positions and verify the background color
				mockScreen := te.screen.(*MockScreen)

				// Sample a few characters across the line
				lineLen := len(diffLine.Content)
				samplePositions := []int{0, lineLen/4, lineLen/2, 3*lineLen/4, lineLen-1}

				for _, samplePos := range samplePositions {
					if samplePos >= lineLen {
						continue
					}

					screenX := contentStartX + samplePos
					_, style := mockScreen.GetCell(screenX, lineIdx)
					_, actualBg, _ := style.Decompose()

					// Check if this position should be highlighted based on word ranges
					shouldBeHighlighted := isInWordRange(samplePos, diffLine.WordRanges)

					if shouldBeHighlighted {
						if actualBg != expectedModifiedBg {
							t.Errorf("Line %d, pos %d (%q): expected highlighted background %v, got %v", 
								lineIdx, samplePos, string(diffLine.Content[samplePos]), expectedModifiedBg, actualBg)
						} else {
							t.Logf("✓ Line %d, pos %d (%q): correctly highlighted with %v", 
								lineIdx, samplePos, string(diffLine.Content[samplePos]), actualBg)
						}
					} else {
						if actualBg == expectedModifiedBg {
							t.Errorf("Line %d, pos %d (%q): unexpected highlighted background %v (should not be highlighted)", 
								lineIdx, samplePos, string(diffLine.Content[samplePos]), actualBg)
						} else {
							t.Logf("✓ Line %d, pos %d (%q): correctly not highlighted (background %v)", 
								lineIdx, samplePos, string(diffLine.Content[samplePos]), actualBg)
						}
					}
				}
			} else if diffLine.LineType == DiffLineContext {
				t.Logf("Context line %d: %q (should not be highlighted)", lineIdx, diffLine.Content)
			}
		}
	})

	t.Run("word_ranges_computed_correctly", func(t *testing.T) {
		// Verify that word ranges are computed for modified lines
		modifiedLineFound := false
		for _, diffLine := range diffResult.RightLines {
			if diffLine.LineType == DiffLineModified {
				modifiedLineFound = true
				if len(diffLine.WordRanges) == 0 {
					t.Errorf("Modified line should have word ranges: %q", diffLine.Content)
				} else {
					t.Logf("Modified line %q has %d word ranges", diffLine.Content, len(diffLine.WordRanges))
					for i, wordRange := range diffLine.WordRanges {
						if wordRange.Start < 0 || wordRange.End > len(diffLine.Content) || wordRange.Start >= wordRange.End {
							t.Errorf("Invalid word range %d: [%d, %d) for line %q", i, wordRange.Start, wordRange.End, diffLine.Content)
						} else {
							t.Logf("  Word range %d: [%d, %d) = %q", i, wordRange.Start, wordRange.End, 
								diffLine.Content[wordRange.Start:wordRange.End])
						}
					}
				}
			}
		}

		if !modifiedLineFound {
			t.Error("Expected to find at least one modified line in diff")
		}
	})

	t.Run("context_lines_not_highlighted", func(t *testing.T) {
		// Switch to current buffer and render
		te.activeWindow = 1
		te.switchToBuffer(currentBuffer)
		te.drawEditor()

		contentStartX := te.CurrentBuffer().LineNumberWidth
		mockScreen := te.screen.(*MockScreen)

		// Check that context lines (unchanged lines) are not highlighted
		for lineIdx, diffLine := range diffResult.RightLines {
			if lineIdx >= 24-1 {
				break
			}

			if diffLine.LineType == DiffLineContext {
				// Sample a few characters in the context line
				lineLen := len(diffLine.Content)
				if lineLen > 0 {
					samplePositions := []int{0, lineLen/2, lineLen-1}

					for _, samplePos := range samplePositions {
						if samplePos >= lineLen {
							continue
						}

						screenX := contentStartX + samplePos
						_, style := mockScreen.GetCell(screenX, lineIdx)
						_, actualBg, _ := style.Decompose()

						if actualBg == expectedModifiedBg {
							t.Errorf("Context line %d, pos %d should not have diff highlighting background %v", 
								lineIdx, samplePos, actualBg)
						}
					}
				}
			}
		}
	})
}

// TestWordDiffAlgorithm tests the word-level diff computation function directly
func TestWordDiffAlgorithm(t *testing.T) {
	testCases := []struct {
		name       string
		left       string
		right      string
		expectLen  int // Expected number of ranges
	}{
		{
			name:      "simple_word_change",
			left:      "Hello world",
			right:     "Hello universe",
			expectLen: 1, // "universe" should be highlighted
		},
		{
			name:      "multiple_changes",
			left:      "The quick brown fox",
			right:     "The slow brown cat",
			expectLen: 2, // "slow" and "cat" should be highlighted
		},
		{
			name:      "no_changes",
			left:      "Same content",
			right:     "Same content",
			expectLen: 0, // No differences
		},
		{
			name:      "complete_change",
			left:      "abc",
			right:     "xyz",
			expectLen: 1, // Entire content is different
		},
		{
			name:      "insertion_at_end",
			left:      "Hello",
			right:     "Hello world",
			expectLen: 1, // " world" should be highlighted
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ranges := computeWordDiff(tc.left, tc.right)
			
			if len(ranges) != tc.expectLen {
				t.Errorf("Expected %d ranges, got %d for left=%q, right=%q", 
					tc.expectLen, len(ranges), tc.left, tc.right)
			}

			// Log the ranges for debugging
			t.Logf("Word diff for %q -> %q:", tc.left, tc.right)
			for i, r := range ranges {
				if r.Start < len(tc.right) && r.End <= len(tc.right) {
					t.Logf("  Range %d: [%d, %d) = %q", i, r.Start, r.End, tc.right[r.Start:r.End])
				} else {
					t.Logf("  Range %d: [%d, %d) (invalid bounds)", i, r.Start, r.End)
				}
			}

			// Verify ranges are valid
			for i, r := range ranges {
				if r.Start < 0 || r.End > len(tc.right) || r.Start >= r.End {
					t.Errorf("Invalid range %d: [%d, %d) for string of length %d", 
						i, r.Start, r.End, len(tc.right))
				}
			}
		})
	}
}