package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"veda/internal/proxy"
)

func TestGoToDefinitionInDiffView(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LSP integration test in short mode")
	}
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH; skipping test")
	}

	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	repoDir := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	goMod := "module example.com/test\n\ngo 1.22\n"
	original := `package main

func add(a, b int) int {
	return a + b
}

func main() {
	result := add(1, 2)
	println(result)
}
`
	modified := `package main

func add(a, b int) int {
	return a + b
}

func main() {
	result := add(1, 2)
	println("result", result)
}
`

	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	filePath := filepath.Join(repoDir, "main.go")
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	if err := os.WriteFile(filePath, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified main.go: %v", err)
	}

	if _, err := pm.GetFile(filePath, true); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "gopls") || strings.Contains(err.Error(), "not found") {
			t.Skipf("gopls not available through proxy: %v", err)
		}
		t.Fatalf("GetFile failed: %v", err)
	}
	time.Sleep(3 * time.Second)

	te := NewTestEditorWithContent(120, 30, modified)
	te.Editor.proxyManager = pm
	buf := te.CurrentBuffer()
	oldPath := buf.FilePath
	buf.FilePath = filePath
	buf.RealPath = filePath
	te.Editor.lspExtensionStatus[filepath.Ext(filePath)] = "ready"
	delete(te.Editor.buffers, oldPath)
	te.Editor.buffers[filePath] = buf

	if err := te.StartDiffViewForCurrentBuffer(); err != nil {
		t.Fatalf("StartDiffViewForCurrentBuffer failed: %v", err)
	}
	if te.CurrentBuffer() == nil || !strings.HasSuffix(te.CurrentBuffer().FilePath, "@current") {
		t.Fatalf("Expected to be on @current diff buffer, got %v", te.CurrentBuffer().FilePath)
	}

	diffContent := te.CurrentBuffer().Rope.String()
	callLine, callCol, ok := findLineCol(diffContent, "add(1, 2)")
	if !ok {
		t.Fatalf("did not find call site in diff buffer content")
	}
	te.SetDiffCursorPosition(callCol, callLine)

	defLine, _, ok := findLineCol(modified, "func add(")
	if !ok {
		t.Fatalf("did not find definition in modified content")
	}

	if err := te.ExecuteCommand("go-to-definition"); err != nil {
		t.Fatalf("go-to-definition failed: %v", err)
	}

	if te.CurrentBuffer() == nil {
		t.Fatalf("no current buffer after go-to-definition")
	}
	if strings.HasSuffix(te.CurrentBuffer().FilePath, "@current") || strings.HasSuffix(te.CurrentBuffer().FilePath, "@HEAD") {
		t.Fatalf("expected navigation to real file buffer, still on %s", te.CurrentBuffer().FilePath)
	}
	if te.CurrentBuffer().CursorIterator.LineNumber() != defLine {
		t.Fatalf("expected definition line %d, got %d", defLine, te.CurrentBuffer().CursorIterator.LineNumber())
	}
}

func findLineCol(content, needle string) (int, int, bool) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if col := strings.Index(line, needle); col >= 0 {
			return i, col, true
		}
	}
	return 0, 0, false
}
