package editor

import (
	"strings"
	"testing"

	"veda/internal/commands"
)

func TestInternalDiffBuffersWrap(t *testing.T) {
	const paneWidth = 30
	te := NewTestEditorWithContent(paneWidth*2, 12, "")
	leftLine := strings.Repeat("a", 45)
	rightLine := strings.Repeat("b", 65)
	_, left, right := setupWrapTestDiffView(te, leftLine+"\nnext", rightLine+"\nnext")
	left.IsInternal, right.IsInternal = true, true
	left.IsReadOnly, right.IsReadOnly = true, true

	te.render()

	for pane, want := range []string{leftLine, rightLine} {
		var got strings.Builder
		buf := left
		if pane == 1 {
			buf = right
		}
		// Both lines need three aligned rows; read the actual rendered text.
		for y := 0; y < 3; y++ {
			for x := pane*paneWidth + buf.LineNumberWidth; x < (pane+1)*paneWidth; x++ {
				ch, _ := te.mockScreen.GetCell(x, y)
				if ch != ' ' && ch != 0 {
					got.WriteRune(ch)
				}
			}
		}
		if got.String() != want {
			t.Errorf("pane %d rendered %q, want %q", pane, got.String(), want)
		}
	}
}

func TestTerminalBufferSkipsEditorWrappingInAllModes(t *testing.T) {
	for _, mode := range []commands.Mode{commands.NormalMode, commands.TerminalMode} {
		te := NewTestEditorWithContent(30, 12, "")
		line := strings.Repeat("x", 65)
		setupTerminalBufferForTest(te, 80, 12, line, NewTerminalEmulator(80, 12))
		te.mode = mode
		wrapped := te.CalculateWrappedLines()
		if len(wrapped) != 1 || wrapped[0].Text != line {
			t.Errorf("mode %v: terminal text was rewrapped: %+v", mode, wrapped)
		}
	}
}
