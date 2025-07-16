package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/commands"
)

func TestAutoInsertPairedCharacters(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		char           rune
		expectedBuffer string
		expectedX      int
		expectedY      int
	}{
		{
			name:           "parentheses",
			initialContent: "fmt.Printf",
			char:           '(',
			expectedBuffer: "fmt.Printf()",
			expectedX:      11, // Cursor between ( and )
			expectedY:      0,
		},
		{
			name:           "square brackets",
			initialContent: "array",
			char:           '[',
			expectedBuffer: "array[]",
			expectedX:      6, // Cursor between [ and ]
			expectedY:      0,
		},
		{
			name:           "curly braces",
			initialContent: "func main() ",
			char:           '{',
			expectedBuffer: "func main() {}",
			expectedX:      13, // Cursor between { and }
			expectedY:      0,
		},
		{
			name:           "double quotes",
			initialContent: "fmt.Printf(",
			char:           '"',
			expectedBuffer: "fmt.Printf(\"\"",
			expectedX:      12, // Cursor between quotes
			expectedY:      0,
		},
		{
			name:           "single quotes",
			initialContent: "var c = ",
			char:           '\'',
			expectedBuffer: "var c = ''",
			expectedX:      9, // Cursor between quotes
			expectedY:      0,
		},
		{
			name:           "backticks",
			initialContent: "raw := ",
			char:           '`',
			expectedBuffer: "raw := ``",
			expectedX:      8, // Cursor between backticks
			expectedY:      0,
		},
		{
			name:           "middle of line parentheses",
			initialContent: "if condition { fmt.Printf }",
			char:           '(',
			expectedBuffer: "if condition { fmt.Printf() }",
			expectedX:      26, // Cursor between ( and )
			expectedY:      0,
		},
		{
			name:           "multiline parentheses",
			initialContent: "func test() {\n\tfmt.Printf\n}",
			char:           '(',
			expectedBuffer: "func test() {\n\tfmt.Printf()\n}",
			expectedX:      12, // Cursor between ( and ) on line 2
			expectedY:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.initialContent)

			// Enter insert mode first
			te.ExecuteCommand("enter-insert-mode")

			// Move cursor to correct position
			switch tt.name {
			case "middle of line parentheses":
				// Position cursor after "Printf" but before " }"
				te.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(te.buffer, 0)
				te.buffer.CursorX = 25 // After 'f' in Printf
			case "multiline parentheses":
				// Position cursor on second line after Printf
				te.ExecuteCommand("down-insert")
				te.ExecuteCommand("end-of-line-insert")
			default:
				// Default: position at end of line
				te.ExecuteCommand("end-of-line-insert")
			}

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

func TestAutoInsertDoesNotAffectNormalCharacters(t *testing.T) {
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

func TestAutoInsertInEmptyBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")

	// Enter insert mode
	te.ExecuteCommand("enter-insert-mode")

	// Insert paired character
	te.insertChar('(')

	// Check result
	AssertBufferEquals(t, te, "()")
	AssertCursorAt(t, te, 1, 0) // Between ( and )
}

func TestAutoInsertAtBeginningOfLine(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "existing text")

	// Enter insert mode at beginning
	te.ExecuteCommand("enter-insert-mode")
	te.ExecuteCommand("beginning-of-line")

	// Insert paired character
	te.insertChar('[')

	// Check result
	AssertBufferEquals(t, te, "[]existing text")
	AssertCursorAt(t, te, 1, 0) // Between [ and ]
}