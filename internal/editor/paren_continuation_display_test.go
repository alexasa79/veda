package editor

import (
	"testing"
	"veda/internal/config"
)

// Reproduces a scenario where an aligned multi-line function call contains a
// same-line cast on one of the argument lines. Both that line and the following
// aligned line should display the base brace level (L1) with +C (continuation),
// not a deep level derived from the alignment column.
func TestDisplayBaseLevelForSameLineCastContinuation(t *testing.T) {
	content := "" +
		"int main(void)\n" +
		"{\n" +
		"    flag = 1;\n" +
		"    result = compute_values(/*alpha=*/1,\n" +
		"                            /*beta=*/2,\n" +
		"                            (size_t)DEFAULT_SIZE,\n" + // same-line cast on this aligned line
		"                            /*gamma=*/3,\n" +
		"                            /*delta=*/4);\n" +
		"}\n"

	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().SetIndentationType(config.IndentSpaces)
	te.CurrentBuffer().SetIndentationWidth(4)

	// Line indices (0-based):
	// 0:int main, 1:{, 2:flag, 3:compute_values(, 4:beta, 5:(size_t)DEFAULT_SIZE, 6:gamma, 7:delta), 8:}

	// Check the cast line (index 5)
	te.SetCursorPosition(0, 5)
	info := te.getCurrentLineIndentInfo(te.CurrentBuffer().CursorIterator)
	if !info.IsContinuation || info.Level != 1 {
		t.Fatalf("cast line should display as L1+C; got L%d (continuation=%v)", info.Level, info.IsContinuation)
	}

	// Check the following aligned line (index 6)
	te.SetCursorPosition(0, 6)
	info = te.getCurrentLineIndentInfo(te.CurrentBuffer().CursorIterator)
	if !info.IsContinuation || info.Level != 1 {
		t.Fatalf("following line should display as L1+C; got L%d (continuation=%v)", info.Level, info.IsContinuation)
	}
}
