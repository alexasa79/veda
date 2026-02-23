package editor

import (
	"fmt"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestNavigationHistory(t *testing.T) {
	// Create editor with enough lines
	content := ""
	for i := 0; i < 20; i++ {
		if i > 0 {
			content += "\n"
		}
		content += fmt.Sprintf("Line %d", i)
	}
	te := NewTestEditorWithContent(80, 24, content)

	// Set file path and position
	te.CurrentBuffer().FilePath = "test.go"
	te.SetCursorPosition(5, 10)

	// Push to navigation history
	te.pushNavigationHistory()

	// Check that history was recorded
	if len(te.navHistory) != 1 {
		t.Errorf("Expected 1 entry in navigation history, got %d", len(te.navHistory))
	}

	entry := te.navHistory[0]
	if entry.filePath != "test.go" {
		t.Errorf("Expected filePath to be 'test.go', got %s", entry.filePath)
	}
	if entry.cursorX != 5 {
		t.Errorf("Expected cursorX to be 5, got %d", entry.cursorX)
	}
	if entry.cursorLineNumber != 10 {
		t.Errorf("Expected cursorLineNumber to be 10, got %d", entry.cursorLineNumber)
	}
}

func TestNavigationHistorySkipsReferencesBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "test content")

	// First, push a normal file position
	te.CurrentBuffer().FilePath = "main.go"
	te.SetCursorPosition(5, 10)
	te.pushNavigationHistory()

	// Verify it was added
	if len(te.navHistory) != 1 {
		t.Errorf("Expected 1 entry in navigation history, got %d", len(te.navHistory))
	}

	// Now set to references buffer and try to push
	te.CurrentBuffer().FilePath = "*References: test*"
	te.SetCursorPosition(3, 5)
	te.pushNavigationHistory()

	// Verify that references buffer was NOT added
	if len(te.navHistory) != 1 {
		t.Errorf("Expected navigation history to still have 1 entry (references should be skipped), got %d", len(te.navHistory))
	}

	// Verify the original entry is still there
	if te.navHistory[0].filePath != "main.go" {
		t.Errorf("Expected first entry to still be main.go, got %s", te.navHistory[0].filePath)
	}

	// Try another normal file
	te.CurrentBuffer().FilePath = "other.go"
	te.SetCursorPosition(7, 12)
	te.pushNavigationHistory()

	// Now we should have 2 entries
	if len(te.navHistory) != 2 {
		t.Errorf("Expected 2 entries in navigation history, got %d", len(te.navHistory))
	}

	// Verify the new entry
	if te.navHistory[1].filePath != "other.go" {
		t.Errorf("Expected second entry to be other.go, got %s", te.navHistory[1].filePath)
	}
}

func TestNavigationHistoryMaxSize(t *testing.T) {
	// Create editor with enough lines
	content := ""
	for i := 0; i < 60; i++ {
		if i > 0 {
			content += "\n"
		}
		content += fmt.Sprintf("Line %d", i)
	}
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().FilePath = "test.go"

	// Push more entries than maxNavHistory
	for i := 0; i < 60; i++ {
		te.SetCursorPosition(0, i)
		te.pushNavigationHistory()
	}

	// Should only keep the last 50 entries
	if len(te.navHistory) != 50 {
		t.Errorf("Expected navigation history to be capped at 50 entries, got %d", len(te.navHistory))
	}

	// First entry should be from position 10 (0-59, keeping last 50)
	if te.navHistory[0].cursorLineNumber != 10 {
		t.Errorf("Expected first entry cursorLineNumber to be 10, got %d", te.navHistory[0].cursorLineNumber)
	}
}

