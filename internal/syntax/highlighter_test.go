package syntax

import (
	"strings"
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

func TestMarkdownFencedCodeHighlighting(t *testing.T) {
	t.Run("supported language", func(t *testing.T) {
		source := "```c\nprintf(\"Hello world\\n\");\n```"
		h := NewHighlighter()
		if err := h.SetLanguage("markdown"); err != nil {
			t.Fatal(err)
		}
		if err := h.ParseFull([]byte(source)); err != nil {
			t.Fatal(err)
		}

		lineStart := strings.Index(source, "printf")
		lineEnd := lineStart + len(`printf("Hello world\n");`)
		tokens := h.GetTokensForLine(lineStart, lineEnd)
		if !lineHasToken(source[lineStart:lineEnd], tokens, "printf", TokenTypeFunction) {
			t.Fatalf("expected C function highlighting, got %v", tokens)
		}
	})

	for _, test := range []struct {
		name  string
		fence string
	}{
		{name: "no language", fence: "```"},
		{name: "unsupported language", fence: "```rust"},
		{name: "ambiguous label", fence: "```c extra"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := test.fence + "\nprintf(\"Hello world\\n\");\n```"
			h := NewHighlighter()
			if err := h.SetLanguage("markdown"); err != nil {
				t.Fatal(err)
			}
			if err := h.ParseFull([]byte(source)); err != nil {
				t.Fatal(err)
			}

			if len(h.markdownCodeBlocks) != 0 {
				t.Fatalf("expected generic Markdown highlighting for %q", test.fence)
			}
		})
	}
}

func TestMarkdownFenceSupportsEveryHighlightedLanguage(t *testing.T) {
	// Exact SetLanguage names become Markdown fence names automatically. This
	// table records every language currently highlighted by Veda except Markdown.
	languages := []struct {
		language  string
		code      string
		tokenText string
		tokenType TokenType
	}{
		{language: "javascript", code: "const value = 1;", tokenText: "const", tokenType: TokenTypeKeyword},
		{language: "typescript", code: "const value: number = 1;", tokenText: "const", tokenType: TokenTypeKeyword},
		{language: "tsx", code: "const view = <div />;", tokenText: "const", tokenType: TokenTypeKeyword},
		{language: "go", code: "func main() {}", tokenText: "func", tokenType: TokenTypeKeyword},
		{language: "c", code: "int main(void) {}", tokenText: "int", tokenType: TokenTypeType},
		{language: "cpp", code: "class Thing {};", tokenText: "class", tokenType: TokenTypeKeyword},
		{language: "fish", code: "if true; end", tokenText: "if", tokenType: TokenTypeKeyword},
		{language: "bash", code: "if true; then echo ok; fi", tokenText: "if", tokenType: TokenTypeKeyword},
		{language: "python", code: "def greet(): pass", tokenText: "def", tokenType: TokenTypePythonKeyword},
		{language: "make", code: "all:", tokenText: "all", tokenType: TokenTypeConstant},
		{language: "tlaplus", code: "VARIABLE value", tokenText: "VARIABLE", tokenType: TokenTypeKeyword},
		{language: "latex", code: `\section{Title}`, tokenText: `\section`, tokenType: TokenTypeKeyword},
	}

	for _, test := range languages {
		t.Run(test.language, func(t *testing.T) {
			source := "```" + test.language + "\n" + test.code + "\n```"
			h := NewHighlighter()
			if err := h.SetLanguage("markdown"); err != nil {
				t.Fatal(err)
			}
			if err := h.ParseFull([]byte(source)); err != nil {
				t.Fatal(err)
			}

			if len(h.markdownCodeBlocks) != 1 {
				t.Fatalf("expected %q Markdown fence to be supported", test.language)
			}
			if got := h.markdownCodeBlocks[0].highlighter.GetCurrentLanguage(); got != test.language {
				t.Fatalf("fence language = %q, want %q", got, test.language)
			}

			lineStart := strings.Index(source, test.code)
			lineEnd := lineStart + len(test.code)
			tokens := h.GetTokensForLine(lineStart, lineEnd)
			if !lineHasToken(test.code, tokens, test.tokenText, test.tokenType) {
				t.Fatalf("expected %q to have %s highlighting, got %v", test.tokenText, test.tokenType, tokens)
			}
		})
	}
}

