package syntax

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"veda/internal/theme"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	markdown_inline "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown-inline"
)

// TokenType represents different syntax elements
type TokenType string

const (
	TokenTypeText       TokenType = "text"
	TokenTypeKeyword    TokenType = "keyword"
	TokenTypeFunction   TokenType = "function"
	TokenTypeVariable   TokenType = "variable"
	TokenTypeConstant   TokenType = "constant"
	TokenTypeString     TokenType = "string"
	TokenTypeComment    TokenType = "comment"
	TokenTypeNumber     TokenType = "number"
	TokenTypeOperator   TokenType = "operator"
	TokenTypePunctuation TokenType = "punctuation"
	TokenTypeType       TokenType = "type"
	TokenTypeNamespace  TokenType = "namespace"
	TokenTypeAttribute  TokenType = "attribute"
	TokenTypeTag        TokenType = "tag"
	
	// Markdown-specific token types
	TokenTypeMarkdownH1        TokenType = "markdown_h1"
	TokenTypeMarkdownH2        TokenType = "markdown_h2"
	TokenTypeMarkdownH3        TokenType = "markdown_h3"
	TokenTypeMarkdownH4        TokenType = "markdown_h4"
	TokenTypeMarkdownH5        TokenType = "markdown_h5"
	TokenTypeMarkdownH6        TokenType = "markdown_h6"
	TokenTypeMarkdownBold      TokenType = "markdown_bold"
	TokenTypeMarkdownItalic    TokenType = "markdown_italic"
	TokenTypeMarkdownBoldItalic TokenType = "markdown_bold_italic"
	TokenTypeMarkdownCode      TokenType = "markdown_code"
	TokenTypeMarkdownCodeBlock TokenType = "markdown_code_block"
	TokenTypeMarkdownBlockquote TokenType = "markdown_blockquote"
	TokenTypeMarkdownLink      TokenType = "markdown_link"
	TokenTypeMarkdownLinkText  TokenType = "markdown_link_text"
	TokenTypeMarkdownLinkUrl   TokenType = "markdown_link_url"
	TokenTypeMarkdownImage     TokenType = "markdown_image"
	TokenTypeMarkdownList      TokenType = "markdown_list"
	TokenTypeMarkdownListMarker TokenType = "markdown_list_marker"
	TokenTypeMarkdownHorizontalRule TokenType = "markdown_horizontal_rule"
	TokenTypeMarkdownStrikethrough TokenType = "markdown_strikethrough"
	TokenTypeMarkdownTable     TokenType = "markdown_table"
	TokenTypeMarkdownTableBorder TokenType = "markdown_table_border"
)

// Token represents a syntax-highlighted token
type Token struct {
	Start     int
	End       int
	TokenType TokenType
}

// Highlighter manages syntax highlighting for different languages
type Highlighter struct {
	parser         *sitter.Parser
	currentLang    string
	currentTree    *sitter.Tree
	currentSource  []byte
	tokenCache     map[int][]Token  // Cache key: startByte
}

// NewHighlighter creates a new syntax highlighter
func NewHighlighter() *Highlighter {
	parser := sitter.NewParser()
	return &Highlighter{
		parser:     parser,
		tokenCache: make(map[int][]Token),
	}
}

// SetLanguageFromFile sets the language based on file extension
func (h *Highlighter) SetLanguageFromFile(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".go":
		h.parser.SetLanguage(golang.GetLanguage())
		h.currentLang = "go"
	case ".c", ".h":
		h.parser.SetLanguage(c.GetLanguage())
		h.currentLang = "c"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		h.parser.SetLanguage(cpp.GetLanguage())
		h.currentLang = "cpp"
	case ".md", ".markdown":
		h.parser.SetLanguage(markdown_inline.GetLanguage())
		h.currentLang = "markdown"
	default:
		// No language set for unknown extensions
		h.currentLang = ""
		h.currentTree = nil
		h.currentSource = nil
		h.tokenCache = make(map[int][]Token)
		return nil
	}

	return nil
}

