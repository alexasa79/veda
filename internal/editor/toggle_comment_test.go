package editor

import (
	"testing"
	"veda/internal/events"
	"veda/internal/syntax"
)

func TestToggleCommentSingleLine(t *testing.T) {
	content := "int main() {\n    return 0;\n}\n"
	te := NewTestEditorWithContent(80, 24, content)
	// Use a C-like language to get '//' style
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	_ = te.CurrentBuffer().Highlighter.SetLanguageFromFile("test.c")

	// Place cursor on line with code
	te.SetCursorPosition(0, 1)

	// Toggle comment once -> adds // after indentation
	if err := te.ExecuteCommand("toggle-comment"); err != nil {
		t.Fatalf("toggle-comment failed: %v", err)
	}

	expected := "int main() {\n    // return 0;\n}\n"
	AssertBufferEquals(t, te, expected)

	// Toggle again -> removes comment
	if err := te.ExecuteCommand("toggle-comment"); err != nil {
		t.Fatalf("toggle-comment failed (second): %v", err)
	}
	AssertBufferEquals(t, te, content)
}

func TestToggleCommentWithCount(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\n"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	_ = te.CurrentBuffer().Highlighter.SetLanguageFromFile("test.go")

	// Start on line2 (index 1)
	te.SetCursorPosition(0, 1)

	// Apply count = 3 -> comment lines 2..4
	te.repeatCount = 3
	if err := te.executeWithRepeatCount("toggle-comment"); err != nil {
		t.Fatalf("toggle-comment with count failed: %v", err)
	}

	expected := "line1\n// line2\n// line3\n// line4\nline5\n"
	AssertBufferEquals(t, te, expected)

	// Now uncomment the same 3 lines
	te.repeatCount = 3
	if err := te.executeWithRepeatCount("toggle-comment"); err != nil {
		t.Fatalf("toggle-comment with count (uncomment) failed: %v", err)
	}
	AssertBufferEquals(t, te, content)
}

func TestCtrlSlashKeybindingWithCount(t *testing.T) {
	content := "a\nb\nc\n"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	_ = te.CurrentBuffer().Highlighter.SetLanguageFromFile("main.cpp")

	// Move to second line (index 1)
	te.SetCursorPosition(0, 1)

	// Build numeric count prefix '2'
	te.TypeKeys("2")

	// Press Ctrl+/
	te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '/', Modifiers: events.ModCtrl})

	expected := "a\n// b\n// c\n"
	AssertBufferEquals(t, te, expected)

	// Now uncomment the same two lines using count again
	te.TypeKeys("2")
	te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '/', Modifiers: events.ModCtrl})
	AssertBufferEquals(t, te, content)
}

func TestToggleCommentPythonLinePrefix(t *testing.T) {
	content := "def f():\n    return 42\n"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	_ = te.CurrentBuffer().Highlighter.SetLanguageFromFile("script.py")

	// Toggle on second line (index 1)
	te.SetCursorPosition(0, 1)
	if err := te.ExecuteCommand("toggle-comment"); err != nil {
		t.Fatalf("toggle-comment failed: %v", err)
	}
	expected := "def f():\n    # return 42\n"
	AssertBufferEquals(t, te, expected)

	// Toggle back
	if err := te.ExecuteCommand("toggle-comment"); err != nil {
		t.Fatalf("toggle-comment failed: %v", err)
	}
	AssertBufferEquals(t, te, content)
}

func TestToggleCommentUnsupportedLanguageNoOp(t *testing.T) {
	content := "# Heading\nSome text\n"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	_ = te.CurrentBuffer().Highlighter.SetLanguageFromFile("README.md") // markdown -> no line comment

	// Attempt to toggle
	te.SetCursorPosition(0, 1)
	if err := te.ExecuteCommand("toggle-comment"); err != nil {
		t.Fatalf("toggle-comment failed: %v", err)
	}
	// Expect unchanged
	AssertBufferEquals(t, te, content)
}
