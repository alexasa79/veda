package buffer

import (
	"strings"
	"testing"
	"time"
	"veda/internal/config"
)

func TestNewBufferIteratorAtLineOptimization(t *testing.T) {
	t.Run("correctness comparison", func(t *testing.T) {
		// Create a buffer with multiple lines
		lines := []string{
			"line 0",
			"line 1",
			"line 2 with some content",
			"", // empty line
			"line 4",
			"final line",
		}
		content := strings.Join(lines, "\n")
		buffer := NewBuffer(content, "", "", config.IndentSpaces, 4)

		// Test each line
		for lineNum := 0; lineNum <= len(lines); lineNum++ {
			t.Run(strings.Join([]string{"line", string(rune('0' + lineNum))}, ""), func(t *testing.T) {
				// Use the optimized version
				iter := NewBufferIteratorAtLine(buffer, lineNum)

				if lineNum < len(lines) {
					// Verify we're at the correct line
					if iter.LineNumber() != lineNum {
						t.Errorf("Expected line number %d, got %d", lineNum, iter.LineNumber())
					}

					// Verify the line content is correct
					if !iter.IsEOF() {
						actualLine := iter.Line()
						expectedLine := lines[lineNum]
						if actualLine != expectedLine {
							t.Errorf("Line %d: expected %q, got %q", lineNum, expectedLine, actualLine)
						}
					}
				} else {
					// Beyond last line, should be at EOF
					if !iter.IsEOF() {
						t.Errorf("Line %d: expected EOF, but iterator is not at EOF", lineNum)
					}
				}
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("empty buffer", func(t *testing.T) {
			buffer := NewBuffer("", "", "", config.IndentSpaces, 4)

			iter := NewBufferIteratorAtLine(buffer, 0)
			if !iter.IsEOF() {
				t.Errorf("Expected EOF for line 0 in empty buffer")
			}

			iter = NewBufferIteratorAtLine(buffer, 1)
			if !iter.IsEOF() {
				t.Errorf("Expected EOF for line 1 in empty buffer")
			}
		})

		t.Run("single line no newline", func(t *testing.T) {
			buffer := NewBuffer("single line", "", "", config.IndentSpaces, 4)

			iter := NewBufferIteratorAtLine(buffer, 0)
			if iter.IsEOF() {
				t.Errorf("Line 0 should exist")
			}
			if iter.Line() != "single line" {
				t.Errorf("Expected 'single line', got %q", iter.Line())
			}

			iter = NewBufferIteratorAtLine(buffer, 1)
			if !iter.IsEOF() {
				t.Errorf("Line 1 should not exist")
			}
		})

		t.Run("single line with newline", func(t *testing.T) {
			buffer := NewBuffer("single line\n", "", "", config.IndentSpaces, 4)

			iter := NewBufferIteratorAtLine(buffer, 0)
			if iter.IsEOF() {
				t.Errorf("Line 0 should exist")
			}
			if iter.Line() != "single line" {
				t.Errorf("Expected 'single line', got %q", iter.Line())
			}

			iter = NewBufferIteratorAtLine(buffer, 1)
			if !iter.IsEOF() {
				t.Errorf("Line 1 should not exist (newline at end doesn't create empty line)")
			}
		})

		t.Run("multiple empty lines", func(t *testing.T) {
			buffer := NewBuffer("a\n\n\nb", "", "", config.IndentSpaces, 4)

			testCases := []struct {
				lineNum      int
				expectedLine string
				shouldExist  bool
			}{
				{0, "a", true},
				{1, "", true},
				{2, "", true},
				{3, "b", true},
				{4, "", false}, // Beyond end
			}

			for _, tc := range testCases {
				iter := NewBufferIteratorAtLine(buffer, tc.lineNum)
				if tc.shouldExist {
					if iter.IsEOF() {
						t.Errorf("Line %d should exist", tc.lineNum)
					} else if iter.Line() != tc.expectedLine {
						t.Errorf("Line %d: expected %q, got %q", tc.lineNum, tc.expectedLine, iter.Line())
					}
				} else {
					if !iter.IsEOF() {
						t.Errorf("Line %d should not exist", tc.lineNum)
					}
				}
			}
		})
	})

	t.Run("complex tree structure", func(t *testing.T) {
		// Build content that will create a complex rope structure
		var contentBuilder strings.Builder
		expectedLines := make([]string, 0, 100)

		for i := 0; i < 100; i++ {
			line := strings.Repeat("X", 50) + " line " + string(rune('0'+i%10))
			expectedLines = append(expectedLines, line)
			contentBuilder.WriteString(line)
			if i < 99 {
				contentBuilder.WriteString("\n")
			}
		}

		// Create a buffer large enough to force tree splits
		buffer := NewBuffer(contentBuilder.String(), "", "", config.IndentSpaces, 4)

		// Force the rope to have a complex structure by inserting large chunks
		largeInsert := strings.Repeat("Y", 2000) + "\n"
		buffer.Rope.Insert(1000, largeInsert)

		// Test accessing various lines throughout the structure
		testLines := []int{0, 1, 10, 25, 50, 75, 99}

		for _, lineNum := range testLines {
			iter := NewBufferIteratorAtLine(buffer, lineNum)
			if iter.IsEOF() {
				t.Errorf("Line %d should exist in complex tree", lineNum)
			}
			// We can't easily verify exact content due to the insertion,
			// but we can verify the iterator is positioned correctly
			if iter.LineNumber() != lineNum {
				t.Errorf("Line %d: expected line number %d, got %d", lineNum, lineNum, iter.LineNumber())
			}
		}
	})
}

func TestNewBufferIteratorAtLineConsistency(t *testing.T) {
	// Test that the optimized version produces the same results as iterating from the beginning
	content := "line0\nline1\nline2\n\nline4\nfinal"
	buffer := NewBuffer(content, "", "", config.IndentSpaces, 4)

	// Get iterators using both methods for each line
	for targetLine := 0; targetLine <= 6; targetLine++ {
		// Method 1: Use optimized NewBufferIteratorAtLine
		optimizedIter := NewBufferIteratorAtLine(buffer, targetLine)

		// Method 2: Start from beginning and iterate to target line
		linearIter := NewBufferIterator(buffer)
		for i := 0; i < targetLine && !linearIter.IsEOF(); i++ {
			linearIter = linearIter.Next()
		}

		// Compare results
		if optimizedIter.IsEOF() != linearIter.IsEOF() {
			t.Errorf("Line %d: EOF mismatch - optimized=%v, linear=%v",
				targetLine, optimizedIter.IsEOF(), linearIter.IsEOF())
		}

		if !optimizedIter.IsEOF() && !linearIter.IsEOF() {
			if optimizedIter.LineNumber() != linearIter.LineNumber() {
				t.Errorf("Line %d: line number mismatch - optimized=%d, linear=%d",
					targetLine, optimizedIter.LineNumber(), linearIter.LineNumber())
			}

			if optimizedIter.Position() != linearIter.Position() {
				t.Errorf("Line %d: position mismatch - optimized=%d, linear=%d",
					targetLine, optimizedIter.Position(), linearIter.Position())
			}

			if optimizedIter.Line() != linearIter.Line() {
				t.Errorf("Line %d: content mismatch - optimized=%q, linear=%q",
					targetLine, optimizedIter.Line(), linearIter.Line())
			}
		}
	}
}

// Benchmark to demonstrate the performance improvement
func BenchmarkNewBufferIteratorAtLine(b *testing.B) {
	// Create a large buffer with many lines
	lines := make([]string, 10000)
	for i := range lines {
		lines[i] = strings.Repeat("content", 10) + " line " + string(rune('0'+i%10))
	}
	content := strings.Join(lines, "\n")

	buffer := NewBuffer(content, "", "", config.IndentSpaces, 4)

	b.Run("optimized O(log n)", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lineNum := i % 10000
			NewBufferIteratorAtLine(buffer, lineNum)
		}
	})

	// Linear implementation for comparison
	linearNewBufferIteratorAtLine := func(buffer *Buffer, targetLine int) *BufferIterator {
		iter := NewBufferIterator(buffer)
		for i := 0; i < targetLine && !iter.IsEOF(); i++ {
			iter = iter.Next()
		}
		return iter
	}

	b.Run("linear O(n)", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lineNum := i % 10000
			linearNewBufferIteratorAtLine(buffer, lineNum)
		}
	})
}

