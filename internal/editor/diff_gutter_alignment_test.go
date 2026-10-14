package editor

import (
	"os"
	"path/filepath"
	"testing"

	"veda/internal/buffer"
)

func TestDiffGutterContextHasNoChangeMarker(t *testing.T) {
	repo, err := os.MkdirTemp("../../artifacts", "diff-gutter-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(repo) })
	repo, err = filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", ".")
	runGit(t, repo, "config", "core.hooksPath", "/dev/null")
	path := filepath.Join(repo, "sample.cpp")
	before := "void store() {\n\twrite();\n}\n\nvoid next() {\n\tnext_entry();\n}\n"
	after := "void store() {\n\twrite();\n}\n\nvoid register_entries() {\n#ifdef FEATURE\n\tnew_entry();\n#endif\n}\n\nvoid next() {\n\tnext_entry();\n}\n"
	if err := os.WriteFile(path, []byte(before), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "sample.cpp")
	runGit(t, repo, "commit", "-m", "initial")
	if err := os.WriteFile(path, []byte(after), 0644); err != nil {
		t.Fatal(err)
	}
	te := NewTestEditorWithContent(120, 24, after)
	fc, err := te.proxyManager.GetFile(path)
	if err != nil {
		t.Fatal(err)
	}
	buf := te.CurrentBuffer()
	buf.FilePath = path
	buf.RealPath = path
	buf.GitChanges = &fc.Changes
	te.buffers[path] = buf
	if err := te.StartDiffViewForCurrentBuffer(); err != nil {
		t.Fatal(err)
	}
	te.render()
	view := te.windows[te.activeWindow].View.(*DiffBufferView)
	current := view.RightBuffer
	markerX := te.windows[te.activeWindow].Width/2 + current.LineNumberWidth - 3
	checkMarkers := func() {
		t.Helper()
		for row, line := range current.DiffHints {
			marker, _ := te.mockScreen.GetCell(markerX, row)
			want := ' '
			if line.LineType == buffer.DiffLineAdded && !line.IsFillerLine {
				want = '+'
			}
			if marker != want {
				t.Errorf("file line %d (%q) has marker %q, want %q", line.RightLine+1, line.Content, marker, want)
			}
		}
	}
	checkMarkers()

	// Stage an added line through the diff view and keep its staging tick.
	const addedRow = 6
	te.SetCursorPosition(0, addedRow)
	view.SyncCursors()
	if err := te.ExecuteCommand("toggle-stage-line"); err != nil {
		t.Fatal(err)
	}
	te.render()
	checkMarkers()
	staged, _ := te.mockScreen.GetCell(markerX+2, addedRow)
	if staged != '✓' {
		t.Errorf("staged added line has marker %q, want '✓'", staged)
	}
	if err := te.ExecuteCommand("toggle-stage-line"); err != nil {
		t.Fatal(err)
	}
	te.render()
	checkMarkers()
	staged, _ = te.mockScreen.GetCell(markerX+2, addedRow)
	if staged != ' ' {
		t.Errorf("unstaged added line has marker %q, want space", staged)
	}
}
