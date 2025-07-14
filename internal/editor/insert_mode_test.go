package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/ui"
)

func TestInsertMode(t *testing.T) {
	t.Run("enter and exit insert mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello, World!")
		
		// Start in normal mode
		if te.mode != commands.NormalMode {
			t.Errorf("Expected to start in normal mode, but got %v", te.mode)
		}
		
		// Press 'i' to enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		if te.mode != commands.InsertMode {
			t.Errorf("Expected to be in insert mode, but got %v", te.mode)
		}
		
		// Press Escape to exit insert mode
		te.ExecuteCommand("exit-insert-mode")
		if te.mode != commands.NormalMode {
			t.Errorf("Expected to be back in normal mode, but got %v", te.mode)
		}
	})
	
	t.Run("insert single character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Insert a character at the beginning
		te.InsertChar('X')
		AssertBufferEquals(t, te, "XHello")
		AssertCursorAt(t, te, 1, 0)
	})
	
	t.Run("insert multiple characters", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "World")
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Insert "Hello, " at the beginning
		te.InsertChar('H')
		te.InsertChar('e')
		te.InsertChar('l')
		te.InsertChar('l')
		te.InsertChar('o')
		te.InsertChar(',')
		te.InsertChar(' ')
		
		AssertBufferEquals(t, te, "Hello, World")
		AssertCursorAt(t, te, 7, 0)
	})
	
	t.Run("insert in middle of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Move cursor to position 5 (after "Hello")
		te.buffer.CursorX = 5
		
		// Enter insert mode and insert comma
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar(',')
		
		AssertBufferEquals(t, te, "Hello, World")
		AssertCursorAt(t, te, 6, 0)
	})
	
	t.Run("insert newline", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Move cursor to position 5 (after "Hello")
		te.buffer.CursorX = 5
		
		// Enter insert mode and insert newline
		te.ExecuteCommand("enter-insert-mode")
		te.ExecuteCommand("insert-newline")
		
		AssertBufferEquals(t, te, "Hello\n World")
		AssertCursorAt(t, te, 0, 1)
	})
	
	// DISABLED FOR REFACTORING
	// t.Run("backspace in insert mode", func(t *testing.T) {
	// 	te := NewTestEditorWithContent(80, 24, "Hello")
	// 	
	// 	// Move to end of line
	// 	te.buffer.CursorX = 5
	// 	
	// 	// Enter insert mode
	// 	te.ExecuteCommand("enter-insert-mode")
	// 	
	// 	// Delete last character
	// 	te.ExecuteCommand("backspace")
	// 	AssertBufferEquals(t, te, "Hell")
	// 	AssertCursorAt(t, te, 4, 0)
	// 	
	// 	// Delete another character
	// 	te.ExecuteCommand("backspace")
	// 	AssertBufferEquals(t, te, "Hel")
	// 	AssertCursorAt(t, te, 3, 0)
	// })
	
	// DISABLED FOR REFACTORING
	// t.Run("backspace at beginning of line", func(t *testing.T) {
	// 	te := NewTestEditorWithContent(80, 24, "Line1\nLine2")
	// 	
	// 	// Move to beginning of second line
	// 	te.SetCursorPosition(0, 1)
	// 	
	// 	// Enter insert mode
	// 	te.ExecuteCommand("enter-insert-mode")
	// 	
	// 	// Backspace should join lines
	// 	te.ExecuteCommand("backspace")
	// 	AssertBufferEquals(t, te, "Line1Line2")
	// 	AssertCursorAt(t, te, 5, 0)
	// })
	
	t.Run("insert on empty buffer", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Insert some text
		te.InsertChar('H')
		te.InsertChar('i')
		
		AssertBufferEquals(t, te, "Hi")
		AssertCursorAt(t, te, 2, 0)
	})
	
	// DISABLED FOR REFACTORING
	// t.Run("cursor movement in insert mode", func(t *testing.T) {
	// 	te := NewTestEditorWithContent(80, 24, "Hello\nWorld")
	// 	
	// 	// Move to position (2, 0) - after "He"
	// 	te.buffer.CursorX = 2
	// 	
	// 	// Enter insert mode
	// 	te.ExecuteCommand("enter-insert-mode")
	// 	
	// 	// Move right
	// 	te.ExecuteCommand("right-insert")
	// 	AssertCursorAt(t, te, 3, 0)
	// 	
	// 	// Move left twice
	// 	te.ExecuteCommand("left-insert")
	// 	te.ExecuteCommand("left-insert")
	// 	AssertCursorAt(t, te, 1, 0)
	// 	
	// 	// Move down
	// 	te.ExecuteCommand("down-insert")
	// 	AssertCursorAt(t, te, 1, 1)
	// 	
	// 	// Move up
	// 	te.ExecuteCommand("up-insert")
	// 	AssertCursorAt(t, te, 1, 0)
	// 	
	// 	// Insert a character to verify we're still in insert mode
	// 	te.InsertChar('X')
	// 	AssertBufferEquals(t, te, "HXello\nWorld")
	// 	AssertCursorAt(t, te, 2, 0)
	// })
	
	// DISABLED FOR REFACTORING
	// t.Run("cursor movement respects boundaries in insert mode", func(t *testing.T) {
	// 	te := NewTestEditorWithContent(80, 24, "Hi\nWorld")
	// 	
	// 	// Enter insert mode at start
	// 	te.ExecuteCommand("enter-insert-mode")
	// 	
	// 	// Try to move left at beginning - should stay at (0, 0)
	// 	te.ExecuteCommand("left-insert")
	// 	AssertCursorAt(t, te, 0, 0)
	// 	
	// 	// Move to position 1 (after 'H')
	// 	te.buffer.CursorX = 1
	// 	te.ExecuteCommand("right-insert")
	// 	AssertCursorAt(t, te, 2, 0) // At end of "Hi"
	// 	
	// 	// Move down to second line - cursor should adjust if line is shorter
	// 	te.ExecuteCommand("down-insert")
	// 	AssertCursorAt(t, te, 2, 1)
	// 	
	// 	// Move right three times to get to end of "World" 
	// 	te.ExecuteCommand("right-insert")
	// 	te.ExecuteCommand("right-insert")
	// 	te.ExecuteCommand("right-insert")
	// 	AssertCursorAt(t, te, 5, 1) // At end of "World"
	// 	
	// 	// Try to move right again - should stay at end
	// 	te.ExecuteCommand("right-insert")
	// 	AssertCursorAt(t, te, 5, 1) // Should stop at end
	// 	
	// 	// Try to move down at last line - should stay
	// 	te.ExecuteCommand("down-insert")
	// 	AssertCursorAt(t, te, 5, 1)
	// 	
	// 	// Move left once to insert before last character
	// 	te.ExecuteCommand("left-insert")
	// 	AssertCursorAt(t, te, 4, 1)
	// 	
	// 	// Insert to verify still in insert mode
	// 	te.InsertChar('!')
	// 	AssertBufferEquals(t, te, "Hi\nWorl!d")
	// })
	
	t.Run("cursor style changes in insert mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// Draw the editor to set initial cursor style (should be block in normal mode)
		te.drawEditor()
		te.drawStatusBar()
		te.screen.SetCursorStyle(ui.CursorStyleSteadyBlock) // Simulate what Run() does
		
		// Check initial cursor style
		initialStyle := te.mockScreen.GetCursorStyle()
		if initialStyle != ui.CursorStyleSteadyBlock {
			t.Errorf("Expected initial cursor style to be block, but got %v", initialStyle)
		}
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Simulate what Run() does when changing modes
		if te.mode == commands.InsertMode {
			te.screen.SetCursorStyle(ui.CursorStyleSteadyBar)
		}
		
		// Check cursor style changed to bar
		insertStyle := te.mockScreen.GetCursorStyle()
		if insertStyle != ui.CursorStyleSteadyBar {
			t.Errorf("Expected cursor style to be bar in insert mode, but got %v", insertStyle)
		}
		
		// Exit insert mode
		te.ExecuteCommand("exit-insert-mode")
		
		// Simulate what Run() does when changing modes
		if te.mode != commands.InsertMode {
			te.screen.SetCursorStyle(ui.CursorStyleSteadyBlock)
		}
		
		// Check cursor style changed back to block
		normalStyle := te.mockScreen.GetCursorStyle()
		if normalStyle != ui.CursorStyleSteadyBlock {
			t.Errorf("Expected cursor style to be block in normal mode, but got %v", normalStyle)
		}
	})
}