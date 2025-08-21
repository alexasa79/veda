package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestDebugAutoIndentBehavior tests what's actually happening with auto-indent
func TestDebugAutoIndentBehavior(t *testing.T) {
	// Create a mock screen
	mockScreen := NewMockScreen(80, 24)

	// Create editor
	editor, err := NewWithScreen(mockScreen)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}
	defer editor.Close()

	// Create a C file with properly indented content
	initialContent := `int main()
{
    if (1) {
        printf("hello");
    }
}`

	editor.buffer = buffer.NewBuffer(initialContent, "test.c", config.IndentSpaces, 4)

	// Test auto-indent calculation directly
	t.Log("Testing auto-indent calculation directly:")

	// Test line 2: "    if (1) {"
	editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(editor.buffer, 2)
	currentLine := editor.buffer.CursorIterator.Line()
	t.Logf("Line 2 content: %q", currentLine)

	autoIndent := editor.buffer.CalculateAutoIndent(currentLine, len(currentLine))
	t.Logf("Auto-indent result: IndentStr=%q (len=%d), Column=%d",
		autoIndent.IndentStr, len(autoIndent.IndentStr), autoIndent.Column)

	// Now test the openLineAbove function
	t.Log("\nTesting openLineAbove on line with indentation:")
	editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(editor.buffer, 3) // Line with "        printf..."
	editor.buffer.CursorX = 8                                                       // After the indentation

	t.Logf("Current line before O: %q", editor.buffer.CursorIterator.Line())
	t.Logf("Cursor position: line=%d, x=%d", editor.buffer.CursorIterator.LineNumber(), editor.buffer.CursorX)

	// Execute openLineAbove
	err = editor.openLineAbove([]string{})
	if err != nil {
		t.Fatalf("Failed to execute openLineAbove: %v", err)
	}

	t.Log("After openLineAbove:")
	logBufferContent(t, editor.buffer)

	// Check if any lines have trailing whitespace
	content := editor.buffer.Rope.String()
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		if len(line) > 0 && strings.TrimSpace(line) == "" {
			t.Logf("*** Line %d has trailing whitespace: %q (length: %d) ***", i, line, len(line))
		}
	}
}
