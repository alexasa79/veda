package editor

import (
	"testing"
	"veda/internal/buffer"
)

func TestSignatureVisibleOnFunctionName(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "f1(a, b)\nzz")
	te.CurrentBuffer().FilePath = "test.go"

	// Seed a signature as if received from LSP
	te.CurrentBuffer().CurrentSignature = &buffer.SignatureHelp{
		Label:           "f1(a, b)",
		ActiveParameter: -1,
		Parameters:      []buffer.ParameterInfo{{Label: "a"}, {Label: "b"}},
	}

	// Place cursor on the function name 'f1'
	te.SetCursorPosition(0, 0)
	te.updateCursorState()

	if te.CurrentBuffer().CurrentSignature == nil {
		t.Fatalf("signature should remain visible on function name")
	}
}

func TestSignatureHiddenOutsideCallSpan(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "f1(a, b)\nzz")
	te.CurrentBuffer().FilePath = "test.go"

	// Seed signature
	te.CurrentBuffer().CurrentSignature = &buffer.SignatureHelp{
		Label:           "f1(a, b)",
		ActiveParameter: 0,
		Parameters:      []buffer.ParameterInfo{{Label: "a"}, {Label: "b"}},
	}

	// Move cursor to a position clearly outside the call span (next line)
	te.SetCursorPosition(0, 1)
	te.updateCursorState()

	if te.CurrentBuffer().CurrentSignature != nil {
		t.Fatalf("signature should be hidden when cursor moves outside call span")
	}
}
