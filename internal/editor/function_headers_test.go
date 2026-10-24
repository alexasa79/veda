package editor

import (
	"testing"

	"veda/internal/buffer"
	"veda/internal/syntax"
	"veda/internal/ui"
)

func newFunctionHeaderTestEditor(t *testing.T, content string) *TestEditor {
	t.Helper()
	te := NewTestEditorWithContent(80, 16, content)
	highlighter := te.CurrentBuffer().Highlighter
	if err := highlighter.SetLanguageFromFile("sample.cc"); err != nil {
		t.Fatal(err)
	}
	if err := highlighter.ParseFull([]byte(content)); err != nil {
		t.Fatal(err)
	}
	te.SetCursorPosition(0, te.CurrentBuffer().LineCount-1)
	return te
}

func TestFunctionHeaderRendering(t *testing.T) {
	content := "int\nmeasure(\n    int length) {\n  return length;\n}\n\n"
	te := newFunctionHeaderTestEditor(t, content)
	te.render()
	startX := te.CurrentBuffer().LineNumberWidth
	headerBg := te.config.Theme.FunctionHeaderBackground()
	normalBg := te.config.Theme.GetStyle(te.config.Theme.Normal).Bg
	for _, cell := range []struct{ x, y int }{{startX, 0}, {startX, 1}, {startX + 4, 2}} {
		_, bg, _ := te.mockScreen.GetCellStyle(cell.x, cell.y)
		if bg != headerBg {
			t.Errorf("header (%d,%d) background = %v; want %v", cell.x, cell.y, bg, headerBg)
		}
	}
	width, _ := te.mockScreen.Size()
	for row := 0; row < 3; row++ {
		for x := startX; x < width; x++ {
			_, bg, _ := te.mockScreen.GetCellStyle(x, row)
			if bg != headerBg {
				t.Errorf("header band (%d,%d) = %v; want %v", x, row, bg, headerBg)
			}
		}
		_, gutterBg, _ := te.mockScreen.GetCellStyle(0, row)
		if gutterBg == headerBg {
			t.Errorf("header band spilled into gutter on row %d", row)
		}
	}
	_, bodyBg, _ := te.mockScreen.GetCellStyle(startX+2, 3)
	if bodyBg != normalBg {
		t.Errorf("body background = %v; want %v", bodyBg, normalBg)
	}
	tokens := te.CurrentBuffer().Highlighter.GetTokensForLine(0, 3)
	if len(tokens) == 0 {
		t.Fatal("missing syntax token")
	}
	wantFg := te.config.Theme.GetStyle(syntax.GetStyleForToken(tokens[0].TokenType, te.config.Theme)).Fg
	_, cellStyle := te.mockScreen.GetCell(startX, 0)
	fg := cellStyle.Fg
	if fg != wantFg {
		t.Errorf("syntax foreground changed: got %v; want %v", fg, wantFg)
	}
}

func TestInlineFunctionHeaderRendering(t *testing.T) {
	content := "int measure() { return 7; }\n\n"
	te := newFunctionHeaderTestEditor(t, content)
	te.render()
	want := te.config.Theme.FunctionHeaderBackground()
	for x := te.CurrentBuffer().LineNumberWidth; x < 80; x++ {
		_, bg, _ := te.mockScreen.GetCellStyle(x, 0)
		if bg != want {
			t.Fatalf("inline function band at %d = %v; want %v", x, bg, want)
		}
	}
}

func TestFunctionHeaderWrappedRendering(t *testing.T) {
	content := "int measure(int length, int width) {\n  return length;\n}\n\n"
	te := newFunctionHeaderTestEditor(t, content)
	iter := buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 0)
	tokens := te.CurrentBuffer().Highlighter.GetTokensForLine(iter.Position(), iter.LineEnd())
	startX := te.CurrentBuffer().LineNumberWidth
	te.drawWrappedLine(iter, WrappedLine{
		LogicalLineNum: 0,
		StartChar:      16,
		EndChar:        len(iter.Line()),
	}, tokens, startX, 0)
	_, bg, _ := te.mockScreen.GetCellStyle(startX, 0)
	if bg != te.config.Theme.FunctionHeaderBackground() {
		t.Fatal("wrapped continuation did not receive header background")
	}
	_, paddingBg, _ := te.mockScreen.GetCellStyle(79, 0)
	if paddingBg != bg {
		t.Fatal("wrapped header background must reach the window edge")
	}
}

