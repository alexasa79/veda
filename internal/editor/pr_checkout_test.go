package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/events"
	"veda/internal/proxy"
)

func TestPRCheckoutProtectsUnsavedBuffers(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "unsaved")
	t.Cleanup(te.Close)
	te.CurrentBuffer().IsModified = true
	if _, err := te.preparePRCheckout(); err == nil {
		t.Fatal("allowed checkout with unsaved file")
	}
	if _, checkingOut := te.asyncOperations[prCheckoutOperation]; checkingOut {
		t.Fatal("checkout started after failed preflight")
	}
	te.CurrentBuffer().IsModified = false
	files, err := te.preparePRCheckout()
	if err != nil || len(files) == 0 {
		t.Fatalf("clean preflight: %v, %v", files, err)
	}
	defer te.endAsyncOperation(prCheckoutOperation)
	if _, err := te.preparePRCheckout(); err == nil {
		t.Fatal("allowed concurrent checkout")
	}
	te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'x'})
	if te.CurrentBuffer().Rope.String() != "unsaved" {
		t.Fatal("edited during checkout")
	}
}

func TestPRCheckoutRefreshesBuffersAndInvalidatesSnapshots(t *testing.T) {
	te := NewTestEditorWithContent(100, 30, "old contents")
	t.Cleanup(te.Close)
	b := te.CurrentBuffer()
	b.RealPath = "host:/project/main.go"
	q := proxy.PRReviewRequest{Action: "file", Base: strings.Repeat("a", 40), Head: strings.Repeat("b", 40), Path: "main.go"}
	te.openPRDiff(prLink("host:/project", q), q, computeDiff("old\n", "new\n"), b.RealPath)
	view := te.findDiffViewForBuffer(te.CurrentBuffer())
	te.applyPRCheckout(&proxy.PRCheckoutResult{Root: "host:/project", Files: []proxy.PRCheckoutFile{{Path: b.RealPath, Content: "new contents\nsecond line"}}})
	if b.Rope.String() != "new contents\nsecond line" || b.IsModified || b.LineCount != 2 {
		t.Fatal("working buffer not refreshed")
	}
	for _, pane := range view.AllBuffers() {
		if pane.HasFile() || !pane.IsReadOnly {
			t.Fatal("old snapshot still attached to checkout")
		}
	}
	if _, _, _, err := te.getLSPTargetForBuffer(view.RightBuffer); err == nil {
		t.Fatal("stale snapshot allowed navigation")
	}
	te.runInvariants()
	// Reopening the same snapshot after verification restores its association.
	te.openPRDiff(prLink("host:/project", q), q, computeDiff("old\n", "new\n"), "host:/project/main.go")
	if view.RightBuffer.RealPath != "host:/project/main.go" {
		t.Fatal("snapshot did not reattach")
	}
	te.applyPRCheckout(&proxy.PRCheckoutResult{Root: "host:/project", Files: []proxy.PRCheckoutFile{{Path: b.RealPath, Error: "file removed"}}})
	if b.HasFile() || !b.IsReadOnly || b.Rope.String() != "new contents\nsecond line" {
		t.Fatal("removed file was not preserved and detached")
	}
}

func TestPRDiffLSPUsesWorkingFileAndUnalignedLine(t *testing.T) {
	te := NewTestEditorWithContent(100, 30, "")
	t.Cleanup(te.Close)
	q := proxy.PRReviewRequest{Action: "file", Base: strings.Repeat("a", 40), Head: strings.Repeat("b", 40), Path: "new.go"}
	previous := te.CurrentBuffer()
	if err := te.ensurePRWorkingBuffer("host:/project/new.go", "keep\ncall()\n"); err != nil {
		t.Fatal(err)
	}
	if te.CurrentBuffer() != previous {
		t.Fatal("preparing source buffer changed the active buffer")
	}
	te.openPRDiff(prLink("host:/project", q), q, computeDiff("deleted\nkeep\ncall()\n", "keep\ncall()\n"), "host:/project/new.go")
	b := te.CurrentBuffer()
	for i, line := range b.DiffHints {
		if line.Content == "call()" {
			b.CursorIterator = buffer.NewBufferIteratorAtLine(b, i)
			b.CursorX = 2
		}
	}
	path, line, col, err := te.getLSPTargetForBuffer(b)
	if err != nil || path != "host:/project/new.go" || line != 1 || col != 2 {
		t.Fatalf("LSP target = %q %d:%d, %v", path, line, col, err)
	}
	te.navigateToBufferTarget(path, path)
	if te.isDiffBuffer(te.CurrentBuffer()) || te.CurrentBuffer().Rope.String() != "keep\ncall()\n" {
		t.Fatal("navigation reopened aligned diff instead of real source")
	}
}
