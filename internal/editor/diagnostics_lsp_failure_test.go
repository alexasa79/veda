package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"veda/internal/events"
	"veda/internal/proxy"
)

// TestLSPServerNotFound tests behavior when the LSP server executable doesn't exist
func TestLSPServerNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LSP failure test in short mode")
	}

	// Start proxy with nonexistent LSP server
	pm, err := proxy.NewProxyManagerWithArgs([]string{
		"--test-lsp-python=/nonexistent/path/to/lsp/server",
	})
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_python.py")

	pythonContent := `def add(a, b):
    return a + b

result = add(1, 2)
print(result)
`

	if err := os.WriteFile(filePath, []byte(pythonContent), 0644); err != nil {
		t.Fatalf("Failed to write Python file: %v", err)
	}

	// Open the file - LSP error should be returned (GetFile will wait for LSP to start/fail)
	fc, err := pm.GetFile(filePath)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	// Create an editor and verify LSP failure is handled
	te := NewTestEditorWithContent(80, 24, fc.Content)
	te.Editor.proxyManager = pm
	buf := te.CurrentBuffer()
	buf.FilePath = filePath
	buf.RealPath = filePath
	buf.HasLSP = true
	buf.Diagnostics = fc.Diagnostics

	// Simulate LSP failure by setting the flag (as would happen in real openFile)
	if fc.LSPError != "" {
		buf.LSPFailed = true
		te.Editor.ShowError("LSP server failed to start: %s", fc.LSPError)
	}

	// Register buffer
	te.Editor.buffers[filePath] = buf

	// Set up diagnostic callback
	pm.SetDiagnosticCallback(func(path string, diagnostics []proxy.Diagnostic) {
		buf, exists := te.Editor.buffers[path]
		if exists {
			buf.Diagnostics = diagnostics
			te.Editor.asyncEvents <- events.NewCustomEvent("render", nil)
		}
	})

	// Render to show the status bar
	te.Editor.render()

	// Verify that the status bar contains "[No LSP]"
	statusBarY := te.Editor.getStatusBarY()
	screenLines := te.GetScreenContent()
	if statusBarY >= 0 && statusBarY < len(screenLines) {
		statusBarLine := screenLines[statusBarY]
		if !strings.Contains(statusBarLine, "[No LSP]") {
			t.Errorf("Status bar does not contain '[No LSP]' indicator. Status bar: %q", statusBarLine)
		}
	} else {
		t.Errorf("Invalid status bar Y position: %d", statusBarY)
	}

	// Verify LSP error is set
	if fc.LSPError == "" {
		t.Errorf("Expected LSP error to be set, but it was empty")
	} else {
		t.Logf("LSP error correctly set: %s", fc.LSPError)
	}

	// The editor should continue to work normally even without LSP
	te.TypeKeys("i# Test comment")
	te.PressKey(events.KeyEscape, 0)

	// Verify the edit worked
	if !strings.Contains(te.GetBufferContent(), "# Test comment") {
		t.Errorf("Edit failed: content does not contain expected text")
	}

	t.Logf("Editor works correctly when LSP server is not found")
}

// TestLSPServerExitsImmediately tests behavior when LSP server exits immediately with error
func TestLSPServerExitsImmediately(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LSP failure test in short mode")
	}

	cwd, _ := os.Getwd()
	lspFixturePath := filepath.Join(cwd, "../../artifacts/lsp-test-fixtures/lsp-server-exits-immediately.sh")

	// Verify the test fixture exists
	if _, err := os.Stat(lspFixturePath); os.IsNotExist(err) {
		t.Fatalf("Test fixture not found: %s", lspFixturePath)
	}

	// Start proxy with LSP server that exits immediately
	pm, err := proxy.NewProxyManagerWithArgs([]string{
		"--test-lsp-python=" + lspFixturePath,
	})
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_python.py")

	pythonContent := `def add(a, b):
    return a + b

result = add(1, undefined_variable)
print(result)
`

	if err := os.WriteFile(filePath, []byte(pythonContent), 0644); err != nil {
		t.Fatalf("Failed to write Python file: %v", err)
	}

	// Try to open the file - LSP error should be returned (GetFile will wait for LSP to start/fail)
	fc, err := pm.GetFile(filePath)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	// Create an editor and verify LSP failure is handled
	te := NewTestEditorWithContent(80, 24, fc.Content)
	te.Editor.proxyManager = pm
	buf := te.CurrentBuffer()
	buf.FilePath = filePath
	buf.RealPath = filePath
	buf.HasLSP = true
	buf.Diagnostics = fc.Diagnostics

	// Simulate LSP failure by setting the flag (as would happen in real openFile)
	if fc.LSPError != "" {
		buf.LSPFailed = true
		te.Editor.ShowError("LSP server failed to start: %s", fc.LSPError)
	}

	te.Editor.buffers[filePath] = buf

	pm.SetDiagnosticCallback(func(path string, diagnostics []proxy.Diagnostic) {
		buf, exists := te.Editor.buffers[path]
		if exists {
			buf.Diagnostics = diagnostics
			te.Editor.asyncEvents <- events.NewCustomEvent("render", nil)
		}
	})

	// Render to show the status bar
	te.Editor.render()

	// Verify that the status bar contains "[No LSP]"
	statusBarY := te.Editor.getStatusBarY()
	screenLines := te.GetScreenContent()
	if statusBarY >= 0 && statusBarY < len(screenLines) {
		statusBarLine := screenLines[statusBarY]
		if !strings.Contains(statusBarLine, "[No LSP]") {
			t.Errorf("Status bar does not contain '[No LSP]' indicator. Status bar: %q", statusBarLine)
		}
	} else {
		t.Errorf("Invalid status bar Y position: %d", statusBarY)
	}

	// Verify LSP error is set
	if fc.LSPError == "" {
		t.Errorf("Expected LSP error to be set, but it was empty")
	} else {
		t.Logf("LSP error correctly set: %s", fc.LSPError)
	}

	// Verify the editor works normally even when LSP server crashes
	te.TypeKeys(":5\n")
	te.TypeKeys("ccresult = add(1, 2)")
	te.PressKey(events.KeyEscape, 0)

	content := te.GetBufferContent()
	if !strings.Contains(content, "result = add(1, 2)") {
		t.Errorf("Edit failed: expected 'result = add(1, 2)', got:\n%s", content)
	}

	t.Logf("Editor works correctly when LSP server exits immediately")
}