func TestFunctionHeaderHighlightPrecedence(t *testing.T) {
	te := newFunctionHeaderTestEditor(t, "int measure() { return 7; }\n\n")
	startX := te.CurrentBuffer().LineNumberWidth
	normal := te.config.Theme.GetStyle(te.config.Theme.Normal)
	for _, override := range []ui.Style{
		te.config.Theme.GetStyle(te.config.Theme.CurrentLine),
		te.config.Theme.GetStyle(te.config.Theme.Context),
		te.config.Theme.GetStyle(te.config.Theme.DiffAdded),
	} {
		te.drawText(startX, 0, normal.Background(override.Bg), "int", 0, nil, 0, 0, -1)
		_, bg, _ := te.mockScreen.GetCellStyle(startX, 0)
		if bg != override.Bg {
			t.Errorf("background override lost: got %v; want %v", bg, override.Bg)
		}
	}
	te.searchPattern = "int"
	te.drawText(startX, 0, normal, "int", 0, []int{0}, 0, 0, -1)
	_, bg, _ := te.mockScreen.GetCellStyle(startX, 0)
	if bg != te.config.Theme.GetStyle(te.config.Theme.SearchHighlight).Bg {
		t.Fatal("function header hid search highlight")
	}
	te.searchPattern = ""
	te.CurrentBuffer().DiffHints = []buffer.DiffLine{{LineType: buffer.DiffLineAdded}}
	if got := te.functionHeaderRanges(startX, 0, 0, 3, normal); len(got) != 0 {
		t.Fatal("header must not override changed diff lines")
	}
	if got := te.functionHeaderRanges(0, 0, 0, 3, normal); len(got) != 0 {
		t.Fatal("header must not tint line numbers")
	}
}

func TestFunctionHeaderBandWindowBounds(t *testing.T) {
	te := newFunctionHeaderTestEditor(t, "int\nmeasure() {\n  return 7;\n}\n\n")
	te.screen = newWindowScreen(te.mockScreen, 5, 0, 30, 10, false)
	sentinel := ui.NewRGBColor(90, 90, 90)
	te.mockScreen.SetContent(35, 0, 'x', nil, ui.StyleDefault.Background(sentinel))
	iter := buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 0)
	te.drawLineInternal(iter, te.CurrentBuffer().LineNumberWidth, 0, nil)
	_, edgeBg, _ := te.mockScreen.GetCellStyle(34, 0)
	if edgeBg != te.config.Theme.FunctionHeaderBackground() {
		t.Fatal("header band did not reach the pane edge")
	}
	_, outsideBg, _ := te.mockScreen.GetCellStyle(35, 0)
	if outsideBg != sentinel {
		t.Fatal("header band escaped its pane")
	}
}

func TestFunctionHeaderBandKeepsCurrentLineBackground(t *testing.T) {
	te := newFunctionHeaderTestEditor(t, "int\nmeasure() {\n  return 7;\n}\n\n")
	te.SetCursorPosition(0, 0)
	te.render()
	want := te.config.Theme.GetStyle(te.config.Theme.CurrentLine).Bg
	for _, x := range []int{te.CurrentBuffer().LineNumberWidth, 79} {
		_, bg, _ := te.mockScreen.GetCellStyle(x, 0)
		if bg != want {
			t.Errorf("current line background at %d = %v; want %v", x, bg, want)
		}
	}
}

func TestFunctionHeaderBandAcrossBlankSignatureLine(t *testing.T) {
	te := newFunctionHeaderTestEditor(t, "int\n\nmeasure() {\n  return 7;\n}\n\n")
	te.render()
	_, bg, _ := te.mockScreen.GetCellStyle(79, 1)
	if bg != te.config.Theme.FunctionHeaderBackground() {
		t.Fatal("blank line inside signature must keep the header band continuous")
	}
}

func TestInlineFunctionHeaderBand(t *testing.T) {
	tests := []struct {
		name, content string
		row           int
	}{
		{"nonempty function", "int measure() { return 7; }\n\n", 0},
		{"destructor", "class Parcel {\n public:\n  ~Parcel() { Reset(); }\n};\n\n", 2},
		{"empty function", "void reset() {}\n\n", 0},
		{"spaces inside body", "void reset() {  }\n\n", 0},
		{"split header", "void\nreset() {}\n\n", 1},
		{"separate brace line", "void reset()\n{}\n\n", 1},
		{"constructor initializer", "class Parcel {\n public:\n  explicit Parcel(Store& store) : store_(store) {}\n};\n\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := newFunctionHeaderTestEditor(t, tt.content)
			te.render()
			want := te.config.Theme.FunctionHeaderBackground()
			for x := te.CurrentBuffer().LineNumberWidth; x < 80; x++ {
				_, bg, _ := te.mockScreen.GetCellStyle(x, tt.row)
				if bg != want {
					t.Fatalf("inline function band at (%d,%d) = %v; want %v", x, tt.row, bg, want)
				}
			}
		})
	}
}
