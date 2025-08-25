package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestDiffLineNumbers verifies that diff buffers show correct original line numbers
func TestDiffLineNumbers(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")
	
	// Create content with line insertions and deletions to test line number mapping
	originalContent := "line 1\nline 2\nline 3\nline 4\nline 5"
	modifiedContent := "line 1\nnew line A\nline 2\nline 4\nnew line B\nline 5"
	
	// originalContent:     modifiedContent:
	// 1: line 1            1: line 1         (context)
	// 2: line 2            -: new line A     (added - filler on left)
	// 3: line 3            2: line 2         (context, but line 3 was deleted)
	// 4: line 4            4: line 4         (context)
	// 5: line 5            -: new line B     (added - filler on left)
	//                      5: line 5         (context)
	
	diffResult := computeDiff(originalContent, modifiedContent)
	
	// Create aligned buffer content
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)
	
	// Create diff buffers
	headBufferName := "test.go@HEAD"
	currentBufferName := "test.go@current"
	
	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, config.IndentTabs, 4)
	
	// Store buffers and diff line info
	te.buffers[headBufferName] = headBuffer
	te.buffers[currentBufferName] = currentBuffer
	te.diffLineInfo[headBuffer] = diffResult.LeftLines
	te.diffLineInfo[currentBuffer] = diffResult.RightLines
	
	// Set up windows for side-by-side diff view
	te.createSecondWindow()
	te.windows[0].Buffer = headBuffer    // HEAD on left
	te.windows[1].Buffer = currentBuffer // Current on right
	
	t.Run("head_buffer_line_numbers", func(t *testing.T) {
		// Switch to head buffer and render
		te.activeWindow = 0
		te.switchToBuffer(headBuffer)
		te.render()
		
		// Verify diff line structure first
		if len(diffResult.LeftLines) == 0 {
			t.Fatal("No diff lines generated")
		}
		
		t.Logf("HEAD buffer diff structure:")
		for i, line := range diffResult.LeftLines {
			t.Logf("  Buffer line %d: type=%v, filler=%v, LeftLine=%d, content=%q", 
				i, line.LineType, line.IsFillerLine, line.LeftLine, line.Content)
		}
		
		// Check rendered line numbers
		mockScreen := te.screen.(*MockScreen)
		_, height := mockScreen.Size()
		
		// Extract line numbers from the screen
		for screenY := 0; screenY < height && screenY < len(diffResult.LeftLines); screenY++ {
			diffLine := diffResult.LeftLines[screenY]
			
			// Extract line number from screen (first few characters before space)
			var lineNumStr string
			for x := 0; x < te.CurrentBuffer().LineNumberWidth; x++ {
				ch, _ := mockScreen.GetCell(x, screenY)
				if ch == ' ' {
					break
				}
				if ch >= '0' && ch <= '9' {
					lineNumStr += string(ch)
				}
			}
			
			if diffLine.IsFillerLine {
				// Filler lines should not have line numbers
				if lineNumStr != "" {
					t.Errorf("Screen line %d: filler line should not have line number, got %q", screenY, lineNumStr)
				} else {
					t.Logf("✓ Screen line %d: filler line correctly has no line number", screenY)
				}
			} else {
				// Real lines should show original line number
				expectedLineNum := ""
				if diffLine.LeftLine >= 0 {
					expectedLineNum = fmt.Sprintf("%d", diffLine.LeftLine+1) // Convert to 1-based
				}
				
				if lineNumStr != expectedLineNum {
					t.Errorf("Screen line %d: expected line number %q, got %q (LeftLine=%d)", 
						screenY, expectedLineNum, lineNumStr, diffLine.LeftLine)
				} else {
					t.Logf("✓ Screen line %d: correctly shows line number %q (LeftLine=%d)", 
						screenY, lineNumStr, diffLine.LeftLine)
				}
			}
		}
	})
	
	t.Run("current_buffer_line_numbers", func(t *testing.T) {
		// Switch to current buffer and render
		te.activeWindow = 1
		te.switchToBuffer(currentBuffer)
		te.render()
		
		t.Logf("CURRENT buffer diff structure:")
		for i, line := range diffResult.RightLines {
			t.Logf("  Buffer line %d: type=%v, filler=%v, RightLine=%d, content=%q", 
				i, line.LineType, line.IsFillerLine, line.RightLine, line.Content)
		}
		
		// Check rendered line numbers
		mockScreen := te.screen.(*MockScreen)
		_, height := mockScreen.Size()
		
		// Extract line numbers from the screen
		for screenY := 0; screenY < height && screenY < len(diffResult.RightLines); screenY++ {
			diffLine := diffResult.RightLines[screenY]
			
			// Extract line number from screen (first few characters before space)
			var lineNumStr string
			for x := 0; x < te.CurrentBuffer().LineNumberWidth; x++ {
				ch, _ := mockScreen.GetCell(x, screenY)
				if ch == ' ' {
					break
				}
				if ch >= '0' && ch <= '9' {
					lineNumStr += string(ch)
				}
			}
			
			if diffLine.IsFillerLine {
				// Filler lines should not have line numbers
				if lineNumStr != "" {
					t.Errorf("Screen line %d: filler line should not have line number, got %q", screenY, lineNumStr)
				} else {
					t.Logf("✓ Screen line %d: filler line correctly has no line number", screenY)
				}
			} else {
				// Real lines should show original line number
				expectedLineNum := ""
				if diffLine.RightLine >= 0 {
					expectedLineNum = fmt.Sprintf("%d", diffLine.RightLine+1) // Convert to 1-based
				}
				
				if lineNumStr != expectedLineNum {
					t.Errorf("Screen line %d: expected line number %q, got %q (RightLine=%d)", 
						screenY, expectedLineNum, lineNumStr, diffLine.RightLine)
				} else {
					t.Logf("✓ Screen line %d: correctly shows line number %q (RightLine=%d)", 
						screenY, lineNumStr, diffLine.RightLine)
				}
			}
		}
	})
	
	t.Run("regular_buffer_unchanged", func(t *testing.T) {
		// Create a regular buffer to ensure normal line numbering still works
		regularBuffer := buffer.NewBuffer("line 1\nline 2\nline 3", "regular.go", config.IndentTabs, 4)
		te.buffers["regular.go"] = regularBuffer
		te.switchToBuffer(regularBuffer)
		te.render()
		
		// Check that regular buffers still use sequential line numbers
		mockScreen := te.screen.(*MockScreen)
		
		for screenY := 0; screenY < 3; screenY++ {
			var lineNumStr string
			var fullLine string
			for x := 0; x < te.CurrentBuffer().LineNumberWidth; x++ {
				ch, _ := mockScreen.GetCell(x, screenY)
				fullLine += string(ch)
				if ch == ' ' && lineNumStr != "" {
					break // Stop after we have digits and hit a space
				}
				if ch >= '0' && ch <= '9' {
					lineNumStr += string(ch)
				}
			}
			
			t.Logf("Regular buffer screen line %d: gutter content %q, extracted line number %q", 
				screenY, fullLine, lineNumStr)
			
			expectedLineNum := fmt.Sprintf("%d", screenY+1)
			if lineNumStr != expectedLineNum {
				t.Errorf("Regular buffer screen line %d: expected line number %q, got %q", 
					screenY, expectedLineNum, lineNumStr)
			} else {
				t.Logf("✓ Regular buffer screen line %d: correctly shows line number %q", 
					screenY, lineNumStr)
			}
		}
	})
}

