package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestMultipleCursorsTextEditing(t *testing.T) {
	t.Run("insert character at multiple cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello\nWorld\nTest")

		// Enter cursor mode
		te.TypeKeys("m")

		// Add virtual cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)

		// Move down and add another virtual cursor at (0,1)
		te.PressKey(events.KeyEnter, 0)

		// Exit cursor mode
		te.TypeKeys("m")

		// Enter insert mode
		te.TypeKeys("i")

		// Type "X" - should insert at all 3 cursors
		te.TypeKeys("X")

		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)

		// Check buffer content
		expected := "XHello\nXWorld\nXTest"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
	})

	t.Run("backspace at multiple cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello\nWorld\nTest")

		// Enter cursor mode
		te.TypeKeys("m")

		// Add virtual cursor at end of "Hello" (5,0)
		te.TypeKeys("$")
		te.PressKey(events.KeyEnter, 0)

		// Now at line 1 after auto-move, go to end of "World" and add cursor
		te.TypeKeys("$")
		te.PressKey(events.KeyEnter, 0)

		// Now at line 2 after auto-move, go to end of "Test"
		te.TypeKeys("$")

		// Exit cursor mode
		te.TypeKeys("m")

		// Enter insert mode with 'a' (append) to position cursor after last char
		te.TypeKeys("a")

		// Backspace - should delete last char from each line
		te.ExecuteCommand("backspace")

		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)

		// Check buffer content
		expected := "Hell\nWorl\nTes"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
	})

	t.Run("movement commands apply to all cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "abc def\nabc def\nabc def")

		// Enter cursor mode
		te.TypeKeys("m")

		// Add virtual cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)

		// Move down and add another virtual cursor at (0,1)
		te.PressKey(events.KeyEnter, 0)

		// Exit cursor mode
		te.TypeKeys("m")

		// Move all cursors to next word
		te.TypeKeys("w")

		// Enter insert mode
		te.TypeKeys("i")

		// Type "X" - should insert at position 4 on all lines
		te.TypeKeys("X")

		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)

		// Check buffer content
		expected := "abc Xdef\nabc Xdef\nabc Xdef"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}

		// Verify virtual cursors are at correct positions
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors, got %d", len(virtualCursors))
		}
		if len(virtualCursors) >= 2 {
			// After inserting X and exiting insert mode, cursor should be at position 4
			if virtualCursors[0].X != 4 || virtualCursors[0].Y != 0 {
				t.Errorf("Expected first virtual cursor at (4,0), got (%d,%d)",
					virtualCursors[0].X, virtualCursors[0].Y)
			}
			if virtualCursors[1].X != 4 || virtualCursors[1].Y != 1 {
				t.Errorf("Expected second virtual cursor at (4,1), got (%d,%d)",
					virtualCursors[1].X, virtualCursors[1].Y)
			}
		}
	})

	t.Run("cursor mode movement only affects primary cursor", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line 1\nLine 2\nLine 3")

		// Enter cursor mode
		te.TypeKeys("m")

		// Add virtual cursor at (0,0)
		te.PressKey(events.KeyEnter, 0)

		// Move down (should only move primary cursor)
		te.TypeKeys("down")

		// Add another virtual cursor
		te.PressKey(events.KeyEnter, 0)

		// Exit cursor mode
		te.TypeKeys("m")

		// Verify virtual cursors
		virtualCursors := te.GetVirtualCursors()
		if len(virtualCursors) != 2 {
			t.Errorf("Expected 2 virtual cursors, got %d", len(virtualCursors))
		}
		if len(virtualCursors) >= 2 {
			// First cursor should still be at line 0
			if virtualCursors[0].Y != 0 {
				t.Errorf("Expected first virtual cursor on line 0, got line %d",
					virtualCursors[0].Y)
			}
			// Second cursor should be at line 2 (where we added it after moving down twice)
			// First move is automatic after adding first cursor, second is explicit "down"
			if virtualCursors[1].Y != 2 {
				t.Errorf("Expected second virtual cursor on line 2, got line %d",
					virtualCursors[1].Y)
			}
		}
	})

	t.Run("cw should work with multiple cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world\nhello world\nhello world")

		// First, move to the 'w' in "world" on first line (position 6)
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}

		// Enter cursor mode
		te.TypeKeys("m")

		// Add virtual cursor at current position (6,0)
		te.PressKey(events.KeyEnter, 0)

		// Move down and add another virtual cursor at (6,1)
		te.PressKey(events.KeyEnter, 0)

		// Exit cursor mode - now we have 3 cursors all pointing to 'w' in "world"
		te.TypeKeys("m")

		// Execute cw to change "world" to something else
		te.TypeKeys("cw")

		// Should be in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'cw', got %v", te.mode)
		}

		// Verify that words were deleted and we're in insert mode with virtual cursors preserved
		afterCw := te.GetBufferContent()
		expectedAfterCw := "hello \nhello \nhello "
		if afterCw != expectedAfterCw {
			t.Errorf("Expected buffer after cw:\n%q\nBut got:\n%q", expectedAfterCw, afterCw)
		}

		// Check that virtual cursors are preserved
		virtualCursorsAfterCw := te.GetVirtualCursors()
		if len(virtualCursorsAfterCw) != 2 {
			t.Errorf("Expected 2 virtual cursors after cw, got %d", len(virtualCursorsAfterCw))
		}

		// Type "hello" to replace "world" at all cursor positions
		te.TypeKeys("hello")

		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)

		// Expected result: all three lines should say "hello hello"
		expected := "hello hello\nhello hello\nhello hello"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
	})

	t.Run("dw followed by insert should work with multiple cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world\nhello world\nhello world")

		// First, move to the 'w' in "world" on first line (position 6)
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}

		// Enter cursor mode
		te.TypeKeys("m")

		// Add virtual cursor at current position (6,0)
		te.PressKey(events.KeyEnter, 0)

		// Move down and add another virtual cursor at (6,1)
		te.PressKey(events.KeyEnter, 0)

		// Exit cursor mode - now we have 3 cursors all pointing to 'w' in "world"
		te.TypeKeys("m")

		// Execute dw to delete "world" from all positions
		te.TypeKeys("dw")

		// Should be in normal mode after dw
		if te.mode != commands.NormalMode {
			t.Errorf("Expected normal mode after 'dw', got %v", te.mode)
		}

		// Verify that "world" was deleted from all lines
		afterDw := te.GetBufferContent()
		expectedAfterDw := "hello \nhello \nhello "
		if afterDw != expectedAfterDw {
			t.Errorf("Expected buffer after dw:\n%q\nBut got:\n%q", expectedAfterDw, afterDw)
		}

		// Check virtual cursor state after dw
		virtualCursorsAfterDw := te.GetVirtualCursors()
		if len(virtualCursorsAfterDw) == 0 {
			t.Errorf("Expected virtual cursors to be preserved after dw, but they were cleared")
		}

		// Now enter insert mode
		te.TypeKeys("i")

		// Should be in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected insert mode after 'i', got %v", te.mode)
		}

		// Type "hello" - this should work at all cursor positions
		te.TypeKeys("hello")

		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)

		// Expected result: all three lines should say "hello hello"
		expected := "hello hello\nhello hello\nhello hello"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer:\n%q\nBut got:\n%q", expected, actual)
		}
	})

	// New test: undo after multi-cursor cw restores content and cursor positions
	t.Run("undo after multi-cursor cw restores cursors", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world\nhello world\nhello world")

		// Move to the 'w' in "world"
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}

		// Enter cursor mode and add cursors on first two lines
		te.TypeKeys("m")
		te.PressKey(events.KeyEnter, 0) // add at (6,0), auto-move to line 1
		te.PressKey(events.KeyEnter, 0) // add at (6,1), auto-move to line 2
		te.TypeKeys("m")                // exit cursor mode; primary at (6,2)

		// Snapshot pre-change cursor positions
		preX, preY := te.GetCursorPosition()
		preVirtual := te.GetVirtualCursors()
		if len(preVirtual) != 2 {
			t.Fatalf("expected 2 virtual cursors before change, got %d", len(preVirtual))
		}

		// Perform cw and type replacement
		te.TypeKeys("cw")
		te.TypeKeys("hello")
		te.PressKey(events.KeyEscape, 0)

		// Single undo should revert entire multi-cursor change and restore cursors
		te.ExecuteCommand("undo")

		// Content restored
		expected := "hello world\nhello world\nhello world"
		actual := te.GetBufferContent()
		if actual != expected {
			t.Errorf("Expected buffer after undo:\n%q\nBut got:\n%q", expected, actual)
		}

		// Primary cursor restored
		AssertCursorAt(t, te, preX, preY)

		// Virtual cursors restored
		postVirtual := te.GetVirtualCursors()
		if len(postVirtual) != len(preVirtual) {
			t.Fatalf("expected %d virtual cursors after undo, got %d", len(preVirtual), len(postVirtual))
		}
		for i := range preVirtual {
			if preVirtual[i].X != postVirtual[i].X || preVirtual[i].Y != postVirtual[i].Y {
				t.Errorf("virtual cursor %d mismatch: expected (%d,%d), got (%d,%d)", i, preVirtual[i].X, preVirtual[i].Y, postVirtual[i].X, postVirtual[i].Y)
			}
		}
	})

	// New test: undoing multiple added cursors should clear them all in one step
	t.Run("undo added cursors clears all virtual cursors (after exiting cursor mode)", func(t *testing.T) {
		content := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12"
		te := NewTestEditorWithContent(80, 24, content)

		// Enter cursor mode and add 10 cursors
		te.TypeKeys("m")
		for i := 0; i < 10; i++ {
			te.PressKey(events.KeyEnter, 0)
		}
		// Exit cursor mode to finalize the composite
		te.TypeKeys("m")

		vc := te.GetVirtualCursors()
		if len(vc) != 10 {
			t.Fatalf("expected 10 virtual cursors before undo, got %d", len(vc))
		}

		// Undo once should clear all virtual cursors
		te.TypeKeys("u")
		vc = te.GetVirtualCursors()
		if len(vc) != 0 {
			t.Fatalf("expected 0 virtual cursors after undo, got %d", len(vc))
		}
	})

	// New behavior: pressing 'u' while still in cursor mode removes the last-added cursor
	t.Run("cursor mode 'u' removes last virtual cursor (step-wise)", func(t *testing.T) {
		content := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\nm\nn\no"
		te := NewTestEditorWithContent(80, 24, content)

		// Enter cursor mode and add 10 cursors
		te.TypeKeys("m")
		for i := 0; i < 10; i++ {
			te.PressKey(events.KeyEnter, 0)
		}

		vc := te.GetVirtualCursors()
		if len(vc) != 10 {
			t.Fatalf("expected 10 virtual cursors before undo, got %d", len(vc))
		}

		// Press 'u' in cursor mode; it should remove just the last-added cursor
		te.TypeKeys("u")
		vc = te.GetVirtualCursors()
		if len(vc) != 9 {
			t.Fatalf("expected 9 virtual cursors after single removal in cursor mode, got %d", len(vc))
		}
		// Still in cursor mode
		if te.mode != commands.CursorMode {
			t.Fatalf("expected to remain in cursor mode after removal with 'u'")
		}
	})

	// End-to-end: cw -> move -> cw with 10 cursors, then layered undos
	t.Run("multi-cursor cw/edit/undo chain restores cursors and content in order", func(t *testing.T) {
		// Prepare 10 identical lines: "hello world hello"
		line := "hello world hello"
		content := line
		for i := 1; i < 10; i++ {
			content += "\n" + line
		}
		te := NewTestEditorWithContent(80, 24, content)

		// Move to the 'w' in "world" on line 0
		te.TypeKeys("w")

		// Enter cursor mode and create total 10 cursors across 10 lines
		// (9 virtual cursors + 1 primary)
		te.TypeKeys("m")
		for i := 0; i < 9; i++ {
			te.PressKey(events.KeyEnter, 0)
		}
		if got := len(te.GetVirtualCursors()); got != 9 {
			t.Fatalf("expected 9 virtual cursors, got %d", got)
		}

		// Change current word (world -> hello) at all cursors
		te.TypeKeys("cw")
		te.TypeKeys("hello")
		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)
		// After first change, each line should be: "hello hello hello"
		expectedHelloHelloHello := "hello hello hello"
		expected := expectedHelloHelloHello
		for i := 1; i < 10; i++ {
			expected += "\n" + expectedHelloHelloHello
		}
		AssertBufferEquals(t, te, expected)

		// Move to the next word (the third "hello") and change it to "world"
		te.TypeKeys("w")
		te.TypeKeys("cw")
		te.TypeKeys("world")
		// Exit insert mode
		te.PressKey(events.KeyEscape, 0)
		// Buffer should now be: "hello hello world" x 10 lines
		expectedHelloHelloWorld := "hello hello world"
		expected = expectedHelloHelloWorld
		for i := 1; i < 10; i++ {
			expected += "\n" + expectedHelloHelloWorld
		}
		AssertBufferEquals(t, te, expected)

		// Ensure only one cursor remains before undo by clearing virtual cursors with ESC in normal mode
		te.PressKey(events.KeyEscape, 0)
		if got := len(te.GetVirtualCursors()); got != 0 {
			t.Fatalf("expected 0 virtual cursors after clearing with ESC, got %d", got)
		}

		// 1st undo: restores the cursors (content stays "hello hello world")
		te.TypeKeys("u")
		if got := len(te.GetVirtualCursors()); got != 9 {
			t.Fatalf("after first undo, expected 9 virtual cursors restored, got %d", got)
		}
		AssertBufferEquals(t, te, expected) // still "hello hello world"

		// 2nd undo: undo second cw (world -> hello), back to "hello hello hello"
		te.TypeKeys("u")
		expected = expectedHelloHelloHello
		for i := 1; i < 10; i++ {
			expected += "\n" + expectedHelloHelloHello
		}
		AssertBufferEquals(t, te, expected)

		// 3rd undo: undo first cw (hello -> world), back to original "hello world hello"
		te.TypeKeys("u")
		expectedOriginal := line
		for i := 1; i < 10; i++ {
			expectedOriginal += "\n" + line
		}
		AssertBufferEquals(t, te, expectedOriginal)

		// 4th undo: undo cursor additions, leaving only the primary cursor
		te.TypeKeys("u")
		if got := len(te.GetVirtualCursors()); got != 0 {
			t.Fatalf("after fourth undo, expected 0 virtual cursors, got %d", got)
		}
	})
}

