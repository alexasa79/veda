package editor

import (
	"strings"
	"testing"
	"time"
	"veda/internal/clipboard"
	"veda/internal/commands"
)

const markdownOpeningMoves = `| Opening move in ATR14 units | Positions | Profit | Profit per $10,000 traded | Win rate |
|---|---:|---:|---:|---:|
| Q1: 0.0002 through 0.1233 | 977 | $2,279.01 | $9.07 | 29.0% |
| Q2: 0.1235 through 0.2616 | 976 | $3,220.37 | $12.78 | 29.8% |
| Q3: 0.2621 through 0.4676 | 976 | $4,541.87 | $17.94 | 28.6% |
| Q4: 0.4677 through 2.7823 | 976 | $5,654.47 | $23.32 | 24.4% |`

func TestMarkdownTableAutomaticAlignmentAndRulerReflow(t *testing.T) {
	te := NewTestEditorWithContent(120, 24, markdownOpeningMoves)
	te.CurrentBuffer().FilePath = "table.md"
	te.config.VerticalRulers = []int{80, 100}
	generation := te.CurrentBuffer().Generation
	checkWidth := func(want int) {
		t.Helper()
		te.renderPane()
		for row, source := range strings.Split(te.GetBufferContent(), "\n") {
			if got := te.markdownPresentedLine(row).displayWidth(0, len(source)); got != want {
				t.Fatalf("row %d width = %d, want %d", row, got, want)
			}
			if strings.Contains(markdownPresentationScreenLine(te, row), "\\") {
				t.Fatalf("row %d displays a continuation marker", row)
			}
		}
	}
	checkWidth(94)
	if te.GetBufferContent() != markdownOpeningMoves || te.CurrentBuffer().Generation != generation {
		t.Fatal("automatic alignment modified source")
	}
	if len(te.CurrentBuffer().History.SnapshotUndo(1)) != 0 {
		t.Fatal("rendering created undo history")
	}
	te.mode = commands.InsertMode
	te.renderPane()
	for row, source := range strings.Split(markdownOpeningMoves, "\n") {
		if got := markdownPresentationScreenLine(te, row); got != source {
			t.Fatalf("insert row %d = %q", row, got)
		}
	}
	te.mode = commands.NormalMode
	te.SetCursorPosition(3, 2)
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	narrow := te.GetBufferContent()
	if narrow == markdownOpeningMoves || !strings.Contains(narrow, "| \\\n") {
		t.Fatal("Alt+Q did not rewrite source with continuations")
	}
	checkWidth(80)
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatal(err)
	}
	wide := te.GetBufferContent()
	if len(strings.Split(wide, "\n")) != 6 || strings.Contains(wide, "| \\\n") {
		t.Fatal("wider ruler did not restore original rows")
	}
	checkWidth(94)
	for _, want := range []string{narrow, markdownOpeningMoves} {
		if err := te.ExecuteCommand("undo"); err != nil {
			t.Fatal(err)
		}
		if te.GetBufferContent() != want {
			t.Fatal("undo lost source")
		}
		te.renderPane()
	}
	for _, want := range []string{narrow, wide} {
		if err := te.ExecuteCommand("redo"); err != nil {
			t.Fatal(err)
		}
		if te.GetBufferContent() != want {
			t.Fatal("redo lost source")
		}
		te.renderPane()
	}
}

