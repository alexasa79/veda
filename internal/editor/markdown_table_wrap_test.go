package editor

import (
	"strings"
	"testing"
)

func TestMarkdownTableWithoutHeader(t *testing.T) {
	for _, prefix := range []string{"", "Before\n\n"} {
		content := prefix + "| --- | --- |\n| DFS | used for non-weighted |"
		te := newMarkdownTableTestEditor(content)
		row := strings.Count(prefix, "\n")
		te.SetCursorPosition(2, row+1)
		if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
			t.Fatal(err)
		}
		want := prefix + "| --- | --------------------- |\n| DFS | used for non-weighted |"
		if got := te.GetBufferContent(); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
		te.renderPane()
		if got := markdownPresentationScreenLine(te, row); got != "┏━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━┓" {
			t.Fatalf("top border = %q", got)
		}
		if got := markdownPresentationScreenLine(te, row+1); got != "┃ DFS ┃ used for non-weighted ┃" {
			t.Fatalf("row = %q", got)
		}
	}
}

func TestMarkdownTableRewrapCyclesWidthsAndRestoresRows(t *testing.T) {
	content := "| --- | --- |\n| DFS | used for non-weighted graphs with many vertices |\n| BFS | another independent row |"
	te := newMarkdownTableTestEditor(content)
	te.config.VerticalRulers = []int{32, 80}
	te.SetCursorPosition(strings.Index(strings.Split(content, "\n")[1], "graphs"), 1)
	originalX := te.CurrentBuffer().CursorX
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	narrow := te.GetBufferContent()
	if !strings.Contains(narrow, "| \\\n") {
		t.Fatalf("no continuation in %q", narrow)
	}
	for row, line := range strings.Split(narrow, "\n") {
		presented := te.markdownPresentedLine(row)
		if width := presented.displayWidth(0, len(line)); width != 32 {
			t.Fatalf("displayed table width = %d, want ruler width 32: %q", width, line)
		}
	}
	buf := te.CurrentBuffer()
	if !strings.HasPrefix(buf.CursorIterator.Line()[buf.CursorX:], "graphs") {
		t.Fatalf("cursor lost word: %q at %d", buf.CursorIterator.Line(), buf.CursorX)
	}
	narrowRow, narrowX := buf.CursorIterator.LineNumber(), buf.CursorX
	te.renderPane()
	for row := range strings.Split(narrow, "\n") {
		if strings.HasSuffix(markdownPresentationScreenLine(te, row), "\\") {
			t.Fatal("continuation marker visible in normal mode")
		}
	}
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	wide := te.GetBufferContent()
	if strings.Count(wide, "\n") != 2 || strings.Contains(wide, "| \\\n") {
		t.Fatalf("rows not rejoined: %q", wide)
	}
	if !strings.Contains(wide, "used for non-weighted graphs with many vertices") {
		t.Fatalf("cell text changed: %q", wide)
	}
	if !strings.HasPrefix(buf.CursorIterator.Line()[buf.CursorX:], "graphs") {
		t.Fatal("wide cursor lost word")
	}
	if err := te.ExecuteCommand("undo"); err != nil {
		t.Fatal(err)
	}
	if te.GetBufferContent() != narrow || buf.CursorIterator.LineNumber() != narrowRow || buf.CursorX != narrowX {
		t.Fatalf("undo narrow: content matches %v, cursor (%d,%d), want (%d,%d)", te.GetBufferContent() == narrow, buf.CursorIterator.LineNumber(), buf.CursorX, narrowRow, narrowX)
	}
	if err := te.ExecuteCommand("undo"); err != nil {
		t.Fatal(err)
	}
	if te.GetBufferContent() != content || buf.CursorIterator.LineNumber() != 1 || buf.CursorX != originalX {
		t.Fatal("undo did not restore original table and cursor")
	}
	for i := 0; i < 2; i++ {
		if err := te.ExecuteCommand("redo"); err != nil {
			t.Fatal(err)
		}
	}
	if te.GetBufferContent() != wide {
		t.Fatal("redo did not restore wide table")
	}
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	if te.GetBufferContent() != narrow {
		t.Fatal("third rewrap did not cycle to first ruler")
	}
}

