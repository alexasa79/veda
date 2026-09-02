package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
	"veda/internal/ui"
)

func markdownPresentationScreenLine(te *TestEditor, row int) string {
	line := GetScreenLine(te.mockScreen.GetContent(), row)
	gutter := te.CurrentBuffer().LineNumberWidth
	if len(line) <= gutter {
		return ""
	}
	return strings.TrimRight(line[gutter:], " ")
}

func TestMarkdownTableUsesUnicodeGridInNormalMode(t *testing.T) {
	content := strings.Join([]string{
		"| hello | world |",
		"| ----- | ----- |",
		"| one   | two   |",
		"| ----- | ----- |",
	}, "\n")
	te := newMarkdownTableTestEditor(content)
	te.renderPane()

	want := []string{
		"┃ hello ┃ world ┃",
		"┣━━━━━━━╋━━━━━━━┫",
		"┃ one   ┃ two   ┃",
		"┗━━━━━━━┻━━━━━━━┛",
	}
	for row, expected := range want {
		if got := markdownPresentationScreenLine(te, row); got != expected {
			t.Fatalf("row %d = %q, want %q", row, got, expected)
		}
	}
	if got := te.GetBufferContent(); got != content {
		t.Fatalf("render changed buffer to %q", got)
	}
}

func TestMarkdownTableGridUsesOneStyle(t *testing.T) {
	te := newMarkdownTableTestEditor("| hello | world |\n| ----- | ----- |")
	te.renderPane()

	screen := te.mockScreen.GetContent()
	gutter := te.CurrentBuffer().LineNumberWidth
	vertical := screen[0][gutter].Style
	junction := screen[1][gutter].Style
	horizontal := screen[1][gutter+1].Style
	want := te.config.Theme.GetStyle(te.config.Theme.MarkdownTable)

	for name, got := range map[string]ui.Style{
		"vertical":   vertical,
		"junction":   junction,
		"horizontal": horizontal,
	} {
		if got.Fg != want.Fg || got.Attrs != want.Attrs {
			t.Fatalf("%s grid style = (%v, %v), want (%v, %v)", name, got.Fg, got.Attrs, want.Fg, want.Attrs)
		}
	}
}

func TestMarkdownTableShowsRawTextInInsertMode(t *testing.T) {
	te := newMarkdownTableTestEditor("| hello | world |\n| ----- | ----- |")
	te.mode = commands.InsertMode
	te.renderPane()

	if got := markdownPresentationScreenLine(te, 0); got != "| hello | world |" {
		t.Fatalf("insert mode row = %q, want raw Markdown", got)
	}
	if got := markdownPresentationScreenLine(te, 1); got != "| ----- | ----- |" {
		t.Fatalf("insert mode delimiter = %q, want raw Markdown", got)
	}
}

func TestMarkdownLinkPresentationWrapsDisplayedText(t *testing.T) {
	te := NewTestEditorWithContent(24, 10, "Before [link](https://example.com/a/very/long/path) after")
	te.CurrentBuffer().FilePath = "test.md"
	te.renderPane()

	if got := markdownPresentationScreenLine(te, 0); got != "Before link after" {
		t.Fatalf("normal mode row = %q, want displayed link text", got)
	}
}

func TestMarkdownLinkPresentationMapsScreenColumnsToSource(t *testing.T) {
	content := "Click [here](target.md) after"
	te := NewTestEditorWithContent(80, 10, content)
	te.CurrentBuffer().FilePath = "test.md"

	line, sourceX := te.MapVisualToLogical(0, te.CurrentBuffer().LineNumberWidth+len("Click "))
	if line != 0 || sourceX != strings.Index(content, "here") {
		t.Fatalf("link start mapped to (%d, %d), want (0, %d)", line, sourceX, strings.Index(content, "here"))
	}
	_, sourceX = te.MapVisualToLogical(0, te.CurrentBuffer().LineNumberWidth+len("Click here"))
	if want := strings.Index(content, " after"); sourceX != want {
		t.Fatalf("link end mapped to %d, want %d", sourceX, want)
	}
}

func TestMarkdownTableFormattingUsesUnicodeDisplayWidth(t *testing.T) {
	te := newMarkdownTableTestEditor("| Name | Value |\n| --- | --- |\n| 界 | 😀 |")
	te.SetCursorPosition(2, 2)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	want := "| Name | Value |\n| ---- | ----- |\n| 界   | 😀    |"
	if got := te.GetBufferContent(); got != want {
		t.Fatalf("formatted table:\n%s\nwant:\n%s", got, want)
	}
}
