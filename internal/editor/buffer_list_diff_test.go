package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestBufferList_DiffViewShowsSingleEntry verifies that when a DiffBufferView
// is active, the buffer list shows the original file with a [diff] annotation
// and does NOT show separate @HEAD/@current entries.
func TestBufferList_DiffViewShowsSingleEntry(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")

	// Remember the original buffer path — this is the real file.
	origPath := te.CurrentBuffer().FilePath

	// Create diff buffers
	diffResult := computeDiff("line1\nline2\nline3", "line1\nmodified\nline3")
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), origPath+"@HEAD", origPath+"@HEAD", config.IndentTabs, 4)
	head.IsReadOnly = true
	head.DiffHints = diffResult.LeftLines
	head.RealPath = origPath
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), origPath+"@current", origPath+"@current", config.IndentTabs, 4)
	curr.DiffHints = diffResult.RightLines
	curr.RealPath = origPath

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr

	// Set up DiffBufferView (RealPath = origPath)
	te.SetupDiffView(head, curr, origPath)

	// Simulate what showBuffers does: collect buffer display entries using
	// the same algorithm as the real code.
	bufferDisplays := collectBufferDisplayEntries(te)

	// Count diff-related entries
	diffAnnotated := 0
	headEntries := 0
	currentEntries := 0
	for _, d := range bufferDisplays {
		if strings.Contains(d, "[diff]") {
			diffAnnotated++
		}
		if strings.Contains(d, "@HEAD") {
			headEntries++
		}
		if strings.Contains(d, "@current") {
			currentEntries++
		}
	}

	if diffAnnotated != 1 {
		t.Errorf("Expected 1 entry with [diff] annotation, got %d. Entries: %v", diffAnnotated, bufferDisplays)
	}
	if headEntries != 0 {
		t.Errorf("Expected 0 @HEAD entries, got %d. Entries: %v", headEntries, bufferDisplays)
	}
	if currentEntries != 0 {
		t.Errorf("Expected 0 @current entries, got %d. Entries: %v", currentEntries, bufferDisplays)
	}
}

// TestBufferList_IndependentOfViewState verifies that the buffer list is the
// same regardless of whether a DiffBufferView is currently active or not.
// Diff internal buffers should never appear.
func TestBufferList_IndependentOfViewState(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	origPath := te.CurrentBuffer().FilePath
	origBuf := te.CurrentBuffer()

	// Add a second file
	file2 := buffer.NewBuffer("other content", "other.go", "other.go", config.IndentTabs, 4)
	te.buffers["other.go"] = file2
	te.windows[te.activeWindow].addBufferToWindow(file2)

	// Create diff buffers
	diffResult := computeDiff("line1\nline2\nline3", "line1\nmodified\nline3")
	head := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), origPath+"@HEAD", origPath+"@HEAD", config.IndentTabs, 4)
	head.IsReadOnly = true
	head.DiffHints = diffResult.LeftLines
	head.RealPath = origPath
	curr := buffer.NewBuffer(createBufferContent(diffResult.RightLines), origPath+"@current", origPath+"@current", config.IndentTabs, 4)
	curr.DiffHints = diffResult.RightLines
	curr.RealPath = origPath

	te.buffers[head.FilePath] = head
	te.buffers[curr.FilePath] = curr
	te.SetupDiffView(head, curr, origPath)

	// Collect entries while diff view is active
	entriesInDiffView := collectBufferDisplayEntries(te)

	// Now switch away from diff view (make a SingleBufferView active)
	te.windows[te.activeWindow].View = &SingleBufferView{Buf: origBuf}
	te.switchToBuffer(origBuf)

	entriesOutsideDiffView := collectBufferDisplayEntries(te)

	// Both should have the same set of entries (no @HEAD/@current in either)
	for _, entries := range [][]string{entriesInDiffView, entriesOutsideDiffView} {
		for _, d := range entries {
			if strings.Contains(d, "@HEAD") || strings.Contains(d, "@current") {
				t.Errorf("Diff internal buffer should never appear in buffer list: %q (all: %v)", d, entries)
			}
		}
	}

	// Both should show "other.go"
	for label, entries := range map[string][]string{"in_diff_view": entriesInDiffView, "outside_diff_view": entriesOutsideDiffView} {
		found := false
		for _, d := range entries {
			if strings.Contains(d, "other.go") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: expected other.go in buffer list, got: %v", label, entries)
		}
	}
}

// collectBufferDisplayEntries mirrors the buffer collection logic in showBuffers.
// It iterates Views (not Buffers) to match the production code.
func collectBufferDisplayEntries(te *TestEditor) []string {
	var entries []string
	seenViews := make(map[BufferView]bool)
	seenPaths := make(map[string]bool)

	for _, window := range te.windows {
		if window == nil {
			continue
		}
		for _, view := range window.Views {
			if view == nil || seenViews[view] {
				continue
			}
			seenViews[view] = true
			name := view.DisplayName()
			seenPaths[view.ViewPath()] = true
			for _, b := range view.AllBuffers() {
				if b != nil {
					seenPaths[b.FilePath] = true
				}
			}
			entries = append(entries, name)
		}
	}

	for path, buf := range te.buffers {
		if seenPaths[path] {
			continue
		}
		seenPaths[path] = true
		entries = append(entries, buf.DisplayName())
	}

	return entries
}
