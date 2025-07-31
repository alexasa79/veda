package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestContinuationIndentForFunctionArguments(t *testing.T) {
	tests := []struct {
		name                 string
		initialContent       string
		cursorLine           int
		cursorX              int
		expectedIndentLevel  int
		expectedIndentColumn int
		description          string
	}{
		{
			name: "first argument after opening paren with content",
			initialContent: `int main() {
    func(1,`,
			cursorLine:           1,
			cursorX:              11, // After the comma
			expectedIndentLevel:  0,
			expectedIndentColumn: 9, // Align with character after opening paren
			description:          "First argument should align after opening paren when content exists",
		},
		{
			name: "second argument continues alignment",
			initialContent: `int main() {
    func(1,`,
			cursorLine:           1,
			cursorX:              11, // After the comma
			expectedIndentLevel:  0,
			expectedIndentColumn: 9, // Should maintain same alignment
			description:          "Second argument should maintain alignment",
		},
		{
			name: "third argument continues alignment",
			initialContent: `int main() {
    func(1,
         2,`,
			cursorLine:           2,
			cursorX:              11, // After the comma on line with "2,"
			expectedIndentLevel:  0,
			expectedIndentColumn: 9, // Should maintain same alignment
			description:          "Third argument should maintain alignment",
		},
		{
			name: "function with existing multi-line args",
			initialContent: `void func(int arg1,
         int arg2,`,
			cursorLine:           1,
			cursorX:              19, // After "arg2,"
			expectedIndentLevel:  0,
			expectedIndentColumn: 9, // Should maintain alignment
			description:          "Additional arguments should maintain alignment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor with the initial content
			te := NewTestEditorWithContent(80, 24, tt.initialContent)
			
			// Set cursor position
			te.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(te.buffer, tt.cursorLine)
			te.buffer.CursorX = tt.cursorX
			
			// Get the current line
			currentLine := te.buffer.CursorIterator.Line()
			
			// Calculate auto-indent
			autoIndent := te.buffer.CalculateAutoIndent(currentLine, te.buffer.CursorX)
			
			
			// Check the indent column
			if autoIndent.Column != tt.expectedIndentColumn {
				t.Errorf("%s: Expected indent column %d, got %d", 
					tt.description, tt.expectedIndentColumn, autoIndent.Column)
			}
			
			// Also verify the indent string length matches expected column
			// (for space indentation)
			if te.buffer.Indentation.Type == config.IndentSpaces {
				if len(autoIndent.IndentStr) != tt.expectedIndentColumn {
					t.Errorf("%s: Expected indent string length %d, got %d (string: %q)", 
						tt.description, tt.expectedIndentColumn, len(autoIndent.IndentStr), autoIndent.IndentStr)
				}
			}
		})
	}
}

// Test the actual behavior when pressing Enter in these scenarios
func TestContinuationIndentWithEnter(t *testing.T) {
	// Test case: Adding third argument to function call
	te := NewTestEditorWithContent(80, 24, `int main() {
    func(1,
         2,`)
	
	// Position cursor after "2,"
	te.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(te.buffer, 2)
	te.buffer.CursorX = 11
	
	// Press Enter (insert newline)
	err := te.ExecuteCommand("insert-newline")
	if err != nil {
		t.Fatalf("Failed to insert newline: %v", err)
	}
	
	// Check cursor position - should be indented to column 9
	if te.buffer.CursorX != 9 {
		t.Errorf("After Enter, expected cursor at column 9, got %d", te.buffer.CursorX)
	}
	
	// Verify the content
	expectedContent := `int main() {
    func(1,
         2,
         `
	
	actualContent := te.buffer.Rope.String()
	if actualContent != expectedContent {
		t.Errorf("Content mismatch.\nExpected:\n%q\nGot:\n%q", expectedContent, actualContent)
	}
}