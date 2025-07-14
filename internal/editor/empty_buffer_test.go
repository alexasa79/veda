package editor

import (
	"testing"
)

func TestEmptyBufferRendering(t *testing.T) {
	t.Run("empty buffer shows line numbers", func(t *testing.T) {
		// Create editor with empty content
		te := NewTestEditorWithContent(80, 24, "")
		
		// Force a render
		te.drawEditor()
		te.drawStatusBar()
		
		// Check buffer properties
		t.Logf("Buffer LineCount: %d", te.buffer.LineCount)
		t.Logf("Buffer LineNumberWidth: %d", te.buffer.LineNumberWidth)
		t.Logf("Buffer content: %q", te.buffer.String())
		t.Logf("ViewIterator.IsEOF(): %v", te.buffer.ViewIterator.IsEOF())
		t.Logf("ViewIterator.LineNumber(): %d", te.buffer.ViewIterator.LineNumber())
		
		// Check if line 1 is rendered
		line := te.mockScreen.GetLine(0)
		if len(line) > 0 {
			endIdx := 10
			if len(line) < 10 {
				endIdx = len(line)
			}
			t.Logf("First line on screen: %q", line[:endIdx])
		}
		
		// Check specific positions
		ch1, _ := te.mockScreen.GetCell(0, 0)
		ch2, _ := te.mockScreen.GetCell(1, 0)
		t.Logf("Characters at (0,0) and (1,0): '%c' '%c'", ch1, ch2)
		
		// The first character should be '1' (line number)
		if ch1 != '1' {
			t.Errorf("Expected line number '1' at position (0,0), got '%c'", ch1)
		}
	})
}