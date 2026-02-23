package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestDiffBufferClosing verifies that closing one diff buffer closes both
func TestDiffBufferClosing(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "initial content")

	// Create diff buffers
	headBufferName := "test.go@HEAD"
	currentBufferName := "test.go@current"

	headBuffer := buffer.NewBuffer("head content", headBufferName, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer("current content", currentBufferName, currentBufferName, config.IndentTabs, 4)

	// Create dummy diff line info
	diffResult := &buffer.DiffResult{
		LeftLines: []buffer.DiffLine{
			{Content: "head content", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
		},
		RightLines: []buffer.DiffLine{
			{Content: "current content", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
		},
	}

	// Add buffers to editor
	te.buffers[headBufferName] = headBuffer
	te.buffers[currentBufferName] = currentBuffer
	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on a single window
	te.SetupDiffView(headBuffer, currentBuffer, "test.go")

	t.Run("helper_functions", func(t *testing.T) {
		// Test DiffHints-based detection
		if headBuffer.DiffHints == nil {
			t.Error("Should identify HEAD buffer as diff buffer via DiffHints")
		}
		if currentBuffer.DiffHints == nil {
			t.Error("Should identify current buffer as diff buffer via DiffHints")
		}
		normalBuf := buffer.NewBuffer("normal", "normal.go", "normal.go", config.IndentTabs, 4)
		if normalBuf.DiffHints != nil {
			t.Error("Should not identify normal buffer as diff buffer")
		}

		// Test that DiffBufferView references both buffers
		win := te.windows[te.activeWindow]
		dv, ok := win.View.(*DiffBufferView)
		if !ok {
			t.Fatal("Expected DiffBufferView on active window")
		}
		if dv.LeftBuffer != headBuffer {
			t.Error("DiffBufferView.LeftBuffer should reference HEAD buffer")
		}
		if dv.RightBuffer != currentBuffer {
			t.Error("DiffBufferView.RightBuffer should reference current buffer")
		}
	})

	t.Run("close_diff_buffer_closes_both", func(t *testing.T) {
		// Verify both buffers exist before closing
		if te.buffers[headBufferName] == nil {
			t.Fatal("HEAD buffer should exist before closing")
		}
		if te.buffers[currentBufferName] == nil {
			t.Fatal("Current buffer should exist before closing")
		}

		// Verify diff line info exists
		if len(headBuffer.DiffHints) == 0 {
			t.Error("HEAD buffer should have diff line info")
		}
		if len(currentBuffer.DiffHints) == 0 {
			t.Error("Current buffer should have diff line info")
		}

		// Close one of the diff buffers
		err := te.closeBuffer([]string{})
		if err != nil {
			t.Fatalf("Failed to close diff buffer: %v", err)
		}

		// Verify both buffers are removed
		if te.buffers[headBufferName] != nil {
			t.Error("HEAD buffer should be removed after closing diff buffer")
		}
		if te.buffers[currentBufferName] != nil {
			t.Error("Current buffer should be removed after closing diff buffer")
		}

		// Verify diff line info is cleaned up
		if len(headBuffer.DiffHints) > 0 {
			t.Error("HEAD buffer diff line info should be cleaned up")
		}
		if len(currentBuffer.DiffHints) > 0 {
			t.Error("Current buffer diff line info should be cleaned up")
		}

		// Verify editor still has a valid buffer (should create an unnamed buffer)
		if te.CurrentBuffer() == nil {
			t.Error("Editor should still have a buffer after closing diff buffers")
		}
		if len(te.buffers) == 0 {
			t.Error("Editor should still have at least one buffer")
		}
	})
}

// TestNormalBufferClosing verifies that normal buffers close normally without affecting other buffers
func TestNormalBufferClosing(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")

	// Create a normal buffer and a diff buffer pair
	normalBuffer := buffer.NewBuffer("normal content", "normal.go", "normal.go", config.IndentTabs, 4)
	headBuffer := buffer.NewBuffer("head content", "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
	currentBuffer := buffer.NewBuffer("current content", "test.go@current", "test.go@current", config.IndentTabs, 4)

	// Add all buffers to editor
	te.buffers["normal.go"] = normalBuffer
	te.buffers["test.go@HEAD"] = headBuffer
	te.buffers["test.go@current"] = currentBuffer

	// Add some diff line info to the diff buffers
	diffResult := &buffer.DiffResult{
		LeftLines: []buffer.DiffLine{
			{Content: "head content", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
		},
		RightLines: []buffer.DiffLine{
			{Content: "current content", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
		},
	}
	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer.DiffHints = diffResult.RightLines

	// Switch to the normal buffer
	te.switchToBuffer(normalBuffer)

	// Close the normal buffer
	err := te.closeBuffer([]string{})
	if err != nil {
		t.Fatalf("Failed to close normal buffer: %v", err)
	}

	// Verify only the normal buffer is removed
	if te.buffers["normal.go"] != nil {
		t.Error("Normal buffer should be removed")
	}
	if te.buffers["test.go@HEAD"] == nil {
		t.Error("HEAD buffer should still exist")
	}
	if te.buffers["test.go@current"] == nil {
		t.Error("Current buffer should still exist")
	}

	// Verify diff line info for diff buffers is preserved
	if len(headBuffer.DiffHints) == 0 {
		t.Error("HEAD buffer diff line info should be preserved")
	}
	if len(currentBuffer.DiffHints) == 0 {
		t.Error("Current buffer diff line info should be preserved")
	}
}

// TestDiffBufferClosingWithMultipleWindows tests diff buffer closing with multiple windows
func TestDiffBufferClosingWithMultipleWindows(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")

	// Create diff buffers
	headBuffer := buffer.NewBuffer("head content", "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
	currentBuffer := buffer.NewBuffer("current content", "test.go@current", "test.go@current", config.IndentTabs, 4)
	normalBuffer := buffer.NewBuffer("normal content", "normal.go", "normal.go", config.IndentTabs, 4)

	// Add buffers to editor
	te.buffers["test.go@HEAD"] = headBuffer
	te.buffers["test.go@current"] = currentBuffer
	te.buffers["normal.go"] = normalBuffer

	// Create diff line info
	diffResult := &buffer.DiffResult{
		LeftLines: []buffer.DiffLine{
			{Content: "head content", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
		},
		RightLines: []buffer.DiffLine{
			{Content: "current content", LineType: buffer.DiffLineContext, LeftLine: 0, RightLine: 0, IsFillerLine: false},
		},
	}
	headBuffer.DiffHints = diffResult.LeftLines
	currentBuffer.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on window 0, normal buffer on window 1
	te.SetupDiffView(headBuffer, currentBuffer, "test.go")

	// Add a second window with a normal buffer
	normalWin := newWindow(normalBuffer)
	te.windows = append(te.windows, normalWin)
	te.recomputeWindowsLayout()

	// Close the diff buffer
	err := te.closeBuffer([]string{})
	if err != nil {
		t.Fatalf("Failed to close diff buffer: %v", err)
	}

	// Verify both diff buffers are removed
	if te.buffers["test.go@HEAD"] != nil {
		t.Error("HEAD buffer should be removed")
	}
	if te.buffers["test.go@current"] != nil {
		t.Error("Current buffer should be removed")
	}

	// Verify normal buffer is preserved
	if te.buffers["normal.go"] == nil {
		t.Error("Normal buffer should be preserved")
	}

	// Verify we still have windows
	if len(te.windows) == 0 {
		t.Error("Should still have at least one window")
	}

	// Verify the editor has a valid buffer
	if te.CurrentBuffer() == nil {
		t.Error("Editor should have a valid buffer")
	}

	// The important thing is that the editor is still functional and has some buffer
	if len(te.buffers) == 0 {
		t.Error("Editor should have at least one buffer remaining")
	}

	// Verify that it's not one of the closed diff buffers
	if te.CurrentBuffer().FilePath == "test.go@HEAD" || te.CurrentBuffer().FilePath == "test.go@current" {
		t.Error("Editor should not be using a closed diff buffer")
	}
}
