package editor

import (
	"testing"

	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/proxy"
)

func TestDiffGutterSkipsDeletedFiller(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")
	left := "before\nremoved one\nremoved two\nremoved three\nremoved four\nafter\nend"
	right := "before\nafter\nadded elsewhere\nend"
	diff := te.computeDiffViaProxy(left, right)
	head := buffer.NewBuffer(createBufferContent(diff.LeftLines), "gutter.go@HEAD", "gutter.go@HEAD", config.IndentTabs, 4)
	current := buffer.NewBuffer(createBufferContent(diff.RightLines), "gutter.go@current", "gutter.go@current", config.IndentTabs, 4)
	head.DiffHints = diff.LeftLines
	current.DiffHints = diff.RightLines
	current.RealPath = "gutter.go"
	// Git line numbers refer to the file without alignment filler rows.
	current.GitChanges = &proxy.FileChanges{Added: []int{3}, Deleted: []int{2}}
	te.buffers[head.FilePath] = head
	te.buffers[current.FilePath] = current
	te.SetupDiffView(head, current, "gutter.go")
	te.render()

	const fillerRow = 2 // Display row 3 overlaps the added file line number.
	if !diff.RightLines[fillerRow].IsFillerLine || diff.RightLines[fillerRow].RightLine != -1 {
		t.Fatal("expected a blank alignment row opposite deleted text")
	}
	win := te.windows[te.activeWindow]
	markerX := win.X + win.Width/2 + current.LineNumberWidth - 3
	foundAddition := false
	for row, line := range diff.RightLines {
		marker, _ := te.screen.(*MockScreen).GetCell(markerX, win.Y+row)
		if line.IsFillerLine && marker != ' ' {
			t.Errorf("blank alignment row %d has gutter marker %q, want space", row+1, marker)
		}
		if !line.IsFillerLine && line.RightLine == 2 {
			foundAddition = true
			if marker != '+' {
				t.Errorf("added file line 3 has gutter marker %q, want '+'", marker)
			}
		}
	}
	if !foundAddition {
		t.Fatal("expected the added file line to be rendered")
	}
}
