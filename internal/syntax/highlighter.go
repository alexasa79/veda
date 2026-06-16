package syntax

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	tsfish "veda/internal/syntax/treesitter/fish"
	tslatex "veda/internal/syntax/treesitter/latex"
	tsmake "veda/internal/syntax/treesitter/make"
	tstlaplus "veda/internal/syntax/treesitter/tlaplus"
	"veda/internal/theme"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	markdown_inline "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown-inline"
	"github.com/smacker/go-tree-sitter/python"
	tsx "github.com/smacker/go-tree-sitter/typescript/tsx"
	typescript "github.com/smacker/go-tree-sitter/typescript/typescript"
)

// TokenType represents different syntax elements
type TokenType string

const (
	TokenTypeText        TokenType = "text"
	TokenTypeKeyword     TokenType = "keyword"
	TokenTypeFunction    TokenType = "function"
	TokenTypeVariable    TokenType = "variable"
	TokenTypeConstant    TokenType = "constant"
	TokenTypeString      TokenType = "string"
	TokenTypeComment     TokenType = "comment"
	TokenTypeNumber      TokenType = "number"
	TokenTypeOperator    TokenType = "operator"
	TokenTypePunctuation TokenType = "punctuation"
	TokenTypeType        TokenType = "type"
	TokenTypeNamespace   TokenType = "namespace"
	TokenTypeAttribute   TokenType = "attribute"
	TokenTypeTag         TokenType = "tag"

	// Markdown-specific token types
	TokenTypeMarkdownH1             TokenType = "markdown_h1"
	TokenTypeMarkdownH2             TokenType = "markdown_h2"
	TokenTypeMarkdownH3             TokenType = "markdown_h3"
	TokenTypeMarkdownH4             TokenType = "markdown_h4"
	TokenTypeMarkdownH5             TokenType = "markdown_h5"
	TokenTypeMarkdownH6             TokenType = "markdown_h6"
	TokenTypeMarkdownBold           TokenType = "markdown_bold"
	TokenTypeMarkdownItalic         TokenType = "markdown_italic"
	TokenTypeMarkdownBoldItalic     TokenType = "markdown_bold_italic"
	TokenTypeMarkdownCode           TokenType = "markdown_code"
	TokenTypeMarkdownCodeBlock      TokenType = "markdown_code_block"
	TokenTypeMarkdownBlockquote     TokenType = "markdown_blockquote"
	TokenTypeMarkdownLink           TokenType = "markdown_link"
	TokenTypeMarkdownLinkText       TokenType = "markdown_link_text"
	TokenTypeMarkdownLinkUrl        TokenType = "markdown_link_url"
	TokenTypeMarkdownImage          TokenType = "markdown_image"
	TokenTypeMarkdownList           TokenType = "markdown_list"
	TokenTypeMarkdownListMarker     TokenType = "markdown_list_marker"
	TokenTypeMarkdownHorizontalRule TokenType = "markdown_horizontal_rule"
	TokenTypeMarkdownStrikethrough  TokenType = "markdown_strikethrough"
	TokenTypeMarkdownTable          TokenType = "markdown_table"
	TokenTypeMarkdownTableBorder    TokenType = "markdown_table_border"

	// Python-specific token types
	TokenTypePythonKeyword   TokenType = "python_keyword"
	TokenTypePythonBuiltin   TokenType = "python_builtin"
	TokenTypePythonDecorator TokenType = "python_decorator"
	TokenTypePythonString    TokenType = "python_string"
	TokenTypePythonDocstring TokenType = "python_docstring"
	TokenTypePythonFString   TokenType = "python_fstring"
	TokenTypePythonNumber    TokenType = "python_number"
	TokenTypePythonBoolean   TokenType = "python_boolean"
	TokenTypePythonNone      TokenType = "python_none"
	TokenTypePythonFunction  TokenType = "python_function"
	TokenTypePythonClass     TokenType = "python_class"
	TokenTypePythonSelf      TokenType = "python_self"
	TokenTypePythonComment   TokenType = "python_comment"
	TokenTypePythonOperator  TokenType = "python_operator"
	TokenTypePythonException TokenType = "python_exception"
	TokenTypePythonImport    TokenType = "python_import"
)

// Token represents a syntax-highlighted token
type Token struct {
	Start     int
	End       int
	TokenType TokenType
}

// Highlighter manages syntax highlighting for different languages
type Highlighter struct {
	parser        *sitter.Parser
	currentLang   string
	currentTree   *sitter.Tree
	currentSource []byte
	tokenCache    map[int][]Token // Cache key: startByte
}

// NewHighlighter creates a new syntax highlighter
func NewHighlighter() *Highlighter {
	parser := sitter.NewParser()
	return &Highlighter{
		parser:     parser,
		tokenCache: make(map[int][]Token),
	}
}

// SetLanguageFromFile sets the language based on file extension.
func (h *Highlighter) SetLanguageFromFile(filename string) error {
	return h.SetLanguageFromFileAndContent(filename, nil)
}

