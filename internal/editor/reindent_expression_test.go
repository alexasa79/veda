package editor

import (
	"strings"
	"testing"

	"veda/internal/proxy"
)

func TestReindentExpression_UsesBlockAndDedentsBraces(t *testing.T) {
	content := strings.Join([]string{
		"int main() {",
		"    if (x) {",
		"    foo();",
		"        }",
		"}",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	// Place cursor inside the if block content
	te.SetCursorPosition(4, 2) // zero-based: line "    foo();"

	// Ensure spaces indentation with width 4
	te.CurrentBuffer().SetIndentationType(te.CurrentBuffer().Indentation.Type) // keep defaults

	// Execute expression reindent
	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	got := te.GetBufferContent()
	want := strings.Join([]string{
		"int main() {",
		"    if (x) {",
		"        foo();",
		"    }",
		"}",
	}, "\n")
	if got != want {
		t.Errorf("unexpected content after reindent-expression.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestReindentExpression_CyclesRulerIndexWithoutChangingIndent(t *testing.T) {
	content := strings.Join([]string{
		"void f() {",
		"    a(\n        1,\n        2\n    );",
		"}",
	}, "\n")
	te := NewTestEditorWithContent(80, 24, content)
	te.config.VerticalRulers = []int{40, 60, 80}

	te.SetCursorPosition(4, 1)

	// First call sets index to 0
	_ = te.ExecuteCommand("reindent-expression")
	if te.Editor.lastRewrapIndex != 0 {
		t.Fatalf("expected lastRewrapIndex 0, got %d", te.Editor.lastRewrapIndex)
	}
	// Second call at same position cycles to 1
	_ = te.ExecuteCommand("reindent-expression")
	if te.Editor.lastRewrapIndex != 1 {
		t.Fatalf("expected lastRewrapIndex 1, got %d", te.Editor.lastRewrapIndex)
	}
}

func TestReindentExpression_AlignsFunctionArguments(t *testing.T) {
	content := strings.Join([]string{
		"int res = function(arg1,",
		"              arg2,",
		"              arg3,",
		"              arg4);",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	te.SetCursorPosition(len("              "), 1) // Cursor on the start of arg2

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	got := te.GetBufferContent()
	want := strings.Join([]string{
		"int res = function(arg1,",
		"                   arg2,",
		"                   arg3,",
		"                   arg4);",
	}, "\n")
	if got != want {
		t.Fatalf("unexpected content after reindent-expression.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestReindentExpression_TrailingCommentWithinFoldingRange(t *testing.T) {
	content := strings.Join([]string{
		"int sum(int a, int b) {",
		"    return a + b;",
		"}",
		"    // helper summary lives outside the function",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().FoldingRanges = []proxy.FoldingRange{
		{StartLine: 0, EndLine: 3},
	}

	te.SetCursorPosition(4, 1) // Inside the function body

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	got := te.GetBufferContent()
	if got != content {
		t.Fatalf("trailing comment outside function was modified.\nGot:\n%q\nWant:\n%q", got, content)
	}
}
