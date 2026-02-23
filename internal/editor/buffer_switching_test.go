package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// createInternalBuffer creates a buffer that mimics an internal buffer like *Project Changes*.
func createInternalBuffer(te *TestEditor, name, content string) *buffer.Buffer {
	buf := buffer.NewBuffer(content, name, name, te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf.IsInternal = true
	buf.IsReadOnly = true
	te.buffers[name] = buf
	te.bufferOrder = append(te.bufferOrder, name)
	return buf
}

// setupDiffViewOnFile creates HEAD and current diff buffers for the given
// original buffer and installs a DiffBufferView on the active window.
func setupDiffViewOnFile(te *TestEditor, origBuf *buffer.Buffer) *DiffBufferView {
	origPath := origBuf.FilePath
	diffResult := computeDiff(origBuf.Rope.String(), origBuf.Rope.String()+"modified\n")
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), origPath+"@HEAD", origPath+"@HEAD", config.IndentTabs, 4)
	head.IsReadOnly = true
	head.DiffHints = diffResult.LeftLines
	head.RealPath = origPath
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), origPath+"@current", origPath+"@current", config.IndentTabs, 4)
	curr.DiffHints = diffResult.RightLines
	curr.RealPath = origPath

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr

	return te.SetupDiffView(head, curr, origPath)
}

// assertBufferListContains checks that the buffer list contains an entry
// matching the given substring.
func assertBufferListContains(t *testing.T, entries []string, substr string) {
	t.Helper()
	for _, e := range entries {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("Expected buffer list to contain entry with %q, got: %v", substr, entries)
}

// assertBufferListNotContains checks that the buffer list does NOT contain
// an entry matching the given substring.
func assertBufferListNotContains(t *testing.T, entries []string, substr string) {
	t.Helper()
	for _, e := range entries {
		if strings.Contains(e, substr) {
			t.Errorf("Expected buffer list NOT to contain entry with %q, but found: %q (all: %v)", substr, e, entries)
			return
		}
	}
}

// --- Tests ---

// TestSwitch_NormalToProjectChangesAndBack verifies basic switching between
// a normal file buffer and an internal buffer like *Project Changes*.
func TestSwitch_NormalToProjectChangesAndBack(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "file content")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	// Create *Project Changes* internal buffer
	pcBuf := createInternalBuffer(te, "*Project Changes*", "# Project Changes\n\nSome changes")

	// Switch to project changes
	te.switchToBuffer(pcBuf)
	if te.CurrentBuffer() != pcBuf {
		t.Fatalf("Expected current buffer to be *Project Changes*, got %s", te.CurrentBuffer().FilePath)
	}
	if te.previousBuffer != origBuf {
		t.Fatalf("Expected previousBuffer to be original file, got %v", te.previousBuffer)
	}

	// Switch back to original
	te.switchToBuffer(origBuf)
	if te.CurrentBuffer() != origBuf {
		t.Fatalf("Expected current buffer to be original, got %s", te.CurrentBuffer().FilePath)
	}
	if te.previousBuffer != pcBuf {
		t.Fatalf("Expected previousBuffer to be *Project Changes*, got %v", te.previousBuffer)
	}

	// Buffer list should show both
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, origPath)
	assertBufferListContains(t, entries, "*Project Changes*")
}

// TestSwitch_ProjectChangesToDiffViewAndBack is the exact scenario from the
// bug report: switch to project changes, then to a buffer, open diff view,
// then try to go back to project changes.
func TestSwitch_ProjectChangesToDiffViewAndBack(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	pcBuf := createInternalBuffer(te, "*Project Changes*", "# Project Changes\n\nSome changes")

	// Step 1: Switch to project changes
	te.switchToBuffer(pcBuf)
	if te.CurrentBuffer().FilePath != "*Project Changes*" {
		t.Fatalf("Step 1: Expected *Project Changes*, got %s", te.CurrentBuffer().FilePath)
	}

	// Step 2: Switch back to the file buffer
	te.switchToBuffer(origBuf)
	if te.CurrentBuffer().FilePath != origPath {
		t.Fatalf("Step 2: Expected %s, got %s", origPath, te.CurrentBuffer().FilePath)
	}

	// Step 3: Open diff view on the file
	setupDiffViewOnFile(te, origBuf)
	if !te.isDiffViewActive() {
		t.Fatalf("Step 3: Expected diff view to be active")
	}

	// Step 4: Check buffer list — *Project Changes* must still be present
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, "*Project Changes*")
	assertBufferListNotContains(t, entries, "@HEAD")
	assertBufferListNotContains(t, entries, "@current")

	// Step 5: Switch to *Project Changes* from diff view
	te.switchToBuffer(pcBuf)
	if te.CurrentBuffer().FilePath != "*Project Changes*" {
		t.Fatalf("Step 5: Expected *Project Changes*, got %s", te.CurrentBuffer().FilePath)
	}

	// Step 6: Buffer list should still show both the file (with diff) and project changes
	entries = collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, "*Project Changes*")
	assertBufferListContains(t, entries, origPath)
	assertBufferListNotContains(t, entries, "@HEAD")
	assertBufferListNotContains(t, entries, "@current")
}

