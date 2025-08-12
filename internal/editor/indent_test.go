package editor

import (
	"testing"
	"veda/internal/config"
	"veda/internal/events"
)

func TestIndentLine(t *testing.T) {
	t.Run("indent line with spaces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Execute indent command directly
		te.ExecuteCommand("indent-line")
		
		// Check that line is indented with 4 spaces (default)
		AssertBufferEquals(t, te, "    hello world")
		
		// Cursor should be at first non-whitespace character
		AssertCursorAt(t, te, 4, 0)
	})
	
	t.Run("indent already indented line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "  hello world")
		
		// Execute indent command
		te.ExecuteCommand("indent-line")
		
		// Check that line is further indented
		AssertBufferEquals(t, te, "      hello world")
		
		// Cursor should be at first non-whitespace character
		AssertCursorAt(t, te, 6, 0)
	})
	
	t.Run("indent empty line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// Execute indent command
		te.ExecuteCommand("indent-line")
		
		// Check that line is indented
		AssertBufferEquals(t, te, "    ")
		
		// Cursor should be at end of indentation for empty line
		AssertCursorAt(t, te, 4, 0)
	})
	
	t.Run("indent with tabs", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Set buffer to use tabs
		te.buffer.SetIndentationType(config.IndentTabs)
		
		// Execute indent command
		te.ExecuteCommand("indent-line")
		
		// Check that line is indented with a tab
		AssertBufferEquals(t, te, "\thello world")
		
		// Cursor should be at first non-whitespace character
		AssertCursorAt(t, te, 1, 0)
	})
	
	t.Run("undo indent", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Execute indent command
		te.ExecuteCommand("indent-line")
		AssertBufferEquals(t, te, "    hello world")
		
		// Undo
		te.TypeKeys("u")
		AssertBufferEquals(t, te, "hello world")
	})
}

func TestUnindentLine(t *testing.T) {
	t.Run("unindent line with spaces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "    hello world")
		
		// Execute unindent command
		te.ExecuteCommand("unindent-line")
		
		// Check that indentation is removed
		AssertBufferEquals(t, te, "hello world")
		
		// Cursor should be at first non-whitespace character
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("unindent partially indented line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "  hello world")
		
		// Execute unindent command
		te.ExecuteCommand("unindent-line")
		
		// Check that indentation is removed
		AssertBufferEquals(t, te, "hello world")
		
		// Cursor should be at first non-whitespace character
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("unindent line with more than one indent level", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "        hello world")
		
		// Execute unindent command
		te.ExecuteCommand("unindent-line")
		
		// Check that one level of indentation is removed
		AssertBufferEquals(t, te, "    hello world")
		
		// Cursor should be at first non-whitespace character
		AssertCursorAt(t, te, 4, 0)
	})
	
	t.Run("unindent line with no indentation", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Execute unindent command
		te.ExecuteCommand("unindent-line")
		
		// Check that nothing changes
		AssertBufferEquals(t, te, "hello world")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("unindent line with tabs", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "\thello world")
		
		// Set buffer to use tabs
		te.buffer.SetIndentationType(config.IndentTabs)
		
		// Execute unindent command
		te.ExecuteCommand("unindent-line")
		
		// Check that tab is removed
		AssertBufferEquals(t, te, "hello world")
		
		// Cursor should be at first non-whitespace character
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("unindent mixed tabs and spaces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "\t  hello world")
		
		// Set buffer to use tabs
		te.buffer.SetIndentationType(config.IndentTabs)
		
		// Execute unindent command - should remove the tab
		te.ExecuteCommand("unindent-line")
		
		// Check that tab is removed, spaces remain
		AssertBufferEquals(t, te, "  hello world")
		
		// Cursor should be at first non-whitespace character
		AssertCursorAt(t, te, 2, 0)
	})
	
	t.Run("undo unindent", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "    hello world")
		
		// Execute unindent command
		te.ExecuteCommand("unindent-line")
		AssertBufferEquals(t, te, "hello world")
		
		// Undo
		te.TypeKeys("u")
		AssertBufferEquals(t, te, "    hello world")
	})
}

