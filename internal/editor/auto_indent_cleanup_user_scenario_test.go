package editor

import (
    "strings"
    "testing"
)

// TestAutoIndentCleanupAfterMultipleEntersAndTyping reproduces the user scenario:
// - Cursor at end of an indented code line in insert mode
// - Press Enter twice (two auto-indented lines are created)
// - Type a bunch of non-whitespace characters on the second new line
// - Exit insert mode
// Expected: any empty line created by auto-indent should not retain trailing spaces
func TestAutoIndentCleanupAfterMultipleEntersAndTyping(t *testing.T) {
    te := NewTestEditorWithContent(80, 24, `int main()
{
    printf("hello world");
}`)

    // Position cursor at the end of the printf line (line 2)
    te.SetCursorPosition(len("    printf(\"hello world\");"), 2)

    // Enter insert mode
    if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
        t.Fatalf("failed to enter insert mode: %v", err)
    }

    // Press Enter twice (two auto-indented lines should be created)
    if err := te.ExecuteCommand("insert-newline"); err != nil {
        t.Fatalf("failed to insert newline: %v", err)
    }
    if err := te.ExecuteCommand("insert-newline"); err != nil {
        t.Fatalf("failed to insert second newline: %v", err)
    }

    // Type many non-whitespace characters to grow buffer.Generation significantly
    longText := "someFunctionCallWithQuiteALongNameAndManyChars();"
    for _, ch := range longText {
        if err := te.InsertChar(ch); err != nil {
            t.Fatalf("failed to insert char %q: %v", ch, err)
        }
    }

    // Exit insert mode which should trigger auto-indent cleanup
    if err := te.ExecuteCommand("exit-insert-mode"); err != nil {
        t.Fatalf("failed to exit insert mode: %v", err)
    }

    // Verify no line is whitespace-only with trailing spaces
    content := te.GetBufferContent()
    lines := strings.Split(content, "\n")
    for i, line := range lines {
        if strings.TrimSpace(line) == "" && len(line) > 0 {
            t.Fatalf("line %d has trailing whitespace left over: %q", i, line)
        }
    }
}
