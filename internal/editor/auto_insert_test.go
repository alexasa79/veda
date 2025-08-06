package editor

import (
	"testing"
	"veda/internal/commands"
)

// TestAutoInsertDisabled tests that auto-pairing has been disabled
func TestAutoInsertDisabled(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		char           rune
		expectedBuffer string
		expectedX      int
		expectedY      int
	}{
		{
			name:           "parentheses_no_pairing",
			initialContent: "fmt.Printf",
			char:           '(',
			expectedBuffer: "fmt.Printf(",
			expectedX:      11, // Cursor after (
			expectedY:      0,
		},
		{
			name:           "square_brackets_no_pairing",
			initialContent: "array",
			char:           '[',
			expectedBuffer: "array[",
			expectedX:      6, // Cursor after [
			expectedY:      0,
		},
		{
			name:           "curly_braces_no_pairing",
			initialContent: "func main() ",
			char:           '{',
			expectedBuffer: "func main() {",
			expectedX:      13, // Cursor after {
			expectedY:      0,
		},
		{
			name:           "double_quotes_no_pairing",
			initialContent: "fmt.Printf(",
			char:           '"',
			expectedBuffer: "fmt.Printf(\"",
			expectedX:      12, // Cursor after quote
			expectedY:      0,
		},
		{
			name:           "single_quotes_no_pairing",
			initialContent: "var c = ",
			char:           '\'',
			expectedBuffer: "var c = '",
			expectedX:      9, // Cursor after quote
			expectedY:      0,
		},
		{
			name:           "backticks_no_pairing",
			initialContent: "raw := ",
			char:           '`',
			expectedBuffer: "raw := `",
			expectedX:      8, // Cursor after backtick
			expectedY:      0,
		},
		{
			name:           "closing_paren_inserts",
			initialContent: "test(",
			char:           ')',
			expectedBuffer: "test()",
			expectedX:      6, // Cursor after )
			expectedY:      0,
		},
		{
			name:           "closing_bracket_inserts",
			initialContent: "array[",
			char:           ']',
			expectedBuffer: "array[]",
			expectedX:      7, // Cursor after ]
			expectedY:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.initialContent)

			// Enter insert mode first
			te.ExecuteCommand("enter-insert-mode")

			// Position at end of line
			te.ExecuteCommand("end-of-line-insert")

			// Insert the character
			te.insertChar(tt.char)

			// Check buffer content
			AssertBufferEquals(t, te, tt.expectedBuffer)

			// Check cursor position
			AssertCursorAt(t, te, tt.expectedX, tt.expectedY)

			// Verify we're still in insert mode
			if te.mode != commands.InsertMode {
				t.Errorf("Expected to remain in insert mode")
			}
		})
	}
}

func TestNormalCharactersStillWork(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "Hello")

	// Enter insert mode
	te.ExecuteCommand("enter-insert-mode")
	te.ExecuteCommand("end-of-line-insert")

	// Type normal characters
	normalChars := []rune{' ', 'w', 'o', 'r', 'l', 'd', '!'}
	for _, ch := range normalChars {
		te.insertChar(ch)
	}

	// Check result
	AssertBufferEquals(t, te, "Hello world!")
	AssertCursorAt(t, te, 12, 0) // At end of line
}

func TestInsertInEmptyBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")

	// Enter insert mode
	te.ExecuteCommand("enter-insert-mode")

	// Insert single character (no pairing)
	te.insertChar('(')

	// Check result
	AssertBufferEquals(t, te, "(")
	AssertCursorAt(t, te, 1, 0) // After (
}

func TestInsertAtBeginningOfLine(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "existing text")

	// Enter insert mode at beginning
	te.ExecuteCommand("enter-insert-mode")
	te.ExecuteCommand("beginning-of-line")

	// Insert single character (no pairing)
	te.insertChar('[')

	// Check result
	AssertBufferEquals(t, te, "[existing text")
	AssertCursorAt(t, te, 1, 0) // After [
}

// TestManuallyTypingPairs tests that users can still manually type pairs
func TestManuallyTypingPairs(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")

	// Enter insert mode
	te.ExecuteCommand("enter-insert-mode")

	// Manually type a pair
	te.insertChar('(')
	te.insertChar(')')

	// Check result
	AssertBufferEquals(t, te, "()")
	AssertCursorAt(t, te, 2, 0) // After )

	// Type more characters
	te.insertChar('[')
	te.insertChar(']')

	AssertBufferEquals(t, te, "()[]")
	AssertCursorAt(t, te, 4, 0) // After ]
}