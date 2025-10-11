package editor

import (
	"testing"
	"veda/internal/buffer"
)

func TestSearchRepeatStaysInBufferWithoutMatches(t *testing.T) {
	te := NewTestEditor(80, 24)

	buf1 := buffer.NewBuffer("match here\nand match again\n", "file1.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf1.ViewIterator = buffer.NewBufferIterator(buf1)
	buf1.CursorIterator = buffer.NewBufferIterator(buf1)
	buf1.OnContentChange = func() {
		te.updateSearchMatchesOnContentChange()
	}

	buf2 := buffer.NewBuffer("second buffer content\n", "file2.txt", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	buf2.ViewIterator = buffer.NewBufferIterator(buf2)
	buf2.CursorIterator = buffer.NewBufferIterator(buf2)
	buf2.OnContentChange = func() {
		te.updateSearchMatchesOnContentChange()
	}

	te.buffers[buf1.FilePath] = buf1
	te.buffers[buf2.FilePath] = buf2
	te.bufferOrder = append(te.bufferOrder, buf1.FilePath, buf2.FilePath)

	te.windows = []*Window{newWindow(buf1)}
	te.activeWindow = 0
	te.SetCurrentBuffer(buf1)

	te.windows[0].addBufferToWindow(buf2)
	te.SetCurrentBuffer(buf1)

	te.enterSearchMode()
	te.searchLine.SetText("/match")
	te.executeSearch()

	if te.CurrentBuffer() != buf1 {
		t.Fatalf("expected primary buffer to remain active during initial search")
	}
	if len(te.searchMatches) == 0 {
		t.Fatalf("expected search to find matches in first buffer")
	}

	te.SetCurrentBuffer(buf2)
	if te.CurrentBuffer() != buf2 {
		t.Fatalf("expected to switch to second buffer")
	}

	initialAbsolute := te.getAbsolutePosition()
	initialLine := te.CurrentBuffer().CursorIterator.LineNumber()
	initialX := te.CurrentBuffer().CursorX

	if err := te.findNextCmd(nil); err != nil {
		t.Fatalf("findNextCmd returned unexpected error: %v", err)
	}

	if te.getAbsolutePosition() != initialAbsolute || te.CurrentBuffer().CursorIterator.LineNumber() != initialLine || te.CurrentBuffer().CursorX != initialX {
		t.Fatalf("expected repeating search in buffer without matches to keep cursor in place; got line %d, column %d", te.CurrentBuffer().CursorIterator.LineNumber(), te.CurrentBuffer().CursorX)
	}
}
