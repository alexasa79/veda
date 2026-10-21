package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/proxy"
	"veda/internal/syntax"
	"veda/internal/ui"
)

func TestChangePreviewBlocks(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		line                int
		want                string
	}{
		{"modified", "start\nold\nend", "start\nnew\nend", 1, "old"},
		{"unchanged", "start\nold\nend", "start\nnew\nend", 0, ""},
		{"added", "start\nend", "start\nadded\nend", 1, ""},
		{"deleted", "start\nremoved\nend", "start\nend", 1, "removed"},
		{"deleted at EOF", "start\nremoved", "start", 0, "removed"},
		{"deleted at start", "removed\nend", "end", 0, "removed"},
		{"empty current", "removed", "", 0, "removed"},
		{"unequal replacement", "start\nold one\nold two\nend", "start\nnew\nend", 1, "old one\nold two"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var lines []string
			if block := changePreviewLines(computeDiff(tc.before, tc.after), tc.line, strings.Count(tc.after, "\n")+1); block != nil {
				lines = block.lines
			}
			if got := strings.Join(lines, "\n"); got != tc.want {
				t.Fatalf("preview = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestChangePreviewWrap(t *testing.T) {
	lines := []string{"\t" + strings.Repeat("界", 50), "", strings.Repeat("x", 170)}
	rows := changePreviewRows(lines, 80)
	if len(rows) < 6 {
		t.Fatalf("expected wrapped rows, got %v", rows)
	}
	for _, row := range rows {
		if strings.ContainsRune(row.Text, '\t') || wrapTextWidth(row.Text) > 80 {
			t.Fatalf("invalid wrapped row %q", row.Text)
		}
	}
}

func previewTestBuffer(content string) *buffer.Buffer {
	return buffer.NewBuffer(content, "artifacts/preview.go", "artifacts/preview.go", config.IndentTabs, 4)
}

func seedChangePreview(te *TestEditor, buf *buffer.Buffer, before string) {
	te.changePreview = &changePreviewState{
		buffer: buf, generation: buf.Generation, changes: buf.GitChanges,
		diff: computeDiff(before, buf.Rope.String()),
	}
}

func previewTitle(screen *MockScreen) (int, int) {
	width, height := screen.Size()
	for y := 0; y < height; y++ {
		if x := strings.Index(screen.GetLine(y), "Before"); x >= 0 && x < width {
			return x, y
		}
	}
	return -1, -1
}

func TestChangePreviewPlacement(t *testing.T) {
	for _, zen := range []bool{false, true} {
		for _, cursor := range []int{0, 4, 16} {
			t.Run(fmt.Sprintf("zen=%v/cursor=%d", zen, cursor), func(t *testing.T) {
				te := NewTestEditor(160, 30)
				lines := make([]string, 22)
				for i := range lines {
					lines[i] = fmt.Sprintf("line %d", i)
				}
				before := strings.Join(lines, "\n")
				lines[cursor] = "changed"
				buf := previewTestBuffer(strings.Join(lines, "\n"))
				te.SetCurrentBuffer(buf)
				buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, cursor)
				te.windows[0].CursorLine = cursor
				te.zenMode = zen
				seedChangePreview(te, buf, before)
				te.render()
				x, y := previewTitle(te.mockScreen)
				if x < 0 {
					t.Fatal("preview not rendered")
				}
				win := te.windows[0]
				if want := win.Y + len(buf.ContextCache) + 2; y != want {
					t.Fatalf("preview y=%d, want fixed top position %d", y, want)
				}
				if x < win.X || x >= win.X+win.Width {
					t.Fatalf("preview outside active window: %d", x)
				}
				if y < win.Y+2 {
					t.Fatalf("preview at %d lacks top margin", y)
				}
				for row, cells := range te.mockScreen.GetContent() {
					for col, cell := range cells {
						if cell.Rune == '╮' && col != win.X+win.Width-4 {
							t.Fatalf("right border at %d lacks three-column margin", col)
						}
						if cell.Rune == '╯' && row > win.Y+win.Height-3 {
							t.Fatalf("bottom border at %d lacks two-row margin", row)
						}
					}
				}
			})
		}
	}
}

func TestChangePreviewGlobalToggle(t *testing.T) {
	te := NewTestEditor(100, 24)
	te.TypeKeys("sp")
	state := te.changePreview
	te.SetCurrentBuffer(previewTestBuffer("another buffer"))
	if state == nil || te.changePreview != state {
		t.Fatal("toggle did not survive switching buffers")
	}
	te.ExecuteCommand("toggle-change-preview")
	if te.changePreview != nil {
		t.Fatal("preview still enabled")
	}
}

func TestChangePreviewFollowsCursorWithinBlock(t *testing.T) {
	old, current := make([]string, 30), make([]string, 30)
	for i := range old {
		old[i] = fmt.Sprintf("old %02d", i)
		current[i] = fmt.Sprintf("new %02d", i)
	}
	te := NewTestEditor(160, 30)
	buf := previewTestBuffer(strings.Join(current, "\n"))
	te.SetCurrentBuffer(buf)
	seedChangePreview(te, buf, strings.Join(old, "\n"))
	for _, line := range []int{0, 20} {
		buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, line)
		te.render()
		_, titleY := previewTitle(te.mockScreen)
		found := false
		for y := titleY + 1; y < titleY+14; y++ {
			found = found || strings.Contains(te.mockScreen.GetLine(y), old[line])
		}
		if titleY < 0 || !found {
			t.Fatalf("preview does not follow cursor at line %d", line)
		}
	}
}

func TestChangePreviewInvalidatesEditedSnapshot(t *testing.T) {
	te := NewTestEditor(100, 24)
	buf := previewTestBuffer("new")
	te.SetCurrentBuffer(buf)
	seedChangePreview(te, buf, "old")
	// With no proxy, a stale comparison must be hidden until recomputed.
	te.proxyManager = nil
	buf.Insert(0, "edited ")
	if diff := te.changePreviewDiff(buf); diff != nil {
		t.Fatal("reused comparison after editing")
	}
}

func TestChangePreviewHiddenWithoutRoom(t *testing.T) {
	for _, size := range [][2]int{{20, 24}, {80, 6}} {
		te := NewTestEditor(size[0], size[1])
		buf := previewTestBuffer("new")
		te.SetCurrentBuffer(buf)
		seedChangePreview(te, buf, "old")
		te.render()
		if x, _ := previewTitle(te.mockScreen); x >= 0 || te.changePreview == nil {
			t.Fatal("small window must hide preview and retain toggle")
		}
	}
}

func TestChangePreviewPreservesContext(t *testing.T) {
	for _, zen := range []bool{false, true} {
		t.Run(fmt.Sprintf("zen=%v", zen), func(t *testing.T) {
			te := NewTestEditor(160, 30)
			lines := make([]string, 22)
			for i := range lines {
				lines[i] = fmt.Sprintf("line %d", i)
			}
			before := strings.Join(lines, "\n")
			lines[16] = "changed"
			buf := previewTestBuffer(strings.Join(lines, "\n"))
			te.SetCurrentBuffer(buf)
			buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 16)
			buf.ContextLine = 16
			buf.ContextCache = []int{0, 1, 2}
			te.showContext = true
			te.zenMode = zen
			te.render()
			win := te.windows[0]
			context := make([]string, len(buf.ContextCache))
			for i := range context {
				context[i] = te.mockScreen.GetLine(win.Y + i)
			}
			seedChangePreview(te, buf, before)
			te.render()
			_, y := previewTitle(te.mockScreen)
			if y < win.Y+len(context) {
				t.Fatalf("preview at %d overlaps context ending at %d", y, win.Y+len(context))
			}
			for i, want := range context {
				if got := te.mockScreen.GetLine(win.Y + i); got != want {
					t.Fatalf("context row %d changed: %q => %q", i, want, got)
				}
			}
			left := -1
			for x := win.X; x < win.X+win.Width; x++ {
				if ch, _ := te.mockScreen.GetCell(x, y); ch == '╭' {
					left = x
					break
				}
			}
			right := win.X + win.Width - 4
			for _, corner := range []struct {
				x, y int
				want rune
			}{{left, y, '╭'}, {right, y, '╮'}, {left, y + 2, '╰'}, {right, y + 2, '╯'}} {
				got, style := te.mockScreen.GetCell(corner.x, corner.y)
				if got != corner.want {
					t.Errorf("corner at (%d,%d) = %q, want %q", corner.x, corner.y, got, corner.want)
				}
				if style.Attrs&ui.AttrBold == 0 || style.Fg != te.config.Theme.GetStyle(te.config.Theme.Normal).Fg {
					t.Errorf("border at (%d,%d) should use bold code foreground", corner.x, corner.y)
				}
			}
		})
	}
}