// SetLanguageFromFileAndContent sets the language based on file name first,
// then falls back to shebang detection from content.
func (h *Highlighter) SetLanguageFromFileAndContent(filename string, content []byte) error {
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
			h.parser.SetLanguage(tsmake.GetLanguage())
			h.currentLang = "make"
			return nil
		}
	}

	switch ext {
	case ".js", ".mjs", ".cjs", ".jsx":
		h.parser.SetLanguage(javascript.GetLanguage())
		h.currentLang = "javascript"
	case ".ts", ".mts", ".cts":
		h.parser.SetLanguage(typescript.GetLanguage())
		h.currentLang = "typescript"
	case ".tsx":
		h.parser.SetLanguage(tsx.GetLanguage())
		h.currentLang = "tsx"
	case ".go":
		h.parser.SetLanguage(golang.GetLanguage())
		h.currentLang = "go"
	case ".c", ".h", ".m":
		h.parser.SetLanguage(c.GetLanguage())
		h.currentLang = "c"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		h.parser.SetLanguage(cpp.GetLanguage())
		h.currentLang = "cpp"
	case ".md", ".markdown":
		h.parser.SetLanguage(markdown_inline.GetLanguage())
		h.currentLang = "markdown"
	case ".fish":
		h.parser.SetLanguage(tsfish.GetLanguage())
		h.currentLang = "fish"
	case ".sh", ".bash":
		h.parser.SetLanguage(bash.GetLanguage())
		h.currentLang = "bash"
	case ".py", ".pyw":
		h.parser.SetLanguage(python.GetLanguage())
		h.currentLang = "python"
	case ".mk", ".mak":
		h.parser.SetLanguage(tsmake.GetLanguage())
		h.currentLang = "make"
	case ".tla":
		h.parser.SetLanguage(tstlaplus.GetLanguage())
		h.currentLang = "tlaplus"
	case ".tex", ".latex", ".sty", ".cls", ".dtx", ".ltx":
		h.parser.SetLanguage(tslatex.GetLanguage())
		h.currentLang = "latex"
	default:
		if lang := languageFromShebang(content); lang != "" {
			return h.SetLanguage(lang)
		}

		// No language set for unknown extensions
		h.currentLang = ""
		h.currentTree = nil
		h.currentSource = nil
		h.tokenCache = make(map[int][]Token)
		return nil
	}

	return nil
}

func languageFromShebang(content []byte) string {
	lineEnd := bytes.IndexByte(content, '\n')
	if lineEnd < 0 {
		lineEnd = len(content)
	}
	line := strings.ToLower(string(bytes.TrimSpace(content[:lineEnd])))
	if !strings.HasPrefix(line, "#!") {
		return ""
	}

	switch {
	case strings.Contains(line, "python"):
		return "python"
	case strings.Contains(line, "bash"),
		strings.Contains(line, "/sh"),
		strings.Contains(line, " sh"),
		strings.Contains(line, "/zsh"),
		strings.Contains(line, " zsh"),
		strings.Contains(line, "/ksh"),
		strings.Contains(line, " ksh"),
		strings.Contains(line, "/dash"),
		strings.Contains(line, " dash"):
		return "bash"
	}

	return ""
}

// GetCurrentLanguage returns the currently set language
func (h *Highlighter) GetCurrentLanguage() string {
	return h.currentLang
}