func TestMarkdownTableAutomaticallyAlignsCompactRows(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         []string
	}{
		{"compact", "|A|Count|\n|---|---:|\n|longer|2|", []string{"┃ A      ┃ Count ┃", "┣━━━━━━━━╋━━━━━━━┫", "┃ longer ┃     2 ┃"}},
		{"alignment", "|Left|Center|Right|\n|:---|:---:|---:|\n|x|y|z|", []string{"┃ Left ┃ Center ┃ Right ┃", "┣━━━━━━╋━━━━━━━━╋━━━━━━━┫", "┃ x    ┃   y    ┃     z ┃"}},
		{"links", "|A|B|\n|---|---|\n|[x](very-long-target.md)|z|", []string{"┃ A   ┃ B   ┃", "┣━━━━━╋━━━━━┫", "┃ x   ┃ z   ┃"}},
		{"no outer pipes", "A | B\n--- | ---\nx | y", []string{"┃ A   ┃ B   ┃", "┣━━━━━╋━━━━━┫", "┃ x   ┃ y   ┃"}},
		{"headerless", "|---|---:|\n|longer|2|", []string{"┏━━━━━━━━┳━━━━━━┓", "┃ longer ┃    2 ┃"}},
		{"incomplete", "|A|B|\n|---|---|\n|longer", []string{"┃ A      ┃ B   ┃", "┣━━━━━━━━╋━━━━━┫", "┃ longer ┃     ┃"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			te := newMarkdownTableTestEditor(test.source)
			te.renderPane()
			for row, want := range test.want {
				if got := markdownPresentationScreenLine(te, row); got != want {
					t.Fatalf("row %d = %q, want %q", row, got, want)
				}
			}
			if te.GetBufferContent() != test.source {
				t.Fatal("rendering changed source")
			}
		})
	}
}

func TestMarkdownTableWideRowsScrollTogether(t *testing.T) {
	te := NewTestEditorWithContent(40, 20, markdownOpeningMoves)
	te.CurrentBuffer().FilePath = "table.md"
	rows := strings.Split(markdownOpeningMoves, "\n")
	te.SetCursorPosition(strings.Index(rows[2], "29.0%"), 2)
	if len(te.CalculateWrappedLines()) != len(rows) {
		t.Fatal("wide table was screen-wrapped")
	}
	te.renderPane()
	y, x := te.GetCursorVisualPosition()
	if x < te.CurrentBuffer().LineNumberWidth || x >= 40 || y != 2 {
		t.Fatalf("cursor offscreen: (%d,%d)", x, y)
	}
	if got := te.mockScreen.GetContent()[y][x].Rune; got != '2' {
		t.Fatalf("cursor on %q", got)
	}
	row, source := te.MapVisualToLogical(y, x)
	if row != 2 || source != strings.Index(rows[2], "29.0%") {
		t.Fatal("scroll mapping lost source position")
	}
	for row := range rows {
		if te.markdownTableOffset(te.markdownPresentedLine(row)) != te.markdownTableOffset(te.markdownPresentedLine(2)) {
			t.Fatal("table rows scrolled separately")
		}
	}
	if err := te.ExecuteCommand("down"); err != nil {
		t.Fatal(err)
	}
	AssertCursorAt(t, te, strings.Index(rows[3], "29.8%"), 3)
	te.SetCursorPosition(0, 2)
	te.renderPane()
	if got := te.mockScreen.GetContent()[2][te.CurrentBuffer().LineNumberWidth].Rune; got != '┃' {
		t.Fatal("table did not scroll back")
	}
	te.mode = commands.InsertMode
	if len(te.CalculateWrappedLines()) <= len(rows) {
		t.Fatal("insert mode lost ordinary wrapping")
	}
}

func TestMarkdownTableDisplayMapsUnicodeAndHighlights(t *testing.T) {
	te := newMarkdownTableTestEditor("|Name|Count|\n|---|---:|\n|界|2|")
	te.SetCursorPosition(len("|界|"), 2)
	te.searchPattern = "2"
	te.renderPane()
	y, x := te.GetCursorVisualPosition()
	cell := te.mockScreen.GetContent()[y][x]
	if cell.Rune != '2' || cell.Style.Bg != te.config.Theme.GetStyle(te.config.Theme.SearchHighlight).Bg {
		t.Fatal("search highlight missed cell")
	}
	row, source := te.MapVisualToLogical(y, x)
	if row != 2 || source != len("|界|") {
		t.Fatal("Unicode mapping lost source position")
	}
	if err := te.ExecuteCommand("enter-visual-mode"); err != nil {
		t.Fatal(err)
	}
	te.renderPane()
	if te.mockScreen.GetContent()[y][x].Style.Bg != te.config.Theme.GetStyle(te.config.Theme.Selection).Bg {
		t.Fatal("selection missed cell")
	}
}

type markdownTestClipboard struct{ text string }

