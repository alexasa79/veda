package editor

import (
	"os"
	"path/filepath"
	"testing"
	"veda/internal/buffer"
)

func TestDiffToggleOpensWorkingFile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		head    string
		current string
		row     int
		line    int
	}{
		{"changed line", "first\nold\nlast", "first\nnew\nlast", 1, 1},
		{"after deletion", "first\ndeleted\nlast", "first\nlast", 2, 1},
		{"deleted line", "first\ndeleted\nlast", "first\nlast", 1, 1},
		{"deleted end", "first\ndeleted", "first", 1, 0},
		{"empty file", "deleted", "", 0, 0},
	} {
		for _, pane := range []int{0, 1} {
			t.Run(tc.name+string(rune('0'+pane)), func(t *testing.T) {
				path, err := filepath.Abs("../../artifacts/diff-toggle.txt")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tc.current), 0644); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Remove(path) })
				te := NewTestEditorWithContent(120, 30, "Project Changes")
				projectChanges := te.CurrentBuffer()
				diff := computeDiff(tc.head, tc.current)
				left := buffer.NewBuffer(createBufferContent(diff.LeftLines), path+"@HEAD", "HEAD", te.config.DefaultIndentType, 4)
				right := buffer.NewBuffer(createBufferContent(diff.RightLines), path+"@current", "current", te.config.DefaultIndentType, 4)
				left.RealPath, right.RealPath = path, path
				left.DiffHints, right.DiffHints = diff.LeftLines, diff.RightLines
				left.IsReadOnly = true
				te.buffers[left.FilePath], te.buffers[right.FilePath] = left, right
				view := te.SetupDiffView(left, right, path)
				view.ActivePane = pane
				te.switchToBuffer(view.ActiveBuffer())
				te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), tc.row)
				view.SyncCursors()
				te.updateCursorState()
				te.TypeKeys("sc")
				if te.CurrentBuffer().FilePath != path || te.CurrentBuffer().IsReadOnly {
					t.Fatal("s c did not open the editable working file")
				}
				AssertBufferEquals(t, te, tc.current)
				AssertCursorAt(t, te, 0, tc.line)
				if te.buffers[projectChanges.FilePath] != projectChanges {
					t.Fatal("project changes buffer was lost")
				}
			})
		}
	}
}
