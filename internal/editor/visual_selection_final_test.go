package editor

import (
	"testing"
)

func TestVisualSelectionFinal(t *testing.T) {
	t.Run("visual selection is visually distinct", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "The quick brown fox jumps over the lazy dog")

		// Get theme colors
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		currentLineStyle := te.config.Theme.GetStyle(te.config.Theme.CurrentLine)

		_, selBg, _ := selectionStyle.Decompose()
		_, currentLineBg, _ := currentLineStyle.Decompose()

		// Move to start of "quick"
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}

		// Enter visual mode at position 4
		te.ExecuteCommand("enter-visual-mode")

		// Select through "quick brown" (10 more characters)
		for i := 0; i < 10; i++ {
			te.ExecuteCommand("right")
		}

		// Render
		te.Editor.render()

		// Find where the text starts (after line number)
		line := te.mockScreen.GetLine(0)
		textStart := -1
		for i := 0; i < len(line); i++ {
			if line[i] == 'T' && i+2 < len(line) && line[i+1] == 'h' && line[i+2] == 'e' {
				textStart = i
				break
			}
		}

		if textStart == -1 {
			t.Fatalf("Could not find text in rendered line")
		}

		// Check backgrounds
		// "The " should have current line background
		for i := 0; i < 4; i++ {
			_, bg, _ := te.mockScreen.GetCellStyle(textStart+i, 0)
			if bg != currentLineBg {
				t.Errorf("Character %d should have current line bg, got %v", i, bg)
			}
		}

		// "quick brown" should have selection background
		for i := 4; i < 15; i++ {
			_, bg, _ := te.mockScreen.GetCellStyle(textStart+i, 0)
			if bg != selBg {
				t.Errorf("Character %d should have selection bg, got %v", i, bg)
			}
		}

		// Rest should have current line background again
		_, bg, _ := te.mockScreen.GetCellStyle(textStart+15, 0)
		if bg != currentLineBg {
			t.Errorf("Character after selection should have current line bg, got %v", bg)
		}

		// Calculate color differences to ensure visibility
		selR := (selBg >> 16) & 0xFF
		selG := (selBg >> 8) & 0xFF
		selB := selBg & 0xFF

		clR := (currentLineBg >> 16) & 0xFF
		clG := (currentLineBg >> 8) & 0xFF
		clB := currentLineBg & 0xFF

		// Calculate total color difference
		diffR := int(selR) - int(clR)
		diffG := int(selG) - int(clG)
		diffB := int(selB) - int(clB)

		if diffR < 0 {
			diffR = -diffR
		}
		if diffG < 0 {
			diffG = -diffG
		}
		if diffB < 0 {
			diffB = -diffB
		}

		totalDiff := diffR + diffG + diffB

		t.Logf("Color difference between selection and current line:")
		t.Logf("  Selection RGB: (%d, %d, %d)", selR, selG, selB)
		t.Logf("  CurrentLine RGB: (%d, %d, %d)", clR, clG, clB)
		t.Logf("  Total difference: %d", totalDiff)

		// Ensure there's enough visual difference
		if totalDiff < 30 {
			t.Errorf("Selection color is not distinct enough from current line (diff=%d, want >= 30)", totalDiff)
		}
	})

	t.Run("visual line mode highlights entire lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3\nLine 4")

		// Get selection style
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Move to line 2
		te.ExecuteCommand("down")

		// Enter visual line mode
		te.ExecuteCommand("enter-visual-line-mode")

		// Select down to line 3
		te.ExecuteCommand("down")

		// Render
		te.Editor.render()

		// Check that lines 2 and 3 are fully highlighted
		for lineIdx := 1; lineIdx <= 2; lineIdx++ {
			line := te.mockScreen.GetLine(lineIdx)

			// Find where "Line" text starts
			lineStart := -1
			for i := 0; i < len(line); i++ {
				if i+3 < len(line) && line[i] == 'L' && line[i+1] == 'i' && line[i+2] == 'n' && line[i+3] == 'e' {
					lineStart = i
					break
				}
			}

			if lineStart != -1 {
				// Check that the entire line text has selection background
				for i := 0; i < 6; i++ { // "Line X"
					_, bg, _ := te.mockScreen.GetCellStyle(lineStart+i, lineIdx)
					if bg != selBg {
						t.Errorf("Line %d char %d should have selection bg", lineIdx+1, i)
					}
				}
			}
		}
	})

	t.Run("selection works across multiple lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "First line\nSecond line\nThird line")

		// Get selection style
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Start visual mode at "line" in first line
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}

		te.ExecuteCommand("enter-visual-mode")

		// Select to "Second" in second line
		te.ExecuteCommand("down")
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}

		// Render
		te.Editor.render()

		// Check first line - "line" should be selected
		line0 := te.mockScreen.GetLine(0)
		lineStart := -1
		for i := 0; i < len(line0); i++ {
			if i+3 < len(line0) && line0[i] == 'l' && line0[i+1] == 'i' && line0[i+2] == 'n' && line0[i+3] == 'e' {
				lineStart = i
				break
			}
		}

		if lineStart != -1 {
			for i := 0; i < 4; i++ {
				_, bg, _ := te.mockScreen.GetCellStyle(lineStart+i, 0)
				if bg != selBg {
					t.Errorf("First line 'line' char %d should be selected", i)
				}
			}
		}

		// Check second line - "Second" should be selected
		line1 := te.mockScreen.GetLine(1)
		secondStart := -1
		for i := 0; i < len(line1); i++ {
			if i+5 < len(line1) && line1[i] == 'S' && line1[i+1] == 'e' {
				secondStart = i
				break
			}
		}

		if secondStart != -1 {
			for i := 0; i < 6; i++ {
				_, bg, _ := te.mockScreen.GetCellStyle(secondStart+i, 1)
				if bg != selBg {
					t.Errorf("Second line 'Second' char %d should be selected", i)
				}
			}
		}
	})
}
