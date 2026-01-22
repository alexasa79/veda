package editor

import (
	"testing"
	"veda/internal/syntax"
)

func TestMarkdownSublistMarkerHighlighting(t *testing.T) {
	content := "- item 1\n  - sub-item"

	te := NewTestEditorWithContent(40, 6, content)
	te.CurrentBuffer().FilePath = "test.md"

	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	err := te.CurrentBuffer().Highlighter.SetLanguageFromFile(te.CurrentBuffer().FilePath)
	if err != nil {
		t.Fatalf("Failed to set markdown language: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(te.CurrentBuffer().Rope.String()))

	te.render()

	markerStyle := getCellStyle(te, 1, "- ")
	if markerStyle == nil {
		t.Fatal("Could not find sub-list marker")
	}

	expectedStyle := te.config.Theme.GetStyle(te.config.Theme.MarkdownListMarker)
	expectedFg, _, _ := expectedStyle.Decompose()
	actualFg, _, _ := markerStyle.Decompose()

	if actualFg != expectedFg {
		t.Errorf("Sub-list marker has wrong color: got %v, expected %v", actualFg, expectedFg)
	}
}
