package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestDiffView_OpenFileExitsDiffView(t *testing.T) {
	te := NewTestEditorWithContent(120, 30, "line1\nline2\nline3")

	// Create diff buffers for a fake file and attach them to windows.
	diffResult := computeDiff(te.CurrentBuffer().Rope.String(), te.CurrentBuffer().Rope.String())
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), "dummy.go@HEAD", "dummy.go@HEAD", config.IndentTabs, 4)
	head.IsReadOnly = true
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), "dummy.go@current", "dummy.go@current", config.IndentTabs, 4)

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr
	te.diffLineInfo[head] = diffResult.LeftLines
	te.diffLineInfo[curr] = diffResult.RightLines

	// Left window keeps original buffer; add HEAD diff buffer on top.
	te.windows[0].addBufferToWindow(head)

	// Right window shows @current.
	te.createSecondWindow()
	te.windows[1].addBufferToWindow(curr)
	te.activeWindow = 1
	te.switchToBuffer(curr)
	te.recomputeWindowsLayout()

	if !te.isDiffViewActive() {
		t.Fatalf("expected diff view to be active before opening a file")
	}

	target := "internal/editor/contextual_navigation.go"
	te.openFile(target)

	if te.isDiffViewActive() {
		t.Fatalf("expected diff view to be closed after opening a file")
	}
	if len(te.windows) != 1 {
		t.Fatalf("expected a single window after exiting diff view, got %d", len(te.windows))
	}
	expectedPath := te.resolvePath(target)
	if te.CurrentBuffer() == nil || te.CurrentBuffer().FilePath != expectedPath {
		t.Fatalf("expected current buffer to be %s, got %v", expectedPath, te.CurrentBuffer())
	}
	if _, ok := te.buffers[head.FilePath]; ok {
		t.Fatalf("expected head diff buffer to be removed")
	}
	if _, ok := te.buffers[curr.FilePath]; ok {
		t.Fatalf("expected current diff buffer to be removed")
	}
}
