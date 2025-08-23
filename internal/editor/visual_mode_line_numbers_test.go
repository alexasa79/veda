package editor

import (
	"fmt"
	"strings"
	"testing"
)

func TestVisualModeLineNumberRendering(t *testing.T) {
	t.Run("visual mode from first character should not affect line numbers", func(t *testing.T) {
		// Create editor with multiple lines to see line numbers
		content := "# TODO List\nFirst item\nSecond item\nThird item"
		te := NewTestEditorWithContent(80, 24, content)
		te.CurrentBuffer().FilePath = "TODO.md"

		// Initialize markdown highlighter
        if te.CurrentBuffer() != nil && te.CurrentBuffer().Highlighter != nil {
            te.CurrentBuffer().Highlighter.SetLanguageFromFile(te.CurrentBuffer().FilePath)
		}

		// Get theme styles
		lineNumberStyle := te.config.Theme.GetStyle(te.config.Theme.LineNumber)
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, lineNumBg, _ := lineNumberStyle.Decompose()
		_, selBg, _ := selectionStyle.Decompose()

		// Ensure cursor is at position 0,0 (first character)
		te.SetCursorPosition(0, 0)

		// Enter visual mode from the very first character
		te.ExecuteCommand("enter-visual-mode")

		// Select a few characters
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}

		// Force render
		te.drawEditor()

		// Check each visible line
		for lineIdx := 0; lineIdx < 4; lineIdx++ {
			screenLine := te.mockScreen.GetLine(lineIdx)
			t.Logf("Line %d: %q", lineIdx, screenLine)

			// Find where the line number ends (look for space after number)
			lineNumEnd := -1
			for i := 0; i < len(screenLine) && i < 10; i++ {
				if screenLine[i] >= '0' && screenLine[i] <= '9' {
					// Found a digit, look for space after last digit
					for j := i; j < len(screenLine); j++ {
						if screenLine[j] == ' ' {
							lineNumEnd = j
							break
						}
						if screenLine[j] < '0' || screenLine[j] > '9' {
							break
						}
					}
					break
				}
			}

			if lineNumEnd > 0 {
				// Check background color of line number characters
				for i := 0; i < lineNumEnd; i++ {
					_, bg, _ := te.mockScreen.GetCellStyle(i, lineIdx)
					if bg == selBg {
						t.Errorf("Line %d: Line number character at position %d has selection background. Should have line number background.", lineIdx+1, i)
					}
					if bg != lineNumBg {
						t.Errorf("Line %d: Line number character at position %d has wrong background. Got %v, want %v", lineIdx+1, i, bg, lineNumBg)
					}
				}
			}
		}

		// Additionally check that the actual selected text has selection background
		// First line should have "# TOD" selected (positions after line number)
		firstLine := te.mockScreen.GetLine(0)
		hashPos := strings.Index(firstLine, "#")
		if hashPos != -1 {
			// Check first 5 characters of content have selection bg
			for i := 0; i < 5 && hashPos+i < len(firstLine); i++ {
				_, bg, _ := te.mockScreen.GetCellStyle(hashPos+i, 0)
				if bg != selBg {
					t.Errorf("Selected character %d should have selection background", i)
				}
			}
		}
	})

	t.Run("visual mode line numbers across multiple lines", func(t *testing.T) {
		// Test with visual mode spanning multiple lines
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
		te := NewTestEditorWithContent(80, 24, content)

		// Get styles
		lineNumberStyle := te.config.Theme.GetStyle(te.config.Theme.LineNumber)
		_, lineNumBg, _ := lineNumberStyle.Decompose()

		// Start visual mode from line 1
		te.ExecuteCommand("enter-visual-mode")

		// Select down to line 3
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")

		// Render
		te.drawEditor()

		// Check that line numbers for all lines retain their style
		for lineIdx := 0; lineIdx < 5; lineIdx++ {
			line := te.mockScreen.GetLine(lineIdx)
			// Find line number position (usually first character)
			if len(line) > 0 && line[0] >= '1' && line[0] <= '5' {
				_, bg, _ := te.mockScreen.GetCellStyle(0, lineIdx)
				if bg != lineNumBg {
					t.Errorf("Line %d: Line number should have line number background, got %v", lineIdx+1, bg)
				}
			}
		}
	})

	t.Run("visual line mode should not affect line numbers", func(t *testing.T) {
		content := "First line\nSecond line\nThird line"
		te := NewTestEditorWithContent(80, 24, content)

		// Get styles
		selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
		_, selBg, _ := selectionStyle.Decompose()

		// Enter visual line mode
		te.ExecuteCommand("enter-visual-line-mode")

		// Select down one line
		te.ExecuteCommand("down")

		// Render
		te.drawEditor()

		// Check first 3 lines
		for lineIdx := 0; lineIdx < 3; lineIdx++ {
			line := te.mockScreen.GetLine(lineIdx)
			t.Logf("Visual line mode - Line %d: %q", lineIdx, line)

			// Check line number background
			if len(line) > 0 {
				_, bg, _ := te.mockScreen.GetCellStyle(0, lineIdx)
				if bg == selBg {
					t.Errorf("Line %d: Line number has selection background in visual line mode", lineIdx+1)
				}
			}
		}
	})
}

