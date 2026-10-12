package editor

import (
	"testing"
	"veda/internal/events"
)

func TestMultiCursorDeleteJoinsLinesAndCollapsesCursors(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "100\n200\n300\n400\n500\n600")

	te.TypeKeys("m")
	for i := 0; i < 5; i++ {
		te.PressKey(events.KeyEnter, 0)
	}
	te.PressKey(events.KeyEscape, 0)

	te.TypeKeys("A+")
	te.PressKey(events.KeyDelete, 0)

	const want = "100+200+300+400+500+600+"
	AssertBufferEquals(t, te, want)
	if got := len(te.GetVirtualCursors()); got != 0 {
		t.Errorf("expected joined cursors to collapse into the primary cursor, got %d virtual cursors", got)
	}
	if x, y := te.GetCursorPosition(); x != len(want) || y != 0 {
		t.Errorf("expected primary cursor at (%d, 0), got (%d, %d)", len(want), x, y)
	}
}