// Parse parses the given source code
func (h *Highlighter) Parse(source []byte) error {
	if h.currentLang == "" {
		return nil // No language set, nothing to parse
	}

	oldTree := h.currentTree
	tree, err := h.parser.ParseCtx(context.Background(), oldTree, source)
	if err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	h.currentTree = tree
	h.currentSource = source
	// Clear the token cache when source changes
	h.tokenCache = make(map[int][]Token)
	return nil
}

// ParseFull forces a full reparse without using the old tree
func (h *Highlighter) ParseFull(source []byte) error {
	if h.currentLang == "" {
		return nil // No language set, nothing to parse
	}

	// Parse without old tree to force full reparse
	tree, err := h.parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	h.currentTree = tree
	h.currentSource = source
	// Clear the token cache when source changes
	h.tokenCache = make(map[int][]Token)
	return nil
}

// ClearCache clears the token cache
func (h *Highlighter) ClearCache() {
	h.tokenCache = make(map[int][]Token)
}

// GetTree returns the current syntax tree
func (h *Highlighter) GetTree() *sitter.Tree {
	return h.currentTree
}

// GetTokensForLine returns syntax tokens for a specific line
func (h *Highlighter) GetTokensForLine(lineStart, lineEnd int) []Token {
	if h.currentTree == nil || h.currentLang == "" || h.currentSource == nil {
		return nil
	}

	// Check cache first
	if cached, ok := h.tokenCache[lineStart]; ok {
		return cached
	}

	tokens := []Token{}
	root := h.currentTree.RootNode()

	// Query the tree for nodes in this line
	h.walkNode(root, lineStart, lineEnd, h.currentSource, &tokens)

	// Sort tokens by start position and merge overlapping
	tokens = h.mergeOverlappingTokens(tokens)

	// Post-process markdown tokens
	if h.currentLang == "markdown" {
		tokens = h.postProcessMarkdownTokens(tokens, lineStart, lineEnd, h.currentSource)
	}

	// Cache the result
	h.tokenCache[lineStart] = tokens

	return tokens
}

// GetTokensForString parses and returns syntax tokens for a single string
func (h *Highlighter) GetTokensForString(text string) []Token {
	if h.currentLang == "" {
		return nil
	}

	// Parse the string as a standalone piece of code
	source := []byte(text)
	tree, err := h.parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil
	}

	tokens := []Token{}
	root := tree.RootNode()

	// Walk the tree and collect tokens
	h.walkNode(root, 0, len(source), source, &tokens)

	// Sort tokens by start position and merge overlapping
	tokens = h.mergeOverlappingTokens(tokens)

	// Post-process markdown tokens
	if h.currentLang == "markdown" {
		tokens = h.postProcessMarkdownTokens(tokens, 0, len(source), source)
	}

	return tokens
}

