package editor

import (
	"strings"
	"testing"

	"veda/internal/buffer"
	"veda/internal/ui"
)

func TestVerticalRulersPreserveBackgrounds(t *testing.T) {
	te := newFunctionHeaderTestEditor(t, "int measure() {\n  return 7;\n}\n\n")
	te.config.VerticalRulers = []int{32, 48}
	te.SetCursorPosition(0, 1)
	te.render()
	gutter := te.CurrentBuffer().LineNumberWidth
	for _, row := range []struct {
		y  int
		bg ui.Color
	}{
		{0, te.config.Theme.FunctionHeaderBackground()},
		{1, te.config.Theme.GetStyle(te.config.Theme.CurrentLine).Bg},
		{2, te.config.Theme.GetStyle(te.config.Theme.Normal).Bg},
		{8, te.config.Theme.GetStyle(te.config.Theme.Normal).Bg},
	} {
		for _, column := range te.config.VerticalRulers {
			ch, style := te.mockScreen.GetCell(gutter+column, row.y)
			if ch != '│' || style.Bg != row.bg {
				t.Errorf("ruler at (%d,%d) = %q with background %v; want │ with %v", column, row.y, ch, style.Bg, row.bg)
			}
			if style.Fg != te.config.Theme.GetStyle(te.config.Theme.VerticalRuler).Fg {
				t.Errorf("wrong ruler foreground at (%d,%d)", column, row.y)
			}
		}
	}
}

func TestVerticalRulersKeepTextAndWhitespaceMarkers(t *testing.T) {
	te := NewTestEditorWithContent(80, 16, "ab cd\n\tword\n界x\ntrail   \n\n")
	te.config.VerticalRulers = []int{1, 2, 3, 4, 6}
	te.whitespaceStyle.showTabs = true
	te.whitespaceStyle.showTrailingSpaces = true
	te.SetCursorPosition(0, 4)
	te.render()
	gutter := te.CurrentBuffer().LineNumberWidth
	for _, cell := range []struct {
		x, y int
		want rune
	}{
		{1, 0, 'b'},
		{2, 0, '│'},
		{3, 0, 'c'},
		{4, 0, 'd'},
		{0, 1, tabChar},
		{1, 1, '│'},
		{2, 1, '│'},
		{3, 1, '│'},
		{4, 1, 'w'},
		{0, 2, '界'},
		{2, 2, 'x'},
		{6, 3, spaceChar},
	} {
		if ch, _ := te.mockScreen.GetCell(gutter+cell.x, cell.y); ch != cell.want {
			t.Errorf("cell (%d,%d) = %q; want %q", cell.x, cell.y, ch, cell.want)
		}
	}
	if got := te.GetBufferContent(); got != "ab cd\n\tword\n界x\ntrail   \n\n" {
		t.Fatal("rulers modified the buffer")
	}
}

func TestVerticalRulersWrappedLines(t *testing.T) {
	te := NewTestEditorWithContent(30, 16, strings.Repeat("a", 60)+"\n\n")
	te.config.VerticalRulers = []int{10}
	te.SetCursorPosition(0, 1)
	te.render()
	gutter := te.CurrentBuffer().LineNumberWidth
	wrapped := te.CalculateWrappedLines()
	sawContinuation := false
	for i, wl := range wrapped {
		if wl.LogicalLineNum != 0 || wl.IsFirstLine {
			continue
		}
		sawContinuation = true
		want := 'a'
		if wl.EndChar-wl.StartChar <= 10 {
			want = '│'
		}
		if ch, _ := te.mockScreen.GetCell(gutter+10, te.getContentStartY()+i); ch != want {
			t.Errorf("wrapped row %d ruler cell = %q; want %q", i, ch, want)
		}
	}
	if !sawContinuation {
		t.Fatal("test did not wrap")
	}
}

