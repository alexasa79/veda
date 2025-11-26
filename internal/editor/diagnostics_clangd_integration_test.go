package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"veda/internal/buffer"
	"veda/internal/events"
	"veda/internal/proxy"
	"veda/internal/ui"
)

// This test drives the editor with real key presses, saves with :w, and waits
// for clangd diagnostics to appear and then disappear after the code is fixed.
func TestClangdDiagnosticsClearAfterFix(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping clangd integration test in short mode")
	}
	if _, err := exec.LookPath("clangd"); err != nil {
		t.Skip("clangd not found in PATH; skipping test")
	}

	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	cwd, _ := os.Getwd()
	filePath := filepath.Join(cwd, "../../artifacts/c-diagnostics-project/diag_toggle.c")

	goodContent := `#include <stdio.h>

int add(int a, int b) {
	return a + b;
}

int main(void) {
	int value = add(1, 2);
	printf("%d\n", value);
	return 0;
}
`

	if err := os.WriteFile(filePath, []byte(goodContent), 0644); err != nil {
		t.Fatalf("Failed to write good content: %v", err)
	}

	fc, err := pm.GetFile(filePath)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "clangd") || strings.Contains(err.Error(), "not found") {
			t.Skipf("clangd not available through proxy: %v", err)
		}
		t.Fatalf("GetFile failed: %v", err)
	}

	te := NewTestEditorWithContent(80, 24, fc.Content)
	te.Editor.proxyManager = pm
	buf := te.CurrentBuffer()
	buf.FilePath = filePath
	buf.RealPath = filePath
	buf.HasLSP = true
	buf.Diagnostics = fc.Diagnostics

	// Register buffer in the buffers map so diagnostic notifications can find it
	te.Editor.buffers[filePath] = buf

	// Set up diagnostic callback to update buffer when diagnostics arrive
	pm.SetDiagnosticCallback(func(path string, diagnostics []proxy.Diagnostic) {
		buf, exists := te.Editor.buffers[path]
		if exists {
			buf.Diagnostics = diagnostics
			te.Editor.asyncEvents <- events.NewCustomEvent("render", nil)
		}
	})

	diagLine := 8 // 1-based line with the call to add(...)
	diagX := diagnosticsIndicatorX(buf)
	diagY := diagLine - 1

	expectIndicator(t, te, diagX, diagY, false)

	// Break the file using normal-mode keys, then save with :w
	te.TypeKeys(":8\n")
	te.TypeKeys("ccint value = add(1, missing_symbol);")
	te.PressKey(events.KeyEscape, 0)
	te.TypeKeys(":w\n")
	te.pendingSaves.Wait()

	if err := waitForIndicator(te, diagX, diagY, true, 25*time.Second); err != nil {
		t.Fatalf("Did not observe diagnostic after breaking file: %v", err)
	}

	// Fix the file and save again with :w
	te.TypeKeys(":8\n")
	te.TypeKeys("ccint value = add(1, 2);")
	te.PressKey(events.KeyEscape, 0)
	te.TypeKeys(":w\n")
	te.pendingSaves.Wait()

	if err := waitForIndicator(te, diagX, diagY, false, 25*time.Second); err != nil {
		t.Fatalf("Diagnostic did not clear after fix: %v", err)
	}
}

func diagnosticsIndicatorX(buf *buffer.Buffer) int {
	digitsWidth := buf.LineNumberWidth - 3
	if digitsWidth < 1 {
		digitsWidth = 1
	}
	return digitsWidth + 1
}

func renderCell(te *TestEditor, x, y int) (rune, ui.Style) {
	te.drawEditor()
	te.drawErrorLine()
	te.drawStatusBar()
	return te.mockScreen.GetCell(x, y)
}

func expectIndicator(t *testing.T, te *TestEditor, x, y int, want bool) {
	t.Helper()
	r, _ := renderCell(te, x, y)
	if (r == '●') != want {
		t.Fatalf("expected indicator=%v at (%d,%d), got %q", want, x, y, r)
	}
}

func waitForIndicator(te *TestEditor, x, y int, want bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if te.drainAsyncEventsOnce() {
			// keep looping after processing events
		}
		r, _ := renderCell(te, x, y)
		if (r == '●') == want {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
		te.TypeKeys(":w\n") // pull fresh diagnostics from proxy cache
		te.pendingSaves.Wait()
	}
	state := "clear"
	if want {
		state = "appear"
	}
	return fmt.Errorf("timed out waiting for indicator to %s", state)
}
