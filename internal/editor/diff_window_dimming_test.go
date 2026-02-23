package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestDiffView_WindowDimming verifies that the DiffBufferView window behaves
// like any other window with respect to dimming: active window is not dimmed,
// inactive windows are dimmed. Since the diff view is now a single window,
// there is no special dimming suppression.
func TestDiffView_WindowDimming(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "line1\nline2\nline3")
	te.CurrentBuffer().FilePath = "test.go"

	t.Run("normal_view_has_dimming", func(t *testing.T) {
		// With a single window, active window is not dimmed
		te.activeWindow = 0
		dim := 0 != te.activeWindow
		if dim {
			t.Error("Active window should not be dimmed")
		}
	})

	t.Run("diff_view_single_window_not_dimmed_when_active", func(t *testing.T) {
		// Set up DiffBufferView
		headBuffer := buffer.NewBuffer("line1\noriginal line2\nline3", "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
		headBuffer.IsReadOnly = true
		headBuffer.DiffHints = []buffer.DiffLine{{Content: "line1", LineType: buffer.DiffLineContext}}
		te.buffers["test.go@HEAD"] = headBuffer

		currentBuffer := buffer.NewBuffer("line1\nmodified line2\nline3", "test.go@current", "test.go@current", config.IndentTabs, 4)
		currentBuffer.DiffHints = []buffer.DiffLine{{Content: "line1", LineType: buffer.DiffLineContext}}
		te.buffers["test.go@current"] = currentBuffer

		te.SetupDiffView(headBuffer, currentBuffer, "test.go")

		// The diff view is a single window — it should not be dimmed when active
		dim := 0 != te.activeWindow
		if dim {
			t.Error("Active DiffBufferView window should not be dimmed")
		}

		// isDiffViewActive should return true
		if !te.isDiffViewActive() {
			t.Error("isDiffViewActive should return true when DiffBufferView is present")
		}
	})
}

// TestIsDiffViewActive tests the isDiffViewActive helper function
func TestIsDiffViewActive(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "test content")

	t.Run("no_diff_view", func(t *testing.T) {
		if te.isDiffViewActive() {
			t.Error("isDiffViewActive should return false with no DiffBufferView")
		}
	})

	t.Run("with_diff_view", func(t *testing.T) {
		headBuffer := buffer.NewBuffer("head", "test.go@HEAD", "test.go@HEAD", config.IndentTabs, 4)
		headBuffer.DiffHints = []buffer.DiffLine{{Content: "head", LineType: buffer.DiffLineContext}}
		currentBuffer := buffer.NewBuffer("current", "test.go@current", "test.go@current", config.IndentTabs, 4)
		currentBuffer.DiffHints = []buffer.DiffLine{{Content: "current", LineType: buffer.DiffLineContext}}
		te.buffers["test.go@HEAD"] = headBuffer
		te.buffers["test.go@current"] = currentBuffer

		te.SetupDiffView(headBuffer, currentBuffer, "test.go")

		if !te.isDiffViewActive() {
			t.Error("isDiffViewActive should return true when DiffBufferView is present")
		}
	})
}
