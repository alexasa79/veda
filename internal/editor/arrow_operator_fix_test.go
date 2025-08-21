package editor

import (
	"strings"
	"testing"
	"veda/internal/syntax"
	"veda/internal/ui"
)

func TestArrowOperatorFix(t *testing.T) {
	t.Run("arrow operator should have python_operator token type", func(t *testing.T) {
		// Test that the arrow operator now gets the correct token type
		h := syntax.NewHighlighter()
		err := h.SetLanguage("python")
		if err != nil {
			t.Skipf("Skipping test: Python syntax highlighting not available: %v", err)
		}

		code := `def func() -> List[Task]: pass`
		err = h.ParseFull([]byte(code))
		if err != nil {
			t.Fatalf("Failed to parse code: %v", err)
		}

		// Get tokens for the line
		tokens := h.GetTokensForLine(0, len(code))
		t.Logf("Tokens: %+v", tokens)

		// Find the arrow token
		var arrowToken *syntax.Token
		for _, token := range tokens {
			if token.Start == 11 && token.End == 13 { // Arrow should be at positions 11-13
				arrowToken = &token
				break
			}
		}

		if arrowToken == nil {
			t.Fatal("Arrow token not found at expected position")
		}

		if arrowToken.TokenType != syntax.TokenTypePythonOperator {
			t.Errorf("Expected arrow token type to be %s, got %s",
				syntax.TokenTypePythonOperator, arrowToken.TokenType)
		} else {
			t.Logf("SUCCESS: Arrow token has correct type: %s", arrowToken.TokenType)
		}
	})

	t.Run("arrow operator color should be consistent", func(t *testing.T) {
		// Test that arrows get consistent colors in different contexts
		testCases := []struct {
			name string
			code string
		}{
			{"simple function", "def func() -> int: pass"},
			{"function with class above", "class C:\n    pass\n\ndef func() -> int: pass"},
			{"complex type annotation", "def func() -> Dict[str, List[int]]: pass"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create test editor
				te := NewTestEditorWithContent(80, 24, tc.code)
				te.buffer.FilePath = "test.py"

				// Initialize syntax highlighting
				te.highlighter = syntax.NewHighlighter()
				err := te.highlighter.SetLanguageFromFile("test.py")
				if err != nil {
					t.Skipf("Skipping test: Python syntax highlighting not available: %v", err)
				}
				te.highlighter.ParseFull([]byte(tc.code))

				// Force render
				te.render()

				// Find the arrow operator and check its color
				arrowStyle := findArrowStyleInEditor(te)
				if arrowStyle == nil {
					t.Error("Could not find arrow operator style")
					return
				}

				// Check that it has the operator color
				expectedOperatorStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxOperator)
				expectedFg, _, _ := expectedOperatorStyle.Decompose()
				actualFg, _, _ := arrowStyle.Decompose()

				if actualFg == expectedFg {
					t.Logf("SUCCESS: Arrow has correct operator color: %v", actualFg)
				} else {
					t.Errorf("Arrow has wrong color: got %v, expected %v", actualFg, expectedFg)
				}
			})
		}
	})
}

// Helper function to find the arrow operator style in the editor
func findArrowStyleInEditor(te *TestEditor) *ui.Style {
	content := te.GetBufferContent()
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		if strings.Contains(line, "->") {
			return getCellStyle(te, lineNum, "->")
		}
	}
	return nil
}