func TestCursorModeUndoLikeRemovalOfLastVirtualCursor(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "l1\nl2\nl3\nl4")

	// Enter cursor mode and add 3 cursors (on lines 0,1,2; primary on line 3)
	te.TypeKeys("m")
	te.PressKey(events.KeyEnter, 0)
	te.PressKey(events.KeyEnter, 0)
	te.PressKey(events.KeyEnter, 0)

	if got := len(te.GetVirtualCursors()); got != 3 {
		t.Fatalf("expected 3 virtual cursors, got %d", got)
	}
	// Remain in cursor mode
	if te.mode != commands.CursorMode {
		t.Fatalf("expected to remain in cursor mode after adding cursors")
	}

	// Press 'u' in cursor mode -> remove last virtual cursor step-wise
	te.TypeKeys("u")
	if got := len(te.GetVirtualCursors()); got != 2 {
		t.Fatalf("after first removal, expected 2 virtual cursors, got %d", got)
	}
	// Primary should have moved up one line from line 3 to line 2
	px, py := te.GetCursorPosition()
	if py != 2 {
		t.Fatalf("primary Y after removal expected 2, got %d (X=%d)", py, px)
	}

	// Remove again
	te.TypeKeys("u")
	if got := len(te.GetVirtualCursors()); got != 1 {
		t.Fatalf("after second removal, expected 1 virtual cursor, got %d", got)
	}
	_, py = te.GetCursorPosition()
	if py != 1 {
		t.Fatalf("primary Y after second removal expected 1, got %d", py)
	}

	// Exit cursor mode and verify that Normal-mode 'u' still undoes the composite (clears all)
	te.PressKey(events.KeyEscape, 0)
	// Now undo once should clear all added cursors from the session
	te.TypeKeys("u")
	if got := len(te.GetVirtualCursors()); got != 0 {
		t.Fatalf("after normal-mode undo, expected 0 virtual cursors, got %d", got)
	}
}