// Helper to debug what's being rendered
func TestDebugLineNumberArea(t *testing.T) {
	t.Run("debug line number rendering", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "# Header\nLine 2\nLine 3")
		te.CurrentBuffer().FilePath = "test.md"

		// Set up markdown
        if te.CurrentBuffer() != nil && te.CurrentBuffer().Highlighter != nil {
            te.CurrentBuffer().Highlighter.SetLanguageFromFile(te.CurrentBuffer().FilePath)
		}

		// Enter visual mode at start
		te.ExecuteCommand("enter-visual-mode")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")

		// Render
		te.drawEditor()

		// Debug output
		t.Logf("Line number width: %d", te.CurrentBuffer().LineNumberWidth)

		// Print first 10 characters of each line with their backgrounds
		for lineIdx := 0; lineIdx < 3; lineIdx++ {
			line := te.mockScreen.GetLine(lineIdx)
			t.Logf("Line %d content: %q", lineIdx, line[:min(20, len(line))])

			// Show background colors for first 10 positions
			bgInfo := ""
			for x := 0; x < 10 && x < len(line); x++ {
				r, bg, _ := te.mockScreen.GetCellStyle(x, lineIdx)
				bgInfo += fmt.Sprintf("[%c:%v]", r, bg)
			}
			t.Logf("Line %d backgrounds: %s", lineIdx, bgInfo)
		}
	})
}

func TestLineNumberSelectionBug(t *testing.T) {
	t.Run("selection starting at position 0 affects line numbers", func(t *testing.T) {
		// This test specifically reproduces the bug where selecting from
		// the very first character causes line numbers to get selection background
		te := NewTestEditorWithContent(80, 24, "First line\nSecond line")

		// Verify we're at position 0,0
		x, y := te.GetCursorPosition()
		if x != 0 || y != 0 {
			t.Fatalf("Expected cursor at (0,0), got (%d,%d)", x, y)
		}

		// Enter visual mode from position 0,0
		te.ExecuteCommand("enter-visual-mode")

		// Select just one character
		te.ExecuteCommand("right")

		// Get selection range
		start, end, hasSelection := te.getSelectionRange()
		t.Logf("Selection range: start=%d, end=%d, hasSelection=%v", start, end, hasSelection)

		// The selection should be from byte 0 to byte 2
		if start != 0 {
			t.Errorf("Selection should start at byte 0, got %d", start)
		}

		// Render
		te.drawEditor()

		// Get line number style
		lineNumberStyle := te.config.Theme.GetStyle(te.config.Theme.LineNumber)
		_, lineNumBg, _ := lineNumberStyle.Decompose()

		// Check first character (should be '1' for line number)
		r, bg, _ := te.mockScreen.GetCellStyle(0, 0)
		t.Logf("First character: '%c' with background %v", r, bg)

		if r == '1' && bg != lineNumBg {
			t.Errorf("Line number '1' has wrong background. Got %v, want %v", bg, lineNumBg)
		}
	})
}