// TestSwitch_DiffInternalBuffersNeverAppearInList verifies that @HEAD and
// @current buffers never appear in the buffer list under any combination of
// switches.
func TestSwitch_DiffInternalBuffersNeverAppearInList(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "content")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")
	msgBuf := createInternalBuffer(te, "*Messages*", "messages")

	setupDiffViewOnFile(te, origBuf)

	// Check list after each switch
	scenarios := []struct {
		name   string
		target *buffer.Buffer
	}{
		{"on_diff_buffer", te.CurrentBuffer()},
		{"on_project_changes", pcBuf},
		{"on_messages", msgBuf},
		{"back_on_diff", te.buffers[origPath+"@current"]},
		{"on_original_file", origBuf},
		{"on_project_changes_again", pcBuf},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			te.switchToBuffer(sc.target)
			entries := collectBufferDisplayEntries(te)
			assertBufferListNotContains(t, entries, "@HEAD")
			assertBufferListNotContains(t, entries, "@current")

			// Internal buffers should always be present
			assertBufferListContains(t, entries, "*Project Changes*")
			assertBufferListContains(t, entries, "*Messages*")
		})
	}
}

// TestSwitch_PreviousBufferAcrossDiffTransitions verifies that previousBuffer
// is tracked correctly when switching in and out of diff views.
func TestSwitch_PreviousBufferAcrossDiffTransitions(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "content")
	origBuf := te.CurrentBuffer()

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")

	// Start on original file
	if te.previousBuffer != nil {
		// previousBuffer may be [No Name] from NewTestEditor; clear for test clarity
		te.previousBuffer = nil
	}

	// Switch to project changes -> previous should be original
	te.switchToBuffer(pcBuf)
	if te.previousBuffer != origBuf {
		t.Errorf("After switch to PC: expected previous=%s, got %v", origBuf.FilePath, previousPath(te))
	}

	// Switch back to original -> previous should be project changes
	te.switchToBuffer(origBuf)
	if te.previousBuffer != pcBuf {
		t.Errorf("After switch back: expected previous=*Project Changes*, got %v", previousPath(te))
	}

	// Enter diff view — SetupDiffView sets win.Buffer directly before calling
	// switchToBuffer, so previousBuffer is NOT updated (CurrentBuffer() already
	// equals the target). It stays as pcBuf from the previous switch.
	setupDiffViewOnFile(te, origBuf)
	currBuf := te.CurrentBuffer()
	if te.previousBuffer != pcBuf {
		t.Errorf("After diff view: expected previous=*Project Changes* (unchanged), got %v", previousPath(te))
	}

	// Switch to project changes from diff -> previous should be @current buffer
	te.switchToBuffer(pcBuf)
	if te.previousBuffer != currBuf {
		t.Errorf("After switch to PC from diff: expected previous=%s, got %v", currBuf.FilePath, previousPath(te))
	}

	// Switch back to @current -> previous should be project changes
	te.switchToBuffer(currBuf)
	if te.previousBuffer != pcBuf {
		t.Errorf("After switch back to diff: expected previous=*Project Changes*, got %v", previousPath(te))
	}
}

func previousPath(te *TestEditor) string {
	if te.previousBuffer == nil {
		return "<nil>"
	}
	return te.previousBuffer.FilePath
}

// TestSwitch_CursorPositionPreservedAcrossTransitions verifies that cursor
// position is preserved when switching away from and back to buffers.
func TestSwitch_CursorPositionPreservedAcrossTransitions(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3\nline4\nline5")
	origBuf := te.CurrentBuffer()

	// Set cursor to line 3, col 2
	te.SetCursorPosition(2, 3)

	pcBuf := createInternalBuffer(te, "*Project Changes*", "# Changes\nline a\nline b\nline c")

	// Switch to PC, move cursor there
	te.switchToBuffer(pcBuf)
	te.SetCursorPosition(1, 2)

	// Switch back — cursor should be at line 3, col 2
	te.switchToBuffer(origBuf)
	x, y := te.GetCursorPosition()
	if x != 2 || y != 3 {
		t.Errorf("Expected cursor at (2,3), got (%d,%d)", x, y)
	}

	// Open diff view
	setupDiffViewOnFile(te, origBuf)

	// Switch to PC
	te.switchToBuffer(pcBuf)

	// Check PC cursor is preserved
	x, y = te.GetCursorPosition()
	if x != 1 || y != 2 {
		t.Errorf("Expected PC cursor at (1,2), got (%d,%d)", x, y)
	}
}

