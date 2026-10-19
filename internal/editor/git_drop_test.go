package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// helper to run git commands in a temp repo
func runGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}

func TestDropLine_SimpleModified(t *testing.T) {
	repo := NewGitTestRepo(t)
	runGit(t, repo, "init", ".")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")

	base := "a\nb\nc\nd\n"
	path := filepath.Join(repo, "f.txt")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	runGit(t, repo, "add", filepath.Base(path))
	runGit(t, repo, "commit", "-m", "init")

	// Modify line 3
	modified := "a\nb\ncX\nd\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	te := NewTestEditorWithContent(80, 24, modified)
	old := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, old)

	// Prime git changes
	fc, err := te.Editor.proxyManager.GetFile(path, true)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	// Move to line 3 and drop it
	te.TypeKeys(":3\n")
	if err := te.ExecuteCommand("drop-change-line"); err != nil {
		t.Fatalf("drop-change-line failed: %v", err)
	}

	// Refresh from disk and verify content reverts to base
	fc2, err := te.Editor.proxyManager.GetFile(path, true)
	if err != nil {
		t.Fatalf("GetFile after drop: %v", err)
	}
	if fc2.Content != base {
		t.Fatalf("expected file to match HEAD after drop line.\nGot:\n%q\nWant:\n%q", fc2.Content, base)
	}
}

func TestDropHunk_ContiguousBlock(t *testing.T) {
	repo := NewGitTestRepo(t)
	runGit(t, repo, "init", ".")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")

	// Base with 9 lines
	var baseLines []string
	for i := 1; i <= 9; i++ {
		baseLines = append(baseLines, fmt.Sprintf("L%d", i))
	}
	base := strings.Join(baseLines, "\n") + "\n"
	path := filepath.Join(repo, "h.txt")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	runGit(t, repo, "add", filepath.Base(path))
	runGit(t, repo, "commit", "-m", "init")

	// Modify a contiguous block lines 3..5
	modLines := append([]string{}, baseLines...)
	modLines[2] = "L3X"
	modLines[3] = "L4X"
	modLines[4] = "L5X"
	modified := strings.Join(modLines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	te := NewTestEditorWithContent(100, 30, modified)
	old := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, old)

	fc, err := te.Editor.proxyManager.GetFile(path, true)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	// Move into middle of hunk and drop hunk
	te.TypeKeys(":4\n")
	if err := te.ExecuteCommand("drop-change-hunk"); err != nil {
		t.Fatalf("drop-change-hunk failed: %v", err)
	}

	fc2, err := te.Editor.proxyManager.GetFile(path, true)
	if err != nil {
		t.Fatalf("GetFile after drop: %v", err)
	}
	if fc2.Content != base {
		t.Fatalf("expected block reverted to base.\nGot:\n%q\nWant:\n%q", fc2.Content, base)
	}
}

func TestDiffAutoRefreshAfterDrop(t *testing.T) {
	repo := NewGitTestRepo(t)
	runGit(t, repo, "init", ".")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")

	base := "x1\nx2\nx3\n"
	path := filepath.Join(repo, "d.txt")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	runGit(t, repo, "add", filepath.Base(path))
	runGit(t, repo, "commit", "-m", "init")

	modified := "x1\nx2X\nx3\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	te := NewTestEditorWithContent(100, 30, modified)
	old := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, old)

	// Open diff view
	if err := te.ExecuteCommand("diff-current-buffer"); err != nil {
		t.Fatalf("diff-current-buffer failed: %v", err)
	}

	// Right (@current) pane is active; drop the modified line via command
	te.TypeKeys(":2\n")
	if err := te.ExecuteCommand("drop-change-line"); err != nil {
		t.Fatalf("drop-change-line in diff failed: %v", err)
	}

	// Simulate file-reloaded event payload to trigger in-place diff refresh
	fc2, err := te.Editor.proxyManager.GetFile(path, true)
	if err != nil {
		t.Fatalf("GetFile after drop: %v", err)
	}
	te.Editor.handleFileReloaded(map[string]interface{}{
		"path":        path,
		"content":     fc2.Content,
		"changes":     fc2.Changes,
		"branch":      fc2.Branch,
		"diagnostics": fc2.Diagnostics,
	})

	// Verify right pane now shows base content (aligned)
	curName := path + "@current"
	buf := te.buffers[curName]
	if buf == nil {
		t.Fatalf("right diff buffer not found")
	}
	if got := buf.Rope.String(); !strings.Contains(got, "x2\n") {
		t.Fatalf("right pane not refreshed; expected to contain reverted line 'x2'\nGot:\n%s", got)
	}
}