// walkNode recursively walks the syntax tree and collects tokens
func (h *Highlighter) walkNode(node *sitter.Node, lineStart, lineEnd int, source []byte, tokens *[]Token) {
	nodeStart := int(node.StartByte())
	nodeEnd := int(node.EndByte())

	// Skip nodes that don't overlap with our line
	if nodeEnd <= lineStart || nodeStart >= lineEnd {
		return
	}

	// Check if this node has children
	childCount := int(node.ChildCount())
	
	// For markdown, we need to handle certain parent nodes as tokens
	if h.currentLang == "markdown" {
		nodeType := node.Type()
		// These markdown nodes should be tokenized even if they have children
		isMarkdownToken := nodeType == "emphasis" || nodeType == "strong_emphasis" || 
			nodeType == "code_span" || nodeType == "strikethrough" ||
			nodeType == "inline_link" || nodeType == "shortcut_link" ||
			nodeType == "fenced_code_block" || nodeType == "indented_code_block" ||
			nodeType == "link_destination"
		
		if isMarkdownToken {
			tokenType := h.mapNodeToTokenType(node, source)
			if tokenType != TokenTypeText {
				// Clip token to line boundaries
				tokenStart := nodeStart
				tokenEnd := nodeEnd
				if tokenStart < lineStart {
					tokenStart = lineStart
				}
				if tokenEnd > lineEnd {
					tokenEnd = lineEnd
				}

				*tokens = append(*tokens, Token{
					Start:     tokenStart - lineStart, // Make relative to line start
					End:       tokenEnd - lineStart,
					TokenType: tokenType,
				})
			}
			// For links, we need to continue processing children to get link text and URL
			if nodeType != "inline_link" && nodeType != "link_destination" {
				return // Don't process other children for non-link nodes
			}
			// For inline_link and link_destination, continue to process children
		}
	}
	
	// Check if this node should be tokenized
	// Some nodes like string_literal have children but should still be tokenized as a whole
	nodeType := node.Type()
	shouldTokenize := childCount == 0 || // Leaf nodes
		nodeType == "string_literal" || // C/C++ strings
		nodeType == "char_literal" || // C/C++ chars
		nodeType == "raw_string_literal" // C++ raw strings
	
	if shouldTokenize {
		// Map node type to token type
		tokenType := h.mapNodeToTokenType(node, source)
		if tokenType != TokenTypeText {
			// Clip token to line boundaries
			tokenStart := nodeStart
			tokenEnd := nodeEnd
			if tokenStart < lineStart {
				tokenStart = lineStart
			}
			if tokenEnd > lineEnd {
				tokenEnd = lineEnd
			}

			*tokens = append(*tokens, Token{
				Start:     tokenStart - lineStart, // Make relative to line start
				End:       tokenEnd - lineStart,
				TokenType: tokenType,
			})
		}
	}

	// Always recurse into children
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		h.walkNode(child, lineStart, lineEnd, source, tokens)
	}
}

// mapNodeToTokenType maps tree-sitter node types to our token types
func (h *Highlighter) mapNodeToTokenType(node *sitter.Node, source []byte) TokenType {
	nodeType := node.Type()

	switch h.currentLang {
	case "go":
		return h.mapGoNodeToTokenType(node, nodeType, source)
	case "c":
		return h.mapCNodeToTokenType(node, nodeType, source)
	case "cpp":
		return h.mapCppNodeToTokenType(node, nodeType, source)
	case "markdown":
		return h.mapMarkdownNodeToTokenType(node, nodeType, source)
	default:
		return TokenTypeText
	}
}

// mapGoNodeToTokenType maps Go-specific node types to token types
func (h *Highlighter) mapGoNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// Keywords
	case "package", "import", "func", "var", "const", "type",
		"if", "else", "for", "range", "switch", "case", "default",
		"return", "break", "continue", "goto", "fallthrough",
		"defer", "go", "select", "chan", "interface", "struct",
		"map", "make", "new", "nil":
		return TokenTypeKeyword

	// Comments
	case "comment":
		return TokenTypeComment

	// Strings
	case "interpreted_string_literal", "raw_string_literal", "rune_literal":
		return TokenTypeString

	// Numbers
	case "int_literal", "float_literal", "imaginary_literal":
		return TokenTypeNumber

	// Types
	case "type_identifier":
		return TokenTypeType

	// Functions
	case "identifier":
		// Check if this identifier is a function name
		parent := node.Parent()
		if parent != nil {
			parentType := parent.Type()
			if parentType == "function_declaration" && node == parent.ChildByFieldName("name") {
				return TokenTypeFunction
			}
			if parentType == "call_expression" && node == parent.ChildByFieldName("function") {
				return TokenTypeFunction
			}
			// Check if it's a method name
			if parentType == "method_declaration" && node == parent.ChildByFieldName("name") {
				return TokenTypeFunction
			}
		}
		return TokenTypeVariable

	// Field identifiers (method names in selector expressions like fmt.Println)
	case "field_identifier":
		parent := node.Parent()
		if parent != nil && parent.Type() == "selector_expression" {
			// Check if the selector expression is used as a function call
			grandparent := parent.Parent()
			if grandparent != nil && grandparent.Type() == "call_expression" &&
				parent == grandparent.ChildByFieldName("function") {
				return TokenTypeFunction
			}
		}
		return TokenTypeVariable

	// Punctuation and operators
	case "(", ")", "[", "]", "{", "}", ",", ";", ".":
		return TokenTypePunctuation

	case "+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>", "&^",
		"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", "&^=",
		"&&", "||", "<-", "++", "--",
		"==", "!=", "<", "<=", ">", ">=",
		"=", ":=",
		"!", "...", "~":
		return TokenTypeOperator

	// Package names
	case "package_identifier":
		return TokenTypeNamespace

	default:
		return TokenTypeText
	}
}

