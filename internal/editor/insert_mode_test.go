package editor

import (
	"testing"
	"veda/internal/commands"
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
		te.cursorX = 5
		
		// Enter insert mode and insert comma
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar(',')
		
		AssertBufferEquals(t, te, "Hello, World")
		AssertCursorAt(t, te, 6, 0)
	})
	
	t.Run("insert newline", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Move cursor to position 5 (after "Hello")
		te.cursorX = 5
		
		// Enter insert mode and insert newline
		te.ExecuteCommand("enter-insert-mode")
		te.ExecuteCommand("insert-newline")
		
		AssertBufferEquals(t, te, "Hello\n World")
		AssertCursorAt(t, te, 0, 1)
	})
	
	t.Run("backspace in insert mode", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// Move to end of line
		te.cursorX = 5
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Delete last character
		te.ExecuteCommand("backspace")
		AssertBufferEquals(t, te, "Hell")
		AssertCursorAt(t, te, 4, 0)
		
		// Delete another character
		te.ExecuteCommand("backspace")
		AssertBufferEquals(t, te, "Hel")
		AssertCursorAt(t, te, 3, 0)
	})
	
	t.Run("backspace at beginning of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line1\nLine2")
		
		// Move to beginning of second line
		te.cursorX = 0
		te.cursorY = 1
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Backspace should join lines
		te.ExecuteCommand("backspace")
		AssertBufferEquals(t, te, "Line1Line2")
		AssertCursorAt(t, te, 5, 0)
	})
	
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
}