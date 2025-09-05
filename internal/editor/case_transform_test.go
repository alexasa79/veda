package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestCaseTransform_NormalMode_Word(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "abc def\ngHi")
	// Cursor at 'b' in first word
	te.SetCursorPosition(1, 0)
	// Press Alt+U to uppercase
	if seq, err := commands.ParseKeySequence("alt+u"); err == nil {
		for _, k := range seq {
			te.handleEvent(&events.KeyEvent{Key: k.Key, Rune: k.Rune, Modifiers: k.Mod})
		}
	} else {
		t.Fatalf("failed to parse key sequence: %v", err)
	}
	AssertBufferEquals(t, te, "ABC def\ngHi")

	// Lowercase same word via command line
	te.SetCursorPosition(1, 0) // on 'B' in ABC
	if seq, err := commands.ParseKeySequence("alt+l"); err == nil {
		for _, k := range seq {
			te.handleEvent(&events.KeyEvent{Key: k.Key, Rune: k.Rune, Modifiers: k.Mod})
		}
	} else {
		t.Fatalf("failed to parse key sequence: %v", err)
	}
	AssertBufferEquals(t, te, "abc def\ngHi")
}

func TestCaseTransform_NormalMode_MultiCursor(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "foo\nbar\nbaz")
	// Place primary at line 0 col 0
	te.SetCursorPosition(0, 0)
	// Enter cursor mode and add a virtual cursor on line 1
	te.TypeKeys("m")
	te.PressKey(events.KeyEnter, 0) // add virtual at (0,0)
	// Auto-advance to next line, add cursor again
	te.PressKey(events.KeyEnter, 0) // now we have cursors on line 0 and 1
	te.TypeKeys("m")                // exit cursor mode

	// Use Alt+U to run to-upper
	if seq, err := commands.ParseKeySequence("alt+u"); err == nil {
		for _, k := range seq {
			te.handleEvent(&events.KeyEvent{Key: k.Key, Rune: k.Rune, Modifiers: k.Mod})
		}
	} else {
		t.Fatalf("failed to parse key sequence: %v", err)
	}
	AssertBufferEquals(t, te, "FOO\nBAR\nBAZ")
}

func TestCaseTransform_NormalMode_MultiCursor_ToLower(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "FOO\nBAR\nBAZ")
	// Place primary at line 0 col 0
	te.SetCursorPosition(0, 0)
	// Enter cursor mode and add a virtual cursor on line 1 and 2
	te.TypeKeys("m")
	te.PressKey(events.KeyEnter, 0) // add virtual at (0,0), auto-move to line 1
	te.PressKey(events.KeyEnter, 0) // add virtual at (0,1), auto-move to line 2
	te.TypeKeys("m")                // exit cursor mode

	if seq, err := commands.ParseKeySequence("alt+l"); err == nil {
		for _, k := range seq {
			te.handleEvent(&events.KeyEvent{Key: k.Key, Rune: k.Rune, Modifiers: k.Mod})
		}
	} else {
		t.Fatalf("failed to parse key sequence: %v", err)
	}
	AssertBufferEquals(t, te, "foo\nbar\nbaz")
}

func TestCaseTransform_VisualMode_Selection(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "const value;\nnext")
	// Enter visual mode and select "const"
	if err := te.ExecuteCommand("enter-visual-mode"); err != nil {
		t.Fatalf("enter visual failed: %v", err)
	}
	// Ensure cursor starts at (0,0)
	te.SetCursorPosition(0, 0)
	// Extend selection to cover first word (5 chars)
	for i := 0; i < 4; i++ { // move right 4 times (cursor initially at 0, selection is inclusive)
		te.TypeKeys("right")
	}
	// Sanity-check the selected text prior to transformation
	if start, end, has := te.getSelectionRange(); has {
		sLine, sChar := te.byteOffsetToLineChar(start)
		eLine, eChar := te.byteOffsetToLineChar(end)
		selected := te.getTextInRange(sLine, sChar, eLine, eChar)
		ropeSel := te.CurrentBuffer().Rope.Extract(start, end)
		if selected != "const" || ropeSel != "const" {
			t.Fatalf("precondition failed: expected selected text 'const'; getTextInRange=%q, rope.Extract=%q (range %d..%d => (%d,%d)-(%d,%d))", selected, ropeSel, start, end, sLine, sChar, eLine, eChar)
		}
	} else {
		t.Fatalf("precondition failed: expected to have a selection")
	}
	if seq, err := commands.ParseKeySequence("alt+u"); err == nil {
		for _, k := range seq {
			te.handleEvent(&events.KeyEvent{Key: k.Key, Rune: k.Rune, Modifiers: k.Mod})
		}
	} else {
		t.Fatalf("failed to parse key sequence: %v", err)
	}
	AssertBufferEquals(t, te, "CONST value;\nnext")

	// Select "value" and to-lower
	// Move to space then next word start
	te.PressKey(events.KeyEscape, 0)
	te.SetCursorPosition(6, 0) // 'v' in value (index 6)
	if err := te.ExecuteCommand("enter-visual-mode"); err != nil {
		t.Fatalf("enter visual failed: %v", err)
	}
	for i := 0; i < 4; i++ {
		te.TypeKeys("right")
	}
	if seq, err := commands.ParseKeySequence("alt+l"); err == nil {
		for _, k := range seq {
			te.handleEvent(&events.KeyEvent{Key: k.Key, Rune: k.Rune, Modifiers: k.Mod})
		}
	} else {
		t.Fatalf("failed to parse key sequence: %v", err)
	}
	AssertBufferEquals(t, te, "CONST value;\nnext")
}

// Repeat with '.' should replay last case transform at cursor
func TestCaseTransform_RepeatDot(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "foo bar baz")
    // Uppercase the first word using Alt+U
    te.SetCursorPosition(1, 0) // on 'o' in foo
    if seq, err := commands.ParseKeySequence("alt+u"); err == nil {
        for _, k := range seq {
            te.handleEvent(&events.KeyEvent{Key: k.Key, Rune: k.Rune, Modifiers: k.Mod})
        }
    } else {
        t.Fatalf("failed to parse key sequence: %v", err)
    }
	AssertBufferEquals(t, te, "FOO bar baz")

	// Move to start of next word and press '.'
	te.SetCursorPosition(4, 0) // at 'b' in bar
	te.TypeKeys(".")
	AssertBufferEquals(t, te, "FOO BAR baz")

	// Repeat again on next word
	te.SetCursorPosition(8, 0) // at 'b' in baz
	te.TypeKeys(".")
	AssertBufferEquals(t, te, "FOO BAR BAZ")
}
