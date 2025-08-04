package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
	"veda/internal/ui"
)

func TestVisualModeRendering(t *testing.T) {
	t.Run("visual mode selection is highlighted with theme style", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "Hello World\nThis is a test\nThird line")
		
		// Get the selection style from theme
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()
		
		// Enter visual mode
		te.ExecuteCommand("enter-visual-mode")
		
		// Verify we're in visual mode
		if te.mode != commands.VisualMode {
			t.Errorf("Expected visual mode, got %v", te.mode)
		}
		
		// Move right to select "Hello"
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}
		
		// Force a render
		te.drawEditor()
		
		// Get the rendered content for the first line
		// The line should have line numbers, so we need to find where "Hello" starts
		screenLine := te.mockScreen.GetLine(0)
		
		// Find where "Hello" starts (after line number)
		helloStart := strings.Index(screenLine, "Hello")
		if helloStart == -1 {
			t.Fatalf("Could not find 'Hello' in rendered line: %q", screenLine)
		}
		
		// Check that the first 5 characters of "Hello" have the selection background
		for i := 0; i < 5; i++ {
			x := helloStart + i
			_, bg, _ := te.mockScreen.GetCellStyle(x, 0)
			if bg != selBg {
				t.Errorf("Character %d of 'Hello' does not have selection background. Expected %v, got %v", i, selBg, bg)
			}
		}
		
		// Check that characters after "Hello" don't have selection background
		if helloStart+6 < len(screenLine) {
			_, bg, _ := te.mockScreen.GetCellStyle(helloStart+6, 0)
			if bg == selBg {
				t.Errorf("Character after 'Hello' should not have selection background")
			}
		}
	})
	
	t.Run("visual line mode highlights entire lines", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")
		
		// Get the selection style from theme
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()
		
		// Enter visual line mode
		te.ExecuteCommand("enter-visual-line-mode")
		
		// Move down to select two lines
		te.ExecuteCommand("down")
		
		// Force a render
		te.drawEditor()
		
		// Check that entire first line has selection background
		line1 := te.mockScreen.GetLine(0)
		line1Start := strings.Index(line1, "Line 1")
		if line1Start != -1 {
			for i := 0; i < 6; i++ { // Check "Line 1"
				_, bg, _ := te.mockScreen.GetCellStyle(line1Start + i, 0)
				if bg != selBg {
					t.Errorf("Line 1 character %d does not have selection background", i)
				}
			}
		}
		
		// Check that entire second line has selection background
		line2 := te.mockScreen.GetLine(1)
		line2Start := strings.Index(line2, "Line 2")
		if line2Start != -1 {
			for i := 0; i < 6; i++ { // Check "Line 2"
				_, bg, _ := te.mockScreen.GetCellStyle(line2Start + i, 1)
				if bg != selBg {
					t.Errorf("Line 2 character %d does not have selection background", i)
				}
			}
		}
		
		// Check that third line doesn't have selection background
		line3 := te.mockScreen.GetLine(2)
		line3Start := strings.Index(line3, "Line 3")
		if line3Start != -1 {
			_, bg, _ := te.mockScreen.GetCellStyle(line3Start, 2)
			if bg == selBg {
				t.Errorf("Line 3 should not have selection background")
			}
		}
	})
	
	t.Run("visual mode selection updates as cursor moves", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "ABCDEFGHIJ")
		
		// Get the selection style from theme
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()
		
		// Enter visual mode
		te.ExecuteCommand("enter-visual-mode")
		
		// Move right twice to select "ABC"
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		
		// Force a render
		te.drawEditor()
		
		// Find where text starts
		screenLine := te.mockScreen.GetLine(0)
		textStart := strings.Index(screenLine, "ABCDEFGHIJ")
		if textStart == -1 {
			t.Fatalf("Could not find text in rendered line")
		}
		
		// Check that A, B, C have selection background
		for i, ch := range []string{"A", "B", "C"} {
			_, bg, _ := te.mockScreen.GetCellStyle(textStart + i, 0)
			if bg != selBg {
				t.Errorf("Character '%s' should have selection background", ch)
			}
		}
		
		// Check that D doesn't have selection background
		_, bg, _ := te.mockScreen.GetCellStyle(textStart + 3, 0)
		if bg == selBg {
			t.Errorf("Character 'D' should not have selection background")
		}
	})
}

// Helper method to get cell style from MockScreen
func (m *MockScreen) GetCellStyle(x, y int) (rune, ui.Color, ui.AttrMask) {
	if y < 0 || y >= m.height || x < 0 || x >= m.width {
		return 0, ui.ColorDefault, 0
	}
	
	// Check if row exists
	row, exists := m.cells[y]
	if !exists {
		return 0, ui.ColorDefault, 0
	}
	
	// Check if cell exists
	cell, exists := row[x]
	if !exists {
		return 0, ui.ColorDefault, 0
	}
	
	fg, bg, attrs := cell.Style.Decompose()
	_ = fg // Not used in this context
	return cell.Rune, bg, attrs
}