package editor

import (
	"testing"
	"veda/internal/syntax"
)

func TestMarkdownBlockquoteQuoteAndTextShareStyle(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")
	te.CurrentBuffer().FilePath = "test.md"

	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	if err := te.CurrentBuffer().Highlighter.SetLanguageFromFile(te.CurrentBuffer().FilePath); err != nil {
		t.Fatalf("failed to set markdown highlighter: %v", err)
	}

	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatalf("failed to enter insert mode: %v", err)
	}
	for _, ch := range "> quoted text" {
		if err := te.InsertChar(ch); err != nil {
			t.Fatalf("failed to insert %q: %v", string(ch), err)
		}
	}
	_ = te.ExecuteCommand("escape")

	if err := te.CurrentBuffer().Highlighter.ParseFull([]byte(te.CurrentBuffer().Rope.String())); err != nil {
		t.Fatalf("failed to parse markdown buffer: %v", err)
	}

	te.renderPane()

	contentY := te.getContentStartY()
	lineNumberWidth := te.CurrentBuffer().LineNumberWidth

	symbolRune, symbolStyle := te.mockScreen.GetCell(lineNumberWidth, contentY)
	if symbolRune != '>' {
		t.Fatalf("expected '>' at start of line, got %q", string(symbolRune))
	}

	textRune, textStyle := te.mockScreen.GetCell(lineNumberWidth+2, contentY)
	if textRune != 'q' {
		t.Fatalf("expected 'q' after quote marker, got %q", string(textRune))
	}

	if symbolStyle != textStyle {
		t.Fatalf("blockquote symbol and text styles differ: symbol=%v, text=%v", symbolStyle, textStyle)
	}
}
