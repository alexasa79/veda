package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/proxy"
)

func TestGnDiffChangeAfterDeletedLine(t *testing.T) {
	previous := "top\ndeleted\nanchor1\nanchor2\nanchor3\nold\ntail"
	current := "top\nanchor1\nanchor2\nanchor3\nnew\ntail"
	te := NewTestEditorWithContent(120, 30, current)
	original := te.CurrentBuffer()
	original.GitChanges = &proxy.FileChanges{Deleted: []int{2}, Modified: []int{5}}

	diff := computeDiff(previous, current)
	head := buffer.NewBuffer(createBufferContent(diff.LeftLines), "jump.txt@HEAD", "jump.txt@HEAD", config.IndentTabs, 4)
	head.IsReadOnly = true
	head.DiffHints = diff.LeftLines
	right := buffer.NewBuffer(createBufferContent(diff.RightLines), "jump.txt@current", "jump.txt@current", config.IndentTabs, 4)
	right.DiffHints = diff.RightLines
	// Production diff setup shares the original file's GitChanges with @current.
	right.GitChanges = original.GitChanges
	te.SetupDiffView(head, right, "jump.txt")

	// The deletion adds a filler row: file line 5 is now diff row 6.
	if !right.DiffHints[1].IsFillerLine || right.DiffHints[4].LineType != buffer.DiffLineContext ||
		right.DiffHints[5].LineType != buffer.DiffLineModified || right.DiffHints[5].RightLine != 4 {
		t.Fatal("expected a filler row before the modification at diff row 6")
	}
	te.goToLine(2) // Start after the deleted hunk.
	te.TypeKeys("g n")

	_, row := te.GetCursorPosition()
	if row != 5 {
		t.Fatalf("g n should land on changed diff row 6 (file line 5), got diff row %d with content %q", row+1, right.CursorIterator.Line())
	}
}

// TestGnFileGitChangesExact ensures that "g n" jumps exactly to the first line
// of each changed hunk in the current file (not a few lines before).
func TestGnFileGitChangesExact(t *testing.T) {
	repo := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	// Init repo
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Base file with 200 lines
	var base []byte
	for i := 1; i <= 200; i++ {
		base = append(base, []byte(fmt.Sprintf("L %d\n", i))...)
	}
	path := filepath.Join(repo, "jump.txt")
	if err := os.WriteFile(path, base, 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", "jump.txt")
	run("git", "commit", "-m", "initial")

	// Modify content to create two hunks:
	// - Replace line 50 with two lines (1->2 split), so first changed line is 50
	// - Insert single line before line 150, so first changed line is 150
	var mod []byte
	for i := 1; i <= 49; i++ {
		mod = append(mod, []byte(fmt.Sprintf("L %d\n", i))...)
	}
	mod = append(mod, []byte("L fifty-A\n")...)
	mod = append(mod, []byte("L fifty-B\n")...)
	for i := 51; i <= 149; i++ {
		mod = append(mod, []byte(fmt.Sprintf("L %d\n", i))...)
	}
	mod = append(mod, []byte("L INSERT\n")...)
	for i := 150; i <= 200; i++ {
		mod = append(mod, []byte(fmt.Sprintf("L %d\n", i))...)
	}
	if err := os.WriteFile(path, mod, 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	// Start editor with modified content and wire buffer
	te := NewTestEditorWithContent(120, 30, string(mod))
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	// Load git changes into buffer
	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	// Compute expected new-file line numbers by scanning modified content
	var wantFirst, wantSecond int // 0-based
	{
		lines := string(mod)
		idx := 0
		curLine := 0
		for idx < len(lines) {
			// find end of line
			j := idx
			for j < len(lines) && lines[j] != '\n' {
				j++
			}
			line := lines[idx:j]
			if wantFirst == 0 && line == "L fifty-A" {
				wantFirst = curLine
			}
			if wantSecond == 0 && line == "L INSERT" {
				wantSecond = curLine
			}
			// advance
			curLine++
			if j < len(lines) {
				j++
			}
			idx = j
		}
	}

	// Move to top and press g n -> should land exactly on first hunk line
	te.goToLine(0)
	te.TypeKeys("g n")
	_, y := te.GetCursorPosition()
	if y != wantFirst {
		t.Fatalf("after first g n, expected line %d, got %d", wantFirst, y)
	}

	// Another g n -> should land exactly on the inserted line
	te.TypeKeys("g n")
	_, y = te.GetCursorPosition()
	if y != wantSecond {
		t.Fatalf("after second g n, expected line %d, got %d", wantSecond, y)
	}
}
