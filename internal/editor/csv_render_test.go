package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
	"veda/internal/ui"
)

func newCSVTestEditor(width, height int, content string) *TestEditor {
	te := NewTestEditorWithContent(width, height, content)
	te.CurrentBuffer().FilePath = "people.csv"
	return te
}

func csvScreenText(te *TestEditor, row int) string {
	line := te.mockScreen.GetLine(row)
	gutter := te.CurrentBuffer().LineNumberWidth
	if len(line) <= gutter {
		return ""
	}
	return strings.TrimRight(line[gutter:], " ")
}

func TestCSVRenderingAlignsColumns(t *testing.T) {
	content := "name,age,city\nAlice,31,\"New York\"\nBob,8,Boston"
	te := newCSVTestEditor(80, 10, content)
	te.GetScreenContent()

	want := []string{
		"name   age  city",
		"Alice  31   New York",
		"Bob    8    Boston",
	}
	for row, expected := range want {
		if got := csvScreenText(te, row); got != expected {
			t.Fatalf("row %d = %q, want %q", row, got, expected)
		}
	}
	if got := te.GetBufferContent(); got != content {
		t.Fatalf("render changed buffer to %q", got)
	}
}

func TestCSVQuotedCommaRemainsPartOfValue(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age\n\"Doe, Jane\",31\n\"Said \"\"hello\"\"\",40")
	te.GetScreenContent()

	if got := csvScreenText(te, 1); got != "Doe, Jane     31" {
		t.Fatalf("quoted comma row = %q, want %q", got, "Doe, Jane     31")
	}
	if got := csvScreenText(te, 2); got != "Said \"hello\"  40" {
		t.Fatalf("escaped quote row = %q, want %q", got, "Said \"hello\"  40")
	}
}

func TestCSVInsertModeShowsRawText(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age\nAlice,31")
	te.mode = commands.InsertMode
	te.GetScreenContent()

	if got := csvScreenText(te, 0); got != "name,age" {
		t.Fatalf("insert mode row = %q, want raw CSV", got)
	}
}

func TestCSVColumnsUseAlternatingBackgrounds(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age\nAlice,31")
	te.GetScreenContent()

	gutter := te.CurrentBuffer().LineNumberWidth
	_, evenStyle := te.mockScreen.GetCell(gutter, 1)
	_, oddStyle := te.mockScreen.GetCell(gutter+len("Alice")+csvColumnGap, 1)
	_, normalBg, _ := te.config.Theme.GetStyle(te.config.Theme.Normal).Decompose()
	if evenStyle.Bg == normalBg || oddStyle.Bg == normalBg {
		t.Fatalf("a CSV column uses the normal background: even=%v odd=%v normal=%v", evenStyle.Bg, oddStyle.Bg, normalBg)
	}
	evenR, _, evenB := evenStyle.Bg.RGB()
	oddR, _, oddB := oddStyle.Bg.RGB()
	if evenB <= evenR {
		t.Fatalf("even column is not blue tinted: red=%d blue=%d", evenR, evenB)
	}
	if oddB <= oddR {
		t.Fatalf("odd column is not blue tinted: red=%d blue=%d", oddR, oddB)
	}
	if evenStyle.Bg == oddStyle.Bg {
		t.Fatalf("CSV columns use the same blue tint: %v", evenStyle.Bg)
	}
}

func TestCSVHeaderRowUsesRedBackground(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age\nAlice,31")
	te.GetScreenContent()

	gutter := te.CurrentBuffer().LineNumberWidth
	_, headerStyle := te.mockScreen.GetCell(gutter, 0)
	_, dataStyle := te.mockScreen.GetCell(gutter, 1)
	if headerStyle.Attrs&ui.AttrBold == 0 {
		t.Fatal("CSV header is not bold")
	}
	if headerStyle.Bg == dataStyle.Bg {
		t.Fatal("CSV header uses the data row background")
	}
	headerR, _, headerB := headerStyle.Bg.RGB()
	if headerR <= headerB {
		t.Fatalf("CSV header is not red tinted: red=%d blue=%d", headerR, headerB)
	}
}

func TestCSVCursorMapsToRenderedField(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age,city\nAlice,31,\"New York\"")
	te.SetCursorPosition(len("Alice,31,\""), 1)
	te.GetScreenContent()

	x, y := te.GetScreenCursorPosition()
	wantX := te.CurrentBuffer().LineNumberWidth + len("Alice") + csvColumnGap + len("age") + csvColumnGap
	if x != wantX || y != 1 {
		t.Fatalf("screen cursor = (%d, %d), want (%d, 1)", x, y, wantX)
	}
}

func TestCSVVerticalMovementPreservesDisplayedColumn(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age,city\nAlice,31,Boston")
	te.SetCursorPosition(len("name,"), 0)

	te.ExecuteCommand("down")

	if got := te.CurrentBuffer().CursorX; got != len("Alice,") {
		t.Fatalf("cursor X after moving down = %d, want %d", got, len("Alice,"))
	}
}

func TestCSVRowsDoNotWrap(t *testing.T) {
	te := newCSVTestEditor(20, 10, "a very long first column,second\nshort,value")

	lines := te.CalculateWrappedLines()
	if len(lines) != 2 {
		t.Fatalf("CSV rendered as %d visual lines, want 2", len(lines))
	}
}

func TestCSVMalformedRowFallsBackToRawText(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age\nAlice,\"unterminated")
	te.GetScreenContent()

	if got := csvScreenText(te, 1); got != "Alice,\"unterminated" {
		t.Fatalf("malformed row = %q, want raw text", got)
	}
}

func TestCSVRenderingHandlesCRLF(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age\r\nAlice,31\r\n")
	te.GetScreenContent()

	if got := csvScreenText(te, 0); got != "name   age" {
		t.Fatalf("CRLF header = %q, want %q", got, "name   age")
	}
	if got := csvScreenText(te, 1); got != "Alice  31" {
		t.Fatalf("CRLF row = %q, want %q", got, "Alice  31")
	}
}

func TestCSVEmptyBufferHasEditableRow(t *testing.T) {
	te := newCSVTestEditor(80, 10, "")

	lines := te.CalculateWrappedLines()
	if len(lines) != 1 || lines[0].LogicalLineNum != 0 {
		t.Fatalf("empty CSV lines = %+v, want one line at zero", lines)
	}
}

func TestCSVBlankRowUsesNormalBackground(t *testing.T) {
	te := newCSVTestEditor(80, 10, "name,age\nAlice,31\n")
	te.GetScreenContent()

	gutter := te.CurrentBuffer().LineNumberWidth
	_, blankStyle := te.mockScreen.GetCell(gutter, 2)
	_, normalBg, _ := te.config.Theme.GetStyle(te.config.Theme.Normal).Decompose()
	if blankStyle.Bg != normalBg {
		t.Fatalf("blank row background = %v, want normal background %v", blankStyle.Bg, normalBg)
	}
}