func TestChangePreviewSyntaxHighlighting(t *testing.T) {
	for _, tc := range []struct {
		name, before string
		probe        rune
		token        syntax.TokenType
	}{
		{"keyword", "func oldName() {}", 'f', syntax.TokenTypeKeyword},
		{"wrapped string", "var label = \"" + strings.Repeat("界", 50) + "\"", '界', syntax.TokenTypeString},
		{"multiline comment", "/* first\n\t界old\nlast */", '界', syntax.TokenTypeComment},
	} {
		t.Run(tc.name, func(t *testing.T) {
			te := NewTestEditor(120, 30)
			buf := previewTestBuffer("replacement")
			te.SetCurrentBuffer(buf)
			te.render()
			tree := buf.Highlighter.GetTree()
			seedChangePreview(te, buf, tc.before)
			te.render()
			_, titleY := previewTitle(te.mockScreen)
			if titleY < 0 {
				t.Fatal("preview not rendered")
			}
			background := te.config.Theme.GetStyle(te.config.Theme.Normal).Bg
			want := te.config.Theme.GetStyle(syntax.GetStyleForToken(tc.token, te.config.Theme)).Background(background)
			count := 0
			for y, cells := range te.mockScreen.GetContent() {
				if y <= titleY {
					continue
				}
				for x := range cells {
					ch, style := te.mockScreen.GetCell(x, y)
					if ch == tc.probe {
						count++
						if style != want {
							t.Fatalf("style at (%d,%d) = %v, want %v", x, y, style, want)
						}
					}
				}
			}
			if count == 0 || tc.name == "wrapped string" && count != 50 {
				t.Fatalf("highlighted %d probe characters", count)
			}
			if buf.Highlighter.GetTree() != tree {
				t.Fatal("preview replaced the current buffer's syntax tree")
			}
		})
	}
}