// mergeOverlappingTokens sorts and merges overlapping tokens
func (h *Highlighter) mergeOverlappingTokens(tokens []Token) []Token {
	if len(tokens) <= 1 {
		return tokens
	}

	// Simple insertion sort for small token lists
	for i := 1; i < len(tokens); i++ {
		j := i
		for j > 0 && tokens[j].Start < tokens[j-1].Start {
			tokens[j], tokens[j-1] = tokens[j-1], tokens[j]
			j--
		}
	}

	// Merge overlapping tokens, preferring more specific types
	merged := []Token{tokens[0]}
	for i := 1; i < len(tokens); i++ {
		last := &merged[len(merged)-1]
		curr := tokens[i]

		if curr.Start >= last.End {
			// No overlap
			merged = append(merged, curr)
		} else if curr.Start == last.Start && curr.End == last.End {
			// Same span, prefer more specific type
			if h.isMoreSpecific(curr.TokenType, last.TokenType) {
				*last = curr
			}
		} else if curr.Start >= last.Start && curr.End <= last.End {
			// Current is contained within last, keep last unless current is more specific
			if h.isMoreSpecific(curr.TokenType, last.TokenType) {
				// Split last token around current
				if curr.Start > last.Start {
					merged = append(merged, Token{
						Start:     last.Start,
						End:       curr.Start,
						TokenType: last.TokenType,
					})
				}
				merged = append(merged, curr)
				if curr.End < last.End {
					merged = append(merged, Token{
						Start:     curr.End,
						End:       last.End,
						TokenType: last.TokenType,
					})
				}
			}
		} else {
			// Other overlap cases - just keep the more specific one
			if h.isMoreSpecific(curr.TokenType, last.TokenType) {
				merged[len(merged)-1] = curr
			}
		}
	}

	return merged
}