// TestSwitch_ExitDiffViewRestoresOriginalBuffer verifies that exiting diff
// view restores the original file buffer and that all buffers remain accessible.
func TestSwitch_ExitDiffViewRestoresOriginalBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")

	setupDiffViewOnFile(te, origBuf)

	if !te.isDiffViewActive() {
		t.Fatalf("Expected diff view to be active")
	}

	// Exit diff view
	te.exitDiffViewIfActive()

	if te.isDiffViewActive() {
		t.Fatalf("Expected diff view to be inactive after exit")
	}

	// Current buffer should be the original file
	if te.CurrentBuffer().FilePath != origPath {
		t.Errorf("Expected current buffer to be %s, got %s", origPath, te.CurrentBuffer().FilePath)
	}

	// Buffer list should show both original and project changes, no diff entries
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, origPath)
	assertBufferListContains(t, entries, "*Project Changes*")
	assertBufferListNotContains(t, entries, "@HEAD")
	assertBufferListNotContains(t, entries, "@current")
	assertBufferListNotContains(t, entries, "[diff]")

	// Switching to project changes should still work
	te.switchToBuffer(pcBuf)
	if te.CurrentBuffer().FilePath != "*Project Changes*" {
		t.Errorf("Expected to switch to *Project Changes*, got %s", te.CurrentBuffer().FilePath)
	}
}

// TestSwitch_MultipleInternalBuffersWithDiffView tests switching between
// multiple internal buffers when a diff view is active.
func TestSwitch_MultipleInternalBuffersWithDiffView(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "content")
	origBuf := te.CurrentBuffer()

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")
	msgBuf := createInternalBuffer(te, "*Messages*", "messages")
	diagBuf := createInternalBuffer(te, "*Diagnostics*", "diagnostics")

	setupDiffViewOnFile(te, origBuf)

	// Rapidly switch between all buffers and verify each switch works
	buffers := []*buffer.Buffer{pcBuf, msgBuf, diagBuf, te.CurrentBuffer(), pcBuf, origBuf, diagBuf, msgBuf}
	for i, target := range buffers {
		te.switchToBuffer(target)
		if te.CurrentBuffer() != target {
			t.Errorf("Switch %d: expected %s, got %s", i, target.FilePath, te.CurrentBuffer().FilePath)
		}
	}

	// Buffer list should show all non-diff buffers
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, "*Project Changes*")
	assertBufferListContains(t, entries, "*Messages*")
	assertBufferListContains(t, entries, "*Diagnostics*")
	assertBufferListNotContains(t, entries, "@HEAD")
	assertBufferListNotContains(t, entries, "@current")
}

// TestSwitch_BufferListConsistencyAcrossViews verifies that the buffer list
// has the same non-diff entries regardless of which buffer is currently active.
func TestSwitch_BufferListConsistencyAcrossViews(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2")
	origBuf := te.CurrentBuffer()

	file2 := buffer.NewBuffer("other content", "other.go", "other.go", config.IndentTabs, 4)
	te.buffers["other.go"] = file2
	te.windows[te.activeWindow].addBufferToWindow(file2)

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")

	setupDiffViewOnFile(te, origBuf)

	// Collect entries from different active buffers
	targets := map[string]*buffer.Buffer{
		"diff_buffer":     te.CurrentBuffer(),
		"other_file":      file2,
		"project_changes": pcBuf,
		"original_file":   origBuf,
	}

	// All should contain the same set of non-diff buffer names
	expectedNames := []string{"other.go", "*Project Changes*"}

	for name, target := range targets {
		te.switchToBuffer(target)
		entries := collectBufferDisplayEntries(te)

		for _, expected := range expectedNames {
			found := false
			for _, e := range entries {
				if strings.Contains(e, expected) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Active=%s: expected %q in buffer list, got: %v", name, expected, entries)
			}
		}

		// Never diff internals
		assertBufferListNotContains(t, entries, "@HEAD")
		assertBufferListNotContains(t, entries, "@current")
	}
}

