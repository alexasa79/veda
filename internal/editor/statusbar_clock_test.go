package editor

import (
	"regexp"
	"testing"
)

func TestStatusBarClockFitsWithTrailingPadding(t *testing.T) {
	te := NewTestEditorWithContent(100, 16, "hello\n")
	te.drawStatusBar()
	width, _ := te.mockScreen.Size()
	row := make([]rune, width)
	for x := range row {
		row[x], _ = te.mockScreen.GetCell(x, te.getStatusBarY())
	}
	status := string(row)
	if !regexp.MustCompile(`\[\d+\]  \d{2}:\d{2} $`).MatchString(status) {
		t.Fatalf("clock must have two separator spaces and one trailing space: %q", status)
	}
}
