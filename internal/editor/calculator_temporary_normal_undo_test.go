package editor

import (
	"testing"
	"veda/internal/events"
)

func TestCalculateAppendFromTemporaryNormalModeUndo(t *testing.T) {
	const expression = "100+200+300+400+500+600"
	te := NewTestEditorWithContent(80, 24, expression+"+")

	te.TypeKeys("$xA")
	te.TypeKeys("ctrl+o - -")
	AssertBufferEquals(t, te, expression+" → 2100")

	te.PressKey(events.KeyEscape, 0)
	te.TypeKeys("u")
	AssertBufferEquals(t, te, expression)
}

func TestCalculateAppendUndoRestoresCursor(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "100+200")
	te.SetCursorPosition(6, 0)

	te.TypeKeys("- -")
	AssertBufferEquals(t, te, "100+200 → 300")
	te.TypeKeys("u")

	AssertBufferEquals(t, te, "100+200")
	AssertCursorAt(t, te, 6, 0)
}