// mapCNodeToTokenType maps C-specific node types to token types
func (h *Highlighter) mapCNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// Keywords
	case "auto", "break", "case", "char", "const", "continue", "default",
		"do", "double", "else", "enum", "extern", "float", "for", "goto",
		"if", "int", "long", "register", "return", "short", "signed",
		"sizeof", "static", "struct", "switch", "typedef", "union",
		"unsigned", "void", "volatile", "while", "_Bool", "_Complex",
		"_Imaginary", "inline", "restrict":
		return TokenTypeKeyword

	// Comments
	case "comment":
		return TokenTypeComment

	// Strings
	case "string_literal", "char_literal":
		return TokenTypeString

	// Numbers
	case "number_literal":
		return TokenTypeNumber

	// Types and type identifiers
	case "type_identifier", "primitive_type", "sized_type_specifier":
		return TokenTypeType

	// Function and identifier handling
	case "identifier":
		parent := node.Parent()
		if parent != nil {
			parentType := parent.Type()
			// Function declaration name
			if parentType == "function_declarator" && node == parent.ChildByFieldName("declarator") {
				return TokenTypeFunction
			}
			// Function call
			if parentType == "call_expression" && node == parent.ChildByFieldName("function") {
				return TokenTypeFunction
			}
			// Type declaration
			if parentType == "type_definition" {
				return TokenTypeType
			}
		}
		return TokenTypeVariable

	// Field identifiers (struct member function calls like obj->method or obj.method)
	case "field_identifier":
		parent := node.Parent()
		if parent != nil && parent.Type() == "field_expression" {
			// Check if the field expression is used as a function call
			grandparent := parent.Parent()
			if grandparent != nil && grandparent.Type() == "call_expression" &&
				parent == grandparent.ChildByFieldName("function") {
				return TokenTypeFunction
			}
		}
		return TokenTypeVariable

	// Preprocessor directives - don't tokenize the whole line, let children be processed
	case "preproc_include", "preproc_def", "preproc_function_def",
		"preproc_call", "preproc_if", "preproc_ifdef", "preproc_else",
		"preproc_elif":
		return TokenTypeText
		
	// Preprocessor directive keywords
	case "#include", "#define", "#ifdef", "#ifndef", "#if", "#else", "#elif", "#endif", 
		"#undef", "preproc_directive":
		return TokenTypeAttribute

	// Punctuation
	case "(", ")", "[", "]", "{", "}", ",", ";", ".", "->":
		return TokenTypePunctuation

	// Operators
	case "+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>",
		"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=",
		"&&", "||", "++", "--",
		"==", "!=", "<", "<=", ">", ">=",
		"=", "!", "~", "?", ":":
		return TokenTypeOperator

	default:
		return TokenTypeText
	}
}

// mapCppNodeToTokenType maps C++-specific node types to token types
func (h *Highlighter) mapCppNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// C++ Keywords (includes C keywords)
	case "break", "case", "char", "const", "continue", "default",
		"do", "double", "else", "enum", "extern", "float", "for", "goto",
		"if", "int", "long", "register", "return", "short", "signed",
		"sizeof", "static", "struct", "switch", "typedef", "union",
		"unsigned", "void", "volatile", "while",
		// C++ specific keywords
		"alignas", "alignof", "and", "and_eq", "asm", "bitand", "bitor",
		"bool", "catch", "class", "compl", "concept", "const_cast",
		"consteval", "constexpr", "constinit", "delete",
		"dynamic_cast", "explicit", "export", "friend", "inline",
		"mutable", "namespace", "new", "noexcept", "not", "not_eq",
		"operator", "or", "or_eq", "private", "protected",
		"public", "reinterpret_cast", "requires", "static_assert",
		"static_cast", "template", "this", "thread_local", "throw",
		"try", "typeid", "typename", "using", "virtual",
		"wchar_t", "xor", "xor_eq", "co_await", "co_return", "co_yield":
		return TokenTypeKeyword

	// Comments
	case "comment":
		return TokenTypeComment

	// Strings
	case "string_literal", "char_literal", "raw_string_literal":
		return TokenTypeString

	// Numbers and boolean/null literals
	case "number_literal", "true", "false", "nullptr":
		return TokenTypeNumber

	// Types
	case "type_identifier", "primitive_type", "sized_type_specifier",
		"auto", "decltype":
		return TokenTypeType

	// Namespace
	case "namespace_identifier":
		return TokenTypeNamespace

	// Function and identifier handling
	case "identifier", "field_identifier":
		parent := node.Parent()
		if parent != nil {
			parentType := parent.Type()
			// Function declaration
			if parentType == "function_declarator" && node == parent.ChildByFieldName("declarator") {
				return TokenTypeFunction
			}
			// Method declaration
			if parentType == "function_definition" && node == parent.ChildByFieldName("declarator") {
				return TokenTypeFunction
			}
			// Function call
			if parentType == "call_expression" && node == parent.ChildByFieldName("function") {
				return TokenTypeFunction
			}
			// Constructor
			if parentType == "declaration" && node.Type() == "identifier" {
				// Check if it might be a constructor
				return TokenTypeFunction
			}
			// Class or struct name
			if parentType == "class_specifier" || parentType == "struct_specifier" {
				return TokenTypeType
			}
		}
		return TokenTypeVariable

	// Preprocessor directives - don't tokenize the whole line, let children be processed
	case "preproc_include", "preproc_def", "preproc_function_def",
		"preproc_call", "preproc_if", "preproc_ifdef", "preproc_else",
		"preproc_elif":
		return TokenTypeText
		
	// Preprocessor directive keywords
	case "#include", "#define", "#ifdef", "#ifndef", "#if", "#else", "#elif", "#endif", 
		"#undef", "preproc_directive":
		return TokenTypeAttribute

	// Punctuation
	case "(", ")", "[", "]", "{", "}", ",", ";", ".", "->", "::", "...":
		return TokenTypePunctuation

	// Operators
	case "+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>",
		"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=",
		"&&", "||", "++", "--",
		"==", "!=", "<", "<=", ">", ">=",
		"=", "!", "~", "?", ":", ".*", "->*":
		return TokenTypeOperator

	default:
		return TokenTypeText
	}
}

