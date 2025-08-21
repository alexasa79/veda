package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/clipboard"
	"veda/internal/events"
)

func TestMultiCursorYank(t *testing.T) {
	t.Run("yank word at multiple cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "one two\nthree four\nfive six")

		// Create multiple cursors at beginning of each line
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at line 0, moves to line 1
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at line 1, moves to line 2
		te.PressKey(events.KeyEscape, 0)        // Exit cursor mode
		// Now we have cursors at lines 0, 1, and primary at line 2

		// Yank word at each cursor (yw)
		te.TypeKeys("yw")

		// Get clipboard content
		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("Failed to get clipboard content: %v", err)
		}

		// Should have yanked "one ", "three ", "five " (word includes trailing space)
		expected := "one \nthree \nfive "
		if content != expected {
			t.Errorf("Expected clipboard content:\n%q\nGot:\n%q", expected, content)
		}
	})

	t.Run("yank line at multiple cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "hello world\nfoo bar\nbaz qux")

		// Create multiple cursors
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at line 0, moves to line 1
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at line 1, moves to line 2
		te.PressKey(events.KeyEscape, 0)        // Exit cursor mode
		// Now we have cursors at lines 0, 1, and primary at line 2

		// Yank line at each cursor (yy)
		te.TypeKeys("yy")

		// Get clipboard content
		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("Failed to get clipboard content: %v", err)
		}

		// Should have yanked all three lines
		expected := "hello world\nfoo bar\nbaz qux"
		if content != expected {
			t.Errorf("Expected clipboard content:\n%q\nGot:\n%q", expected, content)
		}
	})

	t.Run("yank different text objects at multiple cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "func foo() {\n\treturn 123\n}\nfunc bar() {\n\treturn 456\n}")

		// Create cursors on "foo" and "bar"
		te.ExecuteCommand("enter-cursor-mode")
		// Position on "foo"
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		te.ExecuteCommand("add-virtual-cursor")

		// Move to "bar" on line 3 ("func bar() {")
		// After add-virtual-cursor, we need to navigate to the "b" in "bar"
		for i := 0; i < 2; i++ {
			te.ExecuteCommand("down")
		}
		// Move to beginning of line 3, then move to "b" in "bar" (position 5)
		te.ExecuteCommand("beginning-of-line")
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode

		// Yank word at each cursor
		te.TypeKeys("yw")

		// Get clipboard content
		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("Failed to get clipboard content: %v", err)
		}

		// Should have yanked "foo" and "bar"
		expected := "foo\nbar"
		if content != expected {
			t.Errorf("Expected clipboard content:\n%q\nGot:\n%q", expected, content)
		}
	})
}

func TestMultiCursorPaste(t *testing.T) {
	t.Run("paste single line at multiple cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")

		// Copy text to clipboard
		clipboard.Copy("XXX")

		// Create multiple cursors at end of each line
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at end of line 0, moves to line 1
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at end of line 1, moves to line 2
		te.ExecuteCommand("end-of-line")
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode
		// Now we have cursors at end of lines 0, 1, and primary at end of line 2

		// Paste at each cursor
		te.TypeKeys("p")

		// Check buffer content
		expected := "line1XXX\nline2XXX\nline3XXX"
		actual := getBufferContent(te)
		if actual != expected {
			t.Errorf("Expected buffer:\n%s\nGot:\n%s", expected, actual)
		}
	})

	t.Run("paste multi-line content at multiple cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "a:\nb:\nc:")

		// Copy multi-line text to clipboard
		clipboard.Copy("1\n2\n3")

		// Create multiple cursors after the colons
		te.ExecuteCommand("enter-cursor-mode")

		// First cursor at end of line 0
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (1,0), moves to line 1

		// Second cursor at end of line 1
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (1,1), moves to line 2

		// Move to end of line 2
		te.ExecuteCommand("end-of-line")

		te.PressKey(events.KeyEscape, 0) // Exit cursor mode

		// Paste at each cursor - should distribute lines
		te.TypeKeys("p")

		// Check buffer content
		expected := "a:1\nb:2\nc:3"
		actual := getBufferContent(te)
		if actual != expected {
			t.Errorf("Expected buffer:\n%s\nGot:\n%s", expected, actual)
		}
	})

	t.Run("paste more lines than cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "first:\nsecond:")

		// Copy multi-line text to clipboard with more lines than we have cursors
		clipboard.Copy("A\nB\nC\nD\nE")

		// Create two cursors
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("add-virtual-cursor")
		te.ExecuteCommand("end-of-line")
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode

		// Paste - first cursor gets A, second cursor gets B,C,D,E
		te.TypeKeys("p")

		actual := getBufferContent(te)

		// Check buffer content
		expected := "first:A\nsecond:B\nC\nD\nE"
		if actual != expected {
			t.Errorf("Expected buffer:\n%s\nGot:\n%s", expected, actual)
		}
	})

	t.Run("yank and paste workflow", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "apple 1\nbanana 2\ncherry 3")

		// Create multiple cursors at beginning of each line
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("add-virtual-cursor")
		te.ExecuteCommand("add-virtual-cursor")
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode

		// Yank first word at each cursor
		te.TypeKeys("yw")

		// Move to end of each line
		te.ExecuteCommand("end-of-line")

		// Paste
		te.ExecuteCommand("paste-after")

		// Check buffer content
		// Note: yw yanks word with trailing space, so we get "apple " etc.
		expected := "apple 1apple \nbanana 2banana \ncherry 3cherry "
		actual := getBufferContent(te)
		if actual != expected {
			t.Errorf("Expected buffer:\n%s\nGot:\n%s", expected, actual)
		}
	})
}

func TestMultiCursorYankInside(t *testing.T) {
	t.Run("yank inside word at multiple cursors", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "hello_world\nfoo_bar_baz\ntest_case")

		// Create multiple cursors in middle of words
		te.ExecuteCommand("enter-cursor-mode")
		// Position in "world" (line 0, position 7)
		for i := 0; i < 7; i++ {
			te.ExecuteCommand("right")
		}
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (7,0), moves to line 1

		// Position in "bar" (line 1, need to move from position 7)
		// First go to beginning of line
		te.ExecuteCommand("beginning-of-line")
		// Then move to "bar" at position 4
		for i := 0; i < 5; i++ {
			te.ExecuteCommand("right")
		}
		te.ExecuteCommand("add-virtual-cursor") // Adds cursor at (5,1), moves to line 2

		// Position in "case" (line 2)
		// First go to beginning of line
		te.ExecuteCommand("beginning-of-line")
		// Then move to "case" at position 5
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}
		te.PressKey(events.KeyEscape, 0) // Exit cursor mode

		// Yank inside word
		te.TypeKeys("yiw")

		// Get clipboard content
		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("Failed to get clipboard content: %v", err)
		}

		// Should have yanked the full words (underscore is part of word)
		expected := "hello_world\nfoo_bar_baz\ntest_case"
		if content != expected {
			t.Errorf("Expected clipboard content:\n%q\nGot:\n%q", expected, content)
		}
	})
}

// Helper function to get buffer content as string
func getBufferContent(te *TestEditor) string {
	var lines []string
	for i := 0; i < te.buffer.LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(te.buffer, i)
		lines = append(lines, iter.Line())
	}
	return strings.Join(lines, "\n")
}
