package syntax

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func TestMarkdownHighlighting(t *testing.T) {
	highlighter := NewHighlighter()

	// Set language to markdown
	err := highlighter.SetLanguageFromFile("test.md")
	if err != nil {
		t.Fatalf("Failed to set markdown language: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected []struct {
			text      string
			tokenType TokenType
		}
	}{
		{
			name:  "Bold text",
			input: "This is **bold** text",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"**bold**", TokenTypeMarkdownBold},
			},
		},
		{
			name:  "Italic text",
			input: "This is *italic* text",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"*italic*", TokenTypeMarkdownItalic},
			},
		},
		{
			name:  "Bold italic text",
			input: "This is ***bold italic*** text",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"***bold italic***", TokenTypeMarkdownBoldItalic},
			},
		},
		{
			name:  "Inline code",
			input: "This is `code` text",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"`code`", TokenTypeMarkdownCode},
			},
		},
		{
			name:  "Link",
			input: "This is [a link](https://example.com)",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"a link", TokenTypeMarkdownLinkText},
				{"https://example.com", TokenTypeMarkdownLinkUrl},
			},
		},
		{
			name:  "Heading 1",
			input: "# Heading 1",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"# Heading 1", TokenTypeMarkdownH1},
			},
		},
		{
			name:  "Heading 2",
			input: "## Heading 2",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"## Heading 2", TokenTypeMarkdownH2},
			},
		},
		{
			name:  "List item",
			input: "- List item",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"-", TokenTypeMarkdownListMarker},
			},
		},
		{
			name:  "Strikethrough",
			input: "This is ~~strikethrough~~ text",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"~~strikethrough~~", TokenTypeMarkdownStrikethrough},
			},
		},
		{
			name:  "Code block",
			input: "```\ncode block\n```",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"```\ncode block\n```", TokenTypeMarkdownCode},
			},
		},
		{
			name:  "Checkbox unchecked",
			input: "- [ ] Todo item",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"-", TokenTypeMarkdownListMarker},
				// [ ] should NOT be highlighted as a link
			},
		},
		{
			name:  "Checkbox checked",
			input: "- [x] Done item",
			expected: []struct {
				text      string
				tokenType TokenType
			}{
				{"-", TokenTypeMarkdownListMarker},
				// [x] should NOT be highlighted as a link
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			err := highlighter.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			// Use GetTokensForString since we're testing full markdown parsing
			tokens := highlighter.GetTokensForString(tt.input)

			// Debug: print all tokens found
			t.Logf("Input: %q", tt.input)
			t.Logf("Found %d tokens:", len(tokens))
			for i, token := range tokens {
				text := tt.input[token.Start:token.End]
				t.Logf("  Token %d: %q [%d-%d] type=%s", i, text, token.Start, token.End, token.TokenType)
			}

			// Check if expected tokens are found
			for _, expected := range tt.expected {
				found := false
				for _, token := range tokens {
					tokenText := tt.input[token.Start:token.End]
					if tokenText == expected.text && token.TokenType == expected.tokenType {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected token %q with type %s not found", expected.text, expected.tokenType)
				}
			}
		})
	}
}

func TestMarkdownTreeStructure(t *testing.T) {
	highlighter := NewHighlighter()

	// Set language to markdown
	err := highlighter.SetLanguageFromFile("test.md")
	if err != nil {
		t.Fatalf("Failed to set markdown language: %v", err)
	}

	// Test input with various markdown elements
	input := `# Heading 1
## Heading 2
This is **bold** and *italic* text.
This is ***bold italic*** text.
Here's some ` + "`code`" + ` inline.
[Link text](https://example.com)
- List item
- [ ] Unchecked todo
- [x] Checked todo
> Blockquote
~~strikethrough~~`

	// Parse the input
	err = highlighter.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Get the tree
	tree := highlighter.GetTree()
	if tree == nil {
		t.Fatal("Tree is nil")
	}

	// Print tree structure for debugging
	t.Log("Tree structure:")
	printNodeTree(t, tree.RootNode(), []byte(input), 0)
}

func printNodeTree(t *testing.T, node *sitter.Node, source []byte, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	start := node.StartByte()
	end := node.EndByte()
	text := ""
	if end-start < 50 && end <= uint32(len(source)) {
		text = string(source[start:end])
	} else if end <= uint32(len(source)) {
		text = string(source[start:start+47]) + "..."
	}

	t.Logf("%s%s [%d-%d] %q", indent, node.Type(), start, end, text)

	for i := 0; i < int(node.ChildCount()); i++ {
		printNodeTree(t, node.Child(i), source, depth+1)
	}
}

