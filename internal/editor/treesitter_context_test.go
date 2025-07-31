package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/syntax"
)

// TestMultilineFunctionContext tests that multiline function definitions
// show the correct context line (the line with the function name)
func TestMultilineFunctionContext(t *testing.T) {
	// Test C code with multiline function definition
	testCode := `#include <stdio.h>

static const void *
function_name(int arg1, int arg2)
{
    // Line 5: Inside function
    if (arg1 > 0) {
        // Line 7: Inside if statement
        return NULL;
    }
    // Line 10: Still inside function
    return NULL;
}

int main() {
    // Line 15: Inside main
    return 0;
}`

	// Create buffer with the test code
	buf := buffer.NewBuffer(testCode, "test.c", config.IndentSpaces, 4)
	
	// Create syntax highlighter for C
	highlighter := syntax.NewHighlighter()
	if highlighter == nil {
		t.Skip("Syntax highlighter not available")
	}
	
	// Set language to C
	err := highlighter.SetLanguageFromFile("test.c")
	if err != nil {
		t.Skipf("C language not available: %v", err)
	}
	
	// Parse the code
	highlighter.ParseFull([]byte(testCode))
	
	tests := []struct {
		name           string
		cursorLine     int
		viewStartLine  int
		viewEndLine    int
		expectedLines  []int // Expected context lines
		expectedNotIn  []int // Lines that should NOT be in context
	}{
		{
			name:          "cursor inside multiline function",
			cursorLine:    7,  // Inside the if statement
			viewStartLine: 6,  // View starts at line 6
			viewEndLine:   10, // View ends at line 10
			expectedLines: []int{3}, // Should show line 3 which has "function_name"
			expectedNotIn: []int{2}, // Should NOT show line 2 with "static const void *"
		},
		{
			name:          "cursor in main function",
			cursorLine:    15,
			viewStartLine: 15,
			viewEndLine:   17,
			expectedLines: []int{14}, // Line with "int main()"
		},
		{
			name:          "cursor at function body start",
			cursorLine:    5,
			viewStartLine: 5,
			viewEndLine:   8,
			expectedLines: []int{3}, // Should still show function_name line
			expectedNotIn: []int{2},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextLines := getTreeSitterContext(buf, highlighter, tt.cursorLine, tt.viewStartLine, tt.viewEndLine, 5)
			
			// Check that expected lines are present
			for _, expectedLine := range tt.expectedLines {
				found := false
				for _, line := range contextLines {
					if line == expectedLine {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected context to include line %d, but it didn't. Got: %v", expectedLine, contextLines)
				}
			}
			
			// Check that lines that should NOT be present are indeed not present
			for _, notExpectedLine := range tt.expectedNotIn {
				for _, line := range contextLines {
					if line == notExpectedLine {
						t.Errorf("Expected context to NOT include line %d, but it did. Got: %v", notExpectedLine, contextLines)
					}
				}
			}
		})
	}
}

// TestComplexMultilineFunctionContext tests more complex multiline scenarios
func TestComplexMultilineFunctionContext(t *testing.T) {
	testCode := `typedef struct {
    int x;
    int y;
} Point;

static inline const Point *
get_point_reference(const Point *p,
                   int offset,
                   bool validate)
{
    // Line 10: Function body
    if (validate && offset < 0) {
        return NULL;
    }
    return &p[offset];
}

static const char *const
get_string_constant(int index)
{
    // Line 20: Another function
    switch (index) {
    case 0:
        return "zero";
    case 1:
        return "one";
    default:
        return "unknown";
    }
}`

	// Create buffer with the test code
	buf := buffer.NewBuffer(testCode, "test.c", config.IndentSpaces, 4)
	
	// Create syntax highlighter for C
	highlighter := syntax.NewHighlighter()
	if highlighter == nil {
		t.Skip("Syntax highlighter not available")
	}
	
	// Set language to C
	err := highlighter.SetLanguageFromFile("test.c")
	if err != nil {
		t.Skipf("C language not available: %v", err)
	}
	
	// Parse the code
	highlighter.ParseFull([]byte(testCode))
	
	tests := []struct {
		name          string
		cursorLine    int
		viewStartLine int
		viewEndLine   int
		expectedLine  int  // The line we expect to see in context
		description   string
	}{
		{
			name:          "multiline function with complex return type",
			cursorLine:    12, // Inside get_point_reference
			viewStartLine: 11,
			viewEndLine:   14,
			expectedLine:  6, // Line with "get_point_reference"
			description:   "Should show function name line, not return type line",
		},
		{
			name:          "const char pointer function",
			cursorLine:    22, // Inside switch statement
			viewStartLine: 21,
			viewEndLine:   25,
			expectedLine:  18, // Line with "get_string_constant"
			description:   "Should show function name for const char * const function",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextLines := getTreeSitterContext(buf, highlighter, tt.cursorLine, tt.viewStartLine, tt.viewEndLine, 5)
			
			found := false
			for _, line := range contextLines {
				if line == tt.expectedLine {
					found = true
					break
				}
			}
			
			if !found {
				// Get the actual line content for debugging
				iter := buffer.NewBufferIteratorAtLine(buf, tt.expectedLine)
				expectedContent := ""
				if !iter.IsEOF() {
					expectedContent = iter.Line()
				}
				
				t.Errorf("%s: Expected to see line %d (%q) in context, got: %v", 
					tt.description, tt.expectedLine, expectedContent, contextLines)
			}
		})
	}
}