func (c *markdownTestClipboard) CopyToClipboard(text string) error              { c.text = text; return nil }
func (c *markdownTestClipboard) QueryClipboard(_ time.Duration) (string, error) { return c.text, nil }

func TestMarkdownTableDisplayCopyPasteUsesSource(t *testing.T) {
	const content = "|Name|Positions|\n|---|---:|\n|Q1|977|"
	te := newMarkdownTableTestEditor(content)
	clip := &markdownTestClipboard{}
	clipboard.ResetNativeClipboardState()
	clipboard.SetTerminalClipboard(clip)
	t.Cleanup(func() { clipboard.SetTerminalClipboard(nil); clipboard.ResetNativeClipboardState() })
	t.Setenv("PATH", "")
	te.SetCursorPosition(3, 2)
	_, before := te.GetCursorVisualPosition()
	if err := te.ExecuteCommand("right"); err != nil {
		t.Fatal(err)
	}
	_, after := te.GetCursorVisualPosition()
	AssertCursorAt(t, te, 4, 2)
	if after-before <= 1 {
		t.Fatal("cursor failed to skip virtual padding")
	}
	for _, reverse := range []bool{false, true} {
		te.SetCursorPosition(1, 2)
		direction := "right"
		if reverse {
			te.SetCursorPosition(6, 2)
			direction = "left"
		}
		if err := te.ExecuteCommand("enter-visual-mode"); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			if err := te.ExecuteCommand(direction); err != nil {
				t.Fatal(err)
			}
		}
		te.renderPane()
		if err := te.ExecuteCommand("yank-selection"); err != nil {
			t.Fatal(err)
		}
		if clip.text != "Q1|977" {
			t.Fatalf("copied %q, want source", clip.text)
		}
	}
	te.SetCursorPosition(0, 0)
	te.TypeKeys("3yy")
	if clip.text != content+"\n" {
		t.Fatalf("whole-table copy = %q", clip.text)
	}
	te.SetCursorPosition(0, 2)
	if err := te.ExecuteCommand("paste-after"); err != nil {
		t.Fatal(err)
	}
	if te.GetBufferContent() != content+"\n"+content+"\n" {
		t.Fatal("paste changed Markdown")
	}
	if err := te.ExecuteCommand("undo"); err != nil {
		t.Fatal(err)
	}
	if te.GetBufferContent() != content {
		t.Fatal("paste undo lost source")
	}
}

func TestMarkdownTableDisplayUpdatesAfterEditAndUndo(t *testing.T) {
	te := newMarkdownTableTestEditor("|A|B|\n|---|---|\n|x|y|")
	te.renderPane()
	te.SetCursorPosition(1, 2)
	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatal(err)
	}
	for _, r := range "longer" {
		if err := te.InsertChar(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := te.ExecuteCommand("exit-insert-mode"); err != nil {
		t.Fatal(err)
	}
	te.renderPane()
	if got := markdownPresentationScreenLine(te, 0); got != "┃ A       ┃ B   ┃" {
		t.Fatalf("stale header after edit: %q", got)
	}
	if err := te.ExecuteCommand("undo"); err != nil {
		t.Fatal(err)
	}
	te.renderPane()
	if got := markdownPresentationScreenLine(te, 0); got != "┃ A   ┃ B   ┃" {
		t.Fatalf("stale header after undo: %q", got)
	}
}

func TestMarkdownTableDisplayVirtualCursorScroll(t *testing.T) {
	te := NewTestEditorWithContent(40, 20, markdownOpeningMoves)
	te.CurrentBuffer().FilePath = "table.md"
	rows := strings.Split(markdownOpeningMoves, "\n")
	te.SetCursorPosition(strings.Index(rows[2], "29.0%"), 2)
	te.setVirtualCursorsFor(te.CurrentBuffer(), []VirtualCursor{{X: strings.Index(rows[3], "29.8%"), Y: 3}})
	te.renderPane()
	_, x := te.GetCursorVisualPosition()
	cell := te.mockScreen.GetContent()[3][x]
	if cell.Rune != '2' || cell.Style.Bg != te.config.Theme.GetStyle(te.config.Theme.VirtualCursor).Bg {
		t.Fatalf("virtual cursor missed scrolled cell: %+v", cell)
	}
}
