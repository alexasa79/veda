package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestStatusBar_ShowsDiffMarkerForDiffBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2")

	diffBuf := buffer.NewBuffer("line1\nline2", "dummy.go@HEAD", "dummy.go@HEAD", config.IndentTabs, 4)
	te.buffers[diffBuf.FilePath] = diffBuf
	te.windows[te.activeWindow].addBufferToWindow(diffBuf)
	te.switchToBuffer(diffBuf)

	te.render()
	lines := te.GetScreenContent()
	if len(lines) == 0 {
		t.Fatalf("no screen content rendered")
	}

	status := lines[len(lines)-1]
	if !strings.Contains(status, "[DIFF]") {
		t.Fatalf("expected status bar to contain [DIFF], got %q", status)
	}
}

func TestStatusBar_HidesDiffMarkerForNormalBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2")

	te.render()
	lines := te.GetScreenContent()
	if len(lines) == 0 {
		t.Fatalf("no screen content rendered")
	}

	status := lines[len(lines)-1]
	if strings.Contains(status, "[DIFF]") {
		t.Fatalf("did not expect [DIFF] in status bar for normal buffer, got %q", status)
	}
}
