package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
	"veda/internal/syntax"
	"veda/internal/theme"
)

func TestCursorMode(t *testing.T) {
	t.Run("enter and exit cursor mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")
		
		// Initially in normal mode
		if te.mode != commands.NormalMode {
			t.Errorf("Expected initial mode to be NORMAL, got %s", te.mode)
		}
		
		// Enter cursor mode with 'm'
		te.TypeKeys("m")
		if te.mode != commands.CursorMode {
			t.Errorf("Expected mode to be CURSOR after pressing 'm', got %s", te.mode)
		}
		
		// Exit cursor mode with 'm' again
		te.TypeKeys("m")
		if te.mode != commands.NormalMode {
			t.Errorf("Expected mode to be NORMAL after pressing 'm' again, got %s", te.mode)
		}
		
		// Enter cursor mode again
		te.TypeKeys("m")
		if te.mode != commands.CursorMode {
			t.Errorf("Expected mode to be CURSOR, got %s", te.mode)
		}
		
		// Exit with Escape
		te.PressKey(events.KeyEscape, 0)
		if te.mode != commands.NormalMode {
			t.Errorf("Expected mode to be NORMAL after escape, got %s", te.mode)
		}
	})
	
	t.Run("add virtual cursor with enter", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add a virtual cursor at current position (0, 0)
		te.PressKey(events.KeyEnter, 0)
		
		// Check that we have one virtual cursor
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 1 {
			t.Errorf("Expected 1 virtual cursor, got %d", len(virtualCursors))
		}
		
		// Check that virtual cursor is at (0, 0)
		if len(virtualCursors) > 0 && (virtualCursors[0].X != 0 || virtualCursors[0].Y != 0) {
			t.Errorf("Expected virtual cursor at (0, 0), got (%d, %d)", 
				virtualCursors[0].X, virtualCursors[0].Y)
		}
		
		// Check that primary cursor moved down one line
		x, y := te.GetCursorPosition()
		if x != 0 || y != 1 {
			t.Errorf("Expected primary cursor at (0, 1) after adding cursor, got (%d, %d)", x, y)
		}
	})
	
	t.Run("multiple virtual cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3\nLine 4")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add three virtual cursors
		te.PressKey(events.KeyEnter, 0) // Adds cursor at (0,0), moves to (0,1)
		te.PressKey(events.KeyEnter, 0) // Adds cursor at (0,1), moves to (0,2)
		te.PressKey(events.KeyEnter, 0) // Adds cursor at (0,2), moves to (0,3)
		
		// Check that we have three virtual cursors
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 3 {
			t.Errorf("Expected 3 virtual cursors, got %d", len(virtualCursors))
		}
		
		// Verify cursor positions
		expectedPositions := []struct{ x, y int }{
			{0, 0},
			{0, 1},
			{0, 2},
		}
		for i, expected := range expectedPositions {
			cursor := virtualCursors[i]
			if cursor.X != expected.x || cursor.Y != expected.y {
				t.Errorf("Virtual cursor %d: expected (%d, %d), got (%d, %d)",
					i, expected.x, expected.y, cursor.X, cursor.Y)
			}
		}
		
		// Check primary cursor is on line 4
		x, y := te.GetCursorPosition()
		if x != 0 || y != 3 {
			t.Errorf("Expected primary cursor at (0, 3), got (%d, %d)", x, y)
		}
	})
	
	t.Run("movement in cursor mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World\nSecond Line")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)
		
		// Move primary cursor with 'w' (word forward)
		te.TypeKeys("w")
		x, y := te.GetCursorPosition()
		// After adding virtual cursor, primary cursor is at (0,1) - start of "Second"
		// 'w' moves to next word "Line" at position 7
		if x != 7 || y != 1 {
			t.Errorf("Expected cursor at (7, 1) after 'w', got (%d, %d)", x, y)
		}
		
		// Virtual cursor should stay at (0,0)
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) > 0 && (virtualCursors[0].X != 0 || virtualCursors[0].Y != 0) {
			t.Errorf("Virtual cursor should stay at (0, 0), got (%d, %d)",
				virtualCursors[0].X, virtualCursors[0].Y)
		}
	})
	
	t.Run("exit cursor mode on buffer modification", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Test")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add a virtual cursor
		te.PressKey(events.KeyEnter, 0)
		
		// Enter insert mode (should exit cursor mode)
		te.TypeKeys("i")
		if te.mode != commands.InsertMode {
			t.Errorf("Expected INSERT mode after 'i', got %s", te.mode)
		}
		
		// Virtual cursors should NOT be cleared - they persist for multiple cursor editing
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 1 {
			t.Errorf("Expected virtual cursors to persist, got %d instead of 1", len(virtualCursors))
		}
	})
	
	t.Run("exit cursor mode on command mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Test")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add a virtual cursor
		te.PressKey(events.KeyEnter, 0)
		
		// Enter command mode
		te.TypeKeys(":")
		if te.mode != commands.CommandMode {
			t.Errorf("Expected COMMAND mode after ':', got %s", te.mode)
		}
		
		// Virtual cursors ARE cleared when entering command mode
		// (this is intentional - see line 39 in mode_management.go)
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 0 {
			t.Errorf("Expected virtual cursors to be cleared, got %d", len(virtualCursors))
		}
	})
}

