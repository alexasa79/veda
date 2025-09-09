package editor

import (
	"testing"
	"veda/internal/buffer"
)

// Verifies that active parameter detection updates when moving cursor up/down
// across a multi-line function call while a signature is visible.
func TestSignatureActiveParameterUpdatesOnVerticalMovement(t *testing.T) {
	content := "f1(a,\n  b,\n  c)"
	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().FilePath = "test.go"

	// Seed a visible signature as if received from LSP
	te.CurrentBuffer().CurrentSignature = &buffer.SignatureHelp{
		Label:           "f1(a, b, c)",
		ActiveParameter: 0,
		Parameters: []buffer.ParameterInfo{
			{Label: "a"}, {Label: "b"}, {Label: "c"},
		},
	}

	// Place cursor on 'a' (line 0, char index 3)
	te.SetCursorPosition(3, 0)
	te.updateCursorState()
	if ap := te.CurrentBuffer().CurrentSignature.ActiveParameter; ap != 0 {
		t.Fatalf("expected ActiveParameter=0 at 'a', got %d", ap)
	}

	// Move down to 'b' (line 1, char index 2)
	te.SetCursorPosition(2, 1)
	te.updateCursorState()
	if ap := te.CurrentBuffer().CurrentSignature.ActiveParameter; ap != 1 {
		t.Fatalf("expected ActiveParameter=1 at 'b', got %d", ap)
	}

	// Move down to 'c' (line 2, char index 2)
	te.SetCursorPosition(2, 2)
	te.updateCursorState()
	if ap := te.CurrentBuffer().CurrentSignature.ActiveParameter; ap != 2 {
		t.Fatalf("expected ActiveParameter=2 at 'c', got %d", ap)
	}

	// Move back up to 'b'
	te.SetCursorPosition(2, 1)
	te.updateCursorState()
	if ap := te.CurrentBuffer().CurrentSignature.ActiveParameter; ap != 1 {
		t.Fatalf("expected ActiveParameter=1 after moving up to 'b', got %d", ap)
	}
}
