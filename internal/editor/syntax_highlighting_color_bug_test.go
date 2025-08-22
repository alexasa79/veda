package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/syntax"
)

func TestSyntaxHighlightingKeywordColorBug(t *testing.T) {
	// This test reproduces a bug where typing "class" above an existing "def"
	// causes the "def" keyword to lose its syntax highlighting color and become white

	t.Run("def keyword loses color when class is typed above", func(t *testing.T) {
		// Create simple Python content to test with
		pythonCode := `#!/usr/bin/env python3

def create_sample_tasks():
    """Create some sample tasks."""
    pass`

		// Create test editor with Python content
		te := NewTestEditorWithContent(80, 24, pythonCode)
		te.buffer.FilePath = "test.py"

        // Initialize syntax highlighting (per-buffer)
        te.buffer.Highlighter = syntax.NewHighlighter()
        err := te.buffer.Highlighter.SetLanguageFromFile("test.py")
		if err != nil {
			t.Skipf("Skipping test: Python syntax highlighting not available: %v", err)
		}
        te.buffer.Highlighter.ParseFull([]byte(te.buffer.Rope.String()))

		// Force render to ensure syntax highlighting is applied
		te.render()

		// Get the expected keyword style from the theme
		expectedKeywordStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxKeyword)

		// Find the "def" keyword on line 2 (0-indexed)
		initialDefLineNum := 2
		defStyle := getCellStyle(te, initialDefLineNum, "def")
		if defStyle == nil {
			t.Fatal("Could not find 'def' keyword on line 2")
		}

		// Verify "def" has keyword style before making changes
		defFg, _, _ := defStyle.Decompose()
		expectedFg, _, _ := expectedKeywordStyle.Decompose()

		if defFg != expectedFg {
			t.Logf("Before typing 'class': def has color %v, expected %v", defFg, expectedFg)
		} else {
			t.Logf("Before typing 'class': def has correct color %v", defFg)
		}

		// Navigate to line 1 (the empty line above "def create_sample_tasks()")
		te.SetCursorPosition(0, 1)

		// Enter insert mode and type "class"
		te.ExecuteCommand("enter-insert-mode")

		// Type "class"
		for _, ch := range "class" {
			te.InsertChar(ch)
		}

		// Exit insert mode
		te.PressKey(KeyEscape, 0)

		// Force render after the edit
		te.render()

		// Debug: Print buffer content after typing "class"
		t.Logf("Buffer content after typing 'class':")
		lines := strings.Split(te.GetBufferContent(), "\n")
		for i, line := range lines {
			t.Logf("Line %d: %s", i, line)
		}

		// Find which line now contains "class" and "def"
		var classLineNum, defLineNum int = -1, -1

		for i, line := range lines {
			if strings.Contains(line, "class") && classLineNum == -1 {
				classLineNum = i
			}
			if strings.Contains(line, "def create_sample_tasks") && defLineNum == -1 {
				defLineNum = i
			}
		}

		if classLineNum == -1 {
			t.Fatal("Could not find 'class' keyword after typing")
		}
		if defLineNum == -1 {
			t.Fatal("Could not find 'def' keyword after typing 'class'")
		}

		t.Logf("Found 'class' on line %d, 'def' on line %d", classLineNum, defLineNum)

		// Check that "class" is highlighted correctly
		classStyle := getCellStyle(te, classLineNum, "class")
		if classStyle != nil {
			classFg, _, _ := classStyle.Decompose()
			if classFg != expectedFg {
				t.Errorf("'class' keyword has wrong color: got %v, expected %v", classFg, expectedFg)
			} else {
				t.Logf("'class' keyword has correct color: %v", classFg)
			}
		}

		// Debug: Check what tokens the highlighter has for the def line
		defIter := buffer.NewBufferIteratorAtLine(te.buffer, defLineNum)
		defLineStart := defIter.Position()
		defLineEnd := defIter.LineEnd()
		defLineContent := defIter.Line()
        defTokens := te.buffer.Highlighter.GetTokensForLine(defLineStart, defLineEnd)
		t.Logf("Def line content: %q", defLineContent)
		t.Logf("Tokens for def line (line %d, range %d-%d): %+v", defLineNum, defLineStart, defLineEnd, defTokens)

		// Check what the tree looks like
        tree := te.buffer.Highlighter.GetTree()
		if tree != nil {
			t.Logf("Tree root node type: %s", tree.RootNode().Type())
			t.Logf("Tree root node range: %d-%d", tree.RootNode().StartByte(), tree.RootNode().EndByte())
			t.Logf("Total buffer size: %d", len(te.GetBufferContent()))
		}

		// Now check that "def" still has the correct color
		defStyleAfter := getCellStyle(te, defLineNum, "def")
		if defStyleAfter == nil {
			t.Fatal("Could not find 'def' keyword after typing 'class'")
		}

		defFgAfter, _, _ := defStyleAfter.Decompose()

		if defFgAfter != expectedFg {
			t.Errorf("BUG REPRODUCED: 'def' keyword lost its color after typing 'class' above it. Got %v, expected %v", defFgAfter, expectedFg)

			// Try manually calling updateSyntaxHighlighting and re-render to see if that fixes it
			t.Log("Testing if manual updateSyntaxHighlighting fixes the issue...")
			te.updateSyntaxHighlighting()
			te.render()

			defStyleAfterUpdate := getCellStyle(te, defLineNum, "def")
			if defStyleAfterUpdate != nil {
				defFgAfterUpdate, _, _ := defStyleAfterUpdate.Decompose()
				if defFgAfterUpdate == expectedFg {
					t.Log("SUCCESS: Manual updateSyntaxHighlighting fixes the issue")
				} else {
					t.Logf("FAILURE: Even manual updateSyntaxHighlighting doesn't fix it. Got %v, expected %v", defFgAfterUpdate, expectedFg)

					// Check tokens after manual update
            defTokensAfterUpdate := te.buffer.Highlighter.GetTokensForLine(defLineStart, defLineEnd)
					t.Logf("Tokens for def line after manual update: %+v", defTokensAfterUpdate)
				}
			}
		} else {
			t.Logf("'def' keyword still has correct color after typing 'class': %v", defFgAfter)
		}

		// The bug is that def becomes white (normal text color) instead of keeping keyword color
		normalTextStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxText)
		normalFg, _, _ := normalTextStyle.Decompose()

		if defFgAfter == normalFg {
			t.Errorf("BUG CONFIRMED: 'def' keyword has normal text color (%v) instead of keyword color (%v)", normalFg, expectedFg)
		}
	})

	t.Run("debug tokens for simple python code", func(t *testing.T) {
		// Test with the exact same content as a fresh parse to see if the issue is with the content
		freshCode := `#!/usr/bin/env python3
class SomeClass:
def create_sample_tasks():
    """Create some sample tasks."""
    pass`

		// Create a fresh editor with this content
		te := NewTestEditorWithContent(80, 24, freshCode)
		te.buffer.FilePath = "test.py"

        // Initialize syntax highlighting (per-buffer)
        te.buffer.Highlighter = syntax.NewHighlighter()
        err := te.buffer.Highlighter.SetLanguageFromFile("test.py")
		if err != nil {
			t.Skipf("Skipping test: Python syntax highlighting not available: %v", err)
		}
        te.buffer.Highlighter.ParseFull([]byte(freshCode))

		// Force render
		te.render()

		// Check tokens for the def line (line 2)
		defIter := buffer.NewBufferIteratorAtLine(te.buffer, 2)
		defLineStart := defIter.Position()
		defLineEnd := defIter.LineEnd()
		defLineContent := defIter.Line()
        defTokens := te.buffer.Highlighter.GetTokensForLine(defLineStart, defLineEnd)

		t.Logf("Fresh parse - def line content: %q", defLineContent)
		t.Logf("Fresh parse - tokens for def line: %+v", defTokens)

		// Now let's check if the def gets highlighted correctly in this fresh parse
		defStyle := getCellStyle(te, 2, "def")
		expectedKeywordStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxKeyword)
		expectedFg, _, _ := expectedKeywordStyle.Decompose()

		if defStyle != nil {
			defFg, _, _ := defStyle.Decompose()
			t.Logf("Fresh parse - def color: %v, expected: %v", defFg, expectedFg)
			if defFg == expectedFg {
				t.Log("Fresh parse: def has correct color - so the issue is with incremental updates")
			} else {
				t.Log("Fresh parse: def has wrong color - the issue is with the Python content itself")
			}
		}
	})
}
