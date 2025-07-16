package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestOpenLineBelowWithAutoIndent(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		cursorLine     int
		cursorCol      int
		expectedBuffer string
		expectedCursor struct{ line, col int }
	}{
		{
			name:           "function with parameter - o command",
			initialContent: "    function_call(parameter,",
			cursorLine:     0,
			cursorCol:      0,
			expectedBuffer: "    function_call(parameter,\n                  ",
			expectedCursor: struct{ line, col int }{1, 18},
		},
		{
			name:           "function with just open paren - o command",
			initialContent: "    function_call(",
			cursorLine:     0,
			cursorCol:      0,
			expectedBuffer: "    function_call(\n        ",
			expectedCursor: struct{ line, col int }{1, 8},
		},
		{
			name:           "no parens - o command",
			initialContent: "    let x = 5",
			cursorLine:     0,
			cursorCol:      0,
			expectedBuffer: "    let x = 5\n    ",
			expectedCursor: struct{ line, col int }{1, 4},
		},
		{
			name:           "nested parens - o command",
			initialContent: "    outer(inner(value),",
			cursorLine:     0,
			cursorCol:      0,
			expectedBuffer: "    outer(inner(value),\n          ",
			expectedCursor: struct{ line, col int }{1, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.initialContent)
			
			// Position cursor
			for i := 0; i < tt.cursorLine; i++ {
				te.ExecuteCommand("down")
			}
			for i := 0; i < tt.cursorCol; i++ {
				te.ExecuteCommand("right")
			}
			
			// Execute open-line-below command
			te.ExecuteCommand("open-line-below")
			
			// Check buffer content
			AssertBufferEquals(t, te, tt.expectedBuffer)
			
			// Check cursor position
			AssertCursorAt(t, te, tt.expectedCursor.col, tt.expectedCursor.line)
			
			// Verify we're in insert mode
			if te.Editor.mode != commands.InsertMode {
				t.Errorf("Expected to be in insert mode")
			}
		})
	}
}

func TestOpenLineAboveWithAutoIndent(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		cursorLine     int
		cursorCol      int
		expectedBuffer string
		expectedCursor struct{ line, col int }
	}{
		{
			name:           "function on previous line - O command",
			initialContent: "    function_call(parameter,\n    next line",
			cursorLine:     1,
			cursorCol:      0,
			expectedBuffer: "    function_call(parameter,\n                  \n    next line",
			expectedCursor: struct{ line, col int }{1, 18},
		},
		{
			name:           "simple indent - O command",
			initialContent: "    first line\n    second line",
			cursorLine:     1,
			cursorCol:      0,
			expectedBuffer: "    first line\n    \n    second line",
			expectedCursor: struct{ line, col int }{1, 4},
		},
		{
			name:           "O on first line",
			initialContent: "first line",
			cursorLine:     0,
			cursorCol:      0,
			expectedBuffer: "\nfirst line",
			expectedCursor: struct{ line, col int }{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.initialContent)
			
			// Position cursor
			for i := 0; i < tt.cursorLine; i++ {
				te.ExecuteCommand("down")
			}
			for i := 0; i < tt.cursorCol; i++ {
				te.ExecuteCommand("right")
			}
			
			// Execute open-line-above command
			te.ExecuteCommand("open-line-above")
			
			// Check buffer content
			AssertBufferEquals(t, te, tt.expectedBuffer)
			
			// Check cursor position
			AssertCursorAt(t, te, tt.expectedCursor.col, tt.expectedCursor.line)
			
			// Verify we're in insert mode
			if te.Editor.mode != commands.InsertMode {
				t.Errorf("Expected to be in insert mode")
			}
		})
	}
}

func TestInsertNewlineWithAutoIndent(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		cursorLine     int
		cursorCol      int
		expectedBuffer string
		expectedCursor struct{ line, col int }
	}{
		{
			name:           "newline in middle of function call",
			initialContent: "    function_call(param,",
			cursorLine:     0,
			cursorCol:      23, // After 'param'
			expectedBuffer: "    function_call(param\n                  ,",
			expectedCursor: struct{ line, col int }{1, 18},
		},
		{
			name:           "newline after open paren",
			initialContent: "    function_call(",
			cursorLine:     0,
			cursorCol:      18, // After '('
			expectedBuffer: "    function_call(\n        ",
			expectedCursor: struct{ line, col int }{1, 8},
		},
		{
			name:           "newline in normal line",
			initialContent: "    let x = 5",
			cursorLine:     0,
			cursorCol:      10, // After ' ='
			expectedBuffer: "    let x \n    = 5",
			expectedCursor: struct{ line, col int }{1, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.initialContent)
			
			// Enter insert mode
			te.ExecuteCommand("enter-insert-mode")
			
			// Position cursor
			for i := 0; i < tt.cursorLine; i++ {
				te.ExecuteCommand("down")
			}
			te.Editor.buffer.CursorX = tt.cursorCol
			
			// Insert newline
			te.ExecuteCommand("insert-newline")
			
			// Check buffer content
			AssertBufferEquals(t, te, tt.expectedBuffer)
			
			// Check cursor position
			AssertCursorAt(t, te, tt.expectedCursor.col, tt.expectedCursor.line)
		})
	}
}