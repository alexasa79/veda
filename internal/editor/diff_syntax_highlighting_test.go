package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/syntax"
)

func TestDiffViewSyntaxHighlighting(t *testing.T) {
	// Create a test editor
	te := NewTestEditorWithContent(80, 24, "")

	// Go code for testing syntax highlighting
	goCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`

	// Set up the main buffer with Go content
	mainBuffer := buffer.NewBuffer(goCode, "test.go", config.IndentTabs, 4)
	te.SetCurrentBuffer(mainBuffer)

	// Initialize syntax highlighting for the original buffer
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	err := te.CurrentBuffer().Highlighter.SetLanguageFromFile("test.go")
	if err != nil {
		t.Skipf("Go syntax highlighting not available: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(goCode))

	// Modified version for diff
	modifiedCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, Diff View!")
	// Added comment
}`

	// Simulate starting diff view
	te.startDiffSessionForCurrentBuffer(goCode) // Use original as "previous"

	// Create diff buffers manually (simulating what StartDiffViewForCurrentBuffer does)
	diffResult := computeDiff(goCode, modifiedCode)
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)

	// Create HEAD buffer
	headBufferName := "test.go@HEAD"
	headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, config.IndentTabs, 4)
	headBuffer.IsReadOnly = true
	te.buffers[headBufferName] = headBuffer

	// Create and set highlighter for HEAD buffer
	headHighlighter := syntax.NewHighlighter()
	err = headHighlighter.SetLanguageFromFile("test.go")
	if err != nil {
		t.Skipf("Go syntax highlighting not available: %v", err)
	}
	headHighlighter.ParseFull([]byte(alignedHeadContent))
	headBuffer.Highlighter = headHighlighter

	// Create current buffer
	currentBufferName := "test.go@current"
	currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, config.IndentTabs, 4)
	te.buffers[currentBufferName] = currentBuffer

	// Create and set highlighter for current buffer
	currentHighlighter := syntax.NewHighlighter()
	err = currentHighlighter.SetLanguageFromFile("test.go")
	if err != nil {
		t.Skipf("Go syntax highlighting not available: %v", err)
	}
	currentHighlighter.ParseFull([]byte(alignedCurrentContent))
	currentBuffer.Highlighter = currentHighlighter

	// Test that HEAD buffer has syntax highlighting
	t.Run("HEAD buffer has highlighter", func(t *testing.T) {
		if headBuffer.Highlighter == nil {
			t.Error("HEAD buffer should have a syntax highlighter")
		}
	})

	// Test that current buffer has syntax highlighting
	t.Run("current buffer has highlighter", func(t *testing.T) {
		if currentBuffer.Highlighter == nil {
			t.Error("current buffer should have a syntax highlighter")
		}
	})

	// Test that keywords are highlighted in HEAD buffer
	t.Run("HEAD buffer highlights keywords", func(t *testing.T) {
		// Check for "package" keyword on line 1
		tokens := headBuffer.Highlighter.GetTokensForLine(0, len("package main"))
		if len(tokens) == 0 {
			t.Error("Expected syntax tokens for 'package' keyword in HEAD buffer")
			return
		}

		// Find the token for "package"
		foundKeyword := false
		for _, token := range tokens {
			if strings.Contains("package", string([]byte(alignedHeadContent)[token.Start:token.End])) {
				if token.TokenType == syntax.TokenTypeKeyword {
					foundKeyword = true
					break
				}
			}
		}

		if !foundKeyword {
			t.Error("'package' should be highlighted as a keyword in HEAD buffer")
		}
	})

	// Test that keywords are highlighted in current buffer
	t.Run("current buffer highlights keywords", func(t *testing.T) {
		// Simply verify that the highlighter exists and is working
		// Check for any keyword token in the entire content
		allTokens := currentBuffer.Highlighter.GetTokensForLine(0, len(alignedCurrentContent))

		foundAnyKeyword := false
		var keywordsSeen []string
		for _, token := range allTokens {
			if token.TokenType == syntax.TokenTypeKeyword {
				tokenText := string([]byte(alignedCurrentContent)[token.Start:token.End])
				keywordsSeen = append(keywordsSeen, tokenText)
				foundAnyKeyword = true
			}
		}

		if !foundAnyKeyword {
			t.Errorf("Expected to find at least one keyword in current buffer. Content length: %d, tokens found: %d",
				len(alignedCurrentContent), len(allTokens))
		} else {
			// Verify we found expected Go keywords
			expectedKeywords := map[string]bool{
				"package": false,
				"import":  false,
				"func":    false,
			}

			for _, kw := range keywordsSeen {
				if _, expected := expectedKeywords[kw]; expected {
					expectedKeywords[kw] = true
				}
			}

			for kw, found := range expectedKeywords {
				if !found {
					t.Errorf("Expected keyword '%s' was not highlighted", kw)
				}
			}
		}
	})

	// Test that strings are highlighted
	t.Run("strings are highlighted in diff buffers", func(t *testing.T) {
		// Look for the "Hello, World!" string in HEAD buffer
		content := alignedHeadContent
		helloIndex := strings.Index(content, `"Hello, World!"`)
		if helloIndex >= 0 {
			tokens := headBuffer.Highlighter.GetTokensForLine(helloIndex, helloIndex+len(`"Hello, World!"`))

			foundString := false
			for _, token := range tokens {
				if token.TokenType == syntax.TokenTypeString {
					foundString = true
					break
				}
			}

			if !foundString {
				t.Error("String literals should be highlighted in HEAD buffer")
			}
		}

		// Look for the "Hello, Diff View!" string in current buffer
		content = alignedCurrentContent
		diffIndex := strings.Index(content, `"Hello, Diff View!"`)
		if diffIndex >= 0 {
			tokens := currentBuffer.Highlighter.GetTokensForLine(diffIndex, diffIndex+len(`"Hello, Diff View!"`))

			foundString := false
			for _, token := range tokens {
				if token.TokenType == syntax.TokenTypeString {
					foundString = true
					break
				}
			}

			if !foundString {
				t.Error("String literals should be highlighted in current buffer")
			}
		}
	})

	// Test that comments are highlighted
	t.Run("comments are highlighted in current buffer", func(t *testing.T) {
		content := alignedCurrentContent
		commentIndex := strings.Index(content, "// Added comment")
		if commentIndex >= 0 {
			tokens := currentBuffer.Highlighter.GetTokensForLine(commentIndex, commentIndex+len("// Added comment"))

			foundComment := false
			for _, token := range tokens {
				if token.TokenType == syntax.TokenTypeComment {
					foundComment = true
					break
				}
			}

			if !foundComment {
				t.Error("Comments should be highlighted in current buffer")
			}
		}
	})
}
