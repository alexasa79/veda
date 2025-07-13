package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestIteratorPreviousWithEmptyLines(t *testing.T) {
	// Test content with empty lines (two consecutive \n characters)
	content := "Line 1\n\nLine 3\n\n\nLine 6"
	buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)
	
	// Expected lines from start to end:
	// Line 0: "Line 1"
	// Line 1: "" (empty)
	// Line 2: "Line 3"
	// Line 3: "" (empty)
	// Line 4: "" (empty)
	// Line 5: "Line 6"
	
	// Start from the end and go backwards
	iter := NewBufferIterator(buf)
	
	// Move to line 5 (last line)
	for i := 0; i < 5; i++ {
		iter = iter.Next()
	}
	
	// Verify we're at line 5
	if iter.LineNumber() != 5 || iter.Line() != "Line 6" {
		t.Fatalf("Expected to be at line 5 with content 'Line 6', got line %d with content %q", 
			iter.LineNumber(), iter.Line())
	}
	
	// Now test Previous() through all lines
	testCases := []struct {
		expectedLineNum int
		expectedContent string
	}{
		{4, ""},       // empty line
		{3, ""},       // empty line
		{2, "Line 3"},
		{1, ""},       // empty line
		{0, "Line 1"},
	}
	
	for _, tc := range testCases {
		iter = iter.Previous()
		if iter.LineNumber() != tc.expectedLineNum {
			t.Errorf("Expected line number %d, got %d", tc.expectedLineNum, iter.LineNumber())
		}
		if iter.Line() != tc.expectedContent {
			t.Errorf("Expected line content %q at line %d, got %q", 
				tc.expectedContent, tc.expectedLineNum, iter.Line())
		}
	}
	
	// Try to go before line 0
	iter = iter.Previous()
	if iter.LineNumber() != 0 {
		t.Errorf("Should stay at line 0 when trying to go before start")
	}
}

func TestIteratorNavigationThroughEmptyLines(t *testing.T) {
	// Test with multiple consecutive empty lines
	content := "\n\n\nLine 4\n\n"
	buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)
	
	// Forward iteration first to establish expected behavior
	iter := NewBufferIterator(buf)
	
	expectedForward := []struct {
		lineNum int
		content string
	}{
		{0, ""},       // First empty line
		{1, ""},       // Second empty line
		{2, ""},       // Third empty line
		{3, "Line 4"},
		{4, ""},       // Last empty line
		{5, ""},       // After final \n
	}
	
	// Test forward iteration
	for i, expected := range expectedForward {
		if iter.LineNumber() != expected.lineNum {
			t.Errorf("Forward[%d]: Expected line %d, got %d", i, expected.lineNum, iter.LineNumber())
		}
		if iter.Line() != expected.content {
			t.Errorf("Forward[%d]: Expected content %q, got %q", i, expected.content, iter.Line())
		}
		
		if i < len(expectedForward)-1 {
			iter = iter.Next()
		}
	}
	
	// Now test backward iteration
	for i := len(expectedForward) - 2; i >= 0; i-- {
		iter = iter.Previous()
		expected := expectedForward[i]
		
		if iter.LineNumber() != expected.lineNum {
			t.Errorf("Backward[%d]: Expected line %d, got %d", i, expected.lineNum, iter.LineNumber())
		}
		if iter.Line() != expected.content {
			t.Errorf("Backward[%d]: Expected content %q, got %q", i, expected.content, iter.Line())
		}
	}
}

func TestIteratorPreviousEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		startLine int
		expectedPrevLine int
		expectedPrevContent string
	}{
		{
			name: "two consecutive newlines",
			content: "A\n\nB",
			startLine: 2,  // Start at "B"
			expectedPrevLine: 1,
			expectedPrevContent: "",
		},
		{
			name: "three consecutive newlines",
			content: "A\n\n\nB",
			startLine: 3,  // Start at "B"
			expectedPrevLine: 2,
			expectedPrevContent: "",
		},
		{
			name: "empty line at start",
			content: "\nA",
			startLine: 1,  // Start at "A"
			expectedPrevLine: 0,
			expectedPrevContent: "",
		},
		{
			name: "empty line at end",
			content: "A\n",
			startLine: 1,  // Start at empty line after "A"
			expectedPrevLine: 0,
			expectedPrevContent: "A",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffer(tt.content, "test.txt", config.IndentSpaces, 4)
			iter := NewBufferIterator(buf)
			
			// Move to start line
			for i := 0; i < tt.startLine; i++ {
				iter = iter.Next()
			}
			
			// Verify we're at the right starting position
			if iter.LineNumber() != tt.startLine {
				t.Fatalf("Failed to reach start line %d, at line %d", tt.startLine, iter.LineNumber())
			}
			
			// Test Previous()
			prev := iter.Previous()
			if prev.LineNumber() != tt.expectedPrevLine {
				t.Errorf("Expected previous line to be %d, got %d", tt.expectedPrevLine, prev.LineNumber())
			}
			if prev.Line() != tt.expectedPrevContent {
				t.Errorf("Expected previous line content %q, got %q", tt.expectedPrevContent, prev.Line())
			}
		})
	}
}