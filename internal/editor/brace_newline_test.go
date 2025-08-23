package editor

import (
	"testing"
)

func TestInsertNewlineBetweenBracesProperIndent(t *testing.T) {
	tests := []struct {
		name            string
		initialContent  string
		cursorLine      int
		cursorX         int
		expectedContent string
		expectedCursorX int
		description     string
	}{
		{
			name:            "newline between braces on same line",
			initialContent:  "if (true) {}",
			cursorLine:      0,
			cursorX:         11, // between { and }
			expectedContent: "if (true) {\n    \n}",
			expectedCursorX: 4, // indented position
			description:     "Should split braces and indent cursor, but keep closing brace at original level",
		},
		{
			name:            "newline between braces with base indent",
			initialContent:  "    if (true) {}",
			cursorLine:      0,
			cursorX:         15, // between { and }
			expectedContent: "    if (true) {\n        \n    }",
			expectedCursorX: 8, // indented position
			description:     "Should maintain closing brace at original indent level",
		},
		{
			name:            "newline between braces - function",
			initialContent:  "void foo() {}",
			cursorLine:      0,
			cursorX:         12, // between { and }
			expectedContent: "void foo() {\n    \n}",
			expectedCursorX: 4,
			description:     "Function braces should behave the same way",
		},
		{
			name:            "newline between braces - nested",
			initialContent:  "    void bar() { if (x) {} }",
			cursorLine:      0,
			cursorX:         25, // between inner { and }
			expectedContent: "    void bar() { if (x) {\n        \n    } }",
			expectedCursorX: 8,
			description:     "Nested braces should maintain proper indentation",
		},
		{
			name:            "newline after open brace only",
			initialContent:  "if (true) {",
			cursorLine:      0,
			cursorX:         11, // after {
			expectedContent: "if (true) {\n    ",
			expectedCursorX: 4,
			description:     "When no closing brace, just indent normally",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor with content
			te := NewTestEditorWithContent(80, 24, tt.initialContent)

			// Position cursor
			for i := 0; i < tt.cursorLine; i++ {
				te.ExecuteCommand("down")
			}
			te.CurrentBuffer().CursorX = tt.cursorX

			// Enter insert mode and press Enter
			te.ExecuteCommand("enter-insert-mode")
			te.ExecuteCommand("insert-newline")

			// Check content
			actual := te.CurrentBuffer().Rope.String()
			if actual != tt.expectedContent {
				t.Errorf("%s\nExpected:\n%q\nGot:\n%q\nDescription: %s",
					tt.name, tt.expectedContent, actual, tt.description)
			}

			// Check cursor position
			if te.CurrentBuffer().CursorX != tt.expectedCursorX {
				t.Errorf("%s: Cursor position mismatch\nExpected: %d\nGot: %d",
					tt.name, tt.expectedCursorX, te.CurrentBuffer().CursorX)
			}
		})
	}
}