func TestIndentUnindentRepeat(t *testing.T) {
	t.Run("repeat indent with dot", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
		
		// Indent first line
		te.ExecuteCommand("indent-line")
		AssertBufferEquals(t, te, "    line1\nline2\nline3")
		
		// Move to second line
		te.TypeKeys("down")
		
		// Repeat indent with dot
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "    line1\n    line2\nline3")
	})
	
	t.Run("repeat unindent with dot", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "    line1\n    line2\n    line3")
		
		// Unindent first line
		te.ExecuteCommand("unindent-line")
		AssertBufferEquals(t, te, "line1\n    line2\n    line3")
		
		// Move to second line
		te.TypeKeys("down")
		
		// Repeat unindent with dot
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "line1\nline2\n    line3")
	})
	
	t.Run("repeat indent multiple times on same line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Indent once
		te.ExecuteCommand("indent-line")
		AssertBufferEquals(t, te, "    hello world")
		
		// Repeat indent with dot (should add another level)
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "        hello world")
		
		// Repeat again
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "            hello world")
	})
	
	t.Run("repeat unindent multiple times on same line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "            hello world")
		
		// Unindent once
		te.ExecuteCommand("unindent-line")
		AssertBufferEquals(t, te, "        hello world")
		
		// Repeat unindent with dot
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "    hello world")
		
		// Repeat again
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "hello world")
	})
	
	t.Run("repeat indent across different lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3\nline4")
		
		// Indent first line
		te.ExecuteCommand("indent-line")
		AssertBufferEquals(t, te, "    line1\nline2\nline3\nline4")
		
		// Move to third line (skip line2)
		te.TypeKeys("j")
		te.TypeKeys("j")
		
		// Repeat indent with dot on line3
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "    line1\nline2\n    line3\nline4")
		
		// Move to fourth line
		te.TypeKeys("j")
		
		// Repeat again on line4
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "    line1\nline2\n    line3\n    line4")
	})
	
	t.Run("repeat is overridden by insert mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello\nworld")
		
		// Indent first line
		te.ExecuteCommand("indent-line")
		AssertBufferEquals(t, te, "    hello\nworld")
		
		// Do some other editing (insert mode)
		te.TypeKeys("A")  // Append at end of line
		te.TypeKeys("!")
		te.PressKey(events.KeyEscape, 0)
		AssertBufferEquals(t, te, "    hello!\nworld")
		
		// Move to second line
		te.TypeKeys("j")
		
		// Dot should repeat the insert (not the indent)
		// Insert mode changes take precedence over simple commands
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "    hello!\nworld!")
	})
	
	t.Run("repeat unindent with tabs", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "\t\thello\n\t\tworld")
		
		// Set buffer to use tabs
		te.buffer.SetIndentationType(config.IndentTabs)
		
		// Unindent first line
		te.ExecuteCommand("unindent-line")
		AssertBufferEquals(t, te, "\thello\n\t\tworld")
		
		// Move to second line
		te.TypeKeys("j")
		
		// Repeat unindent
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "\thello\n\tworld")
	})
	
	t.Run("repeat switches between indent and unindent", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "    hello\nworld")
		
		// Unindent first line
		te.ExecuteCommand("unindent-line")
		AssertBufferEquals(t, te, "hello\nworld")
		
		// Move to second line
		te.TypeKeys("j")
		
		// Dot should repeat unindent (do nothing since line has no indent)
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "hello\nworld")
		
		// Now indent the second line
		te.ExecuteCommand("indent-line")
		AssertBufferEquals(t, te, "hello\n    world")
		
		// Move back to first line
		te.TypeKeys("k")
		
		// Dot should now repeat indent (not unindent)
		te.TypeKeys(".")
		AssertBufferEquals(t, te, "    hello\n    world")
	})
}

func TestIndentWithMultipleCursors(t *testing.T) {
	t.Run("indent multiple lines with cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at current position (0,0) and move down
		te.PressKey(events.KeyEnter, 0)
		
		// Add another virtual cursor at (0,1) and move down
		te.PressKey(events.KeyEnter, 0)
		
		// Now we have cursors at lines 0, 1, and 2
		// Exit cursor mode
		te.TypeKeys("m")
		
		// Execute indent command
		te.ExecuteCommand("indent-line")
		
		// All lines should be indented
		AssertBufferEquals(t, te, "    line1\n    line2\n    line3")
	})
	
	t.Run("unindent multiple lines with cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "    line1\n    line2\n    line3")
		
		// Enter cursor mode
		te.TypeKeys("m")
		
		// Add virtual cursor at current position (0,0) and move down
		te.PressKey(events.KeyEnter, 0)
		
		// Add another virtual cursor at (0,1) and move down
		te.PressKey(events.KeyEnter, 0)
		
		// Now we have cursors at lines 0, 1, and 2
		// Exit cursor mode
		te.TypeKeys("m")
		
		// Execute unindent command
		te.ExecuteCommand("unindent-line")
		
		// All lines should be unindented
		AssertBufferEquals(t, te, "line1\nline2\nline3")
	})
}