// SetLanguage sets the language directly by name
func (h *Highlighter) SetLanguage(language string) error {
	switch language {
	case "javascript":
		h.parser.SetLanguage(javascript.GetLanguage())
		h.currentLang = "javascript"
	case "typescript":
		h.parser.SetLanguage(typescript.GetLanguage())
		h.currentLang = "typescript"
	case "tsx":
		h.parser.SetLanguage(tsx.GetLanguage())
		h.currentLang = "tsx"
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
	case "fish":
		h.parser.SetLanguage(tsfish.GetLanguage())
		h.currentLang = "fish"
	case "bash":
		h.parser.SetLanguage(bash.GetLanguage())
		h.currentLang = "bash"
	case "python":
		h.parser.SetLanguage(python.GetLanguage())
		h.currentLang = "python"
	case "make":
		h.parser.SetLanguage(tsmake.GetLanguage())
		h.currentLang = "make"
	case "tlaplus":
		h.parser.SetLanguage(tstlaplus.GetLanguage())
		h.currentLang = "tlaplus"
	case "latex":
		h.parser.SetLanguage(tslatex.GetLanguage())
		h.currentLang = "latex"
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

// ReparseIncremental computes a single contiguous edit between the current
// source and the provided newSource, applies a tree-sitter InputEdit to the
// current tree, and performs an incremental parse using the previous tree.
// If no previous tree exists or language is unset, it falls back to ParseFull.
func (h *Highlighter) ReparseIncremental(newSource []byte) error {
	if h.currentLang == "" || h.currentTree == nil || h.currentSource == nil {
		return h.ParseFull(newSource)
	}

	old := h.currentSource
	// Fast path: identical buffers
	if len(old) == len(newSource) {
		same := true
		for i := range old {
			if old[i] != newSource[i] {
				same = false
				break
			}
		}
		if same {
			return nil
		}
	}

	// Longest common prefix
	i := 0
	maxPrefix := len(old)
	if len(newSource) < maxPrefix {
		maxPrefix = len(newSource)
	}
	for i < maxPrefix && old[i] == newSource[i] {
		i++
	}
	startByte := i

	// Longest common suffix (avoid overlapping the prefix)
	oldRemain := len(old) - startByte
	newRemain := len(newSource) - startByte
	j := 0
	for j < oldRemain && j < newRemain {
		if old[len(old)-1-j] != newSource[len(newSource)-1-j] {
			break
		}
		j++
	}

	oldEndByte := len(old) - j
	newEndByte := len(newSource) - j

	// Compute points (row/column in bytes) for InputEdit
	startRow, startCol := byteIndexToPoint(old, startByte)
	oldEndRow, oldEndCol := byteIndexToPoint(old, oldEndByte)
	newEndRow, newEndCol := byteIndexToPoint(newSource, newEndByte)

	h.currentTree.Edit(sitter.EditInput{
		StartIndex:  uint32(startByte),
		OldEndIndex: uint32(oldEndByte),
		NewEndIndex: uint32(newEndByte),
		StartPoint:  sitter.Point{Row: uint32(startRow), Column: uint32(startCol)},
		OldEndPoint: sitter.Point{Row: uint32(oldEndRow), Column: uint32(oldEndCol)},
		NewEndPoint: sitter.Point{Row: uint32(newEndRow), Column: uint32(newEndCol)},
	})

	// Incremental parse using edited tree
	tree, err := h.parser.ParseCtx(context.Background(), h.currentTree, newSource)
	if err != nil {
		return fmt.Errorf("failed to incremental-parse source: %w", err)
	}

	h.currentTree = tree
	h.currentSource = newSource
	// Clear token cache (could be optimized to changed ranges)
	h.tokenCache = make(map[int][]Token)
	return nil
}

// ApplyEdit mutates the currentSource by replacing the span [startByte, startByte+len(oldText))
// with newText, applies the corresponding tree-sitter edit to the current tree, and performs
// an incremental parse using the edited tree. Points are computed in bytes.
// If the highlighter has no existing tree/source, this is a no-op (call ParseFull first).
func (h *Highlighter) ApplyEdit(startByte int, oldText, newText string) error {
	if h.currentLang == "" || h.currentTree == nil || h.currentSource == nil {
		// No prior tree/source: the caller should perform an initial full parse first
		return nil
	}
	if startByte < 0 {
		startByte = 0
	}
	if startByte > len(h.currentSource) {
		startByte = len(h.currentSource)
	}

	// Compute byte indices
	oldLen := len(oldText)
	newLen := len(newText)
	oldStart := startByte
	oldEnd := oldStart + oldLen
	if oldEnd > len(h.currentSource) {
		oldEnd = len(h.currentSource)
	}

	// Compute points on the old source
	startRow, startCol := byteIndexToPoint(h.currentSource, oldStart)
	oldEndRow, oldEndCol := byteIndexToPoint(h.currentSource, oldEnd)

	// Compute new end point relative to start by scanning newText
	// (row/col are byte-based; count newlines and bytes after last newline)
	newEndRow := startRow
	newEndCol := startCol
	if newLen == 0 {
		// no-op
	} else {
		// Count newlines
		nl := 0
		lastNL := -1
		for i := 0; i < newLen; i++ {
			if newText[i] == '\n' {
				nl++
				lastNL = i
			}
		}
		if nl == 0 {
			newEndCol += newLen
		} else {
			newEndRow += nl
			newEndCol = newLen - (lastNL + 1)
		}
	}

	// Apply edit to the tree
	h.currentTree.Edit(sitter.EditInput{
		StartIndex:  uint32(oldStart),
		OldEndIndex: uint32(oldEnd),
		NewEndIndex: uint32(oldStart + newLen),
		StartPoint:  sitter.Point{Row: uint32(startRow), Column: uint32(startCol)},
		OldEndPoint: sitter.Point{Row: uint32(oldEndRow), Column: uint32(oldEndCol)},
		NewEndPoint: sitter.Point{Row: uint32(newEndRow), Column: uint32(newEndCol)},
	})

	// Mutate the source mirror
	pre := h.currentSource[:oldStart]
	post := h.currentSource[oldEnd:]
	newSrc := make([]byte, 0, len(pre)+newLen+len(post))
	newSrc = append(newSrc, pre...)
	newSrc = append(newSrc, newText...)
	newSrc = append(newSrc, post...)

	// Incremental parse using the edited tree and updated source
	tree, err := h.parser.ParseCtx(context.Background(), h.currentTree, newSrc)
	if err != nil {
		return fmt.Errorf("failed to incremental-parse apply-edit: %w", err)
	}

	h.currentTree = tree
	h.currentSource = newSrc
	h.tokenCache = make(map[int][]Token)
	return nil
}

// byteIndexToPoint converts a byte index into (row, column) in bytes.
// Row is zero-based; Column counts bytes since last newline.
func byteIndexToPoint(src []byte, idx int) (row int, col int) {
	if idx <= 0 {
		return 0, 0
	}
	if idx > len(src) {
		idx = len(src)
	}
	lastNL := -1
	r := 0
	for k := 0; k < idx; k++ {
		if src[k] == '\n' {
			r++
			lastNL = k
		}
	}
	c := idx - (lastNL + 1)
	if lastNL == -1 {
		c = idx
	}
	return r, c
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

// PreloadTokenCache pre-loads syntax highlighting for a range of lines to improve scrolling performance
func (h *Highlighter) PreloadTokenCache(startLine, endLine int) {
	if h.currentTree == nil || h.currentLang == "" || h.currentSource == nil {
		return
	}

	// Convert line numbers to byte positions
	source := h.currentSource
	lines := 0
	lineStarts := []int{0} // byte positions where each line starts

	for i, b := range source {
		if b == '\n' {
			lines++
			if i+1 < len(source) {
				lineStarts = append(lineStarts, i+1)
			}
		}
	}

	// Process each line in the range
	for line := startLine; line <= endLine && line < len(lineStarts); line++ {
		lineStart := lineStarts[line]
		var lineEnd int
		if line+1 < len(lineStarts) {
			lineEnd = lineStarts[line+1] - 1 // Exclude the newline
		} else {
			lineEnd = len(source)
		}

		// Skip if already cached
		if _, ok := h.tokenCache[lineStart]; ok {
			continue
		}

		// Pre-load this line
		h.GetTokensForLine(lineStart, lineEnd)
	}
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

	if !shouldTokenize && h.currentLang == "fish" {
		switch nodeType {
		case "double_quote_string", "single_quote_string", "variable_expansion", "list_element_access", "command_substitution":
			shouldTokenize = true
		}
	}

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

	// Prefer optimized traversal that skips irrelevant children when there are multiple children
	if childCount > 4 {
		h.walkNodeOptimized(node, lineStart, lineEnd, source, tokens)
	} else {
		for i := 0; i < childCount; i++ {
			child := node.Child(i)
			h.walkNode(child, lineStart, lineEnd, source, tokens)
		}
	}
}

// walkNodeOptimized provides faster traversal for large nodes with many children
func (h *Highlighter) walkNodeOptimized(node *sitter.Node, lineStart, lineEnd int, source []byte, tokens *[]Token) {
	childCount := int(node.ChildCount())

	// Binary search to find the first child whose EndByte > lineStart
	low, high := 0, childCount
	for low < high {
		mid := (low + high) / 2
		ch := node.Child(mid)
		if int(ch.EndByte()) <= lineStart {
			low = mid + 1
		} else {
			high = mid
		}
	}
	firstChild := low

	// Binary search to find the last child whose StartByte < lineEnd
	low, high = firstChild, childCount
	for low < high {
		mid := (low + high) / 2
		ch := node.Child(mid)
		if int(ch.StartByte()) < lineEnd {
			low = mid + 1
		} else {
			high = mid
		}
	}
	lastChild := low - 1
	if lastChild < firstChild {
		return
	}

	// Process only the relevant children
	for i := firstChild; i <= lastChild; i++ {
		child := node.Child(i)
		h.walkNode(child, lineStart, lineEnd, source, tokens)
	}
}

// mapNodeToTokenType maps tree-sitter node types to our token types
func (h *Highlighter) mapNodeToTokenType(node *sitter.Node, source []byte) TokenType {
	nodeType := node.Type()

	switch h.currentLang {
	case "javascript":
		return h.mapJavaScriptNodeToTokenType(node, nodeType, source)
	case "typescript", "tsx":
		return h.mapTypeScriptNodeToTokenType(node, nodeType, source)
	case "go":
		return h.mapGoNodeToTokenType(node, nodeType, source)
	case "c":
		return h.mapCNodeToTokenType(node, nodeType, source)
	case "cpp":
		return h.mapCppNodeToTokenType(node, nodeType, source)
	case "markdown":
		return h.mapMarkdownNodeToTokenType(node, nodeType, source)
	case "fish":
		return h.mapFishNodeToTokenType(node, nodeType, source)
	case "bash":
		return h.mapBashNodeToTokenType(node, nodeType, source)
	case "python":
		return h.mapPythonNodeToTokenType(node, nodeType, source)
	case "make":
		return h.mapMakeNodeToTokenType(node, nodeType, source)
	case "tlaplus":
		return h.mapTLAPlusNodeToTokenType(node, nodeType, source)
	case "latex":
		return h.mapLatexNodeToTokenType(node, nodeType, source)
	default:
		return TokenTypeText
	}
}

// mapJavaScriptNodeToTokenType maps JavaScript-specific node types to token types
func (h *Highlighter) mapJavaScriptNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// Keywords (JS)
	case "function", "return", "if", "else", "for", "while", "do",
		"switch", "case", "default", "break", "continue",
		"var", "let", "const", "class", "extends", "super", "new",
		"try", "catch", "finally", "throw", "typeof", "instanceof",
		"in", "of", "delete", "yield", "await", "async", "import",
		"export", "from", "this":
		return TokenTypeKeyword

	// Comments
	case "comment":
		return TokenTypeComment

	// Strings and templates
	case "string", "template_string", "template_substitution", "regex":
		return TokenTypeString

	// Numbers
	case "number":
		return TokenTypeNumber

	// Boolean and null literals
	case "true", "false", "null", "undefined":
		return TokenTypeConstant

	// Identifiers and function names
	case "identifier", "property_identifier":
		parent := node.Parent()
		if parent != nil {
			pt := parent.Type()
			if pt == "call_expression" && (parent.ChildByFieldName("function") == node || parent.Child(0) == node) {
				return TokenTypeFunction
			}
			if pt == "function_declaration" || pt == "method_definition" {
				return TokenTypeFunction
			}
			if pt == "class_declaration" || pt == "class" {
				return TokenTypeType
			}
		}
		return TokenTypeVariable

	// Punctuation
	case "(", ")", "[", "]", "{", "}", ",", ";", ".", ":":
		return TokenTypePunctuation

	// Operators
	case "+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>",
		"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=",
		"&&", "||", "++", "--",
		"==", "!=", "===", "!==", "<", "<=", ">", ">=",
		"=", "!", "~", "?", "=>":
		return TokenTypeOperator

	default:
		return TokenTypeText
	}
}

// mapTypeScriptNodeToTokenType maps TypeScript-specific node types to token types
func (h *Highlighter) mapTypeScriptNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// TS keywords (includes JS + TS types)
	case "function", "return", "if", "else", "for", "while", "do",
		"switch", "case", "default", "break", "continue",
		"var", "let", "const", "class", "extends", "super", "new",
		"try", "catch", "finally", "throw", "typeof", "instanceof",
		"in", "of", "delete", "yield", "await", "async", "import",
		"export", "from", "this", "interface", "implements", "enum",
		"type", "readonly", "abstract", "private", "public", "protected",
		"as":
		return TokenTypeKeyword

	// Comments
	case "comment":
		return TokenTypeComment

	// Strings / templates / regex
	case "string", "template_string", "template_substitution", "regex":
		return TokenTypeString

	// Numbers
	case "number":
		return TokenTypeNumber

	// Boolean and null literals
	case "true", "false", "null", "undefined":
		return TokenTypeConstant

	// Type-related nodes
	case "type_identifier", "predefined_type", "type_alias_declaration":
		return TokenTypeType

	// Identifiers and function contexts
	case "identifier", "property_identifier":
		parent := node.Parent()
		if parent != nil {
			pt := parent.Type()
			if pt == "call_expression" && (parent.ChildByFieldName("function") == node || parent.Child(0) == node) {
				return TokenTypeFunction
			}
			if pt == "function_declaration" || pt == "method_signature" || pt == "method_definition" {
				return TokenTypeFunction
			}
			if pt == "class_declaration" || pt == "interface_declaration" {
				return TokenTypeType
			}
		}
		return TokenTypeVariable

	// Punctuation (exclude '<' and '>' which we classify as operators)
	case "(", ")", "[", "]", "{", "}", ",", ";", ".", ":":
		return TokenTypePunctuation

	// Operators
	case "+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>",
		"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=",
		"&&", "||", "++", "--",
		"==", "!=", "===", "!==", "<", "<=", ">", ">=",
		"=", "!", "~", "?", "=>":
		return TokenTypeOperator

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
		// Treat list markers as list markers if they are the first non-whitespace on the line.
		start := int(node.StartByte())
		lineStart := start
		for lineStart > 0 && source[lineStart-1] != '\n' {
			lineStart--
		}
		for i := lineStart; i < start; i++ {
			if source[i] != ' ' && source[i] != '\t' {
				return TokenTypeText
			}
		}
		return TokenTypeMarkdownListMarker

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

// postProcessMarkdownTokens extends header and blockquote tokens to cover the entire line
func (h *Highlighter) postProcessMarkdownTokens(tokens []Token, lineStart, lineEnd int, source []byte) []Token {
	processed := make([]Token, 0, len(tokens))

	for _, token := range tokens {
		extendLine := false
		if token.Start == 0 {
			switch token.TokenType {
			case TokenTypeMarkdownH1,
				TokenTypeMarkdownH2,
				TokenTypeMarkdownH3,
				TokenTypeMarkdownH4,
				TokenTypeMarkdownH5,
				TokenTypeMarkdownH6,
				TokenTypeMarkdownBlockquote:
				extendLine = true
			}
		}

		if extendLine {
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
			continue
		}

		processed = append(processed, token)
	}

	return h.mergeOverlappingTokens(processed)
}

// mapFishNodeToTokenType maps fish shell node types to token types.
func (h *Highlighter) mapFishNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	case "comment":
		return TokenTypeComment
	case "double_quote_string", "single_quote_string":
		return TokenTypeString
	case "integer", "float":
		return TokenTypeNumber
	case "variable_name", "variable_expansion", "list_element_access":
		return TokenTypeVariable
	case "command_substitution", "brace_expansion", "home_dir_expansion", "range", "glob":
		return TokenTypeConstant
	case "stream_redirect", "file_redirect", "direction", "pipe", "conditional_execution":
		return TokenTypeOperator
	case "!", "&&", "&", "&|", "|", "||", "..", "2>|":
		return TokenTypeOperator
	case "(", ")", "{", "}", "[", "]", ",", ";", "\"", "'":
		return TokenTypePunctuation
	case "begin", "end", "if", "else", "switch", "case", "while", "for", "in", "and", "or", "return", "break", "continue", "function", "command":
		return TokenTypeKeyword
	case "escape_sequence":
		return TokenTypeConstant
	case "word":
		parent := node.Parent()
		if parent != nil {
			switch parent.Type() {
			case "command":
				if name := parent.ChildByFieldName("name"); name != nil && name.Equal(node) {
					return TokenTypeFunction
				}
			case "function_definition":
				if name := parent.ChildByFieldName("name"); name != nil && name.Equal(node) {
					return TokenTypeFunction
				}
			}
		}
		return TokenTypeText
	default:
		return TokenTypeText
	}
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

