package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenProjectChangesStartsWithoutFileList(t *testing.T) {
	repo := NewGitTestRepo(t)
	runGit(t, repo, "init", ".")

	if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("new\n"), 0644); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	te := NewTestEditor(80, 24)
	t.Cleanup(te.Close)
	te.OpenProjectChanges(repo)
	te.handlePendingPath()
	te.WaitForAsyncOperations()

	if te.workingDirectory != repo {
		t.Fatalf("working directory = %q, want %q", te.workingDirectory, repo)
	}
	if te.renderingState.IsOverlayActive() {
		t.Fatal("file list overlay opened")
	}
	if te.CurrentBuffer() == nil || te.CurrentBuffer().FilePath != "*Project Changes*" {
		t.Fatalf("current buffer = %v, want *Project Changes*", te.CurrentBuffer())
	}
}
