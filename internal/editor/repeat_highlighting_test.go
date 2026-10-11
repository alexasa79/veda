package editor

import (
	"reflect"
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/syntax"
)

func TestRepeatLastChangeSyntaxHighlighting(t *testing.T) {
	const source = "void detach_nodes(void) {\n\tlist_delete(&first);\n\tlist_delete(&second);\n\tint remaining = 0;\n}\n"
	for _, tc := range []struct {
		name        string
		keys        string
		text        string
		replacement string
	}{
		{"change word", "cw", "list_unlink", "list_unlink"},
		{"insert text", "i", "new_", "new_list_delete"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, source)
			buf := te.CurrentBuffer()
			buf.FilePath = "repeat.c"
			buf.RealPath = "repeat.c"
			if err := buf.Highlighter.SetLanguage("c"); err != nil {
				t.Fatal(err)
			}
			if err := buf.Highlighter.ParseFull([]byte(source)); err != nil {
				t.Fatal(err)
			}

			assertHighlighting := func(stage string) {
				t.Helper()
				expected := syntax.NewHighlighter()
				if err := expected.SetLanguage("c"); err != nil {
					t.Fatal(err)
				}
				if err := expected.ParseFull([]byte(te.GetBufferContent())); err != nil {
					t.Fatal(err)
				}
				for line := 0; line < buf.LineCount; line++ {
					iter := buffer.NewBufferIteratorAtLine(buf, line)
					got := buf.Highlighter.GetTokensForLine(iter.Position(), iter.LineEnd())
					want := expected.GetTokensForLine(iter.Position(), iter.LineEnd())
					if !reflect.DeepEqual(got, want) {
						t.Errorf("%s: line %d highlighting differs from a fresh parse:\ngot  %v\nwant %v", stage, line+1, got, want)
					}
				}
			}

			assertHighlighting("before editing")
			te.SetCursorPosition(1, 1)
			te.TypeKeys(tc.keys)
			for _, ch := range tc.text {
				te.InsertChar(ch)
			}
			te.PressKey(KeyEscape, 0)
			AssertBufferEquals(t, te, strings.Replace(source, "list_delete", tc.replacement, 1))
			assertHighlighting("after original edit")

			te.SetCursorPosition(1, 2)
			te.TypeKeys(".")
			AssertBufferEquals(t, te, strings.ReplaceAll(source, "list_delete", tc.replacement))
			assertHighlighting("after dot repeat")
		})
	}
}
