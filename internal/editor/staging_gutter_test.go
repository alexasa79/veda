package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestLineStagingGutter verifies staging specific lines updates git and gutter indicators,
// and that re-editing a staged line shows both changed and staged markers.
func TestLineStagingGutter(t *testing.T) {
	repo := t.TempDir()
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Base file content and initial commit (9 lines to separate hunks >3 lines apart)
	base := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\n"
	path := filepath.Join(repo, "file.txt")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", "file.txt")
	run("git", "commit", "-m", "initial")

	// Modify two lines far apart (2 and 8)
	modified := "line1\nline2X\nline3\nline4\nline5\nline6\nline7\nline8X\nline9\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	// Open editor with modified content
	te := NewTestEditorWithContent(100, 25, modified)
	// Re-key the buffer in editor maps under the real path so reload-file can find it
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	// Load git changes via proxy directly (setup step)
	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	// Stage lines 2 and 8 using keybinding "g space"
	// Move to line 2
	te.TypeKeys("down") // now on line 2 (1-based)
	te.TypeKeys("g space")
	// Move to line 8 (from 2 -> need 6 downs)
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("g space")

	// Verify staged sets include 2 and 8
	changes := te.CurrentBuffer().GitChanges
	if changes == nil {
		t.Fatalf("GitChanges is nil")
	}
	has := func(arr []int, v int) bool {
		for _, x := range arr {
			if x == v {
				return true
			}
		}
		return false
	}
	if !(has(changes.StagedModified, 2) || has(changes.StagedAdded, 2)) {
		t.Fatalf("expected line 2 to be staged; got %+v", changes)
	}
	if !(has(changes.StagedModified, 8) || has(changes.StagedAdded, 8)) {
		t.Fatalf("expected line 8 to be staged; got %+v", changes)
	}
	if has(changes.StagedModified, 3) || has(changes.StagedAdded, 3) {
		t.Fatalf("did not expect line 3 staged; got %+v", changes)
	}

	// Render to draw gutter
	te.render()
	// Helper to read gutter staged marker (rightmost indicator column)
	readGutterRune := func(oneBasedLine int) rune {
		lnw := te.CurrentBuffer().LineNumberWidth
		digits := lnw - 3
		if digits < 1 {
			digits = 1
		}
		x := digits + 2 // rightmost indicator column for staged
		// Compute screen Y
		contentStartY := len(te.CurrentBuffer().ContextCache)
		viewTop := te.CurrentBuffer().ViewIterator.LineNumber()
		screenY := contentStartY + (oneBasedLine - 1 - viewTop)
		r, _ := te.mockScreen.GetCell(x, screenY)
		return r
	}
	// Right column shows '✓' for staged lines 2 and 4; not for 3
	if r := readGutterRune(2); r != '✓' {
		t.Fatalf("expected staged marker '✓' at line 2, got %q", r)
	}
	if r := readGutterRune(8); r != '✓' {
		t.Fatalf("expected staged marker '✓' at line 8, got %q", r)
	}
	if r := readGutterRune(3); r == '✓' {
		t.Fatalf("did not expect staged marker at line 3")
	}

	// Now modify a staged line again (line 2): enter insert, append 'Y', exit
	// Move cursor back to line 2
	te.TypeKeys("up")
	te.TypeKeys("up")
	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatal(err)
	}
	_ = te.InsertChar('Y')
	if err := te.ExecuteCommand("exit-insert-mode"); err != nil {
		t.Fatal(err)
	}

	// Reload git changes to reflect new unstaged diff on top of staged
	if err := te.ExecuteCommand("reload-file"); err != nil {
		t.Fatalf("reload-file after edit: %v", err)
	}
	te.render()

	// Expect both an unstaged change marker in left column and a staged marker in right column for line 2
	// Left column index: digits
	lnw := te.CurrentBuffer().LineNumberWidth
	digits := lnw - 3
	if digits < 1 {
		digits = 1
	}
	leftX := digits // leftmost indicator
	rightX := digits + 2
	contentStartY := len(te.CurrentBuffer().ContextCache)
	viewTop := te.CurrentBuffer().ViewIterator.LineNumber()
	y := contentStartY + (2 - 1 - viewTop)
	lch, _ := te.mockScreen.GetCell(leftX, y)
	rch, _ := te.mockScreen.GetCell(rightX, y)
	if lch == 0 || rch != '✓' {
		t.Fatalf("expected both changed (left) and staged (right) markers on line 2; got left=%q right=%q", lch, rch)
	}
}