// mapMarkdownNodeToTokenType maps Markdown-specific node types to token types
func (h *Highlighter) mapMarkdownNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// Emphasis (italic and bold)
	case "emphasis":
		// Check if parent is emphasis (this would be the inner emphasis in ***text***)
		parent := node.Parent()
		if parent != nil && parent.Type() == "emphasis" {
			// This is bold italic
			return TokenTypeMarkdownBoldItalic
		}
		// Check if it contains strong_emphasis (***text***)
		hasStrongEmphasis := false
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "strong_emphasis" {
				hasStrongEmphasis = true
				break
			}
		}
		if hasStrongEmphasis {
			return TokenTypeMarkdownBoldItalic
		}
		return TokenTypeMarkdownItalic
	
	case "strong_emphasis":
		// If parent is emphasis, this is part of bold italic
		parent := node.Parent()
		if parent != nil && parent.Type() == "emphasis" {
			return TokenTypeMarkdownBoldItalic
		}
		return TokenTypeMarkdownBold
	
	// Code
	case "code_span":
		return TokenTypeMarkdownCode
	
	case "fenced_code_block", "indented_code_block":
		return TokenTypeMarkdownCodeBlock
	
	// Links and images  
	case "inline_link":
		return TokenTypeMarkdownLink
	
	case "shortcut_link":
		// Check if this is a checkbox pattern [ ] or [x]
		text := string(source[node.StartByte():node.EndByte()])
		if text == "[ ]" || text == "[x]" || text == "[X]" {
			// This is a checkbox, not a link
			return TokenTypeText
		}
		return TokenTypeMarkdownLink
	
	case "link_text":
		// Check if parent is a shortcut_link that's actually a checkbox
		parent := node.Parent()
		if parent != nil && parent.Type() == "shortcut_link" {
			parentText := string(source[parent.StartByte():parent.EndByte()])
			if parentText == "[ ]" || parentText == "[x]" || parentText == "[X]" {
				return TokenTypeText
			}
		}
		return TokenTypeMarkdownLinkText
		
	case "link_destination":
		return TokenTypeMarkdownLinkUrl
		
	case "image":
		return TokenTypeMarkdownImage
	
	// Lists - note: in markdown-inline, list markers are just "-", "+", "*" characters
	case "-", "+", "*":
		// Check if this is at the beginning of a line (list marker)
		if node.StartByte() == 0 || (node.StartByte() > 0 && source[node.StartByte()-1] == '\n') {
			return TokenTypeMarkdownListMarker
		}
		return TokenTypeText
	
	// Block quotes - ">" character at start of line
	case ">":
		// Check if this is at the beginning of a line
		if node.StartByte() == 0 || (node.StartByte() > 0 && source[node.StartByte()-1] == '\n') {
			return TokenTypeMarkdownBlockquote
		}
		return TokenTypeText
	
	// Headings - "#" characters at start of line
	case "#":
		// Check if this is at the beginning of a line
		if node.StartByte() == 0 || (node.StartByte() > 0 && source[node.StartByte()-1] == '\n') {
			// Count consecutive # to determine heading level
			count := 1
			pos := node.StartByte() + 1
			for pos < uint32(len(source)) && source[pos] == '#' {
				count++
				pos++
			}
			switch count {
			case 1:
				return TokenTypeMarkdownH1
			case 2:
				return TokenTypeMarkdownH2
			case 3:
				return TokenTypeMarkdownH3
			case 4:
				return TokenTypeMarkdownH4
			case 5:
				return TokenTypeMarkdownH5
			default:
				return TokenTypeMarkdownH6
			}
		}
		return TokenTypeText
	
	// Strikethrough
	case "strikethrough":
		return TokenTypeMarkdownStrikethrough
	
	// Tables
	case "table":
		return TokenTypeMarkdownTable
		
	case "|":
		return TokenTypeMarkdownTableBorder
	
	// Horizontal rule - would need to check for "---", "***", "___"
	case "thematic_break":
		return TokenTypeMarkdownHorizontalRule
	
	// Special characters that might be part of markdown syntax
	case "[", "]", "(", ")", "`", "~", "emphasis_delimiter", "code_span_delimiter":
		// These are handled by their parent nodes
		return TokenTypeText
	
	default:
		return TokenTypeText
	}
}

