package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"veda/internal/buffer"
)

func TestStartDiffViewPreservesCursorScreenLine(t *testing.T) {
	repo := NewGitTestRepo(t)
	runGit(t, repo, "init", ".")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")

	currentLines := make([]string, 100)
	for i := range currentLines {
		currentLines[i] = fmt.Sprintf("line %03d", i)
	}
	headLines := append([]string{}, currentLines[:41]...)
	for i := 0; i < 6; i++ {
		headLines = append(headLines, fmt.Sprintf("HEAD-only line %d", i))
	}
	headLines = append(headLines, currentLines[41:]...)

	currentContent := strings.Join(currentLines, "\n") + "\n"
	headContent := strings.Join(headLines, "\n") + "\n"
	path := filepath.Join(repo, "position.txt")
	if err := os.WriteFile(path, []byte(headContent), 0644); err != nil {
		t.Fatalf("write HEAD content: %v", err)
	}
	runGit(t, repo, "add", filepath.Base(path))
	runGit(t, repo, "commit", "-m", "initial")
	if err := os.WriteFile(path, []byte(currentContent), 0644); err != nil {
		t.Fatalf("write current content: %v", err)
	}

	te := NewTestEditorWithContent(120, 50, currentContent)
	buf := te.CurrentBuffer()
	oldPath := buf.FilePath
	buf.FilePath = path
	buf.RealPath = path
	delete(te.buffers, oldPath)
	te.buffers[path] = buf

	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, 30)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, 65)
	win := te.windows[te.activeWindow]
	win.ViewLine = 30
	win.CursorLine = 65
	te.render()
	_, beforeY := te.GetScreenCursorPosition()

	if err := te.StartDiffViewForCurrentBuffer(); err != nil {
		t.Fatalf("start diff view: %v", err)
	}
	te.scroll()
	te.render()
	_, afterY := te.GetScreenCursorPosition()
	if difference := abs(beforeY - afterY); difference > 1 {
		t.Fatalf("cursor moved from screen row %d to %d", beforeY, afterY)
	}

	diffView := te.windows[te.activeWindow].View.(*DiffBufferView)
	sourceLine, err := diffView.ResolveDiffLine(te.CurrentBuffer().CursorIterator.LineNumber())
	if err != nil {
		t.Fatalf("resolve cursor diff row: %v", err)
	}
	if sourceLine != 65 {
		t.Fatalf("cursor moved from source line 65 to %d", sourceLine)
	}
}
