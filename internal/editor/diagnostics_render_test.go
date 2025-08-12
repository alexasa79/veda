package editor

import (
    "strings"
    "testing"
    "veda/internal/proxy"
)

// This test validates diagnostic gutter indicators and the persistent banner.
// It injects diagnostics directly into the buffer to avoid LSP dependency.
func TestDiagnosticsGutterAndBannerRendering(t *testing.T) {
    content := "l1\nl2\nl3\nl4\n"
    te := NewTestEditorWithContent(40, 10, content)

    // Inject diagnostics (1-based lines)
    te.buffer.Diagnostics = []proxy.Diagnostic{
        {Line: 2, Severity: "error", Message: "Syntax error"},
        {Line: 4, Severity: "warning", Message: "Unused variable"},
    }

    // Initial render
    te.drawEditor()
    te.drawErrorLine()
    te.drawStatusBar()

    // Indicator is drawn in the middle gutter column: digits + 1
    digitsWidth := te.buffer.LineNumberWidth - 3
    if digitsWidth < 1 { digitsWidth = 1 }
    diagX := digitsWidth + 1

    // Line 2 (row 1) should have ●
    ch, _ := te.mockScreen.GetCell(diagX, 1)
    if ch != '●' {
        t.Fatalf("expected '●' at (x=%d,y=%d), got '%c'", diagX, 1, ch)
    }

    // Line 1 (row 0) should not have ●
    ch, _ = te.mockScreen.GetCell(diagX, 0)
    if ch == '●' {
        t.Fatalf("did not expect diagnostics indicator at (x=%d,y=%d)", diagX, 0)
    }

    // Move cursor to line 2 and expect error banner text
    te.SetCursorPosition(0, 1)
    te.drawEditor()
    te.drawErrorLine()
    te.drawStatusBar()
    bannerY := te.getStatusBarY() - 1
    line := te.mockScreen.GetLine(bannerY)
    if !strings.Contains(line, "Syntax error") {
        t.Fatalf("expected banner to contain 'Syntax error', got: %q", line)
    }

    // Move cursor to line 4 and expect warning banner text
    te.SetCursorPosition(0, 3)
    te.drawEditor()
    te.drawErrorLine()
    te.drawStatusBar()
    line = te.mockScreen.GetLine(bannerY)
    if !strings.Contains(line, "Unused variable") {
        t.Fatalf("expected banner to contain 'Unused variable', got: %q", line)
    }

    // Temporary error overrides diagnostic banner
    te.ShowError("OVERRIDE")
    te.drawEditor()
    te.drawErrorLine()
    te.drawStatusBar()
    line = te.mockScreen.GetLine(bannerY)
    if !strings.Contains(line, "OVERRIDE") {
        t.Fatalf("expected temporary error override, got: %q", line)
    }
}
