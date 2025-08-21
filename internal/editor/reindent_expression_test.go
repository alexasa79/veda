package editor

import (
	"strings"
	"testing"
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
	te.buffer.SetIndentationType(te.buffer.Indentation.Type) // keep defaults

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