func TestGoBack(t *testing.T) {
	// Create a test editor with enough content for the cursor positions
	content := ""
	for i := 0; i < 35; i++ {
		content += fmt.Sprintf("Line %d with some text to make it long enough\n", i)
	}
	te := NewTestEditorWithContent(80, 24, content)

	// Set up initial state with a buffer
	te.CurrentBuffer().FilePath = "test1.go"
	te.buffers["test1.go"] = te.CurrentBuffer()

	// Push a few positions to history
	te.SetCursorPosition(5, 10)
	te.pushNavigationHistory()

	te.SetCursorPosition(15, 20)
	te.pushNavigationHistory()

	// Move to a different position
	te.SetCursorPosition(25, 30)

	// Go back once
	err := te.goBack(nil)
	if err != nil {
		t.Errorf("goBack returned error: %v", err)
	}

	// Should be at the second position
	x, y := te.GetCursorPosition()
	if x != 15 || y != 20 {
		t.Errorf("Expected cursor at (15, 20), got (%d, %d)", x, y)
	}

	// History should have one entry left
	if len(te.navHistory) != 1 {
		t.Errorf("Expected 1 entry in navigation history, got %d", len(te.navHistory))
	}

	// Go back again
	err = te.goBack(nil)
	if err != nil {
		t.Errorf("goBack returned error: %v", err)
	}

	// Should be at the first position
	x, y = te.GetCursorPosition()
	if x != 5 || y != 10 {
		t.Errorf("Expected cursor at (5, 10), got (%d, %d)", x, y)
	}

	// History should be empty
	if len(te.navHistory) != 0 {
		t.Errorf("Expected navigation history to be empty, got %d entries", len(te.navHistory))
	}
}

func TestGoBackEmptyHistory(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Going back with empty history should not error
	err := te.goBack(nil)
	if err != nil {
		t.Errorf("goBack with empty history returned error: %v", err)
	}
}

func TestGoBack_DiffHistoryEntryRestoresRealFile(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")

	realPath := te.resolvePath("internal/editor/navigation.go")
	diffPath := realPath + "@current"

	// Simulate being on a diff buffer when navigation history is pushed (e.g. before gd).
	te.CurrentBuffer().FilePath = diffPath
	te.CurrentBuffer().RealPath = realPath
	te.SetCursorPosition(1, 1)
	te.pushNavigationHistory()

	if len(te.navHistory) != 1 {
		t.Fatalf("expected 1 navigation history entry, got %d", len(te.navHistory))
	}

	// Simulate leaving diff view so the synthetic diff buffers are no longer present.
	te.CurrentBuffer().FilePath = realPath
	te.CurrentBuffer().RealPath = realPath
	delete(te.buffers, diffPath)
	delete(te.buffers, realPath+"@HEAD")

	// Desired behavior: go-back should restore the real file location and not panic.
	if err := te.ExecuteCommand("go-back"); err != nil {
		t.Fatalf("go-back returned error: %v", err)
	}
	if te.CurrentBuffer() == nil {
		t.Fatalf("expected active buffer after go-back")
	}
	if te.CurrentBuffer().FilePath != realPath {
		t.Fatalf("expected go-back to restore real file %s, got %s", realPath, te.CurrentBuffer().FilePath)
	}
}

func TestGoBack_ReturnsToDiffBufferWhenDiffViewStillOpen(t *testing.T) {
	te := NewTestEditorWithContent(120, 30, "line1\nline2\nline3")

	realPath := te.CurrentBuffer().FilePath
	diffResult := computeDiff(te.CurrentBuffer().Rope.String(), te.CurrentBuffer().Rope.String())
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), realPath+"@HEAD", realPath+"@HEAD", config.IndentTabs, 4)
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), realPath+"@current", realPath+"@current", config.IndentTabs, 4)
	head.RealPath = realPath
	curr.RealPath = realPath
	head.IsReadOnly = true

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr
	head.DiffHints = diffResult.LeftLines
	curr.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on a single window
	te.SetupDiffView(head, curr, realPath)

	if !te.isDiffViewActive() {
		t.Fatalf("expected diff view to be active")
	}

	te.pushNavigationHistory()

	forward := buffer.NewBuffer("forward", realPath+".forward", realPath+".forward", config.IndentTabs, 4)
	te.buffers[forward.FilePath] = forward
	te.windows[te.activeWindow].addBufferToWindow(forward)
	te.switchToBuffer(forward)

	if err := te.ExecuteCommand("go-back"); err != nil {
		t.Fatalf("go-back returned error: %v", err)
	}

	if te.CurrentBuffer() == nil {
		t.Fatalf("expected current buffer after go-back")
	}
	if te.CurrentBuffer().FilePath != curr.FilePath {
		t.Fatalf("expected go-back to return to diff buffer %s, got %s", curr.FilePath, te.CurrentBuffer().FilePath)
	}
	if !te.isDiffBuffer(te.CurrentBuffer()) {
		t.Fatalf("expected go-back target to be a diff buffer, got %s", te.CurrentBuffer().FilePath)
	}
	if !te.isDiffViewActive() {
		t.Fatalf("expected diff view to remain active after go-back")
	}
}

