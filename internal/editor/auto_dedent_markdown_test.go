package editor

import (
	"strings"
	"testing"

	"veda/internal/buffer"
)

func TestAutoDedentMarkdownListNumber(t *testing.T) {
	tests := []struct {
		name            string
		initialContent  string
		filePath        string
		cursorLine      int
		typedChars      string
		expectedContent string
	}{
		{
			name:            "basic dedent after numbered list",
			initialContent:  "1. first\n2. second\n   ",
			filePath:        "test.md",
			cursorLine:      2,
			typedChars:      "3.",
			expectedContent: "1. first\n2. second\n3.",
		},
		{
			name:            "non-matching number no dedent",
			initialContent:  "2. second\n   ",
			filePath:        "test.md",
			cursorLine:      1,
			typedChars:      "4.",
			expectedContent: "2. second\n   4.",
		},
		{
			name:            "already correct indent",
			initialContent:  "1. first\n2. second\n",
			filePath:        "test.md",
			cursorLine:      2,
			typedChars:      "3.",
			expectedContent: "1. first\n2. second\n3.",
		},
		{
			name:            "indented list snaps to matching indent",
			initialContent:  "  1. first\n  2. second\n      ",
			filePath:        "test.md",
			cursorLine:      2,
			typedChars:      "3.",
			expectedContent: "  1. first\n  2. second\n  3.",
		},
		{
			name:            "multi-digit number",
			initialContent:  "9. ninth\n   ",
			filePath:        "test.md",
			cursorLine:      1,
			typedChars:      "10.",
			expectedContent: "9. ninth\n10.",
		},
		{
			name:            "non-markdown file no dedent",
			initialContent:  "1. first\n2. second\n   ",
			filePath:        "test.go",
			cursorLine:      2,
			typedChars:      "3.",
			expectedContent: "1. first\n2. second\n   3.",
		},
		{
			name:            "text before number no dedent",
			initialContent:  "1. first\n2. second\n   x",
			filePath:        "test.md",
			cursorLine:      2,
			typedChars:      "3.",
			expectedContent: "1. first\n2. second\n   x3.",
		},
		{
			name:            "multi-line list item skips continuation",
			initialContent:  "7. this is a long item\n   that continues here\n   ",
			filePath:        "test.md",
			cursorLine:      2,
			typedChars:      "8.",
			expectedContent: "7. this is a long item\n   that continues here\n8.",
		},
		{
			name:            "multi-line item with blank line",
			initialContent:  "7. long item\n   continued\n\n   ",
			filePath:        "test.md",
			cursorLine:      3,
			typedChars:      "8.",
			expectedContent: "7. long item\n   continued\n\n8.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.initialContent)
			te.CurrentBuffer().FilePath = tt.filePath

			// Enter insert mode first, then position cursor at end of target line
			te.ExecuteCommand("enter-insert-mode")

			lines := strings.Split(tt.initialContent, "\n")
			lastLineLen := len(lines[tt.cursorLine])
			te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), tt.cursorLine)
			te.CurrentBuffer().CursorX = lastLineLen

			for _, ch := range tt.typedChars {
				te.InsertChar(ch)
			}

			te.ExecuteCommand("exit-insert-mode")

			actual := te.CurrentBuffer().Rope.String()
			if actual != tt.expectedContent {
				t.Errorf("%s\nExpected:\n%q\nGot:\n%q",
					tt.name, tt.expectedContent, actual)
			}
		})
	}
}

func TestAutoDedentMarkdownListBullet(t *testing.T) {
	const initial = "- first\n- second\n  "
	te := NewTestEditorWithContent(80, 24, initial)
	te.CurrentBuffer().FilePath = "test.md"
	te.ExecuteCommand("enter-insert-mode")
	te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 2)
	te.CurrentBuffer().CursorX = 2

	te.InsertChar('-')
	te.ExecuteCommand("exit-insert-mode")

	const expected = "- first\n- second\n-"
	if actual := te.CurrentBuffer().Rope.String(); actual != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, actual)
	}

	te.ExecuteCommand("undo")
	if actual := te.CurrentBuffer().Rope.String(); actual != initial {
		t.Errorf("undo expected:\n%q\ngot:\n%q", initial, actual)
	}

	te.ExecuteCommand("redo")
	if actual := te.CurrentBuffer().Rope.String(); actual != expected {
		t.Errorf("redo expected:\n%q\ngot:\n%q", expected, actual)
	}
}
