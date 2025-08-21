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
		te.ExecuteCommand("end-of-line") // Cursor on '1' at position 4
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
		te.ExecuteCommand("right")  // to 'B'
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

	t.Run("repeat dw with undo inserts correct text", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "apple banana cherry")

		// Delete "apple "
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, "banana cherry")

		// Repeat to delete "banana "
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "cherry")

		// Undo should restore "banana ", not "apple "
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "banana cherry")

		// Another undo should restore "apple "
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "apple banana cherry")
	})

	t.Run("repeat same text object type", func(t *testing.T) {
		// Our implementation repeats the exact same text object, not a smart contextual one
		te := NewTestEditorWithContent(80, 24, "func test() { return 42 } and { more }")

		// Move to inside the first braces
		for i := 0; i < 14; i++ {
			te.ExecuteCommand("right")
		}

		// Delete inside braces
		te.TypeKeys("di{")
		AssertBufferEquals(t, te, "func test() {} and { more }")

		// Move to inside second braces
		te.ExecuteCommand("end-of-line")
		for i := 0; i < 3; i++ {
			te.ExecuteCommand("left")
		}

		// Repeat should delete inside the second braces
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "func test() {} and {}")
	})

	t.Run("repeat diw on different words", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world foo bar")

		// Move to middle of "hello"
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")

		// Delete inside word (just "hello", not the space)
		te.TypeKeys("diw")
		AssertBufferEquals(t, te, " world foo bar")
		AssertCursorAt(t, te, 0, 0) // Cursor at space

		// Move to middle of "world"
		for i := 0; i < 3; i++ {
			te.ExecuteCommand("right")
		}

		// Repeat should delete "world"
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "  foo bar")

		// Undo should restore "world"
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, " world foo bar")
	})

	t.Run("repeat dd line deletion", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3\nline4")

		// Delete first line
		te.TypeKeys("dd")
		AssertBufferEquals(t, te, "line2\nline3\nline4")

		// Move to line3
		te.ExecuteCommand("down")

		// Repeat should delete current line (line3)
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "line2\nline4")

		// Undo should restore line3
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "line2\nline3\nline4")

		// Another undo should restore line1
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "line1\nline2\nline3\nline4")
	})

	t.Run("repeat dW deletes to end of current word consistently", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "one->two three")

		// Start at beginning and delete to end of first word ("one")
		te.TypeKeys("dW")
		AssertBufferEquals(t, te, "->two three")
		AssertCursorAt(t, te, 0, 0)

		// Repeat should delete the next punctuation character '-'
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, ">two three")
		AssertCursorAt(t, te, 0, 0)

		// Repeat again should delete the next punctuation character '>'
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "two three")
		AssertCursorAt(t, te, 0, 0)

		// Repeat again should delete "two"
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, " three")
	})

	t.Run("repeat daw includes surrounding whitespace", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "one two three four")

		// Move to "two"
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}

		// Delete around word (includes trailing space)
		te.TypeKeys("daw")
		AssertBufferEquals(t, te, "one three four")

		// Move to "three"
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}

		// Repeat
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "one four")

		// Undo should restore "three "
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "one three four")
	})

	t.Run("delete character command", func(t *testing.T) {
		// First test delete-character on its own
		te := NewTestEditorWithContent(80, 24, "world test")

		// Delete character at cursor (should delete 'w')
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "orld test")
		AssertCursorAt(t, te, 0, 0) // Cursor should stay at position 0, now on 'o'

		// Delete another character (should delete 'o')
		te.ExecuteCommand("delete-current-character")
		AssertBufferEquals(t, te, "rld test")
		AssertCursorAt(t, te, 0, 0) // Cursor still at 0, now on 'r'
	})

	t.Run("repeat clears operator command", func(t *testing.T) {
		// This test verifies that non-operator commands clear the lastCommand
		te := NewTestEditorWithContent(80, 24, "abc def ghi")

		// Do an operator command (delete word)
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, "def ghi")

		// Verify lastCommand is set
		if te.lastCommand == nil {
			t.Fatal("lastCommand should be set after dw")
		}

		// Do a non-operator command (insert mode and add text)
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar('X')
		te.ExecuteCommand("exit-insert-mode")
		AssertBufferEquals(t, te, "Xdef ghi")

		// Verify lastCommand is cleared
		if te.lastCommand != nil {
			t.Fatal("lastCommand should be nil after insert mode")
		}

		// Move cursor and repeat - should repeat the insert, not the dw
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "XdXef ghi")
	})

	t.Run("repeat operator on different lines", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "word1 extra\nword2 stuff\nword3 things")

		// Delete first word
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, "extra\nword2 stuff\nword3 things")

		// Move to word2
		te.ExecuteCommand("down")

		// Repeat should delete "word2 "
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "extra\nstuff\nword3 things")

		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "extra\nword2 stuff\nword3 things")
	})

	t.Run("repeat cw change word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "a 10 b\nc 10 d")

		// Move cursor to "1" in the first line (position 2)
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 0)

		// Change word: cw to change "10" to "20"
		te.TypeKeys("cw")
		te.InsertChar('2')
		te.InsertChar('0')
		te.ExecuteCommand("exit-insert-mode")

		// Verify first line changed correctly
		// Note: cw on "10" should delete just "10" (not the space),
		// so we get "a 20 b"
		AssertBufferEquals(t, te, "a 20 b\nc 10 d")

		// Move down to "1" in the second line
		te.ExecuteCommand("down")
		te.ExecuteCommand("beginning-of-line")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 1)

		// Repeat the change with .
		te.ExecuteCommand("repeat-last-change")

		// Should result in "c 20 d"
		AssertBufferEquals(t, te, "a 20 b\nc 20 d")
	})

	t.Run("repeat cw with iterator validation", func(t *testing.T) {
		// Test to ensure iterators are properly updated during cw repeat
		te := NewTestEditorWithContent(80, 24, "hello world\nfoo bar\nbaz qux")

		// Move to "world"
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}

		// Change "world" to "test"
		te.TypeKeys("cw")
		te.InsertChar('t')
		te.InsertChar('e')
		te.InsertChar('s')
		te.InsertChar('t')
		te.ExecuteCommand("exit-insert-mode")

		AssertBufferEquals(t, te, "hello test\nfoo bar\nbaz qux")

		// Move to "bar" on second line
		te.ExecuteCommand("down")
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}

		// Verify iterators are valid before repeat
		if te.buffer.CursorIterator.Generation != te.buffer.Generation {
			t.Fatalf("CursorIterator generation mismatch before repeat: iterator has %d, buffer has %d",
				te.buffer.CursorIterator.Generation, te.buffer.Generation)
		}
		if te.buffer.ViewIterator.Generation != te.buffer.Generation {
			t.Fatalf("ViewIterator generation mismatch before repeat: iterator has %d, buffer has %d",
				te.buffer.ViewIterator.Generation, te.buffer.Generation)
		}

		// Repeat the change
		te.ExecuteCommand("repeat-last-change")

		// Verify iterators are still valid after repeat
		if te.buffer.CursorIterator.Generation != te.buffer.Generation {
			t.Fatalf("CursorIterator generation mismatch after repeat: iterator has %d, buffer has %d",
				te.buffer.CursorIterator.Generation, te.buffer.Generation)
		}
		if te.buffer.ViewIterator.Generation != te.buffer.Generation {
			t.Fatalf("ViewIterator generation mismatch after repeat: iterator has %d, buffer has %d",
				te.buffer.ViewIterator.Generation, te.buffer.Generation)
		}

		AssertBufferEquals(t, te, "hello test\nfoo test\nbaz qux")
	})

	t.Run("repeat command with count prefix", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "abcdefghijklmnop")

		// Manually execute "3x" by setting the count and executing delete command
		te.repeatCount = 3
		te.lastCommandCount = 3
		te.executeWithRepeatCount("delete-current-character")
		AssertBufferEquals(t, te, "defghijklmnop")
		AssertCursorAt(t, te, 0, 0) // Cursor should be at 'd'

		// Move to position 2 (on 'f')
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 2, 0) // Cursor on 'f'

		// Repeat with "." should delete 3 characters again
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "deijklmnop")
		AssertCursorAt(t, te, 2, 0) // Cursor should be on 'i'
	})

	t.Run("repeat simple command with count", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello\nworld")

		// Manually execute "3>>" (indent 3 times) by setting count and executing
		te.repeatCount = 3
		te.lastCommandCount = 3
		te.executeWithRepeatCount("indent-line")
		// Each indent should add the configured indent (assuming default of 4 spaces)
		// So 3 indents should add 12 spaces total
		expectedContent := "            hello\nworld" // 12 spaces added
		AssertBufferEquals(t, te, expectedContent)

		// Move to second line to test repeat
		te.ExecuteCommand("down")
		te.ExecuteCommand("beginning-of-line") // Move to start of line
		AssertCursorAt(t, te, 0, 1)            // Cursor at 'w' on line 1

		// Repeat should indent 3 times again
		te.ExecuteCommand("repeat-last-change")
		expectedContent = "            hello\n            world" // 12 spaces on second line
		AssertBufferEquals(t, te, expectedContent)
	})
}
