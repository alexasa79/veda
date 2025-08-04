package editor

import (
	"strings"
	"testing"
)

func TestRewrapPreservesEmptyLines(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		cursorLine     int
		expectedResult string
	}{
		{
			name: "empty line between paragraphs",
			content: `This is a very long line that should be wrapped when we press alt+q to make it fit within the specified width.

This is another very long line that should also be wrapped but it's in a different paragraph separated by an empty line.`,
			cursorLine: 0,
			expectedResult: `This is a very long line that should be wrapped when we press alt+q to make it
fit within the specified width.

This is another very long line that should also be wrapped but it's in a different paragraph separated by an empty line.`,
		},
		{
			name: "multiple empty lines between paragraphs",
			content: `This is a very long line that should be wrapped when we press alt+q to make it fit within the specified width.


This is another very long line that should also be wrapped but it's in a different paragraph separated by multiple empty lines.`,
			cursorLine: 0,
			expectedResult: `This is a very long line that should be wrapped when we press alt+q to make it
fit within the specified width.


This is another very long line that should also be wrapped but it's in a different paragraph separated by multiple empty lines.`,
		},
		{
			name: "rewrap second paragraph",
			content: `First paragraph stays as is.

This is a very long line in the second paragraph that should be wrapped when we press alt+q to make it fit.`,
			cursorLine: 2,
			expectedResult: `First paragraph stays as is.

This is a very long line in the second paragraph that should be wrapped when we
press alt+q to make it fit.`,
		},
		{
			name: "paragraph with trailing empty line",
			content: `This is a very long line that should be wrapped when we press alt+q to make it fit within the specified width.

`,
			cursorLine: 0,
			expectedResult: `This is a very long line that should be wrapped when we press alt+q to make it
fit within the specified width.

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor with content
			te := NewTestEditorWithContent(120, 24, tt.content)
			
			// Move cursor to the specified line
			for i := 0; i < tt.cursorLine; i++ {
				te.ExecuteCommand("down")
			}
			
			// Execute rewrap
			te.ExecuteCommand("rewrap-paragraph")
			
			// Get the result
			result := te.GetBufferContent()
			
			// Check if empty lines are preserved
			if result != tt.expectedResult {
				t.Errorf("Empty lines not preserved.\nExpected:\n%s\n\nGot:\n%s", 
					tt.expectedResult, result)
				
				// Show line-by-line comparison
				expectedLines := strings.Split(tt.expectedResult, "\n")
				gotLines := strings.Split(result, "\n")
				
				t.Errorf("\nLine by line comparison:")
				maxLines := len(expectedLines)
				if len(gotLines) > maxLines {
					maxLines = len(gotLines)
				}
				
				for i := 0; i < maxLines; i++ {
					var expectedLine, gotLine string
					if i < len(expectedLines) {
						expectedLine = expectedLines[i]
					}
					if i < len(gotLines) {
						gotLine = gotLines[i]
					}
					
					if expectedLine != gotLine {
						t.Errorf("Line %d differs:\nExpected: %q\nGot:      %q", 
							i, expectedLine, gotLine)
					}
				}
			}
		})
	}
}

func TestRewrapParagraphBounds(t *testing.T) {
	// Test that rewrap only affects the current paragraph
	content := `First paragraph that is short.

This is a very long second paragraph that should be wrapped when we execute the rewrap command on it.

Third paragraph is also short.`

	te := NewTestEditorWithContent(120, 24, content)
	
	// Move to second paragraph
	te.ExecuteCommand("down")
	te.ExecuteCommand("down")
	
	// Rewrap second paragraph
	te.ExecuteCommand("rewrap-paragraph")
	
	result := te.GetBufferContent()
	
	// Check that first and third paragraphs are unchanged
	lines := strings.Split(result, "\n")
	if lines[0] != "First paragraph that is short." {
		t.Error("First paragraph was modified when it shouldn't have been")
	}
	
	if lines[len(lines)-1] != "Third paragraph is also short." {
		t.Error("Third paragraph was modified when it shouldn't have been")
	}
	
	// Check that empty lines are still there
	if lines[1] != "" {
		t.Error("Empty line after first paragraph was removed")
	}
	
	// Find the empty line before the third paragraph
	foundEmptyLine := false
	for i := 2; i < len(lines)-1; i++ {
		if lines[i] == "" && lines[i+1] == "Third paragraph is also short." {
			foundEmptyLine = true
			break
		}
	}
	
	if !foundEmptyLine {
		t.Error("Empty line before third paragraph was removed")
	}
}