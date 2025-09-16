package editor

import "testing"

// Yank the first word from a single-line buffer and paste it at end-of-line.
// Expect the pasted word to appear immediately after the cursor on the same line.
// Note: We use "yiw" to yank just the word without trailing space.
func TestYankWordThenPasteAtEndOfLine(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello world")

	// Yank just the word "hello" (no trailing space)
	te.TypeKeys("yiw")

	// Go to end of line and paste after
	te.ExecuteCommand("end-of-line")
	te.ExecuteCommand("paste-after")

	// The word should be pasted right after, on the same line
	expected := "hello worldhello"
	actual := te.GetBufferContent()
	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}
