package editor

import "testing"

func TestZenModeWidthUsesVerticalRulers(t *testing.T) {
	te := NewTestEditor(200, 60)
	te.config.VerticalRulers = []int{80, 100, 120}
	te.zenMode = true
	te.recomputeWindowsLayout()

	active := te.windows[te.activeWindow]
	if active == nil || active.Buffer == nil {
		t.Fatalf("expected active window with buffer")
	}
	active.Buffer.UpdateLineNumberWidth()
	gutterWidth := active.Buffer.LineNumberWidth
	maxRuler := 0
	for _, r := range te.config.VerticalRulers {
		if r > maxRuler {
			maxRuler = r
		}
	}
	expectedWidth := gutterWidth + maxRuler + zenModeContentMargin
	if active.Width != expectedWidth {
		t.Fatalf("expected zen mode width %d, got %d", expectedWidth, active.Width)
	}
	expectedX := (200 - expectedWidth) / 2
	if expectedX < zenModePaddingX {
		expectedX = zenModePaddingX
	}
	if active.X != expectedX {
		t.Fatalf("expected zen mode window centered at x=%d, got %d", expectedX, active.X)
	}
}

func TestZenModeWidthFallbackWithoutRulers(t *testing.T) {
	te := NewTestEditor(120, 40)
	te.config.VerticalRulers = nil
	te.zenMode = true
	te.recomputeWindowsLayout()

	pad := zenModePaddingX * 2
	expectedWidth := 120 - pad
	if expectedWidth < 1 {
		expectedWidth = 1
	}
	active := te.windows[te.activeWindow]
	if active.Width != expectedWidth {
		t.Fatalf("expected zen mode width %d, got %d", expectedWidth, active.Width)
	}
}