// TestSwitch_RendererRespectsActiveBuffer verifies that even though
// DiffBufferView persists on the window, the renderer shows the correct content
// based on which buffer is active. When the active buffer is NOT part of the
// diff, the content area shows that buffer, not the diff.
func TestSwitch_RendererRespectsActiveBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "original file text")
	origBuf := te.CurrentBuffer()

	pcBuf := createInternalBuffer(te, "*Project Changes*", "# Changes\n\nSummary of changes")

	// Enter diff view
	setupDiffViewOnFile(te, origBuf)
	win := te.windows[te.activeWindow]
	if _, isDiff := win.View.(*DiffBufferView); !isDiff {
		t.Fatalf("Expected DiffBufferView after SetupDiffView")
	}

	// Render diff — should show original file text in panes
	te.render()
	mockScreen := te.screen.(*MockScreen)
	if !screenContainsText(mockScreen, "original file text") {
		t.Errorf("Diff render should show file content")
	}

	// Switch to project changes and render
	te.switchToBuffer(pcBuf)
	te.render()
	if !screenContainsText(mockScreen, "Summary of changes") {
		t.Errorf("After switching to *Project Changes*, screen must show its content")
	}
	if screenContainsText(mockScreen, "original file text") {
		t.Errorf("After switching to *Project Changes*, diff content should not be visible")
	}

	// Switch back to diff buffer and render — diff should reappear
	te.switchToBuffer(te.buffers[origBuf.FilePath+"@current"])
	te.render()
	if !screenContainsText(mockScreen, "original file text") {
		t.Errorf("After switching back to diff buffer, diff content should reappear")
	}

	// Exit diff view
	te.exitDiffViewIfActive()
	if _, isDiff := win.View.(*DiffBufferView); isDiff {
		t.Errorf("Expected DiffBufferView to be cleared after exitDiffViewIfActive")
	}
}

// TestSwitch_RenderShowsCorrectBufferAfterSwitch is the definitive reproduction
// of the bug: open diff view, switch to *Project Changes*, render — the screen
// must show the project changes content, not the diff view.
func TestSwitch_RenderShowsCorrectBufferAfterSwitch(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "file content here")
	origBuf := te.CurrentBuffer()

	// Use body text that won't appear in the status bar
	pcContent := "# Project Changes\n\nModified files:\n- editor.go"
	pcBuf := createInternalBuffer(te, "*Project Changes*", pcContent)

	// Step 1: Switch to project changes — verify body content renders
	te.switchToBuffer(pcBuf)
	te.render()
	mockScreen := te.screen.(*MockScreen)
	if !screenContainsText(mockScreen, "Modified files") {
		t.Errorf("Step 1: Expected screen to show 'Modified files' body content")
	}

	// Step 2: Switch back to file
	te.switchToBuffer(origBuf)

	// Step 3: Open diff view
	setupDiffViewOnFile(te, origBuf)
	te.render()
	// Verify diff content is on screen (the original file text appears in both diff panes)
	if !screenContainsText(mockScreen, "file content here") {
		t.Errorf("Step 3: Expected screen to show diff content")
	}

	// Step 4: Switch to project changes from diff view
	te.switchToBuffer(pcBuf)
	te.render()

	// The screen MUST show project changes body content, not the diff
	if !screenContainsText(mockScreen, "Modified files") {
		t.Errorf("Step 4: After switching from diff view, screen must show *Project Changes* body content ('Modified files'), not the diff view")
	}
	// The diff content should NOT be on screen anymore
	if screenContainsText(mockScreen, "file content here") {
		t.Errorf("Step 4: Screen still shows diff content ('file content here') — the diff view is being rendered instead of *Project Changes*")
	}
}

// screenContainsText checks if the given text appears anywhere on the mock screen.
func screenContainsText(screen *MockScreen, text string) bool {
	width, height := screen.Size()
	for y := 0; y < height; y++ {
		var line strings.Builder
		for x := 0; x < width; x++ {
			ch, _ := screen.GetCell(x, y)
			if ch == 0 {
				line.WriteByte(' ')
			} else {
				line.WriteRune(ch)
			}
		}
		if strings.Contains(line.String(), text) {
			return true
		}
	}
	return false
}