func TestMarkdownFenceAliasesUseSupportedLanguages(t *testing.T) {
	for alias, language := range markdownFenceAliases {
		t.Run(alias, func(t *testing.T) {
			source := "```" + alias + "\ncode\n```"
			h := NewHighlighter()
			if err := h.SetLanguage("markdown"); err != nil {
				t.Fatal(err)
			}
			if err := h.ParseFull([]byte(source)); err != nil {
				t.Fatal(err)
			}

			if len(h.markdownCodeBlocks) != 1 {
				t.Fatalf("expected %q Markdown fence to be supported", alias)
			}
			if got := h.markdownCodeBlocks[0].highlighter.GetCurrentLanguage(); got != language {
				t.Fatalf("fence language = %q, want %q", got, language)
			}
		})
	}
}

func TestMarkdownFenceHighlightingUpdatesAfterEdit(t *testing.T) {
	source := "```\nprintf(\"Hello world\\n\");\n```"
	h := NewHighlighter()
	if err := h.SetLanguage("markdown"); err != nil {
		t.Fatal(err)
	}
	if err := h.ParseFull([]byte(source)); err != nil {
		t.Fatal(err)
	}
	if len(h.markdownCodeBlocks) != 0 {
		t.Fatal("unlabeled fence should use generic Markdown highlighting")
	}

	if err := h.ApplyEdit(3, "", "c"); err != nil {
		t.Fatal(err)
	}
	if len(h.markdownCodeBlocks) != 1 {
		t.Fatal("adding a supported fence label should enable language highlighting")
	}

	if err := h.ApplyEdit(3, "c", ""); err != nil {
		t.Fatal(err)
	}
	if len(h.markdownCodeBlocks) != 0 {
		t.Fatal("removing a fence label should restore generic Markdown highlighting")
	}
}

func lineHasToken(line string, tokens []Token, text string, tokenType TokenType) bool {
	for _, token := range tokens {
		if line[token.Start:token.End] == text && token.TokenType == tokenType {
			return true
		}
	}
	return false
}

func TestMarkdownRejectsIntrawordUnderscoreEmphasisAcrossLines(t *testing.T) {
	source := "1. RECORD_KIND_ALPHA_00 - this is a `sample_record_type_t` object.\n" +
		"2. RECORD_KIND_BETA_00 - this is a `sample_link_type_t` object."
	h := NewHighlighter()
	if err := h.SetLanguage("markdown"); err != nil {
		t.Fatal(err)
	}
	if err := h.ParseFull([]byte(source)); err != nil {
		t.Fatal(err)
	}

	lineEnd := strings.IndexByte(source, '\n')
	line := source[:lineEnd]
	tokens := h.GetTokensForLine(0, lineEnd)
	if !lineHasToken(line, tokens, "`sample_record_type_t`", TokenTypeMarkdownCode) {
		t.Fatalf("expected inline code highlighting, got %v", tokens)
	}
	for _, token := range tokens {
		if token.TokenType == TokenTypeMarkdownItalic {
			t.Fatalf("identifier underscore became emphasis: %q", line[token.Start:token.End])
		}
	}
}

func TestMarkdownAllowsUnderscoreEmphasisNextToPunctuation(t *testing.T) {
	source := "**********_YYY_***********"
	h := NewHighlighter()
	if err := h.SetLanguage("markdown"); err != nil {
		t.Fatal(err)
	}
	if err := h.ParseFull([]byte(source)); err != nil {
		t.Fatal(err)
	}

	node := findMarkdownNode(h.GetTree().RootNode(), []byte(source), "emphasis", "_YYY_")
	if node == nil {
		t.Fatalf("expected underscore emphasis node, tree is %s", h.GetTree().RootNode().String())
	}
	if !hasValidMarkdownEmphasisDelimiters(node, []byte(source)) {
		t.Fatal("punctuation-delimited underscore emphasis was rejected")
	}
}

func TestCommonMarkWhitespace(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{name: "space", r: ' ', want: true},
		{name: "tab", r: '\t', want: true},
		{name: "line feed", r: '\n', want: true},
		{name: "form feed", r: '\f', want: true},
		{name: "carriage return", r: '\r', want: true},
		{name: "non-breaking space", r: '\u00a0', want: true},
		{name: "vertical tab", r: '\v', want: false},
		{name: "next line", r: '\u0085', want: false},
		{name: "line separator", r: '\u2028', want: false},
		{name: "paragraph separator", r: '\u2029', want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isCommonMarkWhitespace(test.r); got != test.want {
				t.Fatalf("isCommonMarkWhitespace(%U) = %t, want %t", test.r, got, test.want)
			}
		})
	}
}

func findMarkdownNode(node *sitter.Node, source []byte, nodeType, text string) *sitter.Node {
	if node.Type() == nodeType && string(source[node.StartByte():node.EndByte()]) == text {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if found := findMarkdownNode(node.Child(i), source, nodeType, text); found != nil {
			return found
		}
	}
	return nil
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