func TestIndentEdgeCases(t *testing.T) {
	t.Run("indent line with only whitespace", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "   ")
		
		// Execute indent command
		te.ExecuteCommand("indent-line")
		
		// Should add indentation
		AssertBufferEquals(t, te, "       ")
	})
	
	t.Run("unindent line with mixed indentation levels", func(t *testing.T) {
		// Line starts with: 2 spaces, 1 tab, 2 spaces, then "hello"
		te := NewTestEditorWithContent(80, 24, "  \t  hello")
		
		// Execute unindent command
		// In default spaces mode, this should try to remove up to 4 spaces of indentation
		// Since the line starts with only 2 spaces before the tab,
		// it should remove those 2 spaces
		te.ExecuteCommand("unindent-line")
		
		// Expected: tab and remaining content after removing the 2 leading spaces
		AssertBufferEquals(t, te, "\t  hello")
	})
	
	t.Run("indent at end of file", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "line1\nline2")
		
		// Move to last line
		te.TypeKeys("G")
		
		// Execute indent command
		te.ExecuteCommand("indent-line")
		
		// Last line should be indented
		AssertBufferEquals(t, te, "line1\n    line2")
	})
	
	t.Run("unindent with different indent widths", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "  hello")
		
		// Set indent width to 2
		te.buffer.SetIndentationWidth(2)
		
		// Execute unindent command
		te.ExecuteCommand("unindent-line")
		
		// Should remove 2 spaces
		AssertBufferEquals(t, te, "hello")
	})
	
	t.Run("indent preserves cursor relative position", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")
		
		// Move cursor to 'w' in world
		te.SetCursorPosition(6, 0)
		
		// Execute indent command
		te.ExecuteCommand("indent-line")
		
		// Cursor should be at 'h' in hello (first non-whitespace)
		AssertCursorAt(t, te, 4, 0)
	})
	
	t.Run("consecutive indents", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello")
		
		// Execute indent command twice
		te.ExecuteCommand("indent-line")
		te.ExecuteCommand("indent-line")
		
		// Should have two levels of indentation
		AssertBufferEquals(t, te, "        hello")
	})
	
	t.Run("consecutive unindents", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "        hello")
		
		// Execute unindent command twice
		te.ExecuteCommand("unindent-line")
		te.ExecuteCommand("unindent-line")
		
		// Should remove two levels of indentation
		AssertBufferEquals(t, te, "hello")
	})
}

func TestIndentWithVisualMode(t *testing.T) {
	t.Run("indent in visual line mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
		
		// Enter visual line mode
		te.TypeKeys("V")
		
        // Select two lines
        te.TypeKeys("down")
		
        // Execute visual indent (>>)
        te.TypeKeys(">")
        te.TypeKeys(">")
        
        // First two lines should be indented
        AssertBufferEquals(t, te, "    line1\n    line2\nline3")

        // Cursor should return to start of selection (line1)
        AssertCursorAt(t, te, 0, 0)
	})

    t.Run("unindent in visual line mode", func(t *testing.T) {
        te := NewTestEditorWithContent(80, 24, "    line1\n    line2\nline3")

        te.TypeKeys("V")
        te.TypeKeys("down")
        // Execute visual unindent (<<)
        te.TypeKeys("<")
        te.TypeKeys("<")

        AssertBufferEquals(t, te, "line1\nline2\nline3")
        AssertCursorAt(t, te, 0, 0)
    })

    t.Run("counted visual indent (3>>)", func(t *testing.T) {
        te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")

        te.TypeKeys("V")
        te.TypeKeys("down")
        // Simulate count prefix by setting repeatCount before executing command
        te.repeatCount = 3
        _ = te.executeWithRepeatCount("indent-selection")

        AssertBufferEquals(t, te, "            line1\n            line2\nline3")
    })

    t.Run("counted visual unindent (2<<)", func(t *testing.T) {
        te := NewTestEditorWithContent(80, 24, "        line1\n        line2\n    line3")

        te.TypeKeys("V")
        te.TypeKeys("down")
        // Simulate count prefix by setting repeatCount before executing command
        te.repeatCount = 2
        _ = te.executeWithRepeatCount("unindent-selection")

        AssertBufferEquals(t, te, "line1\nline2\n    line3")
    })
}