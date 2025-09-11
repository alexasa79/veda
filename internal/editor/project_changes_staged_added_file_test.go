package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectChangesListsStagedAddedFile verifies that newly added and staged files
// appear in the Project Changes buffer and are marked as staged (A●).
func TestProjectChangesListsStagedAddedFile(t *testing.T) {
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

	// Initialize repository and base commit
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", "base.txt")
	run("git", "commit", "-m", "initial")

	// Create a new file and stage it
	var content []byte
	for i := 1; i <= 120; i++ {
		content = append(content, []byte(fmt.Sprintf("S %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "added.txt"), content, 0644); err != nil {
		t.Fatalf("write new file: %v", err)
	}
	run("git", "add", "added.txt")

	// Open Project Changes buffer
	te := NewTestEditor(120, 30)
	te.Editor.workingDirectory = repo
	te.TypeKeys("s C")

	pc, ok := te.buffers["*Project Changes*"]
	if !ok || pc == nil {
		t.Fatalf("Project Changes buffer not opened")
	}
	bufText := pc.Rope.String()

	// Expect the added file to be listed and flagged as staged (A●)
	if !strings.Contains(bufText, "]("+"added.txt"+")") {
		t.Fatalf("expected Project Changes to list added.txt, but got:\n%s", bufText)
	}
	if !strings.Contains(bufText, "[A●] [added.txt]") {
		t.Fatalf("expected Project Changes to mark added.txt as staged (A●), but got:\n%s", bufText)
	}
}
