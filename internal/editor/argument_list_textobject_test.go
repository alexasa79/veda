package editor

import "testing"

func TestDaaOnPythonInnerListElement(t *testing.T) {
	content := "VARIABLES = [\n    Variable(\"bdl-bev\",    \"bdl_bev\",    \"boolint\", \"bdl_bev_default\",    [\"orca\", \"storage\"])\n]"
	te := NewTestEditorWithContent(80, 24, content)
	// Move cursor to the line containing the inner list element "orca"
	te.SetCursorPosition(0, 1)
	// Find column of 'o' in "orca"
	line := te.CurrentBuffer().CursorIterator.Line()
	col := -1
	for i := 0; i < len(line); i++ {
		if i+4 <= len(line) && line[i:i+4] == "orca" {
			col = i
			break
		}
	}
	if col == -1 {
		t.Fatalf("did not find 'orca' in line: %q", line)
	}
	te.SetCursorPosition(col, 1)

	// Delete around argument (should delete "orca" and the following comma/space)
	te.TypeKeys("daa")

	// Expected: only "storage" remains inside the inner list
	want := "VARIABLES = [\n    Variable(\"bdl-bev\",    \"bdl_bev\",    \"boolint\", \"bdl_bev_default\",    [\"storage\"])\n]"
	AssertBufferEquals(t, te, want)
}