// TestMarkdownLinkTokenization tests how markdown links are tokenized
func TestMarkdownLinkTokenization(t *testing.T) {
	highlighter := NewHighlighter()

	// Set language to markdown
	err := highlighter.SetLanguageFromFile("test.md")
	if err != nil {
		t.Fatalf("Failed to set markdown language: %v", err)
	}

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Simple link",
			input: "[link]",
		},
		{
			name:  "Link with URL",
			input: "[link](https://example.com)",
		},
		{
			name:  "Multiple links",
			input: "Here is [link1] and [link2]",
		},
		{
			name:  "Link in a sentence",
			input: "Click [here] to continue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			err := highlighter.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			// Get tokens for the line
			tokens := highlighter.GetTokensForLine(0, len(tt.input))

			// Print out all tokens for debugging
			t.Logf("Input: %q", tt.input)
			t.Logf("Tokens found:")
			for _, token := range tokens {
				if token.Start < len(tt.input) && token.End <= len(tt.input) {
					tokenText := tt.input[token.Start:token.End]
					t.Logf("  [%d:%d] %q -> %s", token.Start, token.End, tokenText, token.TokenType)
				}
			}

			// Also print the syntax tree for debugging
			if tree := highlighter.GetTree(); tree != nil {
				t.Logf("Syntax tree:")
				printNodeTree(t, tree.RootNode(), []byte(tt.input), 0)
			}
		})
	}
}

// TestMarkdownReparseConsistency tests that reparsing doesn't duplicate content
func TestMarkdownReparseConsistency(t *testing.T) {
	highlighter := NewHighlighter()

	// Set language to markdown
	err := highlighter.SetLanguageFromFile("test.md")
	if err != nil {
		t.Fatalf("Failed to set markdown language: %v", err)
	}

	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "Simple link",
			input: "[link]",
		},
		{
			name:  "Link being typed",
			input: "[link",
		},
		{
			name:  "Complete link with URL",
			input: "[link](https://example.com)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First parse
			err := highlighter.Parse([]byte(tc.input))
			if err != nil {
				t.Fatalf("First parse failed: %v", err)
			}

			firstTokens := highlighter.GetTokensForLine(0, len(tc.input))

			// Second parse (simulating what happens when exiting insert mode)
			err = highlighter.ParseFull([]byte(tc.input))
			if err != nil {
				t.Fatalf("Second parse failed: %v", err)
			}

			secondTokens := highlighter.GetTokensForLine(0, len(tc.input))

			// Compare tokens
			if len(firstTokens) != len(secondTokens) {
				t.Errorf("Token count changed after reparse: first=%d, second=%d",
					len(firstTokens), len(secondTokens))
				t.Logf("First tokens:")
				for _, tok := range firstTokens {
					t.Logf("  [%d:%d] %s", tok.Start, tok.End, tok.TokenType)
				}
				t.Logf("Second tokens:")
				for _, tok := range secondTokens {
					t.Logf("  [%d:%d] %s", tok.Start, tok.End, tok.TokenType)
				}
			}

			// Check that tokens are identical
			for i := 0; i < len(firstTokens) && i < len(secondTokens); i++ {
				if firstTokens[i].Start != secondTokens[i].Start ||
					firstTokens[i].End != secondTokens[i].End ||
					firstTokens[i].TokenType != secondTokens[i].TokenType {
					t.Errorf("Token %d differs after reparse: first=[%d:%d %s], second=[%d:%d %s]",
						i, firstTokens[i].Start, firstTokens[i].End, firstTokens[i].TokenType,
						secondTokens[i].Start, secondTokens[i].End, secondTokens[i].TokenType)
				}
			}
		})
	}
}

func TestSetLanguageFromObjectiveCFile(t *testing.T) {
	highlighter := NewHighlighter()

	if err := highlighter.SetLanguageFromFile("main.m"); err != nil {
		t.Fatalf("SetLanguageFromFile failed: %v", err)
	}

	if got := highlighter.GetCurrentLanguage(); got != "c" {
		t.Fatalf("expected language c for .m file, got %q", got)
	}
}