func TestGoBack_ReturnsToRealFileWhenDiffViewClosed(t *testing.T) {
	te := NewTestEditorWithContent(120, 30, "line1\nline2\nline3")

	realPath := te.CurrentBuffer().FilePath
	diffResult := computeDiff(te.CurrentBuffer().Rope.String(), te.CurrentBuffer().Rope.String())
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), realPath+"@HEAD", realPath+"@HEAD", config.IndentTabs, 4)
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), realPath+"@current", realPath+"@current", config.IndentTabs, 4)
	head.RealPath = realPath
	curr.RealPath = realPath
	head.IsReadOnly = true

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr
	head.DiffHints = diffResult.LeftLines
	curr.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on a single window
	te.SetupDiffView(head, curr, realPath)

	if !te.isDiffViewActive() {
		t.Fatalf("expected diff view to be active")
	}

	te.pushNavigationHistory()

	forward := buffer.NewBuffer("forward", realPath+".forward", realPath+".forward", config.IndentTabs, 4)
	te.buffers[forward.FilePath] = forward
	te.windows[te.activeWindow].addBufferToWindow(forward)
	te.switchToBuffer(forward)

	te.exitDiffViewIfActive()
	if te.isDiffViewActive() {
		t.Fatalf("expected diff view to be closed before go-back")
	}

	// Expected behavior: if diff view is no longer available, go-back should restore the real file.
	if err := te.ExecuteCommand("go-back"); err != nil {
		t.Fatalf("go-back returned error: %v", err)
	}

	if te.CurrentBuffer() == nil {
		t.Fatalf("expected current buffer after go-back")
	}
	if te.CurrentBuffer().FilePath != realPath {
		t.Fatalf("expected go-back to return to real file %s, got %s", realPath, te.CurrentBuffer().FilePath)
	}
	if te.isDiffBuffer(te.CurrentBuffer()) {
		t.Fatalf("expected go-back target to be non-diff buffer when diff view is closed")
	}
}

// TestGoBack_RestoresDiffViewAfterOpenFile verifies that navigating to a
// different file via openFile (e.g. gd) keeps the diff view alive, and
// go-back restores it.
func TestGoBack_RestoresDiffViewAfterOpenFile(t *testing.T) {
	te := NewTestEditorWithContent(120, 30, "line1\nline2\nline3")

	realPath := te.CurrentBuffer().FilePath
	diffResult := computeDiff(te.CurrentBuffer().Rope.String(), te.CurrentBuffer().Rope.String())
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), realPath+"@HEAD", realPath+"@HEAD", config.IndentTabs, 4)
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), realPath+"@current", realPath+"@current", config.IndentTabs, 4)
	head.RealPath = realPath
	curr.RealPath = realPath
	head.IsReadOnly = true

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr
	head.DiffHints = diffResult.LeftLines
	curr.DiffHints = diffResult.RightLines

	te.SetupDiffView(head, curr, realPath)

	if !te.isDiffViewActive() {
		t.Fatalf("expected diff view to be active")
	}

	// Save position, then navigate away via openFile (simulates gd)
	te.pushNavigationHistory()

	target := buffer.NewBuffer("target content", realPath+".target", realPath+".target", config.IndentTabs, 4)
	te.buffers[target.FilePath] = target
	te.openFile(target.FilePath)

	if te.CurrentBuffer().FilePath != target.FilePath {
		t.Fatalf("expected current buffer to be %s, got %s", target.FilePath, te.CurrentBuffer().FilePath)
	}

	// Navigate back — should restore the diff view
	if err := te.ExecuteCommand("go-back"); err != nil {
		t.Fatalf("go-back returned error: %v", err)
	}

	if te.CurrentBuffer() == nil {
		t.Fatalf("expected current buffer after go-back")
	}
	if te.CurrentBuffer().FilePath != curr.FilePath {
		t.Fatalf("expected go-back to return to diff buffer %s, got %s", curr.FilePath, te.CurrentBuffer().FilePath)
	}
	if !te.isDiffViewActive() {
		t.Fatalf("expected diff view to be restored after go-back")
	}
}