func TestMarkdownTableWrappedHeaderAndMarkup(t *testing.T) {
	content := "| Long heading with words | Other heading |\n| :--- | ---: |\n| [two words](target.md) and more words | `a | b` plus \\| escaped |\n| | independent empty first cell |\n| --- | --- |"
	te := newMarkdownTableTestEditor(content)
	te.config.VerticalRulers = []int{44, 120}
	te.SetCursorPosition(3, 0)
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	narrow := te.GetBufferContent()
	for _, atom := range []string{"[two words](target.md)", "`a | b`", "\\|"} {
		if !strings.Contains(narrow, atom) {
			t.Fatalf("split atom %q in %q", atom, narrow)
		}
	}
	table, ok := findMarkdownTable(strings.Split(narrow, "\n"), 0)
	if !ok || !table.valid || table.startLine != 0 {
		t.Fatal("wrapped header not recognized")
	}
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	wide := strings.Split(te.GetBufferContent(), "\n")
	if len(wide) != 5 {
		t.Fatalf("lost row boundaries: %q", wide)
	}
	row, _ := parseMarkdownTableRow(wide[3])
	if row.cells[0].text != "" || row.cells[1].text != "independent empty first cell" {
		t.Fatalf("empty first cell changed: %q", wide[3])
	}
}

func TestMarkdownTableWrapUnicodeAndNarrowGuide(t *testing.T) {
	for _, width := range []int{1, 26} {
		te := newMarkdownTableTestEditor("  | --- | --- |\n  | 界 界 界 界 | 😀 😀 😀 😀 😀 |")
		te.config.VerticalRulers = []int{width}
		te.SetCursorPosition(4, 1)
		if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
			t.Fatal(err)
		}
		for row, line := range strings.Split(te.GetBufferContent(), "\n") {
			if !strings.HasPrefix(line, "  |") {
				t.Fatalf("lost indentation: %q", line)
			}
			if width == 26 && te.markdownPresentedLine(row).displayWidth(0, len(line)) != width {
				t.Fatalf("wide Unicode line: %q", line)
			}
		}
	}
}

func TestMarkdownTableWrapPreservesVirtualCursors(t *testing.T) {
	content := "| --- | --- |\n| DFS | used for non-weighted graphs with many vertices |\n\nAfter"
	te := newMarkdownTableTestEditor(content)
	te.config.VerticalRulers = []int{32}
	te.SetCursorPosition(2, 1)
	graphX := strings.Index(strings.Split(content, "\n")[1], "graphs")
	original := []VirtualCursor{{X: graphX, Y: 1}, {X: 2, Y: 3}}
	te.setVirtualCursorsFor(te.CurrentBuffer(), original)
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	wrapped := strings.Split(te.GetBufferContent(), "\n")
	cursors := te.GetVirtualCursors()
	if !strings.HasPrefix(wrapped[cursors[0].Y][cursors[0].X:], "graphs") {
		t.Fatal("virtual cursor lost word")
	}
	if wrapped[cursors[1].Y] != "After" {
		t.Fatal("virtual cursor below table lost line")
	}
	if err := te.ExecuteCommand("undo"); err != nil {
		t.Fatal(err)
	}
	for i, cursor := range te.GetVirtualCursors() {
		if cursor != original[i] {
			t.Fatal("undo lost virtual cursor")
		}
	}
	if err := te.ExecuteCommand("redo"); err != nil {
		t.Fatal(err)
	}
	for i, cursor := range te.GetVirtualCursors() {
		if cursor != cursors[i] {
			t.Fatal("redo lost virtual cursor")
		}
	}
}

func TestMarkdownTableWrapKeepsDashesAsCellText(t *testing.T) {
	te := newMarkdownTableTestEditor("| --- | --- |\n| alpha --- | bravo --- |")
	te.config.VerticalRulers = []int{19, 80}
	te.SetCursorPosition(2, 1)
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(te.GetBufferContent(), "\n")
	if len(lines) != 2 {
		t.Fatalf("dashes became separator: %q", lines)
	}
	row, _ := parseMarkdownTableRow(lines[1])
	if row.cells[0].text != "alpha ---" || row.cells[1].text != "bravo ---" {
		t.Fatalf("dashes changed: %q", lines[1])
	}
}

func TestMarkdownTableWrapUsesDefaultWidth(t *testing.T) {
	te := newMarkdownTableTestEditor("| --- | --- |\n| DFS | " + strings.Repeat("many vertices ", 12) + "|")
	te.config.VerticalRulers = nil
	te.SetCursorPosition(2, 1)
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	content := te.GetBufferContent()
	if !strings.Contains(content, "| \\\n") {
		t.Fatal("table did not wrap at default width")
	}
	for row, line := range strings.Split(content, "\n") {
		if te.markdownPresentedLine(row).displayWidth(0, len(line)) != defaultWrapWidth {
			t.Fatalf("line exceeds default width: %q", line)
		}
	}
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	if te.GetBufferContent() != content {
		t.Fatal("repeating the same width changed the table")
	}
}
