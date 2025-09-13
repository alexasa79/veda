package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDeletedLinesGutter verifies that when lines are deleted from a file
// there is a visible indicator in the left gutter at the deletion location.
// The test mirrors the structure of the staging gutter test.
func TestDeletedLinesGutter(t *testing.T) {
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

	// Base file content (12 lines to keep hunks separate and ensure indices remain valid)
	base := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12\n"
	path := filepath.Join(repo, "file.txt")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", filepath.Base(path))
	run("git", "commit", "-m", "initial")

	// Delete two lines far apart (3 and 8)
	modified := "line1\nline2\nline4\nline5\nline6\nline7\nline9\nline10\nline11\nline12\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	// Open editor with modified content
	te := NewTestEditorWithContent(100, 25, modified)
	// Re-key the buffer under the real path so reload-file and proxy lookups work
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	// Load git changes via proxy and attach to buffer
	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	// Render to draw gutter
	te.render()

	// Helper to read the left gutter indicator rune for a given 1-based line
	readLeftGutterRune := func(oneBasedLine int) rune {
		lnw := te.CurrentBuffer().LineNumberWidth
		digits := lnw - 3
		if digits < 1 {
			digits = 1
		}
		leftX := digits
		contentStartY := len(te.CurrentBuffer().ContextCache)
		viewTop := te.CurrentBuffer().ViewIterator.LineNumber()
		screenY := contentStartY + (oneBasedLine - 1 - viewTop)
		r, _ := te.mockScreen.GetCell(leftX, screenY)
		return r
	}

	// We removed old lines 3 and 8. Expect a deletion indicator glyph in the
	// left gutter at the corresponding positions in the current buffer. The
	// exact glyph is a minus-like marker (e.g., '−').
	if r := readLeftGutterRune(3); r == ' ' || r == 0 {
		t.Fatalf("expected deletion indicator at line 3; got %q", r)
	}
	if r := readLeftGutterRune(8); r == ' ' || r == 0 {
		t.Fatalf("expected deletion indicator at line 8; got %q", r)
	}
}
