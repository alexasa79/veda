package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"veda/internal/syntax"
)

// TestDiffViewLeftPaneSyntaxHighlighting verifies that the left (@HEAD) side of diff view
// renders with syntax highlighting by creating a tiny git repo with a Go file, committing it,
// modifying the file, constructing a diff, and inspecting rendered cell styles.
func TestDiffViewLeftPaneSyntaxHighlighting(t *testing.T) {
	// Skip if Go grammar is not available
	h := syntax.NewHighlighter()
	if err := h.SetLanguageFromFile("test.go"); err != nil {
		t.Skipf("Skipping: Go syntax highlighting not available: %v", err)
	}

	// Create temporary git repo
	repoDir := t.TempDir()
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Initial Go content (contains 'func' keyword for highlighting)
	orig := "package main\n\nfunc main() {\n}\n"
	filePath := filepath.Join(repoDir, "main.go")
	if err := os.WriteFile(filePath, []byte(orig), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("git", "add", "main.go")
	run("git", "commit", "-m", "initial")

	// Modify working tree to produce a diff
	modified := "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"hi\")\n}\n"
	if err := os.WriteFile(filePath, []byte(modified), 0644); err != nil {
		t.Fatalf("modify file: %v", err)
	}

	// Read previous content from HEAD using git show
	cmd := exec.Command("git", "show", "HEAD:main.go")
	cmd.Dir = repoDir
	prevBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git show failed: %v\n%s", err, string(prevBytes))
	}
	previous := string(prevBytes)

	// Prepare editor with current (modified) content using high-level API
	te := NewTestEditorWithContent(120, 30, modified)
	te.CurrentBuffer().FilePath = filePath
	te.CurrentBuffer().RealPath = filePath
	// Trigger diff view via key sequence ("s c")
	te.TypeKeys("s c")
	// We are now on the right (@current) pane; switch to the left with window-next
	te.TypeKeys("space tab")
	// Ensure we are on @HEAD; if not, cycle once more
	if te.CurrentBuffer() == nil || !strings.HasSuffix(te.CurrentBuffer().BufferName, "@HEAD") {
		te.TypeKeys("space tab")
	}
	// Render
	te.render()

	// Find a line containing 'func' in the left (HEAD) content
	// Use the actual buffer content now shown on the left pane
	headLines := strings.Split(te.CurrentBuffer().Rope.String(), "\n")
	funcLine := -1
	for i, ln := range headLines {
		if strings.Contains(ln, "func ") {
			funcLine = i
			break
		}
	}
	if funcLine == -1 {
		t.Fatalf("did not find 'func' in HEAD content: \n%s", previous)
	}

	// Expected keyword style
	expectedStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxKeyword)
	funcStyle := getCellStyle(te, funcLine, "func")
	if funcStyle == nil {
		t.Fatalf("could not locate 'func' on screen in left pane")
	}

	// Compare foreground color and attributes (ignore background)
	if !stylesMatch(*funcStyle, expectedStyle) {
		aFg, _, aAttr := funcStyle.Decompose()
		eFg, _, eAttr := expectedStyle.Decompose()
		t.Fatalf("left pane syntax highlighting missing or wrong: got fg=%v attr=%v, want fg=%v attr=%v", aFg, aAttr, eFg, eAttr)
	}
}
