package widgets

import (
	"testing"
	"veda/internal/events"
)

func TestLineEditorAltBackspaceDeletesWord(t *testing.T) {
	le := NewLineEditor(0, 0, 40)
	le.SetText("foo bar")
	le.SetCursor(len("foo bar"))

	if handled := le.HandleKey(events.NewKeyEvent(events.KeyBackspace, 0, events.ModAlt)); !handled {
		t.Fatalf("expected alt+backspace to be handled")
	}

	if got := le.GetText(); got != "foo " {
		t.Fatalf("expected text %q, got %q", "foo ", got)
	}

	if le.GetCursor() != len("foo ") {
		t.Fatalf("expected cursor at %d, got %d", len("foo "), le.GetCursor())
	}
}
