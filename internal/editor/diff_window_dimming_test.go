package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestDiffView_WindowDimmingDisabled verifies that window dimming is disabled when diff view is active
func TestDiffView_WindowDimmingDisabled(t *testing.T) {
	// Create test editor with two windows
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	te.buffer.FilePath = "test.go"
	
	// Create second window
	te.createSecondWindow()
	if len(te.windows) < 2 {
		t.Fatalf("Expected at least 2 windows, got %d", len(te.windows))
	}
	
	// Test 1: Without diff view, dimming should be active for inactive windows
	t.Run("normal_view_has_dimming", func(t *testing.T) {
		// Set active window to 0
		te.activeWindow = 0
		
		// Check that isDiffViewActive returns false
		if te.isDiffViewActive() {
			t.Error("isDiffViewActive should return false when no diff buffers are present")
		}
		
		// Manually test the dimming logic (simulate what happens in editor.go)
		diffViewActive := te.isDiffViewActive()
		
		// Window 0 (active) should not be dimmed
		dim0 := !diffViewActive && (0 != te.activeWindow)
		if dim0 {
			t.Error("Active window should not be dimmed")
		}
		
		// Window 1 (inactive) should be dimmed
		dim1 := !diffViewActive && (1 != te.activeWindow)
		if !dim1 {
			t.Error("Inactive window should be dimmed in normal view")
		}
	})
	
	// Test 2: Set up diff view and verify dimming is disabled
	t.Run("diff_view_disables_dimming", func(t *testing.T) {
		// Create diff buffers manually (simulating what StartDiffViewForCurrentBuffer does)
		headBufferName := "test.go@HEAD"
		currentBufferName := "test.go@current"
		
		// Create head buffer (read-only)
		headBuffer := buffer.NewBuffer("line1\noriginal line2\nline3", headBufferName, config.IndentTabs, 4)
		headBuffer.IsReadOnly = true
		te.buffers[headBufferName] = headBuffer
		
		// Create current buffer copy
		currentBuffer := buffer.NewBuffer("line1\nmodified line2\nline3", currentBufferName, config.IndentTabs, 4)
		currentBuffer.IsReadOnly = false
		te.buffers[currentBufferName] = currentBuffer
		
		// Assign diff buffers to windows
		te.windows[0].Buffer = headBuffer    // HEAD on left
		te.windows[1].Buffer = currentBuffer // Current on right
		te.activeWindow = 1 // Active on right side
		
		// Check that isDiffViewActive returns true
		if !te.isDiffViewActive() {
			t.Error("isDiffViewActive should return true when diff buffers are present")
		}
		
		// Test the dimming logic with diff view active
		diffViewActive := te.isDiffViewActive()
		
		// Both windows should not be dimmed, regardless of which is active
		dim0 := !diffViewActive && (0 != te.activeWindow)
		dim1 := !diffViewActive && (1 != te.activeWindow)
		
		if dim0 {
			t.Error("Left window should not be dimmed in diff view")
		}
		if dim1 {
			t.Error("Right window should not be dimmed in diff view")
		}
		
		// Test with different active window
		te.activeWindow = 0 // Switch to left side
		
		dim0 = !diffViewActive && (0 != te.activeWindow)
		dim1 = !diffViewActive && (1 != te.activeWindow)
		
		if dim0 {
			t.Error("Left window should not be dimmed in diff view (when active)")
		}
		if dim1 {
			t.Error("Right window should not be dimmed in diff view (when left is active)")
		}
	})
	
	// Test 3: Verify dimming is re-enabled when diff view is closed
	t.Run("dimming_reenabled_after_diff_view", func(t *testing.T) {
		// Remove diff buffers from windows (simulating closing diff view)
		te.windows[0].Buffer = nil
		te.windows[1].Buffer = te.buffer // Back to normal buffer
		te.activeWindow = 1
		
		// Check that isDiffViewActive returns false again
		if te.isDiffViewActive() {
			t.Error("isDiffViewActive should return false after diff view is closed")
		}
		
		// Dimming should be active again for inactive windows
		diffViewActive := te.isDiffViewActive()
		
		// Window 0 (inactive, empty) should be dimmed
		dim0 := !diffViewActive && (0 != te.activeWindow)
		if !dim0 {
			t.Error("Inactive empty window should be dimmed after diff view is closed")
		}
		
		// Window 1 (active) should not be dimmed
		dim1 := !diffViewActive && (1 != te.activeWindow)
		if dim1 {
			t.Error("Active window should not be dimmed")
		}
	})
}

// TestIsDiffViewActive tests the isDiffViewActive helper function specifically
func TestIsDiffViewActive(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "test content")
	
	// Test with no windows
	te.windows = nil
	if te.isDiffViewActive() {
		t.Error("isDiffViewActive should return false with no windows")
	}
	
	// Test with empty windows
	te.createSecondWindow()
	if len(te.windows) >= 2 {
		te.windows[0].Buffer = nil
		te.windows[1].Buffer = nil
	}
	if te.isDiffViewActive() {
		t.Error("isDiffViewActive should return false with empty windows")
	}
	
	// Test with normal buffers
	normalBuffer := buffer.NewBuffer("normal content", "normal.go", config.IndentTabs, 4)
	if len(te.windows) >= 2 {
		te.windows[0].Buffer = normalBuffer
		te.windows[1].Buffer = normalBuffer
		if te.isDiffViewActive() {
			t.Error("isDiffViewActive should return false with normal buffers")
		}
	}
	
	// Test with one diff buffer
	diffBuffer := buffer.NewBuffer("diff content", "test.go@HEAD", config.IndentTabs, 4)
	if len(te.windows) >= 2 {
		te.windows[0].Buffer = diffBuffer
		te.windows[1].Buffer = normalBuffer
		if !te.isDiffViewActive() {
			t.Error("isDiffViewActive should return true with one diff buffer")
		}
	}
	
	// Test with both diff buffers
	diffBuffer2 := buffer.NewBuffer("diff content 2", "test.go@current", config.IndentTabs, 4)
	if len(te.windows) >= 2 {
		te.windows[0].Buffer = diffBuffer
		te.windows[1].Buffer = diffBuffer2
		if !te.isDiffViewActive() {
			t.Error("isDiffViewActive should return true with both diff buffers")
		}
	}
	
	// Test with mixed diff and normal buffers in multiple windows
	te.createSecondWindow() // Add third window
	if len(te.windows) >= 3 {
		te.windows[2].Buffer = normalBuffer
		if !te.isDiffViewActive() {
			t.Error("isDiffViewActive should return true when any window has diff buffer")
		}
	}
}