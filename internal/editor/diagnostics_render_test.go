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
	te.CurrentBuffer().Diagnostics = []proxy.Diagnostic{
		{Line: 2, Severity: "error", Message: "Syntax error"},
		{Line: 4, Severity: "warning", Message: "Unused variable"},
	}

	// Initial render
	te.drawEditor()
	te.drawErrorLine()
	te.drawStatusBar()

	// Indicator is drawn in the middle gutter column: digits + 1
	digitsWidth := te.CurrentBuffer().LineNumberWidth - 3
	if digitsWidth < 1 {
		digitsWidth = 1
	}
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

// TestMultiLineDiagnosticRendering tests that diagnostic messages with newlines are rendered across multiple lines
func TestMultiLineDiagnosticRendering(t *testing.T) {
	content := "line1\nline2\nline3\n"
	te := NewTestEditorWithContent(80, 20, content) // Large screen to accommodate multi-line display

	// Inject a diagnostic with newlines in the message
	te.CurrentBuffer().Diagnostics = []proxy.Diagnostic{
		{Line: 2, Severity: "error", Message: "This is a multi-line\nerror message that\nspans multiple lines"},
	}

	// Position cursor on the diagnostic line
	te.SetCursorPosition(0, 1) // Line 2 (0-indexed)
	te.drawEditor()
	te.drawErrorLine()
	te.drawStatusBar()

	statusBarY := te.getStatusBarY()

	// Check first diagnostic line (3 lines above status bar)
	line1Y := statusBarY - 3
	line1 := te.mockScreen.GetLine(line1Y)
	if !strings.Contains(line1, "This is a multi-line") {
		t.Errorf("Expected first line of diagnostic, got: %q", line1)
	}

	// Check second diagnostic line (2 lines above status bar)
	line2Y := statusBarY - 2
	line2 := te.mockScreen.GetLine(line2Y)
	if !strings.Contains(line2, "error message that") {
		t.Errorf("Expected second line of diagnostic, got: %q", line2)
	}

	// Check third diagnostic line (1 line above status bar)
	line3Y := statusBarY - 1
	line3 := te.mockScreen.GetLine(line3Y)
	if !strings.Contains(line3, "spans multiple lines") {
		t.Errorf("Expected third line of diagnostic, got: %q", line3)
	}
}

// TestDiagnosticLineCountCalculation tests the getDiagnosticLineCount function
func TestDiagnosticLineCountCalculation(t *testing.T) {
	te := NewTestEditorWithContent(40, 10, "test\n")

	// Test single line message
	count := te.getDiagnosticLineCount("Single line message")
	if count != 1 {
		t.Errorf("Expected 1 line for single line message, got %d", count)
	}

	// Test multi-line message
	count = te.getDiagnosticLineCount("Line 1\nLine 2\nLine 3")
	if count != 3 {
		t.Errorf("Expected 3 lines for 3-line message, got %d", count)
	}

	// Test message exceeding max lines (should be limited to 3)
	count = te.getDiagnosticLineCount("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")
	if count != 3 {
		t.Errorf("Expected 3 lines (max) for 5-line message, got %d", count)
	}

	// Test message with carriage returns
	count = te.getDiagnosticLineCount("Line 1\r\nLine 2")
	if count != 2 {
		t.Errorf("Expected 2 lines for message with carriage returns, got %d", count)
	}

	// Test message with empty lines (should be filtered out)
	count = te.getDiagnosticLineCount("Line 1\n\nLine 2\n   \nLine 3")
	if count != 3 {
		t.Errorf("Expected 3 lines for message with empty lines filtered out, got %d", count)
	}

	// Test message with only empty lines (should result in 0)
	count = te.getDiagnosticLineCount("\n\n   \n\t\n")
	if count != 0 {
		t.Errorf("Expected 0 lines for message with only empty lines, got %d", count)
	}
}

// TestDiagnosticRenderingWithEmptyLines tests that empty lines in diagnostic messages are filtered out
func TestDiagnosticRenderingWithEmptyLines(t *testing.T) {
	content := "line1\nline2\nline3\n"
	te := NewTestEditorWithContent(80, 20, content)

	// Inject a diagnostic with empty lines in the message (like the clang example)
	te.CurrentBuffer().Diagnostics = []proxy.Diagnostic{
		{Line: 2, Severity: "error", Message: "Too few arguments to function call, single argument 'arg1' was not specified\n\nbroken.c:1:5: note: 'f' declared here"},
	}

	// Position cursor on the diagnostic line
	te.SetCursorPosition(0, 1) // Line 2 (0-indexed)
	te.drawEditor()
	te.drawErrorLine()
	te.drawStatusBar()

	statusBarY := te.getStatusBarY()

	// Should only have 2 lines rendered (empty line filtered out)
	// Check first diagnostic line (2 lines above status bar)
	line1Y := statusBarY - 2
	line1 := te.mockScreen.GetLine(line1Y)
	if !strings.Contains(line1, "Too few arguments to function call") {
		t.Errorf("Expected first line of diagnostic, got: %q", line1)
	}

	// Check second diagnostic line (1 line above status bar)
	line2Y := statusBarY - 1
	line2 := te.mockScreen.GetLine(line2Y)
	if !strings.Contains(line2, "broken.c:1:5: note: 'f' declared here") {
		t.Errorf("Expected second line of diagnostic, got: %q", line2)
	}

	// Verify no third line was rendered (would be 3 lines above status bar)
	line3Y := statusBarY - 3
	line3 := te.mockScreen.GetLine(line3Y)
	// This line should be empty or contain content unrelated to diagnostics
	if strings.Contains(line3, "Too few arguments") || strings.Contains(line3, "broken.c") {
		t.Errorf("Expected no third diagnostic line, but got: %q", line3)
	}
}
