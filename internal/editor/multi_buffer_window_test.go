package editor

import (
	"testing"
	"veda/internal/buffer"
)

// TestMultiBufferWindowBasics tests the fundamental multi-buffer window functionality
func TestMultiBufferWindowBasics(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create buffers
	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	// Add to global registry
	te.buffers["file1.txt"] = buf1
	te.buffers["file2.txt"] = buf2

	// Create initial window with buf1
	te.windows = []*Window{newWindow(buf1)}
	te.activeWindow = 0
	te.SetCurrentBuffer(buf1)

	// Verify initial state
	if len(te.windows[0].Buffers) != 1 {
		t.Errorf("Expected 1 buffer in window, got %d", len(te.windows[0].Buffers))
	}
	if te.windows[0].Buffer != buf1 {
		t.Error("Expected buf1 to be active buffer")
	}
	if te.windows[0].Buffers[0] != buf1 {
		t.Error("Expected buf1 to be in buffers list")
	}
}

// TestMoveBufferToNewWindow tests moving a buffer to create a new window
func TestMoveBufferToNewWindow(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create buffer and set up initial state
	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	te.buffers["file1.txt"] = buf1
	te.windows = []*Window{newWindow(buf1)}
	te.activeWindow = 0
	te.SetCurrentBuffer(buf1)

	// Move buffer right to create new window
	err := te.ExecuteCommand("move-buffer-right")
	if err != nil {
		t.Fatalf("Failed to execute move-buffer-right: %v", err)
	}

	// Verify we now have 2 windows
	if len(te.windows) != 2 {
		t.Errorf("Expected 2 windows after move, got %d", len(te.windows))
	}

	// Verify first window is now empty
	if te.windows[0].Buffer != nil {
		t.Error("Expected first window to be empty after move")
	}
	if len(te.windows[0].Buffers) != 0 {
		t.Errorf("Expected first window to have 0 buffers, got %d", len(te.windows[0].Buffers))
	}

	// Verify second window has the buffer
	if te.windows[1].Buffer != buf1 {
		t.Error("Expected second window to have buf1 as active buffer")
	}
	if len(te.windows[1].Buffers) != 1 {
		t.Errorf("Expected second window to have 1 buffer, got %d", len(te.windows[1].Buffers))
	}
	if te.windows[1].Buffers[0] != buf1 {
		t.Error("Expected buf1 to be in second window's buffer list")
	}

	// Verify active window switched
	if te.activeWindow != 1 {
		t.Errorf("Expected active window to be 1, got %d", te.activeWindow)
	}
}

// TestMoveBufferToExistingWindow tests the core issue - moving buffers should not displace others
func TestMoveBufferToExistingWindow(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create buffers
	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	te.buffers["file1.txt"] = buf1
	te.buffers["file2.txt"] = buf2

	// Set up initial state: buf1 in window 0, buf2 in window 1
	win1 := newWindow(buf1)
	win2 := newWindow(buf2)
	te.windows = []*Window{win1, win2}
	te.activeWindow = 1 // Start from window with buf2
	te.SetCurrentBuffer(buf2)

	// Move buf2 from window 1 to window 0 (where buf1 is)
	err := te.ExecuteCommand("move-buffer-left")
	if err != nil {
		t.Fatalf("Failed to execute move-buffer-left: %v", err)
	}

	// CRITICAL: buf1 should still be in window 0, just not active
	if len(te.windows[0].Buffers) != 2 {
		t.Errorf("Expected window 0 to have 2 buffers, got %d", len(te.windows[0].Buffers))
	}

	// buf1 should still be in the buffers list
	found := false
	for _, b := range te.windows[0].Buffers {
		if b == buf1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("buf1 should still be in window 0's buffer list")
	}

	// buf2 should now be the active buffer in window 0
	if te.windows[0].Buffer != buf2 {
		t.Error("Expected buf2 to be active buffer in window 0")
	}

	// buf2 should also be in the buffers list
	found = false
	for _, b := range te.windows[0].Buffers {
		if b == buf2 {
			found = true
			break
		}
	}
	if !found {
		t.Error("buf2 should be in window 0's buffer list")
	}

	// Window 1 should now be empty (buf2 was removed)
	if te.windows[1].Buffer != nil {
		t.Error("Expected window 1 to be empty after buf2 was moved")
	}
	if len(te.windows[1].Buffers) != 0 {
		t.Errorf("Expected window 1 to have 0 buffers, got %d", len(te.windows[1].Buffers))
	}

	// Active window should have switched to window 0
	if te.activeWindow != 0 {
		t.Errorf("Expected active window to be 0, got %d", te.activeWindow)
	}
}

