package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
)

func TestMoveLinesUpReindent(t *testing.T) {
	content := strings.Join([]string{
		"if (foo) {",
		"    baz();",
		"}",
		"bar();",
	}, "\n") + "\n"

	te := NewTestEditorWithContent(80, 24, content)
	te.SetCursorPosition(0, 3)

	if err := te.ExecuteCommand("move-lines-up"); err != nil {
		t.Fatalf("move-lines-up failed: %v", err)
	}

	want := strings.Join([]string{
		"if (foo) {",
		"    baz();",
		"    bar();",
		"}",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("unexpected buffer after move-lines-up.\nGot:\n%s\nWant:\n%s", got, want)
	}

	if line := te.CurrentBuffer().CursorIterator.LineNumber(); line != 2 {
		t.Fatalf("expected cursor line 2 after move, got %d", line)
	}
}

func TestMoveLinesDownReindent(t *testing.T) {
	content := strings.Join([]string{
		"if (foo) {",
		"    bar();",
		"}",
		"baz();",
	}, "\n") + "\n"

	te := NewTestEditorWithContent(80, 24, content)
	te.SetCursorPosition(0, 1)

	if err := te.ExecuteCommand("move-lines-down"); err != nil {
		t.Fatalf("move-lines-down failed: %v", err)
	}

	want := strings.Join([]string{
		"if (foo) {",
		"}",
		"bar();",
		"baz();",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("unexpected buffer after move-lines-down.\nGot:\n%s\nWant:\n%s", got, want)
	}

	if line := te.CurrentBuffer().CursorIterator.LineNumber(); line != 2 {
		t.Fatalf("expected cursor line 2 after move down, got %d", line)
	}
}

func TestMoveLinesVisualLineSelection(t *testing.T) {
	content := strings.Join([]string{
		"line1",
		"line2",
		"line3",
		"line4",
	}, "\n") + "\n"

	te := NewTestEditorWithContent(80, 24, content)
	te.SetCursorPosition(0, 1)

	if err := te.ExecuteCommand("enter-visual-line-mode"); err != nil {
		t.Fatalf("enter-visual-line-mode failed: %v", err)
	}
	if err := te.ExecuteCommand("down"); err != nil {
		t.Fatalf("extending selection failed: %v", err)
	}

	if err := te.ExecuteCommand("move-lines-down"); err != nil {
		t.Fatalf("move-lines-down failed: %v", err)
	}

	want := strings.Join([]string{
		"line1",
		"line4",
		"line2",
		"line3",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("unexpected buffer after visual line move down.\nGot:\n%s\nWant:\n%s", got, want)
	}

	if te.mode != commands.VisualLineMode {
		t.Fatalf("expected to remain in visual line mode, got %v", te.mode)
	}
	if anchor := te.CurrentBuffer().VisualAnchorLine; anchor != 2 {
		t.Fatalf("expected anchor line 2 after move, got %d", anchor)
	}
	if line := te.CurrentBuffer().CursorIterator.LineNumber(); line != 3 {
		t.Fatalf("expected cursor line 3 after move, got %d", line)
	}
}

func TestMoveLinesBounds(t *testing.T) {
	content := strings.Join([]string{
		"top",
		"bottom",
	}, "\n") + "\n"

	te := NewTestEditorWithContent(80, 24, content)
	te.SetCursorPosition(0, 0)
	_ = te.ExecuteCommand("move-lines-up")
	if got := te.GetBufferContent(); got != content {
		t.Fatalf("moving top line up should be no-op")
	}

	te.SetCursorPosition(0, 1)
	_ = te.ExecuteCommand("move-lines-down")
	if got := te.GetBufferContent(); got != content {
		t.Fatalf("moving bottom line down should be no-op")
	}
}