func TestCursorRendering(t *testing.T) {
	t.Run("virtual cursor background style", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "ABC\nDEF\nGHI")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at (0,0) - on character 'A'
		te.PressKey(events.KeyEnter, 0)
		
		// Force a render
		te.render()
		
		// Get the cell at position (0,0) where virtual cursor is
		// Note: We need to account for line numbers which shift content right
        lineNumWidth := te.buffer.LineNumberWidth
		char, style := te.mockScreen.GetCell(lineNumWidth, 0)
		
		// Check that the character is still 'A'
		if char != 'A' {
			t.Errorf("Expected character 'A' at virtual cursor position, got '%c'", char)
		}
		
		// Check that background style is applied
		// Get the virtual cursor background color from the theme
		theme := theme.NewTheme()
		virtualCursorStyle := theme.GetStyle(theme.VirtualCursor)
		virtualCursorBg := virtualCursorStyle.Bg
		if style.Bg != virtualCursorBg {
			t.Errorf("Expected virtual cursor background, got %v, expected %v", style.Bg, virtualCursorBg)
		}
	})
	
	t.Run("multiple virtual cursors rendering", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "ABC\nDEF\nGHI")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add three virtual cursors
		te.PressKey(events.KeyEnter, 0) // At (0,0)
		te.PressKey(events.KeyEnter, 0) // At (0,1)
		te.PressKey(events.KeyEnter, 0) // At (0,2)
		
		// Force a render
		te.render()
		
		// Check each virtual cursor position has styled background
        lineNumWidth := te.buffer.LineNumberWidth
		positions := []struct {
			x, y int
			char rune
		}{
			{0, 0, 'A'},
			{0, 1, 'D'},
			{0, 2, 'G'},
		}
		
		for _, pos := range positions {
			char, style := te.mockScreen.GetCell(lineNumWidth+pos.x, pos.y)
			
			// Check character
			if char != pos.char {
				t.Errorf("Expected '%c' at (%d,%d), got '%c'", 
					pos.char, pos.x, pos.y, char)
			}
			
			// Check background style - should match theme's VirtualCursor style
			theme := theme.NewTheme()
		virtualCursorStyle := theme.GetStyle(theme.VirtualCursor)
		virtualCursorBg := virtualCursorStyle.Bg
			if style.Bg != virtualCursorBg {
				t.Errorf("Expected virtual cursor background at (%d,%d), got %v, expected %v", pos.x, pos.y, style.Bg, virtualCursorBg)
			}
		}
	})
	
	t.Run("virtual cursor position in text area", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Move cursor to position 6 (start of "World")
		te.TypeKeys("w")
		
		// Verify cursor is at position 6
		x, y := te.GetCursorPosition()
		if x != 6 || y != 0 {
			t.Errorf("Expected cursor at (6,0) before adding virtual cursor, got (%d,%d)", x, y)
		}
		
		// Enter cursor mode and add virtual cursor
		te.TypeKeys("m")
		te.PressKey(events.KeyEnter, 0)
		
		// Verify virtual cursor was added at position 6
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 1 {
			t.Fatalf("Expected 1 virtual cursor, got %d", len(virtualCursors))
		}
		if virtualCursors[0].X != 6 || virtualCursors[0].Y != 0 {
			t.Errorf("Expected virtual cursor at (6,0), got (%d,%d)", 
				virtualCursors[0].X, virtualCursors[0].Y)
		}
		
		// Force a render
		te.render()
		
		// Virtual cursor should be at column 6 in the text
		// With line numbers, this is at screen position lineNumWidth + 6
        lineNumWidth := te.buffer.LineNumberWidth
		char, style := te.mockScreen.GetCell(lineNumWidth+6, 0)
		
		// Check that we're looking at 'W' from "World"
		if char != 'W' {
			t.Errorf("Expected 'W' at virtual cursor position (screen x=%d), got '%c'", lineNumWidth+6, char)
		}
		
		// Check that it has the correct virtual cursor background
		theme := theme.NewTheme()
		virtualCursorStyle := theme.GetStyle(theme.VirtualCursor)
		virtualCursorBg := virtualCursorStyle.Bg
		if style.Bg != virtualCursorBg {
			t.Errorf("Expected virtual cursor background at screen position %d, got %v, expected %v", lineNumWidth+6, style.Bg, virtualCursorBg)
		}
		
		// Also check the characters before and after don't have virtual cursor styling
		if lineNumWidth+5 >= 0 {
			_, beforeStyle := te.mockScreen.GetCell(lineNumWidth+5, 0) // Space before 'W'
			if beforeStyle.Bg == virtualCursorBg {
				t.Errorf("Character before virtual cursor should not have virtual cursor styling")
			}
		}
		
		if lineNumWidth+7 < 80 {
			_, afterStyle := te.mockScreen.GetCell(lineNumWidth+7, 0) // 'o' after 'W'
			if afterStyle.Bg == virtualCursorBg {
				t.Errorf("Character after virtual cursor should not have virtual cursor styling")
			}
		}
		
		// Verify that line number area does NOT have virtual cursor styling
		for x := 0; x < lineNumWidth; x++ {
			_, style := te.mockScreen.GetCell(x, 0)
			if style.Bg == virtualCursorBg {
				t.Errorf("Line number area at column %d should not have virtual cursor styling", x)
			}
		}
	})
	
	t.Run("virtual cursor at beginning of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "ABC\nDEF\nGHI")
		
		// Cursor starts at (0,0)
		// Enter cursor mode and add virtual cursor
		te.TypeKeys("m")
		te.PressKey(events.KeyEnter, 0)
		
		// Force a render
		te.render()
		
		// Check that line numbers don't have virtual cursor styling
		lineNumWidth := te.buffer.LineNumberWidth
		theme := theme.NewTheme()
		virtualCursorStyle := theme.GetStyle(theme.VirtualCursor)
		virtualCursorBg := virtualCursorStyle.Bg
		
		// Check line number area
		for x := 0; x < lineNumWidth; x++ {
			_, style := te.mockScreen.GetCell(x, 0)
			if style.Bg == virtualCursorBg {
				t.Errorf("Line number area at column %d incorrectly has virtual cursor styling", x)
			}
		}
		
		// Virtual cursor should be at the first character of text (after line numbers)
		char, style := te.mockScreen.GetCell(lineNumWidth, 0)
		if char != 'A' {
			t.Errorf("Expected 'A' at position 0 in text, got '%c'", char)
		}
		if style.Bg != virtualCursorBg {
			t.Errorf("Expected virtual cursor background at beginning of line, got %v, expected %v", style.Bg, virtualCursorBg)
		}
	})
	
	t.Run("primary cursor remains as terminal cursor", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line1\nLine2")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)
		
		// Primary cursor should be at (0,1) now
		x, y := te.GetCursorPosition()
		if x != 0 || y != 1 {
			t.Errorf("Expected primary cursor at (0,1), got (%d,%d)", x, y)
		}
		
		// Force a render to update screen cursor position
		te.render()
		
		// Check that mock screen's cursor position matches
		// (accounting for line numbers)
        lineNumWidth := te.buffer.LineNumberWidth
		screenX, screenY := te.mockScreen.GetCursorPosition()
		if screenX != lineNumWidth || screenY != 1 {
			t.Errorf("Expected screen cursor at (%d,1), got (%d,%d)", 
				lineNumWidth, screenX, screenY)
		}
	})
}