// TestBufferPersistenceAfterMovement tests that buffers persist when other buffers are moved around
func TestBufferPersistenceAfterMovement(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create three buffers
	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf3 := buffer.NewBuffer("content3", "file3.txt", "file3.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	te.buffers["file1.txt"] = buf1
	te.buffers["file2.txt"] = buf2
	te.buffers["file3.txt"] = buf3

	// Set up: buf1 in window 0, buf2 in window 1
	win1 := newWindow(buf1)
	win2 := newWindow(buf2)
	te.windows = []*Window{win1, win2}
	te.activeWindow = 0
	te.SetCurrentBuffer(buf1)

	// Move buf3 to window 0 (should add it to buf1's window)
	te.SetCurrentBuffer(buf3) // Simulate having buf3 active in some way
	te.windows[0].addBufferToWindow(buf3)

	// Now window 0 should have buf1 and buf3
	if len(te.windows[0].Buffers) != 2 {
		t.Errorf("Expected window 0 to have 2 buffers, got %d", len(te.windows[0].Buffers))
	}

	// Move buf3 to window 1
	te.activeWindow = 0
	te.SetCurrentBuffer(buf3)
	err := te.ExecuteCommand("move-buffer-right")
	if err != nil {
		t.Fatalf("Failed to move buf3 right: %v", err)
	}

	// buf1 should still be in window 0
	if len(te.windows[0].Buffers) != 1 {
		t.Errorf("Expected window 0 to have 1 buffer after moving buf3, got %d", len(te.windows[0].Buffers))
	}
	if te.windows[0].Buffers[0] != buf1 {
		t.Error("buf1 should still be in window 0 after buf3 was moved")
	}
	if te.windows[0].Buffer != buf1 {
		t.Error("buf1 should be active in window 0 after buf3 was moved")
	}

	// Window 1 should now have buf2 and buf3
	if len(te.windows[1].Buffers) != 2 {
		t.Errorf("Expected window 1 to have 2 buffers, got %d", len(te.windows[1].Buffers))
	}

	// buf3 should be active in window 1
	if te.windows[1].Buffer != buf3 {
		t.Error("buf3 should be active in window 1")
	}
}

// TestRemoveLastBufferFromWindow tests what happens when the last buffer is removed from a window
func TestRemoveLastBufferFromWindow(t *testing.T) {
	te := NewTestEditor(80, 24)

	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	te.buffers["file1.txt"] = buf1

	win1 := newWindow(buf1)
	te.windows = []*Window{win1, newWindow(nil)} // Second window starts empty
	te.activeWindow = 0
	te.SetCurrentBuffer(buf1)

	// Move the only buffer from window 0 to window 1
	err := te.ExecuteCommand("move-buffer-right")
	if err != nil {
		t.Fatalf("Failed to move buffer right: %v", err)
	}

	// Window 0 should now be completely empty
	if te.windows[0].Buffer != nil {
		t.Error("Expected window 0 to have no active buffer")
	}
	if len(te.windows[0].Buffers) != 0 {
		t.Errorf("Expected window 0 to have 0 buffers, got %d", len(te.windows[0].Buffers))
	}

	// Window 1 should have the buffer
	if te.windows[1].Buffer != buf1 {
		t.Error("Expected window 1 to have buf1 as active buffer")
	}
	if len(te.windows[1].Buffers) != 1 {
		t.Errorf("Expected window 1 to have 1 buffer, got %d", len(te.windows[1].Buffers))
	}
}

// TestBufferOrderingInWindow tests that buffers are added to the end of the list
func TestBufferOrderingInWindow(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create three buffers
	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf3 := buffer.NewBuffer("content3", "file3.txt", "file3.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	win1 := newWindow(buf1)

	// Add buffers in order
	win1.addBufferToWindow(buf2)
	win1.addBufferToWindow(buf3)

	// Check order
	if len(win1.Buffers) != 3 {
		t.Errorf("Expected 3 buffers, got %d", len(win1.Buffers))
	}

	if win1.Buffers[0] != buf1 {
		t.Error("Expected buf1 to be first in buffer list")
	}
	if win1.Buffers[1] != buf2 {
		t.Error("Expected buf2 to be second in buffer list")
	}
	if win1.Buffers[2] != buf3 {
		t.Error("Expected buf3 to be third in buffer list")
	}

	// Active buffer should be the last added (buf3)
	if win1.Buffer != buf3 {
		t.Error("Expected buf3 to be the active buffer")
	}
}

// TestActiveBufferSwitchingOnRemoval tests that when active buffer is removed, another becomes active
func TestActiveBufferSwitchingOnRemoval(t *testing.T) {
	te := NewTestEditor(80, 24)

	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf3 := buffer.NewBuffer("content3", "file3.txt", "file3.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	win1 := newWindow(buf1)
	win1.addBufferToWindow(buf2)
	win1.addBufferToWindow(buf3) // buf3 becomes active

	// Remove the active buffer (buf3)
	win1.removeBufferFromWindow(buf3)

	// buf2 should now be active (last in remaining list)
	if win1.Buffer != buf2 {
		t.Error("Expected buf2 to become active after buf3 was removed")
	}

	// Should have 2 buffers left
	if len(win1.Buffers) != 2 {
		t.Errorf("Expected 2 buffers after removal, got %d", len(win1.Buffers))
	}

	// Remove buf2 (now active)
	win1.removeBufferFromWindow(buf2)

	// buf1 should now be active
	if win1.Buffer != buf1 {
		t.Error("Expected buf1 to become active after buf2 was removed")
	}

	if len(win1.Buffers) != 1 {
		t.Errorf("Expected 1 buffer after second removal, got %d", len(win1.Buffers))
	}
}

// TestAddSameBufferTwice tests that adding the same buffer twice just makes it active
func TestAddSameBufferTwice(t *testing.T) {
	te := NewTestEditor(80, 24)

	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	win1 := newWindow(buf1)
	win1.addBufferToWindow(buf2) // buf2 becomes active

	// Add buf1 again (should just make it active, not duplicate)
	win1.addBufferToWindow(buf1)

	// Should still have only 2 buffers
	if len(win1.Buffers) != 2 {
		t.Errorf("Expected 2 buffers after adding duplicate, got %d", len(win1.Buffers))
	}

	// buf1 should now be active
	if win1.Buffer != buf1 {
		t.Error("Expected buf1 to be active after re-adding it")
	}
}

// TestMoveBufferCenter tests the move-buffer-center command
func TestMoveBufferCenter(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create 3 windows with buffers
	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf3 := buffer.NewBuffer("content3", "file3.txt", "file3.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	te.buffers["file1.txt"] = buf1
	te.buffers["file2.txt"] = buf2
	te.buffers["file3.txt"] = buf3

	win1 := newWindow(buf1)
	win2 := newWindow(buf2)
	win3 := newWindow(buf3)
	te.windows = []*Window{win1, win2, win3}
	te.activeWindow = 2 // Start from window 2
	te.SetCurrentBuffer(buf3)

	// Move buf3 to center (window 1)
	err := te.ExecuteCommand("move-buffer-center")
	if err != nil {
		t.Fatalf("Failed to execute move-buffer-center: %v", err)
	}

	// Window 1 should now have buf2 and buf3
	if len(te.windows[1].Buffers) != 2 {
		t.Errorf("Expected window 1 to have 2 buffers, got %d", len(te.windows[1].Buffers))
	}

	// buf3 should be active in window 1
	if te.windows[1].Buffer != buf3 {
		t.Error("Expected buf3 to be active in window 1")
	}

	// buf2 should still be in window 1
	found := false
	for _, b := range te.windows[1].Buffers {
		if b == buf2 {
			found = true
			break
		}
	}
	if !found {
		t.Error("buf2 should still be in window 1 after buf3 was moved there")
	}

	// Window 2 should now be empty
	if te.windows[2].Buffer != nil {
		t.Error("Expected window 2 to be empty after buf3 was moved")
	}
	if len(te.windows[2].Buffers) != 0 {
		t.Errorf("Expected window 2 to have 0 buffers, got %d", len(te.windows[2].Buffers))
	}
}

// TestNoGlobalBuffersAfterMovement tests that regular buffers don't become "global" after movement
func TestNoGlobalBuffersAfterMovement(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create buffers
	buf1 := buffer.NewBuffer("content1", "file1.txt", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2 := buffer.NewBuffer("content2", "file2.txt", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

	te.buffers["file1.txt"] = buf1
	te.buffers["file2.txt"] = buf2

	// Set up initial state
	win1 := newWindow(buf1)
	win2 := newWindow(buf2)
	te.windows = []*Window{win1, win2}
	te.activeWindow = 1
	te.SetCurrentBuffer(buf2)

	// Move buf2 to window 0 (should add to buf1's window)
	err := te.ExecuteCommand("move-buffer-left")
	if err != nil {
		t.Fatalf("Failed to move buffer left: %v", err)
	}

	// Now check what findWindowContainingBuffer returns
	window1Idx := te.findWindowContainingBuffer("file1.txt")
	window2Idx := te.findWindowContainingBuffer("file2.txt")

	// Both buffers should be found in windows (not -1 which would indicate "global")
	if window1Idx == -1 {
		t.Error("buf1 should not be global - should be found in a window")
	}
	if window2Idx == -1 {
		t.Error("buf2 should not be global - should be found in a window")
	}

	// Both should be in window 0 now
	if window1Idx != 0 {
		t.Errorf("Expected buf1 to be in window 0, got %d", window1Idx)
	}
	if window2Idx != 0 {
		t.Errorf("Expected buf2 to be in window 0, got %d", window2Idx)
	}
}