// TestSwitch_TwoWindowsDiffAndNormal verifies buffer switching when one window
// has a diff view and another has a normal buffer.
func TestSwitch_TwoWindowsDiffAndNormal(t *testing.T) {
	te := NewTestEditorWithContent(120, 30, "file1 content")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	// Create a second file in a new window
	file2 := buffer.NewBuffer("file2 content", "file2.go", "file2.go", config.IndentTabs, 4)
	te.buffers["file2.go"] = file2
	te.windows = append(te.windows, newWindow(file2))
	te.windows[1].addBufferToWindow(file2)
	te.recomputeWindowsLayout()

	// Open diff view on window 0
	te.activeWindow = 0
	setupDiffViewOnFile(te, origBuf)

	// Window 0 should have diff view, window 1 should not
	if _, isDiff := te.windows[0].View.(*DiffBufferView); !isDiff {
		t.Fatalf("Window 0 should have DiffBufferView")
	}
	if _, isDiff := te.windows[1].View.(*DiffBufferView); isDiff {
		t.Fatalf("Window 1 should NOT have DiffBufferView")
	}

	// Buffer list should show file1 with [diff] and file2 without
	entries := collectBufferDisplayEntries(te)
	foundDiff := false
	foundFile2 := false
	for _, e := range entries {
		if strings.Contains(e, origPath) && strings.Contains(e, "[diff]") {
			foundDiff = true
		}
		if strings.Contains(e, "file2.go") && !strings.Contains(e, "[diff]") {
			foundFile2 = true
		}
	}
	if !foundDiff {
		t.Errorf("Expected file1 with [diff] annotation, entries: %v", entries)
	}
	if !foundFile2 {
		t.Errorf("Expected file2 without [diff] annotation, entries: %v", entries)
	}

	// Switch to window 1 and verify buffer list is the same
	te.activateWindow(1)
	entries2 := collectBufferDisplayEntries(te)
	// Same assertions
	assertBufferListNotContains(t, entries2, "@HEAD")
	assertBufferListNotContains(t, entries2, "@current")
	assertBufferListContains(t, entries2, "file2.go")
}

// TestSwitch_ExitDiffThenReopenDiffView tests the cycle: open diff, exit,
// reopen diff. All buffers should remain accessible throughout.
func TestSwitch_ExitDiffThenReopenDiffView(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")

	// Open diff view
	setupDiffViewOnFile(te, origBuf)
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, "*Project Changes*")

	// Exit diff view
	te.exitDiffViewIfActive()
	entries = collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, origPath)
	assertBufferListContains(t, entries, "*Project Changes*")
	assertBufferListNotContains(t, entries, "[diff]")

	// Reopen diff view
	setupDiffViewOnFile(te, origBuf)
	entries = collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, "*Project Changes*")
	assertBufferListNotContains(t, entries, "@HEAD")
	assertBufferListNotContains(t, entries, "@current")

	// Switch to project changes — should work
	te.switchToBuffer(pcBuf)
	if te.CurrentBuffer().FilePath != "*Project Changes*" {
		t.Errorf("Expected *Project Changes*, got %s", te.CurrentBuffer().FilePath)
	}
}

// TestSwitch_OpenDiffFromInternalBuffer tests opening a diff view while
// currently viewing an internal buffer like *Project Changes*.
func TestSwitch_OpenDiffFromInternalBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2")
	origBuf := te.CurrentBuffer()

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")

	// Switch to project changes first
	te.switchToBuffer(pcBuf)
	if te.CurrentBuffer().FilePath != "*Project Changes*" {
		t.Fatalf("Expected *Project Changes*, got %s", te.CurrentBuffer().FilePath)
	}

	// Switch back to original and open diff
	te.switchToBuffer(origBuf)
	setupDiffViewOnFile(te, origBuf)

	// Project changes should still be accessible
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, "*Project Changes*")

	te.switchToBuffer(pcBuf)
	if te.CurrentBuffer().FilePath != "*Project Changes*" {
		t.Errorf("Expected *Project Changes* after switch from diff, got %s", te.CurrentBuffer().FilePath)
	}
}

// TestSwitch_OriginalBufferStaysInWindowAfterDiffSetup verifies the original
// file buffer remains in the window's Buffers list when a diff view is set up.
func TestSwitch_OriginalBufferStaysInWindowAfterDiffSetup(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "content")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	setupDiffViewOnFile(te, origBuf)

	// The original buffer should still be in window.Buffers
	win := te.windows[te.activeWindow]
	found := false
	for _, buf := range win.Buffers {
		if buf != nil && buf.FilePath == origPath {
			found = true
			break
		}
	}
	if !found {
		var paths []string
		for _, buf := range win.Buffers {
			if buf != nil {
				paths = append(paths, buf.FilePath)
			}
		}
		t.Errorf("Original buffer %s not found in window.Buffers: %v", origPath, paths)
	}

	// And it should appear in the buffer list
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, origPath)
}

