package proxy

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

// TestClangdDiagnosticsForBrokenCFile verifies that the proxy can start clangd
// and retrieve real diagnostics for a deliberately broken C file under artifacts.
func TestClangdDiagnosticsForBrokenCFile(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping clangd integration test in short mode")
    }

    pm, err := NewProxyManager()
    if err != nil {
        t.Fatalf("Failed to create proxy manager: %v", err)
    }
    defer pm.Close()

    cwd, _ := os.Getwd()
    testFile := filepath.Join(cwd, "../../artifacts/c-diagnostics-project/broken.c")

    if _, err := os.Stat(testFile); err != nil {
        t.Fatalf("Test file not found: %s", testFile)
    }

    // Open the file via the proxy to trigger LSP server startup and didOpen
    if _, err := pm.GetFile(testFile); err != nil {
        // If clangd is unavailable in this environment, skip gracefully
        if strings.Contains(strings.ToLower(err.Error()), "clangd") || strings.Contains(err.Error(), "not found") {
            t.Skipf("clangd not available through proxy: %v", err)
        }
        t.Fatalf("GetFile failed: %v", err)
    }

    // Nudge clangd by sending didChange with the same content to force parsing
    if content, err := os.ReadFile(testFile); err == nil {
        _ = pm.NotifyFileChange(testFile, string(content))
    }

    // Wait for clangd to initialize and publish diagnostics
    deadline := time.Now().Add(25 * time.Second)
    found := false
    var lastErr error

    for time.Now().Before(deadline) {
        time.Sleep(300 * time.Millisecond)
        diagnostics, err := pm.GetDiagnostics(testFile)
        if err != nil {
            lastErr = err
            continue
        }
        for uri, list := range diagnostics {
            if strings.Contains(uri, "broken.c") {
                t.Logf("Diagnostics for %s: %d", uri, len(list))
                if len(list) > 0 {
                    found = true
                    break
                }
            }
        }
        if found {
            break
        }
    }

    if !found {
        // Try one final time via get_file translated diagnostics
        if fc, err := pm.GetFile(testFile); err == nil && len(fc.Diagnostics) > 0 {
            t.Logf("Recovered via get_file: got %d translated diagnostics", len(fc.Diagnostics))
        } else {
            if lastErr != nil {
                t.Fatalf("Expected diagnostics for broken.c, but GetDiagnostics kept failing: %v", lastErr)
            }
            t.Fatalf("Expected diagnostics for broken.c, got none within timeout")
        }
    }

    // Optional: verify that a subsequent GetFile returns translated diagnostics too
    if fc, err := pm.GetFile(testFile); err == nil {
        if len(fc.Diagnostics) == 0 {
            t.Log("Note: no translated diagnostics via get_file yet; publish timing may vary")
        } else {
            t.Logf("get_file returned %d translated diagnostics", len(fc.Diagnostics))
        }
    }
}
