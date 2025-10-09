package editor

import (
	"strings"
	"testing"

	"veda/internal/config"
)

func TestInsertNewlineAdjustsIndentWithAnalyzer(t *testing.T) {
	content := "void foo() {\n\tfor (size_t i = 0; i < N; ++i) { spin_payload_handler(&slot_cache[i]); }\n}\n"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().FilePath = "/tmp/test.c"
	te.CurrentBuffer().Indentation.Type = config.IndentTabs
	te.CurrentBuffer().Indentation.Width = 4

	targetLine := "\tfor (size_t i = 0; i < N; ++i) { spin_payload_handler(&slot_cache[i]); }"
	col := strings.Index(targetLine, "spin_payload_handler")
	if col < 0 {
		t.Fatalf("failed to locate payload handler call in test content")
	}
	te.SetCursorPosition(col, 1)

	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatalf("enter-insert-mode failed: %v", err)
	}
	if err := te.ExecuteCommand("insert-newline"); err != nil {
		t.Fatalf("insert-newline failed: %v", err)
	}

	lines := strings.Split(te.GetBufferContent(), "\n")
	if len(lines) < 4 {
		t.Fatalf("unexpected buffer after first newline: %q", te.GetBufferContent())
	}
	if !strings.HasPrefix(lines[2], "\t\tspin_payload_handler") {
		t.Fatalf("expected line 2 to have double-tab indent, got %q", lines[2])
	}

	if err := te.ExecuteCommand("exit-insert-mode"); err != nil {
		t.Fatalf("exit-insert-mode failed: %v", err)
	}
	braceCol := strings.Index(lines[2], "}")
	if braceCol < 0 {
		t.Fatalf("failed to locate closing brace after first newline: %q", lines[2])
	}
	te.SetCursorPosition(braceCol, 2)

	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatalf("re-enter insert mode failed: %v", err)
	}
	if err := te.ExecuteCommand("insert-newline"); err != nil {
		t.Fatalf("second insert-newline failed: %v", err)
	}
	if err := te.ExecuteCommand("exit-insert-mode"); err != nil {
		t.Fatalf("exit-insert-mode failed: %v", err)
	}

	expected := "void foo() {\n\tfor (size_t i = 0; i < N; ++i) {\n\t\tspin_payload_handler(&slot_cache[i]);\n\t}\n}\n"
	got := te.GetBufferContent()
	if got != expected {
		t.Fatalf("unexpected buffer after analyzer-adjusted newlines.\nGot:\n%q\nWant:\n%q", got, expected)
	}
}