// TestSwitch_DiffExitRestoresWindowWhenOnInternalBuffer verifies that exiting
// diff view works correctly when the active buffer is an internal buffer.
func TestSwitch_DiffExitRestoresWindowWhenOnInternalBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "content")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")

	setupDiffViewOnFile(te, origBuf)

	// Switch to project changes while diff view is active
	te.switchToBuffer(pcBuf)

	// Now exit diff view — should restore original file and project changes
	// should remain accessible
	te.exitDiffViewIfActive()

	// The diff view should be gone
	if te.isDiffViewActive() {
		t.Fatalf("Diff view should be inactive")
	}

	// Both original and project changes should be accessible
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, origPath)
	assertBufferListContains(t, entries, "*Project Changes*")

	// Should be able to switch between them
	te.switchToBuffer(origBuf)
	if te.CurrentBuffer().FilePath != origPath {
		t.Errorf("Expected %s, got %s", origPath, te.CurrentBuffer().FilePath)
	}
	te.switchToBuffer(pcBuf)
	if te.CurrentBuffer().FilePath != "*Project Changes*" {
		t.Errorf("Expected *Project Changes*, got %s", te.CurrentBuffer().FilePath)
	}
}

// TestSwitch_RapidSwitchingDoesNotCorruptState performs rapid buffer switches
// between diff and non-diff buffers to check for state corruption.
func TestSwitch_RapidSwitchingDoesNotCorruptState(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")
	msgBuf := createInternalBuffer(te, "*Messages*", "messages")

	setupDiffViewOnFile(te, origBuf)
	currBuf := te.buffers[origPath+"@current"]
	headBuf := te.buffers[origPath+"@HEAD"]

	// Rapid switching: mix of diff internal, normal, and internal buffers
	sequence := []*buffer.Buffer{
		pcBuf, currBuf, msgBuf, origBuf, headBuf, pcBuf, currBuf, origBuf, msgBuf, pcBuf,
	}

	for i, target := range sequence {
		te.switchToBuffer(target)
		if te.CurrentBuffer() != target {
			t.Fatalf("Step %d: expected %s, got %s", i, target.FilePath, te.CurrentBuffer().FilePath)
		}
		// Verify no nil buffer
		if te.CurrentBuffer() == nil {
			t.Fatalf("Step %d: current buffer is nil", i)
		}
		// Verify iterators are valid
		if te.CurrentBuffer().ViewIterator == nil {
			t.Fatalf("Step %d: ViewIterator is nil for %s", i, te.CurrentBuffer().FilePath)
		}
		if te.CurrentBuffer().CursorIterator == nil {
			t.Fatalf("Step %d: CursorIterator is nil for %s", i, te.CurrentBuffer().FilePath)
		}
	}

	// Final buffer list check
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, "*Project Changes*")
	assertBufferListContains(t, entries, "*Messages*")
	assertBufferListContains(t, entries, origPath)
	assertBufferListNotContains(t, entries, "@HEAD")
	assertBufferListNotContains(t, entries, "@current")
}

// TestSwitch_BufferListCurrentTag verifies the [current] tag correctly
// identifies the active buffer in the list, including when in diff view.
func TestSwitch_BufferListCurrentTag(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "content")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")

	// Without diff view: original file should be [current]
	entries := collectBufferDisplayEntriesWithCurrent(te)
	assertCurrentIs(t, entries, origPath)

	// Switch to PC: PC should be [current]
	te.switchToBuffer(pcBuf)
	entries = collectBufferDisplayEntriesWithCurrent(te)
	assertCurrentIs(t, entries, "*Project Changes*")

	// Switch back and enter diff view: the diff file should have [current]
	te.switchToBuffer(origBuf)
	setupDiffViewOnFile(te, origBuf)
	entries = collectBufferDisplayEntriesWithCurrent(te)
	// The current buffer is @current, but in the list it should resolve to
	// the original file path (since @current is filtered out)
	assertCurrentIs(t, entries, origPath)

	// Switch to PC while in diff: PC should be [current]
	te.switchToBuffer(pcBuf)
	entries = collectBufferDisplayEntriesWithCurrent(te)
	assertCurrentIs(t, entries, "*Project Changes*")
}

