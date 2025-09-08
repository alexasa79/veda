package editor

import (
	"strings"
	"testing"
	"veda/internal/config"
)

// This test reproduces a scenario with a long, aligned multi-line call where
// arguments are indented far to the right (continuation alignment). After the
// closing ")" and statement terminator ";", opening a new line (like Vim 'o')
// should return to the base block indentation (L1), not keep the deep alignment.
func TestOpenLineBelowAfterAlignedCallReturnsToBaseIndent(t *testing.T) {
	content := strings.Join([]string{
		"int main(void)",
		"{",
		"    result = long_function_name_with_many_parts(",
		"                                            /*opt_a*/1,",
		"                                            /*opt_b*/2,",
		"                                            /*opt_c*/3);",
		"}",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().SetIndentationType(config.IndentSpaces)
	te.CurrentBuffer().SetIndentationWidth(4)

	// Place cursor at end of the line containing the ");"
	lines := strings.Split(content, "\n")
	endLineIndex := 5 // zero-based index of the line with the closing ");"
	te.SetCursorPosition(len(lines[endLineIndex]), endLineIndex)

	// Execute open-line-below (like pressing 'o' in normal mode)
	if err := te.ExecuteCommand("open-line-below"); err != nil {
		t.Fatalf("open-line-below failed: %v", err)
	}

	// The editor inserts a newline and auto-indentation, and moves to that line in insert mode.
	// Verify that the new line is indented to the base block level (4 spaces: L1), not the deep alignment.
	// Current cursor is at start of content (after indentation).
	_, y := te.GetCursorPosition()
	if y != endLineIndex+1 {
		t.Fatalf("expected cursor to be on new line %d, got %d", endLineIndex+1, y)
	}

	// Fetch the new line text
	it := te.CurrentBuffer().CursorIterator
	line := it.Line()
	// Count leading spaces
	leading := 0
	for leading < len(line) && line[leading] == ' ' {
		leading++
	}
	if leading != 4 {
		t.Fatalf("expected new line indentation of 4 spaces (L1), got %d", leading)
	}

	// Also verify the displayed indent info for the continuation lines shows L1+C
	// Move cursor to a continuation argument line and read indent info via editor helper
	te.SetCursorPosition(0, 3) // a continuation line (/*opt_b*/2,)
	info := te.getCurrentLineIndentInfo(te.CurrentBuffer().CursorIterator)
	if !info.IsContinuation || info.Level != 1 {
		t.Fatalf("expected continuation line to display as L1+C; got L%d (continuation=%v)", info.Level, info.IsContinuation)
	}
}
