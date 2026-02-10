package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/syntax"
)

// TestLeftDiffPaneUsesSyntaxTokens ensures the left diff buffer (@HEAD), which has no RealPath,
// still renders using syntax tokens if a highlighter is set up.
func TestLeftDiffPaneUsesSyntaxTokens(t *testing.T) {
	// Minimal Go snippet with a keyword we can probe
	src := "package main\n\nfunc main() {}\n"

	// Create editor and a left (@HEAD) buffer with no real path
	te := NewTestEditorWithContent(100, 20, "")

	headName := "test.go@HEAD"
	headBuf := buffer.NewBuffer(src, headName, headName, config.IndentTabs, 4)
	headBuf.IsReadOnly = true
	headBuf.BufferName = headName
	headBuf.RealPath = "" // virtual

	// Right buffer (not strictly needed for this test but mirrors diff layout)
	curName := "test.go@current"
	curBuf := buffer.NewBuffer(src, curName, curName, config.IndentTabs, 4)
	curBuf.BufferName = curName
	curBuf.RealPath = "test.go" // pretend real file path

	// Install highlighters with Go language on both buffers
	headBuf.Highlighter = syntax.NewHighlighter()
	if err := headBuf.Highlighter.SetLanguageFromFile("test.go"); err != nil {
		t.Skipf("Skipping: Go grammar not available: %v", err)
	}
	_ = headBuf.Highlighter.ParseFull([]byte(src))

	curBuf.Highlighter = syntax.NewHighlighter()
	_ = curBuf.Highlighter.SetLanguageFromFile("test.go")
	_ = curBuf.Highlighter.ParseFull([]byte(src))

	// Register buffers and set up two windows
	te.buffers[headName] = headBuf
	te.buffers[curName] = curBuf
	te.createSecondWindow()
	te.windows[0].setActiveBuffer(headBuf)
	te.windows[1].setActiveBuffer(curBuf)
	te.activeWindow = 0
	te.switchToBuffer(headBuf)

	// Render left pane
	te.render()

	// Locate the line containing "func" in the head buffer content
	lines := strings.Split(src, "\n")
	funcLine := -1
	for i, ln := range lines {
		if strings.Contains(ln, "func ") {
			funcLine = i
			break
		}
	}
	if funcLine == -1 {
		t.Fatalf("did not find 'func' in test content")
	}

	// Compare foreground+attrs with the theme's keyword style
	expected := te.config.Theme.GetStyle(te.config.Theme.SyntaxKeyword)
	style := getCellStyle(te, funcLine, "func")
	if style == nil {
		t.Fatalf("could not locate 'func' on screen in left pane")
	}
	if !stylesMatch(*style, expected) {
		aFg, _, aAttr := style.Decompose()
		eFg, _, eAttr := expected.Decompose()
		t.Fatalf("left pane not using syntax tokens: got fg=%v attr=%v, want fg=%v attr=%v", aFg, aAttr, eFg, eAttr)
	}
}