// mapMakeNodeToTokenType maps Make-specific node types to token types
func (h *Highlighter) mapMakeNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// Comments
	case "comment":
		return TokenTypeComment

	// Variable assignments and references
	case "variable_assignment":
		// The variable_assignment node itself should not be styled - its children will be
		return TokenTypeText

	case "variable_reference":
		// The variable_reference node itself should not be styled - its children will be
		return TokenTypeConstant

	// Targets
	case "targets":
		// Check if it's a standard target
		if node.ChildCount() > 0 {
			wordNode := node.Child(0)
			if wordNode.Type() == "word" {
				targetName := string(source[wordNode.StartByte():wordNode.EndByte()])
				if h.isMakeStandardTarget(targetName) || h.isMakeBuiltinTarget(targetName) {
					return TokenTypeConstant
				}
			}
		}
		return TokenTypeFunction

	// Words in different contexts
	case "word":
		parent := node.Parent()
		if parent != nil {
			parentType := parent.Type()
			switch parentType {
			case "variable_assignment":
				// Variable names
				varName := string(source[node.StartByte():node.EndByte()])
				if h.isMakeBuiltinVariable(varName) {
					return TokenTypeConstant
				}
				return TokenTypeVariable
			case "variable_reference":
				// Variable references
				return TokenTypeConstant
			case "targets":
				// Target names
				targetName := string(source[node.StartByte():node.EndByte()])
				if h.isMakeStandardTarget(targetName) || h.isMakeBuiltinTarget(targetName) {
					return TokenTypeConstant
				}
				return TokenTypeFunction
			}
		}
		// Check if it's a Make function
		wordText := string(source[node.StartByte():node.EndByte()])
		if h.isMakeFunction(wordText) {
			return TokenTypeKeyword
		}
		if h.isMakeKeyword(wordText) {
			return TokenTypeKeyword
		}
		return TokenTypeText

	// Operators
	case "=", ":=", "::=", "?=", "+=", "!=", "@", "-", "+":
		return TokenTypeOperator
	case ":", "&:", "::", "|", ";":
		return TokenTypeOperator

	// Punctuation
	case "(", ")", "{", "}", "\"", "'", ",":
		return TokenTypePunctuation
	case "$", "$$":
		return TokenTypeOperator

	// Automatic variables
	case "automatic_variable":
		return TokenTypeConstant

	// Strings and text
	case "text", "string", "raw_text":
		return TokenTypeString
	case "shell_text":
		return TokenTypeText

	// Recipe lines
	case "recipe_line":
		return TokenTypeText

	// Control flow
	case "ifeq", "ifneq", "ifdef", "ifndef", "else", "endif", "if":
		return TokenTypeKeyword
	case "foreach":
		return TokenTypeKeyword

	// Includes
	case "include", "sinclude", "-include":
		return TokenTypeKeyword

	// Other keywords
	case "define", "endef", "vpath", "undefine", "export", "unexport", "override", "private":
		return TokenTypeKeyword

	// Functions
	case "function_call":
		return TokenTypeFunction

	default:
		return TokenTypeText
	}
}

