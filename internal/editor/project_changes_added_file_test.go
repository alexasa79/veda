package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectChangesListsUntrackedAddedFile verifies that the "*Project Changes*" buffer
// lists files that were newly created (untracked) in the project.
func TestProjectChangesListsUntrackedAddedFile(t *testing.T) {
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

	// Initialize repository with a base commit
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Create and commit an initial file
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", "base.txt")
	run("git", "commit", "-m", "initial")

	// Create a new (untracked) file with sufficient size (100 lines)
	var content []byte
	for i := 1; i <= 100; i++ {
		content = append(content, []byte(fmt.Sprintf("N %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "added.txt"), content, 0644); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	// Start editor and open the Project Changes buffer via key binding "s C"
	te := NewTestEditor(120, 30)
	te.Editor.workingDirectory = repo
	te.TypeKeys("s C")

	// Locate the *Project Changes* buffer and check for the new file
	pc, ok := te.buffers["*Project Changes*"]
	if !ok || pc == nil {
		t.Fatalf("Project Changes buffer not opened")
	}
	bufText := pc.Rope.String()
	if !strings.Contains(bufText, "](added.txt)") {
		t.Fatalf("expected *Project Changes* to list added.txt, but it did not. Buffer was:\n%s", bufText)
	}
}
