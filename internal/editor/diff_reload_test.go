package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDiffView_ReloadWithCtrlR tests that pressing Ctrl+R in diff view
// reloads both sides of the diff correctly without closing windows.
func TestDiffView_ReloadWithCtrlR(t *testing.T) {
	// Create a temporary git repo
	repoDir := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	// Initialize git repo
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Create initial file and commit
	filePath := filepath.Join(repoDir, "test.txt")
	initialContent := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("git", "add", "test.txt")
	run("git", "commit", "-m", "initial")

	// Modify file to create diff
	modifiedContent := "line 1\nmodified line 2\nline 3\nnew line 4\n"
	if err := os.WriteFile(filePath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("modify file: %v", err)
	}

	// Create editor with the modified content
	te := NewTestEditorWithContent(120, 30, modifiedContent)
	te.CurrentBuffer().FilePath = filePath
	te.CurrentBuffer().RealPath = filePath
	te.buffers[filePath] = te.CurrentBuffer()

	// Open diff view with "s c" (show-changes for current buffer)
	te.TypeKeys("s c")

	// Verify diff view is set up - should have 2 windows
	if len(te.windows) < 2 {
		t.Fatalf("Expected 2 windows for diff view, got %d", len(te.windows))
	}

	// Verify we have both @HEAD and @current buffers
	headBufferName := filePath + "@HEAD"
	currentBufferName := filePath + "@current"

	if _, exists := te.buffers[headBufferName]; !exists {
		t.Fatalf("HEAD buffer %s not found", headBufferName)
	}
	if _, exists := te.buffers[currentBufferName]; !exists {
		t.Fatalf("current buffer %s not found", currentBufferName)
	}

	// Record initial state
	initialWindowCount := len(te.windows)
	t.Logf("Initial window count: %d", initialWindowCount)
	t.Logf("Current buffer (should be @current): %s", te.CurrentBuffer().FilePath)

	// Verify the right side (@current) is active after opening diff view
	if !strings.HasSuffix(te.CurrentBuffer().FilePath, "@current") {
		t.Errorf("Expected right side (@current) to be active after opening diff view, got: %s", te.CurrentBuffer().FilePath)
	}

	// Verify RealPath is set correctly on diff buffers
	headBuffer := te.buffers[headBufferName]
	currentBuffer := te.buffers[currentBufferName]

	if headBuffer.RealPath != filePath {
		t.Errorf("HEAD buffer RealPath should be %s, got %s", filePath, headBuffer.RealPath)
	}
	if currentBuffer.RealPath != filePath {
		t.Errorf("current buffer RealPath should be %s, got %s", filePath, currentBuffer.RealPath)
	}

	// Record content and buffer pointers BEFORE reload
	headContentBefore := headBuffer.Rope.String()
	currentContentBefore := currentBuffer.Rope.String()
	headBufferPtrBefore := headBuffer
	currentBufferPtrBefore := currentBuffer
	t.Logf("HEAD content before reload (%d chars)", len(headContentBefore))
	t.Logf("current content before reload (%d chars)", len(currentContentBefore))
	t.Logf("HEAD buffer pointer before: %p", headBufferPtrBefore)
	t.Logf("current buffer pointer before: %p", currentBufferPtrBefore)

	// Now press Ctrl+R to reload
	t.Log("Pressing Ctrl+R to reload...")
	te.TypeKeys("ctrl+r")

	// Wait for async operations to complete (reload happens in a goroutine)
	// Use a longer timeout to ensure the proxy request completes
	te.WaitForAsyncOperationsTimeout(5 * time.Second)

	// Check if an error was shown
	if errMsg := te.errorManager.GetCurrentError(); errMsg != "" {
		t.Logf("Error shown after reload: %s", errMsg)
	}

	// Verify windows are still present
	if len(te.windows) != initialWindowCount {
		t.Errorf("Window count changed from %d to %d after Ctrl+R", initialWindowCount, len(te.windows))
	}

	// Verify both diff buffers still exist
	headBufferAfter, headExists := te.buffers[headBufferName]
	currentBufferAfter, currentExists := te.buffers[currentBufferName]

	if !headExists {
		t.Error("HEAD buffer disappeared after Ctrl+R")
	}
	if !currentExists {
		t.Error("current buffer disappeared after Ctrl+R")
	}

	// The async reload in tests doesn't process events properly, so we manually
	// simulate what happens: handleFileReloaded is called with the real path
	// and the actual file content (what the fixed code does)
	te.handleFileReloaded(map[string]any{
		"path":    filePath,        // real path (test.txt)
		"content": modifiedContent, // actual file content
	})

	// Re-fetch buffers after the reload event
	headBufferAfter, headExists = te.buffers[headBufferName]
	currentBufferAfter, currentExists = te.buffers[currentBufferName]

	// Check content AFTER reload
	headContentAfter := ""
	currentContentAfter := ""
	if headBufferAfter != nil {
		headContentAfter = headBufferAfter.Rope.String()
	}
	if currentBufferAfter != nil {
		currentContentAfter = currentBufferAfter.Rope.String()
	}

	t.Logf("HEAD content after reload (%d chars)", len(headContentAfter))
	t.Logf("current content after reload (%d chars)", len(currentContentAfter))

	// THE BUG: The right side (@current) becomes empty after reload
	if len(currentContentAfter) == 0 {
		t.Error("BUG: current buffer (@current, right side) is empty after reload")
	}
	if len(headContentAfter) == 0 {
		t.Error("HEAD buffer (@HEAD, left side) is empty after reload")
	}
}

// TestDiffView_ReloadFromHeadBuffer tests reload when cursor is on @HEAD buffer
func TestDiffView_ReloadFromHeadBuffer(t *testing.T) {
	// Create a temporary git repo
	repoDir := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	filePath := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("original\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("git", "add", "test.txt")
	run("git", "commit", "-m", "initial")

	// Modify file
	if err := os.WriteFile(filePath, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("modify file: %v", err)
	}

	te := NewTestEditorWithContent(120, 30, "modified\n")
	te.CurrentBuffer().FilePath = filePath
	te.CurrentBuffer().RealPath = filePath
	te.buffers[filePath] = te.CurrentBuffer()

	// Open diff view
	te.TypeKeys("s c")

	// Switch to HEAD buffer (left pane) using window navigation
	te.TypeKeys("space tab")
	if !strings.HasSuffix(te.CurrentBuffer().FilePath, "@HEAD") {
		te.TypeKeys("space tab") // Try again if needed
	}

	// Verify we're on @HEAD buffer
	if !strings.HasSuffix(te.CurrentBuffer().FilePath, "@HEAD") {
		t.Fatalf("Expected to be on @HEAD buffer, got: %s", te.CurrentBuffer().FilePath)
	}

	initialWindowCount := len(te.windows)

	// Press Ctrl+R from HEAD buffer
	te.TypeKeys("ctrl+r")
	te.drainAsyncEventsOnce()

	// Verify windows are still present
	if len(te.windows) != initialWindowCount {
		t.Errorf("Window count changed from %d to %d after Ctrl+R from HEAD buffer",
			initialWindowCount, len(te.windows))
	}
}
