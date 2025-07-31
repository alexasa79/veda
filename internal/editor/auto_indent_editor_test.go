package editor

import (
	"testing"
)

func TestInsertNewlineWithBraces(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		cursorLine     int
		cursorX        int
		expectedContent string
		description    string
	}{
		{
			name:           "newline between braces",
			initialContent: "if (true) {}",
			cursorLine:     0,
			cursorX:        11,
			expectedContent: "if (true) {\n    \n}",
			description:    "Should indent and split braces when pressing Enter between braces",
		},
		{
			name:           "newline between braces with indent",
			initialContent: "    if (true) {}",
			cursorLine:     0,
			cursorX:        15,
			expectedContent: "    if (true) {\n        \n    }",
			description:    "Should add indent level and maintain closing brace indentation",
		},
		{
			name:           "newline after open brace",
			initialContent: "void foo() {",
			cursorLine:     0,
			cursorX:        12,
			expectedContent: "void foo() {\n    ",
			description:    "Should indent after open brace at end of line",
		},
		{
			name:           "newline in nested function",
			initialContent: "    void bar() {",
			cursorLine:     0,
			cursorX:        16,
			expectedContent: "    void bar() {\n        ",
			description:    "Should maintain base indent and add one level",
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
			// Move cursor to specific position
			te.buffer.CursorX = tt.cursorX
			
			// Enter insert mode and press Enter
			te.ExecuteCommand("enter-insert-mode")
			te.ExecuteCommand("insert-newline")
			
			// Check content
			actual := te.buffer.Rope.String()
			if actual != tt.expectedContent {
				t.Errorf("%s\nExpected:\n%q\nGot:\n%q\nDescription: %s",
					tt.name, tt.expectedContent, actual, tt.description)
			}
		})
	}
}

func TestInsertNewlineBetweenBracesWithTyping(t *testing.T) {
	// Create test editor
	te := NewTestEditorWithContent(80, 24, "")
	
	// Enter insert mode and type a function with braces
	te.ExecuteCommand("enter-insert-mode")
	
	// Type: int main() {}
	text := "int main() {}"
	for _, ch := range text {
		te.InsertChar(ch)
	}
	
	// Move cursor back between braces
	te.ExecuteCommand("left-insert")
	
	// Press Enter - should create proper brace structure
	te.ExecuteCommand("insert-newline")
	
	// Type some code
	code := "return 0;"
	for _, ch := range code {
		te.InsertChar(ch)
	}
	
	// Exit insert mode
	te.ExecuteCommand("exit-insert-mode")
	
	// Check the result
	expected := "int main() {\n    return 0;\n}"
	actual := te.buffer.Rope.String()
	
	if actual != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, actual)
	}
}