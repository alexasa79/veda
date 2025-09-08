package editor

import (
	"testing"
	"veda/internal/events"
	"veda/internal/syntax"
)

func TestCtrlSlashCommentsVisualLineSelection(t *testing.T) {
	content := "int a;\nint b;\nint c;\n"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	_ = te.CurrentBuffer().Highlighter.SetLanguageFromFile("main.cpp")

	// Start on first code line
	te.SetCursorPosition(0, 0)
	// Enter visual line mode and extend selection to cover first two lines
	if err := te.ExecuteCommand("enter-visual-line-mode"); err != nil {
		t.Fatalf("enter-visual-line-mode failed: %v", err)
	}
	if err := te.ExecuteCommand("down"); err != nil { // include next line
		t.Fatalf("down failed: %v", err)
	}

	// Press Ctrl+/ to toggle comment on the selection
	te.handleEvent(&events.KeyEvent{Key: events.KeyCtrlUnderscore})

	expected := "// int a;\n// int b;\nint c;\n"
	AssertBufferEquals(t, te, expected)

	// Toggle back (selection still active)
	te.handleEvent(&events.KeyEvent{Key: events.KeyCtrlUnderscore})
	AssertBufferEquals(t, te, content)
}
