package editor

import (
	"strings"
	"testing"

	"veda/internal/indent"
)

func setBufferPath(te *TestEditor, path string) {
	buf := te.CurrentBuffer()
	oldPath := buf.FilePath
	buf.FilePath = path
	buf.RealPath = path
	if oldPath != path {
		delete(te.buffers, oldPath)
		te.buffers[path] = buf
		for i, p := range te.bufferOrder {
			if p == oldPath {
				te.bufferOrder[i] = path
				break
			}
		}
	}
}

func TestReindentExpression_FixesMisalignedInnerBlock(t *testing.T) {
	content := strings.Join([]string{
		"int main() {",
		"    if (x) {",
		"    foo();",
		"        }",
		"}",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "test1.c")
	te.SetCursorPosition(4, 2)

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
		t.Fatalf("unexpected buffer after reindent-expression.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestReindentExpression_MultilineConditionStaysAligned(t *testing.T) {
	content := strings.Join([]string{
		"void",
		"qux_override(void)",
		"{",
		"    if (FLAG_ZERO ||",
		"        mode_kind == MODE_PRIMARY ||",
		"        mode_kind == MODE_TRANSITION)",
		"    {",
		"        if (ctx.link_active) {",
		"            trace(LOG_INFO,",
		"                  \"placeholder\",",
		"                  mode_label(mode_kind));",
		"            ctx.link_active = FLAG_ZERO;",
		"        }",
		"    }",
		"}",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "test2.c")
	te.SetCursorPosition(0, 3)

	before := te.GetBufferContent()
	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}
	after := te.GetBufferContent()
	if after != before {
		t.Fatalf("expected content to remain unchanged, but got:\n%s", after)
	}
}

func TestReindentExpression_RespectsOuterComments(t *testing.T) {
	content := strings.Join([]string{
		"int sum(int a, int b) {",
		"    return a + b;",
		"}",
		"    // helper summary lives outside the function",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "test3.c")
	te.SetCursorPosition(4, 1)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	got := te.GetBufferContent()
	if got != content {
		t.Fatalf("trailing comment was modified.\nGot:\n%q\nWant:\n%q", got, content)
	}
}

func TestReindentExpression_ReflowsCallArguments(t *testing.T) {
	content := strings.Join([]string{
		"if outer {",
		"    foo(",
		"        a,",
		"    b,",
		"    c,",
		"    )",
		"}",
	}, "\n") + "\n"

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "test4.c")
	te.SetCursorPosition(0, 2)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	want := strings.Join([]string{
		"if outer {",
		"    foo(",
		"        a,",
		"        b,",
		"        c,",
		"    )",
		"}",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("unexpected content after reindent-expression.\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestReindentExpression_ReducesIndentAfterBlockExit(t *testing.T) {
	content := strings.Join([]string{
		"if (foo) {",
		"    baz();",
		"}",
		"    bar();",
	}, "\n") + "\n"

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "move_down_case.c")
	te.SetCursorPosition(0, 3)

	lang := indent.NewCLanguage()
	tokens, err := lang.Tokenize([]rune(content))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}
	annotated, err := indent.NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	line0 := make(map[int][2]int)
	for _, at := range annotated {
		if at.Token.Kind == indent.TokenWhitespace || at.Token.Kind == indent.TokenNewline {
			continue
		}
		if _, ok := line0[at.Token.Start.Line]; !ok {
			line0[at.Token.Start.Line] = [2]int{at.IndentLevel, at.ContinuationLevel}
		}
	}

	t.Logf("lineCount before: %d", te.CurrentBuffer().LineCount)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	want := strings.Join([]string{
		"if (foo) {",
		"    baz();",
		"}",
		"bar();",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("unexpected buffer after reindent-expression.\nGot:\n%s\nWant:\n%s\nindent map: %+v", got, want, line0)
	}
}
