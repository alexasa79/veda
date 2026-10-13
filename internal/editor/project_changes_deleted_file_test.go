package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"veda/internal/buffer"
)

// TestProjectChangesListsAndSkipsDeletedFile ensures deleted files are listed
// in the project changes buffer but navigation (g n) skips them and jumps to
// the next available change.
func TestProjectChangesListsAndSkipsDeletedFile(t *testing.T) {
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

	// Initialize repo
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Create two files, commit
	var f1 []byte
	for i := 1; i <= 120; i++ {
		f1 = append(f1, []byte(fmt.Sprintf("X %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "deleted.txt"), f1, 0644); err != nil {
		t.Fatalf("write deleted: %v", err)
	}
	var f2 []byte
	for i := 1; i <= 120; i++ {
		f2 = append(f2, []byte(fmt.Sprintf("Y %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "kept.txt"), f2, 0644); err != nil {
		t.Fatalf("write kept: %v", err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	// Modify kept.txt and delete deleted.txt (staged deletion)
	var f2mod []byte
	for i := 1; i <= 50; i++ {
		f2mod = append(f2mod, []byte(fmt.Sprintf("Y %d\n", i))...)
	}
	f2mod = append(f2mod, []byte("Y INSERT\n")...)
	for i := 50; i <= 120; i++ {
		f2mod = append(f2mod, []byte(fmt.Sprintf("Y %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "kept.txt"), f2mod, 0644); err != nil {
		t.Fatalf("modify kept: %v", err)
	}
	run("git", "rm", "deleted.txt")

	// Open project changes
	te := NewTestEditor(120, 30)
	te.Editor.workingDirectory = repo
	te.TypeKeys("s C")
	te.WaitForAsyncOperations()

	pc, ok := te.buffers["*Project Changes*"]
	if !ok || pc == nil {
		t.Fatalf("Project Changes buffer not opened")
	}
	bufText := pc.Rope.String()
	deletedPath := canonicalTestPath(filepath.Join(repo, "deleted.txt"))
	keptPath := canonicalTestPath(filepath.Join(repo, "kept.txt"))

	if !strings.Contains(bufText, "[deleted.txt](") || !strings.Contains(bufText, "[D●]") {
		t.Fatalf("expected staged deletion for deleted.txt, got:\n%s", bufText)
	}
	if !strings.Contains(bufText, fmt.Sprintf("](%s)", keptPath)) {
		t.Fatalf("expected kept.txt to be listed, got:\n%s", bufText)
	}

	// Move selection to the deleted file's link line
	delLine := -1
	for i := 0; i < pc.LineCount; i++ {
		it := buffer.NewBufferIteratorAtLine(pc, i)
		line := it.Line()
		if strings.Contains(line, fmt.Sprintf("](%s)", deletedPath)) {
			delLine = i
			break
		}
	}
	if delLine < 0 {
		t.Fatalf("could not locate deleted.txt link line")
	}
	// :LINE jump (1-based)
	te.TypeKeys(":" + fmt.Sprint(delLine+1) + "\n")

	// Press g n: should skip deleted file and open kept.txt at its change
	te.TypeKeys("g n")
	if te.CurrentBuffer() == pc || te.CurrentBuffer() == nil {
		t.Fatalf("expected to open kept.txt after skipping deletion; still at project changes")
	}
	if filepath.Base(te.CurrentBuffer().FilePath) != "kept.txt" {
		t.Fatalf("expected to open kept.txt, got %s", te.CurrentBuffer().FilePath)
	}
}

// TestProjectChangesEnterOnDeletedFileShowsNotification ensures that pressing Enter
// on a deleted file link in the Project Changes buffer shows a notification instead
// of trying to open the file.
func TestProjectChangesEnterOnDeletedFileShowsNotification(t *testing.T) {
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

	// Initialize repo
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Create a file and commit
	var f1 []byte
	for i := 1; i <= 50; i++ {
		f1 = append(f1, []byte(fmt.Sprintf("Line %d\n", i))...)
	}
	if err := os.WriteFile(filepath.Join(repo, "deleted.txt"), f1, 0644); err != nil {
		t.Fatalf("write deleted: %v", err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	// Delete the file (staged deletion)
	run("git", "rm", "deleted.txt")

	// Open project changes
	te := NewTestEditor(120, 30)
	te.Editor.workingDirectory = repo
	te.TypeKeys("s C")
	te.WaitForAsyncOperations()

	pc, ok := te.buffers["*Project Changes*"]
	if !ok || pc == nil {
		t.Fatalf("Project Changes buffer not opened")
	}
	bufText := pc.Rope.String()
	deletedPath := canonicalTestPath(filepath.Join(repo, "deleted.txt"))

	if !strings.Contains(bufText, "[deleted.txt](") || !strings.Contains(bufText, "[D●]") {
		t.Fatalf("expected staged deletion for deleted.txt, got:\n%s", bufText)
	}

	// Find the line with the deleted file link
	delLine := -1
	for i := 0; i < pc.LineCount; i++ {
		it := buffer.NewBufferIteratorAtLine(pc, i)
		line := it.Line()
		if strings.Contains(line, fmt.Sprintf("](%s)", deletedPath)) {
			delLine = i
			break
		}
	}
	if delLine < 0 {
		t.Fatalf("could not locate deleted.txt link line")
	}

	// Select the File link on the same line as the Diff link.
	te.TypeKeys(":" + fmt.Sprint(delLine+1) + "\n")
	line := buffer.NewBufferIteratorAtLine(pc, delLine).Line()
	pc.CursorX = strings.Index(line, "[File]") + 1

	// Clear any existing messages
	te.Editor.errorManager.ClearError()

	// Press Enter to try to open the deleted file
	te.TypeKeys("\n")
	te.WaitForAsyncOperations()

	// The buffer should still be *Project Changes* - we shouldn't have opened the deleted file
	if te.CurrentBuffer().DisplayName() != "*Project Changes*" {
		t.Fatalf("expected to stay on *Project Changes*, but switched to %s", te.CurrentBuffer().DisplayName())
	}

	// There should be a notification about the file being deleted
	msg, _ := te.Editor.errorManager.GetCurrentBanner()
	if !strings.Contains(msg, "deleted") && !strings.Contains(msg, "does not exist") {
		t.Fatalf("expected notification about deleted file, got: %q", msg)
	}
}
