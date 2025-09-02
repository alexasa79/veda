package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestWindowFullscreenToggle(t *testing.T) {
	// Create editor with 80x24 screen
	te := NewTestEditor(80, 24)
	te.config = config.New()
	te.minActiveWindowWidth = 0 // Disable minimum width for testing
	
	// Create three windows with buffers
	firstBuffer := buffer.NewBuffer("First buffer content", "test1.txt", config.IndentTabs, 4)
	secondBuffer := buffer.NewBuffer("Second buffer content", "test2.txt", config.IndentTabs, 4)
	thirdBuffer := buffer.NewBuffer("Third buffer content", "test3.txt", config.IndentTabs, 4)
	
	te.buffers["test1.txt"] = firstBuffer
	te.buffers["test2.txt"] = secondBuffer
	te.buffers["test3.txt"] = thirdBuffer
	
	// Set up three windows
	te.windows = []*Window{
		newWindow(firstBuffer),
		newWindow(secondBuffer),
		newWindow(thirdBuffer),
	}
	te.activeWindow = 1 // Middle window is active
	
	t.Run("normal_layout_with_three_windows", func(t *testing.T) {
		// Initially fullscreen should be off
		if te.windowFullscreen {
			t.Error("Fullscreen should be off initially")
		}
		
		// Compute normal layout
		te.recomputeWindowsLayout()
		
		// Each window should get roughly equal width (80/3 = 26,27,27)
		expectedWidths := []int{27, 27, 26} // Distribution with remainder
		for i, win := range te.windows {
			if win.Width != expectedWidths[i] {
				t.Errorf("Window %d width: expected %d, got %d", i, expectedWidths[i], win.Width)
			}
		}
	})
	
	t.Run("toggle_fullscreen_on", func(t *testing.T) {
		// Toggle fullscreen on
		te.ExecuteCommand("toggle-window-fullscreen")
		
		if !te.windowFullscreen {
			t.Error("Fullscreen should be on after toggle")
		}
		
		// Active window (1) should get most of the space
		// Other windows should get 10 characters each
		if te.windows[0].Width != 10 {
			t.Errorf("Window 0 width in fullscreen: expected 10, got %d", te.windows[0].Width)
		}
		
		// Active window gets 80 - 20 = 60 columns
		if te.windows[1].Width != 60 {
			t.Errorf("Active window width in fullscreen: expected 60, got %d", te.windows[1].Width)
		}
		
		if te.windows[2].Width != 10 {
			t.Errorf("Window 2 width in fullscreen: expected 10, got %d", te.windows[2].Width)
		}
	})
	
	t.Run("switch_window_maintains_fullscreen", func(t *testing.T) {
		// Switch to window 0
		te.activeWindow = 0
		te.recomputeWindowsLayout()
		
		// Now window 0 should be large
		if te.windows[0].Width != 60 {
			t.Errorf("New active window width: expected 60, got %d", te.windows[0].Width)
		}
		
		if te.windows[1].Width != 10 {
			t.Errorf("Window 1 width: expected 10, got %d", te.windows[1].Width)
		}
		
		if te.windows[2].Width != 10 {
			t.Errorf("Window 2 width: expected 10, got %d", te.windows[2].Width)
		}
	})
	
	t.Run("toggle_fullscreen_off", func(t *testing.T) {
		// Toggle fullscreen off
		te.ExecuteCommand("toggle-window-fullscreen")
		
		if te.windowFullscreen {
			t.Error("Fullscreen should be off after second toggle")
		}
		
		// Should return to equal distribution
		expectedWidths := []int{27, 27, 26}
		for i, win := range te.windows {
			if win.Width != expectedWidths[i] {
				t.Errorf("Window %d width after toggle off: expected %d, got %d", 
					i, expectedWidths[i], win.Width)
			}
		}
	})
}

func TestWindowFullscreenWithTwoWindows(t *testing.T) {
	te := NewTestEditor(80, 24)
	te.config = config.New()
	te.minActiveWindowWidth = 0 // Disable minimum width for testing
	
	// Create two windows
	firstBuffer := buffer.NewBuffer("First", "test1.txt", config.IndentTabs, 4)
	secondBuffer := buffer.NewBuffer("Second", "test2.txt", config.IndentTabs, 4)
	
	te.buffers["test1.txt"] = firstBuffer
	te.buffers["test2.txt"] = secondBuffer
	
	te.windows = []*Window{
		newWindow(firstBuffer),
		newWindow(secondBuffer),
	}
	te.activeWindow = 0
	
	t.Run("two_windows_fullscreen", func(t *testing.T) {
		// Toggle fullscreen
		te.ExecuteCommand("toggle-window-fullscreen")
		
		// Active window should get 80 - 10 = 70 columns
		if te.windows[0].Width != 70 {
			t.Errorf("Active window width: expected 70, got %d", te.windows[0].Width)
		}
		
		if te.windows[1].Width != 10 {
			t.Errorf("Inactive window width: expected 10, got %d", te.windows[1].Width)
		}
	})
}

func TestWindowFullscreenSingleWindow(t *testing.T) {
	te := NewTestEditor(80, 24)
	te.config = config.New()
	te.minActiveWindowWidth = 0 // Disable minimum width for testing
	
	// Create single window
	buf := buffer.NewBuffer("Content", "test.txt", config.IndentTabs, 4)
	te.buffers["test.txt"] = buf
	te.windows = []*Window{newWindow(buf)}
	te.activeWindow = 0
	
	t.Run("single_window_no_effect", func(t *testing.T) {
		// Toggle should have no effect with single window
		te.ExecuteCommand("toggle-window-fullscreen")
		
		// Fullscreen state should not change
		if te.windowFullscreen {
			t.Error("Fullscreen should not activate with single window")
		}
		
		// Window should still be full width
		te.recomputeWindowsLayout()
		if te.windows[0].Width != 80 {
			t.Errorf("Single window width: expected 80, got %d", te.windows[0].Width)
		}
	})
}

func TestWindowFullscreenSmallScreen(t *testing.T) {
	// Test with very small screen width
	te := NewTestEditor(30, 24)
	te.config = config.New()
	te.minActiveWindowWidth = 0 // Disable minimum width for testing
	
	// Create three windows
	te.windows = []*Window{
		newWindow(nil),
		newWindow(nil),
		newWindow(nil),
	}
	te.activeWindow = 1
	
	t.Run("small_screen_fullscreen", func(t *testing.T) {
		// Toggle fullscreen
		te.windowFullscreen = true
		te.recomputeWindowsLayout()
		
		// With 30 columns and 3 windows:
		// Non-active windows get 10 each = 20
		// Active window gets 30 - 20 = 10 (but should be at least 20)
		// So it becomes: active=20, others share 10 columns
		
		// The active window should get at least 20 columns
		if te.windows[1].Width < 20 {
			t.Errorf("Active window width on small screen: expected at least 20, got %d", 
				te.windows[1].Width)
		}
		
		// Other windows share the remaining space
		totalOtherWidth := te.windows[0].Width + te.windows[2].Width
		if totalOtherWidth != 10 {
			t.Errorf("Total width of non-active windows: expected 10, got %d", 
				totalOtherWidth)
		}
	})
}