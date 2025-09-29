package editor

import (
	"strings"
	"testing"
)

func TestGoReindent_CompositeLiteralIdempotent(t *testing.T) {
	lines := []string{
		"package sample",
		"",
		"func build() map[string]int {",
		"    values := map[string]int{",
		"        \"alpha\": 1,",
		"        \"beta\": 2,",
		"    }",
		"    return values",
		"}",
	}
	content := strings.Join(lines, "\n") + "\n"

	te := NewTestEditorWithContent(120, 40, content)
	setBufferPath(te, "compose.go")
	te.SetCursorPosition(8, 4)

	before := te.GetBufferContent()
	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}
	afterFirst := te.GetBufferContent()
	if afterFirst != before {
		t.Fatalf("Go composite literal changed unexpectedly.\nBefore:\n%s\nAfter:\n%s", before, afterFirst)
	}

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("second Go reindent failed: %v", err)
	}
	afterSecond := te.GetBufferContent()
	if afterSecond != afterFirst {
		t.Fatalf("Go composite literal drifted across runs.\nFirst:\n%s\nSecond:\n%s", afterFirst, afterSecond)
	}
}

func TestGoReindent_FixesMisalignedBlock(t *testing.T) {
	lines := []string{
		"package sample",
		"",
		"func handle(flag bool) {",
		"    if flag {",
		"    println(\"flagged\")",
		"        println(\"done\")",
		"    }",
		"}",
	}
	content := strings.Join(lines, "\n") + "\n"

	te := NewTestEditorWithContent(120, 40, content)
	setBufferPath(te, "handle.go")
	te.SetCursorPosition(4, 4)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	want := strings.Join([]string{
		"package sample",
		"",
		"func handle(flag bool) {",
		"    if flag {",
		"        println(\"flagged\")",
		"        println(\"done\")",
		"    }",
		"}",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("Go block not corrected.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
