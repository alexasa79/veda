package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

// Ensure calculator includes the last character in Normal mode (cursor ON char)
// and also works in Insert mode (cursor AFTER char).
func TestCalculatorExpressionEndPositionByMode(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "1+2")
	te.CurrentBuffer().FilePath = "calc.txt"

	// Place cursor ON the last character ('2') to simulate Normal mode semantics
	te.CurrentBuffer().CursorIterator = te.CurrentBuffer().CursorIterator // same line
	te.CurrentBuffer().CursorX = 2                                        // index of '2'

	if err := te.ExecuteCommand("calculate-append"); err != nil {
		t.Fatalf("calculate-append failed in normal mode case: %v", err)
	}
	got := te.CurrentBuffer().Rope.String()
	if got != "1+2 → 3" {
		t.Fatalf("normal mode end position: expected '1+2 → 3', got %q", got)
	}

	// Reset buffer to original and test Insert mode semantics (cursor AFTER '2')
	te = NewTestEditorWithContent(80, 24, "3*4")
	te.CurrentBuffer().FilePath = "calc2.txt"

	// Enter insert mode then move cursor to end (after '4')
	te.ExecuteCommand("enter-insert-mode")
	te.CurrentBuffer().CursorIterator = te.CurrentBuffer().CursorIterator
	te.CurrentBuffer().CursorX = 3 // after last char

	if err := te.ExecuteCommand("calculate-append"); err != nil {
		t.Fatalf("calculate-append failed in insert mode case: %v", err)
	}
	got2 := te.CurrentBuffer().Rope.String()
	if got2 != "3*4 → 12" {
		t.Fatalf("insert mode end position: expected '3*4 → 12', got %q", got2)
	}
}

func TestCalculatorKeyBindings(t *testing.T) {
	tests := []struct {
		name    string
		mode    commands.Mode
		cursorX int
		key     events.Key
		want    string
	}{
		{name: "append in normal mode", mode: commands.NormalMode, cursorX: 2, key: events.KeyCtrlE, want: "3*4 → 12"},
		{name: "replace in normal mode", mode: commands.NormalMode, cursorX: 2, key: events.KeyCtrlQ, want: "12"},
		{name: "append in insert mode", mode: commands.InsertMode, cursorX: 3, key: events.KeyCtrlE, want: "3*4 → 12"},
		{name: "replace in insert mode", mode: commands.InsertMode, cursorX: 3, key: events.KeyCtrlQ, want: "12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, "3*4")
			if tt.mode == commands.InsertMode {
				te.ExecuteCommand("enter-insert-mode")
			}
			te.CurrentBuffer().CursorX = tt.cursorX

			te.handleEvent(&events.KeyEvent{
				Key: tt.key,
			})

			AssertBufferEquals(t, te, tt.want)
			if te.mode != tt.mode {
				t.Fatalf("expected to remain in %s mode, got %s", tt.mode, te.mode)
			}
		})
	}
}