func TestChangePreviewExpandsIntoUnusedSpace(t *testing.T) {
	for _, tc := range []struct {
		name  string
		width int
		text  string
	}{
		{"short code", 180, "short"},
		{"long code keeps minimum", 180, strings.Repeat("x", 120)},
		{"wide Unicode", 180, strings.Repeat("界", 30)},
		{"tabs", 180, "\t\tshort"},
		{"trailing whitespace", 180, "short" + strings.Repeat(" ", 100)},
		{"narrow window", 60, "short"},
	} {
		for _, cursor := range []int{0, 20} {
			t.Run(fmt.Sprintf("%s/cursor=%d", tc.name, cursor), func(t *testing.T) {
				te := NewTestEditor(tc.width, 30)
				lines := make([]string, 22)
				for i := range lines {
					lines[i] = fmt.Sprintf("%02d%s", i, tc.text)
				}
				before := strings.Join(lines, "\n")
				lines[cursor] = "changed"
				buf := previewTestBuffer(strings.Join(lines, "\n"))
				te.SetCurrentBuffer(buf)
				buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, cursor)
				seedChangePreview(te, buf, before)
				te.render()
				_, y := previewTitle(te.mockScreen)
				if y < 0 {
					t.Fatal("preview not rendered")
				}
				left, right := -1, -1
				for x := 0; x < tc.width; x++ {
					ch, _ := te.mockScreen.GetCell(x, y)
					if ch == '╭' {
						left = x
					}
					if ch == '╮' {
						right = x
					}
				}
				textRight := buf.LineNumberWidth + 2 + wrapTextWidth(strings.TrimRight(tc.text, " \t"))
				want := min(tc.width-4, max(82, tc.width-textRight-4))
				if left < 0 || right-left+1 != want {
					t.Fatalf("preview width = %d, want %d (left=%d, right=%d)", right-left+1, want, left, right)
				}
				if want > 82 && left <= textRight {
					t.Fatalf("expanded preview at %d covers text ending at %d", left, textRight)
				}
			})
		}
	}
}

