package editor

import (
	"testing"
	"veda/internal/buffer"
)

func TestArgumentTextObject(t *testing.T) {
	t.Run("daa deletes around argument", func(t *testing.T) {
		content := `result := calculate(10, 20, 30)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to second argument
		// Content: "result := calculate(10, 20, 30)"
		//          01234567890123456789012345678901
		//                                   ^24 (on '2' in '20')
		te.buffer.CursorX = 24

		// Execute daa - should delete around the second argument including comma
		te.TypeKeys("daa")

		// The second argument and surrounding commas should be deleted
		expected := `result := calculate(10, 30)`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("dia deletes inside argument", func(t *testing.T) {
		content := `result := calculate(firstValue, secondValue, thirdValue)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to middle of second argument "secondValue"
		// Content: "result := calculate(firstValue, secondValue, thirdValue)"
		//          0123456789012345678901234567890123456789012345678901234567
		//                                        ^35 (on 's' in 'secondValue')
		te.buffer.CursorX = 35

		// Execute dia - should delete only the argument content, not commas
		te.TypeKeys("dia")

		// Only the argument text should be deleted, not the commas
		expected := `result := calculate(firstValue, , thirdValue)`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("daa on first argument", func(t *testing.T) {
		content := `func(arg1, arg2, arg3)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to first argument
		for i := 0; i < 7; i++ {
			te.ExecuteCommand("right")
		}

		// Execute daa
		te.TypeKeys("daa")

		// First argument and trailing comma should be deleted
		expected := `func(arg2, arg3)`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("daa on last argument", func(t *testing.T) {
		content := `func(arg1, arg2, arg3)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to last argument
		for i := 0; i < 18; i++ {
			te.ExecuteCommand("right")
		}

		// Execute daa
		te.TypeKeys("daa")

		// Last argument and preceding comma should be deleted
		expected := `func(arg1, arg2)`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("dia on single argument", func(t *testing.T) {
		content := `print(message)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to the argument
		for i := 0; i < 9; i++ {
			te.ExecuteCommand("right")
		}

		// Execute dia
		te.TypeKeys("dia")

		// Only the argument should be deleted
		expected := `print()`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("daa with nested parentheses", func(t *testing.T) {
		content := `outer(inner(a, b), c, d)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to 'a' inside inner function
		for i := 0; i < 13; i++ {
			te.ExecuteCommand("right")
		}

		// Execute daa - should delete 'a' and comma within inner()
		te.TypeKeys("daa")

		// First argument of inner function should be deleted
		expected := `outer(inner(b), c, d)`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("daa with string containing comma", func(t *testing.T) {
		content := `func("hello, world", arg2, arg3)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to second argument
		for i := 0; i < 22; i++ {
			te.ExecuteCommand("right")
		}

		// Execute daa
		te.TypeKeys("daa")

		// Second argument should be deleted (comma in string should be ignored)
		expected := `func("hello, world", arg3)`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("argument not found outside parentheses", func(t *testing.T) {
		content := `no function call here`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to middle
		for i := 0; i < 10; i++ {
			te.ExecuteCommand("right")
		}

		// Execute daa - should do nothing
		te.TypeKeys("daa")

		// Content should be unchanged
		expected := `no function call here`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("multiline function call", func(t *testing.T) {
		content := `calculate(
    firstArg,
    secondArg,
    thirdArg
)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to third line (line 2), in middle of "secondArg"
		// Set up cursor position directly on line 2: "    secondArg,"
		te.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(te.buffer, 2)
		te.buffer.CursorX = 8 // Position on 'o' in "secondArg" (after 4 spaces + "seco")

		// Execute dia
		te.TypeKeys("dia")

		// Second argument should be deleted
		expected := `calculate(
    firstArg,
    ,
    thirdArg
)`
		AssertBufferEquals(t, te, expected)
	})
}
