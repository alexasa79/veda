package editor

import (
	"strings"
	"testing"
)

func TestPythonReindent_AlignedCallIdempotent(t *testing.T) {
	lines := []string{
		"def foxtrot():",
		"    cluster = init_cluster(",
		"        # ensure bootstrap happens first",
		"        primary_host,",
		"        standby_host,",
		"    )",
		"",
		"    return cluster",
	}
	content := strings.Join(lines, "\n") + "\n"

	te := NewTestEditorWithContent(120, 40, content)
	setBufferPath(te, "config.py")
	te.SetCursorPosition(8, 2)

	before := te.GetBufferContent()
	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}
	afterFirst := te.GetBufferContent()
	if afterFirst != before {
		t.Fatalf("python call mutated unexpectedly.\nBefore:\n%s\nAfter:\n%s", before, afterFirst)
	}

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("second python reindent failed: %v", err)
	}
	afterSecond := te.GetBufferContent()
	if afterSecond != afterFirst {
		t.Fatalf("python call changed across runs.\nFirst:\n%s\nSecond:\n%s", afterFirst, afterSecond)
	}
}

func TestPythonReindent_RealignsNestedList(t *testing.T) {
	lines := []string{
		"def build():",
		"    values = [",
		"        'alpha',",
		"    'beta',",
		"            'gamma',",
		"    ]",
	}
	content := strings.Join(lines, "\n") + "\n"

	te := NewTestEditorWithContent(120, 40, content)
	setBufferPath(te, testFixturePath("list_fix.py"))
	te.SetCursorPosition(4, 3)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	want := strings.Join([]string{
		"def build():",
		"    values = [",
		"        'alpha',",
		"        'beta',",
		"        'gamma',",
		"    ]",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("python list not realigned.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
