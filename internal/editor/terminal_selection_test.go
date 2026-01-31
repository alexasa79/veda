package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
)

func TestTerminalSelectionHighlightInNormalMode(t *testing.T) {
	te := NewTestEditor(40, 6)

	term := NewTerminalEmulator(40, 6)
	term.Write([]byte("hello world\n"))

	setupTerminalBufferForTest(te, 40, 6, term.Render(), term)

	if te.mode != commands.NormalMode {
		t.Fatalf("expected NORMAL mode at start, got %v", te.mode)
	}

	selectionStyle := te.config.Theme.GetStyle(te.config.Theme.Selection)
	_, selBg, _ := selectionStyle.Decompose()

	if err := te.ExecuteCommand("enter-visual-mode"); err != nil {
		t.Fatalf("enter-visual-mode failed: %v", err)
	}
	if te.mode != commands.VisualMode {
		t.Fatalf("expected VISUAL mode after enter-visual-mode, got %v", te.mode)
	}

	for i := 0; i < 4; i++ {
		if err := te.ExecuteCommand("right"); err != nil {
			t.Fatalf("move right failed: %v", err)
		}
	}

	te.drawEditor()

	screenLine := te.mockScreen.GetLine(0)
	helloStart := strings.Index(screenLine, "hello")
	if helloStart == -1 {
		t.Fatalf("expected to find \"hello\" in rendered line, got %q", screenLine)
	}

	for i := 0; i < 5; i++ {
		_, bg, _ := te.mockScreen.GetCellStyle(helloStart+i, 0)
		if bg != selBg {
			t.Errorf("selected character %d expected selection bg %v, got %v", i, selBg, bg)
		}
	}

	if helloStart+5 < len(screenLine) {
		_, bg, _ := te.mockScreen.GetCellStyle(helloStart+5, 0)
		if bg == selBg {
			t.Errorf("character after selection should not have selection background")
		}
	}
}
