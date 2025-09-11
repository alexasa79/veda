package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"veda/internal/buffer"
)

// TestProjectChangesGnNavigation creates a small git repo with changes across files,
// opens the project changes buffer (s C), then uses "g n" to jump through each change.
// It verifies we visit actual code locations in the right files/lines, not just the list.
func TestProjectChangesGnNavigation(t *testing.T) {
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

	// Initialize repo and an initial commit with two files
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// file1 with 100 lines
	var f1Lines []byte
	for i := 1; i <= 100; i++ {
		f1Lines = append(f1Lines, []byte(fmt.Sprintf("A %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "file1.txt"), f1Lines, 0644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	// file2 with 120 lines (ensure large file)
	var f2Lines []byte
	for i := 1; i <= 120; i++ {
		f2Lines = append(f2Lines, []byte(fmt.Sprintf("B %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "file2.txt"), f2Lines, 0644); err != nil {
		t.Fatalf("write file2: %v", err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	// Modify file1 at two far-apart lines (creates two hunks)
	// Change line 10 and 90
	var f1Mod []byte
	for i := 1; i <= 100; i++ {
		if i == 10 {
			f1Mod = append(f1Mod, []byte("A ten\n")...)
		} else if i == 90 {
			f1Mod = append(f1Mod, []byte("A ninety\n")...)
		} else {
			f1Mod = append(f1Mod, []byte(fmt.Sprintf("A %d\n", i))...)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "file1.txt"), f1Mod, 0644); err != nil {
		t.Fatalf("modify file1: %v", err)
	}

	// Insert a new line into file2 around line 60 (pure insertion hunk)
	var f2Mod []byte
	for i := 1; i <= 59; i++ {
		f2Mod = append(f2Mod, []byte(fmt.Sprintf("B %d\n", i))...)
	}
	f2Mod = append(f2Mod, []byte("B INSERT\n")...)
	for i := 60; i <= 120; i++ {
		f2Mod = append(f2Mod, []byte(fmt.Sprintf("B %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "file2.txt"), f2Mod, 0644); err != nil {
		t.Fatalf("modify file2: %v", err)
	}

	// Start editor, set working directory to the repo, and open Project Changes via key binding "s C"
	te := NewTestEditor(120, 30)
	te.Editor.workingDirectory = repo

	// Open the project changes buffer
	te.TypeKeys("s C")

	// Ensure the project changes buffer exists
	pcBuf, ok := te.buffers["*Project Changes*"]
	if !ok || pcBuf == nil {
		t.Fatalf("Project Changes buffer not opened")
	}
	if te.CurrentBuffer() != pcBuf {
		t.Fatalf("expected to be in *Project Changes* buffer after s C")
	}

	// Collect the expected (path, line) targets by scanning the buffer content for hunk links
	type target struct {
		path string
		line int
	}
	var targets []target
	for i := 0; i < pcBuf.LineCount; i++ {
		it := buffer.NewBufferIteratorAtLine(pcBuf, i)
		line := it.Line()
		if p, ln, ok := parseProjectChangeLink(line); ok {
			// Resolve to absolute for later GetFile calls
			abs := filepath.Join(repo, p)
			targets = append(targets, target{path: abs, line: ln})
		}
	}
	if len(targets) == 0 {
		t.Fatalf("no hunk links found in project changes buffer")
	}

	// Visit each target with "g n" and verify we open the right file at the first actual changed line
	for idx, tg := range targets {
		te.TypeKeys("g n")
		// After navigation, current buffer should be the opened file
		buf := te.CurrentBuffer()
		if buf == nil || buf.IsInternal || buf.FilePath == "*Project Changes*" {
			t.Fatalf("after g n #%d, expected to be in a file buffer, got %v", idx, func() string {
				if buf == nil {
					return "<nil>"
				}
				return buf.FilePath
			}())
		}
		expectedPath := tg.path
		if buf.FilePath != expectedPath {
			t.Fatalf("after g n #%d, expected path %s, got %s", idx, expectedPath, buf.FilePath)
		}
		// Cursor should be at the first actual changed line at or after the hunk start
		fc, err := te.Editor.proxyManager.GetFile(expectedPath)
		if err != nil {
			t.Fatalf("GetFile for %s: %v", expectedPath, err)
		}
		// Find minimal changed line >= tg.line
		minLine := 0
		push := func(lines []int) {
			for _, v := range lines {
				if v >= tg.line && (minLine == 0 || v < minLine) {
					minLine = v
				}
			}
		}
		push(fc.Changes.Added)
		push(fc.Changes.Modified)
		if minLine == 0 {
			t.Fatalf("no changed line found at or after %d in %s", tg.line, expectedPath)
		}
		_, y := te.GetCursorPosition()
		if y != minLine-1 {
			t.Fatalf("after g n #%d, expected line %d, got %d (hunk start was %d)", idx, minLine-1, y, tg.line-1)
		}
		// Also assert that the project changes buffer linked to the first actual changed line
		if tg.line != minLine {
			t.Fatalf("project changes link expected to anchor at %d but it anchors at %d", minLine, tg.line)
		}
		// lastActiveSpecialBuffer should remain pointing to Project Changes for contextual navigation
		if te.Editor.lastActiveSpecialBuffer != "*Project Changes*" {
			t.Fatalf("lastActiveSpecialBuffer lost; expected *Project Changes*, got %s", te.Editor.lastActiveSpecialBuffer)
		}
	}
}