func TestVerticalRulersDiffAndContextBackgrounds(t *testing.T) {
	te := NewTestEditorWithContent(80, 16, "short\n\n")
	te.config.VerticalRulers = []int{32}
	te.SetCursorPosition(0, 1)
	te.CurrentBuffer().DiffHints = []buffer.DiffLine{{LineType: buffer.DiffLineAdded}}
	te.render()
	gutter := te.CurrentBuffer().LineNumberWidth
	ch, style := te.mockScreen.GetCell(gutter+32, 0)
	_, neighbor := te.mockScreen.GetCell(gutter+31, 0)
	if ch != '│' || style.Bg != neighbor.Bg {
		t.Fatal("ruler changed the diff background")
	}
	contextBg := te.config.Theme.GetStyle(te.config.Theme.Context).Bg
	te.drawLineInternal(buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 0), gutter, 0, &contextBg)
	ch, style = te.mockScreen.GetCell(gutter+32, 0)
	if ch != '│' || style.Bg != contextBg {
		t.Fatal("ruler changed the context background")
	}
}

func TestVerticalRulersKeepSearchAndCursorHighlights(t *testing.T) {
	te := NewTestEditorWithContent(80, 16, "a b\n\n")
	te.config.VerticalRulers = []int{1, 32}
	gutter := te.CurrentBuffer().LineNumberWidth
	te.searchPattern = " "
	te.drawText(gutter, 0, te.config.Theme.GetStyle(te.config.Theme.Normal), "a b", 0, []int{1}, 0, 0, -1)
	ch, style := te.mockScreen.GetCell(gutter+1, 0)
	if ch != '│' || style.Bg != te.config.Theme.GetStyle(te.config.Theme.SearchHighlight).Bg {
		t.Fatal("ruler hid the search highlight")
	}
	te.searchPattern = ""
	te.virtualCursorCells = map[int]map[int]bool{0: {gutter + 32: true}}
	te.drawWrappedLine(buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 0), WrappedLine{
		LogicalLineNum: 0,
		StartChar:      0,
		EndChar:        3,
	}, nil, gutter, 0)
	ch, style = te.mockScreen.GetCell(gutter+32, 0)
	if ch != '│' || style.Bg != te.config.Theme.GetStyle(te.config.Theme.VirtualCursor).Bg {
		t.Fatal("ruler hid the virtual cursor")
	}
}

func TestVerticalRulersWindowBoundsAndDisable(t *testing.T) {
	te := NewTestEditorWithContent(80, 16, "short\n\n")
	te.config.VerticalRulers = []int{-1, 0, 20, 32, 100}
	te.screen = newWindowScreen(te.mockScreen, 5, 0, 30, 10, false)
	sentinel := ui.StyleDefault.Background(ui.NewRGBColor(90, 90, 90))
	te.mockScreen.SetContent(35, 0, 'x', nil, sentinel)
	te.renderPane()
	gutter := te.CurrentBuffer().LineNumberWidth
	if ch, _ := te.mockScreen.GetCell(5+gutter+20, 0); ch != '│' {
		t.Fatal("ruler not positioned relative to its pane")
	}
	if ch, style := te.mockScreen.GetCell(35, 0); ch != 'x' || style != sentinel {
		t.Fatal("ruler escaped its pane")
	}
	te.config.VerticalRulers = nil
	te.renderPane()
	if ch, _ := te.mockScreen.GetCell(5+gutter+20, 0); ch != ' ' {
		t.Fatal("disabled ruler remained on screen")
	}
}

func TestVerticalRulersSkipTerminal(t *testing.T) {
	te := NewTestEditor(80, 16)
	setupTerminalBufferForTest(te, 40, 6, "hello", NewTerminalEmulator(40, 6))
	te.config.VerticalRulers = []int{20}
	te.render()
	for y, row := range te.mockScreen.GetContent() {
		for x, cell := range row {
			if cell.Rune == '│' {
				t.Fatalf("ruler drawn over terminal at (%d,%d)", x, y)
			}
		}
	}
}

func TestVerticalRulersSkipStatusText(t *testing.T) {
	te := NewTestEditorWithContent(80, 16, "text")
	te.config.VerticalRulers = []int{0, 1, 2, 3, 4, 5}
	text := "status text"
	style := te.config.Theme.GetStyle(te.config.Theme.StatusBar)
	te.drawText(0, 0, style, text, 0, nil, -1, 0, -1)
	for x, want := range text {
		if ch, gotStyle := te.mockScreen.GetCell(x, 0); ch != want || gotStyle != style {
			t.Fatalf("status cell %d changed to %q with style %v", x, ch, gotStyle)
		}
	}
}
