package editor

import "testing"

func TestVisualTillCharacterExtendsSelection(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "abc(def) ghi")

	te.TypeKeys("v")
	te.TypeKeys("t)")

	if te.CurrentBuffer().CursorX != 6 {
		t.Fatalf("cursorX = %d, want 6", te.CurrentBuffer().CursorX)
	}

	start, end, ok := te.getSelectionRange()
	if !ok {
		t.Fatalf("expected visual selection")
	}
	selected := te.CurrentBuffer().Rope.String()[start:end]
	if selected != "abc(def" {
		t.Fatalf("selection = %q, want %q", selected, "abc(def")
	}
}
