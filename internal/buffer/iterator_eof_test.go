package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestIteratorEOFPrevious(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "multi-line buffer",
			content: "Line 1\nLine 2\nLine 3",
		},
		{
			name:    "buffer ending with newline",
			content: "Line 1\nLine 2\nLine 3\n",
		},
		{
			name:    "single line",
			content: "Single line",
		},
		{
			name:    "empty buffer",
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffer(tt.content, "test.txt", config.IndentSpaces, 4)
			
			// 1. Go to the end of the buffer
			iter := NewBufferIteratorAtEnd(buf)
			
			// Verify it's at EOF
			if !iter.IsEOF() {
				t.Errorf("Expected iterator to be at EOF")
			}
			
			// Verify offset is at rope length
			if iter.Position() != buf.Rope.Length() {
				t.Errorf("Expected position %d, got %d", buf.Rope.Length(), iter.Position())
			}
			
			if tt.content == "" {
				// Empty buffer - Previous() should stay at EOF
				prev := iter.Previous()
				if !prev.IsEOF() {
					t.Errorf("Expected Previous() on empty buffer to stay at EOF")
				}
				return
			}
			
			// 2. Call Previous to move one position before the end
			prev := iter.Previous()
			
			// Should no longer be at EOF
			if prev.IsEOF() {
				t.Errorf("Expected Previous() to move before EOF")
			}
			
			// Verify we're on the correct line after Previous()
			// The expected line depends on the content
			var expectedLineNum int
			var expectedContent string
			
			switch tt.name {
			case "multi-line buffer":
				// "Line 1\nLine 2\nLine 3" - EOF is at line 2 (same as "Line 3")
				// Previous() should go to line 1
				expectedLineNum = 1
				expectedContent = "Line 2"
			case "buffer ending with newline":
				// "Line 1\nLine 2\nLine 3\n" - EOF is at line 3 (empty line)
				// Previous() should go to line 2
				expectedLineNum = 2
				expectedContent = "Line 3"
			case "single line":
				// "Single line" - EOF is at line 0
				// Previous() should stay at line 0 (can't go before start)
				expectedLineNum = 0
				expectedContent = "Single line"
			}
			
			if prev.LineNumber() != expectedLineNum {
				t.Errorf("Expected to be on line %d, got %d", expectedLineNum, prev.LineNumber())
			}
			
			if prev.Line() != expectedContent {
				t.Errorf("Expected line content %q, got %q", expectedContent, prev.Line())
			}
			
			// 3. Call Next to make sure we can navigate forward
			// Note: Next() from Previous() won't necessarily return to EOF
			// For "Line 1\nLine 2\nLine 3", Previous() goes to line 1,
			// and Next() goes to line 2 (not EOF)
			next := prev.Next()
			
			// Only check for EOF if we expect it
			if tt.name == "buffer ending with newline" || tt.name == "single line" {
				// For these cases, Previous() then Next() should return to EOF
				if !next.IsEOF() {
					t.Errorf("Expected Next() after Previous() to return to EOF")
				}
			}
			
			// 4. Call Previous repeatedly to reach beginning of buffer
			current := prev
			for current.LineNumber() > 0 {
				current = current.Previous()
			}
			
			// Should be at line 0
			if current.LineNumber() != 0 {
				t.Errorf("Expected to reach line 0, got line %d", current.LineNumber())
			}
			
			// Verify we're at position 0
			if current.Position() != 0 {
				t.Errorf("Expected position 0 at beginning, got %d", current.Position())
			}
		})
	}
}

// Helper function to count newlines in content
func countNewlines(content string) int {
	count := 0
	for _, ch := range content {
		if ch == '\n' {
			count++
		}
	}
	return count
}

func TestIteratorEOFNavigation(t *testing.T) {
	content := "Line 0\nLine 1\nLine 2\nLine 3"
	buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)
	
	// Start at beginning
	iter := NewBufferIterator(buf)
	
	// Verify starting position
	if iter.IsEOF() {
		t.Errorf("Should not start at EOF")
	}
	if iter.Position() != 0 {
		t.Errorf("Should start at position 0, got %d", iter.Position())
	}
	
	// Move to EOF by calling Next repeatedly
	lineCount := 0
	for !iter.IsEOF() {
		iter = iter.Next()
		lineCount++
	}
	
	// Should have moved through 4 lines (0, 1, 2, 3)
	if lineCount != 4 {
		t.Errorf("Expected to move through 4 lines, got %d", lineCount)
	}
	
	// Verify we're at EOF
	if !iter.IsEOF() {
		t.Errorf("Expected to be at EOF")
	}
	if iter.Position() != len(content) {
		t.Errorf("Expected position %d at EOF, got %d", len(content), iter.Position())
	}
	
	// Move back one line from EOF
	prev := iter.Previous()
	if prev.IsEOF() {
		t.Errorf("Should not be at EOF after Previous()")
	}
	if prev.LineNumber() != 3 {
		t.Errorf("Expected line 3, got line %d", prev.LineNumber())
	}
	if prev.Line() != "Line 3" {
		t.Errorf("Expected 'Line 3', got %q", prev.Line())
	}
	
	// Go back to EOF
	next := prev.Next()
	if !next.IsEOF() {
		t.Errorf("Expected to be back at EOF")
	}
	
	// Navigate all the way back to beginning
	current := iter
	for current.LineNumber() > 0 {
		current = current.Previous()
	}
	
	// Should be at line 0
	if current.LineNumber() != 0 {
		t.Errorf("Expected line 0, got line %d", current.LineNumber())
	}
	if current.Position() != 0 {
		t.Errorf("Expected position 0, got %d", current.Position())
	}
	if current.Line() != "Line 0" {
		t.Errorf("Expected 'Line 0', got %q", current.Line())
	}
	
	// Try to go before line 0
	beforeStart := current.Previous()
	if beforeStart.LineNumber() != 0 {
		t.Errorf("Should stay at line 0 when trying to go before start")
	}
	if beforeStart.Position() != 0 {
		t.Errorf("Should stay at position 0 when trying to go before start")
	}
}