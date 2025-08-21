package editor

import (
	"strings"
	"testing"
)

func TestRewrapPreservesEmptyLines(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		cursorLine     int
		expectedResult string
	}{
		{
			name: "empty line between paragraphs",
			content: `This is a very long line that should be wrapped when we press alt+q to make it fit within the specified width.

This is another very long line that should also be wrapped but it's in a different paragraph separated by an empty line.`,
			cursorLine: 0,
			expectedResult: `This is a very long line that should be wrapped when we press alt+q to make it
fit within the specified width.

This is another very long line that should also be wrapped but it's in a different paragraph separated by an empty line.`,
		},
		{
			name: "multiple empty lines between paragraphs",
			content: `This is a very long line that should be wrapped when we press alt+q to make it fit within the specified width.


This is another very long line that should also be wrapped but it's in a different paragraph separated by multiple empty lines.`,
			cursorLine: 0,
			expectedResult: `This is a very long line that should be wrapped when we press alt+q to make it
fit within the specified width.


This is another very long line that should also be wrapped but it's in a different paragraph separated by multiple empty lines.`,
		},
		{
			name: "rewrap second paragraph",
			content: `First paragraph stays as is.

This is a very long line in the second paragraph that should be wrapped when we press alt+q to make it fit.`,
			cursorLine: 2,
			expectedResult: `First paragraph stays as is.

This is a very long line in the second paragraph that should be wrapped when we
press alt+q to make it fit.`,
		},
		{
			name: "paragraph with trailing empty line",
			content: `This is a very long line that should be wrapped when we press alt+q to make it fit within the specified width.

`,
			cursorLine: 0,
			expectedResult: `This is a very long line that should be wrapped when we press alt+q to make it
fit within the specified width.

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor with content
			te := NewTestEditorWithContent(120, 24, tt.content)

			// Move cursor to the specified line
			for i := 0; i < tt.cursorLine; i++ {
				te.ExecuteCommand("down")
			}

			// Execute rewrap
			te.ExecuteCommand("rewrap-paragraph")

			// Get the result
			result := te.GetBufferContent()

			// Check if empty lines are preserved
			if result != tt.expectedResult {
				t.Errorf("Empty lines not preserved.\nExpected:\n%s\n\nGot:\n%s",
					tt.expectedResult, result)

				// Show line-by-line comparison
				expectedLines := strings.Split(tt.expectedResult, "\n")
				gotLines := strings.Split(result, "\n")

				t.Errorf("\nLine by line comparison:")
				maxLines := len(expectedLines)
				if len(gotLines) > maxLines {
					maxLines = len(gotLines)
				}

				for i := 0; i < maxLines; i++ {
					var expectedLine, gotLine string
					if i < len(expectedLines) {
						expectedLine = expectedLines[i]
					}
					if i < len(gotLines) {
						gotLine = gotLines[i]
					}

					if expectedLine != gotLine {
						t.Errorf("Line %d differs:\nExpected: %q\nGot:      %q",
							i, expectedLine, gotLine)
					}
				}
			}
		})
	}
}

func TestRewrapParagraphBounds(t *testing.T) {
	// Test that rewrap only affects the current paragraph
	content := `First paragraph that is short.

This is a very long second paragraph that should be wrapped when we execute the rewrap command on it.

Third paragraph is also short.`

	te := NewTestEditorWithContent(120, 24, content)

	// Move to second paragraph
	te.ExecuteCommand("down")
	te.ExecuteCommand("down")

	// Rewrap second paragraph
	te.ExecuteCommand("rewrap-paragraph")

	result := te.GetBufferContent()

	// Check that first and third paragraphs are unchanged
	lines := strings.Split(result, "\n")
	if lines[0] != "First paragraph that is short." {
		t.Error("First paragraph was modified when it shouldn't have been")
	}

	if lines[len(lines)-1] != "Third paragraph is also short." {
		t.Error("Third paragraph was modified when it shouldn't have been")
	}

	// Check that empty lines are still there
	if lines[1] != "" {
		t.Error("Empty line after first paragraph was removed")
	}

	// Find the empty line before the third paragraph
	foundEmptyLine := false
	for i := 2; i < len(lines)-1; i++ {
		if lines[i] == "" && lines[i+1] == "Third paragraph is also short." {
			foundEmptyLine = true
			break
		}
	}

	if !foundEmptyLine {
		t.Error("Empty line before third paragraph was removed")
	}
}

func TestRewrapCLineCommentsStopsBeforeCode(t *testing.T) {
	content := `// This is a very long comment line that should be wrapped properly and remain within comment lines only
// Additional comment line continues here and should also be wrapped neatly
int x = 42;`

	te := NewTestEditorWithContent(100, 24, content)
	// Place cursor on the first comment line
	te.SetCursorPosition(0, 0)

	// Rewrap
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap failed: %v", err)
	}

	// Verify that the code line remains intact and comments are still comments
	lines := strings.Split(te.GetBufferContent(), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	if lines[len(lines)-1] != "int x = 42;" {
		t.Fatalf("code line modified: %q", lines[len(lines)-1])
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "//") || !strings.HasPrefix(strings.TrimSpace(lines[1]), "//") {
		t.Fatalf("rewrap broke comment prefixes: %q | %q", lines[0], lines[1])
	}
}

func TestRewrapCBlockCommentMultiLineStopsBeforeCode(t *testing.T) {
	content := `/*
 * This is a block comment that should be wrapped across multiple lines without affecting code below it.
 * It uses the conventional star prefix style.
 */
int y = x + 1;`

	te := NewTestEditorWithContent(100, 24, content)
	te.SetCursorPosition(0, 1) // inside the block

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap failed: %v", err)
	}

	lines := strings.Split(te.GetBufferContent(), "\n")
	// Find the closing marker line index (should still be before code)
	var closeIdx int = -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "*/" {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		t.Fatalf("did not find closing */ after rewrap")
	}
	if closeIdx+1 >= len(lines) || strings.TrimSpace(lines[closeIdx+1]) != "int y = x + 1;" {
		t.Fatalf("code line moved or modified: %q", strings.TrimSpace(lines[closeIdx+1]))
	}
}