func TestDiffLineNumbersRealExample(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")
	
	// Use the original user's example case (from conversation summary)
	originalContent := `func renderWindow(w *Window) {
    // Draw each window using a windowScreen. Inactive windows are dimmed.
    if w.isActive() {
        drawActive(w)
    } else {
        drawInactive(w)
    }
}`
	
	modifiedContent := `func renderWindow(w *Window) {
    // Draw each window using a windowScreen. Inactive windows are dimmed unless in diff view.
    if w.isActive() {
        drawActive(w)
    } else {
        drawInactive(w)
    }
}`
	
	diffResult := computeDiff(originalContent, modifiedContent)
	
	// Create diff buffers
	headBuffer := buffer.NewBuffer(createBufferContent(diffResult.LeftLines), "render.go@HEAD", config.IndentTabs, 4)
	currentBuffer := buffer.NewBuffer(createBufferContent(diffResult.RightLines), "render.go@current", config.IndentTabs, 4)
	
	te.buffers["render.go@HEAD"] = headBuffer
	te.buffers["render.go@current"] = currentBuffer
	te.diffLineInfo[headBuffer] = diffResult.LeftLines
	te.diffLineInfo[currentBuffer] = diffResult.RightLines
	
	te.switchToBuffer(currentBuffer)
	te.render()
	
	t.Logf("Real example diff structure:")
	for i, line := range diffResult.RightLines {
		t.Logf("  Line %d: type=%v, filler=%v, RightLine=%d, content=%q", 
			i, line.LineType, line.IsFillerLine, line.RightLine, line.Content)
	}
	
	// The key test: line 2 (the comment line) should show as line number "2" 
	// even though it has modifications, since it maps to the original line 2
	mockScreen := te.screen.(*MockScreen)
	
	// Find the comment line (should be around line 1, 0-indexed)
	commentLineIndex := -1
	for i, line := range diffResult.RightLines {
		if strings.Contains(line.Content, "Draw each window") {
			commentLineIndex = i
			break
		}
	}
	
	if commentLineIndex == -1 {
		t.Fatal("Could not find comment line in diff result")
	}
	
	// Extract line number for the comment line
	var lineNumStr string
	for x := 0; x < te.CurrentBuffer().LineNumberWidth; x++ {
		ch, _ := mockScreen.GetCell(x, commentLineIndex)
		if ch == ' ' {
			break
		}
		if ch >= '0' && ch <= '9' {
			lineNumStr += string(ch)
		}
	}
	
	diffLine := diffResult.RightLines[commentLineIndex]
	expectedLineNum := fmt.Sprintf("%d", diffLine.RightLine+1)
	
	if lineNumStr != expectedLineNum {
		t.Errorf("Comment line: expected line number %q, got %q (RightLine=%d)", 
			expectedLineNum, lineNumStr, diffLine.RightLine)
	} else {
		t.Logf("✓ Comment line correctly shows original line number %q", lineNumStr)
	}
}