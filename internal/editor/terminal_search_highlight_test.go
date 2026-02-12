package editor

import (
	"strings"
	"testing"
)

func TestTerminalSearchHighlighting(t *testing.T) {
	te := NewTestEditor(40, 6)

	term := NewTerminalEmulator(40, 6)
	term.Write([]byte("hello world\n"))

	setupTerminalBufferForTest(te, 40, 6, term.Render(), term)

	te.searchPattern = "world"

	searchStyle := te.config.Theme.GetStyle(te.config.Theme.SearchHighlight)
	_, searchBg, _ := searchStyle.Decompose()

	te.Editor.render()

	screenLine := te.mockScreen.GetLine(0)
	worldStart := strings.Index(screenLine, "world")
	if worldStart == -1 {
		t.Fatalf("expected to find \"world\" in rendered line, got %q", screenLine)
	}

	for i := 0; i < 5; i++ {
		_, bg, _ := te.mockScreen.GetCellStyle(worldStart+i, 0)
		if bg != searchBg {
			t.Errorf("search highlight char %d expected bg %v, got %v", i, searchBg, bg)
		}
	}
}
