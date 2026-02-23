package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestDiffView_OpenFileKeepsDiffView verifies that opening a file while a diff
// view is active does NOT destroy the diff view. The diff view stays in the
// window's Views list so it can be restored when navigating back.
func TestDiffView_OpenFileKeepsDiffView(t *testing.T) {
	te := NewTestEditorWithContent(120, 30, "line1\nline2\nline3")

	// Create diff buffers for a fake file and attach them via DiffBufferView.
	diffResult := computeDiff(te.CurrentBuffer().Rope.String(), te.CurrentBuffer().Rope.String())
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), "dummy.go@HEAD", "dummy.go@HEAD", config.IndentTabs, 4)
	head.IsReadOnly = true
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), "dummy.go@current", "dummy.go@current", config.IndentTabs, 4)

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr
	head.DiffHints = diffResult.LeftLines
	curr.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on the single window
	te.SetupDiffView(head, curr, "dummy.go")

	if !te.isDiffViewActive() {
		t.Fatalf("expected diff view to be active before opening a file")
	}

	target := "internal/editor/contextual_navigation.go"
	te.openFile(target)

	// The current buffer should be the newly opened file
	expectedPath := te.resolvePath(target)
	if te.CurrentBuffer() == nil || te.CurrentBuffer().FilePath != expectedPath {
		t.Fatalf("expected current buffer to be %s, got %v", expectedPath, te.CurrentBuffer())
	}

	// The diff buffers should still exist
	if _, ok := te.buffers[head.FilePath]; !ok {
		t.Fatalf("expected head diff buffer to still exist")
	}
	if _, ok := te.buffers[curr.FilePath]; !ok {
		t.Fatalf("expected current diff buffer to still exist")
	}

	// The DiffBufferView should still be in the window's Views list
	win := te.windows[te.activeWindow]
	foundDiff := false
	for _, v := range win.Views {
		if _, ok := v.(*DiffBufferView); ok {
			foundDiff = true
			break
		}
	}
	if !foundDiff {
		t.Fatalf("expected DiffBufferView to remain in window Views after opening a file")
	}
}
