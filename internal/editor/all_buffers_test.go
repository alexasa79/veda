package editor

import (
	"testing"
	"veda/internal/buffer"
)

func TestFindWindowContainingBuffer(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create some buffers
	buf1 := buffer.NewBuffer("content1", "artifacts/file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "artifacts/file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf3 := buffer.NewBuffer("content3", "artifacts/file3.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	// Add buffers to the global buffer list
	te.buffers["artifacts/file1.txt"] = buf1
	te.buffers["artifacts/file2.txt"] = buf2
	te.buffers["artifacts/file3.txt"] = buf3

	// Create windows and add buffers to them using the proper multi-buffer system
	win1 := newWindow(buf1)
    win1.addBufferToWindow(buf2)  // Add buf2 to window with buf1
	win2 := newWindow(buf3) // buf3 in window 2

	te.windows = []*Window{win1, win2}

    // Test finding windows: both buf1 and buf2 should be in window 0
    windowIdx1 := te.findWindowContainingBuffer("artifacts/file1.txt")
    if windowIdx1 != 0 {
        t.Errorf("Expected file1.txt to be in window 0, got %d", windowIdx1)
    }

    windowIdx2 := te.findWindowContainingBuffer("artifacts/file2.txt")
    if windowIdx2 != 0 {
        t.Errorf("Expected file2.txt to be in window 0, got %d", windowIdx2)
    }

	windowIdx3 := te.findWindowContainingBuffer("artifacts/file3.txt")
	if windowIdx3 != 1 {
		t.Errorf("Expected file3.txt to be in window 1, got %d", windowIdx3)
	}

	// Test non-existent buffer
	windowIdxNone := te.findWindowContainingBuffer("artifacts/nonexistent.txt")
	if windowIdxNone != -1 {
		t.Errorf("Expected non-existent buffer to return -1, got %d", windowIdxNone)
	}
}

func TestShowBuffersWithMultipleWindows(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create some buffers
	buf1 := buffer.NewBuffer("content1", "artifacts/file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "artifacts/file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf3 := buffer.NewBuffer("content3", "artifacts/file3.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	// Add buffers to the global buffer list
	te.buffers["artifacts/file1.txt"] = buf1
	te.buffers["artifacts/file2.txt"] = buf2
	te.buffers["artifacts/file3.txt"] = buf3

	// Create windows and add buffers to them using the proper multi-buffer system
	win1 := newWindow(buf1)
    win1.addBufferToWindow(buf2)  // Add buf2 to window with buf1
	win2 := newWindow(buf3) // buf3 in window 2

	te.windows = []*Window{win1, win2}
	te.activeWindow = 0
	te.SetCurrentBuffer(buf1)

	// Test that showBuffers doesn't crash and returns no error
	err := te.showBuffers([]string{})
	if err != nil {
		t.Errorf("showBuffers returned error: %v", err)
	}

	// Test showBufferListWithPrevious as well
	te.previousBuffer = buf2
	err = te.showBufferListWithPrevious([]string{})
	if err != nil {
		t.Errorf("showBufferListWithPrevious returned error: %v", err)
	}

	// The actual buffer list display is tested through the UI overlay system,
	// which would require more complex setup. This test ensures the basic
	// logic doesn't crash.
}

func TestWindowSwitchingIntegration(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create some buffers
	buf1 := buffer.NewBuffer("content1", "artifacts/file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "artifacts/file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf3 := buffer.NewBuffer("content3", "artifacts/file3.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	// Add buffers to the global buffer list
	te.buffers["artifacts/file1.txt"] = buf1
	te.buffers["artifacts/file2.txt"] = buf2
	te.buffers["artifacts/file3.txt"] = buf3

	// Create windows and add buffers to them using the proper multi-buffer system
	win1 := newWindow(buf1)
    win1.addBufferToWindow(buf2)  // Add buf2 to window with buf1
	win2 := newWindow(buf3) // buf3 in window 2

	te.windows = []*Window{win1, win2}
	te.activeWindow = 0
	te.SetCurrentBuffer(buf1)

	// Verify initial state
	if te.activeWindow != 0 {
		t.Errorf("Expected active window to be 0, got %d", te.activeWindow)
	}
	if te.CurrentBuffer().FilePath != "artifacts/file1.txt" {
		t.Errorf("Expected current buffer to be file1.txt, got %s", te.CurrentBuffer().FilePath)
	}

	// Test finding buffer in different window
	targetWindowIdx := te.findWindowContainingBuffer("artifacts/file3.txt")
	if targetWindowIdx != 1 {
		t.Errorf("Expected file3.txt to be in window 1, got %d", targetWindowIdx)
	}

	// Simulate activating window 1 (containing buf3)
	te.activateWindow(1)
	if te.activeWindow != 1 {
		t.Errorf("Expected active window to be 1 after activation, got %d", te.activeWindow)
	}
	if te.CurrentBuffer().FilePath != "artifacts/file3.txt" {
		t.Errorf("Expected current buffer to be file3.txt after window switch, got %s", te.CurrentBuffer().FilePath)
	}
}
