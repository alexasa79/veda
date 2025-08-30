package editor

import (
	"testing"
)

func TestWholeBufferMotionWithOperators(t *testing.T) {
	t.Run("yA yanks entire buffer into register and paste restores", func(t *testing.T) {
		content := "line1\nline2\nline3"
		te := NewTestEditorWithContent(80, 24, content)

		// Yank entire buffer into register 'a': ' a y A
		te.TypeKeys("'")
		te.TypeKeys("a")
		te.TypeKeys("y")
		te.TypeKeys("A")

		// Delete entire buffer to verify paste restores
		te.TypeKeys("d")
		te.TypeKeys("A")

		// After dA, buffer should be empty
		if got := te.GetBufferContent(); got != "" {
			t.Fatalf("expected empty buffer after dA, got %q", got)
		}

		// Paste from register 'a': ' a p
		te.TypeKeys("'")
		te.TypeKeys("a")
		te.TypeKeys("p")

		// Yanking full lines appends a trailing newline; expect content + "\n"
		want := content + "\n"
		AssertBufferEquals(t, te, want)
	})

	t.Run("dA deletes entire buffer", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "alpha\nbeta\n")
		te.TypeKeys("d")
		te.TypeKeys("A")
		if got := te.GetBufferContent(); got != "" {
			t.Fatalf("expected empty buffer after dA, got %q", got)
		}
	})

	t.Run("cA replaces entire buffer", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "foo\nbar")
		// Change entire buffer
		te.TypeKeys("c")
		te.TypeKeys("A")
		// Type replacement text and exit insert mode
		for _, ch := range "replaced" {
			_ = te.InsertChar(ch)
		}
		te.PressKey(KeyEscape, 0)

		AssertBufferEquals(t, te, "replaced")
	})
}