// Helper functions for Make-specific classifications
func (h *Highlighter) isMakeBuiltinVariable(name string) bool {
	builtinVars := map[string]bool{
		"AR": true, "AS": true, "CC": true, "CXX": true, "CPP": true, "FC": true,
		"M2C": true, "PC": true, "CO": true, "GET": true, "LEX": true, "YACC": true,
		"LINT": true, "MAKEINFO": true, "TEX": true, "TEXI2DVI": true, "WEAVE": true,
		"CWEAVE": true, "TANGLE": true, "CTANGLE": true, "RM": true, "ARFLAGS": true,
		"ASFLAGS": true, "CFLAGS": true, "CXXFLAGS": true, "COFLAGS": true, "CPPFLAGS": true,
		"FFLAGS": true, "GFLAGS": true, "LDFLAGS": true, "LDLIBS": true, "LFLAGS": true,
		"YFLAGS": true, "PFLAGS": true, "RFLAGS": true, "LINTFLAGS": true, "VPATH": true,
		".RECIPEPREFIX": true, ".DEFAULT_GOAL": true, ".EXTRA_PREREQS": true,
		"MAKEFILE_LIST": true, "MAKE_RESTARTS": true, "MAKE_TERMOUT": true, "MAKE_TERMERR": true,
	}
	return builtinVars[name]
}

