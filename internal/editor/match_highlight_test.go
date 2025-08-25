package editor

import (
    "strings"
    "testing"
)

// Test that matching bracket across lines gets MatchHighlight background
func TestMatchHighlight_BracketsAcrossLines(t *testing.T) {
    te := NewTestEditorWithContent(120, 20, "{\n\n}\n")
    // Place cursor on the opening bracket at (x=0,y=0)
    te.SetCursorPosition(0, 0)

    // Render to compute match and draw
    te.render()

    // Expected background color for match highlight
    matchBg := te.config.Theme.GetStyle(te.config.Theme.MatchHighlight).Bg

    // Compute screen coordinates for matching '}' at line 2, column 0
    lnw := te.CurrentBuffer().LineNumberWidth
    contentTop := te.Editor.getContentStartY()
    screenX := lnw + 0
    screenY := contentTop + 2

    _, style := te.mockScreen.GetCell(screenX, screenY)
    if style.Bg != matchBg {
        t.Fatalf("expected match highlight Bg=%v at (%d,%d), got Bg=%v", matchBg, screenX, screenY, style.Bg)
    }

    // Ensure gutter cell is not highlighted by mistake
    if lnw > 0 {
        _, gutterStyle := te.mockScreen.GetCell(lnw-1, screenY)
        if gutterStyle.Bg == matchBg {
            t.Fatalf("unexpected match highlight applied in gutter at (%d,%d)", lnw-1, screenY)
        }
    }
}

// Test that matching quotes on the same line get MatchHighlight background
func TestMatchHighlight_QuotesSameLine(t *testing.T) {
    line := `var s = "hello"`
    te := NewTestEditorWithContent(120, 10, line)

    // Find positions of the quotes
    first := strings.Index(line, "\"")
    last := strings.LastIndex(line, "\"")
    if first == -1 || last == -1 || first == last {
        t.Fatalf("test setup failed to locate quotes in %q", line)
    }

    // Place cursor on the first quote
    te.SetCursorPosition(first, 0)
    te.render()

    matchBg := te.config.Theme.GetStyle(te.config.Theme.MatchHighlight).Bg

    lnw := te.CurrentBuffer().LineNumberWidth
    contentTop := te.Editor.getContentStartY()
    screenX := lnw + last
    screenY := contentTop + 0

    _, style := te.mockScreen.GetCell(screenX, screenY)
    if style.Bg != matchBg {
        t.Fatalf("expected match highlight Bg=%v at (%d,%d) for matching quote, got Bg=%v", matchBg, screenX, screenY, style.Bg)
    }

    // Also ensure the line number cell on this row is not highlighted
    if lnw > 0 {
        _, gutterStyle := te.mockScreen.GetCell(lnw-1, screenY)
        if gutterStyle.Bg == matchBg {
            t.Fatalf("unexpected match highlight applied in gutter at (%d,%d)", lnw-1, screenY)
        }
    }
}