// postProcessMarkdownTokens extends header tokens to cover the entire line
func (h *Highlighter) postProcessMarkdownTokens(tokens []Token, lineStart, lineEnd int, source []byte) []Token {
	processed := []Token{}
	
	for _, token := range tokens {
		// Check if this is a header token at the start of the line
		if token.Start == 0 && (token.TokenType == TokenTypeMarkdownH1 || 
			token.TokenType == TokenTypeMarkdownH2 || 
			token.TokenType == TokenTypeMarkdownH3 || 
			token.TokenType == TokenTypeMarkdownH4 || 
			token.TokenType == TokenTypeMarkdownH5 || 
			token.TokenType == TokenTypeMarkdownH6) {
			
			// Extend the token to the end of the line
			extendedEnd := token.End
			for i := lineStart + token.End; i < lineEnd && i < len(source); i++ {
				if source[i] == '\n' {
					break
				}
				extendedEnd = i - lineStart + 1
			}
			
			processed = append(processed, Token{
				Start:     token.Start,
				End:       extendedEnd,
				TokenType: token.TokenType,
			})
		} else {
			processed = append(processed, token)
		}
	}
	
	return processed
}

// isMoreSpecific returns true if type1 is more specific than type2
func (h *Highlighter) isMoreSpecific(type1, type2 TokenType) bool {
	// Define specificity hierarchy
	specificity := map[TokenType]int{
		TokenTypeText:        0,
		TokenTypePunctuation: 1,
		TokenTypeOperator:    2,
		TokenTypeVariable:    3,
		TokenTypeNamespace:   4,
		TokenTypeType:        5,
		TokenTypeAttribute:   6,
		TokenTypeTag:         7,
		TokenTypeNumber:      8,
		TokenTypeConstant:    9,
		TokenTypeFunction:    10,
		TokenTypeKeyword:     11,
		TokenTypeString:      12,
		TokenTypeComment:     13,
		// Markdown types with higher specificity
		TokenTypeMarkdownList:           14,
		TokenTypeMarkdownListMarker:     15,
		TokenTypeMarkdownTable:          14,
		TokenTypeMarkdownTableBorder:    15,
		TokenTypeMarkdownHorizontalRule: 15,
		TokenTypeMarkdownBlockquote:     15,
		TokenTypeMarkdownLink:           16,
		TokenTypeMarkdownLinkText:       17,
		TokenTypeMarkdownLinkUrl:        17,
		TokenTypeMarkdownImage:          17,
		TokenTypeMarkdownCode:           18,
		TokenTypeMarkdownCodeBlock:      18,
		TokenTypeMarkdownStrikethrough:  19,
		TokenTypeMarkdownItalic:         20,
		TokenTypeMarkdownBold:           21,
		TokenTypeMarkdownBoldItalic:     22,
		TokenTypeMarkdownH6:             23,
		TokenTypeMarkdownH5:             24,
		TokenTypeMarkdownH4:             25,
		TokenTypeMarkdownH3:             26,
		TokenTypeMarkdownH2:             27,
		TokenTypeMarkdownH1:             28,
	}

	return specificity[type1] > specificity[type2]
}

