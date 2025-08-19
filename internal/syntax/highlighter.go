package syntax

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"veda/internal/theme"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	markdown_inline "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown-inline"
	"github.com/smacker/go-tree-sitter/python"
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
	
	
	// Python-specific token types
	TokenTypePythonKeyword     TokenType = "python_keyword"
	TokenTypePythonBuiltin     TokenType = "python_builtin"
	TokenTypePythonDecorator   TokenType = "python_decorator"
	TokenTypePythonString      TokenType = "python_string"
	TokenTypePythonDocstring   TokenType = "python_docstring"
	TokenTypePythonFString     TokenType = "python_fstring"
	TokenTypePythonNumber      TokenType = "python_number"
	TokenTypePythonBoolean     TokenType = "python_boolean"
	TokenTypePythonNone        TokenType = "python_none"
	TokenTypePythonFunction    TokenType = "python_function"
	TokenTypePythonClass       TokenType = "python_class"
	TokenTypePythonSelf        TokenType = "python_self"
	TokenTypePythonComment     TokenType = "python_comment"
	TokenTypePythonOperator    TokenType = "python_operator"
	TokenTypePythonException   TokenType = "python_exception"
	TokenTypePythonImport      TokenType = "python_import"
	
	// Makefile-specific token types
	TokenTypeMakeTarget        TokenType = "make_target"
	TokenTypeMakeVariable      TokenType = "make_variable"
	TokenTypeMakeVariableRef   TokenType = "make_variable_ref"
	TokenTypeMakeComment       TokenType = "make_comment"
	TokenTypeMakeKeyword       TokenType = "make_keyword"
	TokenTypeMakeFunction      TokenType = "make_function"
	TokenTypeMakeString        TokenType = "make_string"
	TokenTypeMakeOperator      TokenType = "make_operator"
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
	
	// Check for shell scripts without extension (like "bashrc", "profile", etc)
	baseName := filepath.Base(filename)
	// Handle dotfiles specially - if the extension is the entire filename, treat it as no extension
	if ext == "" || ext == baseName {
		switch baseName {
		case "bashrc", ".bashrc", "bash_profile", ".bash_profile", 
			"profile", ".profile", "bash_aliases", ".bash_aliases",
			"bash_logout", ".bash_logout", "zshrc", ".zshrc":
			h.parser.SetLanguage(bash.GetLanguage())
			h.currentLang = "bash"
			return nil
		case "Makefile", "makefile", "GNUmakefile", "gnumakefile":
			h.currentLang = "makefile"
			h.currentTree = nil
			h.currentSource = nil
			h.tokenCache = make(map[int][]Token)
			return nil
		}
	}

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
	case ".sh", ".bash":
		h.parser.SetLanguage(bash.GetLanguage())
		h.currentLang = "bash"
	case ".py", ".pyw":
		h.parser.SetLanguage(python.GetLanguage())
		h.currentLang = "python"
	case ".mk", ".mak":
		h.currentLang = "makefile"
		h.currentTree = nil
		h.currentSource = nil
		h.tokenCache = make(map[int][]Token)
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

// GetCurrentLanguage returns the currently set language
func (h *Highlighter) GetCurrentLanguage() string {
	return h.currentLang
}

// SetLanguage sets the language directly by name
func (h *Highlighter) SetLanguage(language string) error {
	switch language {
	case "go":
		h.parser.SetLanguage(golang.GetLanguage())
		h.currentLang = "go"
	case "c":
		h.parser.SetLanguage(c.GetLanguage())
		h.currentLang = "c"
	case "cpp":
		h.parser.SetLanguage(cpp.GetLanguage())
		h.currentLang = "cpp"
	case "markdown":
		h.parser.SetLanguage(markdown_inline.GetLanguage())
		h.currentLang = "markdown"
	case "bash":
		h.parser.SetLanguage(bash.GetLanguage())
		h.currentLang = "bash"
	case "python":
		h.parser.SetLanguage(python.GetLanguage())
		h.currentLang = "python"
	case "makefile":
		h.currentLang = "makefile"
	default:
		// No language set for unknown languages
		h.currentLang = ""
		h.currentTree = nil
		h.currentSource = nil
		h.tokenCache = make(map[int][]Token)
		return fmt.Errorf("unknown language: %s", language)
	}
	
	// Clear the tree and cache when language changes
	h.currentTree = nil
	h.currentSource = nil
	h.tokenCache = make(map[int][]Token)
	
	return nil
}

