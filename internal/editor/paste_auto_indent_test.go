package editor

import (
	"strings"
	"testing"
	"veda/internal/clipboard"
	"veda/internal/syntax"
)

// These tests focus on line-wise, multi-line paste auto-indentation behavior.
func TestPasteAutoIndent_Linewise_SpacesToSpaces(t *testing.T) {
	// Target file with 4-space indentation, cursor at L1
	initial := strings.Join([]string{
		"func main() {",
		"    // here",
		"}",
	}, "\n")
	te := NewTestEditorWithContent(80, 24, initial)
	te.CurrentBuffer().FilePath = "test.go"
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	if err := te.CurrentBuffer().Highlighter.SetLanguageFromFile(te.CurrentBuffer().FilePath); err != nil {
		t.Skipf("highlighter not available: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(te.CurrentBuffer().Rope.String()))

	// Pasted block is at L2 (inside function) but with 8 spaces (L2)
	// We expect it to be rebased to 4 spaces (L1) when pasted after current line
	pasted := strings.Join([]string{
		"        if ok {",
		"            println(\"x\")",
		"        }",
		"",
	}, "\n")
	if err := clipboard.Copy(pasted); err != nil {
		t.Skipf("clipboard not available: %v", err)
	}

	// Place cursor on line 1 (index 1), which has 4 spaces (L1)
	te.SetCursorPosition(0, 1)
	// Paste after (line-wise)
	if err := te.ExecuteCommand("paste-after"); err != nil {
		t.Fatal(err)
	}

	got := te.GetBufferContent()
	want := strings.Join([]string{
		"func main() {",
		"    // here",
		"    if ok {",
		"        println(\"x\")",
		"    }",
		"}",
	}, "\n")

	if got != want {
		t.Errorf("unexpected content after auto-indent paste.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestPasteAutoIndent_Linewise_TabsToTabs(t *testing.T) {
	// Target file with tabs; make a simple C snippet
	initial := strings.Join([]string{
		"int main() {",
		"\t// here",
		"}",
	}, "\n")
	te := NewTestEditorWithContent(80, 24, initial)
	te.CurrentBuffer().FilePath = "test.c"
	te.CurrentBuffer().SetIndentationType(te.CurrentBuffer().Indentation.Type) // keep defaults
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	if err := te.CurrentBuffer().Highlighter.SetLanguageFromFile(te.CurrentBuffer().FilePath); err != nil {
		t.Skipf("highlighter not available: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(te.CurrentBuffer().Rope.String()))

	// Pasted block with two tabs common indent; expect it to rebase to one tab at target
	pasted := strings.Join([]string{
		"\t\tif (ok) {",
		"\t\t\tputs(\"x\");",
		"\t\t}",
		"",
	}, "\n")
	if err := clipboard.Copy(pasted); err != nil {
		t.Skipf("clipboard not available: %v", err)
	}

	// Place cursor on line 1 (index 1), which has one tab
	te.SetCursorPosition(0, 1)
	if err := te.ExecuteCommand("paste-after"); err != nil {
		t.Fatal(err)
	}

	got := te.GetBufferContent()
	want := strings.Join([]string{
		"int main() {",
		"\t// here",
		"\tif (ok) {",
		"\t\tputs(\"x\");",
		"\t}",
		"}",
	}, "\n")

	if got != want {
		t.Errorf("unexpected content after tabs auto-indent paste.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestPasteAutoIndent_SkipsOnAmbiguousOrNonCode(t *testing.T) {
	// Non-code markdown buffer should not re-indent
	initial := strings.Join([]string{
		"# Title",
		"Some text",
	}, "\n")
	te := NewTestEditorWithContent(80, 24, initial)
	te.CurrentBuffer().FilePath = "doc.md"
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	if err := te.CurrentBuffer().Highlighter.SetLanguageFromFile(te.CurrentBuffer().FilePath); err != nil {
		t.Skipf("highlighter not available: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(te.CurrentBuffer().Rope.String()))

	pasted := "    code block\n    more\n"
	if err := clipboard.Copy(pasted); err != nil {
		t.Skipf("clipboard not available: %v", err)
	}

	// Paste after line 0
	te.SetCursorPosition(0, 0)
	if err := te.ExecuteCommand("paste-after"); err != nil {
		t.Fatal(err)
	}

	got := te.GetBufferContent()
	// Should be verbatim insert after first line
	want := strings.Join([]string{
		"# Title",
		"    code block",
		"    more",
		"Some text",
	}, "\n")

	if got != want {
		t.Errorf("markdown paste should remain verbatim.\nGot:\n%q\nWant:\n%q", got, want)
	}

	// Ambiguous indent: mixed tabs/spaces in pasted block -> skip adjusting
	initial2 := strings.Join([]string{
		"func f() {",
		"    // here",
		"}",
	}, "\n")
	te2 := NewTestEditorWithContent(80, 24, initial2)
	te2.CurrentBuffer().FilePath = "test.go"
	te2.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	if err := te2.CurrentBuffer().Highlighter.SetLanguageFromFile(te2.CurrentBuffer().FilePath); err != nil {
		t.Skipf("highlighter not available: %v", err)
	}
	te2.CurrentBuffer().Highlighter.ParseFull([]byte(te2.CurrentBuffer().Rope.String()))

	// Mixed indent: tab then spaces on the next line
	pasted2 := "\tline1\n    line2\n"
	if err := clipboard.Copy(pasted2); err != nil {
		t.Skipf("clipboard not available: %v", err)
	}

	te2.SetCursorPosition(0, 1)
	if err := te2.ExecuteCommand("paste-after"); err != nil {
		t.Fatal(err)
	}

	got2 := te2.GetBufferContent()
	want2 := strings.Join([]string{
		"func f() {",
		"    // here",
		"\tline1",
		"    line2",
		"}",
	}, "\n")

	if got2 != want2 {
		t.Errorf("ambiguous indent paste should be verbatim.\nGot:\n%q\nWant:\n%q", got2, want2)
	}
}