func (h *Highlighter) isMakeStandardTarget(name string) bool {
	standardTargets := map[string]bool{
		"all": true, "install": true, "install-html": true, "install-dvi": true,
		"install-pdf": true, "install-ps": true, "uninstall": true, "install-strip": true,
		"clean": true, "distclean": true, "mostlyclean": true, "maintainer-clean": true,
		"TAGS": true, "info": true, "dvi": true, "html": true, "pdf": true, "ps": true,
		"dist": true, "check": true, "installcheck": true, "installdirs": true,
	}
	return standardTargets[name]
}

func (h *Highlighter) isMakeBuiltinTarget(name string) bool {
	builtinTargets := map[string]bool{
		".PHONY": true, ".SUFFIXES": true, ".DEFAULT": true, ".PRECIOUS": true,
		".INTERMEDIATE": true, ".SECONDARY": true, ".SECONDEXPANSION": true,
		".DELETE_ON_ERROR": true, ".IGNORE": true, ".LOW_RESOLUTION_TIME": true,
		".SILENT": true, ".EXPORT_ALL_VARIABLES": true, ".NOTPARALLEL": true,
		".ONESHELL": true, ".POSIX": true,
	}
	return builtinTargets[name]
}

func (h *Highlighter) isMakeFunction(name string) bool {
	functions := map[string]bool{
		"subst": true, "patsubst": true, "strip": true, "findstring": true,
		"filter": true, "filter-out": true, "sort": true, "word": true,
		"words": true, "wordlist": true, "firstword": true, "lastword": true,
		"dir": true, "notdir": true, "suffix": true, "basename": true,
		"addsuffix": true, "addprefix": true, "join": true, "wildcard": true,
		"realpath": true, "abspath": true, "call": true, "eval": true,
		"file": true, "value": true, "shell": true, "error": true,
		"warning": true, "info": true,
	}
	return functions[name]
}

