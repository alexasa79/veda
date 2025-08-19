package editor

import (
	"testing"
)

func TestDDOriginalBugRegression(t *testing.T) {
	t.Run("dd deletes exactly one line - the bug that existed", func(t *testing.T) {
		// This test recreates the configgen.py scenario that exposed the original bug
		// where dd was deleting 2 lines instead of 1
		content := `def generate_parser(args):
    component = args.component

    print(f"Generating parser for {component}")

    with open('python/template_c.j2', 'r') as f:`
		
		te := NewTestEditorWithContent(80, 24, content)
		
		// Position cursor on line 33 equivalent (the print statement, line 3 0-indexed)
		te.SetCursorPosition(0, 3)
		
		// Record line count before deletion
		initialLineCount := te.buffer.LineCount
		t.Logf("Before dd: %d lines", initialLineCount)
		
		// Execute dd
		te.TypeKeys("dd")
		
		// The critical test: verify EXACTLY one line was deleted
		finalLineCount := te.buffer.LineCount
		linesDeleted := initialLineCount - finalLineCount
		
		if linesDeleted != 1 {
			t.Errorf("REGRESSION: dd deleted %d lines instead of 1", linesDeleted)
			t.Logf("This reproduces the original bug reported for configgen.py line 33")
		}
		
		// Verify the print line is gone and empty line remains
		expectedContent := `def generate_parser(args):
    component = args.component


    with open('python/template_c.j2', 'r') as f:`
		AssertBufferEquals(t, te, expectedContent)
	})
	
	t.Run("dd vs 2dd behavior verification", func(t *testing.T) {
		// Ensure single dd and multi-line dd behave differently
		// This test would also have caught the original bug
		content := `line1
line2  
line3
line4
line5`
		
		// Test single dd
		te1 := NewTestEditorWithContent(80, 24, content)
		te1.ExecuteCommand("down") // Move to line2
		te1.TypeKeys("dd")
		
		// Test 2dd  
		te2 := NewTestEditorWithContent(80, 24, content)
		te2.ExecuteCommand("down") // Move to line2
		te2.TypeKeys("2dd")
		
		// They should be different - this would have caught the original bug
		// If dd was incorrectly deleting 2 lines, then dd and 2dd would both delete 2 lines
		singleDDLines := te1.buffer.LineCount
		doubleDDLines := te2.buffer.LineCount
		
		if singleDDLines == doubleDDLines {
			t.Errorf("CRITICAL BUG: dd and 2dd produced same result (%d lines)", singleDDLines)
			t.Error("This indicates dd is deleting multiple lines when it should delete only 1")
		} else {
			t.Logf("✓ dd deleted to %d lines, 2dd deleted to %d lines (correct behavior)", singleDDLines, doubleDDLines)
		}
	})
}