// collectBufferDisplayEntriesWithCurrent mirrors the buffer list logic and
// includes [current] markers. Uses the same resolution as showBufferListWithPrevious.
func collectBufferDisplayEntriesWithCurrent(te *TestEditor) []string {
	var entries []string
	seenBuffers := make(map[string]bool)

	diffRealPaths := make(map[string]bool)
	for _, window := range te.windows {
		if window == nil {
			continue
		}
		if dv, ok := window.View.(*DiffBufferView); ok && dv != nil {
			diffRealPaths[dv.RealPath] = true
		}
	}

	// Resolve current path through RealPath for diff buffers
	currentPath := ""
	if te.CurrentBuffer() != nil {
		currentPath = te.CurrentBuffer().FilePath
		if te.CurrentBuffer().DiffHints != nil && te.CurrentBuffer().RealPath != "" {
			currentPath = te.CurrentBuffer().RealPath
		}
	}

	for _, window := range te.windows {
		if window == nil {
			continue
		}
		for _, buf := range window.Buffers {
			if buf == nil {
				continue
			}
			path := buf.FilePath
			if path == "" || seenBuffers[path] {
				continue
			}
			if buf.DiffHints != nil {
				seenBuffers[path] = true
				continue
			}
			seenBuffers[path] = true

			name := buf.DisplayName()
			if diffRealPaths[path] {
				name += " [diff]"
			}
			if currentPath == path {
				name += " [current]"
			}
			entries = append(entries, name)
		}
	}

	for path, buf := range te.buffers {
		if seenBuffers[path] {
			continue
		}
		if buf.DiffHints != nil {
			seenBuffers[path] = true
			continue
		}
		seenBuffers[path] = true
		name := buf.DisplayName()
		if currentPath == path {
			name += " [current]"
		}
		entries = append(entries, name)
	}

	return entries
}

// assertCurrentIs checks that exactly one entry has [current] and it matches
// the expected path substring.
func assertCurrentIs(t *testing.T, entries []string, expectedSubstr string) {
	t.Helper()
	currentCount := 0
	var currentEntry string
	for _, e := range entries {
		if strings.Contains(e, "[current]") {
			currentCount++
			currentEntry = e
		}
	}
	if currentCount != 1 {
		t.Errorf("Expected exactly 1 [current] entry, got %d. Entries: %v", currentCount, entries)
		return
	}
	if !strings.Contains(currentEntry, expectedSubstr) {
		t.Errorf("Expected [current] entry to contain %q, got %q", expectedSubstr, currentEntry)
	}
}

// TestSwitch_DiffViewActivePanePreserved verifies that the active pane in a
// DiffBufferView is preserved after switching away and back.
func TestSwitch_DiffViewActivePanePreserved(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "content")
	origBuf := te.CurrentBuffer()

	pcBuf := createInternalBuffer(te, "*Project Changes*", "changes")

	dv := setupDiffViewOnFile(te, origBuf)

	// Default active pane is 1 (right/current)
	if dv.ActivePane != 1 {
		t.Fatalf("Expected initial active pane to be 1, got %d", dv.ActivePane)
	}

	// Switch active pane to left
	dv.ActivePane = 0

	// Switch away to project changes
	te.switchToBuffer(pcBuf)

	// Switch back to a diff buffer
	te.switchToBuffer(te.buffers[origBuf.FilePath+"@current"])

	// Active pane should still be 0 (left)
	if dv.ActivePane != 0 {
		t.Errorf("Expected active pane to remain 0 after round-trip switch, got %d", dv.ActivePane)
	}
}

// TestSwitch_MultipleFilesInDiffView tests that when multiple files exist and
// only one is in diff view, the others remain unaffected.
func TestSwitch_MultipleFilesInDiffView(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "file1 content")
	file1 := te.CurrentBuffer()
	file1Path := file1.FilePath

	file2 := buffer.NewBuffer("file2 content", "file2.go", "file2.go", config.IndentTabs, 4)
	te.buffers["file2.go"] = file2
	te.windows[te.activeWindow].addBufferToWindow(file2)

	file3 := buffer.NewBuffer("file3 content", "file3.go", "file3.go", config.IndentTabs, 4)
	te.buffers["file3.go"] = file3
	te.windows[te.activeWindow].addBufferToWindow(file3)

	// Switch back to file1 and open diff
	te.switchToBuffer(file1)
	setupDiffViewOnFile(te, file1)

	// All three files should be in the buffer list
	entries := collectBufferDisplayEntries(te)
	assertBufferListContains(t, entries, file1Path)
	assertBufferListContains(t, entries, "file2.go")
	assertBufferListContains(t, entries, "file3.go")

	// Only file1 should have [diff] annotation
	for _, e := range entries {
		if strings.Contains(e, "file2.go") && strings.Contains(e, "[diff]") {
			t.Errorf("file2.go should not have [diff] annotation: %q", e)
		}
		if strings.Contains(e, "file3.go") && strings.Contains(e, "[diff]") {
			t.Errorf("file3.go should not have [diff] annotation: %q", e)
		}
	}

	// Switch to file2, then file3 — should work normally
	te.switchToBuffer(file2)
	if te.CurrentBuffer() != file2 {
		t.Errorf("Expected file2, got %s", te.CurrentBuffer().FilePath)
	}
	te.switchToBuffer(file3)
	if te.CurrentBuffer() != file3 {
		t.Errorf("Expected file3, got %s", te.CurrentBuffer().FilePath)
	}
}

