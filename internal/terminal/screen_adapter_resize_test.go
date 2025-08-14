package terminal

import (
	"testing"
	"veda/internal/ui"
)

func TestScreenAdapterResizeCells(t *testing.T) {
	t.Run("resizeCells method works correctly", func(t *testing.T) {
		// Create a screen adapter with initial size
		adapter := &ScreenAdapter{
			cells:         make([][]cell, 10),
			defaultStyle:  ui.StyleDefault,
			renderedLines: make([]string, 10),
			lineHashes:    make([]uint64, 10),
		}
		
		// Initialize cell buffer
		for i := range adapter.cells {
			adapter.cells[i] = make([]cell, 20)
			for j := range adapter.cells[i] {
				adapter.cells[i][j] = cell{ch: ' ', style: ui.StyleDefault}
			}
		}
		
		// Set some content
		adapter.cells[5][10] = cell{ch: 'T', style: ui.StyleDefault}
		adapter.cells[9][19] = cell{ch: 'E', style: ui.StyleDefault}
		
		// Resize to larger dimensions
		adapter.resizeCells(30, 15)
		
		// Verify new size
		if len(adapter.cells) != 15 {
			t.Errorf("Expected 15 rows after resize, got %d", len(adapter.cells))
		}
		if len(adapter.cells[0]) != 30 {
			t.Errorf("Expected 30 columns after resize, got %d", len(adapter.cells[0]))
		}
		
		// Verify old content was preserved
		if adapter.cells[5][10].ch != 'T' {
			t.Error("Content should be preserved after resize")
		}
		if adapter.cells[9][19].ch != 'E' {
			t.Error("Content at edge should be preserved after resize")
		}
		
		// Verify new cells are initialized properly
		if adapter.cells[14][29].ch != ' ' {
			t.Error("New cells should be initialized with space character")
		}
		
		// Test shrinking
		adapter.resizeCells(15, 8)
		
		// Verify new size
		if len(adapter.cells) != 8 {
			t.Errorf("Expected 8 rows after shrinking, got %d", len(adapter.cells))
		}
		if len(adapter.cells[0]) != 15 {
			t.Errorf("Expected 15 columns after shrinking, got %d", len(adapter.cells[0]))
		}
		
		// Verify preserved content that fits
		if adapter.cells[5][10].ch != 'T' {
			t.Error("Content within bounds should still be preserved after shrinking")
		}
	})
}