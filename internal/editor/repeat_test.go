package editor

import (
	"fmt"
	"testing"
	"veda/internal/buffer"
)

func TestRepeatLastChange(t *testing.T) {
	t.Run("repeat insert text", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Move to position after "Hello" (cursor will be on the space)
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 5, 0) // Cursor on space
		
		// Enter insert mode and add text
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar(',')
		te.ExecuteCommand("exit-insert-mode")
		
		// Verify text and cursor position
		AssertBufferEquals(t, te, "Hello, World")
		AssertCursorAt(t, te, 5, 0) // Cursor back on comma after exit
		
		// Move to end of line (cursor will be on 'd')
		te.ExecuteCommand("end-of-line")
		AssertCursorAt(t, te, 11, 0) // Cursor on 'd'
		
		// Repeat the last change
		te.ExecuteCommand("repeat-last-change")
		
		// Should have inserted comma before 'd' (at cursor position)
		AssertBufferEquals(t, te, "Hello, Worl,d")
	})
	
	t.Run("repeat delete character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Delete first character
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "ello World")
		
		// Move right and repeat
		te.ExecuteCommand("right")
		te.ExecuteCommand("repeat-last-change")
		
		// Should have deleted 'l'
		AssertBufferEquals(t, te, "elo World")
	})
	
	t.Run("repeat multi-character insert", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// End of line puts cursor on 'o' (position 4)
		te.ExecuteCommand("end-of-line")
		AssertCursorAt(t, te, 4, 0)
		
		// Enter insert mode - cursor stays at 4 and we insert at 4
		te.ExecuteCommand("enter-insert-mode")
		AssertCursorAt(t, te, 4, 0) // Cursor still at 4
		// Insert happens at position 4 (where cursor is)
		te.InsertChar(' ')
		te.InsertChar('W')
		te.InsertChar('o')
		te.InsertChar('r')
		te.InsertChar('l')
		te.InsertChar('d')
		te.ExecuteCommand("exit-insert-mode")
		// After exit, cursor moves back one to 'd' (position 9)
		AssertCursorAt(t, te, 9, 0)
		
		AssertBufferEquals(t, te, "Hell Worldo")
		
		// Move to position after "Hell"
		te.ExecuteCommand("beginning-of-line")
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}
		// Cursor is now on 'o' in "Hello"
		te.ExecuteCommand("repeat-last-change")
		
		// Should have inserted " World" at position 4
		AssertBufferEquals(t, te, "Hell World Worldo")
	})
	
	t.Run("repeat with newline", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line1")
		
		// Move to after "Line" (position 4, on '1')
		te.ExecuteCommand("end-of-line")  // Cursor on '1' at position 4
		te.ExecuteCommand("enter-insert-mode")
		// Text will be inserted at position 4
		te.ExecuteCommand("insert-newline")
		te.InsertChar('L')
		te.InsertChar('i')
		te.InsertChar('n')
		te.InsertChar('e')
		te.InsertChar('2')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "Line\nLine21")
		
		// Move to line 2
		te.ExecuteCommand("down")
		te.ExecuteCommand("beginning-of-line")
		// Move cursor to position 4 (on '2')
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}
		// Repeat should insert newline + "Line2" at position 4
		te.ExecuteCommand("repeat-last-change")
		
		// Should have inserted at position 4
		AssertBufferEquals(t, te, "Line\nLine\nLine221")
	})
	
	t.Run("repeat across buffers", func(t *testing.T) {
		// Create a test editor with first buffer
		te := NewTestEditorWithContent(80, 24, "Buffer1")
		
		// Make a change in first buffer
		te.ExecuteCommand("end-of-line") // Cursor on '1' at position 6
		te.ExecuteCommand("enter-insert-mode")
		// Insert at position 6
		te.InsertChar('-')
		te.InsertChar('A')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "Buffer-A1")
		
		// Create a second buffer manually
		buf2 := buffer.NewBuffer("Buffer2", "test2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
		buf2.ViewIterator = buffer.NewBufferIterator(buf2)
		buf2.CursorIterator = buffer.NewBufferIterator(buf2)
		
		// Add to editor's buffer management
		te.buffers["test2.txt"] = buf2
		te.bufferOrder = append(te.bufferOrder, "test2.txt")
		
		// Switch to the new buffer
		te.buffer = buf2
		
		// Move to position after "Buffer"
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}
		// Now repeat the change from buffer 1
		te.ExecuteCommand("repeat-last-change")
		
		// Should have inserted "-A" before '2'
		AssertBufferEquals(t, te, "Buffer-A2")
	})
	
	t.Run("no last change", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello World")
		
		// Try to repeat when there's no last change
		te.ExecuteCommand("repeat-last-change")
		
		// Buffer should remain unchanged
		AssertBufferEquals(t, te, "Hello World")
	})
	
	t.Run("repeat preserves original command", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "ABC")
		
		// First change: insert X at position 0
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar('X')
		te.ExecuteCommand("exit-insert-mode")
		AssertBufferEquals(t, te, "XABC")
		AssertCursorAt(t, te, 0, 0) // Cursor on 'X'
		
		// Move to 'A' and repeat
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 1, 0) // Cursor on 'A'
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "XXABC")
		// After repeat, cursor is at position 2 (on 'A')
		AssertCursorAt(t, te, 2, 0)
		
		// Move to 'B' 
		te.ExecuteCommand("right") // to 'B'
		AssertCursorAt(t, te, 3, 0) // Cursor on 'B'
		// Repeat again should insert X at position 3
		te.ExecuteCommand("repeat-last-change")
		// After insert, cursor is at position 4 (on 'B')
		AssertBufferEquals(t, te, "XXAXBC")
	})
	
	t.Run("repeat dw without crash", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "word1 word2 word3 word4")
		
		// Delete first word with dw
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, "word2 word3 word4")
		AssertCursorAt(t, te, 0, 0)
		
		// Repeat with . should delete "word2 " without crashing
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "word3 word4")
		AssertCursorAt(t, te, 0, 0)
		
		// Repeat again
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "word4")
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("repeat dw with ViewIterator check", func(t *testing.T) {
		// Create a larger buffer to ensure ViewIterator is in use
		content := ""
		for i := 0; i < 50; i++ {
			content += fmt.Sprintf("Line %d with some text\n", i+1)
		}
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move to middle of buffer to ensure ViewIterator is not at line 0
		for i := 0; i < 25; i++ {
			te.ExecuteCommand("down")
		}
		
		// Ensure ViewIterator is valid
		viewGen := te.buffer.ViewIterator.Generation
		bufGen := te.buffer.Generation
		if viewGen != bufGen {
			t.Fatalf("ViewIterator generation mismatch before dw: iterator has %d, buffer has %d", viewGen, bufGen)
		}
		
		// Delete a word with dw
		te.TypeKeys("dw")
		
		// Check ViewIterator is still valid after dw
		viewGen = te.buffer.ViewIterator.Generation
		bufGen = te.buffer.Generation
		if viewGen != bufGen {
			t.Fatalf("ViewIterator generation mismatch after dw: iterator has %d, buffer has %d", viewGen, bufGen)
		}
		
		// Repeat with . - this is where the crash happens
		te.ExecuteCommand("repeat-last-change")
		
		// Check ViewIterator is still valid after repeat
		viewGen = te.buffer.ViewIterator.Generation
		bufGen = te.buffer.Generation
		if viewGen != bufGen {
			t.Fatalf("ViewIterator generation mismatch after repeat: iterator has %d, buffer has %d", viewGen, bufGen)
		}
	})
}