// TestSwitch_NextItemStaysInDiffView reproduces the scenario:
// 1. Open *Project Changes*
// 2. Navigate to a change (switch to a file)
// 3. Enter diff view
// 4. Press g n (next-item)
// Expected: stay in diff view, navigate within the diff hunks.
func TestSwitch_NextItemStaysInDiffView(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	origBuf := te.CurrentBuffer()

	// Create *Project Changes* with a link to a second file
	pcContent := "# Changes\n[file2.go change](file2.go#L5)\n[file3.go change](file3.go#L1)"
	pcBuf := createInternalBuffer(te, "*Project Changes*", pcContent)

	// Step 1: Switch to project changes — sets lastActiveSpecialBuffer
	te.switchToBuffer(pcBuf)
	if te.lastActiveSpecialBuffer != "*Project Changes*" {
		t.Fatalf("Expected lastActiveSpecialBuffer to be *Project Changes*, got %s", te.lastActiveSpecialBuffer)
	}

	// Step 2: Switch back to the file
	te.switchToBuffer(origBuf)

	// Step 3: Enter diff view
	setupDiffViewOnFile(te, origBuf)
	if !te.isDiffViewActive() {
		t.Fatalf("Expected diff view to be active after setup")
	}

	// Step 4: Execute next-item — should navigate within the diff, not exit it
	te.ExecuteCommand("next-item")
	if !te.isDiffViewActive() {
		t.Fatalf("Expected diff view to still be active after next-item, but it was exited")
	}

	// Step 5: Buffer list must NOT show @HEAD or @current
	entries := collectBufferDisplayEntries(te)
	assertBufferListNotContains(t, entries, "@HEAD")
	assertBufferListNotContains(t, entries, "@current")
}

// TestSwitch_BufferListNeverShowsDiffInternalsFullScenario exercises every step of
// the "project changes → navigate → diff → next change" scenario, checking the
// buffer list at each stage.
func TestSwitch_BufferListNeverShowsDiffInternalsFullScenario(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	origBuf := te.CurrentBuffer()
	origPath := origBuf.FilePath

	pcBuf := createInternalBuffer(te, "*Project Changes*", "# Changes\n[change](file.go#L1)")

	checkNoDiffInternals := func(step string) {
		t.Helper()
		entries := collectBufferDisplayEntries(te)
		for _, e := range entries {
			if strings.Contains(e, "@HEAD") || strings.Contains(e, "@current") {
				t.Errorf("%s: found diff internal in buffer list: %q (all: %v)", step, e, entries)
			}
		}
	}

	// 1. Initial state — just orig buffer
	checkNoDiffInternals("initial")

	// 2. Switch to project changes
	te.switchToBuffer(pcBuf)
	checkNoDiffInternals("after switch to PC")

	// 3. Switch back to file
	te.switchToBuffer(origBuf)
	checkNoDiffInternals("after switch back to file")

	// 4. Enter diff view
	setupDiffViewOnFile(te, origBuf)
	checkNoDiffInternals("after entering diff view")

	// 5. Navigate within diff (next-item)
	te.ExecuteCommand("next-item")
	checkNoDiffInternals("after next-item in diff")

	// 6. Switch to project changes while diff view is active
	te.switchToBuffer(pcBuf)
	checkNoDiffInternals("after switch to PC from diff")

	// 7. Switch back to a diff internal buffer
	if currBuf := te.buffers[origPath+"@current"]; currBuf != nil {
		te.switchToBuffer(currBuf)
	}
	checkNoDiffInternals("after switch to @current buffer")

	// 8. Switch to project changes again
	te.switchToBuffer(pcBuf)
	checkNoDiffInternals("after second switch to PC")

	// 9. Exit diff view
	te.exitDiffViewIfActive()
	checkNoDiffInternals("after exiting diff view")

	// 10. Verify origBuf is still accessible
	if _, exists := te.buffers[origPath]; !exists {
		t.Fatalf("Original buffer %s was removed from buffers after diff exit", origPath)
	}
}
