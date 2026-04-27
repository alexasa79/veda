package editor

import (
	"os"
	"strings"
	"testing"
	"time"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestWriteQuitIgnoresOpenTerminalBuffer(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello")
	filePath := te.CurrentBuffer().RealPath
	removeInitialNoNameBuffer(t, te)

	te.TypeKeys("space t")
	if te.CurrentBuffer() == nil || !te.CurrentBuffer().IsInternal {
		t.Fatalf("expected terminal buffer after space t, got %#v", te.CurrentBuffer())
	}

	te.PressKey(KeyEscape, 0)
	if te.mode != commands.NormalMode {
		t.Fatalf("expected normal mode after leaving terminal, got %v", te.mode)
	}

	te.TypeKeys("space space")
	te.PressKey(events.KeyEnter, 0)
	te.WaitForAsyncOperationsTimeout(time.Second)

	if te.CurrentBuffer() == nil || te.CurrentBuffer().RealPath != filePath {
		t.Fatalf("expected to switch back to file buffer %q, got %#v", filePath, te.CurrentBuffer())
	}

	te.TypeKeys("i!")
	te.PressKey(KeyEscape, 0)
	te.TypeKeys(":wq\n")
	te.WaitForAsyncOperationsTimeout(time.Second)

	if errMsg := te.errorManager.GetCurrentError(); strings.Contains(errMsg, "no file name") {
		t.Fatalf("wq tried to save an internal buffer: %q", errMsg)
	}
	if !te.shouldExit {
		t.Fatalf("expected :wq to quit when only an internal terminal buffer remains")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(content) != "!hello" {
		t.Fatalf("saved content = %q, want %q", string(content), "!hello")
	}
}

func removeInitialNoNameBuffer(t *testing.T, te *TestEditor) {
	t.Helper()
	buf := te.buffers["[No Name]"]
	if buf == nil {
		return
	}
	for _, win := range te.windows {
		if win != nil {
			win.removeBufferFromWindow(buf)
		}
	}
	delete(te.buffers, "[No Name]")
	te.bufferOrder = removeFromSlice(te.bufferOrder, "[No Name]")
}