func TestCursorRenderingWithMarkdown(t *testing.T) {
	t.Run("virtual cursors in markdown file", func(t *testing.T) {
		// Create editor with markdown content
		markdownContent := `# Heading
This is a **bold** text.
- List item 1
- List item 2`
		
		te := NewTestEditorWithContent(80, 24, markdownContent)
		
		// Set the filename to trigger markdown syntax highlighting
		te.buffer.FilePath = "test.md"
		te.highlighter = syntax.NewHighlighter()
		if te.highlighter != nil {
			te.highlighter.SetLanguageFromFile("test.md")
			te.highlighter.Parse([]byte(te.buffer.Rope.String()))
		}
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at position 0 (on '#')
		te.PressKey(events.KeyEnter, 0)
		
		// Move to "bold" and add another cursor
		te.TypeKeys("j") // Move to line 2
		te.TypeKeys("w") // Move to "is"
		te.TypeKeys("w") // Move to "a"  
		te.TypeKeys("w") // Move to "**bold**"
		
		// Get cursor position before adding virtual cursor
		x, y := te.GetCursorPosition()
		t.Logf("Cursor position before adding virtual cursor: (%d, %d)", x, y)
		
		// Add another virtual cursor
		te.PressKey(events.KeyEnter, 0)
		
		// Verify we have 2 virtual cursors
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors, got %d", len(virtualCursors))
		}
		
		// Force a render
		te.render()
		
		// Log virtual cursor info for debugging
		t.Logf("Virtual cursors: %+v", virtualCursors)
		t.Logf("Highlighter active: %v", te.highlighter != nil)
		if te.highlighter != nil {
			t.Logf("Language: %s", te.highlighter.GetCurrentLanguage())
		}
		
		// Check first virtual cursor (at beginning of first line)
		lineNumWidth := te.buffer.LineNumberWidth
		char1, style1 := te.mockScreen.GetCell(lineNumWidth, 0)
		theme := theme.NewTheme()
		virtualCursorStyle := theme.GetStyle(theme.VirtualCursor)
		virtualCursorBg := virtualCursorStyle.Bg
		
		if char1 != '#' {
			t.Errorf("Expected '#' at first virtual cursor position, got '%c'", char1)
		}
		if style1.Bg != virtualCursorBg {
			// Log more info about what's happening
			fg, bg, _ := style1.Decompose()
			t.Logf("Style at virtual cursor: fg=%v, bg=%v (expected bg=%v)", fg, bg, virtualCursorBg)
			// Check what's on the screen around this position
			for x := 0; x < 20; x++ {
				ch, st := te.mockScreen.GetCell(x, 0)
				_, bgColor, _ := st.Decompose()
				if x == lineNumWidth {
					t.Logf("  [VC] x=%d: '%c' bg=%v", x, ch, bgColor)
				} else {
					t.Logf("       x=%d: '%c' bg=%v", x, ch, bgColor)
				}
			}
			t.Errorf("Expected virtual cursor background for first virtual cursor, got %v, expected %v", style1.Bg, virtualCursorBg)
		}
		
		// Check second virtual cursor (on "bold")
		// The exact position depends on the cursor movement
		if len(virtualCursors) >= 2 {
			vc2 := virtualCursors[1]
			t.Logf("Second virtual cursor at: (%d, %d)", vc2.X, vc2.Y)
			
			// Check that position has virtual cursor background
			char2, style2 := te.mockScreen.GetCell(lineNumWidth+vc2.X, vc2.Y)
			if style2.Bg != virtualCursorBg {
				t.Errorf("Expected virtual cursor background for second virtual cursor at (%d,%d), got %v, expected %v. Character: '%c'", 
					vc2.X, vc2.Y, style2.Bg, virtualCursorBg, char2)
			}
		}
	})
	
	t.Run("virtual cursors with multiple markdown elements", func(t *testing.T) {
		markdownContent := `## TODO List
- [ ] Task 1
- [x] Task 2
**Important:** Remember to test.
[Link](http://example.com)`
		
		te := NewTestEditorWithContent(80, 24, markdownContent)
		te.buffer.FilePath = "test.md"
		
		// Enter cursor mode and add cursors at different positions
		te.TypeKeys("m")
		te.PressKey(events.KeyEnter, 0) // Add cursor at "##"
		
		te.TypeKeys("j") // Move to line 2
		te.TypeKeys("w") // Move past "-"
		te.TypeKeys("w") // Move past "["
		te.PressKey(events.KeyEnter, 0) // Add cursor at "]"
		
		te.TypeKeys("j") // Move to line 3
		te.TypeKeys("j") // Move to line 4
		te.PressKey(events.KeyEnter, 0) // Add cursor at "**Important**"
		
		// Force render
		te.render()
		
		// Verify all virtual cursors are rendered with the theme's virtual cursor background
		lineNumWidth := te.buffer.LineNumberWidth
		theme := theme.NewTheme()
		virtualCursorStyle := theme.GetStyle(theme.VirtualCursor)
		virtualCursorBg := virtualCursorStyle.Bg
		
		virtualCursors := te.GetVirtualCursors()
		for i, vc := range virtualCursors {
			char, style := te.mockScreen.GetCell(lineNumWidth+vc.X, vc.Y)
			if style.Bg != virtualCursorBg {
				t.Errorf("Virtual cursor %d at (%d,%d) doesn't have expected background. Got %v, expected %v, char '%c'",
					i, vc.X, vc.Y, style.Bg, virtualCursorBg, char)
			}
			
			// Verify line number area doesn't have cursor styling
			for x := 0; x < lineNumWidth; x++ {
				_, lineStyle := te.mockScreen.GetCell(x, vc.Y)
				if lineStyle.Bg == virtualCursorBg {
					t.Errorf("Line number area at (%d,%d) incorrectly has virtual cursor styling", x, vc.Y)
				}
			}
		}
		
		// Log cursor positions for debugging
		t.Logf("Virtual cursors added at:")
		virtualCursors = te.GetVirtualCursors()
		for i, vc := range virtualCursors {
			char, _ := te.mockScreen.GetCell(lineNumWidth+vc.X, vc.Y)
			t.Logf("  Cursor %d: (%d, %d) on character '%c'", i, vc.X, vc.Y, char)
		}
	})
}

