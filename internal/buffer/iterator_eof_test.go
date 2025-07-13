package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestIteratorEOFPrevious(t *testing.T) {
	tests := []struct {
		name    string
		content string
		expectedLastLine string
		expectedLineNum  int
	}{
		{
			name:    "buffer ending with newline",
			content: "Line 1\nLine 2\nLine 3\n",
			expectedLastLine: "",  // Empty line after final newline
			expectedLineNum:  3,   // Line 3 is empty
		},
		{
			name:    "buffer not ending with newline",
			content: "Line 1\nLine 2\nLine 3",
			expectedLastLine: "Line 3",
			expectedLineNum:  2,
		},
		{
			name:    "single line with newline",
			content: "Single line\n",
			expectedLastLine: "",
			expectedLineNum:  1,
		},
		{
			name:    "single line without newline",
			content: "Single line",
			expectedLastLine: "Single line",
			expectedLineNum:  0,
		},
		{
			name:    "empty buffer",
			content: "",
			expectedLastLine: "",
			expectedLineNum:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffer(tt.content, "test.txt", config.IndentSpaces, 4)
			
			// Create iterator at EOF
			iter := NewBufferIteratorAtEnd(buf)
			
			// Verify it's at EOF
			if !iter.IsEOF() {
				t.Errorf("Expected iterator to be at EOF")
			}
			
			// Call Previous()
			prevIter := iter.Previous()
			
			// Check that EOF flag is cleared (except for empty buffer)
			if tt.content != "" && prevIter.IsEOF() {
				t.Errorf("Expected EOF flag to be cleared after Previous()")
			}
			
			// Check line number
			if prevIter.LineNumber() != tt.expectedLineNum {
				t.Errorf("Expected line number %d, got %d", tt.expectedLineNum, prevIter.LineNumber())
			}
			
			// Check line content
			if prevIter.Line() != tt.expectedLastLine {
				t.Errorf("Expected line content %q, got %q", tt.expectedLastLine, prevIter.Line())
			}
			
			// Test that we can continue moving backwards
			if tt.expectedLineNum > 0 {
				prevPrevIter := prevIter.Previous()
				if prevPrevIter.LineNumber() != tt.expectedLineNum - 1 {
					t.Errorf("Expected to move to line %d, got %d", tt.expectedLineNum - 1, prevPrevIter.LineNumber())
				}
			}
		})
	}
}

func TestIteratorEOFNavigation(t *testing.T) {
	content := "Line 0\nLine 1\nLine 2\nLine 3\n"
	buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)
	
	// Start at beginning
	iter := NewBufferIterator(buf)
	
	// Move to EOF
	for !iter.IsEOF() {
		iter = iter.Next()
	}
	
	// Now move back through all lines
	// Note: content ends with \n, so there's an empty line at the end (line 4)
	expectedLines := []string{"", "Line 3", "Line 2", "Line 1", "Line 0"}
	expectedLineNums := []int{4, 3, 2, 1, 0}
	
	for i := 0; i < len(expectedLines); i++ {
		iter = iter.Previous()
		
		if iter.LineNumber() != expectedLineNums[i] {
			t.Errorf("Step %d: Expected line number %d, got %d", i, expectedLineNums[i], iter.LineNumber())
		}
		
		if iter.Line() != expectedLines[i] {
			t.Errorf("Step %d: Expected line %q, got %q", i, expectedLines[i], iter.Line())
		}
		
		// After Previous from EOF, should not be EOF anymore
		if iter.IsEOF() {
			t.Errorf("Step %d: Iterator should not be at EOF", i)
		}
	}
	
	// Try to go before line 0
	beforeStart := iter.Previous()
	if beforeStart.LineNumber() != 0 {
		t.Errorf("Should stay at line 0 when trying to go before start")
	}
}