package editor

import (
    "testing"
)

// Test that Python lines ending with ':' auto-indent on newline
func TestPythonAutoIndentAfterColon(t *testing.T) {
    te := NewTestEditorWithContent(80, 24, "def foo():")
    te.CurrentBuffer().FilePath = "test.py"

    // Place cursor at end of line and insert newline
    te.SetCursorPosition(len("def foo():"), 0)
    if err := te.ExecuteCommand("insert-newline"); err != nil {
        t.Fatalf("insert-newline failed: %v", err)
    }

    // Expect one indent level on the new line (default width)
    line := te.CurrentBuffer().CursorIterator.Line()
    // Cursor should be positioned after indentation
    if te.CurrentBuffer().CursorX == 0 {
        t.Fatalf("expected indentation on new Python block line; got none")
    }
    // All characters before cursor should be spaces
    for i := 0; i < te.CurrentBuffer().CursorX; i++ {
        if line[i] != ' ' && line[i] != '\t' {
            t.Fatalf("expected indentation before cursor; got %q at pos %d", line[i], i)
        }
    }
}

// Test that a newline after an opening bracket in Python increases indentation
func TestPythonListContinuationIndent(t *testing.T) {
    te := NewTestEditorWithContent(80, 24, "items = [")
    te.CurrentBuffer().FilePath = "config.py"

    te.SetCursorPosition(len("items = ["), 0)
    if err := te.ExecuteCommand("insert-newline"); err != nil {
        t.Fatalf("insert-newline failed: %v", err)
    }

    // Expect indentation on continuation line inside brackets
    line := te.CurrentBuffer().CursorIterator.Line()
    if te.CurrentBuffer().CursorX == 0 {
        t.Fatalf("expected indentation inside Python bracket continuation; got none (line=%q)", line)
    }
}

