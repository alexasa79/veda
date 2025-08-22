package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/ui"
)

// getWindowRenderedLine returns the full rendered row string for a given window and logical line index.
func getWindowRenderedLine(te *TestEditor, win *Window, lineNum int) string {
	// Global Y on screen = window top + (line - window view line)
	screenY := win.Y + (lineNum - win.ViewLine)
	if screenY < 0 {
		return ""
	}
	// Read across the window width
	var b strings.Builder
	for x := win.X; x < win.X+win.Width; x++ {
		if row, ok := te.mockScreen.cells[screenY]; ok {
			if cell, ok2 := row[x]; ok2 && cell.Rune != 0 {
				b.WriteRune(cell.Rune)
			} else {
				b.WriteRune(' ')
			}
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.TrimRight(b.String(), " ")
}

// getWindowTextStyle finds the style of a specific text on a line within a window and verifies that
// all runes of that text share the same style. Returns the first cell style and a boolean indicating
// whether all characters had identical styles.
func getWindowTextStyle(te *TestEditor, win *Window, lineNum int, text string) (*ui.Style, bool) {
	line := getWindowRenderedLine(te, win, lineNum)
	idx := strings.Index(line, text)
	if idx == -1 {
		return nil, false
	}
	var first *ui.Style
	screenY := win.Y + (lineNum - win.ViewLine)
	for i := 0; i < len(text); i++ {
		x := win.X + idx + i
		_, style := te.mockScreen.GetCell(x, screenY)
		if first == nil {
			copy := style
			first = &copy
		} else {
			if !stylesMatch(*first, style) {
				// Debug: log the differing run to help diagnose non-uniform styles
				fgs := make([]int32, 0, len(text))
				attrs := make([]ui.AttrMask, 0, len(text))
				for j := 0; j < len(text); j++ {
					cx := win.X + idx + j
					_, st := te.mockScreen.GetCell(cx, screenY)
					fg, _, attr := st.Decompose()
					fgs = append(fgs, int32(fg))
					attrs = append(attrs, attr)
				}
				testingLog := func(vals []int32) string {
					b := strings.Builder{}
					b.WriteString("[")
					for k, v := range vals {
						if k > 0 {
							b.WriteString(",")
						}
						b.WriteString(fmt.Sprintf("%d", v))
					}
					b.WriteString("]")
					return b.String()
				}
				_ = testingLog // avoid unused in non-debug platforms
				return first, false
			}
		}
	}
	return first, true
}

// getWindowTextForegrounds returns the foreground colors for each rune of text in the window line.
func getWindowTextForegrounds(te *TestEditor, win *Window, lineNum int, text string) ([]ui.Color, bool) {
	line := getWindowRenderedLine(te, win, lineNum)
	idx := strings.Index(line, text)
	if idx == -1 {
		return nil, false
	}
	fgs := make([]ui.Color, 0, len(text))
	screenY := win.Y + (lineNum - win.ViewLine)
	for i := 0; i < len(text); i++ {
		x := win.X + idx + i
		_, style := te.mockScreen.GetCell(x, screenY)
		fg, _, _ := style.Decompose()
		fgs = append(fgs, fg)
	}
	return fgs, true
}

// findTextInWindow scans visible lines in the window to locate text and returns its logical line number
// (in buffer coordinates) and the column within the window where the text begins.
func findTextInWindow(te *TestEditor, win *Window, text string) (int, int, bool) {
	// Scan all rows within the window height
	for line := win.ViewLine; line < win.ViewLine+win.Height && line < win.Buffer.LineCount; line++ {
		rendered := getWindowRenderedLine(te, win, line)
		if rendered == "" {
			continue
		}
		idx := strings.Index(rendered, text)
		if idx >= 0 {
			return line, idx, true
		}
	}
	return -1, -1, false
}

// getForegroundRun reads foreground colors for length runes starting at window column startCol
// on a given logical line number.
func getForegroundRun(te *TestEditor, win *Window, lineNum int, startCol int, length int) []ui.Color {
	fgs := make([]ui.Color, 0, length)
	screenY := win.Y + (lineNum - win.ViewLine)
	for i := 0; i < length; i++ {
		x := win.X + startCol + i
		_, style := te.mockScreen.GetCell(x, screenY)
		fg, _, _ := style.Decompose()
		fgs = append(fgs, fg)
	}
	return fgs
}

// Verifies that editing in the left window does not affect syntax highlighting
// of a token displayed in the right window.
func TestTwoWindowEditing_PreservesRightWindowTokenHighlighting(t *testing.T) {
	// Left buffer content (Go) – small file we'll edit
	leftContent := "package main\n\nfunc left() {\n\tfmt.Println(\"hi\")\n}\n"
	// Right buffer content (Go) – includes the identifier 'statusBarY'
	rightContent := "package main\n\nfunc right() {\n\tvar statusBarY int\n}\n"

	te := NewTestEditorWithContent(200, 30, leftContent)
	te.buffer.FilePath = "left.go"
	te.windows[0].Buffer = te.buffer
	te.render()

	te.TypeKeys("space v")
	te.render()

	if len(te.windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(te.windows))
	}

	right := te.windows[1]
	// Replace the right window's buffer with a fresh buffer to ensure a distinct highlighter
	newRight := buffer.NewBuffer(rightContent, "right.go", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
	right.Buffer = newRight
	// Ensure line count/width derived from content
	right.Buffer.UpdateLineNumberWidth()
	right.ViewLine = 0
	right.CursorLine = 3
	right.CursorX = 8

	te.render()

	// Make LEFT the active window so right remains consistently inactive (and narrow) before/after
	te.TypeKeys("space 1")
	te.render()

	// Find 'statusBarY' in the right window and capture per-char foregrounds
	rLine, rCol, ok := findTextInWindow(te, right, "statusBarY")
	if !ok {
		t.Fatal("could not find 'statusBarY' in right window before edits")
	}
	// Capture foreground colors for the token; uniformity is not required, only stability across edits
	fgsBefore := getForegroundRun(te, right, rLine, rCol, len("statusBarY"))

	// Edit LEFT window
	te.TypeKeys("space 1")
	left := te.windows[0]
	te.buffer = left.Buffer
	te.ExecuteCommand("enter-insert-mode")
	te.InsertChar('x')
	te.PressKey(KeyEscape, 0)

	te.render()

	// Re-find and compare 'statusBarY' in the right window
	rLine2, rCol2, ok := findTextInWindow(te, right, "statusBarY")
	if !ok {
		t.Fatal("could not find 'statusBarY' in right window after edits")
	}
	// Uniformity within the token is not required; only that per-char foregrounds remain stable
	fgsAfter := getForegroundRun(te, right, rLine2, rCol2, len("statusBarY"))
	if len(fgsBefore) != len(fgsAfter) {
		t.Fatalf("unexpected lens: %d vs %d", len(fgsBefore), len(fgsAfter))
	}
	for i := range fgsBefore {
		if fgsBefore[i] != fgsAfter[i] {
			t.Errorf("syntax highlighting skew: right 'statusBarY' fg changed at %d: before=%v after=%v", i, fgsBefore[i], fgsAfter[i])
		}
	}
}

// Reproduces skew where editing in a two-window layout corrupts highlighting
// of a token in the edited (left) window.
func TestTwoWindowEditing_PreservesLeftWindowTokenHighlighting(t *testing.T) {
	// Left buffer has the identifier that we will monitor
	leftContent := "package main\n\nfunc left() {\n\tvar statusBarY int\n\tfmt.Println(statusBarY)\n}\n"
	// Right buffer is a small unrelated Go file
	rightContent := "package main\n\nfunc right() {\n\treturn\n}\n"

	te := NewTestEditorWithContent(200, 30, leftContent)
	te.buffer.FilePath = "left.go"
	te.windows[0].Buffer = te.buffer
	te.render()

	// Split to create two windows
	te.TypeKeys("space v")
	te.render()
	if len(te.windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(te.windows))
	}

	// Put content into right window
    right := te.windows[1]
    if right.Buffer == nil {
        right.Buffer = buffer.NewBuffer(rightContent, "right.go", te.config.DefaultIndentType, te.config.DefaultIndentWidth)
    } else {
        right.Buffer.Rope = buffer.NewRopeFromString(rightContent)
        right.Buffer.LineCount = strings.Count(rightContent, "\n") + 1
        right.Buffer.FilePath = "right.go"
        right.Buffer.UpdateLineNumberWidth()
    }
	right.ViewLine = 0
	right.CursorLine = 0
	right.CursorX = 0

	// Keep LEFT active before and after
	te.TypeKeys("space 1")
	te.render()

	// Locate 'statusBarY' in the LEFT window and capture per-char foregrounds
	leftWin := te.windows[0]
	lLine, lCol, ok := findTextInWindow(te, leftWin, "statusBarY")
	if !ok {
		t.Fatal("could not find 'statusBarY' in left window before edits")
	}
	// Allow for different styles inside identifier (themes may color parts differently). Capture fg run only.
	fgsBefore := getForegroundRun(te, leftWin, lLine, lCol, len("statusBarY"))

	// Edit the LEFT buffer with a benign change that keeps the file valid.
	// Move cursor to the println line (index 4) and insert a space at column 0.
	te.buffer = leftWin.Buffer
	te.SetCursorPosition(0, 4)
	te.ExecuteCommand("enter-insert-mode")
	te.InsertChar(' ')
	te.PressKey(KeyEscape, 0)

	te.render()

	// Re-find 'statusBarY' and verify foregrounds unchanged
	lLine2, lCol2, ok := findTextInWindow(te, leftWin, "statusBarY")
	if !ok {
		t.Fatal("could not find 'statusBarY' in left window after edits")
	}
	fgsAfter := getForegroundRun(te, leftWin, lLine2, lCol2, len("statusBarY"))
	if len(fgsBefore) != len(fgsAfter) {
		t.Fatalf("unexpected lens: %d vs %d", len(fgsBefore), len(fgsAfter))
	}
	for i := range fgsBefore {
		if fgsBefore[i] != fgsAfter[i] {
			t.Errorf("syntax highlighting skew: left 'statusBarY' fg changed at %d: before=%v after=%v", i, fgsBefore[i], fgsAfter[i])
		}
	}
}
