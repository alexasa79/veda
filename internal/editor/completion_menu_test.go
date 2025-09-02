package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/widgets"
)

func TestCompletionMenuAppearsInActiveWindow(t *testing.T) {
	// Create editor with initial content
	te := NewTestEditorWithContent(80, 24, "Hello world\nTest line")
	
	// Create second window and second buffer
	te.createSecondWindow()
	
	// Create a second buffer with different content
	secondBuffer := buffer.NewBuffer("Different content\nIn second window", "test2.txt", config.IndentTabs, 4)
	te.buffers["test2.txt"] = secondBuffer
	
	// Set up windows side by side (left and right)
	te.windows[0].Buffer = te.CurrentBuffer()
	te.windows[0].X = 0
	te.windows[0].Y = 0
	te.windows[0].Width = 40
	te.windows[0].Height = 24
	
	te.windows[1].Buffer = secondBuffer
	te.windows[1].X = 40
	te.windows[1].Y = 0
	te.windows[1].Width = 40
	te.windows[1].Height = 24
	
	// Test with window 0 active
	t.Run("completion_in_first_window", func(t *testing.T) {
		te.activeWindow = 0
		te.mode = commands.InsertMode
		
		// Position cursor at column 5 in first window
		te.CurrentBuffer().CursorX = 5
		
		// Manually show completion menu with items
		items := []widgets.CompletionItem{
			{Label: "world", InsertText: "world"},
			{Label: "wonderful", InsertText: "wonderful"},
		}
		te.completionMenu.SetItems(items)
		
		// Calculate position using getCursorScreenPosition
		menuX, menuY := te.getCursorScreenPosition()
		te.completionMenu.SetPosition(menuX, menuY+1) // Position below cursor
		te.completionMenu.Show()
		
		// Get menu position
		actualX, _ := te.completionMenu.GetPosition()
		
		// Menu should be in the first window (X < 40)
		if actualX >= 40 {
			t.Errorf("Completion menu X position should be in first window (< 40), got %d", actualX)
		}
		
		// Hide menu for next test
		te.completionMenu.Hide()
	})
	
	// Test with window 1 active
	t.Run("completion_in_second_window", func(t *testing.T) {
		te.activeWindow = 1
		te.mode = commands.InsertMode
		
		// Switch to second buffer
		te.switchToBuffer(secondBuffer)
		
		// Position cursor at column 9 in second window
		te.CurrentBuffer().CursorX = 9
		
		// Manually show completion menu with items
		items := []widgets.CompletionItem{
			{Label: "content", InsertText: "content"},
			{Label: "context", InsertText: "context"},
		}
		te.completionMenu.SetItems(items)
		
		// Calculate position using getCursorScreenPosition
		menuX, menuY := te.getCursorScreenPosition()
		te.completionMenu.SetPosition(menuX, menuY+1) // Position below cursor
		te.completionMenu.Show()
		
		// Get menu position
		actualX, actualY := te.completionMenu.GetPosition()
		
		// Menu should be in the second window (X >= 40)
		if actualX < 40 {
			t.Errorf("Completion menu X position should be in second window (>= 40), got %d", actualX)
		}
		
		// Verify Y position is reasonable
		if actualY < 0 || actualY >= 24 {
			t.Errorf("Completion menu Y position out of bounds: %d", actualY)
		}
	})
}