func TestDeletedTextAtLine(t *testing.T) {
	for _, tc := range []struct {
		name, content  string
		anchor, cursor int
	}{
		{"shifted deletion", "one\ntwo\nthree", 3, 2},
		{"EOF without newline", "one", 2, 0},
		{"EOF with newline", "one\n", 2, 1},
		{"entire file", "", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := previewTestBuffer(tc.content)
			buf.GitChanges = &proxy.FileChanges{DeletedHunks: []proxy.DeletedHunk{
				{Line: tc.anchor, Lines: []string{"removed"}},
			}}
			if got := deletedTextAtLine(buf, tc.cursor); strings.Join(got, "\n") != "removed" {
				t.Fatalf("deleted text at cursor = %q", got)
			}
		})
	}
}

func TestChangePreviewBackgroundAndRelationship(t *testing.T) {
	for _, tc := range []struct {
		name, before, current, label string
		deleted                      bool
	}{
		{"single replacement", "old one\nold two", "replacement", "replaces L1", false},
		{"block replacement", "old one\nold two", "new one\nnew two", "replaces L1–L2", false},
		{"deletion", "removed", "surviving", "deletion at L1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			te := NewTestEditor(160, 30)
			// Ensure a context background cannot accidentally satisfy the check.
			previewTheme := *te.config.Theme
			te.config.Theme = &previewTheme
			te.config.Theme.Context.Background = "#123456"
			buf := previewTestBuffer(tc.current)
			te.SetCurrentBuffer(buf)
			seedChangePreview(te, buf, tc.before)
			if tc.deleted {
				buf.GitChanges = &proxy.FileChanges{DeletedHunks: []proxy.DeletedHunk{{Line: 1, Lines: []string{tc.before}}}}
			}
			te.render()
			_, titleY := previewTitle(te.mockScreen)
			if titleY < 0 || !strings.Contains(te.mockScreen.GetLine(titleY), tc.label) {
				t.Fatalf("preview does not identify %q: %q", tc.label, te.mockScreen.GetLine(titleY))
			}
			background := te.config.Theme.GetStyle(te.config.Theme.Normal).Bg
			left, right := -1, -1
			for x := 0; x < 160; x++ {
				ch, _ := te.mockScreen.GetCell(x, titleY)
				if ch == '╭' {
					left = x
				}
				if ch == '╮' {
					right = x
				}
			}
			if left < 0 || right < 0 {
				t.Fatal("missing preview borders")
			}
			for y := titleY; y < 30; y++ {
				for x := left; x <= right; x++ {
					_, style := te.mockScreen.GetCell(x, y)
					if style.Bg != background {
						t.Fatalf("background at (%d,%d) = %v, want %v", x, y, style.Bg, background)
					}
				}
				if ch, _ := te.mockScreen.GetCell(left, y); ch == '╰' {
					break
				}
			}
		})
	}
}
