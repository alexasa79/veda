package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestDeleteWithFindMotion(t *testing.T) {
	t.Run("df - delete to character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world test")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "df " (delete to space)
		te.TypeKeys("df ")

		// Should delete "hello " leaving "world test"
		AssertBufferEquals(t, te, "world test")
		AssertCursorAt(t, te, 0, 0) // Cursor at beginning
	})

	t.Run("df) - delete to closing paren", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "function(arg1, arg2) {")

		// Start at 'f' in function
		AssertCursorAt(t, te, 0, 0)

		// Execute "df)" - delete to closing paren
		te.TypeKeys("df)")

		// Should delete "function(arg1, arg2)" leaving " {"
		AssertBufferEquals(t, te, " {")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("df - character not found", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "dfx" - character not found
		te.TypeKeys("dfx")

		// Buffer should be unchanged
		AssertBufferEquals(t, te, "hello world")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dff - delete to 'f' character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "find first function")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "dff" - delete to first 'f'
		te.TypeKeys("dff")

		// Should delete "findf" leaving "irst function"
		AssertBufferEquals(t, te, "irst function")
		AssertCursorAt(t, te, 0, 0)
	})
}

func TestFindCharacterMovementWithMultipleCursors(t *testing.T) {
	// Three identical lines with a space and then 'd' at index 4
	te := NewTestEditorWithContent(80, 24, "abc def\nabc def\nabc def")

	// Enter cursor mode and add two virtual cursors on subsequent lines via Enter
	te.TypeKeys("m")
	te.PressKey(events.KeyEnter, 0) // add cursor on line 0
	te.PressKey(events.KeyEnter, 0) // auto-moved to line 1, add cursor
	te.TypeKeys("m")                // exit cursor mode; now 3 cursors at col 0 on lines 0,1,2

	// Simulate real keystrokes: "f" then "d"
	te.TypeKeys("f")
	te.TypeKeys("d")

	// Verify all cursors are at column 4 (on 'd') for their respective lines
	// Primary cursor (after adding cursors, primary is on last line)
	x, y := te.GetCursorPosition()
	if x != 4 || y != 2 {
		t.Fatalf("primary cursor expected at (4,2), got (%d,%d)", x, y)
	}
	// Virtual cursors
	vcs := te.GetVirtualCursors()
	if len(vcs) != 2 {
		t.Fatalf("expected 2 virtual cursors, got %d", len(vcs))
	}
	if vcs[0].X != 4 || vcs[0].Y != 0 {
		t.Fatalf("virtual cursor 0 expected at (4,0), got (%d,%d)", vcs[0].X, vcs[0].Y)
	}
	if vcs[1].X != 4 || vcs[1].Y != 1 {
		t.Fatalf("virtual cursor 1 expected at (4,1), got (%d,%d)", vcs[1].X, vcs[1].Y)
	}
}

func TestDeleteWithTillMotion(t *testing.T) {
	t.Run("dt - delete till character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world test")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "dt " (delete till space)
		te.TypeKeys("dt ")

		// Should delete "hello" leaving " world test"
		AssertBufferEquals(t, te, " world test")
		AssertCursorAt(t, te, 0, 0) // Cursor at beginning
	})

	t.Run("dt) - delete till closing paren", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "function(arg1, arg2) {")

		// Start at 'f' in function
		AssertCursorAt(t, te, 0, 0)

		// Execute "dt)" - delete till closing paren
		te.TypeKeys("dt)")

		// Should delete "function(arg1, arg2" leaving ") {"
		AssertBufferEquals(t, te, ") {")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dt - character not found", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "dtx" - character not found
		te.TypeKeys("dtx")

		// Buffer should be unchanged
		AssertBufferEquals(t, te, "hello world")
		AssertCursorAt(t, te, 0, 0)
	})

	t.Run("dtt - delete till 't' character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "start test text")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "dtt" - delete till first 't' (should be the 't' at position 1 in "start")
		te.TypeKeys("dtt")

		// Should delete "s" leaving "tart test text"
		AssertBufferEquals(t, te, "tart test text")
		AssertCursorAt(t, te, 0, 0)
	})
}

func TestChangeWithFindMotion(t *testing.T) {
	t.Run("cf - change to character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world test")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "cf " (change to space)
		te.TypeKeys("cf ")

		// Should delete "hello " and enter insert mode
		AssertBufferEquals(t, te, "world test")
		AssertCursorAt(t, te, 0, 0)
		// Should be in insert mode now
		if te.mode != commands.InsertMode {
			t.Errorf("Expected INSERT mode after change operation, got %v", te.mode)
		}
	})

	t.Run("cf) - change to closing paren", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "function(old) {")

		// Move cursor to 'o' in "old"
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		te.ExecuteCommand("right") // Now at 'o'
		AssertCursorAt(t, te, 9, 0)

		// Execute "cf)" - change to closing paren
		te.TypeKeys("cf)")

		// Should delete "old)" and enter insert mode
		AssertBufferEquals(t, te, "function( {")
		AssertCursorAt(t, te, 9, 0)
		if te.mode != commands.InsertMode {
			t.Errorf("Expected INSERT mode after change operation, got %v", te.mode)
		}
	})
}

func TestChangeWithTillMotion(t *testing.T) {
	t.Run("ct - change till character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world test")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "ct " (change till space)
		te.TypeKeys("ct ")

		// Should delete "hello" and enter insert mode, leaving " world test"
		AssertBufferEquals(t, te, " world test")
		AssertCursorAt(t, te, 0, 0)
		if te.mode != commands.InsertMode {
			t.Errorf("Expected INSERT mode after change operation, got %v", te.mode)
		}
	})
}

func TestYankWithFindMotion(t *testing.T) {
	t.Run("yf - yank to character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world test")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "yf " (yank to space)
		te.TypeKeys("yf ")

		// Buffer should be unchanged
		AssertBufferEquals(t, te, "hello world test")
		AssertCursorAt(t, te, 0, 0)

		// Move to end and paste to verify yank worked
		te.ExecuteCommand("end-of-line")
		te.TypeKeys("p") // paste after

		// Should have "hello world testhello "
		AssertBufferEquals(t, te, "hello world testhello ")
	})

	t.Run("yf) - yank to closing paren", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "function(arg) {")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "yf)" - yank to closing paren
		te.TypeKeys("yf)")

		// Buffer should be unchanged
		AssertBufferEquals(t, te, "function(arg) {")

		// Move to end and paste to verify yank worked
		te.ExecuteCommand("end-of-line")
		te.TypeKeys("p") // paste after

		// Should have "function(arg) {function(arg)"
		AssertBufferEquals(t, te, "function(arg) {function(arg)")
	})
}

func TestYankWithTillMotion(t *testing.T) {
	t.Run("yt - yank till character", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "hello world test")

		// Start at beginning
		AssertCursorAt(t, te, 0, 0)

		// Execute "yt " (yank till space)
		te.TypeKeys("yt ")

		// Buffer should be unchanged
		AssertBufferEquals(t, te, "hello world test")
		AssertCursorAt(t, te, 0, 0)

		// Move to end and paste to verify yank worked
		te.ExecuteCommand("end-of-line")
		te.TypeKeys("p") // paste after

		// Should have "hello world testhello"
		AssertBufferEquals(t, te, "hello world testhello")
	})
}
