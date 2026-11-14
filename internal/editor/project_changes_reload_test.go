package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"veda/internal/events"
)

func TestProjectChangesReloadWithCtrlR(t *testing.T) {
	repo := NewGitTestRepo(t)
	runGit(t, repo, "init", ".")
	runGit(t, repo, "commit", "--allow-empty", "-m", "initial")

	te := NewTestEditor(100, 24)
	t.Cleanup(te.Close)
	te.workingDirectory = repo
	te.TypeKeys("s C")
	te.WaitForAsyncOperations()
	buf := te.CurrentBuffer()
	if buf.FilePath != "*Project Changes*" {
		t.Fatalf("current buffer = %q, want *Project Changes*", buf.FilePath)
	}
	if !strings.Contains(buf.Rope.String(), "No changes found.") {
		t.Fatalf("expected no changes, got:\n%s", buf.Rope.String())
	}

	path := filepath.Join(repo, "new.txt")
	if err := os.WriteFile(path, []byte("new\n"), 0644); err != nil {
		t.Fatal(err)
	}
	te.PressKey(events.KeyCtrlR, 0)
	te.WaitForAsyncOperations()
	if !strings.Contains(buf.Rope.String(), "[new.txt](") {
		t.Fatalf("refresh did not list new file:\n%s", buf.Rope.String())
	}
	if want := strings.Count(buf.Rope.String(), "\n") + 1; buf.LineCount != want {
		t.Fatalf("line count = %d, want %d", buf.LineCount, want)
	}
	te.TypeKeys("G")

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	te.PressKey(events.KeyCtrlR, 0)
	te.WaitForAsyncOperations()
	if !strings.Contains(buf.Rope.String(), "No changes found.") || strings.Contains(buf.Rope.String(), "new.txt") {
		t.Fatalf("refresh did not remove old entry:\n%s", buf.Rope.String())
	}
	if te.CurrentBuffer() != buf || !buf.IsReadOnly || !buf.IsInternal || buf.HasFile() || buf.IsModified {
		t.Fatal("refresh changed the active buffer or its internal read-only state")
	}
	if want := strings.Count(buf.Rope.String(), "\n") + 1; buf.LineCount != want {
		t.Fatalf("line count = %d, want %d", buf.LineCount, want)
	}
	if msg := te.errorManager.GetCurrentError(); msg != "" {
		t.Fatalf("refresh showed an error: %s", msg)
	}
	te.TypeKeys("j")
}