func (h *Highlighter) isMakeKeyword(name string) bool {
	keywords := map[string]bool{
		"ifeq": true, "ifneq": true, "ifdef": true, "ifndef": true,
		"else": true, "endif": true, "if": true, "or": true, "and": true,
		"foreach": true, "define": true, "endef": true, "vpath": true,
		"undefine": true, "export": true, "unexport": true, "override": true,
		"private": true, "include": true, "sinclude": true, "-include": true,
	}
	return keywords[name]
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
		"->": // Type annotation arrow
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

// mapTLAPlusNodeToTokenType maps TLA+/PlusCal node types to token types.
func (h *Highlighter) mapTLAPlusNodeToTokenType(node *sitter.Node, nodeType string, source []byte) TokenType {
	switch nodeType {
	// Comments
	case "comment", "block_comment":
		return TokenTypeComment

	// TLA+ keywords
	case "MODULE", "EXTENDS", "VARIABLE", "VARIABLES",
		"CONSTANT", "CONSTANTS", "RECURSIVE",
		"INSTANCE", "WITH", "LOCAL", "LAMBDA",
		"CHOOSE", "IF", "THEN", "ELSE",
		"CASE", "OTHER", "LET", "IN",
		"ASSUME", "ASSUMPTION", "AXIOM",
		"THEOREM", "PROPOSITION", "LEMMA", "COROLLARY",
		"PROVE", "PROOF", "BY", "OBVIOUS", "OMITTED",
		"DEFINE", "HAVE", "WITNESS", "TAKE",
		"SUFFICES", "PICK", "QED",
		"USE", "HIDE", "NEW", "STATE", "ACTION", "TEMPORAL",
		"EXCEPT", "DEF", "DEFS", "ONLY":
		return TokenTypeKeyword

	// PlusCal keywords
	case "algorithm", "fair", "process", "procedure",
		"macro", "define", "variable", "variables",
		"begin", "end", "if", "then", "elsif", "else",
		"while", "do", "either", "or", "with",
		"await", "when", "print", "assert",
		"goto", "call", "pcal_return", "pcal_skip":
		return TokenTypeKeyword

	// Strings
	case "string":
		return TokenTypeString

	// Numbers
	case "nat_number", "real_number", "binary_number", "octal_number", "hex_number":
		return TokenTypeNumber

	// Built-in constants and sets
	case "boolean",
		"nat_number_set", "int_number_set", "real_number_set",
		"string_set", "boolean_set":
		return TokenTypeConstant

	// Operators
	case "def_eq", "set_in", "gets",
		"forall", "exists", "all_map_to", "maps_to",
		"implies", "equiv", "iff", "plus_arrow", "leads_to",
		"land", "lor", "lnot",
		"eq", "neq", "lt", "gt", "leq", "geq",
		"approx", "sim", "simeq", "asymp", "cong", "doteq",
		"gg", "ll", "in", "notin",
		"prec", "succ", "preceq", "succeq", "propto",
		"subset", "supset", "subseteq", "supseteq",
		"sqsubset", "sqsupset", "sqsubseteq", "sqsupseteq",
		"cap", "cup", "uplus", "sqcap", "sqcup",
		"union", "powerset", "domain",
		"plus", "minus", "times", "div",
		"oplus", "ominus", "odot", "oslash", "otimes",
		"circ", "bullet", "star", "bigcirc",
		"vertvert", "cdot", "wr", "qq", "excl",
		"assign", "bnf_rule",
		"always", "eventually",
		"negative", "enabled", "unchanged",
		"dots_2", "dots_3",
		"sup_plus",
		"rs_ttile", "rd_ttile", "ls_ttile", "ld_ttile":
		return TokenTypeOperator

	// Punctuation
	case "(", ")", "[", "]", "{", "}", ",", ":", ";",
		"langle_bracket", "rangle_bracket", "rangle_bracket_sub",
		"single_line", "double_line",
		"case_box", "case_arrow",
		"colon", "address", "label_as",
		"bullet_conj", "bullet_disj":
		return TokenTypePunctuation

	// Identifiers with context
	case "identifier":
		if node.Parent() != nil {
			parentType := node.Parent().Type()
			switch parentType {
			case "operator_definition", "function_definition",
				"pcal_macro_decl", "pcal_proc_decl":
				return TokenTypeFunction
			case "variable_declaration", "pcal_var_decl", "pcal_proc_var_decl":
				return TokenTypeVariable
			case "constant_declaration":
				return TokenTypeType
			}
		}
		return TokenTypeText

	default:
		return TokenTypeText
	}
}

// mapLatexNodeToTokenType maps LaTeX node types to token types.
func (h *Highlighter) mapLatexNodeToTokenType(node *sitter.Node, nodeType string, _ []byte) TokenType {
	switch nodeType {
	// Comments
	case "line_comment", "block_comment", "comment":
		return TokenTypeComment

	// Commands (backslash commands like \textbf, \item)
	case "command_name":
		return TokenTypeFunction

	// Section commands
	case `\section`, `\subsection`, `\subsubsection`, `\paragraph`, `\subparagraph`,
		`\part`, `\chapter`:
		return TokenTypeKeyword

	// Environment markers
	case `\begin`, `\end`:
		return TokenTypeKeyword

	// Item marker
	case `\item`:
		return TokenTypeKeyword

	// Document structure commands
	case `\documentclass`, `\usepackage`, `\title`, `\author`, `\date`:
		return TokenTypeKeyword

	// Definition commands
	case `\newcommand`, `\renewcommand`, `\newenvironment`, `\renewenvironment`,
		`\providecommand`, `\DeclareOption`, `\ProcessOptions`:
		return TokenTypeKeyword

	// Label/reference commands
	case `\label`, `\ref`, `\eqref`, `\pageref`, `\cite`, `\cref`, `\Cref`:
		return TokenTypeVariable

	// Include commands
	case `\input`, `\include`, `\bibliography`, `\bibliographystyle`:
		return TokenTypeNamespace

	// Math delimiters
	case `\[`, `\]`, "$":
		return TokenTypeOperator

	// Math operators and symbols
	case "operator", "superscript", "subscript", "^", "_", "+", "-", "=":
		return TokenTypeOperator

	// Structural node types (highlight the whole construct)
	case "inline_formula", "displayed_equation", "math_environment":
		return TokenTypeConstant

	case "class_include", "package_include":
		return TokenTypeNamespace

	case "new_command_definition", "environment_definition",
		"title_declaration", "author_declaration":
		return TokenTypeKeyword

	case "label_definition", "label_reference", "citation":
		return TokenTypeVariable

	case "section", "subsection", "subsubsection", "paragraph", "subparagraph",
		"part", "chapter":
		return TokenTypeKeyword

	// Punctuation
	case "{", "}", "[", "]", "(", ")":
		return TokenTypePunctuation

	// Labels and paths (inside curly groups)
	case "label", "path":
		return TokenTypeString

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
	case TokenTypeText:
		fallthrough
	default:
		return theme.SyntaxText
	}
}
