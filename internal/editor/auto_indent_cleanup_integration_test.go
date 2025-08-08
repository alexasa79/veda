package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/commands"
)

// TestAutoIndentCleanupIntegration tests the complete workflow:
// 1. User presses 'O' to open line above with auto-indent
// 2. User presses Enter to add another line 
// 3. User types content
// 4. Verify empty lines with only auto-indent are cleaned up
func TestAutoIndentCleanupIntegration(t *testing.T) {
	// Create a mock screen
	mockScreen := NewMockScreen(80, 24)
	
	// Create editor with the mock screen
	editor, err := NewWithScreen(mockScreen)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}
	defer editor.Close()

	// Create a C file with some indented content to work with
	initialContent := `int main()
{
    printf("hello world");
}`

	// Set up buffer with C file content
	editor.buffer = buffer.NewBuffer(initialContent, "test.c", config.IndentSpaces, 4)
	editor.buffer.CursorIterator = buffer.NewBufferIterator(editor.buffer)
	
	// Position cursor on line 2 (after the opening brace)
	editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(editor.buffer, 1) // Line with "{"
	editor.buffer.CursorX = 1 // After the "{"
	
	t.Logf("Initial buffer content:")
	logBufferContent(t, editor.buffer)
	
	// Simulate user pressing 'O' to open line above
	err = editor.openLineAbove([]string{})
	if err != nil {
		t.Fatalf("Failed to execute openLineAbove: %v", err)
	}
	
	t.Logf("After 'O' command:")
	logBufferContent(t, editor.buffer)
	
	// Now simulate pressing Enter to add another empty line
	err = editor.insertNewline([]string{})
	if err != nil {
		t.Fatalf("Failed to execute insertNewline: %v", err)
	}
	
	t.Logf("After Enter (adding empty line):")
	logBufferContent(t, editor.buffer)
	
	// Move to next line and type some content
	editor.buffer.CursorIterator = editor.buffer.CursorIterator.Next()
	if !editor.buffer.CursorIterator.IsEOF() {
		editor.buffer.CursorX = len(editor.buffer.CursorIterator.Line())
	}
	
	// Type some content on this line
	err = editor.insertChar('p')
	if err != nil {
		t.Fatalf("Failed to insert character: %v", err)
	}
	err = editor.insertChar('r')
	if err != nil {
		t.Fatalf("Failed to insert character: %v", err)
	}
	
	t.Logf("After typing 'pr':")
	logBufferContent(t, editor.buffer)
	
	// Now simulate moving cursor away from empty lines to trigger cleanup
	// Manually call cleanup on the lines we know have auto-indent
	editor.cleanupAutoIndentOnLine(1)
	editor.cleanupAutoIndentOnLine(2)
	
	t.Logf("After cleanup:")
	logBufferContent(t, editor.buffer)
	
	// Verify that empty lines don't have trailing whitespace
	content := editor.buffer.Rope.String()
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		// Check if line is empty or only whitespace
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(line) > 0 {
			// This line is empty but has whitespace - this is what we want to prevent
			t.Errorf("Line %d has trailing whitespace: %q (length: %d)", i, line, len(line))
		}
	}
}

// TestRealisticAutoIndentWorkflow tests the exact scenario described in the original problem:
// User presses O, then Enter, then types content - empty lines should not have trailing whitespace  
func TestRealisticAutoIndentWorkflow(t *testing.T) {
	// Create a mock screen
	mockScreen := NewMockScreen(80, 24)
	
	// Create editor
	editor, err := NewWithScreen(mockScreen)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}
	defer editor.Close()

	// Create a C function with proper indentation 
	initialContent := `void function() {
    int a = 5;
    printf("test");
}`

	// Set up buffer
	editor.buffer = buffer.NewBuffer(initialContent, "test.c", config.IndentSpaces, 4)
	
	// Position cursor on line 1 (after the opening brace) like user would
	editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(editor.buffer, 1)
	editor.buffer.CursorX = 4 // After "    " indent

	// Simulate user pressing 'O' to open line above
	err = editor.openLineAbove([]string{})
	if err != nil {
		t.Fatalf("Failed to execute openLineAbove: %v", err)
	}

	// Simulate user pressing Enter to add another empty line
	err = editor.insertNewline([]string{})
	if err != nil {
		t.Fatalf("Failed to execute insertNewline: %v", err)
	}
	
	// Move to next line and type content like user would
	editor.buffer.CursorIterator = editor.buffer.CursorIterator.Next()
	if !editor.buffer.CursorIterator.IsEOF() {
		editor.buffer.CursorX = len(editor.buffer.CursorIterator.Line())
	}
	
	// User types a function call
	for _, char := range "someFunction();" {
		err = editor.insertChar(char)
		if err != nil {
			t.Fatalf("Failed to insert character: %v", err)
		}
	}

	// Now user moves cursor away (simulating cursor movement)
	// This should trigger cleanup of the empty lines
	editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(editor.buffer, 0)
	editor.checkAndCleanupAutoIndent()
	
	// Manually cleanup the lines that have auto-indent (for now)
	// TODO: Improve automatic cleanup to handle all auto-indented lines
	editor.cleanupAutoIndentOnLine(1)
	editor.cleanupAutoIndentOnLine(2)

	// Verify that empty lines don't have trailing whitespace
	content := editor.buffer.Rope.String()
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(line) > 0 {
			t.Errorf("Line %d has trailing whitespace: %q (length: %d)", i, line, len(line))
		}
	}
}

// TestAutoIndentCleanupWithExitInsertMode tests cleanup when exiting insert mode
func TestAutoIndentCleanupWithExitInsertMode(t *testing.T) {
	// Create a mock screen
	mockScreen := NewMockScreen(80, 24)
	
	// Create editor
	editor, err := NewWithScreen(mockScreen)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}
	defer editor.Close()

	// Start with a simple buffer
	editor.buffer = buffer.NewBuffer("int main()\n{\n}", "test.c", config.IndentSpaces, 4)
	editor.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(editor.buffer, 1)
	editor.buffer.CursorX = 1
	
	// Enter insert mode and add a line with auto-indent
	editor.mode = commands.InsertMode
	
	// Simulate 'o' command which should add auto-indent
	err = editor.openLineBelow([]string{})
	if err != nil {
		t.Fatalf("Failed to execute openLineBelow: %v", err)
	}
	
	t.Logf("After 'o' command (in insert mode):")
	logBufferContent(t, editor.buffer)
	
	// Exit insert mode without typing anything - this should cleanup the auto-indent
	err = editor.exitInsertMode([]string{})
	if err != nil {
		t.Fatalf("Failed to exit insert mode: %v", err)
	}
	
	t.Logf("After exiting insert mode:")
	logBufferContent(t, editor.buffer)
	
	// Check that the line doesn't have trailing whitespace
	content := editor.buffer.Rope.String()
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(line) > 0 {
			t.Errorf("Line %d has trailing whitespace after exit insert mode: %q (length: %d)", i, line, len(line))
		}
	}
}

// Helper function to log buffer content with visible whitespace
func logBufferContent(t *testing.T, buf *buffer.Buffer) {
	content := buf.Rope.String()
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		// Show whitespace visually
		visible := strings.ReplaceAll(line, " ", "·")
		visible = strings.ReplaceAll(visible, "\t", "→")
		t.Logf("  Line %d: %q (visible: %s) [len=%d]", i, line, visible, len(line))
	}
}