package editor

import (
	"strings"
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
			te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), tt.cursorLine)
			te.CurrentBuffer().CursorX = tt.cursorX

			// Get the current line
			currentLine := te.CurrentBuffer().CursorIterator.Line()

			// Calculate auto-indent
			autoIndent := te.CurrentBuffer().CalculateAutoIndent(currentLine, te.CurrentBuffer().CursorX)

			// Check the indent column
			if autoIndent.Column != tt.expectedIndentColumn {
				t.Errorf("%s: Expected indent column %d, got %d",
					tt.description, tt.expectedIndentColumn, autoIndent.Column)
			}

			// Also verify the indent string length matches expected column
			// (for space indentation)
			if te.CurrentBuffer().Indentation.Type == config.IndentSpaces {
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
	te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 2)
	te.CurrentBuffer().CursorX = 11

	// Press Enter (insert newline)
	err := te.ExecuteCommand("insert-newline")
	if err != nil {
		t.Fatalf("Failed to insert newline: %v", err)
	}

	// Check cursor position - should be indented to column 9
	if te.CurrentBuffer().CursorX != 9 {
		t.Errorf("After Enter, expected cursor at column 9, got %d", te.CurrentBuffer().CursorX)
	}

	// Verify the content
	expectedContent := `int main() {
    func(1,
         2,
         `

	actualContent := te.CurrentBuffer().Rope.String()
	if actualContent != expectedContent {
		t.Errorf("Content mismatch.\nExpected:\n%q\nGot:\n%q", expectedContent, actualContent)
	}
}

// TestOpenLineBelowCTernaryRegression reproduces the over-indentation that occurs
// when pressing 'o' on a C-style ternary expression that contains a nested
// parenthesized conditional.
func TestOpenLineBelowCTernaryRegression(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, `void report_status(void)
{
    /* Emit status for diagnostics */
    int outcome = response_flag ? STATUS_OK : (needs_reset ? STATUS_INVALID : STATUS_WAITING);
    return outcome;
}`)

	// Position cursor on the ternary line like the user would before pressing 'o'.
	te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 3)

	if err := te.ExecuteCommand("open-line-below"); err != nil {
		t.Fatalf("open-line-below command failed: %v", err)
	}

	lines := strings.Split(te.CurrentBuffer().Rope.String(), "\n")
	if len(lines) < 6 {
		t.Fatalf("expected at least 6 lines after opening a new line, got %d (content=%q)", len(lines), te.CurrentBuffer().Rope.String())
	}

	newLine := lines[4]
	expectedIndent := te.CurrentBuffer().GetIndentationString()
	if newLine != expectedIndent {
		t.Fatalf("expected new line to indent to %q (L1), got %q (len=%d). lines=%q", expectedIndent, newLine, len(newLine), lines)
	}
}