// GetStyleForToken returns the style for a given token type based on the theme
func GetStyleForToken(tokenType TokenType, theme *theme.Theme) theme.StyleDef {
	switch tokenType {
	case TokenTypeKeyword:
		return theme.SyntaxKeyword
	case TokenTypeFunction:
		return theme.SyntaxFunction
	case TokenTypeVariable:
		return theme.SyntaxVariable
	case TokenTypeConstant:
		return theme.SyntaxConstant
	case TokenTypeString:
		return theme.SyntaxString
	case TokenTypeComment:
		return theme.SyntaxComment
	case TokenTypeNumber:
		return theme.SyntaxNumber
	case TokenTypeOperator:
		return theme.SyntaxOperator
	case TokenTypePunctuation:
		return theme.SyntaxPunctuation
	case TokenTypeType:
		return theme.SyntaxType
	case TokenTypeNamespace:
		return theme.SyntaxNamespace
	case TokenTypeAttribute:
		return theme.SyntaxAttribute
	case TokenTypeTag:
		return theme.SyntaxTag
	// Markdown token types
	case TokenTypeMarkdownH1:
		return theme.MarkdownH1
	case TokenTypeMarkdownH2:
		return theme.MarkdownH2
	case TokenTypeMarkdownH3:
		return theme.MarkdownH3
	case TokenTypeMarkdownH4:
		return theme.MarkdownH4
	case TokenTypeMarkdownH5:
		return theme.MarkdownH5
	case TokenTypeMarkdownH6:
		return theme.MarkdownH6
	case TokenTypeMarkdownBold:
		return theme.MarkdownBold
	case TokenTypeMarkdownItalic:
		return theme.MarkdownItalic
	case TokenTypeMarkdownBoldItalic:
		return theme.MarkdownBoldItalic
	case TokenTypeMarkdownCode:
		return theme.MarkdownCode
	case TokenTypeMarkdownCodeBlock:
		return theme.MarkdownCodeBlock
	case TokenTypeMarkdownBlockquote:
		return theme.MarkdownBlockquote
	case TokenTypeMarkdownLink:
		return theme.MarkdownLink
	case TokenTypeMarkdownLinkText:
		return theme.MarkdownLinkText
	case TokenTypeMarkdownLinkUrl:
		return theme.MarkdownLinkUrl
	case TokenTypeMarkdownImage:
		return theme.MarkdownImage
	case TokenTypeMarkdownList:
		return theme.MarkdownList
	case TokenTypeMarkdownListMarker:
		return theme.MarkdownListMarker
	case TokenTypeMarkdownHorizontalRule:
		return theme.MarkdownHorizontalRule
	case TokenTypeMarkdownStrikethrough:
		return theme.MarkdownStrikethrough
	case TokenTypeMarkdownTable:
		return theme.MarkdownTable
	case TokenTypeMarkdownTableBorder:
		return theme.MarkdownTableBorder
	case TokenTypeText:
		fallthrough
	default:
		return theme.SyntaxText
	}
}