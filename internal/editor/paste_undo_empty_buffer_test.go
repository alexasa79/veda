package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestPasteUndoEmptyBufferDoesNotPanic(t *testing.T) {
	buf := buffer.NewBuffer("", "", "", config.IndentSpaces, 4)
	cmd := &PasteCommand{
		buffer:     buf,
		position:   0,
		content:    "{}",
		isLinewise: false,
		oldCursorX: 0,
		oldCursorY: 0,
	}

	if err := cmd.Execute(buf); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("undo panicked: %v", r)
		}
	}()

	if err := cmd.Undo(buf); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
}
