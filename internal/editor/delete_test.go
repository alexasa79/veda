package editor

import (
	"testing"
	"veda/internal/buffer"
)

func TestDeleteOperations(t *testing.T) {
	t.Run("dd deletes current line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Move to line 2
		te.ExecuteCommand("down")
		AssertCursorAt(t, te, 0, 1)

		// Execute dd
		te.TypeKeys("dd")

		// Line 2 should be deleted
		AssertBufferEquals(t, te, "Line 1\nLine 3")
		// Cursor should be on what was line 3 (now line 2)
		AssertCursorAt(t, te, 0, 1)
	})

	t.Run("dd on last line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Move to last line
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")

		// Execute dd
		te.TypeKeys("dd")

		// Last line should be deleted
		AssertBufferEquals(t, te, "Line 1\nLine 2")
		// Cursor should be on line 2
		AssertCursorAt(t, te, 0, 1)
	})

	t.Run("dd on single line file", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Only line")

		// Execute dd
		te.TypeKeys("dd")

		// Buffer should be empty
		AssertBufferEquals(t, te, "")
	})

	t.Run("dw deletes current word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello world test")

		// Execute dw at beginning
		te.TypeKeys("dw")

		// Should delete "Hello " (word + trailing space)
		AssertBufferEquals(t, te, "world test")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dw in middle of word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello world test")

		// Move to middle of "Hello"
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")

		// Execute dw
		te.TypeKeys("dw")

		// Should delete entire word "Hello " (word + trailing space)
		AssertBufferEquals(t, te, "world test")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dW deletes from cursor to end of current word segment", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "func(arg1, arg2) result")

		// Execute dW at beginning
		te.TypeKeys("dW")

		// Should delete from cursor to end of current word ("func")
		AssertBufferEquals(t, te, "(arg1, arg2) result")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("diw deletes inside word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello world test")

		// Move to middle of "world"
		for i := 0; i < 7; i++ {
			te.ExecuteCommand("right")
		}

		// Execute diw
		te.TypeKeys("diw")

		// Should delete "world" but not spaces
		AssertBufferEquals(t, te, "Hello  test")
	})

	t.Run("daw deletes around word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello world test")

		// Move to middle of "world"
		for i := 0; i < 7; i++ {
			te.ExecuteCommand("right")
		}

		// Execute daw
		te.TypeKeys("daw")

		// Should delete "world" and trailing space
		AssertBufferEquals(t, te, "Hello test")
	})

	t.Run("di{ deletes inside braces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "func() {\n\tcode here\n}")

		// Move inside the braces
		te.ExecuteCommand("down")

		// Execute di{
		te.TypeKeys("di{")

		// Should delete content inside braces
		AssertBufferEquals(t, te, "func() {\n}")
	})

	t.Run("da{ deletes around braces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "if (x) { code } else")

		// Move cursor inside braces
		for i := 0; i < 10; i++ {
			te.ExecuteCommand("right")
		}

		// Execute da{
		te.TypeKeys("da{")

		// Should delete braces and content
		AssertBufferEquals(t, te, "if (x)  else")
	})

	t.Run("di( deletes inside parentheses", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "function(arg1, arg2)")

		// Move cursor inside parentheses
		for i := 0; i < 10; i++ {
			te.ExecuteCommand("right")
		}

		// Execute di(
		te.TypeKeys("di(")

		// Should delete content inside parentheses
		AssertBufferEquals(t, te, "function()")
	})

	t.Run("escape cancels pending delete", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello world")

		// Start delete operation but cancel
		te.TypeKeys("d")
		te.PressKey(KeyEscape, 0)

		// Nothing should be deleted
		AssertBufferEquals(t, te, "Hello world")

		// Should be able to move normally
		te.ExecuteCommand("right")
		AssertCursorAt(t, te, 1, 0)
	})

	t.Run("delete operations are undoable", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello world")

		// Delete word
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, "world")

		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Hello world")

		// Redo
		te.ExecuteCommand("redo")
		AssertBufferEquals(t, te, "world")
	})

	t.Run("delete operations are repeatable", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "word1 word2 word3")

		// Delete first word
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, "word2 word3")

		// Repeat with .
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "word3")
	})

	t.Run("dw on leading whitespace deletes only spaces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "    hello world")

		// Cursor is at beginning of line (on first space)
		AssertCursorAt(t, te, 0, 0)

		// Execute dw - should delete only the 4 spaces, not "hello"
		te.TypeKeys("dw")

		// Should leave "hello world" with cursor at 'h'
		AssertBufferEquals(t, te, "hello world")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dw on tabs deletes only tabs", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "\t\thello world")

		// Cursor is at beginning of line (on first tab)
		AssertCursorAt(t, te, 0, 0)

		// Execute dw - should delete only the tabs, not "hello"
		te.TypeKeys("dw")

		// Should leave "hello world" with cursor at 'h'
		AssertBufferEquals(t, te, "hello world")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dw on mixed whitespace deletes only whitespace", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "\t  \thello world")

		// Cursor is at beginning of line (on first tab)
		AssertCursorAt(t, te, 0, 0)

		// Execute dw - should delete only the whitespace, not "hello"
		te.TypeKeys("dw")

		// Should leave "hello world" with cursor at 'h'
		AssertBufferEquals(t, te, "hello world")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dW in middle of word deletes to end of that word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "func(arg1, arg2) result")

		// Move to middle of first WORD
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")

		// Execute dW
		te.TypeKeys("dW")

		// Should delete from cursor to end of current word only (removes 'c')
		AssertBufferEquals(t, te, "fun(arg1, arg2) result")
	})

	t.Run("dw on whitespace", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "word1   word2")

		// Move to whitespace
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}

		// Execute dw on whitespace
		te.TypeKeys("dw")

		// Should delete only whitespace, not the next word
		AssertBufferEquals(t, te, "word1word2")
	})

	t.Run("dw twice on quoted string", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, `Enable: "false is the value"`)

		// Move to 'f' in "false"
		for i := 0; i < 9; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 9, 0)

		// First dw should delete "false " (word + space)
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, `Enable: "is the value"`)
		AssertCursorAt(t, te, 9, 0) // Cursor should stay at 'i'

		// Second dw should delete "is " (word + space)
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, `Enable: "the value"`)
		AssertCursorAt(t, te, 9, 0) // Cursor should still be at same position
	})

	t.Run("dw at end of word deletes next word", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "word1 word2 word3")

		// Move to end of "word1" (on '1')
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 4, 0)

		// dw should delete "word1 " (word + space)
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, "word2 word3")
		AssertCursorAt(t, te, 0, 0)

		// Now on 'w', dw should delete "word2 "
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, "word3")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dw includes trailing whitespace like vim", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, `debugFlag := flag.Bool("d", false, "Enable debug logging")`)

		// Move to 'f' in "false"
		for i := 0; i < 28; i++ {
			te.ExecuteCommand("right")
		}
		AssertCursorAt(t, te, 28, 0)

		// First dw should delete "false" and the space after it
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, `debugFlag := flag.Bool("d", , "Enable debug logging")`)
		AssertCursorAt(t, te, 28, 0) // Cursor at comma

		// Second dw should delete just the comma (punctuation)
		te.TypeKeys("dw")
		AssertBufferEquals(t, te, `debugFlag := flag.Bool("d", "Enable debug logging")`)
		AssertCursorAt(t, te, 28, 0) // Cursor at space
	})

	// New tests: deletion stores into 'd' register and 'dp pastes it back
	t.Run("visual line delete then paste from d register", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3\nLine 4")
		// Select lines 2 and 3
		_ = te.ExecuteCommand("down")
		_ = te.ExecuteCommand("enter-visual-line-mode")
		_ = te.ExecuteCommand("down")
		// Delete selection (stores into 'd' register)
		_ = te.ExecuteCommand("delete-selection")
		// Now paste from delete register after cursor using "'dp"
		te.TypeKeys("'")
		te.TypeKeys("d")
		te.TypeKeys("p")
		AssertBufferEquals(t, te, "Line 1\nLine 2\nLine 3\nLine 4")
	})

	t.Run("dd then paste from d register", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "a\nb\nc")
		// dd delete first line
		te.TypeKeys("dd")
		// paste from delete register
		te.TypeKeys("'")
		te.TypeKeys("d")
		te.TypeKeys("p")
		AssertBufferEquals(t, te, "a\nb\nc")
	})

	t.Run("daa then paste from d register", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "func(a, b, c)")
		// Move cursor to middle of b
		for i := 0; i < 8; i++ {
			_ = te.ExecuteCommand("right")
		}
		// daa delete around argument
		te.TypeKeys("daa")
		// paste the deleted text back
		te.TypeKeys("'")
		te.TypeKeys("d")
		te.TypeKeys("p")
		// Result should restore original
		AssertBufferEquals(t, te, "func(a, b, c)")
	})

	t.Run("go tabs: delete function body and paste back from d register keeps tab indent", func(t *testing.T) {
		// Go code with tab-indented body (L1)
		content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n\tprintln(\"world\")\n}\n"
		te := NewTestEditorWithContent(80, 24, content)
        // Set filename to .go so highlighter selects Go (enables auto-indent code path)
        te.CurrentBuffer().FilePath = "test.go"
        if te.CurrentBuffer().Highlighter != nil {
            _ = te.CurrentBuffer().Highlighter.SetLanguage("go")
        }

		// Move cursor to first line inside function body (line 3)
		// Lines: 0:package,1:blank,2:func...,3:\tprintln("hello"),4:\tprintln("world"),5:}
		te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 3)
		te.CurrentBuffer().CursorX = 0

		// Enter visual line mode and select both body lines
		_ = te.ExecuteCommand("enter-visual-line-mode")
		_ = te.ExecuteCommand("down")

		// Delete selection (stores into 'd' register)
		_ = te.ExecuteCommand("delete-selection")

		// Paste back from delete register using "'dp"
		te.TypeKeys("'")
		te.TypeKeys("d")
		te.TypeKeys("p")

		// Expect original content restored with tabs preserved
		AssertBufferEquals(t, te, content)
	})
}