// Parse parses the given source code
func (h *Highlighter) Parse(source []byte) error {
	if h.currentLang == "" {
		return nil // No language set, nothing to parse
	}

	// Special handling for makefile - no tree-sitter parsing needed
	if h.currentLang == "makefile" {
		h.currentTree = nil
		h.currentSource = source
		h.tokenCache = make(map[int][]Token)
		return nil
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

	// Special handling for makefile - no tree-sitter parsing needed
	if h.currentLang == "makefile" {
		h.currentTree = nil
		h.currentSource = source
		h.tokenCache = make(map[int][]Token)
		return nil
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
	if h.currentLang == "" || h.currentSource == nil {
		return nil
	}

	// Check cache first
	if cached, ok := h.tokenCache[lineStart]; ok {
		return cached
	}

	tokens := []Token{}
	
	// Special handling for makefile
	if h.currentLang == "makefile" {
		tokens = h.parseMakefileLine(lineStart, lineEnd, h.currentSource)
		// Cache the result
		h.tokenCache[lineStart] = tokens
		return tokens
	}

	if h.currentTree == nil {
		return nil
	}

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
		nodeType == "raw_string_literal" || // C++ raw strings
		nodeType == "interpreted_string_literal" || // Go strings
		nodeType == "rune_literal" || // Go runes
		nodeType == "command_name" || // Bash command names
		nodeType == "string" || // Python strings
		nodeType == "decorator" // Python decorators
	
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
	case "bash":
		return h.mapBashNodeToTokenType(node, nodeType, source)
	case "python":
		return h.mapPythonNodeToTokenType(node, nodeType, source)
	case "makefile":
		// This shouldn't be called for makefile since we use custom parsing
		return TokenTypeText
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

// mapBashNodeToTokenType maps Bash-specific node types to token types
func (h *Highlighter) mapBashNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// Keywords and control structures
	case "if", "then", "else", "elif", "fi",
		"case", "esac", "for", "while", "do", "done",
		"function", "return", "break", "continue", "exit",
		"source", "export", "local", "readonly", "declare",
		"unset", "shift", "eval", "exec", "in":
		return TokenTypeKeyword
		
	// Comments
	case "comment":
		return TokenTypeComment
		
	// Strings
	case "string", "raw_string", "ansi_c_string", "string_content":
		return TokenTypeString
		
	// Variables and expansions
	case "variable_name", "simple_expansion":
		return TokenTypeVariable
		
	// Command substitution and process substitution
	case "command_substitution", "process_substitution":
		return TokenTypeConstant
		
	// Numbers
	case "number":
		return TokenTypeNumber
		
	// Operators
	case "||", "&&", "|", "&", ";", ";;", ">&", "&>", ">", "<", ">>", "<<",
		"=", "+=", "-=", "*=", "/=", "%=", "==", "!=", "<=", ">=",
		"-eq", "-ne", "-lt", "-le", "-gt", "-ge", "test_operator":
		return TokenTypeOperator
		
	// Functions
	case "function_definition":
		// Mark the whole definition as a function
		return TokenTypeFunction
		
	// Command names (first word in simple commands)
	case "command_name":
		// For command_name nodes, check their child nodes
		if node.ChildCount() > 0 {
			child := node.Child(0)
			if child != nil && child.Type() == "word" {
				text := string(source[child.StartByte():child.EndByte()])
				// Check if it's a known builtin command
				switch text {
				// Check for keywords first
				case "if", "then", "else", "elif", "fi", "case", "esac",
					"for", "while", "do", "done", "function", "in",
					"return", "break", "continue", "exit", "export", "local",
					"readonly", "declare", "unset", "shift", "eval":
					return TokenTypeKeyword
				// Then check for builtins
				case "echo", "printf", "read", "cd", "pwd", "ls", "cp", "mv", "rm",
					"mkdir", "rmdir", "touch", "cat", "grep", "sed", "awk", "find",
					"test", "[", "[[", "]]", "true", "false", "type", "which",
					"alias", "unalias", "set", "source", ".",
					"exec", "trap", "wait",
					"jobs", "fg", "bg", "kill", "suspend", "logout", "times",
					"ulimit", "umask", "hash", "help", "history", "bind", "builtin",
					"caller", "command", "compgen", "complete", "dirs", "disown",
					"enable", "fc", "getopts", "let", "mapfile", "popd", "pushd",
					"readarray", "shopt", "typeset":
					return TokenTypeFunction
				default:
					return TokenTypeText
				}
			}
		}
		return TokenTypeText
		
	// Special parameters
	case "special_variable_name":
		return TokenTypeConstant
		
	// File descriptors and redirections
	case "file_descriptor", "file_redirect":
		return TokenTypeNumber
		
	default:
		// Check parent context for better classification
		parent := node.Parent()
		if parent != nil {
			parentType := parent.Type()
			
			// If this is a word node, check its context
			if nodeType == "word" {
				text := string(source[node.StartByte():node.EndByte()])
				
				// Check if it's a keyword that might not be caught by the grammar
				switch text {
				case "if", "then", "else", "elif", "fi", "case", "esac",
					"for", "while", "do", "done", "function", "in":
					return TokenTypeKeyword
				}
				
				// If this is inside a function definition, it might be the function name
				if parentType == "function_definition" {
					for i := 0; i < int(parent.ChildCount()); i++ {
						if parent.Child(i) == node {
							if i == 0 || (i == 1 && parent.Child(0).Type() == "function") {
								return TokenTypeFunction
							}
							break
						}
					}
				}
				
				// Check for variable assignment
				if parentType == "variable_assignment" {
					// First word in assignment is the variable name
					if parent.Child(0) == node {
						return TokenTypeVariable
					}
				}
			}
			
			// Special handling for expansion-like constructs
			if strings.HasPrefix(nodeType, "expansion") || nodeType == "substitution" {
				return TokenTypeConstant
			}
		}
		
		return TokenTypeText
	}
}

// parseMakefileLine parses a single line of makefile using pattern matching
func (h *Highlighter) parseMakefileLine(lineStart, lineEnd int, source []byte) []Token {
	if lineStart >= lineEnd || lineStart >= len(source) {
		return nil
	}
	
	tokens := []Token{}
	lineText := string(source[lineStart:lineEnd])
	
	// Skip empty lines
	if len(strings.TrimSpace(lineText)) == 0 {
		return nil
	}
	
	// Comments (lines starting with #)
	if strings.HasPrefix(strings.TrimSpace(lineText), "#") {
		tokens = append(tokens, Token{
			Start:     0,
			End:       len(lineText),
			TokenType: TokenTypeMakeComment,
		})
		return tokens
	}
	
	// Check if line starts with tab (recipe line)
	if strings.HasPrefix(lineText, "\t") {
		// This is a command/recipe line
		h.parseMakefileCommand(lineText, &tokens)
		return tokens
	}
	
	// Check for target definition (contains : not in a variable reference)
	if colonPos := h.findTargetColon(lineText); colonPos != -1 {
		// Target line
		h.parseMakefileTarget(lineText, colonPos, &tokens)
		return tokens
	}
	
	// Check for variable assignment
	if h.containsVariableAssignment(lineText) {
		h.parseMakefileVariable(lineText, &tokens)
		return tokens
	}
	
	// Parse as generic makefile content
	h.parseMakefileGeneric(lineText, &tokens)
	return tokens
}

// findTargetColon finds the colon that indicates a target (not inside variable references or assignment operators)
func (h *Highlighter) findTargetColon(line string) int {
	inVar := 0
	for i, char := range line {
		switch char {
		case '$':
			if i+1 < len(line) && (line[i+1] == '(' || line[i+1] == '{') {
				inVar++
			}
		case '(', '{':
			if i > 0 && line[i-1] == '$' {
				// Already counted above
				continue
			}
		case ')', '}':
			if inVar > 0 {
				inVar--
			}
		case ':':
			if inVar == 0 {
				// Check if this is part of an assignment operator like :=
				if i+1 < len(line) && line[i+1] == '=' {
					// This is :=, not a target separator
					continue
				}
				return i
			}
		}
	}
	return -1
}

// containsVariableAssignment checks if line contains variable assignment
func (h *Highlighter) containsVariableAssignment(line string) bool {
	// Look for =, :=, +=, ?=, etc.
	inVar := 0
	for i, char := range line {
		switch char {
		case '$':
			if i+1 < len(line) && (line[i+1] == '(' || line[i+1] == '{') {
				inVar++
			}
		case '(', '{':
			if i > 0 && line[i-1] == '$' {
				continue
			}
		case ')', '}':
			if inVar > 0 {
				inVar--
			}
		case '=':
			if inVar == 0 {
				return true
			}
		case ':':
			if inVar == 0 && i+1 < len(line) && line[i+1] == '=' {
				return true
			}
		case '+', '?':
			if inVar == 0 && i+1 < len(line) && line[i+1] == '=' {
				return true
			}
		}
	}
	return false
}

// parseMakefileTarget parses a target definition line
func (h *Highlighter) parseMakefileTarget(line string, colonPos int, tokens *[]Token) {
	// Target(s) before colon - find the actual start and end positions
	targetEnd := colonPos
	for targetEnd > 0 && line[targetEnd-1] == ' ' {
		targetEnd-- // Remove trailing spaces
	}
	
	targetStart := 0
	for targetStart < targetEnd && line[targetStart] == ' ' {
		targetStart++ // Remove leading spaces
	}
	
	if targetStart < targetEnd {
		*tokens = append(*tokens, Token{
			Start:     targetStart,
			End:       targetEnd,
			TokenType: TokenTypeMakeTarget,
		})
	}
	
	// Colon
	*tokens = append(*tokens, Token{
		Start:     colonPos,
		End:       colonPos + 1,
		TokenType: TokenTypeOperator,
	})
	
	// Dependencies after colon
	deps := strings.TrimSpace(line[colonPos+1:])
	if deps != "" {
		depStart := colonPos + 1
		for depStart < len(line) && line[depStart] == ' ' {
			depStart++
		}
		h.parseMakefileGeneric(deps, tokens, depStart)
	}
}

// parseMakefileCommand parses a command/recipe line
func (h *Highlighter) parseMakefileCommand(line string, tokens *[]Token) {
	// Skip leading tab
	cmdStart := 0
	for cmdStart < len(line) && (line[cmdStart] == '\t' || line[cmdStart] == ' ') {
		cmdStart++
	}
	
	if cmdStart >= len(line) {
		return
	}
	
	// Parse the command
	h.parseMakefileGeneric(line[cmdStart:], tokens, cmdStart)
}

// parseMakefileVariable parses a variable assignment line
func (h *Highlighter) parseMakefileVariable(line string, tokens *[]Token) {
	// Find the assignment operator
	inVar := 0
	assignPos := -1
	assignLen := 1
	
	for i, char := range line {
		switch char {
		case '$':
			if i+1 < len(line) && (line[i+1] == '(' || line[i+1] == '{') {
				inVar++
			}
		case '(', '{':
			if i > 0 && line[i-1] == '$' {
				continue
			}
		case ')', '}':
			if inVar > 0 {
				inVar--
			}
		case ':':
			if inVar == 0 && i+1 < len(line) && line[i+1] == '=' {
				assignPos = i
				assignLen = 2
				goto found
			}
		case '+', '?':
			if inVar == 0 && i+1 < len(line) && line[i+1] == '=' {
				assignPos = i
				assignLen = 2
				goto found
			}
		case '=':
			if inVar == 0 {
				assignPos = i
				assignLen = 1
				goto found
			}
		}
	}
	
found:
	if assignPos == -1 {
		return
	}
	
	// Variable name - find actual position
	varEnd := assignPos
	for varEnd > 0 && line[varEnd-1] == ' ' {
		varEnd-- // Remove trailing spaces
	}
	
	varStart := 0
	for varStart < varEnd && line[varStart] == ' ' {
		varStart++ // Remove leading spaces
	}
	
	if varStart < varEnd {
		*tokens = append(*tokens, Token{
			Start:     varStart,
			End:       varEnd,
			TokenType: TokenTypeMakeVariable,
		})
	}
	
	// Assignment operator
	*tokens = append(*tokens, Token{
		Start:     assignPos,
		End:       assignPos + assignLen,
		TokenType: TokenTypeMakeOperator,
	})
	
	// Variable value
	if assignPos+assignLen < len(line) {
		valueStart := assignPos + assignLen
		for valueStart < len(line) && line[valueStart] == ' ' {
			valueStart++
		}
		
		if valueStart < len(line) {
			value := line[valueStart:]
			h.parseMakefileGeneric(value, tokens, valueStart)
		}
	}
}

// parseMakefileGeneric parses generic makefile content, looking for variable references, strings, etc.
func (h *Highlighter) parseMakefileGeneric(text string, tokens *[]Token, offset ...int) {
	baseOffset := 0
	if len(offset) > 0 {
		baseOffset = offset[0]
	}
	
	i := 0
	for i < len(text) {
		char := text[i]
		
		switch char {
		case '$':
			// Variable reference
			if i+1 < len(text) {
				if text[i+1] == '(' || text[i+1] == '{' {
					// Find the matching closing bracket
					closer := ')'
					if text[i+1] == '{' {
						closer = '}'
					}
					
					end := i + 2
					depth := 1
					for end < len(text) && depth > 0 {
						if text[end] == text[i+1] {
							depth++
						} else if text[end] == byte(closer) {
							depth--
						}
						end++
					}
					
					*tokens = append(*tokens, Token{
						Start:     baseOffset + i,
						End:       baseOffset + end,
						TokenType: TokenTypeMakeVariableRef,
					})
					i = end
					continue
				} else if i+1 < len(text) && (text[i+1] >= 'A' && text[i+1] <= 'Z' || 
					text[i+1] >= 'a' && text[i+1] <= 'z' || text[i+1] == '_' ||
					text[i+1] == '@' || text[i+1] == '<' || text[i+1] == '^' || 
					text[i+1] == '?' || text[i+1] == '*' || text[i+1] == '+') {
					// Single character variable like $@, $<, etc.
					*tokens = append(*tokens, Token{
						Start:     baseOffset + i,
						End:       baseOffset + i + 2,
						TokenType: TokenTypeMakeVariableRef,
					})
					i += 2
					continue
				}
			}
		case '"':
			// Double quoted string
			end := i + 1
			for end < len(text) && text[end] != '"' {
				if text[end] == '\\' && end+1 < len(text) {
					end++ // Skip escaped character
				}
				end++
			}
			if end < len(text) {
				end++ // Include closing quote
			}
			*tokens = append(*tokens, Token{
				Start:     baseOffset + i,
				End:       baseOffset + end,
				TokenType: TokenTypeMakeString,
			})
			i = end
			continue
		case '\'':
			// Single quoted string
			end := i + 1
			for end < len(text) && text[end] != '\'' {
				if text[end] == '\\' && end+1 < len(text) {
					end++ // Skip escaped character
				}
				end++
			}
			if end < len(text) {
				end++ // Include closing quote
			}
			*tokens = append(*tokens, Token{
				Start:     baseOffset + i,
				End:       baseOffset + end,
				TokenType: TokenTypeMakeString,
			})
			i = end
			continue
		case '#':
			// Comment (from this point to end of text)
			*tokens = append(*tokens, Token{
				Start:     baseOffset + i,
				End:       baseOffset + len(text),
				TokenType: TokenTypeMakeComment,
			})
			return
		}
		i++
	}
	
	// If we have untokenized text segments, add them as generic text
	if strings.TrimSpace(text) != "" {
		// Find gaps between tokens and fill them with text tokens
		tokensBeforeSort := *tokens
		if len(tokensBeforeSort) == 0 {
			// No tokens found, make the entire text a token
			*tokens = append(*tokens, Token{
				Start:     baseOffset,
				End:       baseOffset + len(text),
				TokenType: TokenTypeText,
			})
		} else {
			// Sort tokens by position to find gaps
			for i := 1; i < len(tokensBeforeSort); i++ {
				key := tokensBeforeSort[i]
				j := i - 1
				for j >= 0 && tokensBeforeSort[j].Start > key.Start {
					tokensBeforeSort[j+1] = tokensBeforeSort[j]
					j--
				}
				tokensBeforeSort[j+1] = key
			}
			
			// Add text token for any gap at the beginning
			if tokensBeforeSort[0].Start > baseOffset {
				*tokens = append(*tokens, Token{
					Start:     baseOffset,
					End:       tokensBeforeSort[0].Start,
					TokenType: TokenTypeText,
				})
			}
			
			// Add text tokens for gaps between tokens
			for i := 0; i < len(tokensBeforeSort)-1; i++ {
				if tokensBeforeSort[i].End < tokensBeforeSort[i+1].Start {
					*tokens = append(*tokens, Token{
						Start:     tokensBeforeSort[i].End,
						End:       tokensBeforeSort[i+1].Start,
						TokenType: TokenTypeText,
					})
				}
			}
			
			// Add text token for any gap at the end
			if tokensBeforeSort[len(tokensBeforeSort)-1].End < baseOffset+len(text) {
				*tokens = append(*tokens, Token{
					Start:     tokensBeforeSort[len(tokensBeforeSort)-1].End,
					End:       baseOffset + len(text),
					TokenType: TokenTypeText,
				})
			}
		}
	}
}

// mapPythonNodeToTokenType maps Python-specific node types to token types
func (h *Highlighter) mapPythonNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// Keywords
	case "if", "elif", "else", "for", "while", "break", "continue",
		"def", "class", "return", "yield", "pass", "raise",
		"try", "except", "finally", "with", "as", "assert",
		"import", "from", "global", "nonlocal", "lambda",
		"and", "or", "not", "in", "is", "del", "async", "await":
		return TokenTypePythonKeyword
		
	// Comments
	case "comment":
		return TokenTypePythonComment
		
	// Strings
	case "string", "string_content", "string_start", "string_end":
		// Check if it's a docstring by looking at parent context
		parent := node.Parent()
		if parent != nil {
			grandparent := parent.Parent()
			if grandparent != nil {
				gpType := grandparent.Type()
				if gpType == "function_definition" || gpType == "class_definition" {
					// Check if this is the first statement
					if parent.Type() == "expression_statement" {
						return TokenTypePythonDocstring
					}
				}
			}
		}
		return TokenTypePythonString
		
	// F-strings
	case "interpolation":
		return TokenTypePythonFString
		
	// Numbers
	case "integer", "float":
		return TokenTypePythonNumber
		
	// Booleans
	case "true", "false":
		return TokenTypePythonBoolean
		
	// None
	case "none":
		return TokenTypePythonNone
		
	// Decorators
	case "decorator":
		return TokenTypePythonDecorator
		
	// Function names
	case "function_definition":
		// Get the function name from children
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				// This is the function name
				return TokenTypePythonFunction
			}
		}
		return TokenTypeText
		
	// Class names
	case "class_definition":
		// Get the class name from children
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				// This is the class name
				return TokenTypePythonClass
			}
		}
		return TokenTypeText
		
	// Identifiers - check for special cases
	case "identifier":
		text := string(source[node.StartByte():node.EndByte()])
		
		// self and cls
		if text == "self" || text == "cls" {
			return TokenTypePythonSelf
		}
		
		// Built-in functions
		switch text {
		case "print", "len", "range", "str", "int", "float", "bool",
			"list", "dict", "set", "tuple", "type", "isinstance",
			"hasattr", "getattr", "setattr", "delattr", "dir",
			"abs", "all", "any", "bin", "chr", "ord", "hex", "oct",
			"round", "max", "min", "sum", "sorted", "reversed",
			"enumerate", "zip", "map", "filter", "open", "input",
			"eval", "exec", "compile", "globals", "locals", "vars",
			"__import__", "super", "property", "staticmethod", "classmethod":
			return TokenTypePythonBuiltin
			
		// Common exceptions
		case "Exception", "ValueError", "TypeError", "KeyError",
			"IndexError", "AttributeError", "ImportError", "IOError",
			"OSError", "RuntimeError", "NotImplementedError",
			"StopIteration", "GeneratorExit", "KeyboardInterrupt",
			"SystemExit", "NameError", "UnboundLocalError",
			"ZeroDivisionError", "FileNotFoundError":
			return TokenTypePythonException
		}
		
		// Check if it's a function call
		parent := node.Parent()
		if parent != nil {
			if parent.Type() == "call" {
				// Check if this identifier is the function being called
				if parent.Child(0) == node {
					return TokenTypePythonFunction
				}
			} else if parent.Type() == "attribute" {
				// Check if the attribute is being called
				grandparent := parent.Parent()
				if grandparent != nil && grandparent.Type() == "call" && grandparent.Child(0) == parent {
					// This is a method call - check if this is the method name (last child of attribute)
					if parent.Child(int(parent.ChildCount())-1) == node {
						// Check if it's a builtin method
						switch text {
						case "print", "len", "range", "str", "int", "float", "bool",
							"list", "dict", "set", "tuple", "type", "isinstance",
							"hasattr", "getattr", "setattr", "delattr", "dir",
							"abs", "all", "any", "bin", "chr", "ord", "hex", "oct",
							"round", "max", "min", "sum", "sorted", "reversed",
							"enumerate", "zip", "map", "filter", "open", "input",
							"eval", "exec", "compile", "globals", "locals", "vars",
							"__import__", "super", "property", "staticmethod", "classmethod":
							return TokenTypePythonBuiltin
						default:
							return TokenTypePythonFunction
						}
					}
				}
			}
		}
		
		// Fallback: Check if this identifier is actually a Python keyword
		// This handles cases where Tree-sitter misclassifies keywords as identifiers due to syntax errors
		switch text {
		case "if", "elif", "else", "for", "while", "break", "continue",
			"def", "class", "return", "yield", "pass", "raise",
			"try", "except", "finally", "with", "as", "assert",
			"import", "from", "global", "nonlocal", "lambda",
			"and", "or", "not", "in", "is", "del", "async", "await":
			return TokenTypePythonKeyword
		}
		
		return TokenTypeVariable
		
	// Operators
	case "+", "-", "*", "/", "//", "%", "**",
		"=", "+=", "-=", "*=", "/=", "//=", "%=", "**=",
		"==", "!=", "<", ">", "<=", ">=",
		"&", "|", "^", "~", "<<", ">>",
		"&=", "|=", "^=", "<<=", ">>=",
		"->":  // Type annotation arrow
		return TokenTypePythonOperator
		
	// Import-related
	case "import_statement", "import_from_statement":
		return TokenTypePythonImport
		
	// Punctuation
	case "(", ")", "[", "]", "{", "}", ",", ";", ".", ":", "...":
		return TokenTypePunctuation
		
	default:
		return TokenTypeText
	}
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
		// Bash types with appropriate specificity
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
	// Bash token types
	// Python token types
	case TokenTypePythonKeyword:
		return theme.SyntaxKeyword
	case TokenTypePythonBuiltin:
		return theme.SyntaxFunction
	case TokenTypePythonDecorator:
		return theme.SyntaxAttribute
	case TokenTypePythonString:
		return theme.SyntaxString
	case TokenTypePythonDocstring:
		return theme.SyntaxComment
	case TokenTypePythonFString:
		return theme.SyntaxString
	case TokenTypePythonNumber:
		return theme.SyntaxNumber
	case TokenTypePythonBoolean:
		return theme.SyntaxConstant
	case TokenTypePythonNone:
		return theme.SyntaxConstant
	case TokenTypePythonFunction:
		return theme.SyntaxFunction
	case TokenTypePythonClass:
		return theme.SyntaxType
	case TokenTypePythonSelf:
		return theme.SyntaxVariable
	case TokenTypePythonComment:
		return theme.SyntaxComment
	case TokenTypePythonOperator:
		return theme.SyntaxOperator
	case TokenTypePythonException:
		return theme.SyntaxType
	case TokenTypePythonImport:
		return theme.SyntaxKeyword
	// Makefile token types
	case TokenTypeMakeTarget:
		return theme.SyntaxFunction
	case TokenTypeMakeVariable:
		return theme.SyntaxVariable
	case TokenTypeMakeVariableRef:
		return theme.SyntaxConstant
	case TokenTypeMakeComment:
		return theme.SyntaxComment
	case TokenTypeMakeKeyword:
		return theme.SyntaxKeyword
	case TokenTypeMakeFunction:
		return theme.SyntaxFunction
	case TokenTypeMakeString:
		return theme.SyntaxString
	case TokenTypeMakeOperator:
		return theme.SyntaxOperator
	case TokenTypeText:
		fallthrough
	default:
		return theme.SyntaxText
	}
}