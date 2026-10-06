package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
)

func newMarkdownTableTestEditor(content string) *TestEditor {
	te := NewTestEditorWithContent(100, 20, content)
	te.CurrentBuffer().FilePath = "table.md"
	return te
}

func TestRewrapFormatsMarkdownTable(t *testing.T) {
	content := strings.Join([]string{
		"| Ranking | Gains bought | Losers sold | Aligned | vs ORVM | Opposite |",
		"| --- | ---: | ---: | ---: | ---: | ---: |",
		"| ORVM relative volume | 57 | 51 | 108 | - | 84 |",
		"| Market-adjusted return | 40 | 44 | 84 | -24 | 62 |",
		"| True order flow | n/a | n/a | n/a | n/a | n/a |",
	}, "\n")
	want := strings.Join([]string{
		"| Ranking                | Gains bought | Losers sold | Aligned | vs ORVM | Opposite |",
		"| ---------------------- | -----------: | ----------: | ------: | ------: | -------: |",
		"| ORVM relative volume   |           57 |          51 |     108 |       - |       84 |",
		"| Market-adjusted return |           40 |          44 |      84 |     -24 |       62 |",
		"| True order flow        |          n/a |         n/a |     n/a |     n/a |      n/a |",
	}, "\n")
	te := newMarkdownTableTestEditor(content)
	te.config.VerticalRulers = []int{100}
	te.SetCursorPosition(4, 3)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	if got := te.GetBufferContent(); got != want {
		t.Fatalf("formatted table:\n%s\nwant:\n%s", got, want)
	}
}

func TestMarkdownTableFormattingPreservesCellContent(t *testing.T) {
	content := strings.Join([]string{
		"Name | Expression",
		"--- | ---:",
		"escaped \\| pipe | `left|right`",
	}, "\n")
	want := strings.Join([]string{
		"| Name            |   Expression |",
		"| --------------- | -----------: |",
		"| escaped \\| pipe | `left|right` |",
	}, "\n")
	te := newMarkdownTableTestEditor(content)
	te.SetCursorPosition(10, 2)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	if got := te.GetBufferContent(); got != want {
		t.Fatalf("formatted table:\n%s\nwant:\n%s", got, want)
	}
	if line, x := te.CurrentBuffer().CursorIterator.LineNumber(), te.CurrentBuffer().CursorX; line != 2 || x != 12 {
		t.Fatalf("cursor = (%d, %d), want (2, 12)", line, x)
	}
}

func TestMarkdownTableFormattingUndoRedo(t *testing.T) {
	content := "Before\n\n| A | Count |\n| --- | ---: |\n| longer | 2 |\n\nAfter"
	want := "Before\n\n| A      | Count |\n| ------ | ----: |\n| longer |     2 |\n\nAfter"
	te := newMarkdownTableTestEditor(content)
	te.SetCursorPosition(11, 4)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	formattedX := te.CurrentBuffer().CursorX
	if got := te.GetBufferContent(); got != want {
		t.Fatalf("formatted table = %q, want %q", got, want)
	}
	if err := te.ExecuteCommand("undo"); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if got := te.GetBufferContent(); got != content {
		t.Fatalf("undo content = %q, want %q", got, content)
	}
	if got := te.CurrentBuffer().CursorX; got != 11 {
		t.Fatalf("undo cursor X = %d, want 11", got)
	}
	if err := te.ExecuteCommand("redo"); err != nil {
		t.Fatalf("redo failed: %v", err)
	}
	if got := te.GetBufferContent(); got != want {
		t.Fatalf("redo content = %q, want %q", got, want)
	}
	if got := te.CurrentBuffer().CursorX; got != formattedX {
		t.Fatalf("redo cursor X = %d, want %d", got, formattedX)
	}
}

func TestMarkdownTableFormattingInInsertModeIsSeparateUndoUnit(t *testing.T) {
	te := newMarkdownTableTestEditor("| A | B |\n| --- | ---: |\n| one | 2 |")
	te.mode = commands.InsertMode
	te.CurrentBuffer().History.StartInsertSession("i")
	te.SetCursorPosition(2, 2)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	if te.mode != commands.InsertMode {
		t.Fatalf("mode = %v, want insert mode", te.mode)
	}
	if got := te.CurrentBuffer().History.SnapshotUndo(1); len(got) != 1 || got[0] != "Format Markdown table" {
		t.Fatalf("undo stack = %v", got)
	}
}

func TestRewrapPadsIncompleteMarkdownTableRows(t *testing.T) {
	content := "|hello|world|\n|---|---|\n|something interesting is going on"
	want := "| hello                             | world |\n| --------------------------------- | ----- |\n| something interesting is going on |       |"
	te := newMarkdownTableTestEditor(content)
	te.SetCursorPosition(3, 2)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	if got := te.GetBufferContent(); got != want {
		t.Fatalf("formatted table:\n%s\nwant:\n%s", got, want)
	}
}

func TestRewrapLeavesMarkdownTableWithExtraCellsUntouched(t *testing.T) {
	content := "| A | B |\n| --- | --- |\n| one | two | three |"
	te := newMarkdownTableTestEditor(content)
	te.SetCursorPosition(3, 2)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	if got := te.GetBufferContent(); got != content {
		t.Fatalf("table with extra cells changed to %q", got)
	}
}

func TestMarkdownTableFormattingPreservesAlignmentMarkers(t *testing.T) {
	content := "| Left | Center | Right |\n| :--- | :---: | ---: |\n| x | y | z |"
	want := "| Left | Center | Right |\n| :--- | :----: | ----: |\n| x    |   y    |     z |"
	te := newMarkdownTableTestEditor(content)
	te.SetCursorPosition(2, 0)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	if got := te.GetBufferContent(); got != want {
		t.Fatalf("formatted table:\n%s\nwant:\n%s", got, want)
	}
}

func TestMarkdownTableFormattingExpandsBodySeparator(t *testing.T) {
	content := strings.Join([]string{
		"| hello                             | world                         |",
		"| --------------------------------- | ----------------------------- |",
		"| something interesting is going on | something something something |",
		"|---|---|",
	}, "\n")
	want := strings.Join([]string{
		"| hello                             | world                         |",
		"| --------------------------------- | ----------------------------- |",
		"| something interesting is going on | something something something |",
		"| --------------------------------- | ----------------------------- |",
	}, "\n")
	te := newMarkdownTableTestEditor(content)
	te.SetCursorPosition(2, 3)

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}
	if got := te.GetBufferContent(); got != want {
		t.Fatalf("formatted table:\n%s\nwant:\n%s", got, want)
	}
}
