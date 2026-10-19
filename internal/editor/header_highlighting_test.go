package editor

import (
	"testing"

	"veda/internal/syntax"
)

func TestHeaderHighlightingOnBufferSwitch(t *testing.T) {
	code := "using Index = unsigned long;"
	te := NewTestEditorWithContent(80, 24, code)
	buf := te.CurrentBuffer()
	buf.FilePath = "example.h"
	buf.RealPath = "example.h"
	te.switchToBuffer(buf)
	te.renderPane()
	if got := buf.Highlighter.GetCurrentLanguage(); got != "cpp" {
		t.Fatalf("language = %q, want cpp", got)
	}
	style := getCellStyle(te, 0, "using")
	want := te.config.Theme.GetStyle(te.config.Theme.SyntaxKeyword)
	if style == nil || !stylesMatch(*style, want) {
		t.Fatal("using is not rendered with the keyword style")
	}

	// Switching to the same header again must preserve its detected language.
	te.switchToBuffer(buf)
	if got := buf.Highlighter.GetCurrentLanguage(); got != "cpp" {
		t.Fatalf("language after switching = %q, want cpp", got)
	}
}

func TestDiffHeaderLanguageDetection(t *testing.T) {
	ds := NewDiffSession("int index;", "using Index = unsigned long;", "example.h")
	if got := ds.HighlighterA.GetCurrentLanguage(); got != "c" {
		t.Fatalf("previous language = %q, want c", got)
	}
	if got := ds.HighlighterB.GetCurrentLanguage(); got != "cpp" {
		t.Fatalf("current language = %q, want cpp", got)
	}
}

func TestHeaderHighlightingInitialization(t *testing.T) {
	for _, applyEdit := range []bool{false, true} {
		te := NewTestEditorWithContent(80, 24, "using Index = unsigned long;")
		buf := te.CurrentBuffer()
		buf.FilePath, buf.RealPath = "example.h", "example.h"
		buf.Highlighter = nil
		if applyEdit {
			te.updateSyntaxApplyEdit(0, "", "")
		} else {
			te.updateSyntaxHighlighting()
		}
		if got := buf.Highlighter.GetCurrentLanguage(); got != "cpp" {
			t.Fatalf("applyEdit=%v: language = %q, want cpp", applyEdit, got)
		}
		if tokens := buf.Highlighter.GetTokensForLine(0, 28); len(tokens) == 0 || tokens[0].TokenType != syntax.TokenTypeKeyword {
			t.Fatalf("applyEdit=%v: using is not a keyword: %v", applyEdit, tokens)
		}
	}
}