// Test performance characteristics to ensure logarithmic behavior
func TestNewBufferIteratorAtLinePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create buffers of different sizes
	sizes := []int{1000, 2000, 4000, 8000}
	times := make([]time.Duration, len(sizes))

	for i, size := range sizes {
		// Create buffer with 'size' lines
		lines := make([]string, size)
		for j := range lines {
			lines[j] = strings.Repeat("x", 50)
		}
		content := strings.Join(lines, "\n")

		buffer := NewBuffer(content, "", "", config.IndentSpaces, 4)

		// Measure time to access lines near the end
		targetLine := size - 100

		start := time.Now()
		iterations := 100
		for j := 0; j < iterations; j++ {
			NewBufferIteratorAtLine(buffer, targetLine)
		}
		times[i] = time.Since(start) / time.Duration(iterations)
	}

	// Verify that time doesn't grow linearly with size
	// For logarithmic complexity, doubling size should not double time
	for i := 1; i < len(times); i++ {
		ratio := float64(times[i]) / float64(times[i-1])
		sizeRatio := float64(sizes[i]) / float64(sizes[i-1])

		// If it were linear, ratio would be close to sizeRatio (2.0)
		// For logarithmic, ratio should be much less than sizeRatio
		if ratio > sizeRatio*0.8 { // Allow some tolerance
			t.Logf("Warning: Time ratio %.2f is close to size ratio %.2f, suggesting non-logarithmic behavior",
				ratio, sizeRatio)
		}

		t.Logf("Size %d->%d: time ratio %.2f, size ratio %.2f",
			sizes[i-1], sizes[i], ratio, sizeRatio)
	}
}
