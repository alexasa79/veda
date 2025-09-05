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

	t.Run("yank word then movement works with multiple cursors", func(t *testing.T) {
		// Prepare two lines with a word on each
		te := NewTestEditorWithContent(80, 24, "alpha beta\ngamma delta")

		// Create two cursors at the start of each line
		te.ExecuteCommand("enter-cursor-mode")
		te.ExecuteCommand("add-virtual-cursor") // add at (0,0), move to line 1
		te.PressKey(events.KeyEscape, 0)        // Exit cursor mode; primary at (0,1)

		// Yank word at each cursor (yw)
		te.TypeKeys("yw")

		// Clipboard should contain both words (with trailing spaces due to 'w' semantics)
		if content, err := clipboard.Paste(); err == nil {
			expectedClip := "alpha \ngamma "
			if content != expectedClip {
				t.Errorf("clipboard mismatch after multi-cursor yw:\nexpected=%q\nactual=%q", expectedClip, content)
			}
		} else {
			t.Fatalf("Failed to read clipboard after yank: %v", err)
		}

		// Post-yank navigation: right arrow should move both cursors by 1
		// (verifies pending operator was cleared and movement applies to all cursors)
		te.TypeKeys("right")

		// Primary moved to x=1 on line 1
		px, py := te.GetCursorPosition()
		if py != 1 || px != 1 {
			t.Errorf("primary cursor not moved correctly after yank+right: got (%d,%d), want (1,1)", px, py)
		}
		// Virtual moved to x=1 on line 0
		vcs := te.GetVirtualCursors()
		if len(vcs) != 1 || vcs[0].Y != 0 || vcs[0].X != 1 {
			t.Errorf("virtual cursor not moved correctly after yank+right: got %+v, want [{X:1 Y:0}]", vcs)
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

		// Verify cursor positions: primary should remain on the last line,
		// virtual cursors should remain on their respective lines.
		_, py := te.GetCursorPosition()
		if py != 2 {
			t.Errorf("Primary cursor Y after multi-line paste: got %d, want %d", py, 2)
		}
		vcs := te.GetVirtualCursors()
		if len(vcs) != 2 {
			t.Fatalf("Expected 2 virtual cursors, got %d", len(vcs))
		}
		// Collect Y positions
		var ys []int
		for _, vc := range vcs { ys = append(ys, vc.Y) }
		// The two virtual cursors should be on lines 0 and 1 (order not guaranteed)
		found0, found1 := false, false
		for _, y := range ys { if y == 0 { found0 = true }; if y == 1 { found1 = true } }
		if !found0 || !found1 {
			t.Errorf("Virtual cursor Ys after paste: got %v, want 0 and 1", ys)
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

func TestMultiCursorBracketedPaste_InsertMode(t *testing.T) {
    // Set up three lines with cursors on each line
    te := NewTestEditorWithContent(80, 24, "a\nb\nc")

    // Enter cursor mode and position a cursor at end of each line
    te.ExecuteCommand("enter-cursor-mode")
    te.ExecuteCommand("end-of-line")
    te.ExecuteCommand("add-virtual-cursor") // line 0
    te.ExecuteCommand("end-of-line")
    te.ExecuteCommand("add-virtual-cursor") // line 1
    te.ExecuteCommand("end-of-line")        // line 2
    te.PressKey(events.KeyEscape, 0)         // Exit cursor mode

    // Enter insert mode at end-of-line for all cursors (equivalent to Vim 'A')
    te.TypeKeys("A")

    // Simulate bracketed paste (Cmd+V) of single-line text "XYZ"
    te.handleEvent(events.NewPasteEvent(true))
    for _, r := range []rune("XYZ") {
        te.handleEvent(events.NewKeyEvent(events.KeyRune, r, events.ModNone))
    }
    te.handleEvent(events.NewPasteEvent(false))

    // Each cursor should receive the pasted text at its position
    expected := "aXYZ\nbXYZ\ncXYZ"
    actual := getBufferContent(te)
    if actual != expected {
        t.Errorf("After bracketed paste in insert mode, expected:\n%q\nGot:\n%q", expected, actual)
    }
}

func TestMultiCursorYankThenBracketedPaste_InsertMode(t *testing.T) {
    // Prepare lines with two words so 'yw' yanks the first word + trailing space
    te := NewTestEditorWithContent(80, 24, "one two\nthree four\nfive six")

    // Create cursors at the beginning of each line
    te.ExecuteCommand("enter-cursor-mode")
    te.ExecuteCommand("add-virtual-cursor") // line 0
    te.ExecuteCommand("add-virtual-cursor") // line 1
    te.PressKey(events.KeyEscape, 0)        // Exit cursor mode; primary at line 2

    // Yank first word at each cursor (yw)
    te.TypeKeys("yw")

    // Get clipboard content and choose a single-line slice to paste via bracketed paste
    clip, err := clipboard.Paste()
    if err != nil || clip == "" {
        t.Fatalf("clipboard read failed after multi-cursor yank: %v", err)
    }
    parts := strings.Split(clip, "\n")
    first := parts[0] // e.g., "one "

    // Enter insert mode at end of line for all cursors ('A')
    te.TypeKeys("A")

    // Simulate bracketed paste of the first yanked word
    te.handleEvent(events.NewPasteEvent(true))
    for _, r := range []rune(first) {
        te.handleEvent(events.NewKeyEvent(events.KeyRune, r, events.ModNone))
    }
    te.handleEvent(events.NewPasteEvent(false))

    // Each line should have the first yanked word appended
    expected := "one two" + first + "\n" + "three four" + first + "\n" + "five six" + first
    actual := getBufferContent(te)
    if actual != expected {
        t.Errorf("After yw + bracketed paste, expected:\n%q\nGot:\n%q", expected, actual)
    }
}

// This test pastes the FULL multi-line clipboard (including newlines) across multiple cursors
// in insert mode, reproducing the real-world Cmd+V behavior that currently breaks by applying
// the entire stream at every cursor. The expected (correct) behavior would be to distribute
// one yanked line per cursor. This test asserts that desired behavior and will fail until the
// bracketed paste path buffers and routes to pasteForAllCursors for multi-line content.
func TestMultiCursorYankThenBracketedPaste_InsertMode_FullClipboard(t *testing.T) {
    te := NewTestEditorWithContent(80, 24, "one\ntwo\nthree")

    // Create cursors at the beginning of each line
    te.ExecuteCommand("enter-cursor-mode")
    te.ExecuteCommand("add-virtual-cursor") // line 0
    te.ExecuteCommand("add-virtual-cursor") // line 1
    te.PressKey(events.KeyEscape, 0)        // Exit cursor mode; primary at line 2

    // Yank first word at each cursor (yw) -> clipboard becomes multi-line
    te.TypeKeys("yw")
    clip, err := clipboard.Paste()
    if err != nil || clip == "" {
        t.Fatalf("clipboard read failed after multi-cursor yank: %v", err)
    }

    // Enter insert mode at end of line for all cursors ('A')
    te.TypeKeys("A")

    // Simulate bracketed paste of the full multi-line clipboard
    te.handleEvent(events.NewPasteEvent(true))
    for _, r := range []rune(clip) {
        te.handleEvent(events.NewKeyEvent(events.KeyRune, r, events.ModNone))
    }
    te.handleEvent(events.NewPasteEvent(false))

    // Desired behavior: distribute one yanked line per cursor, appended at EOL (no trailing spaces)
    expected := "oneone\ntwotwo\nthreethree"
    actual := getBufferContent(te)
    if actual != expected {
        t.Errorf("Full clipboard bracketed paste across multi-cursors incorrect.\nExpected:\n%q\nGot:\n%q\nClipboard was:\n%q", expected, actual, clip)
    }
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

// Reproduces a cursor placement issue after multi-cursor bracketed paste in insert mode.
// Steps:
// 1) Three lines: 1, 2, 3; place a cursor on each line
// 2) Yank current word at each cursor (clipboard = "1\n2\n3")
// 3) Press 'A' to append at end-of-line for all cursors (enter insert mode)
// 4) Cmd+V bracketed paste the full clipboard (distribute across cursors)
// 5) Type "hello" while still in insert mode
// Expected correct behavior: numbers are duplicated and then "hello" is appended: 11hello, 22hello, 33hello
// Current buggy behavior: "hello" is inserted before the last character (e.g., 1hello1, 2hello2, 3hello3)
func TestMultiCursorBracketedPasteThenType_InsertMode_CursorAfterPaste(t *testing.T) {
    te := NewTestEditorWithContent(80, 24, "1\n2\n3")

    // Create a cursor on each line
    te.ExecuteCommand("enter-cursor-mode")
    te.ExecuteCommand("add-virtual-cursor") // line 0
    te.ExecuteCommand("add-virtual-cursor") // line 1
    te.PressKey(events.KeyEscape, 0)

    // Yank current word at each cursor (multi-line clipboard: "1\n2\n3")
    te.TypeKeys("yw")
    clip, err := clipboard.Paste()
    if err != nil || clip == "" {
        t.Fatalf("clipboard read failed after multi-cursor yank: %v", err)
    }

    // Append at end-of-line for all cursors and remain in insert mode
    te.TypeKeys("A")

    // Bracketed paste: simulate Cmd+V of the full clipboard content
    te.handleEvent(events.NewPasteEvent(true))
    for _, r := range []rune(clip) {
        te.handleEvent(events.NewKeyEvent(events.KeyRune, r, events.ModNone))
    }
    te.handleEvent(events.NewPasteEvent(false))

    // Now type while still in insert mode
    te.TypeKeys("hello")

    // Desired correct behavior (this will fail until cursor placement bug is fixed):
    expected := "11hello\n22hello\n33hello"
    actual := getBufferContent(te)
    if actual != expected {
        t.Errorf("After paste then typing in insert mode, expected (correct):\n%q\nGot:\n%q\nThis reproduces a cursor-off-by-one bug where typing occurs one char before EOL.", expected, actual)
    }
}

// Helper function to get buffer content as string
func getBufferContent(te *TestEditor) string {
	var lines []string
	for i := 0; i < te.CurrentBuffer().LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), i)
		lines = append(lines, iter.Line())
	}
	return strings.Join(lines, "\n")
}
